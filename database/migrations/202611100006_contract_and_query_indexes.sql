CREATE TABLE legui_private.database_contract(
 singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton),application text NOT NULL CHECK(application='leguiburger'),
 project_ref text NOT NULL CHECK(project_ref ~ '^[a-z0-9]{16,40}$'),installed_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO legui_private.database_contract(application,project_ref) VALUES('leguiburger',current_setting('leguiburger.expected_project'));
CREATE TRIGGER database_contract_immutable BEFORE UPDATE OR DELETE ON legui_private.database_contract
 FOR EACH ROW EXECUTE FUNCTION legui_private.reject_history_change();
ALTER TABLE legui_private.database_contract ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON legui_private.database_contract FROM PUBLIC,anon,authenticated,service_role;

CREATE INDEX orders_scope_period_idx ON public.orders(tenant_id,payment_environment,created_at DESC,id DESC);
CREATE INDEX order_details_order_idx ON public.order_details(order_id);
CREATE INDEX order_detail_extras_detail_idx ON public.order_detail_extras(order_detail_id);
CREATE INDEX sales_scope_period_idx ON public.sales(tenant_id,payment_environment,sale_date) WHERE voided_at IS NULL;
CREATE INDEX sales_scope_order_idx ON public.sales(tenant_id,order_id);
CREATE INDEX inventory_movements_order_idx ON public.inventory_movements(tenant_id,order_id,reason) WHERE order_id IS NOT NULL;
CREATE INDEX activity_events_usage_idx ON public.activity_events(environment,occurred_at,brand_id) WHERE source='server';
CREATE INDEX subscription_events_state_idx ON legui_private.subscription_events(subscription_id,occurred_at DESC,subscription_version DESC);
CREATE INDEX billing_accounts_brand_idx ON legui_private.billing_accounts(brand_id);
CREATE INDEX billing_invoices_account_period_idx ON legui_private.billing_invoices(account_id,environment,period_start);

CREATE OR REPLACE FUNCTION legui_private.prepare_inventory_balance() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE expected_version bigint;
BEGIN
 IF TG_OP='INSERT' THEN NEW.stock_version:=CASE WHEN NEW.current_stock=0 THEN 0 ELSE 1 END;
 ELSE
  expected_version:=nullif(current_setting('leguiburger.expected_stock_version',true),'')::bigint;
  IF NEW.current_stock IS DISTINCT FROM OLD.current_stock AND expected_version IS NOT NULL AND expected_version<>OLD.stock_version THEN
   RAISE EXCEPTION 'Stock version conflict' USING ERRCODE='40001';
  END IF;
  NEW.stock_version:=OLD.stock_version+CASE WHEN NEW.current_stock IS DISTINCT FROM OLD.current_stock THEN 1 ELSE 0 END;
 END IF;
 IF NEW.current_stock='NaN'::numeric THEN RAISE EXCEPTION 'Stock must be finite' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
