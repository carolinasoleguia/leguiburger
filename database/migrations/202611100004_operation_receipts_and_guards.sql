-- Operation receipts make backend retries replay the original result, not repeat side effects.
CREATE TABLE legui_private.operation_receipts(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),scope text NOT NULL CHECK(scope ~ '^[a-z][a-z0-9_.]{1,79}$'),
 operation_key text NOT NULL CHECK(length(btrim(operation_key)) BETWEEN 1 AND 160),
 request_digest text NOT NULL CHECK(request_digest ~ '^[a-f0-9]{64}$'),
 actor_user_id uuid REFERENCES public.users(id),actor_employee_id uuid REFERENCES public.employees(id),
 brand_id uuid REFERENCES public.brands(id),tenant_id uuid,
 http_status integer NOT NULL CHECK(http_status BETWEEN 200 AND 299),response jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 CHECK(num_nonnulls(actor_user_id,actor_employee_id)=1),CHECK(tenant_id IS NULL OR brand_id IS NOT NULL),
 FOREIGN KEY(brand_id,tenant_id) REFERENCES public.tenants(brand_id,id)
);
CREATE UNIQUE INDEX operation_receipts_user_once ON legui_private.operation_receipts(scope,actor_user_id,operation_key) WHERE actor_user_id IS NOT NULL;
CREATE UNIQUE INDEX operation_receipts_employee_once ON legui_private.operation_receipts(scope,actor_employee_id,operation_key) WHERE actor_employee_id IS NOT NULL;
CREATE TRIGGER operation_receipts_immutable BEFORE UPDATE OR DELETE ON legui_private.operation_receipts
 FOR EACH ROW EXECUTE FUNCTION legui_private.reject_history_change();
ALTER TABLE legui_private.operation_receipts ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON legui_private.operation_receipts FROM PUBLIC,anon,authenticated,service_role;

ALTER TABLE public.sales ALTER COLUMN order_detail_id SET NOT NULL,
 ADD CONSTRAINT sales_void_reason_check CHECK((voided_at IS NULL)=(void_reason IS NULL)),
 ADD CONSTRAINT sales_finite_amounts CHECK(frozen_cost_unit<>'NaN'::numeric AND total_list_price<>'NaN'::numeric
  AND total_discount_applied<>'NaN'::numeric AND total_extras_amount<>'NaN'::numeric
  AND total_shipping_amount<>'NaN'::numeric AND total_net_revenue<>'NaN'::numeric);
COMMENT ON COLUMN public.sales.order_detail_id IS 'Required source line. Unique across sale postings; cancellations void the original row instead of deleting/reposting.';
CREATE FUNCTION legui_private.guard_sale_snapshot() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'Sales are retained; void a sale instead of deleting it' USING ERRCODE='23514'; END IF;
 IF (to_jsonb(NEW)-'voided_at'-'void_reason') IS DISTINCT FROM (to_jsonb(OLD)-'voided_at'-'void_reason')
  OR OLD.voided_at IS NOT NULL THEN
  RAISE EXCEPTION 'Sale snapshots are immutable; void a sale once with a reason' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER sales_snapshot_guard BEFORE UPDATE OR DELETE ON public.sales FOR EACH ROW EXECUTE FUNCTION legui_private.guard_sale_snapshot();
ALTER TABLE public.order_details ADD CONSTRAINT order_details_finite_cost CHECK(frozen_cost_unit IS NULL OR frozen_cost_unit<>'NaN'::numeric);
ALTER TABLE public.order_detail_extras ADD CONSTRAINT order_detail_extras_finite_cost CHECK(frozen_cost_unit IS NULL OR frozen_cost_unit<>'NaN'::numeric);
ALTER TABLE public.price_history ADD CONSTRAINT price_history_finite_price CHECK(price<>'NaN'::numeric);
ALTER TABLE public.supply_cost_history ADD CONSTRAINT supply_cost_history_finite_cost CHECK(wholesale_cost<>'NaN'::numeric);
ALTER TABLE public.operating_expenses ADD CONSTRAINT operating_expenses_finite_amount CHECK(amount<>'NaN'::numeric);
ALTER TABLE public.order_payments ADD CONSTRAINT order_payments_finite_settlement CHECK(settled_at IS NULL OR isfinite(settled_at));
ALTER TABLE legui_private.billing_payments ADD CONSTRAINT billing_payments_finite_settlement CHECK(settled_at IS NULL OR isfinite(settled_at));
ALTER TABLE legui_private.subscriptions ADD CONSTRAINT subscriptions_finite_dates CHECK(
 isfinite(started_at) AND (current_period_end IS NULL OR (isfinite(current_period_start) AND isfinite(current_period_end)))
 AND (trial_ends_at IS NULL OR isfinite(trial_ends_at)) AND (ended_at IS NULL OR isfinite(ended_at)));
ALTER TABLE legui_private.billing_invoices
 ADD CONSTRAINT billing_invoices_finite_dates CHECK(isfinite(period_start) AND isfinite(period_end) AND (issued_at IS NULL OR isfinite(issued_at)) AND (due_at IS NULL OR isfinite(due_at))),
 ADD CONSTRAINT billing_invoices_paid_status CHECK(status<>'paid' OR amount_paid>=amount_total-credit_total),
 ADD CONSTRAINT billing_invoices_credited_status CHECK(status<>'credited' OR (credit_total=amount_total AND credit_total>0));
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
