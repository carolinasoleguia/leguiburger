-- Inspired by Al Toque's immutable purchase terms, receipts, environments and private billing boundary.
-- No plans/prices, subscriptions or paying accounts are fabricated by this migration.
CREATE TABLE legui_private.billing_plans(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),code text NOT NULL UNIQUE CHECK(code ~ '^[a-z][a-z0-9_]{1,49}$'),
 name text NOT NULL CHECK(length(btrim(name))>0),is_active boolean NOT NULL DEFAULT true,created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE legui_private.billing_plan_prices(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),plan_id uuid NOT NULL REFERENCES legui_private.billing_plans(id),
 currency text NOT NULL CHECK(currency ~ '^[A-Z]{3}$'),amount numeric(18,2) NOT NULL CHECK(amount>=0 AND amount<>'NaN'::numeric),
 interval_months integer NOT NULL CHECK(interval_months IN (1,12)),version integer NOT NULL CHECK(version>0),
 billing_unit text NOT NULL CHECK(billing_unit IN ('brand','tenant')),
 trial_days integer NOT NULL DEFAULT 0 CHECK(trial_days>=0),maximum_locations integer CHECK(maximum_locations>0),
 maximum_collaborators integer CHECK(maximum_collaborators>0),is_active boolean NOT NULL DEFAULT true,
 created_at timestamptz NOT NULL DEFAULT now(),UNIQUE(plan_id,currency,interval_months,billing_unit,version),UNIQUE(id,currency,interval_months,billing_unit)
);
CREATE FUNCTION legui_private.guard_plan_price() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN
 IF ROW(NEW.id,NEW.plan_id,NEW.currency,NEW.amount,NEW.interval_months,NEW.version,NEW.billing_unit,NEW.trial_days,NEW.maximum_locations,NEW.maximum_collaborators,NEW.created_at)
 IS DISTINCT FROM ROW(OLD.id,OLD.plan_id,OLD.currency,OLD.amount,OLD.interval_months,OLD.version,OLD.billing_unit,OLD.trial_days,OLD.maximum_locations,OLD.maximum_collaborators,OLD.created_at) THEN
  RAISE EXCEPTION 'Published price terms are immutable; create a new price version' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER billing_plan_prices_version_guard BEFORE UPDATE ON legui_private.billing_plan_prices
 FOR EACH ROW EXECUTE FUNCTION legui_private.guard_plan_price();

