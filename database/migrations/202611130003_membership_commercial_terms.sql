-- Commercial terms are versioned with the price. Existing prices retain their semantics.
-- No prices, trials or paid access are fabricated, and no existing brand is enrolled.
ALTER TABLE legui_private.billing_plan_prices
 ADD COLUMN membership_tier text NOT NULL DEFAULT 'legacy' CHECK(membership_tier IN ('legacy','standard','premium')),
 ADD COLUMN included_locations integer NOT NULL DEFAULT 1 CHECK(included_locations=1),
 ADD COLUMN extra_location_amount numeric(18,2) NOT NULL DEFAULT 0 CHECK(extra_location_amount>=0 AND extra_location_amount<>'NaN'::numeric),
 ADD CONSTRAINT commercial_price_terms CHECK(membership_tier='legacy' OR (
  billing_unit='brand' AND interval_months=1 AND trial_days=0 AND
  ((membership_tier='standard' AND maximum_locations IS NOT DISTINCT FROM 1 AND maximum_collaborators IS NOT DISTINCT FROM 2 AND extra_location_amount=0)
   OR (membership_tier='premium' AND maximum_locations IS NULL AND maximum_collaborators IS NOT DISTINCT FROM 5))));
ALTER TABLE legui_private.subscriptions
 ADD COLUMN purchased_locations integer NOT NULL DEFAULT 1 CHECK(purchased_locations BETWEEN 1 AND 10000);

CREATE OR REPLACE FUNCTION legui_private.guard_plan_price() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN
 IF (to_jsonb(NEW)-'is_active') IS DISTINCT FROM (to_jsonb(OLD)-'is_active') THEN
  RAISE EXCEPTION 'Published price terms are immutable; create a new price version' USING ERRCODE='23514';
 END IF;
 IF NEW.is_active AND NEW.membership_tier<>'legacy' AND (NEW.amount<=0 OR (NEW.membership_tier='premium' AND NEW.extra_location_amount<=0)) THEN
  RAISE EXCEPTION 'Commercial prices need a base price and a premium additional location price' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION legui_private.validate_commercial_price() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN
 IF NEW.is_active AND NEW.membership_tier<>'legacy' AND (NEW.amount<=0 OR (NEW.membership_tier='premium' AND NEW.extra_location_amount<=0)) THEN
  RAISE EXCEPTION 'Commercial prices need a base price and a premium additional location price' USING ERRCODE='23514';
 END IF;RETURN NEW;
END $$;
CREATE TRIGGER commercial_price_insert BEFORE INSERT ON legui_private.billing_plan_prices FOR EACH ROW EXECUTE FUNCTION legui_private.validate_commercial_price();

CREATE FUNCTION legui_private.contract_commercial_terms() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE price legui_private.billing_plan_prices; b uuid;
BEGIN
 SELECT * INTO price FROM legui_private.billing_plan_prices WHERE id=NEW.plan_price_id;
 IF TG_OP='UPDATE' THEN
  IF (price.membership_tier<>'legacy' OR EXISTS(SELECT 1 FROM legui_private.billing_plan_prices WHERE id=OLD.plan_price_id AND membership_tier<>'legacy'))
   AND ROW(NEW.plan_price_id,NEW.contracted_amount,NEW.quantity,NEW.interval_months,NEW.purchased_locations)
    IS DISTINCT FROM ROW(OLD.plan_price_id,OLD.contracted_amount,OLD.quantity,OLD.interval_months,OLD.purchased_locations) THEN
   RAISE EXCEPTION 'Commercial contract terms are immutable; a change requires a new agreement' USING ERRCODE='23514';
  END IF;
 ELSIF price.membership_tier<>'legacy' THEN
  IF NOT price.is_active OR NEW.quantity<>1 OR (price.membership_tier='standard' AND NEW.purchased_locations<>1) THEN
   RAISE EXCEPTION 'Invalid commercial contract' USING ERRCODE='23514';
  END IF;
  NEW.contracted_amount:=price.amount+(NEW.purchased_locations-price.included_locations)*price.extra_location_amount;
 END IF;
 IF price.membership_tier<>'legacy' AND (TG_OP='INSERT' OR NEW.status IN ('active','trialing')) THEN
  SELECT brand_id INTO b FROM legui_private.billing_accounts WHERE id=NEW.account_id AND tenant_id IS NULL;
  IF b IS NULL THEN RAISE EXCEPTION 'Commercial membership belongs to a brand' USING ERRCODE='23514'; END IF;
  PERFORM pg_advisory_xact_lock(hashtextextended('business-capacity:'||b::text,0));
  IF (SELECT count(*) FROM public.tenants WHERE brand_id=b AND active)>NEW.purchased_locations THEN
   RAISE EXCEPTION 'Contract needs enough purchased locations for this business' USING ERRCODE='23514',CONSTRAINT='business_plan_capacity';
  END IF;

 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER subscriptions_commercial_terms BEFORE INSERT OR UPDATE ON legui_private.subscriptions FOR EACH ROW EXECUTE FUNCTION legui_private.contract_commercial_terms();

