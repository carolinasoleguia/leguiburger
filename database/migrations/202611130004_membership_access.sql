CREATE TABLE legui_private.membership_trials(
 brand_id uuid PRIMARY KEY REFERENCES public.brands(id),
 started_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 expires_at timestamptz NOT NULL
);
CREATE FUNCTION legui_private.set_trial_expiry() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN NEW.expires_at:=NEW.started_at+interval '168 hours';RETURN NEW;END $$;
CREATE TRIGGER membership_trial_expiry BEFORE INSERT ON legui_private.membership_trials FOR EACH ROW EXECUTE FUNCTION legui_private.set_trial_expiry();
ALTER TABLE legui_private.membership_trials ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON legui_private.membership_trials FROM PUBLIC,anon,authenticated,service_role;
CREATE TRIGGER membership_trial_immutable BEFORE UPDATE OR DELETE ON legui_private.membership_trials
 FOR EACH ROW EXECUTE FUNCTION legui_private.reject_history_change();
-- Existing businesses receive a fresh transition period; never backdate their trial.
INSERT INTO legui_private.membership_trials(brand_id)
 SELECT DISTINCT brand_id FROM public.users WHERE role='admin' AND is_active AND brand_id IS NOT NULL;
CREATE FUNCTION legui_private.start_membership_trial() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN
 IF NEW.role='admin' AND NEW.is_active AND NEW.brand_id IS NOT NULL THEN
  INSERT INTO legui_private.membership_trials(brand_id) VALUES(NEW.brand_id) ON CONFLICT DO NOTHING;
 END IF;RETURN NEW;
END $$;
CREATE TRIGGER user_membership_trial AFTER INSERT OR UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION legui_private.start_membership_trial();

CREATE FUNCTION legui_private.membership_access(b uuid)
 RETURNS TABLE(tier text,state text,operational boolean,premium boolean,locations integer,terminals_per_location integer,valid_until timestamptz)
 LANGUAGE sql STABLE SET search_path='' AS $$
 WITH paid AS(
  SELECT p.membership_tier,s.purchased_locations,p.maximum_locations,p.maximum_collaborators,s.current_period_end
  FROM legui_private.subscriptions s
  JOIN legui_private.billing_accounts a ON a.id=s.account_id AND a.closed_at IS NULL AND a.tenant_id IS NULL
  JOIN legui_private.billing_plan_prices p ON p.id=s.plan_price_id
  WHERE a.brand_id=b AND s.environment='production' AND s.status IN ('active','past_due','paused','canceled')
   AND s.activated_at IS NOT NULL AND s.current_period_start<=now() AND s.current_period_end>now()
  ORDER BY s.current_period_end DESC,s.started_at DESC LIMIT 1
 ), trial AS(SELECT expires_at FROM legui_private.membership_trials WHERE brand_id=b)
 SELECT membership_tier,'paid'::text,true,membership_tier IN ('legacy','premium'),
  CASE WHEN membership_tier='legacy' THEN maximum_locations ELSE purchased_locations END,
  maximum_collaborators,current_period_end FROM paid
 UNION ALL
 SELECT 'trial',CASE WHEN expires_at>now() THEN 'trial' ELSE 'expired' END,expires_at>now(),false,1,2,expires_at
  FROM trial WHERE NOT EXISTS(SELECT 1 FROM paid)
 UNION ALL
 SELECT 'trial','pending',false,false,1,2,NULL::timestamptz
  WHERE NOT EXISTS(SELECT 1 FROM paid) AND NOT EXISTS(SELECT 1 FROM trial)
$$;
CREATE OR REPLACE FUNCTION legui_private.business_capacity(b uuid) RETURNS TABLE(locations integer,collaborators integer,source text)
 LANGUAGE sql STABLE SET search_path='' AS $$
 SELECT a.locations,a.terminals_per_location,CASE WHEN a.state='paid' THEN
  CASE WHEN a.tier='legacy' THEN 'subscription' ELSE 'subscription_per_location' END ELSE a.state END
 FROM legui_private.membership_access(b)a