CREATE TABLE legui_private.billing_accounts(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),brand_id uuid NOT NULL REFERENCES public.brands(id),tenant_id uuid,
 billing_unit text NOT NULL CHECK(billing_unit IN ('brand','tenant')),currency text NOT NULL DEFAULT 'ARS' CHECK(currency ~ '^[A-Z]{3}$'),
 contact_email text,created_at timestamptz NOT NULL DEFAULT now(),closed_at timestamptz,
 CHECK((billing_unit='tenant')=(tenant_id IS NOT NULL)),CHECK(closed_at IS NULL OR closed_at>=created_at),
 FOREIGN KEY(brand_id,tenant_id) REFERENCES public.tenants(brand_id,id),UNIQUE(id,currency,billing_unit),UNIQUE(id,currency)
);
CREATE UNIQUE INDEX billing_accounts_brand_once ON legui_private.billing_accounts(brand_id) WHERE tenant_id IS NULL;
CREATE UNIQUE INDEX billing_accounts_tenant_once ON legui_private.billing_accounts(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE FUNCTION legui_private.guard_billing_account() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN
 IF ROW(NEW.id,NEW.brand_id,NEW.tenant_id,NEW.billing_unit,NEW.currency,NEW.created_at)
 IS DISTINCT FROM ROW(OLD.id,OLD.brand_id,OLD.tenant_id,OLD.billing_unit,OLD.currency,OLD.created_at) THEN
  RAISE EXCEPTION 'Billing account identity is immutable' USING ERRCODE='23514';
 END IF; RETURN NEW;
END $$;
CREATE TRIGGER billing_accounts_identity_guard BEFORE UPDATE ON legui_private.billing_accounts
 FOR EACH ROW EXECUTE FUNCTION legui_private.guard_billing_account();

CREATE TABLE legui_private.subscriptions(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),account_id uuid NOT NULL,plan_price_id uuid NOT NULL,
 currency text NOT NULL CHECK(currency ~ '^[A-Z]{3}$'),interval_months integer NOT NULL CHECK(interval_months IN (1,12)),
 billing_unit text NOT NULL CHECK(billing_unit IN ('brand','tenant')),contracted_amount numeric(18,2) NOT NULL CHECK(contracted_amount>=0 AND contracted_amount<>'NaN'::numeric),
 quantity integer NOT NULL DEFAULT 1 CHECK(quantity>0),
 environment text NOT NULL CHECK(environment IN ('sandbox','production')),
 status text NOT NULL CHECK(status IN ('incomplete','trialing','active','past_due','paused','canceled')),
 started_at timestamptz NOT NULL DEFAULT now(),activated_at timestamptz,trial_ends_at timestamptz,
 current_period_start timestamptz,current_period_end timestamptz,cancel_at_period_end boolean NOT NULL DEFAULT false,
 canceled_at timestamptz,ended_at timestamptz,provider text,provider_reference text,
 version bigint NOT NULL DEFAULT 1 CHECK(version>0),updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK((current_period_start IS NULL)=(current_period_end IS NULL)),
 CHECK(current_period_end IS NULL OR current_period_end>current_period_start),
 CHECK(trial_ends_at IS NULL OR trial_ends_at>started_at),CHECK(status<>'trialing' OR trial_ends_at IS NOT NULL),
 CHECK(status NOT IN ('active','past_due') OR current_period_end IS NOT NULL),
 CHECK(status<>'canceled' OR ended_at IS NOT NULL),CHECK(ended_at IS NULL OR ended_at>=started_at),
 CHECK(provider_reference IS NULL OR provider IS NOT NULL),
 FOREIGN KEY(account_id,currency,billing_unit) REFERENCES legui_private.billing_accounts(id,currency,billing_unit),
 FOREIGN KEY(plan_price_id,currency,interval_months,billing_unit) REFERENCES legui_private.billing_plan_prices(id,currency,interval_months,billing_unit),
 UNIQUE(account_id,id,environment,currency),UNIQUE(account_id,id)
);
CREATE UNIQUE INDEX subscriptions_one_current ON legui_private.subscriptions(account_id,environment) WHERE status<>'canceled';
CREATE UNIQUE INDEX subscriptions_provider_once ON legui_private.subscriptions(provider,environment,provider_reference) WHERE provider_reference IS NOT NULL;
CREATE INDEX subscriptions_expiry_idx ON legui_private.subscriptions(environment,status,current_period_end);

CREATE TABLE legui_private.subscription_events(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),account_id uuid NOT NULL,subscription_id uuid NOT NULL,
 subscription_version bigint NOT NULL,event_type text NOT NULL,
 previous_status text,status text NOT NULL,occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 actor_user_id uuid REFERENCES public.users(id),facts jsonb NOT NULL CHECK(jsonb_typeof(facts)='object'),
 UNIQUE(subscription_id,subscription_version),FOREIGN KEY(account_id,subscription_id) REFERENCES legui_private.subscriptions(account_id,id)
);
CREATE INDEX subscription_events_timeline_idx ON legui_private.subscription_events(account_id,occurred_at);
CREATE TRIGGER subscription_events_immutable BEFORE UPDATE OR DELETE ON legui_private.subscription_events
 FOR EACH ROW EXECUTE FUNCTION legui_private.reject_history_change();
