-- Incremental foundation for restaurant analytics. Existing rows are retained.
CREATE SCHEMA IF NOT EXISTS legui_private;
REVOKE ALL ON SCHEMA legui_private FROM PUBLIC, anon, authenticated;

INSERT INTO public.order_statuses_lookup(id,code,name) VALUES
 (1,'pending','Pendiente'),(2,'accepted','Aceptado'),(3,'preparing','En preparación'),
 (4,'ready','Listo'),(5,'dispatched','En camino'),(6,'delivered','Entregado'),
 (7,'canceled','Cancelado'),(8,'rejected','Rechazado') ON CONFLICT(code) DO NOTHING;
INSERT INTO public.payment_statuses_lookup(id,code,name) VALUES
 (1,'unpaid','Sin pagar'),(2,'partially_paid','Pago parcial'),(3,'paid','Pagado'),
 (4,'refunded','Reintegrado'),(5,'partially_refunded','Reintegro parcial'),(6,'failed','Pago fallido')
 ON CONFLICT(code) DO NOTHING;

ALTER TABLE public.brands
 ADD COLUMN data_classification text NOT NULL DEFAULT 'unclassified'
  CHECK(data_classification IN ('unclassified','production','test','internal')),
 ADD COLUMN activated_at timestamptz,
 ADD COLUMN acquisition_source text;
ALTER TABLE public.tenants
 ADD COLUMN timezone text NOT NULL DEFAULT 'America/Argentina/Buenos_Aires',
 ADD COLUMN activated_at timestamptz,
 ADD CONSTRAINT tenants_brand_id_id_key UNIQUE(brand_id,id);
CREATE FUNCTION legui_private.validate_tenant_timezone() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_timezone_names WHERE name=NEW.timezone) THEN
  RAISE EXCEPTION 'Invalid tenant timezone' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER tenants_timezone_guard BEFORE INSERT OR UPDATE OF timezone ON public.tenants
 FOR EACH ROW EXECUTE FUNCTION legui_private.validate_tenant_timezone();

ALTER TABLE public.customers ADD CONSTRAINT customers_tenant_id_id_key UNIQUE(tenant_id,id);
ALTER TABLE public.employees ADD CONSTRAINT employees_tenant_id_id_key UNIQUE(tenant_id,id);
ALTER TABLE public.shipping_methods ADD CONSTRAINT shipping_methods_tenant_id_id_key UNIQUE(tenant_id,id);
ALTER TABLE public.supplies ADD CONSTRAINT supplies_tenant_id_id_key UNIQUE(tenant_id,id);
ALTER TABLE public.extras ADD CONSTRAINT extras_tenant_id_id_key UNIQUE(tenant_id,id);
ALTER TABLE public.production_reports ADD CONSTRAINT production_reports_tenant_id_id_key UNIQUE(tenant_id,id);

ALTER TABLE public.orders
 ADD COLUMN currency text NOT NULL DEFAULT 'ARS' CHECK(currency ~ '^[A-Z]{3}$'),
 ADD COLUMN payment_environment text NOT NULL DEFAULT 'production' CHECK(payment_environment IN ('sandbox','production')),
 ADD COLUMN channel text NOT NULL DEFAULT 'manual' CHECK(channel IN ('manual','counter','web','delivery','external')),
 ADD COLUMN idempotency_key text,
 ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now(),
 ADD COLUMN total_due numeric GENERATED ALWAYS AS (subtotal-total_discounts+shipping_cost) STORED,
 ADD CONSTRAINT orders_finite_amounts CHECK(subtotal<>'NaN'::numeric AND total_discounts<>'NaN'::numeric AND shipping_cost<>'NaN'::numeric AND total_paid<>'NaN'::numeric),
 ADD CONSTRAINT orders_nonnegative_due CHECK(subtotal-total_discounts+shipping_cost>=0),
 ADD CONSTRAINT orders_nonnegative_shipping CHECK(shipping_cost>=0),
 ADD CONSTRAINT orders_tenant_id_id_key UNIQUE(tenant_id,id),
 ADD CONSTRAINT orders_tenant_id_id_currency_key UNIQUE(tenant_id,id,currency),
 ADD CONSTRAINT orders_payment_scope_key UNIQUE(tenant_id,id,currency,payment_environment),
 ADD CONSTRAINT orders_customer_scope_fk FOREIGN KEY(tenant_id,customer_id)
  REFERENCES public.customers(tenant_id,id) ON DELETE SET NULL(customer_id),
 ADD CONSTRAINT orders_shipping_scope_fk FOREIGN KEY(tenant_id,shipping_method_id)
  REFERENCES public.shipping_methods(tenant_id,id) ON DELETE SET NULL(shipping_method_id);
