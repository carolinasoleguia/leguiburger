ALTER TABLE legui_private.notifications
 ADD COLUMN brand_id uuid REFERENCES public.brands(id),
 ADD COLUMN tenant_id uuid REFERENCES public.tenants(id);
-- Bind capabilities to the same tenant/order pair, including direct SQL writes.
ALTER TABLE legui_private.order_links ADD CONSTRAINT order_link_tenant_order_fk
 FOREIGN KEY(tenant_id,order_id) REFERENCES public.orders(tenant_id,id);
