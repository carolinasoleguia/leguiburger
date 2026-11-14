ALTER TABLE public.tenant_products ADD COLUMN minimum_stock integer NOT NULL DEFAULT 0 CHECK(minimum_stock>=0);
ALTER TABLE public.extras ADD COLUMN minimum_stock integer NOT NULL DEFAULT 0 CHECK(minimum_stock>=0);
ALTER TABLE public.supplies ADD COLUMN minimum_stock numeric NOT NULL DEFAULT 0 CHECK(minimum_stock>=0);
ALTER TABLE public.tenant_products ADD COLUMN stock_version bigint NOT NULL DEFAULT 0 CHECK(stock_version>=0);
ALTER TABLE public.extras ADD COLUMN stock_version bigint NOT NULL DEFAULT 0 CHECK(stock_version>=0);
ALTER TABLE public.supplies ADD COLUMN stock_version bigint NOT NULL DEFAULT 0 CHECK(stock_version>=0);
ALTER TABLE public.price_history
 ADD CONSTRAINT price_history_dates_check CHECK(end_date IS NULL OR end_date>=start_date),
 ADD COLUMN created_by_user_id uuid REFERENCES public.users(id);
ALTER TABLE public.supply_cost_history
 ADD CONSTRAINT supply_cost_history_dates_check CHECK(end_date IS NULL OR end_date>=start_date),
 ADD COLUMN created_by_user_id uuid REFERENCES public.users(id);
CREATE UNIQUE INDEX price_history_one_open_idx ON public.price_history(tenant_id,product_id) WHERE end_date IS NULL;
CREATE UNIQUE INDEX supply_cost_history_one_open_idx ON public.supply_cost_history(tenant_id,supply_id) WHERE end_date IS NULL;
CREATE INDEX price_history_timeline_idx ON public.price_history(tenant_id,product_id,start_date DESC);
CREATE INDEX supply_cost_history_timeline_idx ON public.supply_cost_history(tenant_id,supply_id,start_date DESC);

CREATE TABLE public.inventory_movements(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),tenant_id uuid NOT NULL REFERENCES public.tenants(id),
 item_kind text NOT NULL CHECK(item_kind IN ('product','extra','supply')),item_id uuid NOT NULL,
 item_name text NOT NULL,unit text NOT NULL,quantity_delta numeric NOT NULL CHECK(quantity_delta<>0 AND quantity_delta<>'NaN'::numeric),
 balance_before numeric NOT NULL CHECK(balance_before>=0 AND balance_before<>'NaN'::numeric),
 balance_after numeric NOT NULL CHECK(balance_after>=0 AND balance_after<>'NaN'::numeric),
 balance_version bigint NOT NULL CHECK(balance_version>=0),operation_key text,
 request_digest text CHECK(request_digest ~ '^[a-f0-9]{64}$'),
 reason text NOT NULL CHECK(reason IN ('opening_balance','purchase','production','sale','return','waste','adjustment')),
 order_id uuid,production_report_id uuid,occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 actor_user_id uuid REFERENCES public.users(id),actor_employee_id uuid,
 CHECK(num_nonnulls(actor_user_id,actor_employee_id)<=1),
 CHECK(balance_before+quantity_delta=balance_after),CHECK((operation_key IS NULL)=(request_digest IS NULL)),
 CHECK(reason NOT IN ('opening_balance','purchase','return') OR quantity_delta>0),
 CHECK(reason NOT IN ('sale','waste') OR quantity_delta<0),
 UNIQUE(tenant_id,item_kind,item_id,balance_version),
 FOREIGN KEY(tenant_id,order_id) REFERENCES public.orders(tenant_id,id),
 FOREIGN KEY(tenant_id,production_report_id) REFERENCES public.production_reports(tenant_id,id),
 FOREIGN KEY(tenant_id,actor_employee_id) REFERENCES public.employees(tenant_id,id)
);
COMMENT ON COLUMN public.inventory_movements.item_id IS 'Historical entity identifier; deliberately retained even if the catalogue item is removed. Capture triggers validate current items.';
CREATE INDEX inventory_movements_timeline_idx ON public.inventory_movements(tenant_id,item_kind,item_id,occurred_at);
CREATE INDEX inventory_movements_reason_idx ON public.inventory_movements(tenant_id,reason,occurred_at);
CREATE UNIQUE INDEX inventory_movements_operation_idx ON public.inventory_movements(tenant_id,item_kind,item_id,operation_key) WHERE operation_key IS NOT NULL;
CREATE TRIGGER inventory_movements_immutable BEFORE UPDATE OR DELETE ON public.inventory_movements
 FOR EACH ROW EXECUTE FUNCTION legui_private.reject_history_change();

