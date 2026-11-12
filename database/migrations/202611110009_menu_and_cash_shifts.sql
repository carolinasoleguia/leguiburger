ALTER TABLE public.tenants ADD COLUMN menu_published boolean NOT NULL DEFAULT false;
ALTER TABLE public.products ADD COLUMN category text NOT NULL DEFAULT 'Hamburguesas' CHECK(length(btrim(category)) BETWEEN 1 AND 60);

CREATE TABLE legui_private.cash_shifts(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES public.tenants(id),
 currency text NOT NULL CHECK(currency ~ '^[A-Z]{3}$'),
 opened_at timestamptz NOT NULL DEFAULT clock_timestamp(), closed_at timestamptz,
 opening_balance numeric(18,2) NOT NULL CHECK(opening_balance>=0 AND opening_balance<>'NaN'::numeric),
 expected_balance numeric(18,2), counted_balance numeric(18,2),
 difference numeric(18,2) GENERATED ALWAYS AS (counted_balance-expected_balance) STORED,
 opened_by_user_id uuid REFERENCES public.users(id), opened_by_employee_id uuid REFERENCES public.employees(id),
 closed_by_user_id uuid REFERENCES public.users(id), closed_by_employee_id uuid REFERENCES public.employees(id),
 closing_note text NOT NULL DEFAULT '' CHECK(length(closing_note)<=1000),
 CHECK(num_nonnulls(opened_by_user_id,opened_by_employee_id)=1),
 CHECK((closed_at IS NULL AND counted_balance IS NULL AND expected_balance IS NULL AND num_nonnulls(closed_by_user_id,closed_by_employee_id)=0)
 OR (closed_at IS NOT NULL AND closed_at>=opened_at AND counted_balance IS NOT NULL AND counted_balance>=0 AND counted_balance<>'NaN'::numeric AND expected_balance IS NOT NULL AND expected_balance<>'NaN'::numeric AND num_nonnulls(closed_by_user_id,closed_by_employee_id)=1)),
 UNIQUE(tenant_id,id,currency)
);
CREATE UNIQUE INDEX cash_shift_one_open ON legui_private.cash_shifts(tenant_id,currency) WHERE closed_at IS NULL;
CREATE INDEX cash_shift_history ON legui_private.cash_shifts(tenant_id,opened_at DESC,id DESC);
CREATE TABLE legui_private.cash_entries(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL, shift_id uuid NOT NULL, currency text NOT NULL,
 kind text NOT NULL CHECK(kind IN ('charge','refund','cash_in','cash_out')),
 amount numeric(18,2) NOT NULL CHECK(amount>0 AND amount<>'NaN'::numeric),
 payment_id uuid UNIQUE REFERENCES public.order_payments(id),
 note text NOT NULL DEFAULT '' CHECK(length(note)<=1000), created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 actor_user_id uuid REFERENCES public.users(id), actor_employee_id uuid REFERENCES public.employees(id),
 FOREIGN KEY(tenant_id,shift_id,currency) REFERENCES legui_private.cash_shifts(tenant_id,id,currency),
 CHECK((kind IN ('charge','refund'))=(payment_id IS NOT NULL)),
 CHECK(kind IN ('charge','refund') OR (length(btrim(note))>=4 AND num_nonnulls(actor_user_id,actor_employee_id)=1))
);
CREATE INDEX cash_entries_shift ON legui_private.cash_entries(shift_id,created_at,id);
CREATE FUNCTION legui_private.guard_cash_shift() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE expected numeric;
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'Cash shifts cannot be deleted' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' THEN
  IF OLD.closed_at IS NOT NULL OR ROW(OLD.tenant_id,OLD.currency,OLD.opening_balance,OLD.opened_at,OLD.opened_by_user_id,OLD.opened_by_employee_id)
   IS DISTINCT FROM ROW(NEW.tenant_id,NEW.currency,NEW.opening_balance,NEW.opened_at,NEW.opened_by_user_id,NEW.opened_by_employee_id) THEN
   RAISE EXCEPTION 'Cash shift identity and closed shifts are immutable' USING ERRCODE='23514';
  END IF;
  IF NEW.closed_at IS NOT NULL THEN
   SELECT OLD.opening_balance+coalesce(sum(CASE WHEN kind IN ('charge','cash_in') THEN amount ELSE -amount END),0)
    INTO expected FROM legui_private.cash_entries WHERE shift_id=OLD.id;
   IF NEW.expected_balance IS DISTINCT FROM expected THEN RAISE EXCEPTION 'Cash closure must match its ledger' USING ERRCODE='23514'; END IF;
  END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER cash_shift_guard BEFORE UPDATE OR DELETE ON legui_private.cash_shifts FOR EACH ROW EXECUTE FUNCTION legui_private.guard_cash_shift();
