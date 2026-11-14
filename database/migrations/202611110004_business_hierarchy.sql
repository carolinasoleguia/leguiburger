ALTER TABLE public.brands
 ADD COLUMN primary_owner_id uuid REFERENCES public.users(id),
 ADD COLUMN suspended_at timestamptz,
 ADD COLUMN suspension_reason text NOT NULL DEFAULT '',
 ADD COLUMN base_location_limit integer NOT NULL DEFAULT 1 CHECK(base_location_limit>0),
 ADD COLUMN base_collaborator_limit integer NOT NULL DEFAULT 5 CHECK(base_collaborator_limit>0);
ALTER TABLE public.tenants ADD COLUMN display_name text NOT NULL DEFAULT '',
 ADD COLUMN address text NOT NULL DEFAULT '', ADD COLUMN contact_phone text NOT NULL DEFAULT '',
 ADD COLUMN service_hours text NOT NULL DEFAULT '';

CREATE TABLE legui_private.employee_locations(
 employee_id uuid NOT NULL REFERENCES public.employees(id),tenant_id uuid NOT NULL REFERENCES public.tenants(id),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(employee_id,tenant_id)
);
INSERT INTO legui_private.employee_locations(employee_id,tenant_id)
 SELECT id,tenant_id FROM public.employees WHERE tenant_id IS NOT NULL;
CREATE TABLE legui_private.business_invitations(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),digest text NOT NULL UNIQUE CHECK(length(digest)=64),
 brand_id uuid NOT NULL REFERENCES public.brands(id),email text NOT NULL,
 purpose text NOT NULL CHECK(purpose IN ('onboarding','transfer')),
 previous_owner_id uuid REFERENCES public.users(id),invited_by uuid NOT NULL REFERENCES public.users(id),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),expires_at timestamptz NOT NULL DEFAULT clock_timestamp()+interval '48 hours',
 used_at timestamptz,revoked_at timestamptz
);
CREATE TABLE legui_private.hierarchy_receipts(
 actor_id uuid NOT NULL REFERENCES public.users(id),operation_key text NOT NULL,
 digest text NOT NULL,response jsonb NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(actor_id,operation_key)
);
DO $$ DECLARE n text; BEGIN
 FOREACH n IN ARRAY ARRAY['employee_locations','business_invitations','hierarchy_receipts'] LOOP
 EXECUTE format('ALTER TABLE legui_private.%I ENABLE ROW LEVEL SECURITY',n);
 EXECUTE format('REVOKE ALL ON legui_private.%I FROM PUBLIC,anon,authenticated,service_role',n);
 END LOOP;
END $$;

CREATE FUNCTION legui_private.business_capacity(b uuid) RETURNS TABLE(locations integer,collaborators integer,source text)
 LANGUAGE sql STABLE SET search_path='' AS $$
 WITH plan AS(SELECT p.maximum_locations,p.maximum_collaborators FROM legui_private.subscriptions s
 JOIN legui_private.billing_accounts a ON a.id=s.account_id AND a.closed_at IS NULL
 JOIN legui_private.billing_plan_prices p ON p.id=s.plan_price_id
 WHERE a.brand_id=b AND a.tenant_id IS NULL AND s.environment='production'
 AND ((s.status='active' AND s.current_period_start<=now() AND s.current_period_end>now()) OR (s.status='trialing' AND s.trial_ends_at>now()))
 ORDER BY s.started_at DESC LIMIT 1)
 SELECT p.maximum_locations,p.maximum_collaborators,'subscription'::text FROM plan p
 UNION ALL SELECT x.base_location_limit,x.base_collaborator_limit,'base'::text FROM public.brands x WHERE x.id=b AND NOT EXISTS(SELECT 1 FROM plan)