CREATE UNIQUE INDEX orders_idempotency_key ON public.orders(tenant_id,idempotency_key) WHERE idempotency_key IS NOT NULL;
ALTER TABLE public.orders ALTER COLUMN total_paid SET DEFAULT 0,ALTER COLUMN subtotal SET DEFAULT 0,
 ALTER COLUMN total_discounts SET DEFAULT 0,ALTER COLUMN shipping_cost SET DEFAULT 0,
 ALTER COLUMN order_status_id SET DEFAULT 1,ALTER COLUMN payment_status_id SET DEFAULT 1;
CREATE INDEX orders_customer_date_idx ON public.orders(tenant_id,customer_id,created_at) WHERE customer_id IS NOT NULL;
CREATE INDEX orders_channel_date_idx ON public.orders(tenant_id,channel,created_at);

ALTER TABLE public.order_details ADD COLUMN tenant_id uuid, ADD COLUMN product_name text,
 ADD COLUMN frozen_cost_unit numeric CHECK(frozen_cost_unit>=0);
UPDATE public.order_details d SET tenant_id=o.tenant_id,product_name=coalesce(d.product_name,'Producto no disponible')
 FROM public.orders o WHERE o.id=d.order_id;
UPDATE public.order_details d SET product_name=p.name FROM public.products p WHERE p.id=d.product_id;
ALTER TABLE public.order_details ALTER COLUMN tenant_id SET NOT NULL, ALTER COLUMN product_name SET NOT NULL,
 ADD CONSTRAINT order_details_tenant_order_fk FOREIGN KEY(tenant_id,order_id) REFERENCES public.orders(tenant_id,id),
 ADD CONSTRAINT order_details_scope_key UNIQUE(tenant_id,order_id,id,product_id);
CREATE INDEX order_details_product_idx ON public.order_details(tenant_id,product_id,order_id);
ALTER TABLE public.order_details ALTER COLUMN discount_unit SET DEFAULT 0;
ALTER TABLE public.order_detail_extras ADD COLUMN quantity integer NOT NULL DEFAULT 1 CHECK(quantity>0),
 ADD COLUMN frozen_cost_unit numeric CHECK(frozen_cost_unit>=0);
ALTER TABLE public.sales ADD COLUMN order_detail_id uuid,
 ADD COLUMN voided_at timestamptz, ADD COLUMN void_reason text,
 ADD COLUMN currency text NOT NULL DEFAULT 'ARS' CHECK(currency ~ '^[A-Z]{3}$'),
 ADD COLUMN payment_environment text NOT NULL DEFAULT 'production' CHECK(payment_environment IN ('sandbox','production')),
 ADD CONSTRAINT sales_cost_nonnegative CHECK(frozen_cost_unit>=0),
 ADD CONSTRAINT sales_order_scope_fk FOREIGN KEY(tenant_id,order_id,currency) REFERENCES public.orders(tenant_id,id,currency),
 ADD CONSTRAINT sales_payment_scope_fk FOREIGN KEY(tenant_id,order_id,currency,payment_environment) REFERENCES public.orders(tenant_id,id,currency,payment_environment),
 ADD CONSTRAINT sales_detail_scope_fk FOREIGN KEY(tenant_id,order_id,order_detail_id,product_id)
  REFERENCES public.order_details(tenant_id,order_id,id,product_id),
 ADD CONSTRAINT sales_customer_scope_fk FOREIGN KEY(tenant_id,customer_id) REFERENCES public.customers(tenant_id,id),
 ADD CONSTRAINT sales_shipping_scope_fk FOREIGN KEY(tenant_id,shipping_method_id) REFERENCES public.shipping_methods(tenant_id,id);