$$;

CREATE FUNCTION legui_private.check_operating_terminal() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE a record;used integer;b uuid;
BEGIN
 IF NEW.principal_kind<>'employee' OR NEW.revoked_at IS NOT NULL OR NEW.expires_at<=clock_timestamp() THEN RETURN NEW;END IF;
 IF TG_OP='UPDATE' AND OLD.revoked_at IS NULL AND OLD.expires_at>clock_timestamp()
  AND OLD.last_seen_at>clock_timestamp()-interval '30 minutes'
  AND ROW(OLD.tenant_id,OLD.principal_id,OLD.principal_kind) IS NOT DISTINCT FROM ROW(NEW.tenant_id,NEW.principal_id,NEW.principal_kind) THEN RETURN NEW;END IF;
 SELECT brand_id INTO b FROM public.tenants WHERE id=NEW.tenant_id;
 IF b IS DISTINCT FROM NEW.brand_id THEN RAISE EXCEPTION 'Terminal scope mismatch' USING ERRCODE='23514'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('business-capacity:'||b::text,0));
 SELECT * INTO a FROM legui_private.membership_access(b);
 -- Account access still works when expired; operational endpoints are blocked separately.
 IF NOT a.operational OR a.tier='legacy' THEN RETURN NEW;END IF;
 SELECT count(*) INTO used FROM legui_private.auth_sessions s WHERE s.tenant_id=NEW.tenant_id AND s.principal_kind='employee'
  AND s.id<>NEW.id AND s.revoked_at IS NULL AND s.expires_at>clock_timestamp() AND s.last_seen_at>clock_timestamp()-interval '30 minutes';
 IF used>=a.terminals_per_location THEN
  RAISE EXCEPTION 'Operating terminal capacity reached' USING ERRCODE='23514',CONSTRAINT='operating_terminal_capacity';
 END IF;RETURN NEW;
END $$;
CREATE TRIGGER operating_terminal_capacity BEFORE INSERT OR UPDATE ON legui_private.auth_sessions FOR EACH ROW EXECUTE FUNCTION legui_private.check_operating_terminal();
CREATE INDEX auth_sessions_active_terminal_idx ON legui_private.auth_sessions(tenant_id,last_seen_at) WHERE principal_kind='employee' AND revoked_at IS NULL;

CREATE FUNCTION legui_private.lock_business_admin() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN
 IF NEW.role='admin' AND NEW.brand_id IS NOT NULL THEN
  PERFORM pg_advisory_xact_lock(hashtextextended('business-capacity:'||NEW.brand_id::text,0));
 END IF;RETURN NEW;
END $$;
CREATE TRIGGER business_admin_lock BEFORE INSERT OR UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION legui_private.lock_business_admin();
CREATE FUNCTION legui_private.check_business_admin() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN
 IF NEW.brand_id IS NOT NULL AND (SELECT count(*) FROM public.users WHERE brand_id=NEW.brand_id AND role='admin' AND is_active)>1 THEN
  RAISE EXCEPTION 'A membership has one restaurant administrator' USING ERRCODE='23514',CONSTRAINT='business_admin_capacity';
 END IF;RETURN NULL;
END $$;
CREATE CONSTRAINT TRIGGER business_admin_capacity AFTER INSERT OR UPDATE ON public.users DEFERRABLE INITIALLY DEFERRED
 FOR EACH ROW EXECUTE FUNCTION legui_private.check_business_admin();
-- Basic plans use manual availability. ERP balances are retained for a later upgrade.
CREATE FUNCTION legui_private.tenant_premium(t uuid) RETURNS boolean LANGUAGE sql STABLE SET search_path='' AS $$
 SELECT coalesce((SELECT a.premium AND a.operational FROM public.tenants l CROSS JOIN LATERAL legui_private.membership_access(l.brand_id)a WHERE l.id=t),false)