$$;
CREATE FUNCTION legui_private.check_business_capacity(b uuid,kind text,exclude_id uuid,exclude_email text) RETURNS void
 LANGUAGE plpgsql SET search_path='' AS $$
 DECLARE cap integer; used integer; BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('business-capacity:'||b::text,0));
 IF kind='location' THEN
 SELECT locations INTO cap FROM legui_private.business_capacity(b);
 SELECT count(*) INTO used FROM public.tenants WHERE brand_id=b AND active AND id IS DISTINCT FROM exclude_id;
 ELSE
 SELECT collaborators INTO cap FROM legui_private.business_capacity(b);
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
CREATE FUNCTION legui_private.guard_business_location() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
 BEGIN
 IF TG_OP='UPDATE' AND NEW.brand_id<>OLD.brand_id THEN RAISE EXCEPTION 'Location cannot change business' USING ERRCODE='23514'; END IF;
 IF NEW.active AND (TG_OP='INSERT' OR NOT OLD.active) THEN
 PERFORM legui_private.check_business_capacity(NEW.brand_id,'location',NEW.id,NULL);
 END IF;RETURN NEW;END $$;
CREATE TRIGGER business_location_capacity BEFORE INSERT OR UPDATE ON public.tenants FOR EACH ROW EXECUTE FUNCTION legui_private.guard_business_location();
CREATE FUNCTION legui_private.guard_collaborator_capacity() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
 DECLARE b uuid;BEGIN
 SELECT brand_id INTO b FROM public.tenants WHERE id=NEW.tenant_id;
 IF TG_OP='UPDATE' AND NEW.tenant_id IS DISTINCT FROM OLD.tenant_id THEN
 IF b IS DISTINCT FROM (SELECT brand_id FROM public.tenants WHERE id=OLD.tenant_id) THEN RAISE EXCEPTION 'Collaborator cannot change business' USING ERRCODE='23514'; END IF;END IF;
 IF NEW.is_active AND (TG_OP='INSERT' OR NOT OLD.is_active) THEN
 PERFORM legui_private.check_business_capacity(b,'collaborator',NEW.id,NEW.email);
 END IF;RETURN NEW;END $$;
CREATE TRIGGER collaborator_capacity BEFORE INSERT OR UPDATE ON public.employees FOR EACH ROW EXECUTE FUNCTION legui_private.guard_collaborator_capacity();
CREATE FUNCTION legui_private.guard_team_reservation() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
 DECLARE b uuid;BEGIN
 IF NEW.purpose='team_invitation' AND NEW.used_at IS NULL AND NEW.expires_at>now() THEN
 SELECT t.brand_id INTO b FROM public.tenants t JOIN public.brands x ON x.id=t.brand_id WHERE t.id=NEW.tenant_id AND t.active AND x.suspended_at IS NULL;
 IF b IS NULL THEN RAISE EXCEPTION 'Business unavailable' USING ERRCODE='23514';END IF;
 IF NOT EXISTS(SELECT 1 FROM public.employees e JOIN public.tenants t ON t.id=e.tenant_id WHERE t.brand_id=b AND e.is_active AND lower(e.email)=lower(NEW.email)) THEN
 PERFORM legui_private.check_business_capacity(b,'collaborator',NULL,NEW.email);END IF;
 END IF;RETURN NEW;END $$;
CREATE TRIGGER team_invitation_capacity BEFORE INSERT ON legui_private.auth_action_tokens FOR EACH ROW EXECUTE FUNCTION legui_private.guard_team_reservation();
CREATE FUNCTION legui_private.sync_employee_home() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
 BEGIN IF NEW.tenant_id IS NOT NULL THEN INSERT INTO legui_private.employee_locations(employee_id,tenant_id) VALUES(NEW.id,NEW.tenant_id) ON CONFLICT DO NOTHING; END IF;RETURN NULL;END $$;
CREATE TRIGGER employee_home_location AFTER INSERT OR UPDATE OF tenant_id ON public.employees FOR EACH ROW EXECUTE FUNCTION legui_private.sync_employee_home();
CREATE FUNCTION legui_private.guard_employee_location() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
 DECLARE b uuid;BEGIN
 IF TG_OP<>'DELETE' THEN
 SELECT t.brand_id INTO b FROM public.employees e JOIN public.tenants t ON t.id=e.tenant_id WHERE e.id=NEW.employee_id;
 IF b IS DISTINCT FROM (SELECT brand_id FROM public.tenants WHERE id=NEW.tenant_id) THEN RAISE EXCEPTION 'Cross-business assignment denied' USING ERRCODE='23514'; END IF;
 END IF;
 IF TG_OP='DELETE' THEN
 UPDATE legui_private.auth_sessions SET revoked_at=clock_timestamp() WHERE principal_kind='employee' AND principal_id=OLD.employee_id AND tenant_id=OLD.tenant_id AND revoked_at IS NULL;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD;END IF;RETURN NEW;END $$;