CREATE UNIQUE INDEX sales_order_detail_once ON public.sales(order_detail_id) WHERE order_detail_id IS NOT NULL;
CREATE INDEX sales_product_date_idx ON public.sales(tenant_id,product_id,sale_date);
ALTER TABLE public.sales ALTER COLUMN total_discount_applied SET DEFAULT 0,ALTER COLUMN total_extras_amount SET DEFAULT 0,
 ALTER COLUMN total_shipping_amount SET DEFAULT 0;
COMMENT ON COLUMN public.sales.order_detail_id IS 'Set for all new sale postings; uniqueness prevents duplicate posting of a line. NULL reserved for legacy rows.';

CREATE TABLE public.order_status_events(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL, order_id uuid NOT NULL,
 previous_status_id smallint REFERENCES public.order_statuses_lookup(id),
 status_id smallint NOT NULL REFERENCES public.order_statuses_lookup(id),
 occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 actor_user_id uuid REFERENCES public.users(id), actor_employee_id uuid,
 CHECK(num_nonnulls(actor_user_id,actor_employee_id)<=1),
 FOREIGN KEY(tenant_id,order_id) REFERENCES public.orders(tenant_id,id),
 FOREIGN KEY(tenant_id,actor_employee_id) REFERENCES public.employees(tenant_id,id)
);
CREATE INDEX order_status_events_timeline_idx ON public.order_status_events(tenant_id,order_id,occurred_at);
CREATE FUNCTION legui_private.reject_history_change() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN RAISE EXCEPTION 'Audit history is append-only' USING ERRCODE='23514'; END $$;
CREATE TRIGGER order_status_events_immutable BEFORE UPDATE OR DELETE ON public.order_status_events
 FOR EACH ROW EXECUTE FUNCTION legui_private.reject_history_change();
CREATE FUNCTION legui_private.capture_order_status() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN
 IF TG_OP='INSERT' THEN
  INSERT INTO public.order_status_events(tenant_id,order_id,status_id,actor_user_id,actor_employee_id)
  VALUES(NEW.tenant_id,NEW.id,NEW.order_status_id,nullif(current_setting('leguiburger.actor_user_id',true),'')::uuid,
   nullif(current_setting('leguiburger.actor_employee_id',true),'')::uuid);
 ELSIF OLD.order_status_id IS DISTINCT FROM NEW.order_status_id THEN
  INSERT INTO public.order_status_events(tenant_id,order_id,previous_status_id,status_id,actor_user_id,actor_employee_id)
  VALUES(NEW.tenant_id,NEW.id,OLD.order_status_id,NEW.order_status_id,
   nullif(current_setting('leguiburger.actor_user_id',true),'')::uuid,
   nullif(current_setting('leguiburger.actor_employee_id',true),'')::uuid);
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER orders_status_history AFTER INSERT OR UPDATE OF order_status_id ON public.orders
 FOR EACH ROW EXECUTE FUNCTION legui_private.capture_order_status();