$$;
CREATE OR REPLACE FUNCTION legui_private.product_capacity(local_id uuid,item_id uuid) RETURNS numeric LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT CASE WHEN NOT legui_private.tenant_premium(local_id) THEN NULL WHEN tp.track_stock THEN greatest(tp.current_stock,0)::numeric
 ELSE(SELECT min(CASE WHEN s.is_active THEN greatest(floor(s.current_stock/r.quantity_used),0) ELSE 0 END) FROM public.recipes r JOIN public.supplies s ON s.id=r.supply_id AND s.tenant_id=tp.tenant_id WHERE r.product_id=tp.product_id) END
 FROM public.tenant_products tp WHERE tp.tenant_id=local_id AND tp.product_id=item_id
$$;
CREATE OR REPLACE FUNCTION legui_private.product_options(local_id uuid,item_id uuid) RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT coalesce(jsonb_agg(jsonb_build_object('id',g.id,'name',g.name,'min',g.min_selections,'max',g.max_selections,'options',coalesce((SELECT jsonb_agg(jsonb_build_object('id',e.id,'name',e.name,'price',coalesce(i.price_override,e.current_price)::text,'available',NOT legui_private.tenant_premium(local_id) OR NOT e.track_stock OR e.current_stock>0) ORDER BY i.position,e.id) FROM legui_private.product_option_items i JOIN public.extras e ON e.id=i.extra_id AND e.tenant_id=i.tenant_id AND e.is_active WHERE i.group_id=g.id),'[]'::jsonb)) ORDER BY g.position,g.id),'[]'::jsonb) FROM legui_private.product_option_groups g WHERE g.tenant_id=local_id AND g.product_id=item_id
$$;
CREATE OR REPLACE FUNCTION legui_private.product_groups_ready(local_id uuid,item_id uuid) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT NOT coalesce((SELECT options_configured FROM public.tenant_products WHERE tenant_id=local_id AND product_id=item_id),false) OR NOT EXISTS(SELECT 1 FROM legui_private.product_option_groups g WHERE g.tenant_id=local_id AND g.product_id=item_id AND g.min_selections>coalesce((SELECT sum(CASE WHEN e.track_stock AND legui_private.tenant_premium(local_id) THEN least(greatest(e.current_stock,0),5) ELSE 5 END) FROM legui_private.product_option_items i JOIN public.extras e ON e.id=i.extra_id AND e.tenant_id=i.tenant_id AND e.is_active WHERE i.group_id=g.id),0))
$$;
CREATE OR REPLACE FUNCTION legui_private.tenant_receiving(local_id uuid,at_time timestamptz DEFAULT clock_timestamp()) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT coalesce((SELECT t.active AND (SELECT operational FROM legui_private.membership_access(t.brand_id)) AND t.menu_published AND t.ordering_enabled AND b.suspended_at IS NULL AND b.closed_at IS NULL AND NOT legui_private.kitchen_full(t.id) AND NOT(coalesce(c.paused,false) AND(c.pause_until IS NULL OR c.pause_until>at_time)) AND(NOT coalesce(c.schedule_enabled,false) OR EXISTS(SELECT 1 FROM legui_private.ordering_periods p WHERE p.tenant_id=t.id AND p.day=extract(dow FROM at_time AT TIME ZONE coalesce(c.timezone,'America/Argentina/Buenos_Aires'))::integer AND p.opens<=extract(hour FROM at_time AT TIME ZONE c.timezone)::integer*60+extract(minute FROM at_time AT TIME ZONE c.timezone)::integer AND p.closes>extract(hour FROM at_time AT TIME ZONE c.timezone)::integer*60+extract(minute FROM at_time AT TIME ZONE c.timezone)::integer)) FROM public.tenants t JOIN public.brands b ON b.id=t.brand_id LEFT JOIN legui_private.tenant_ordering c ON c.tenant_id=t.id WHERE t.id=local_id),false)
$$;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
