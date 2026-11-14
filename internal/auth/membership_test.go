package auth

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestCommercialDataBoundary(t *testing.T) {
	ctx := context.WithValue(context.Background(), membershipKey, MembershipAccess{Premium: false})
	raw := json.RawMessage(`{"money":[{"sales":12345678901234.56,"gross_margin":99,"operating_expenses":10}],"items":[{"frozen_cost_unit":2,"name":"Burger","CurrentStock":9,"base_price":100}]}`)
	out := RedactMembershipData(ctx, raw)
	text := string(out)
	for _, hidden := range []string{"gross_margin", "operating_expenses", "frozen_cost_unit", "CurrentStock"} {
		if strings.Contains(text, hidden) {
			t.Fatalf("Premium data leaked: %s", hidden)
		}
	}
	if !strings.Contains(text, "12345678901234.56") || !strings.Contains(text, "base_price") {
		t.Fatal("Sales precision or public price changed")
	}
	premium := context.WithValue(context.Background(), membershipKey, MembershipAccess{Premium: true})
	if string(RedactMembershipData(premium, raw)) != string(raw) {
		t.Fatal("Premium payload changed")
	}
	for _, path := range []string{"/api/inventory", "/api/inventory/movements", "/api/expenses/x/void", "/api/recipes", "/api/supplies/x", "/api/production", "/api/analytics/operations"} {
		if !premiumPath(path) {
			t.Errorf("Ungated module %s", path)
		}
	}
	for _, path := range []string{"/api/team/invitations", "/api/orders", "/api/billing/plans", "/api/products", "/api/inventory-forged"} {
		if premiumPath(path) {
			t.Errorf("Basic feature incorrectly gated %s", path)
		}
	}
}
