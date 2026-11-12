CREATE TABLE legui_private.delivery_settings(tenant_id uuid PRIMARY KEY REFERENCES public.tenants(id),zones_enabled boolean NOT NULL DEFAULT false);
CREATE TABLE legui_private.delivery_zones(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),tenant_id uuid NOT NULL REFERENCES public.tenants(id),shipping_method_id uuid NOT NULL,
 description text NOT NULL DEFAULT '' CHECK(length(description)<=300),minimum numeric(12,2) NOT NULL DEFAULT 0 CHECK(minimum>=0 AND minimum<>'NaN'),postal_codes text[] NOT NULL CHECK(cardinality(postal_codes) BETWEEN 1 AND 100),
 UNIQUE(tenant_id,id),UNIQUE(tenant_id,shipping_method_id),FOREIGN KEY(tenant_id,shipping_method_id) REFERENCES public.shipping_methods(tenant_id,id));
ALTER TABLE public.orders ADD COLUMN delivery_zone_id uuid,ADD COLUMN delivery_zone_name text NOT NULL DEFAULT '',ADD COLUMN delivery_postal_code text NOT NULL DEFAULT '',ADD CONSTRAINT order_delivery_zone_scope FOREIGN KEY(tenant_id,delivery_zone_id) REFERENCES legui_private.delivery_zones(tenant_id,id);
CREATE TABLE legui_private.shopper_favorites(
 shopper_id uuid NOT NULL REFERENCES legui_private.shoppers(id),tenant_id uuid NOT NULL,product_id uuid NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(shopper_id,tenant_id,product_id),FOREIGN KEY(tenant_id,product_id) REFERENCES public.tenant_products(tenant_id,product_id));
-- Capacity uses the same local recipe inputs and stock source as acceptance.
-- NULL means stock is not tracked and this local has no configured recipe.
CREATE FUNCTION legui_private.product_capacity(local_id uuid,item_id uuid) RETURNS numeric LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT CASE WHEN tp.track_stock THEN greatest(tp.current_stock,0)::numeric
 ELSE(SELECT greatest(min(CASE WHEN s.is_active THEN floor(s.current_stock/r.quantity_used) ELSE 0 END),0) FROM public.recipes r JOIN public.supplies s ON s.id=r.supply_id AND s.tenant_id=tp.tenant_id WHERE r.product_id=tp.product_id) END
 FROM public.tenant_products tp WHERE tp.tenant_id=local_id AND tp.product_id=item_id
$$;
CREATE INDEX recipe_supply_lookup ON public.recipes(supply_id,product_id);
CREATE INDEX favorites_by_tenant ON legui_private.shopper_favorites(shopper_id,tenant_id,created_at DESC);
DO $$DECLARE n text;BEGIN FOREACH n IN ARRAY ARRAY['delivery_settings','delivery_zones','shopper_favorites'] LOOP EXECUTE format('ALTER TABLE legui_private.%I ENABLE ROW LEVEL SECURITY',n);EXECUTE format('REVOKE ALL ON legui_private.%I FROM PUBLIC,anon,authenticated,service_role',n);END LOOP;END$$;
REVOKE ALL ON FUNCTION legui_private.product_capacity(uuid,uuid) FROM PUBLIC,anon,authenticated,service_role;
