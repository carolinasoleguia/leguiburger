-- Preserve manual availability when neither finished stock nor a local recipe is configured.
-- GREATEST(NULL,0) returns zero in PostgreSQL; clamp inside the aggregate instead.
CREATE OR REPLACE FUNCTION legui_private.product_capacity(local_id uuid,item_id uuid) RETURNS numeric LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT CASE WHEN tp.track_stock THEN greatest(tp.current_stock,0)::numeric
 ELSE(SELECT min(CASE WHEN s.is_active THEN greatest(floor(s.current_stock/r.quantity_used),0) ELSE 0 END) FROM public.recipes r JOIN public.supplies s ON s.id=r.supply_id AND s.tenant_id=tp.tenant_id WHERE r.product_id=tp.product_id) END
 FROM public.tenant_products tp WHERE tp.tenant_id=local_id AND tp.product_id=item_id
$$;
REVOKE ALL ON FUNCTION legui_private.product_capacity(uuid,uuid) FROM PUBLIC,anon,authenticated,service_role;
