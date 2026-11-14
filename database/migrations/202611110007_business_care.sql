ALTER TABLE public.brands ADD COLUMN closed_at timestamptz;
CREATE TABLE legui_private.support_tickets(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), brand_id uuid REFERENCES public.brands(id),
 actor_kind text NOT NULL CHECK(actor_kind IN ('user','employee')), actor_id uuid NOT NULL,
 subject text NOT NULL CHECK(length(subject) BETWEEN 4 AND 160),
 state text NOT NULL DEFAULT 'open' CHECK(state IN ('open','closed')),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX support_tickets_actor ON legui_private.support_tickets(actor_id,updated_at DESC);
CREATE TABLE legui_private.support_messages(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), ticket_id uuid NOT NULL REFERENCES legui_private.support_tickets(id),
 actor_kind text NOT NULL CHECK(actor_kind IN ('user','employee')), actor_id uuid NOT NULL,
 platform boolean NOT NULL, body text NOT NULL CHECK(length(body) BETWEEN 4 AND 6000), created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX support_messages_ticket ON legui_private.support_messages(ticket_id,created_at);
CREATE TABLE legui_private.care_receipts(
 actor_kind text NOT NULL, actor_id uuid NOT NULL, operation_key text NOT NULL, digest text NOT NULL,
 response jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(actor_kind,actor_id,operation_key)
);
CREATE TABLE legui_private.business_backups(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), brand_id uuid NOT NULL REFERENCES public.brands(id),
 actor_id uuid NOT NULL, state text NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','ready','review')),
 payload_cipher bytea, object_path text NOT NULL UNIQUE, sha256 text NOT NULL,
 report jsonb NOT NULL, attempts integer NOT NULL DEFAULT 0, claim uuid, lease_until timestamptz,
 next_attempt_at timestamptz NOT NULL DEFAULT now(), created_at timestamptz NOT NULL DEFAULT now(), verified_at timestamptz NOT NULL
);
CREATE INDEX business_backups_brand ON legui_private.business_backups(brand_id,created_at DESC);
CREATE TABLE legui_private.business_closures(
 brand_id uuid PRIMARY KEY REFERENCES public.brands(id), actor_id uuid NOT NULL,
 reason text NOT NULL CHECK(length(reason) BETWEEN 4 AND 500), created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE legui_private.runtime_errors(
 component text NOT NULL CHECK(component IN ('api','web','worker')), code text NOT NULL,
 occurrences bigint NOT NULL DEFAULT 1, first_seen_at timestamptz NOT NULL DEFAULT now(), last_seen_at timestamptz NOT NULL DEFAULT now(),
 request_id uuid, relayed_count bigint NOT NULL DEFAULT 0, relay_lease_until timestamptz, PRIMARY KEY(component,code)
);
CREATE TABLE legui_private.staged_product_images(
 object_path text PRIMARY KEY CHECK(object_path ~ '^products/[0-9a-f-]{36}/[0-9a-f-]{36}\.(png|jpg)$'),
 public_url text NOT NULL UNIQUE, brand_id uuid NOT NULL REFERENCES public.brands(id),
 created_at timestamptz NOT NULL DEFAULT now(), claim uuid, lease_until timestamptz
);
CREATE FUNCTION legui_private.guard_staged_image() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE asset legui_private.staged_product_images; BEGIN
 IF TG_OP='UPDATE' AND NEW.image_url IS NOT DISTINCT FROM OLD.image_url THEN RETURN NEW; END IF;
 SELECT * INTO asset FROM legui_private.staged_product_images WHERE public_url=NEW.image_url FOR UPDATE;
 IF FOUND AND (asset.brand_id<>NEW.brand_id OR asset.claim IS NOT NULL) THEN
  RAISE EXCEPTION 'Image publication is unavailable for this business' USING ERRCODE='23514';
 END IF; RETURN NEW;
END $$;
CREATE TRIGGER guard_staged_image BEFORE INSERT OR UPDATE ON public.products FOR EACH ROW EXECUTE FUNCTION legui_private.guard_staged_image();
-- Closed restaurants cannot be reopened through the ordinary suspension toggle.
CREATE FUNCTION legui_private.guard_closed_business() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN
 IF OLD.closed_at IS NOT NULL AND (NEW.closed_at IS DISTINCT FROM OLD.closed_at OR NEW.suspended_at IS NULL) THEN
  RAISE EXCEPTION 'Closed business requires a dedicated restoration procedure' USING ERRCODE='23514';
 END IF; RETURN NEW;
END $$;
CREATE TRIGGER guard_closed_business BEFORE UPDATE ON public.brands FOR EACH ROW EXECUTE FUNCTION legui_private.guard_closed_business();
DO $$ DECLARE n text; BEGIN
 FOREACH n IN ARRAY ARRAY['support_tickets','support_messages','care_receipts','business_backups','business_closures','runtime_errors','staged_product_images'] LOOP
  EXECUTE format('ALTER TABLE legui_private.%I ENABLE ROW LEVEL SECURITY',n);
  EXECUTE format('REVOKE ALL ON legui_private.%I FROM PUBLIC,anon,authenticated,service_role',n);
 END LOOP;
END $$;
REVOKE ALL ON FUNCTION legui_private.guard_closed_business() FROM PUBLIC,anon,authenticated,service_role;
REVOKE ALL ON FUNCTION legui_private.guard_staged_image() FROM PUBLIC,anon,authenticated,service_role;
INSERT INTO storage.buckets(id,name,public,file_size_limit,allowed_mime_types)
 VALUES('business-backups','business-backups',false,25165824,ARRAY['application/octet-stream']) ON CONFLICT(id) DO NOTHING;
DO $$ BEGIN IF EXISTS(SELECT 1 FROM storage.buckets WHERE id='business-backups' AND public) THEN RAISE EXCEPTION 'Backup bucket must be private'; END IF; END $$;