CREATE TABLE public.order_payments(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),tenant_id uuid NOT NULL,order_id uuid NOT NULL,
 currency text NOT NULL CHECK(currency ~ '^[A-Z]{3}$'),
 environment text NOT NULL DEFAULT 'production' CHECK(environment IN ('sandbox','production')),
 kind text NOT NULL DEFAULT 'charge' CHECK(kind IN ('charge','refund')),
 original_payment_id uuid, amount numeric(18,2) NOT NULL CHECK(amount>0 AND amount<>'NaN'::numeric),
 status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','succeeded','failed','canceled')),
 method text NOT NULL CHECK(method IN ('cash','card','transfer','online','other')),
 provider text,provider_reference text,idempotency_key text NOT NULL CHECK(length(btrim(idempotency_key))>0),
 request_digest text NOT NULL CHECK(request_digest ~ '^[a-f0-9]{64}$'),
 proof_digest text CHECK(proof_digest ~ '^[a-f0-9]{64}$'),
 created_at timestamptz NOT NULL DEFAULT now(),settled_at timestamptz,
 CHECK((status='succeeded')=(settled_at IS NOT NULL)),
 CHECK((kind='refund')=(original_payment_id IS NOT NULL)),CHECK(original_payment_id IS DISTINCT FROM id),
 CHECK((provider_reference IS NULL) OR (provider IS NOT NULL)),
 CHECK(status<>'succeeded' OR provider IS NULL OR proof_digest IS NOT NULL),
 UNIQUE(tenant_id,idempotency_key),UNIQUE(tenant_id,order_id,id,currency,environment),
 FOREIGN KEY(tenant_id,order_id,currency) REFERENCES public.orders(tenant_id,id,currency),
 FOREIGN KEY(tenant_id,order_id,currency,environment) REFERENCES public.orders(tenant_id,id,currency,payment_environment),
 FOREIGN KEY(tenant_id,order_id,original_payment_id,currency,environment) REFERENCES public.order_payments(tenant_id,order_id,id,currency,environment)
);
CREATE UNIQUE INDEX order_payments_provider_unique ON public.order_payments(provider,environment,provider_reference) WHERE provider_reference IS NOT NULL;
CREATE INDEX order_payments_settled_idx ON public.order_payments(tenant_id,settled_at) WHERE status='succeeded';
CREATE INDEX order_payments_order_idx ON public.order_payments(tenant_id,order_id);
CREATE INDEX order_payments_original_idx ON public.order_payments(original_payment_id) WHERE original_payment_id IS NOT NULL;

CREATE FUNCTION legui_private.validate_order_payment() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE original public.order_payments; refunded numeric;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'Payment attempts cannot be deleted' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND (OLD.status='succeeded' OR ROW(OLD.tenant_id,OLD.order_id,OLD.currency,OLD.environment,OLD.kind,OLD.original_payment_id,OLD.amount,OLD.method,OLD.idempotency_key,OLD.request_digest)
  IS DISTINCT FROM ROW(NEW.tenant_id,NEW.order_id,NEW.currency,NEW.environment,NEW.kind,NEW.original_payment_id,NEW.amount,NEW.method,NEW.idempotency_key,NEW.request_digest)) THEN
  RAISE EXCEPTION 'Payment identity and successful payments are immutable; record a refund' USING ERRCODE='23514';
 END IF;
 PERFORM 1 FROM public.orders WHERE tenant_id=NEW.tenant_id AND id=NEW.order_id FOR UPDATE;
 IF NEW.kind='refund' THEN
  SELECT * INTO original FROM public.order_payments WHERE id=NEW.original_payment_id;
  IF NOT FOUND OR original.kind<>'charge' OR original.status<>'succeeded' THEN
   RAISE EXCEPTION 'Refund requires a successful original charge' USING ERRCODE='23514';
  END IF;
  SELECT coalesce(sum(amount),0) INTO refunded FROM public.order_payments
   WHERE original_payment_id=NEW.original_payment_id AND status='succeeded' AND id<>NEW.id;
  IF NEW.status='succeeded' AND refunded+NEW.amount>original.amount THEN
   RAISE EXCEPTION 'Refund exceeds original charge' USING ERRCODE='23514';
  END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER order_payments_guard BEFORE INSERT OR UPDATE OR DELETE ON public.order_payments
 FOR EACH ROW EXECUTE FUNCTION legui_private.validate_order_payment();