CREATE TRIGGER employee_location_scope BEFORE INSERT OR UPDATE OR DELETE ON legui_private.employee_locations FOR EACH ROW EXECUTE FUNCTION legui_private.guard_employee_location();
CREATE FUNCTION legui_private.guard_primary_owner() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
 BEGIN IF NEW.primary_owner_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM public.users WHERE id=NEW.primary_owner_id AND brand_id=NEW.id AND role='admin' AND is_active) THEN
 RAISE EXCEPTION 'Business owner must be an active administrator of this business' USING ERRCODE='23514';END IF;RETURN NEW;END $$;
CREATE TRIGGER business_primary_owner BEFORE INSERT OR UPDATE OF primary_owner_id ON public.brands FOR EACH ROW EXECUTE FUNCTION legui_private.guard_primary_owner();
CREATE FUNCTION legui_private.guard_owner_identity() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
 BEGIN IF EXISTS(SELECT 1 FROM public.brands WHERE primary_owner_id=OLD.id) AND (NEW.brand_id IS DISTINCT FROM OLD.brand_id OR NEW.role<>'admin' OR NOT NEW.is_active) THEN
 RAISE EXCEPTION 'Transfer the primary responsibility before removing this owner' USING ERRCODE='23514';END IF;RETURN NEW;END $$;
CREATE TRIGGER primary_owner_identity BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION legui_private.guard_owner_identity();
CREATE FUNCTION legui_private.suspend_business_access() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
 BEGIN IF NEW.suspended_at IS NOT NULL AND OLD.suspended_at IS NULL THEN
 UPDATE legui_private.auth_sessions SET revoked_at=clock_timestamp() WHERE brand_id=NEW.id AND principal_kind='employee' AND revoked_at IS NULL;
 UPDATE legui_private.order_links SET revoked_at=clock_timestamp() WHERE tenant_id IN(SELECT id FROM public.tenants WHERE brand_id=NEW.id) AND revoked_at IS NULL;
 UPDATE legui_private.auth_action_tokens SET used_at=clock_timestamp() WHERE purpose='team_invitation' AND tenant_id IN(SELECT id FROM public.tenants WHERE brand_id=NEW.id) AND used_at IS NULL;
 UPDATE legui_private.business_invitations SET revoked_at=clock_timestamp() WHERE brand_id=NEW.id AND used_at IS NULL AND revoked_at IS NULL;
 END IF;RETURN NULL;END $$;
CREATE TRIGGER business_suspend AFTER UPDATE OF suspended_at ON public.brands FOR EACH ROW EXECUTE FUNCTION legui_private.suspend_business_access();
CREATE FUNCTION legui_private.guard_identity_email() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
 BEGIN
 IF TG_OP='UPDATE' AND lower(NEW.email)=lower(OLD.email) THEN RETURN NEW;END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('identity-email:'||lower(NEW.email),0));
 IF EXISTS(SELECT 1 FROM public.users WHERE lower(email)=lower(NEW.email) AND (TG_TABLE_NAME<>'users' OR id<>NEW.id))
 OR EXISTS(SELECT 1 FROM public.employees WHERE lower(email)=lower(NEW.email) AND (TG_TABLE_NAME<>'employees' OR id<>NEW.id)) THEN
 RAISE EXCEPTION 'Email already belongs to an identity' USING ERRCODE='23505';END IF;
 RETURN NEW;END $$;
CREATE TRIGGER user_identity_email BEFORE INSERT OR UPDATE OF email ON public.users FOR EACH ROW EXECUTE FUNCTION legui_private.guard_identity_email();
CREATE TRIGGER employee_identity_email BEFORE INSERT OR UPDATE OF email ON public.employees FOR EACH ROW EXECUTE FUNCTION legui_private.guard_identity_email();
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