CREATE FUNCTION legui_private.prepare_inventory_balance() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE expected_version bigint;
BEGIN
 IF TG_OP='INSERT' THEN NEW.stock_version:=CASE WHEN NEW.current_stock=0 THEN 0 ELSE 1 END;
 ELSE
  expected_version:=nullif(current_setting('leguiburger.expected_stock_version',true),'')::bigint;
  IF expected_version IS NOT NULL AND expected_version<>OLD.stock_version THEN
   RAISE EXCEPTION 'Stock version conflict' USING ERRCODE='40001';
  END IF;
  NEW.stock_version:=OLD.stock_version+CASE WHEN NEW.current_stock IS DISTINCT FROM OLD.current_stock THEN 1 ELSE 0 END;
 END IF;
 IF NEW.current_stock='NaN'::numeric THEN RAISE EXCEPTION 'Stock must be finite' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER tenant_products_stock_version BEFORE INSERT OR UPDATE ON public.tenant_products
 FOR EACH ROW EXECUTE FUNCTION legui_private.prepare_inventory_balance();
CREATE TRIGGER extras_stock_version BEFORE INSERT OR UPDATE ON public.extras
 FOR EACH ROW EXECUTE FUNCTION legui_private.prepare_inventory_balance();
CREATE TRIGGER supplies_stock_version BEFORE INSERT OR UPDATE ON public.supplies
 FOR EACH ROW EXECUTE FUNCTION legui_private.prepare_inventory_balance();

CREATE FUNCTION legui_private.capture_inventory_balance() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE delta numeric; kind text; item uuid; item_name text; unit_name text; movement_reason text;
BEGIN
 delta:=NEW.current_stock;
 IF TG_OP='UPDATE' THEN delta:=NEW.current_stock-OLD.current_stock; END IF;
 IF delta=0 THEN RETURN NEW; END IF;
 IF TG_TABLE_NAME='tenant_products' THEN
  kind:='product';item:=NEW.product_id;unit_name:='unit';
  SELECT name INTO item_name FROM public.products WHERE id=NEW.product_id;
 ELSIF TG_TABLE_NAME='extras' THEN kind:='extra';item:=NEW.id;unit_name:='unit';item_name:=NEW.name;
 ELSE kind:='supply';item:=NEW.id;unit_name:=NEW.measurement_unit;item_name:=NEW.name;
 END IF;
 movement_reason:=coalesce(nullif(current_setting('leguiburger.inventory_reason',true),''),CASE WHEN TG_OP='INSERT' THEN 'opening_balance' ELSE 'adjustment' END);
 INSERT INTO public.inventory_movements(tenant_id,item_kind,item_id,item_name,unit,quantity_delta,balance_before,balance_after,balance_version,reason,
  operation_key,request_digest,order_id,production_report_id,actor_user_id,actor_employee_id)
 VALUES(NEW.tenant_id,kind,item,item_name,unit_name,delta,NEW.current_stock-delta,NEW.current_stock,NEW.stock_version,movement_reason,
  nullif(current_setting('leguiburger.operation_key',true),''),nullif(current_setting('leguiburger.request_digest',true),''),
  nullif(current_setting('leguiburger.source_order_id',true),'')::uuid,
  nullif(current_setting('leguiburger.source_production_report_id',true),'')::uuid,
  nullif(current_setting('leguiburger.actor_user_id',true),'')::uuid,
  nullif(current_setting('leguiburger.actor_employee_id',true),'')::uuid);
 RETURN NEW;
END $$;
CREATE TRIGGER tenant_products_stock_history AFTER INSERT OR UPDATE OF current_stock ON public.tenant_products
 FOR EACH ROW EXECUTE FUNCTION legui_private.capture_inventory_balance();
CREATE TRIGGER extras_stock_history AFTER INSERT OR UPDATE OF current_stock ON public.extras
 FOR EACH ROW EXECUTE FUNCTION legui_private.capture_inventory_balance();
CREATE TRIGGER supplies_stock_history AFTER INSERT OR UPDATE OF current_stock ON public.supplies
 FOR EACH ROW EXECUTE FUNCTION legui_private.capture_inventory_balance();

CREATE FUNCTION legui_private.record_product_price(p_tenant uuid,p_product uuid,p_price numeric) RETURNS void
 LANGUAGE plpgsql SET search_path='' AS $$
DECLARE at_time timestamptz:=clock_timestamp(); old_price numeric;
BEGIN
 SELECT price INTO old_price FROM public.price_history WHERE tenant_id=p_tenant AND product_id=p_product AND end_date IS NULL FOR UPDATE;
 IF FOUND AND old_price=p_price THEN RETURN; END IF;
 UPDATE public.price_history SET end_date=at_time WHERE tenant_id=p_tenant AND product_id=p_product AND end_date IS NULL;
 INSERT INTO public.price_history(tenant_id,product_id,price,start_date,created_by,created_by_user_id)
 VALUES(p_tenant,p_product,p_price,at_time,nullif(current_setting('leguiburger.actor_employee_id',true),'')::uuid,
  nullif(current_setting('leguiburger.actor_user_id',true),'')::uuid);
