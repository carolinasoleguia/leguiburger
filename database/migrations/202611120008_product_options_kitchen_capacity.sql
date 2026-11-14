ALTER TABLE public.tenant_products ADD COLUMN options_configured boolean NOT NULL DEFAULT false;
CREATE TABLE legui_private.product_option_groups(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),tenant_id uuid NOT NULL,product_id uuid NOT NULL,name text NOT NULL CHECK(length(name) BETWEEN 2 AND 80),min_selections integer NOT NULL DEFAULT 0,max_selections integer NOT NULL DEFAULT 1,position integer NOT NULL DEFAULT 0,
 CHECK(min_selections BETWEEN 0 AND 5 AND max_selections BETWEEN greatest(min_selections,1) AND 5),UNIQUE(tenant_id,product_id,id),FOREIGN KEY(tenant_id,product_id) REFERENCES public.tenant_products(tenant_id,product_id));
CREATE TABLE legui_private.product_option_items(
 tenant_id uuid NOT NULL,product_id uuid NOT NULL,group_id uuid NOT NULL,extra_id uuid NOT NULL,price_override numeric(18,2) CHECK(price_override>=0 AND price_override<>'NaN'::numeric),position integer NOT NULL DEFAULT 0,
 PRIMARY KEY(tenant_id,product_id,extra_id),FOREIGN KEY(tenant_id,product_id,group_id) REFERENCES legui_private.product_option_groups(tenant_id,product_id,id) ON DELETE CASCADE,FOREIGN KEY(tenant_id,extra_id) REFERENCES public.extras(tenant_id,id));
ALTER TABLE public.order_detail_extras ADD COLUMN option_group_name text NOT NULL DEFAULT '' CHECK(length(option_group_name)<=80);
CREATE TRIGGER option_group_snapshot_immutable BEFORE UPDATE OF option_group_name ON public.order_detail_extras FOR EACH ROW WHEN(OLD.option_group_name IS DISTINCT FROM NEW.option_group_name) EXECUTE FUNCTION legui_private.reject_history_change();
CREATE FUNCTION legui_private.product_options(local_id uuid,item_id uuid) RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT coalesce(jsonb_agg(jsonb_build_object('id',g.id,'name',g.name,'min',g.min_selections,'max',g.max_selections,'options',coalesce((SELECT jsonb_agg(jsonb_build_object('id',e.id,'name',e.name,'price',coalesce(i.price_override,e.current_price)::text,'available',NOT e.track_stock OR e.current_stock>0) ORDER BY i.position,e.id) FROM legui_private.product_option_items i JOIN public.extras e ON e.id=i.extra_id AND e.tenant_id=i.tenant_id AND e.is_active WHERE i.group_id=g.id),'[]'::jsonb)) ORDER BY g.position,g.id),'[]'::jsonb) FROM legui_private.product_option_groups g WHERE g.tenant_id=local_id AND g.product_id=item_id
$$;
CREATE FUNCTION legui_private.product_groups_ready(local_id uuid,item_id uuid) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT NOT coalesce((SELECT options_configured FROM public.tenant_products WHERE tenant_id=local_id AND product_id=item_id),false) OR NOT EXISTS(SELECT 1 FROM legui_private.product_option_groups g WHERE g.tenant_id=local_id AND g.product_id=item_id AND g.min_selections>coalesce((SELECT sum(CASE WHEN e.track_stock THEN least(greatest(e.current_stock,0),5) ELSE 5 END) FROM legui_private.product_option_items i JOIN public.extras e ON e.id=i.extra_id AND e.tenant_id=i.tenant_id AND e.is_active WHERE i.group_id=g.id),0))
$$;
ALTER TABLE legui_private.tenant_ordering ADD COLUMN max_active_orders integer NOT NULL DEFAULT 0 CHECK(max_active_orders BETWEEN 0 AND 1000),ADD COLUMN queue_step_minutes integer NOT NULL DEFAULT 0 CHECK(queue_step_minutes BETWEEN 0 AND 30),ADD COLUMN max_queue_minutes integer NOT NULL DEFAULT 60 CHECK(max_queue_minutes BETWEEN 0 AND 180);
CREATE FUNCTION legui_private.kitchen_load(local_id uuid) RETURNS integer LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT count(*)::integer FROM public.orders o JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE o.tenant_id=local_id AND o.payment_environment='production' AND s.code IN ('pending','accepted','preparing')
$$;
CREATE FUNCTION legui_private.kitchen_full(local_id uuid) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT coalesce((SELECT c.max_active_orders>0 AND legui_private.kitchen_load(local_id)>=c.max_active_orders FROM legui_private.tenant_ordering c WHERE c.tenant_id=local_id),false)
$$;
CREATE FUNCTION legui_private.queue_delay(local_id uuid) RETURNS integer LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT coalesce((SELECT least(c.max_queue_minutes,legui_private.kitchen_load(local_id)*c.queue_step_minutes) FROM legui_private.tenant_ordering c WHERE c.tenant_id=local_id),0)
$$;
CREATE OR REPLACE FUNCTION legui_private.set_order_expectations() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $$
BEGIN
 IF NEW.payment_environment='production' THEN
  PERFORM pg_advisory_xact_lock(hashtextextended('kitchen-capacity:'||NEW.tenant_id::text,0));
  IF legui_private.kitchen_full(NEW.tenant_id) THEN RAISE EXCEPTION 'La cocina esta al maximo de su capacidad' USING ERRCODE='23514',CONSTRAINT='kitchen_capacity';END IF;
 END IF;
 IF NEW.channel='web' AND NEW.payment_environment='production' THEN
  SELECT clock_timestamp()+make_interval(mins=>coalesce(c.pending_minutes,15)),coalesce(c.prep_min,20)+legui_private.queue_delay(t.id),coalesce(c.prep_max,35)+legui_private.queue_delay(t.id) INTO NEW.pending_deadline,NEW.prep_min,NEW.prep_max FROM public.tenants t LEFT JOIN legui_private.tenant_ordering c ON c.tenant_id=t.id WHERE t.id=NEW.tenant_id;
 END IF;RETURN NEW;END$$;