CREATE FUNCTION legui_private.guard_cash_entry() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE closed timestamptz; payment public.order_payments;
BEGIN
 IF TG_OP<>'INSERT' THEN RAISE EXCEPTION 'Cash entries are immutable' USING ERRCODE='23514'; END IF;
 SELECT closed_at INTO closed FROM legui_private.cash_shifts WHERE id=NEW.shift_id AND tenant_id=NEW.tenant_id FOR UPDATE;
 IF NOT FOUND OR closed IS NOT NULL THEN RAISE EXCEPTION 'Cash shift is closed' USING ERRCODE='23514'; END IF;
 IF NEW.payment_id IS NOT NULL THEN
  SELECT * INTO payment FROM public.order_payments WHERE id=NEW.payment_id;
  IF NOT FOUND OR payment.tenant_id<>NEW.tenant_id OR payment.currency<>NEW.currency OR payment.environment<>'production'
   OR payment.method<>'cash' OR payment.status<>'succeeded' OR payment.kind<>NEW.kind OR payment.amount<>NEW.amount THEN
   RAISE EXCEPTION 'Cash entry must match its successful cash payment' USING ERRCODE='23514';
  END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER cash_entry_guard BEFORE INSERT OR UPDATE OR DELETE ON legui_private.cash_entries FOR EACH ROW EXECUTE FUNCTION legui_private.guard_cash_entry();
CREATE FUNCTION legui_private.capture_cash_payment() RETURNS trigger LANGUAGE plpgsql SET search_path='' AS $$
DECLARE shift uuid;
BEGIN
 IF NEW.method<>'cash' OR NEW.environment<>'production' OR NEW.status<>'succeeded' THEN RETURN NEW; END IF;
 IF TG_OP='UPDATE' AND OLD.status='succeeded' THEN RETURN NEW; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('cash:'||NEW.tenant_id::text||':'||NEW.currency,0));
 SELECT id INTO shift FROM legui_private.cash_shifts WHERE tenant_id=NEW.tenant_id AND currency=NEW.currency AND closed_at IS NULL FOR UPDATE;
 IF shift IS NULL THEN
  IF EXISTS(SELECT 1 FROM legui_private.cash_shifts WHERE tenant_id=NEW.tenant_id AND currency=NEW.currency) THEN
   RAISE EXCEPTION 'Open a cash shift before recording cash payments' USING ERRCODE='23514';
  END IF;
  RETURN NEW;
 END IF;
 INSERT INTO legui_private.cash_entries(tenant_id,shift_id,currency,kind,amount,payment_id,actor_user_id,actor_employee_id)
 VALUES(NEW.tenant_id,shift,NEW.currency,NEW.kind,NEW.amount,NEW.id,nullif(current_setting('leguiburger.actor_user_id',true),'')::uuid,nullif(current_setting('leguiburger.actor_employee_id',true),'')::uuid);
 RETURN NEW;
END $$;
CREATE TRIGGER cash_payment_capture AFTER INSERT OR UPDATE ON public.order_payments FOR EACH ROW EXECUTE FUNCTION legui_private.capture_cash_payment();
ALTER TABLE legui_private.cash_shifts ENABLE ROW LEVEL SECURITY;
ALTER TABLE legui_private.cash_entries ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON legui_private.cash_shifts,legui_private.cash_entries FROM PUBLIC,anon,authenticated,service_role;
REVOKE ALL ON FUNCTION legui_private.guard_cash_shift(),legui_private.guard_cash_entry(),legui_private.capture_cash_payment() FROM PUBLIC,anon,authenticated,service_role;
