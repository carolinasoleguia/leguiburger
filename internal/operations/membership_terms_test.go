package operations

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"testing"
)

// Older workflow suites explicitly keep their historical paid terms. These are
// rollback-only fixtures, not a bypass or a seed used by the running application.
func paidLegacyFixture(t *testing.T, tx *gorm.DB, brand string) (string, string) {
	t.Helper()
	plan, price, account, subscription := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	queries := []struct {
		q    string
		args []any
	}{
		{`INSERT INTO legui_private.billing_plans(id,code,name) VALUES(?,?,'Historical paid fixture')`, []any{plan, "fixture_" + uuid.NewString()[:8]}},
		{`INSERT INTO legui_private.billing_plan_prices(id,plan_id,currency,amount,interval_months,version,billing_unit,maximum_locations,maximum_collaborators) SELECT ?,?,'ARS',1,1,1,'brand',base_location_limit,base_collaborator_limit FROM public.brands WHERE id=?`, []any{price, plan, brand}},
		{`INSERT INTO legui_private.billing_accounts(id,brand_id,billing_unit,currency) VALUES(?,?,'brand','ARS')`, []any{account, brand}},
		{`INSERT INTO legui_private.subscriptions(id,account_id,plan_price_id,currency,interval_months,billing_unit,contracted_amount,environment,status,current_period_start,current_period_end) VALUES(?,?,?,'ARS',1,'brand',1,'production','active',now()-interval '1 day',now()+interval '28 days')`, []any{subscription, account, price}},
	}
	for _, q := range queries {
		if e := tx.Exec(q.q, q.args...).Error; e != nil {
			t.Fatal(e)
		}
	}
	return account, subscription
}

func TestCommercialPriceTerms(t *testing.T) {
	for _, tier := range []string{"standard", "premium"} {
		r := CreatePriceRequest{Currency: "ARS", Amount: "0", IntervalMonths: 1, BillingUnit: "brand", MembershipTier: tier}
		if err := normalizePriceTerms(&r); err != nil {
			t.Fatal(err)
		}
		if *r.Published {
			t.Fatal("New commercial price was published by default")
		}
		seats := 2
		if tier == "premium" {
			seats = 5
		}
		if *r.MaximumCollaborators != seats {
			t.Fatal("Wrong terminal capacity")
		}
		yes := true
		r.Published = &yes
		if err := normalizePriceTerms(&r); err == nil {
			t.Fatal("Unpriced commercial plan was published")
		}
		r.Amount = "1000"
		if tier == "premium" {
			r.ExtraLocationAmount = "250"
		}
		if err := normalizePriceTerms(&r); err != nil {
			t.Fatal(err)
		}
		r.BillingUnit = "tenant"
		if err := normalizePriceTerms(&r); err == nil {
			t.Fatal("Restaurant subscription became a per-store account")
		}
	}
	legacy := CreatePriceRequest{Amount: "100", IntervalMonths: 1, BillingUnit: "brand"}
	if err := normalizePriceTerms(&legacy); err != nil || !*legacy.Published || legacy.MembershipTier != "legacy" {
		t.Fatal("Existing API semantics changed")
	}
}
