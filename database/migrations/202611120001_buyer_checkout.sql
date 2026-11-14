ALTER TABLE public.tenants ADD COLUMN ordering_enabled boolean NOT NULL DEFAULT false;
ALTER TABLE public.orders
 ADD COLUMN buyer_name text NOT NULL DEFAULT '' CHECK(length(buyer_name)<=100),
 ADD COLUMN buyer_phone text NOT NULL DEFAULT '' CHECK(length(buyer_phone)<=30),
 ADD COLUMN buyer_email text NOT NULL DEFAULT '' CHECK(length(buyer_email)<=254),
 ADD COLUMN fulfillment text NOT NULL DEFAULT 'pickup' CHECK(fulfillment IN ('pickup','delivery')),
 ADD COLUMN buyer_payment_method text NOT NULL DEFAULT 'at_local' CHECK(buyer_payment_method IN ('at_local','mercadopago'));
ALTER TABLE public.order_details ADD COLUMN preparation_note text NOT NULL DEFAULT '' CHECK(length(preparation_note)<=300);
CREATE TABLE legui_private.buyer_orders(
 order_id uuid PRIMARY KEY,tenant_id uuid NOT NULL,creation_key uuid NOT NULL,
 access_digest text NOT NULL UNIQUE CHECK(access_digest ~ '^[a-f0-9]{64}$'),
 request_digest text NOT NULL CHECK(request_digest ~ '^[a-f0-9]{64}$'),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 expires_at timestamptz NOT NULL DEFAULT clock_timestamp()+interval '24 hours',
 FOREIGN KEY(tenant_id,order_id) REFERENCES public.orders(tenant_id,id),
 UNIQUE(tenant_id,creation_key),CHECK(expires_at>created_at)
);
ALTER TABLE legui_private.buyer_orders ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON legui_private.buyer_orders FROM PUBLIC,anon,authenticated,service_role;
CREATE TRIGGER buyer_orders_immutable BEFORE UPDATE OR DELETE ON legui_private.buyer_orders FOR EACH ROW EXECUTE FUNCTION legui_private.reject_history_change();
ALTER TABLE legui_private.mp_checkouts DROP CONSTRAINT mp_checkouts_actor_kind_check;
ALTER TABLE legui_private.mp_checkouts ADD CONSTRAINT mp_checkouts_actor_kind_check CHECK(actor_kind IN ('user','employee','buyer'));
ALTER TABLE legui_private.mp_checkouts ADD CONSTRAINT mp_buyer_order_actor CHECK(actor_kind<>'buyer' OR (purpose='order' AND actor_id=order_id));