CREATE FUNCTION legui_private.prepare_subscription() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN
 IF TG_OP='UPDATE' THEN
  IF ROW(NEW.id,NEW.account_id,NEW.currency,NEW.billing_unit,NEW.environment,NEW.started_at)
   IS DISTINCT FROM ROW(OLD.id,OLD.account_id,OLD.currency,OLD.billing_unit,OLD.environment,OLD.started_at) THEN
   RAISE EXCEPTION 'Subscription identity is immutable' USING ERRCODE='23514';
  END IF;
  IF OLD.status='canceled' THEN RAISE EXCEPTION 'Canceled subscription is terminal; create a new subscription' USING ERRCODE='23514'; END IF;
  IF NEW.version<>OLD.version THEN RAISE EXCEPTION 'Subscription version is maintained by the database' USING ERRCODE='23514'; END IF;
  NEW.version:=OLD.version+1;
 END IF;
 IF NEW.status='active' AND NEW.activated_at IS NULL THEN NEW.activated_at:=clock_timestamp(); END IF;
 IF NEW.status='canceled' THEN NEW.canceled_at:=coalesce(NEW.canceled_at,clock_timestamp());NEW.ended_at:=coalesce(NEW.ended_at,clock_timestamp()); END IF;
 NEW.updated_at:=clock_timestamp(); RETURN NEW;
END $$;
CREATE TRIGGER subscriptions_prepare BEFORE INSERT OR UPDATE ON legui_private.subscriptions
 FOR EACH ROW EXECUTE FUNCTION legui_private.prepare_subscription();
CREATE FUNCTION legui_private.capture_subscription() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE event_name text;old_state text;
BEGIN
 IF TG_OP='INSERT' THEN event_name:='created';
 ELSE
  old_state:=OLD.status;
  event_name:=CASE WHEN OLD.status<>NEW.status THEN NEW.status
   WHEN ROW(OLD.plan_price_id,OLD.contracted_amount,OLD.quantity) IS DISTINCT FROM ROW(NEW.plan_price_id,NEW.contracted_amount,NEW.quantity) THEN 'plan_changed'
   WHEN OLD.current_period_end IS DISTINCT FROM NEW.current_period_end THEN 'period_changed'
   WHEN OLD.cancel_at_period_end IS DISTINCT FROM NEW.cancel_at_period_end THEN 'cancellation_scheduled' ELSE 'updated' END;
 END IF;
 INSERT INTO legui_private.subscription_events(account_id,subscription_id,subscription_version,event_type,previous_status,status,actor_user_id,facts)
 VALUES(NEW.account_id,NEW.id,NEW.version,event_name,old_state,NEW.status,
  nullif(current_setting('leguiburger.actor_user_id',true),'')::uuid,
  jsonb_build_object('plan_price_id',NEW.plan_price_id,'amount',NEW.contracted_amount,'quantity',NEW.quantity,
   'currency',NEW.currency,'interval_months',NEW.interval_months,'environment',NEW.environment,
   'period_start',NEW.current_period_start,'period_end',NEW.current_period_end,'cancel_at_period_end',NEW.cancel_at_period_end));
 RETURN NEW;
END $$;
CREATE TRIGGER subscriptions_history AFTER INSERT OR UPDATE ON legui_private.subscriptions
 FOR EACH ROW EXECUTE FUNCTION legui_private.capture_subscription();

