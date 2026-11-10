package operations

import (
	"encoding/json"
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
	if manager.Platform() || !manager.Manager() {
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