END $$;
CREATE FUNCTION legui_private.capture_local_price() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE effective_price numeric;
BEGIN
 SELECT coalesce(NEW.price_override,p.base_price) INTO effective_price FROM public.products p WHERE p.id=NEW.product_id;
 PERFORM legui_private.record_product_price(NEW.tenant_id,NEW.product_id,effective_price); RETURN NEW;
END $$;
CREATE TRIGGER tenant_products_price_history AFTER INSERT OR UPDATE OF price_override ON public.tenant_products
 FOR EACH ROW EXECUTE FUNCTION legui_private.capture_local_price();
CREATE FUNCTION legui_private.capture_base_price() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE local_record record;
BEGIN
 IF OLD.base_price IS NOT DISTINCT FROM NEW.base_price THEN RETURN NEW; END IF;
 FOR local_record IN SELECT tenant_id FROM public.tenant_products WHERE product_id=NEW.id AND price_override IS NULL ORDER BY tenant_id FOR UPDATE LOOP
  PERFORM legui_private.record_product_price(local_record.tenant_id,NEW.id,NEW.base_price);
 END LOOP;
 RETURN NEW;
END $$;
CREATE TRIGGER products_base_price_history AFTER UPDATE OF base_price ON public.products
 FOR EACH ROW EXECUTE FUNCTION legui_private.capture_base_price();

CREATE FUNCTION legui_private.capture_supply_cost() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE at_time timestamptz:=clock_timestamp();
BEGIN
 IF TG_OP='UPDATE' AND NEW.current_wholesale_cost IS NOT DISTINCT FROM OLD.current_wholesale_cost THEN RETURN NEW; END IF;
 UPDATE public.supply_cost_history SET end_date=at_time WHERE tenant_id=NEW.tenant_id AND supply_id=NEW.id AND end_date IS NULL;
 INSERT INTO public.supply_cost_history(tenant_id,supply_id,wholesale_cost,start_date,created_by,created_by_user_id)
 VALUES(NEW.tenant_id,NEW.id,NEW.current_wholesale_cost,at_time,nullif(current_setting('leguiburger.actor_employee_id',true),'')::uuid,
  nullif(current_setting('leguiburger.actor_user_id',true),'')::uuid);
 RETURN NEW;
END $$;
CREATE TRIGGER supplies_cost_history AFTER INSERT OR UPDATE OF current_wholesale_cost ON public.supplies
 FOR EACH ROW EXECUTE FUNCTION legui_private.capture_supply_cost();

-- Opening snapshots start at migration time: they do not invent past movements/prices.
INSERT INTO public.inventory_movements(tenant_id,item_kind,item_id,item_name,unit,quantity_delta,balance_before,balance_after,balance_version,reason)
 SELECT tp.tenant_id,'product',tp.product_id,p.name,'unit',tp.current_stock,0,tp.current_stock,tp.stock_version,'opening_balance'
 FROM public.tenant_products tp JOIN public.products p ON p.id=tp.product_id WHERE tp.current_stock<>0
 UNION ALL SELECT tenant_id,'extra',id,name,'unit',current_stock,0,current_stock,stock_version,'opening_balance' FROM public.extras WHERE current_stock<>0
 UNION ALL SELECT tenant_id,'supply',id,name,measurement_unit,current_stock,0,current_stock,stock_version,'opening_balance' FROM public.supplies WHERE current_stock<>0;
INSERT INTO public.price_history(tenant_id,product_id,price,start_date)
 SELECT tp.tenant_id,tp.product_id,coalesce(tp.price_override,p.base_price),clock_timestamp()
 FROM public.tenant_products tp JOIN public.products p ON p.id=tp.product_id
 WHERE NOT EXISTS(SELECT 1 FROM public.price_history h WHERE h.tenant_id=tp.tenant_id AND h.product_id=tp.product_id AND h.end_date IS NULL);
INSERT INTO public.supply_cost_history(tenant_id,supply_id,wholesale_cost,start_date)
 SELECT s.tenant_id,s.id,s.current_wholesale_cost,clock_timestamp() FROM public.supplies s
 WHERE NOT EXISTS(SELECT 1 FROM public.supply_cost_history h WHERE h.tenant_id=s.tenant_id AND h.supply_id=s.id AND h.end_date IS NULL);

ALTER TABLE public.inventory_movements ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON public.inventory_movements FROM PUBLIC,anon,authenticated,service_role;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
