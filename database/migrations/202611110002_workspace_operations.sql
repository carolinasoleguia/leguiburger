-- Private capability links, per-person notifications and operational worker health.
ALTER TABLE legui_private.mail_outbox ADD COLUMN lease_token uuid, ADD COLUMN lease_until timestamptz;
CREATE TABLE legui_private.order_links (
 digest text PRIMARY KEY CHECK(length(digest)=64),
 order_id uuid NOT NULL REFERENCES public.orders(id),
 tenant_id uuid NOT NULL REFERENCES public.tenants(id),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 expires_at timestamptz NOT NULL DEFAULT clock_timestamp()+interval '7 days',
 revoked_at timestamptz,
 CHECK(expires_at>created_at)
);
CREATE INDEX order_links_order_idx ON legui_private.order_links(order_id);
CREATE TABLE legui_private.notifications (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 principal_kind text NOT NULL CHECK(principal_kind IN ('user','employee')),
 principal_id uuid NOT NULL,
 event_key text NOT NULL,
 title text NOT NULL,
 message text NOT NULL,
 destination text NOT NULL CHECK(destination LIKE '/%' AND destination NOT LIKE '//%'),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 read_at timestamptz,
 UNIQUE(principal_kind,principal_id,event_key)
);
CREATE INDEX notifications_inbox_idx ON legui_private.notifications(principal_kind,principal_id,created_at DESC);
CREATE TABLE legui_private.onboarding_completions (
 brand_id uuid PRIMARY KEY REFERENCES public.brands(id),
 completed_by uuid NOT NULL REFERENCES public.users(id),
 completed_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE legui_private.worker_heartbeats (
 instance_id uuid PRIMARY KEY,
 started_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 last_seen_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 last_success_at timestamptz,
 failed boolean NOT NULL DEFAULT false
);
CREATE TABLE legui_private.email_changes (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 principal_kind text NOT NULL CHECK(principal_kind IN ('user','employee')),
 principal_id uuid NOT NULL,
 old_email text NOT NULL,
 new_email text NOT NULL,
 credential_digest text NOT NULL,
 old_digest text NOT NULL UNIQUE CHECK(length(old_digest)=64),
 new_digest text NOT NULL UNIQUE CHECK(length(new_digest)=64),
 old_confirmed_at timestamptz,
 new_confirmed_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 expires_at timestamptz NOT NULL DEFAULT clock_timestamp()+interval '30 minutes',
 completed_at timestamptz,
 canceled_at timestamptz,
 CHECK(old_email<>new_email)
);
DO $$ DECLARE name text; BEGIN
 FOREACH name IN ARRAY ARRAY['order_links','notifications','onboarding_completions','worker_heartbeats','email_changes'] LOOP
  EXECUTE format('ALTER TABLE legui_private.%I ENABLE ROW LEVEL SECURITY',name);
  EXECUTE format('REVOKE ALL ON legui_private.%I FROM PUBLIC,anon,authenticated,service_role',name);
 END LOOP;
END $$;