CREATE TABLE legui_private.billing_invoices(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),account_id uuid NOT NULL,subscription_id uuid,
 environment text NOT NULL CHECK(environment IN ('sandbox','production')),currency text NOT NULL CHECK(currency ~ '^[A-Z]{3}$'),
 amount_total numeric(18,2) NOT NULL CHECK(amount_total>=0 AND amount_total<>'NaN'::numeric),
 amount_paid numeric(18,2) NOT NULL DEFAULT 0 CHECK(amount_paid>=0 AND amount_paid<>'NaN'::numeric),
 credit_total numeric(18,2) NOT NULL DEFAULT 0 CHECK(credit_total>=0 AND credit_total<=amount_total AND credit_total<>'NaN'::numeric),
 balance_due numeric GENERATED ALWAYS AS(greatest(amount_total-credit_total-amount_paid,0)) STORED,
 status text NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','open','paid','credited','void','uncollectible')),
 period_start timestamptz NOT NULL,period_end timestamptz NOT NULL CHECK(period_end>period_start),
 created_at timestamptz NOT NULL DEFAULT now(),issued_at timestamptz,due_at timestamptz,
 description text,idempotency_key text NOT NULL CHECK(length(btrim(idempotency_key))>0),
 request_digest text NOT NULL CHECK(request_digest ~ '^[a-f0-9]{64}$'),
 CHECK(status='draft' OR issued_at IS NOT NULL),CHECK(due_at IS NULL OR due_at>=issued_at),
 UNIQUE(account_id,idempotency_key),UNIQUE(account_id,id,environment,currency),
 FOREIGN KEY(account_id,currency) REFERENCES legui_private.billing_accounts(id,currency),
 FOREIGN KEY(account_id,subscription_id,environment,currency) REFERENCES legui_private.subscriptions(account_id,id,environment,currency)
);
COMMENT ON TABLE legui_private.billing_invoices IS 'Commercial membership charges, NOT authorized fiscal invoices. Fiscal issuance requires its own integration and receipts.';
CREATE INDEX billing_invoices_due_idx ON legui_private.billing_invoices(account_id,environment,due_at) WHERE status IN ('open','uncollectible');
CREATE TABLE legui_private.billing_payments(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),account_id uuid NOT NULL,invoice_id uuid NOT NULL,
 environment text NOT NULL CHECK(environment IN ('sandbox','production')),currency text NOT NULL CHECK(currency ~ '^[A-Z]{3}$'),
 kind text NOT NULL CHECK(kind IN ('charge','refund')),original_payment_id uuid,
 amount numeric(18,2) NOT NULL CHECK(amount>0 AND amount<>'NaN'::numeric),
 status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','succeeded','failed','canceled')),
 method text NOT NULL CHECK(method IN ('card','transfer','cash','online','other')),
 provider text,provider_reference text,created_at timestamptz NOT NULL DEFAULT now(),settled_at timestamptz,
 idempotency_key text NOT NULL CHECK(length(btrim(idempotency_key))>0),
 request_digest text NOT NULL CHECK(request_digest ~ '^[a-f0-9]{64}$'),proof_digest text CHECK(proof_digest ~ '^[a-f0-9]{64}$'),
 CHECK((status='succeeded')=(settled_at IS NOT NULL)),CHECK((kind='refund')=(original_payment_id IS NOT NULL)),
 CHECK(original_payment_id IS DISTINCT FROM id),CHECK(provider_reference IS NULL OR provider IS NOT NULL),
 CHECK(status<>'succeeded' OR provider IS NULL OR proof_digest IS NOT NULL),
 UNIQUE(account_id,idempotency_key),UNIQUE(account_id,invoice_id,id,environment,currency),
 FOREIGN KEY(account_id,invoice_id,environment,currency) REFERENCES legui_private.billing_invoices(account_id,id,environment,currency),
 FOREIGN KEY(account_id,invoice_id,original_payment_id,environment,currency) REFERENCES legui_private.billing_payments(account_id,invoice_id,id,environment,currency)
);
CREATE UNIQUE INDEX billing_payments_provider_once ON legui_private.billing_payments(provider,environment,provider_reference) WHERE provider_reference IS NOT NULL;
CREATE INDEX billing_payments_revenue_idx ON legui_private.billing_payments(environment,settled_at,account_id) WHERE status='succeeded';
CREATE INDEX billing_payments_invoice_idx ON legui_private.billing_payments(account_id,invoice_id);
CREATE INDEX billing_payments_refund_idx ON legui_private.billing_payments(original_payment_id) WHERE original_payment_id IS NOT NULL;
CREATE TABLE legui_private.billing_credit_notes(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),account_id uuid NOT NULL,invoice_id uuid NOT NULL,
 environment text NOT NULL CHECK(environment IN ('sandbox','production')),currency text NOT NULL CHECK(currency ~ '^[A-Z]{3}$'),
 amount numeric(18,2) NOT NULL CHECK(amount>0 AND amount<>'NaN'::numeric),reason text NOT NULL CHECK(length(btrim(reason))>0),
 occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),actor_user_id uuid REFERENCES public.users(id),
 idempotency_key text NOT NULL CHECK(length(btrim(idempotency_key))>0),request_digest text NOT NULL CHECK(request_digest ~ '^[a-f0-9]{64}$'),
 UNIQUE(account_id,idempotency_key),FOREIGN KEY(account_id,invoice_id,environment,currency) REFERENCES legui_private.billing_invoices(account_id,id,environment,currency)
);
CREATE INDEX billing_credit_notes_invoice_idx ON legui_private.billing_credit_notes(account_id,invoice_id);
CREATE TRIGGER billing_credit_notes_immutable BEFORE UPDATE OR DELETE ON legui_private.billing_credit_notes
 FOR EACH ROW EXECUTE FUNCTION legui_private.reject_history_change();

