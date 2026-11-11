-- Identity state is private. Browser database roles never receive these secrets.
CREATE TABLE legui_private.auth_sessions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 principal_kind text NOT NULL CHECK(principal_kind IN ('user','employee')),
 principal_id uuid NOT NULL,
 role text NOT NULL,
 brand_id uuid,
 tenant_id uuid,
 credential_digest text NOT NULL,
 mfa_verified boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 last_seen_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 expires_at timestamptz NOT NULL DEFAULT clock_timestamp()+interval '7 days',
 revoked_at timestamptz,
 user_agent text NOT NULL DEFAULT '',
 CHECK(expires_at>created_at)
);
CREATE INDEX auth_sessions_principal_idx ON legui_private.auth_sessions(principal_kind,principal_id);
CREATE TABLE legui_private.auth_refresh_tokens (
 digest text PRIMARY KEY CHECK(length(digest)=64),
 session_id uuid NOT NULL REFERENCES legui_private.auth_sessions(id),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 consumed_at timestamptz
);
CREATE TABLE legui_private.auth_factors (
 principal_kind text NOT NULL CHECK(principal_kind='user'), principal_id uuid NOT NULL,
 secret_cipher text NOT NULL,
 verified_at timestamptz,
 last_step bigint NOT NULL DEFAULT -1,
 PRIMARY KEY(principal_kind,principal_id)
);
CREATE TABLE legui_private.auth_recovery_codes (
 principal_id uuid NOT NULL, digest text PRIMARY KEY CHECK(length(digest)=64), used_at timestamptz
);
CREATE TABLE legui_private.auth_action_tokens (
 id uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
 digest text PRIMARY KEY CHECK(length(digest)=64),
 purpose text NOT NULL CHECK(purpose IN ('password_reset','team_invitation')),
 principal_kind text CHECK(principal_kind IN ('user','employee')), principal_id uuid,
 email text NOT NULL,
 tenant_id uuid REFERENCES public.tenants(id),
 role text CHECK(role IN ('employee','cashier','kitchen')),
 invited_by uuid REFERENCES public.users(id),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 expires_at timestamptz NOT NULL, used_at timestamptz,
 CHECK(expires_at>created_at),
 CHECK(purpose<>'team_invitation' OR (tenant_id IS NOT NULL AND role IS NOT NULL AND invited_by IS NOT NULL))
);
CREATE INDEX auth_actions_expiry_idx ON legui_private.auth_action_tokens(expires_at) WHERE used_at IS NULL;
CREATE TABLE legui_private.security_events (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 actor_kind text CHECK(actor_kind IN ('user','employee')),actor_id uuid,
 brand_id uuid REFERENCES public.brands(id),tenant_id uuid REFERENCES public.tenants(id),
 event_name text NOT NULL,target_id text,
 facts jsonb NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(facts)='object')
);
CREATE INDEX security_events_scope_idx ON legui_private.security_events(brand_id,tenant_id,occurred_at DESC);
CREATE TRIGGER security_events_immutable BEFORE UPDATE OR DELETE ON legui_private.security_events
 FOR EACH ROW EXECUTE FUNCTION legui_private.reject_history_change();
CREATE TABLE legui_private.auth_rate_limits (
 bucket text PRIMARY KEY, window_start timestamptz NOT NULL DEFAULT clock_timestamp(),attempts integer NOT NULL CHECK(attempts>0)
);
CREATE TABLE legui_private.mail_outbox (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),recipient text NOT NULL,subject text NOT NULL,
 body_cipher text,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),expires_at timestamptz NOT NULL,
 delivered_at timestamptz,attempts integer NOT NULL DEFAULT 0,next_attempt_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 CHECK(delivered_at IS NULL OR body_cipher IS NULL)
);
CREATE FUNCTION legui_private.revoke_changed_identity() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN
 IF TG_OP='DELETE' OR (TG_OP='UPDATE' AND (
  (to_jsonb(OLD)->>'password_hash') IS DISTINCT FROM (to_jsonb(NEW)->>'password_hash') OR
  (to_jsonb(OLD)->>'email') IS DISTINCT FROM (to_jsonb(NEW)->>'email') OR
  (to_jsonb(OLD)->>'role') IS DISTINCT FROM (to_jsonb(NEW)->>'role') OR
  (to_jsonb(OLD)->>'is_active') IS DISTINCT FROM (to_jsonb(NEW)->>'is_active') OR
  (to_jsonb(OLD)->>'brand_id') IS DISTINCT FROM (to_jsonb(NEW)->>'brand_id') OR
  (to_jsonb(OLD)->>'tenant_id') IS DISTINCT FROM (to_jsonb(NEW)->>'tenant_id'))) THEN
  UPDATE legui_private.auth_sessions SET revoked_at=clock_timestamp()
   WHERE principal_id=OLD.id AND principal_kind=CASE TG_TABLE_NAME WHEN 'users' THEN 'user' ELSE 'employee' END AND revoked_at IS NULL;
 END IF;
 RETURN NULL;
END $$;
CREATE TRIGGER users_revoke_sessions AFTER UPDATE OR DELETE ON public.users FOR EACH ROW EXECUTE FUNCTION legui_private.revoke_changed_identity();
CREATE TRIGGER employees_revoke_sessions AFTER UPDATE OR DELETE ON public.employees FOR EACH ROW EXECUTE FUNCTION legui_private.revoke_changed_identity();
-- Existing identities are preserved; no future employee can acquire a platform/brand role.
ALTER TABLE public.employees ADD CONSTRAINT employees_collaborator_roles CHECK(role IN ('employee','cashier','kitchen')) NOT VALID;
DO $$ DECLARE item record; BEGIN
 FOR item IN SELECT tablename FROM pg_tables WHERE schemaname='legui_private' LOOP
  EXECUTE format('ALTER TABLE legui_private.%I ENABLE ROW LEVEL SECURITY',item.tablename);
 END LOOP;
END $$;
REVOKE ALL ON ALL TABLES IN SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
