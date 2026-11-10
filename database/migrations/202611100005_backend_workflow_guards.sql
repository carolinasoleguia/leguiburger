-- Integration layer for the new Go operations. Previous migrations remain unchanged.
ALTER TABLE public.sales ALTER COLUMN frozen_cost_unit DROP NOT NULL;
COMMENT ON COLUMN public.sales.frozen_cost_unit IS 'Frozen unit cost at acceptance; NULL means unknown. Analytics must not claim a margin while costs are unknown.';
ALTER TABLE public.extras ADD COLUMN current_unit_cost numeric(18,2) CHECK(current_unit_cost>=0 AND current_unit_cost<>'NaN'::numeric);
ALTER TABLE public.orders ADD COLUMN version bigint NOT NULL DEFAULT 1 CHECK(version>0);
CREATE FUNCTION legui_private.increment_order_version() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN NEW.version:=OLD.version+1;RETURN NEW;END $$;
CREATE TRIGGER orders_version BEFORE UPDATE ON public.orders FOR EACH ROW EXECUTE FUNCTION legui_private.increment_order_version();

CREATE FUNCTION legui_private.guard_billing_scope() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
BEGIN
 IF NEW.tenant_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM public.tenants WHERE id=NEW.tenant_id AND brand_id=NEW.brand_id) THEN
  RAISE EXCEPTION 'Billing location does not belong to its brand' USING ERRCODE='23503';
 END IF;
 PERFORM 1 FROM public.brands WHERE id=NEW.brand_id FOR UPDATE;
 IF NEW.closed_at IS NULL AND EXISTS(SELECT 1 FROM legui_private.billing_accounts a WHERE a.brand_id=NEW.brand_id AND a.id<>NEW.id AND a.closed_at IS NULL AND (a.tenant_id IS NULL OR NEW.tenant_id IS NULL)) THEN
  RAISE EXCEPTION 'A brand cannot be billed both as a whole and per location at the same time' USING ERRCODE='23514';
 END IF;RETURN NEW;
END $$;
CREATE TRIGGER billing_accounts_scope_guard BEFORE INSERT OR UPDATE ON legui_private.billing_accounts
 FOR EACH ROW EXECUTE FUNCTION legui_private.guard_billing_scope();
CREATE UNIQUE INDEX billing_invoices_subscription_period_once ON legui_private.billing_invoices(subscription_id,environment,period_start,period_end)
 WHERE subscription_id IS NOT NULL AND status<>'void';

CREATE FUNCTION legui_private.guard_payment_parent_status() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE parent_status text;
BEGIN
 SELECT s.code INTO parent_status FROM public.orders o JOIN public.order_statuses_lookup s ON s.id=o.order_status_id
 WHERE o.id=NEW.order_id AND o.tenant_id=NEW.tenant_id FOR UPDATE OF o;
 IF NEW.kind='charge' AND parent_status IN ('canceled','rejected') THEN
  RAISE EXCEPTION 'A canceled or rejected order cannot receive charges' USING ERRCODE='23514';
 END IF;
 IF TG_OP='UPDATE' AND ROW(NEW.id,NEW.created_at,NEW.provider,NEW.provider_reference)
  IS DISTINCT FROM ROW(OLD.id,OLD.created_at,OLD.provider,OLD.provider_reference) THEN
  RAISE EXCEPTION 'Payment identity is immutable' USING ERRCODE='23514';
 END IF;RETURN NEW;
END $$;
CREATE TRIGGER order_payments_parent_guard BEFORE INSERT OR UPDATE ON public.order_payments
 FOR EACH ROW EXECUTE FUNCTION legui_private.guard_payment_parent_status();

ALTER TABLE public.operating_expenses ADD COLUMN voided_at timestamptz,ADD COLUMN void_reason text,
 ADD CONSTRAINT operating_expenses_void_check CHECK((voided_at IS NULL)=(void_reason IS NULL));
CREATE TRIGGER operating_expenses_snapshot_guard BEFORE UPDATE OR DELETE ON public.operating_expenses
 FOR EACH ROW EXECUTE FUNCTION legui_private.guard_sale_snapshot();

-- Record useful writes from existing CRUD modules too, without storing customer data or inventing past usage.
CREATE FUNCTION legui_private.capture_business_activity() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE value jsonb;tenant uuid;brand uuid;entity uuid;action text;feature text;
BEGIN
 IF TG_OP='UPDATE' AND to_jsonb(OLD) IS NOT DISTINCT FROM to_jsonb(NEW) THEN RETURN NEW; END IF;
 IF TG_OP='DELETE' THEN value:=to_jsonb(OLD);action:='deleted';ELSE value:=to_jsonb(NEW);action:=CASE WHEN TG_OP='INSERT' THEN 'created' ELSE 'updated' END;END IF;
 tenant:=nullif(value->>'tenant_id','')::uuid;
 brand:=nullif(value->>'brand_id','')::uuid;
 IF TG_TABLE_NAME='recipes' THEN SELECT s.tenant_id INTO tenant FROM public.supplies s WHERE s.id=(value->>'supply_id')::uuid; END IF;
 IF brand IS NULL AND tenant IS NOT NULL THEN SELECT t.brand_id INTO brand FROM public.tenants t WHERE t.id=tenant; END IF;
 IF brand IS NOT NULL THEN
  entity:=coalesce(nullif(value->>'id',''),nullif(value->>'product_id',''))::uuid;
  feature:=CASE TG_TABLE_NAME WHEN 'production_reports' THEN 'production' WHEN 'tenant_products' THEN 'catalog' ELSE TG_TABLE_NAME END;
  INSERT INTO public.activity_events(brand_id,tenant_id,event_name,source,actor_user_id,actor_employee_id,facts)
  VALUES(brand,tenant,feature||'.'||action,'server',nullif(current_setting('leguiburger.actor_user_id',true),'')::uuid,
   nullif(current_setting('leguiburger.actor_employee_id',true),'')::uuid,jsonb_build_object('entity_id',entity));
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD;ELSE RETURN NEW;END IF;
END $$;
CREATE TRIGGER products_activity AFTER INSERT OR UPDATE OR DELETE ON public.products FOR EACH ROW EXECUTE FUNCTION legui_private.capture_business_activity();
CREATE TRIGGER tenant_products_activity AFTER INSERT OR UPDATE OR DELETE ON public.tenant_products FOR EACH ROW EXECUTE FUNCTION legui_private.capture_business_activity();
CREATE TRIGGER production_activity AFTER INSERT OR UPDATE OR DELETE ON public.production_reports FOR EACH ROW EXECUTE FUNCTION legui_private.capture_business_activity();
CREATE TRIGGER employees_activity AFTER INSERT OR UPDATE OR DELETE ON public.employees FOR EACH ROW EXECUTE FUNCTION legui_private.capture_business_activity();
CREATE TRIGGER customers_activity AFTER INSERT OR UPDATE OR DELETE ON public.customers FOR EACH ROW EXECUTE FUNCTION legui_private.capture_business_activity();
CREATE TRIGGER supplies_activity AFTER INSERT OR UPDATE OR DELETE ON public.supplies FOR EACH ROW EXECUTE FUNCTION legui_private.capture_business_activity();
CREATE TRIGGER extras_activity AFTER INSERT OR UPDATE OR DELETE ON public.extras FOR EACH ROW EXECUTE FUNCTION legui_private.capture_business_activity();
CREATE TRIGGER recipes_activity AFTER INSERT OR UPDATE OR DELETE ON public.recipes FOR EACH ROW EXECUTE FUNCTION legui_private.capture_business_activity();
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
