-- Shared risk state, incident evidence and one-order human verification grants.
CREATE TABLE legui_private.abuse_subjects(
 subject_hash text PRIMARY KEY CHECK(length(subject_hash)=64),
 label text NOT NULL CHECK(length(label)<=100),
 window_start timestamptz NOT NULL DEFAULT clock_timestamp(),
 new_sessions integer NOT NULL DEFAULT 0,
 invalid_requests integer NOT NULL DEFAULT 0,
 orders integer NOT NULL DEFAULT 0,
 blocked_until timestamptz,
 released_until timestamptz,
 last_seen_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE legui_private.abuse_incidents(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 subject_hash text NOT NULL REFERENCES legui_private.abuse_subjects(subject_hash),
 tenant_id uuid REFERENCES public.tenants(id) ON DELETE SET NULL,
 reason text NOT NULL CHECK(reason IN ('session_churn','invalid_requests','repeated_orders','rate_limit','captcha_failed','manual')),
 window_start timestamptz NOT NULL,
 occurrences integer NOT NULL DEFAULT 1,
 evidence jsonb NOT NULL DEFAULT '{}'::jsonb,
 first_seen_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 last_seen_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 resolved_at timestamptz,
 resolution_note text,
 UNIQUE(subject_hash,reason,window_start)
);
CREATE INDEX abuse_incidents_recent_idx ON legui_private.abuse_incidents(last_seen_at DESC,id);
CREATE INDEX abuse_incidents_pending_idx ON legui_private.abuse_incidents(last_seen_at DESC) WHERE resolved_at IS NULL;
CREATE TABLE legui_private.abuse_clearances(
 subject_hash text NOT NULL,
 session_hash text NOT NULL,
 tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
 expires_at timestamptz NOT NULL,
 used_key text,
 PRIMARY KEY(subject_hash,session_hash,tenant_id)
);
CREATE TABLE legui_private.abuse_verified_tokens(
 token_digest text PRIMARY KEY,
 expires_at timestamptz NOT NULL
);
DO $$ DECLARE n text; BEGIN
 FOREACH n IN ARRAY ARRAY['abuse_subjects','abuse_incidents','abuse_clearances','abuse_verified_tokens'] LOOP
  EXECUTE format('ALTER TABLE legui_private.%I ENABLE ROW LEVEL SECURITY',n);
  EXECUTE format('REVOKE ALL ON legui_private.%I FROM PUBLIC,anon,authenticated,service_role',n);
 END LOOP;
END $$;

CREATE INDEX abuse_subjects_retention_idx ON legui_private.abuse_subjects(last_seen_at);
CREATE INDEX abuse_clearances_expiry_idx ON legui_private.abuse_clearances(expires_at);
CREATE INDEX abuse_verified_tokens_expiry_idx ON legui_private.abuse_verified_tokens(expires_at);
