-- Keep live kitchen counts independent of archived order history.
CREATE INDEX orders_kitchen_queue_idx ON public.orders(tenant_id,order_status_id) WHERE payment_environment='production';