CREATE FUNCTION legui_private.sync_order_payment_total() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE charged numeric; refunded numeric; net numeric; status_code text; due numeric;
BEGIN
 SELECT total_due INTO due FROM public.orders WHERE tenant_id=NEW.tenant_id AND id=NEW.order_id FOR UPDATE;
 SELECT coalesce(sum(amount) FILTER(WHERE kind='charge'),0),coalesce(sum(amount) FILTER(WHERE kind='refund'),0)
 INTO charged,refunded FROM public.order_payments WHERE tenant_id=NEW.tenant_id AND order_id=NEW.order_id AND status='succeeded';
 net:=charged-refunded;
 IF net<0 OR net>due THEN RAISE EXCEPTION 'Payment balance is outside the order amount' USING ERRCODE='23514'; END IF;
 status_code:=CASE WHEN refunded>0 AND net=0 THEN 'refunded' WHEN refunded>0 THEN 'partially_refunded'
  WHEN net>=due THEN 'paid' WHEN net>0 THEN 'partially_paid' ELSE 'unpaid' END;
 UPDATE public.orders SET total_paid=net,payment_status_id=(SELECT id FROM public.payment_statuses_lookup WHERE code=status_code),updated_at=clock_timestamp()
 WHERE tenant_id=NEW.tenant_id AND id=NEW.order_id;
 RETURN NEW;
END $$;
CREATE TRIGGER order_payments_reconcile AFTER INSERT OR UPDATE ON public.order_payments
 FOR EACH ROW EXECUTE FUNCTION legui_private.sync_order_payment_total();

CREATE FUNCTION legui_private.guard_order_balance() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE actual numeric;
BEGIN
 SELECT coalesce(sum(CASE kind WHEN 'charge' THEN amount ELSE -amount END),0) INTO actual
 FROM public.order_payments WHERE tenant_id=NEW.tenant_id AND order_id=NEW.id AND status='succeeded';
 IF NEW.total_paid<>actual OR actual>NEW.subtotal-NEW.total_discounts+NEW.shipping_cost THEN
  RAISE EXCEPTION 'Order paid total must match its payment ledger' USING ERRCODE='23514';
 END IF;
 NEW.updated_at:=clock_timestamp(); RETURN NEW;
END $$;
CREATE TRIGGER orders_balance_guard BEFORE INSERT OR UPDATE ON public.orders
 FOR EACH ROW EXECUTE FUNCTION legui_private.guard_order_balance();

CREATE TABLE public.operating_expenses(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),tenant_id uuid NOT NULL REFERENCES public.tenants(id),
 category text NOT NULL CHECK(category IN ('payroll','rent','utilities','processing_fees','delivery','marketing','maintenance','other')),
 amount numeric(18,2) NOT NULL CHECK(amount>0),currency text NOT NULL DEFAULT 'ARS' CHECK(currency ~ '^[A-Z]{3}$'),
 incurred_at timestamptz NOT NULL,description text,created_at timestamptz NOT NULL DEFAULT now(),
 created_by_user_id uuid REFERENCES public.users(id),idempotency_key text,UNIQUE(tenant_id,idempotency_key)
);
CREATE INDEX operating_expenses_period_idx ON public.operating_expenses(tenant_id,incurred_at,currency);
COMMENT ON TABLE public.operating_expenses IS 'Operating costs only. Ingredient purchases are not added again to profit if COGS is already taken from frozen sale costs.';

ALTER TABLE public.order_status_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.order_payments ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.operating_expenses ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON public.order_status_events,public.order_payments,public.operating_expenses FROM PUBLIC,anon,authenticated,service_role;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
