-- Only order IDs leave the DB; recipients must reauthorize the public read.
CREATE FUNCTION legui_private.notify_order_stream() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $$
BEGIN
 IF TG_TABLE_NAME='orders' THEN
  PERFORM pg_notify('legui_order_changed',NEW.id::text);
 ELSE
  IF NEW.order_id IS NOT NULL THEN
   PERFORM pg_notify('legui_order_changed',NEW.order_id::text);
  END IF;
 END IF;
 RETURN NEW;
END;
$$;
REVOKE ALL ON FUNCTION legui_private.notify_order_stream() FROM PUBLIC,anon,authenticated,service_role;
CREATE TRIGGER order_stream_updated AFTER UPDATE OF order_status_id,total_paid ON public.orders FOR EACH ROW WHEN(OLD.order_status_id IS DISTINCT FROM NEW.order_status_id OR OLD.total_paid IS DISTINCT FROM NEW.total_paid) EXECUTE FUNCTION legui_private.notify_order_stream();
CREATE TRIGGER checkout_stream_updated AFTER UPDATE OF state ON legui_private.mp_checkouts FOR EACH ROW WHEN(OLD.state IS DISTINCT FROM NEW.state) EXECUTE FUNCTION legui_private.notify_order_stream();