CREATE OR REPLACE FUNCTION legui_private.tenant_receiving(local_id uuid,at_time timestamptz DEFAULT clock_timestamp()) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT coalesce((SELECT t.active AND t.menu_published AND t.ordering_enabled AND b.suspended_at IS NULL AND b.closed_at IS NULL AND NOT legui_private.kitchen_full(t.id) AND NOT(coalesce(c.paused,false) AND(c.pause_until IS NULL OR c.pause_until>at_time)) AND(NOT coalesce(c.schedule_enabled,false) OR EXISTS(SELECT 1 FROM legui_private.ordering_periods p WHERE p.tenant_id=t.id AND p.day=extract(dow FROM at_time AT TIME ZONE coalesce(c.timezone,'America/Argentina/Buenos_Aires'))::integer AND p.opens<=extract(hour FROM at_time AT TIME ZONE c.timezone)::integer*60+extract(minute FROM at_time AT TIME ZONE c.timezone)::integer AND p.closes>extract(hour FROM at_time AT TIME ZONE c.timezone)::integer*60+extract(minute FROM at_time AT TIME ZONE c.timezone)::integer)) FROM public.tenants t JOIN public.brands b ON b.id=t.brand_id LEFT JOIN legui_private.tenant_ordering c ON c.tenant_id=t.id WHERE t.id=local_id),false)
$$;
DO $$DECLARE n text;BEGIN FOREACH n IN ARRAY ARRAY['product_option_groups','product_option_items'] LOOP EXECUTE format('ALTER TABLE legui_private.%I ENABLE ROW LEVEL SECURITY',n);EXECUTE format('REVOKE ALL ON legui_private.%I FROM PUBLIC,anon,authenticated,service_role',n);END LOOP;END$$;
REVOKE ALL ON FUNCTION legui_private.product_options(uuid,uuid),legui_private.product_groups_ready(uuid,uuid),legui_private.kitchen_load(uuid),legui_private.kitchen_full(uuid),legui_private.queue_delay(uuid) FROM PUBLIC,anon,authenticated,service_role;
