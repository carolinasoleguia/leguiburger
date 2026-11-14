package auth

import "testing"

func TestAdministrationIsSeparateFromRestaurantOperations(t *testing.T) {
	for _, tt := range []struct {
		kind, role                 string
		manager, cashier, operator bool
	}{
		{"user", "admin", true, false, false}, {"user", "owner", true, true, true}, {"user", "super_admin", true, true, true},
		{"employee", "cashier", false, true, true}, {"employee", "kitchen", false, false, true}, {"employee", "employee", false, false, true},
		{"employee", "admin", false, false, false}, {"user", "cashier", false, false, false}, {"employee", "owner", false, false, false},
	} {
		t.Run(tt.kind+"/"+tt.role, func(t *testing.T) {
			p := SessionPrincipal{Kind: tt.kind, Role: tt.role, BrandID: "fixture"}
			if p.Manager() != tt.manager || p.Cashier() != tt.cashier || p.Operator() != tt.operator {
				t.Fatalf("unexpected capabilities for %s/%s", tt.kind, tt.role)
			}
		})
	}
}
