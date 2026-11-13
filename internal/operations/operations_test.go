package operations

import (
	"encoding/json"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDecimalValidation(t *testing.T) {
	for _, input := range []string{`"NaN"`, `"Infinity"`, `1e9`, `null`, `-1.23456`, `"1;SELECT 1"`, `"99999999999999999999999"`} {
		var value Decimal
		if json.Unmarshal([]byte(input), &value) == nil {
			t.Errorf("accepted unsafe decimal %s", input)
		}
	}
	for _, input := range []string{`"123.45"`, `0`, `-1.25`, `"0.0001"`} {
		var value Decimal
		if err := json.Unmarshal([]byte(input), &value); err != nil {
			t.Errorf("valid decimal rejected %s", input)
		}
	}
	if validMoney(Decimal("-1"), true) || validMoney(Decimal("0"), true) || validMoney(Decimal("1.001"), true) {
		t.Fatal("money allows negative, zero or excess precision")
	}
	if !validMoney(Decimal("12345678901234.56"), true) {
		t.Fatal("large exact money lost")
	}
	if !integerDecimal(Decimal("2.0000")) || integerDecimal(Decimal("1.25")) {
		t.Fatal("integer inventory precision")
	}
}
func TestOrderLifecycle(t *testing.T) {
	for _, pair := range [][2]string{{"pending", "accepted"}, {"accepted", "preparing"}, {"preparing", "ready"}, {"ready", "dispatched"}, {"dispatched", "delivered"}, {"accepted", "canceled"}} {
		if !validTransition(pair[0], pair[1]) {
			t.Fatal(pair)
		}
	}
	for _, pair := range [][2]string{{"pending", "delivered"}, {"canceled", "accepted"}, {"rejected", "accepted"}, {"delivered", "canceled"}, {"preparing", "accepted"}} {
		if validTransition(pair[0], pair[1]) {
			t.Fatal("invalid lifecycle", pair)
		}
	}
}
func TestRolesDoNotShareFinancialPermissions(t *testing.T) {
	employee := Principal{Kind: "employee", Role: "employee"}
	manager := Principal{Kind: "user", Role: "admin"}
	owner := Principal{Kind: "user", Role: "owner"}
	cashier := Principal{Kind: "employee", Role: "cashier"}
	if employee.Platform() || employee.Manager() || employee.Cashier() {
		t.Fatal("employee financial escalation")
	}
	if manager.Platform() || !manager.Manager() || manager.Cashier() || manager.Operator() {
		t.Fatal("business manager platform escalation")
	}
	if !owner.Platform() || !owner.Manager() {
		t.Fatal("platform owner permissions")
	}
	if !cashier.Cashier() || cashier.Manager() {
		t.Fatal("cashier scope")
	}
}
func TestStrictDTOs(t *testing.T) {
	for _, body := range []string{`{"kind":"charge","amount":10,"method":"cash","provider":"fake"}`, `{"kind":"charge","amount":10,"method":"cash"}{}`, `[]`} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		if _, err := decode[PaymentRequest](r); err == nil {
			t.Fatal("accepted untrusted fields/body", body)
		}
	}
}
func TestPeriodBounds(t *testing.T) {
	r := httptest.NewRequest("GET", "/?from=2026-11-01&to=2026-11-07", nil)
	p, err := readPeriod(r, "America/Argentina/Buenos_Aires")
	if err != nil {
		t.Fatal(err)
	}
	if p.Until.Sub(p.From) != 7*24*time.Hour {
		t.Fatal("inclusive calendar range")
	}
	if p.From.Hour() != 0 || p.From.Location().String() != "America/Argentina/Buenos_Aires" {
		t.Fatal("business timezone")
	}
	for _, path := range []string{"/?from=wrong", "/?from=2026-12-01&to=2026-01-01", "/?from=2020-01-01&to=2026-01-01"} {
		if _, err := readPeriod(httptest.NewRequest("GET", path, nil), "America/Argentina/Buenos_Aires"); err == nil {
			t.Fatal("invalid range accepted", path)
		}
	}
}

func TestMenuManagementAndCashRoutesRequireSession(t *testing.T) {
	mux := http.NewServeMux()
	database, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 dbname=unused", PreferSimpleProtocol: true}), &gorm.Config{DisableAutomaticPing: true, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	RegisterRoutes(mux, database)
	for _, route := range []string{"GET /api/menu/settings", "PATCH /api/menu/settings", "GET /api/cash/shifts", "POST /api/cash/shifts", "GET /api/cash/shifts/00000000-0000-4000-8000-000000000001", "GET /api/cash/shifts/00000000-0000-4000-8000-000000000001/entries", "POST /api/cash/shifts/00000000-0000-4000-8000-000000000001/entries", "POST /api/cash/shifts/00000000-0000-4000-8000-000000000001/close"} {
		parts := strings.SplitN(route, " ", 2)
		r := httptest.NewRequest(parts[0], parts[1], strings.NewReader(`{}`))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Errorf("%s allowed anonymous access: %d", route, w.Code)
		}
	}
}

// Every operational handler must reject restaurant administrators before resolving
// a tenant/resource or decoding input. A forged ID cannot turn ownership into POS access.
func TestRestaurantAdminCannotOperateService(t *testing.T) {
	h := &Handler{}
	admin := Principal{Kind: "user", Role: "admin", BrandID: "fixture"}
	for name, handler := range map[string]endpoint{
		"orders.list": h.listOrders, "orders.create": h.createOrder, "orders.get": h.getOrder,
		"orders.status": h.changeOrderStatus, "orders.payment": h.createOrderPayment, "orders.payments": h.listOrderPayments,
		"orders.inbox": h.orderInbox, "orders.link": h.orderLink, "orders.ticket": h.orderTicket,
		"cash.list": h.cashShifts, "cash.open": h.openCash, "cash.get": h.cashShift,
		"cash.entries": h.cashEntries, "cash.move": h.moveCash, "cash.close": h.closeCash,
		"pos.catalog": h.posCatalog, "kitchen.list": h.kitchenOrders, "kitchen.status": h.kitchenStatus, "kitchen.load": h.kitchenLoad,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := handler(httptest.NewRequest("GET", "/api/forged", nil), admin)
			api, ok := err.(*apiError)
			if !ok || api.Status != http.StatusForbidden {
				t.Fatalf("wanted forbidden, got %v", err)
			}
		})
	}
}