CREATE OR REPLACE FUNCTION legui_private.business_capacity(b uuid) RETURNS TABLE(locations integer,collaborators integer,source text)
 LANGUAGE sql STABLE SET search_path='' AS $$
 WITH plan AS(SELECT CASE WHEN p.membership_tier='legacy' THEN p.maximum_locations ELSE s.purchased_locations END maximum_locations,p.maximum_collaborators,
 p.membership_tier FROM legui_private.subscriptions s
 JOIN legui_private.billing_accounts a ON a.id=s.account_id AND a.closed_at IS NULL
 JOIN legui_private.billing_plan_prices p ON p.id=s.plan_price_id
 WHERE a.brand_id=b AND a.tenant_id IS NULL AND s.environment='production'
 AND ((s.status='active' AND s.current_period_start<=now() AND s.current_period_end>now()) OR (s.status='trialing' AND s.trial_ends_at>now()))
 ORDER BY s.started_at DESC LIMIT 1)
 SELECT p.maximum_locations,p.maximum_collaborators,CASE WHEN p.membership_tier='legacy' THEN 'subscription' ELSE 'subscription_per_location' END FROM plan p
 UNION ALL SELECT x.base_location_limit,x.base_collaborator_limit,'base'::text FROM public.brands x WHERE x.id=b AND NOT EXISTS(SELECT 1 FROM plan)
$$;

-- Commercial quotas are simultaneous operating terminals, not named staff accounts.
CREATE OR REPLACE FUNCTION legui_private.check_business_capacity(b uuid,kind text,exclude_id uuid,exclude_email text) RETURNS void
 LANGUAGE plpgsql SET search_path='' AS $$
 DECLARE cap integer; used integer;src text; BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('business-capacity:'||b::text,0));
 IF kind='location' THEN
 SELECT locations INTO cap FROM legui_private.business_capacity(b);
 SELECT count(*) INTO used FROM public.tenants WHERE brand_id=b AND active AND id IS DISTINCT FROM exclude_id;
 ELSE
 SELECT collaborators,source INTO cap,src FROM legui_private.business_capacity(b);
 IF src IN ('subscription_per_location','trial','expired','pending') THEN RETURN;END IF;
 SELECT count(*) INTO used FROM (
 SELECT lower(e.email) email FROM public.employees e JOIN public.tenants t ON t.id=e.tenant_id
 WHERE t.brand_id=b AND e.is_active AND e.id IS DISTINCT FROM exclude_id AND lower(e.email)<>lower(coalesce(exclude_email,''))
 UNION SELECT lower(a.email) FROM legui_private.auth_action_tokens a JOIN public.tenants t ON t.id=a.tenant_id
 WHERE t.brand_id=b AND a.purpose='team_invitation' AND a.used_at IS NULL AND a.expires_at>now()
 AND lower(a.email)<>lower(coalesce(exclude_email,''))
 AND NOT EXISTS(SELECT 1 FROM public.employees e JOIN public.tenants et ON et.id=e.tenant_id WHERE et.brand_id=b AND e.is_active AND lower(e.email)=lower(a.email))) people;
 END IF;
 IF cap IS NOT NULL AND used>=cap THEN RAISE EXCEPTION 'Business plan capacity reached' USING ERRCODE='23514',CONSTRAINT='business_plan_capacity'; END IF;
 END $$;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