CREATE FUNCTION legui_private.validate_billing_payment() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE original legui_private.billing_payments;invoice legui_private.billing_invoices;refunded numeric;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'Billing payment attempts cannot be deleted' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND (OLD.status='succeeded' OR ROW(OLD.id,OLD.account_id,OLD.invoice_id,OLD.environment,OLD.currency,OLD.kind,OLD.original_payment_id,OLD.amount,OLD.method,OLD.idempotency_key,OLD.request_digest)
 IS DISTINCT FROM ROW(NEW.id,NEW.account_id,NEW.invoice_id,NEW.environment,NEW.currency,NEW.kind,NEW.original_payment_id,NEW.amount,NEW.method,NEW.idempotency_key,NEW.request_digest)) THEN
  RAISE EXCEPTION 'Billing payment identity and successful payments are immutable' USING ERRCODE='23514';
 END IF;
 SELECT * INTO invoice FROM legui_private.billing_invoices WHERE account_id=NEW.account_id AND id=NEW.invoice_id FOR UPDATE;
 IF invoice.status IN ('draft','void') THEN RAISE EXCEPTION 'Invoice must be issued before receiving payments' USING ERRCODE='23514'; END IF;
 IF NEW.kind='refund' THEN
  SELECT * INTO original FROM legui_private.billing_payments WHERE id=NEW.original_payment_id;
  IF NOT FOUND OR original.kind<>'charge' OR original.status<>'succeeded' THEN RAISE EXCEPTION 'Refund requires a successful original membership charge' USING ERRCODE='23514'; END IF;
  SELECT coalesce(sum(amount),0) INTO refunded FROM legui_private.billing_payments WHERE original_payment_id=NEW.original_payment_id AND status='succeeded' AND id<>NEW.id;
  IF NEW.status='succeeded' AND refunded+NEW.amount>original.amount THEN RAISE EXCEPTION 'Membership refund exceeds original charge' USING ERRCODE='23514'; END IF;
 ELSIF NEW.status='succeeded' AND invoice.amount_paid+NEW.amount>invoice.amount_total-invoice.credit_total THEN
  RAISE EXCEPTION 'Membership charge exceeds invoice balance' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER billing_payments_guard BEFORE INSERT OR UPDATE OR DELETE ON legui_private.billing_payments
 FOR EACH ROW EXECUTE FUNCTION legui_private.validate_billing_payment();
