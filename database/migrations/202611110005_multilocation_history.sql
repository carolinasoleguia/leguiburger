-- Attribution belongs to the location where the action happened, independently
-- of the employee's present home or later removal of an assignment.
CREATE TABLE legui_private.employee_location_history(
 employee_id uuid NOT NULL REFERENCES public.employees(id),
 tenant_id uuid NOT NULL REFERENCES public.tenants(id),
 first_assigned_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(employee_id,tenant_id)
);
INSERT INTO legui_private.employee_location_history(employee_id,tenant_id)
 SELECT employee_id,tenant_id FROM legui_private.employee_locations;
ALTER TABLE legui_private.employee_location_history ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON legui_private.employee_location_history FROM PUBLIC,anon,authenticated,service_role;
CREATE TRIGGER employee_location_history_immutable BEFORE UPDATE OR DELETE ON legui_private.employee_location_history
 FOR EACH ROW EXECUTE FUNCTION legui_private.reject_history_change();
CREATE FUNCTION legui_private.remember_employee_location() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
 BEGIN INSERT INTO legui_private.employee_location_history(employee_id,tenant_id) VALUES(NEW.employee_id,NEW.tenant_id) ON CONFLICT DO NOTHING;RETURN NULL;END $$;
CREATE TRIGGER employee_location_remember AFTER INSERT ON legui_private.employee_locations FOR EACH ROW EXECUTE FUNCTION legui_private.remember_employee_location();
DO $$ DECLARE r record; BEGIN
 FOR r IN SELECT conname,conrelid::regclass tab FROM pg_constraint
 WHERE contype='f' AND confrelid='public.employees'::regclass
 AND conrelid IN ('public.order_status_events'::regclass,'public.inventory_movements'::regclass,'public.activity_events'::regclass) LOOP
 EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I',r.tab,r.conname);
 END LOOP;
END $$;
ALTER TABLE public.order_status_events ADD CONSTRAINT order_actor_location_history
 FOREIGN KEY(actor_employee_id,tenant_id) REFERENCES legui_private.employee_location_history(employee_id,tenant_id);
ALTER TABLE public.inventory_movements ADD CONSTRAINT inventory_actor_location_history
 FOREIGN KEY(actor_employee_id,tenant_id) REFERENCES legui_private.employee_location_history(employee_id,tenant_id);
ALTER TABLE public.activity_events ADD CONSTRAINT activity_actor_location_history
 FOREIGN KEY(actor_employee_id,tenant_id) REFERENCES legui_private.employee_location_history(employee_id,tenant_id);
CREATE FUNCTION legui_private.check_current_actor_location() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
 BEGIN IF NEW.actor_employee_id IS NOT NULL AND NOT EXISTS(
 SELECT 1 FROM legui_private.employee_locations a JOIN public.employees e ON e.id=a.employee_id AND e.is_active
 JOIN public.tenants t ON t.id=a.tenant_id AND t.active JOIN public.brands b ON b.id=t.brand_id AND b.suspended_at IS NULL
 WHERE a.employee_id=NEW.actor_employee_id AND a.tenant_id=NEW.tenant_id) THEN
 RAISE EXCEPTION 'Employee is not assigned to this location' USING ERRCODE='23514';END IF;RETURN NEW;END $$;
CREATE TRIGGER order_actor_current BEFORE INSERT ON public.order_status_events FOR EACH ROW EXECUTE FUNCTION legui_private.check_current_actor_location();
CREATE TRIGGER inventory_actor_current BEFORE INSERT ON public.inventory_movements FOR EACH ROW EXECUTE FUNCTION legui_private.check_current_actor_location();
CREATE TRIGGER activity_actor_current BEFORE INSERT ON public.activity_events FOR EACH ROW EXECUTE FUNCTION legui_private.check_current_actor_location();
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
