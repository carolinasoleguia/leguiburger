-- Durable provider operations are committed BEFORE HTTP. No SQL lock spans HTTP.
CREATE TABLE legui_private.mp_connections(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),scope text NOT NULL CHECK(scope IN ('platform','business')),
 brand_id uuid REFERENCES public.brands(id),tenant_id uuid,environment text NOT NULL CHECK(environment IN ('sandbox','production')),
 state text NOT NULL DEFAULT 'disconnected' CHECK(state IN ('disconnected','connecting','connected','needs_reconnect')),
 version uuid NOT NULL DEFAULT gen_random_uuid(),credential_version uuid NOT NULL DEFAULT gen_random_uuid(),
 collector_id text,credential_cipher text,expires_at timestamptz,
 refresh_claim uuid,refresh_lease_until timestamptz,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 CHECK((scope='business')=(brand_id IS NOT NULL)),CHECK(tenant_id IS NULL OR scope='business'),
 FOREIGN KEY(brand_id,tenant_id) REFERENCES public.tenants(brand_id,id),
 CHECK(state<>'connected' OR collector_id IS NOT NULL)
);
CREATE UNIQUE INDEX mp_platform_connection ON legui_private.mp_connections(environment) WHERE scope='platform';
CREATE UNIQUE INDEX mp_brand_connection ON legui_private.mp_connections(brand_id,environment) WHERE scope='business' AND tenant_id IS NULL;
CREATE UNIQUE INDEX mp_tenant_connection ON legui_private.mp_connections(tenant_id,environment) WHERE tenant_id IS NOT NULL;
CREATE TABLE legui_private.mp_oauth_attempts(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),connection_id uuid NOT NULL REFERENCES legui_private.mp_connections(id),connection_version uuid NOT NULL,
 actor_id uuid NOT NULL REFERENCES public.users(id),session_id uuid NOT NULL REFERENCES legui_private.auth_sessions(id),
 state_digest text NOT NULL UNIQUE,binding_digest text NOT NULL,verifier_cipher text,
 status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','submitting','completed','review','superseded')),
 expires_at timestamptz NOT NULL DEFAULT clock_timestamp()+interval '10 minutes',created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE legui_private.mp_checkouts(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),purpose text NOT NULL CHECK(purpose IN ('order','invoice','subscription')),
 connection_id uuid NOT NULL REFERENCES legui_private.mp_connections(id),connection_version uuid NOT NULL,
 brand_id uuid NOT NULL REFERENCES public.brands(id),tenant_id uuid,
 order_id uuid,invoice_id uuid,subscription_id uuid,
 environment text NOT NULL CHECK(environment IN ('sandbox','production')),currency text NOT NULL CHECK(currency='ARS'),
 amount numeric(18,2) NOT NULL CHECK(amount>0 AND amount<>'NaN'::numeric),
 target_version bigint NOT NULL,interval_months integer CHECK(interval_months IN (1,12)),
 payer_email text,reference text NOT NULL UNIQUE,
 actor_kind text NOT NULL CHECK(actor_kind IN ('user','employee')),actor_id uuid NOT NULL,
 operation_key text NOT NULL CHECK(length(operation_key) BETWEEN 1 AND 160),request_digest text NOT NULL,
 state text NOT NULL DEFAULT 'prepared' CHECK(state IN ('prepared','submitting','created','review','superseded','failed')),
 claim uuid,lease_until timestamptz,submitted_at timestamptz,
 remote_id text,checkout_url text,reason text,
 reconcile_at timestamptz NOT NULL DEFAULT clock_timestamp(),reconcile_checks integer NOT NULL DEFAULT 0,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),expires_at timestamptz NOT NULL DEFAULT clock_timestamp()+interval '24 hours',
 UNIQUE(actor_kind,actor_id,purpose,operation_key),
 FOREIGN KEY(brand_id,tenant_id) REFERENCES public.tenants(brand_id,id),
 FOREIGN KEY(tenant_id,order_id) REFERENCES public.orders(tenant_id,id),
 FOREIGN KEY(invoice_id) REFERENCES legui_private.billing_invoices(id),
 FOREIGN KEY(subscription_id) REFERENCES legui_private.subscriptions(id),
 CHECK((purpose='order')=(order_id IS NOT NULL)),CHECK((purpose='invoice')=(invoice_id IS NOT NULL)),
 CHECK((purpose='subscription')=(subscription_id IS NOT NULL)),
 CHECK(state<>'created' OR (remote_id IS NOT NULL AND checkout_url IS NOT NULL AND submitted_at IS NOT NULL))
);
CREATE UNIQUE INDEX mp_checkout_remote_once ON legui_private.mp_checkouts(environment,purpose,remote_id) WHERE remote_id IS NOT NULL;
CREATE INDEX mp_checkout_reconcile_due ON legui_private.mp_checkouts(reconcile_at) WHERE state IN ('submitting','created','review');
CREATE TABLE legui_private.mp_provider_payments(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),connection_id uuid NOT NULL REFERENCES legui_private.mp_connections(id),checkout_id uuid REFERENCES legui_private.mp_checkouts(id),
 environment text NOT NULL CHECK(environment IN ('sandbox','production')),payment_id text NOT NULL,
 collector_id text NOT NULL,provider_status text NOT NULL,provider_detail text,currency text NOT NULL,
 amount numeric(18,2) NOT NULL,refunded_amount numeric(18,2) NOT NULL DEFAULT 0,
 fee_amount numeric(18,2),net_received numeric(18,2),available_at timestamptz,
 provider_updated_at timestamptz NOT NULL,paid_at timestamptz,proof_digest text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(environment,payment_id)
);
CREATE TABLE legui_private.mp_payment_observations(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),connection_id uuid NOT NULL REFERENCES legui_private.mp_connections(id),
 environment text NOT NULL,payment_id text NOT NULL,provider_updated_at timestamptz NOT NULL,
 provider_status text NOT NULL,proof_digest text NOT NULL,proof jsonb NOT NULL CHECK(jsonb_typeof(proof)='object'),
 observed_at timestamptz NOT NULL DEFAULT clock_timestamp(),UNIQUE(environment,payment_id,proof_digest)
);
CREATE TABLE legui_private.mp_payment_allocations(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),provider_payment_id uuid NOT NULL REFERENCES legui_private.mp_provider_payments(id),
 component text NOT NULL,order_payment_id uuid REFERENCES public.order_payments(id),billing_payment_id uuid REFERENCES legui_private.billing_payments(id),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 CHECK(num_nonnulls(order_payment_id,billing_payment_id)=1),UNIQUE(provider_payment_id,component),UNIQUE(order_payment_id),UNIQUE(billing_payment_id)
);
CREATE TABLE legui_private.mp_reviews(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),connection_id uuid NOT NULL REFERENCES legui_private.mp_connections(id),
 checkout_id uuid REFERENCES legui_private.mp_checkouts(id),environment text NOT NULL,resource_id text NOT NULL,
 reason text NOT NULL,proof_digest text NOT NULL,occurrences integer NOT NULL DEFAULT 1,
 first_seen_at timestamptz NOT NULL DEFAULT clock_timestamp(),last_seen_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 resolved_at timestamptz,resolved_by uuid REFERENCES public.users(id),resolution text,resolution_note text,
 CHECK((resolved_at IS NULL)=(resolution IS NULL)),CHECK(resolution IS NULL OR resolution IN ('reprocess','acknowledged')),
 CHECK(resolved_at IS NULL OR (resolved_by IS NOT NULL AND length(resolution_note)>=10))
);
CREATE UNIQUE INDEX mp_open_review ON legui_private.mp_reviews(connection_id,environment,resource_id,reason) WHERE resolved_at IS NULL;
CREATE TABLE legui_private.mp_webhook_inbox(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),connection_id uuid NOT NULL REFERENCES legui_private.mp_connections(id),
 environment text NOT NULL,event_key text NOT NULL,topic text NOT NULL,resource_id text NOT NULL,
 state text NOT NULL DEFAULT 'received' CHECK(state IN ('received','processing','processed','review','failed','ignored')),
 attempts integer NOT NULL DEFAULT 0,claim uuid,lease_until timestamptz,
 next_attempt_at timestamptz NOT NULL DEFAULT clock_timestamp(),received_at timestamptz NOT NULL DEFAULT clock_timestamp(),processed_at timestamptz,error_code text,
 UNIQUE(connection_id,event_key)
);
CREATE INDEX mp_inbox_due ON legui_private.mp_webhook_inbox(next_attempt_at) WHERE state IN ('received','failed','processing');
CREATE TABLE legui_private.mp_operations(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),connection_id uuid NOT NULL REFERENCES legui_private.mp_connections(id),
 kind text NOT NULL CHECK(kind IN ('refund','cancel_subscription')),target_id uuid NOT NULL,resource_id text NOT NULL,
 amount numeric(18,2),operation_key text NOT NULL,request_digest text NOT NULL,
 actor_kind text NOT NULL,actor_id uuid NOT NULL,
 state text NOT NULL DEFAULT 'prepared' CHECK(state IN ('prepared','submitting','completed','review','failed')),
 claim uuid,lease_until timestamptz,remote_id text,reason text,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(actor_kind,actor_id,kind,operation_key)
);
CREATE TABLE legui_private.mp_subscription_mandates(
 subscription_id uuid PRIMARY KEY REFERENCES legui_private.subscriptions(id),checkout_id uuid NOT NULL UNIQUE REFERENCES legui_private.mp_checkouts(id),
 connection_id uuid NOT NULL REFERENCES legui_private.mp_connections(id),environment text NOT NULL,
 remote_id text NOT NULL,status text NOT NULL,provider_updated_at timestamptz NOT NULL,proof_digest text NOT NULL,
 cancel_pending boolean NOT NULL DEFAULT false,cancel_at_period_end boolean NOT NULL DEFAULT false,
 UNIQUE(environment,remote_id)
);
CREATE TABLE legui_private.mp_scan_cursors(
 connection_id uuid PRIMARY KEY REFERENCES legui_private.mp_connections(id),
 scanned_until timestamptz NOT NULL DEFAULT clock_timestamp()-interval '24 hours',
 window_start timestamptz,window_end timestamptz,page_offset integer NOT NULL DEFAULT 0,
 claim uuid,lease_until timestamptz,next_scan_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE FUNCTION legui_private.guard_mp_checkout_terms() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$ BEGIN
 IF ROW(NEW.purpose,NEW.connection_id,NEW.connection_version,NEW.brand_id,NEW.tenant_id,NEW.order_id,NEW.invoice_id,NEW.subscription_id,NEW.environment,NEW.currency,NEW.amount,NEW.target_version,NEW.interval_months,NEW.payer_email,NEW.reference,NEW.actor_kind,NEW.actor_id,NEW.operation_key,NEW.request_digest,NEW.expires_at)
 IS DISTINCT FROM ROW(OLD.purpose,OLD.connection_id,OLD.connection_version,OLD.brand_id,OLD.tenant_id,OLD.order_id,OLD.invoice_id,OLD.subscription_id,OLD.environment,OLD.currency,OLD.amount,OLD.target_version,OLD.interval_months,OLD.payer_email,OLD.reference,OLD.actor_kind,OLD.actor_id,OLD.operation_key,OLD.request_digest,OLD.expires_at)
 THEN RAISE EXCEPTION 'Provider checkout terms are immutable' USING ERRCODE='23514';END IF;RETURN NEW;END $$;
CREATE TRIGGER mp_checkout_immutable_terms BEFORE UPDATE ON legui_private.mp_checkouts FOR EACH ROW EXECUTE FUNCTION legui_private.guard_mp_checkout_terms();
CREATE TRIGGER mp_observations_immutable BEFORE UPDATE OR DELETE ON legui_private.mp_payment_observations FOR EACH ROW EXECUTE FUNCTION legui_private.reject_history_change();
CREATE TRIGGER mp_allocations_immutable BEFORE UPDATE OR DELETE ON legui_private.mp_payment_allocations FOR EACH ROW EXECUTE FUNCTION legui_private.reject_history_change();
CREATE FUNCTION legui_private.guard_provider_refund_evidence() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE original_provider text;BEGIN
 IF NEW.kind='refund' THEN
  IF TG_TABLE_NAME='order_payments' THEN SELECT provider INTO original_provider FROM public.order_payments WHERE id=NEW.original_payment_id;
  ELSE SELECT provider INTO original_provider FROM legui_private.billing_payments WHERE id=NEW.original_payment_id;END IF;
  IF original_provider='mercadopago' AND NEW.provider IS DISTINCT FROM 'mercadopago' THEN
   RAISE EXCEPTION 'Mercado Pago refunds require verified provider evidence' USING ERRCODE='23514';
  END IF;
 END IF;RETURN NEW;END $$;
CREATE TRIGGER order_mp_refund_evidence BEFORE INSERT ON public.order_payments FOR EACH ROW EXECUTE FUNCTION legui_private.guard_provider_refund_evidence();
CREATE TRIGGER billing_mp_refund_evidence BEFORE INSERT ON legui_private.billing_payments FOR EACH ROW EXECUTE FUNCTION legui_private.guard_provider_refund_evidence();
DO $$ DECLARE item record; BEGIN FOR item IN SELECT tablename FROM pg_tables WHERE schemaname='legui_private' AND tablename LIKE 'mp_%' LOOP
 EXECUTE format('ALTER TABLE legui_private.%I ENABLE ROW LEVEL SECURITY',item.tablename);
END LOOP;END $$;
REVOKE ALL ON ALL TABLES IN SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