CREATE FUNCTION legui_private.reconcile_billing_invoice() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE invoice legui_private.billing_invoices;paid numeric;credited numeric;next_state text;
BEGIN
 SELECT * INTO invoice FROM legui_private.billing_invoices WHERE account_id=NEW.account_id AND id=NEW.invoice_id FOR UPDATE;
 SELECT coalesce(sum(CASE kind WHEN 'charge' THEN amount ELSE -amount END),0) INTO paid FROM legui_private.billing_payments WHERE account_id=NEW.account_id AND invoice_id=NEW.invoice_id AND status='succeeded';
 SELECT coalesce(sum(amount),0) INTO credited FROM legui_private.billing_credit_notes WHERE account_id=NEW.account_id AND invoice_id=NEW.invoice_id;
 IF paid<0 OR credited>invoice.amount_total THEN RAISE EXCEPTION 'Invalid membership invoice balance' USING ERRCODE='23514'; END IF;
 IF TG_TABLE_NAME='billing_credit_notes' AND invoice.status IN ('draft','void') THEN RAISE EXCEPTION 'Credits require an issued invoice' USING ERRCODE='23514'; END IF;
 next_state:=CASE WHEN credited=invoice.amount_total AND credited>0 THEN 'credited'
  WHEN paid>=invoice.amount_total-credited THEN 'paid' ELSE 'open' END;
 UPDATE legui_private.billing_invoices SET amount_paid=paid,credit_total=credited,status=next_state WHERE id=invoice.id;
 RETURN NEW;
END $$;
CREATE TRIGGER billing_payments_reconcile AFTER INSERT OR UPDATE ON legui_private.billing_payments
 FOR EACH ROW EXECUTE FUNCTION legui_private.reconcile_billing_invoice();
CREATE TRIGGER billing_credit_notes_reconcile AFTER INSERT ON legui_private.billing_credit_notes
 FOR EACH ROW EXECUTE FUNCTION legui_private.reconcile_billing_invoice();
CREATE FUNCTION legui_private.guard_billing_invoice() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE paid numeric;credited numeric;
BEGIN
 IF TG_OP='UPDATE' THEN
  IF ROW(NEW.id,NEW.account_id,NEW.subscription_id,NEW.currency,NEW.environment,NEW.idempotency_key,NEW.request_digest)
   IS DISTINCT FROM ROW(OLD.id,OLD.account_id,OLD.subscription_id,OLD.currency,OLD.environment,OLD.idempotency_key,OLD.request_digest) THEN
   RAISE EXCEPTION 'Invoice identity is immutable' USING ERRCODE='23514';
  END IF;
  IF OLD.status<>'draft' AND ROW(NEW.amount_total,NEW.period_start,NEW.period_end,NEW.issued_at)
   IS DISTINCT FROM ROW(OLD.amount_total,OLD.period_start,OLD.period_end,OLD.issued_at) THEN
   RAISE EXCEPTION 'Issued invoice terms are immutable; issue a credit note' USING ERRCODE='23514';
  END IF;
  IF OLD.status<>'draft' AND NEW.status='draft' THEN RAISE EXCEPTION 'Issued invoice cannot return to draft' USING ERRCODE='23514'; END IF;
 END IF;
 SELECT coalesce(sum(CASE kind WHEN 'charge' THEN amount ELSE -amount END),0) INTO paid FROM legui_private.billing_payments WHERE account_id=NEW.account_id AND invoice_id=NEW.id AND status='succeeded';
 SELECT coalesce(sum(amount),0) INTO credited FROM legui_private.billing_credit_notes WHERE account_id=NEW.account_id AND invoice_id=NEW.id;
 IF NEW.amount_paid<>paid OR NEW.credit_total<>credited OR (NEW.status='void' AND paid<>0) THEN
  RAISE EXCEPTION 'Invoice balances must match payment and credit ledgers' USING ERRCODE='23514';
 END IF;RETURN NEW;
END $$;
CREATE TRIGGER billing_invoices_guard BEFORE INSERT OR UPDATE ON legui_private.billing_invoices
 FOR EACH ROW EXECUTE FUNCTION legui_private.guard_billing_invoice();

CREATE TABLE legui_private.billing_webhook_events(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),provider text NOT NULL,environment text NOT NULL CHECK(environment IN ('sandbox','production')),
 external_event_id text NOT NULL,event_type text NOT NULL,external_resource_id text,
 payload_digest text NOT NULL CHECK(payload_digest ~ '^[a-f0-9]{64}$'),
 status text NOT NULL DEFAULT 'received' CHECK(status IN ('received','processed','failed','ignored')),
 received_at timestamptz NOT NULL DEFAULT now(),processed_at timestamptz,error_code text,
 CHECK((status IN ('processed','ignored'))=(processed_at IS NOT NULL)),UNIQUE(provider,environment,external_event_id)
);
COMMENT ON TABLE legui_private.billing_webhook_events IS 'Ingress deduplication only. A webhook notification is not proof of a successful payment; verify with the provider and persist receipt proof before granting a membership.';

