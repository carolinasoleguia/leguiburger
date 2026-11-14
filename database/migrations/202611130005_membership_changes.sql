-- A change is a new, explicit agreement. It never rewrites a sold price/contract.
CREATE TABLE legui_private.membership_changes(
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 account_id uuid NOT NULL REFERENCES legui_private.billing_accounts(id),
 source_subscription_id uuid NOT NULL REFERENCES legui_private.subscriptions(id),
 brand_id uuid NOT NULL REFERENCES public.brands(id),
 target_price_id uuid NOT NULL REFERENCES legui_private.billing_plan_prices(id),
 environment text NOT NULL CHECK(environment IN ('sandbox','production')),
 currency text NOT NULL CHECK(currency ~ '^[A-Z]{3}$'),
 source_version bigint NOT NULL CHECK(source_version>0),
 source_locations integer NOT NULL CHECK(source_locations>0),
 purchased_locations integer NOT NULL CHECK(purchased_locations BETWEEN 1 AND 10000 AND purchased_locations>=source_locations),
 source_amount numeric(18,2) NOT NULL CHECK(source_amount>0 AND source_amount<>'NaN'::numeric),
 amount numeric(18,2) NOT NULL CHECK(amount>0 AND amount<>'NaN'::numeric),
 source_period_start timestamptz NOT NULL,
 source_period_end timestamptz NOT NULL CHECK(source_period_end>source_period_start),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 expires_at timestamptz NOT NULL CHECK(expires_at>created_at),
 actor_id uuid NOT NULL REFERENCES public.users(id),
 operation_key text NOT NULL CHECK(length(operation_key) BETWEEN 1 AND 160),
 request_digest text NOT NULL CHECK(request_digest ~ '^[a-f0-9]{64}$'),
 UNIQUE(actor_id,operation_key),
 FOREIGN KEY(account_id,source_subscription_id,environment,currency)
  REFERENCES legui_private.subscriptions(account_id,id,environment,currency)
);
ALTER TABLE legui_private.membership_changes ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON legui_private.membership_changes FROM PUBLIC,anon,authenticated,service_role;
CREATE TRIGGER membership_changes_terms_immutable BEFORE UPDATE OR DELETE ON legui_private.membership_changes
 FOR EACH ROW EXECUTE FUNCTION legui_private.reject_history_change();
CREATE INDEX membership_changes_source_idx ON legui_private.membership_changes(source_subscription_id,created_at DESC);