CREATE TABLE public.activity_events(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),brand_id uuid NOT NULL REFERENCES public.brands(id),tenant_id uuid,
 event_name text NOT NULL CHECK(event_name ~ '^[a-z][a-z0-9_.]{1,79}$'),event_version integer NOT NULL DEFAULT 1 CHECK(event_version>0),
 source text NOT NULL CHECK(source IN ('server','client','integration')),
 environment text NOT NULL DEFAULT 'production' CHECK(environment IN ('sandbox','production')),
 actor_user_id uuid REFERENCES public.users(id),actor_employee_id uuid,
 occurred_at timestamptz NOT NULL DEFAULT now(),recorded_at timestamptz NOT NULL DEFAULT now(),
 facts jsonb NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(facts)='object' AND octet_length(facts::text)<=4096),
 CHECK(num_nonnulls(actor_user_id,actor_employee_id)<=1),
 FOREIGN KEY(brand_id,tenant_id) REFERENCES public.tenants(brand_id,id),
 FOREIGN KEY(tenant_id,actor_employee_id) REFERENCES public.employees(tenant_id,id),
 CHECK(actor_employee_id IS NULL OR tenant_id IS NOT NULL)
);
CREATE INDEX activity_events_brand_period_idx ON public.activity_events(brand_id,occurred_at);
CREATE INDEX activity_events_tenant_period_idx ON public.activity_events(tenant_id,occurred_at) WHERE tenant_id IS NOT NULL;
CREATE INDEX activity_events_kind_period_idx ON public.activity_events(event_name,environment,occurred_at);
CREATE TRIGGER activity_events_immutable BEFORE UPDATE OR DELETE ON public.activity_events
 FOR EACH ROW EXECUTE FUNCTION legui_private.reject_history_change();
CREATE TABLE legui_private.platform_admin_events(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),actor_user_id uuid NOT NULL REFERENCES public.users(id),
 event_name text NOT NULL CHECK(event_name ~ '^[a-z][a-z0-9_.]{1,79}$'),target_brand_id uuid REFERENCES public.brands(id),
 occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),operation_key text NOT NULL,
 request_digest text NOT NULL CHECK(request_digest ~ '^[a-f0-9]{64}$'),facts jsonb NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(facts)='object'),
 UNIQUE(actor_user_id,operation_key)
);
CREATE FUNCTION legui_private.validate_platform_actor() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM public.users WHERE id=NEW.actor_user_id AND role IN ('owner','super_admin') AND is_active) THEN
  RAISE EXCEPTION 'Platform event requires an active platform administrator' USING ERRCODE='42501';
 END IF;RETURN NEW;
END $$;
CREATE TRIGGER platform_admin_events_actor_guard BEFORE INSERT ON legui_private.platform_admin_events
 FOR EACH ROW EXECUTE FUNCTION legui_private.validate_platform_actor();
CREATE TRIGGER platform_admin_events_immutable BEFORE UPDATE OR DELETE ON legui_private.platform_admin_events
 FOR EACH ROW EXECUTE FUNCTION legui_private.reject_history_change();

ALTER TABLE public.activity_events ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON public.activity_events FROM PUBLIC,anon,authenticated,service_role;
DO $$ DECLARE item record; BEGIN
 FOR item IN SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname='legui_private' LOOP
  EXECUTE format('ALTER TABLE legui_private.%I ENABLE ROW LEVEL SECURITY',item.tablename);
 END LOOP;
END $$;
REVOKE ALL ON ALL TABLES IN SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
REVOKE ALL ON SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
