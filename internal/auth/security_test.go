package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTOTPRFC6238VectorsAndReplay(t *testing.T) {
	// RFC 6238 SHA1 vectors, truncated to the six digits used by authenticators.
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	for _, fixture := range []struct {
		at   int64
		code string
	}{{59, "287082"}, {1111111109, "081804"}, {1111111111, "050471"}, {1234567890, "005924"}, {2000000000, "279037"}, {20000000000, "353130"}} {
		step, ok := verifyTOTP(secret, fixture.code, -1, time.Unix(fixture.at, 0))
		if !ok {
			t.Fatalf("RFC vector rejected at %d", fixture.at)
		}
		if _, ok = verifyTOTP(secret, fixture.code, step, time.Unix(fixture.at, 0)); ok {
			t.Fatal("TOTP replay accepted")
		}
	}
	if _, ok := verifyTOTP(secret, "123456", -1, time.Unix(59, 0)); ok {
		t.Fatal("incorrect TOTP accepted")
	}
	if _, ok := verifyTOTP(secret, "287082", -1, time.Unix(180, 0)); ok {
		t.Fatal("expired TOTP accepted")
	}
}
func TestSecretEncryptionIsBoundToIdentity(t *testing.T) {
	t.Setenv("AUTH_ENCRYPTION_KEY", "unit-test-encryption-key")
	sealed, err := sealSecret("sensitive", "totp/person-a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = openSecret(sealed, "totp/person-b"); err == nil {
		t.Fatal("cross-account ciphertext accepted")
	}
	plain, err := openSecret(sealed, "totp/person-a")
	if err != nil || plain != "sensitive" {
		t.Fatal("secret round trip failed")
	}
	if sealed == "sensitive" {
		t.Fatal("secret stored as plaintext")
	}
}
func TestLegacyPermissionMatrix(t *testing.T) {
	employee := SessionPrincipal{Kind: "employee", Role: "employee", TenantID: "tenant-a", BrandID: "brand-a"}
	cashier := employee
	cashier.Role = "cashier"
	kitchen := employee
	kitchen.Role = "kitchen"
	manager := SessionPrincipal{Kind: "user", Role: "admin", BrandID: "brand-a"}
	platform := SessionPrincipal{Kind: "user", Role: "owner"}
	for _, group := range []string{"brands", "users", "employees", "products", "extras", "supplies", "shipping-methods", "tenant-products", "recipes", "tenants"} {
		if CanLegacy(employee, group, "PUT") {
			t.Errorf("employee may mutate %s", group)
		}
	}
	if !CanLegacy(cashier, "customers", "POST") || CanLegacy(kitchen, "customers", "GET") {
		t.Fatal("cashier/kitchen customer boundary failed")
	}
	if !CanLegacy(kitchen, "production", "POST") || CanLegacy(kitchen, "production", "DELETE") {
		t.Fatal("kitchen production boundary failed")
	}
	if !CanLegacy(manager, "products", "POST") || CanLegacy(manager, "tenants", "PUT") {
		t.Fatal("manager administrative boundary failed")
	}
	if !CanLegacy(platform, "tenants", "DELETE") || CanLegacy(platform, "unknown", "GET") {
		t.Fatal("platform/default-deny boundary failed")
	}
	forged := platform
	forged.Kind = "employee"
	if CanLegacy(forged, "users", "GET") {
		t.Fatal("employee platform role accepted")
	}
}
func TestLegacyDenialHappensBeforeHandler(t *testing.T) {
	req := httptest.NewRequest("PUT", "/api/employees/another", nil)
	req = req.WithContext(context.WithValue(req.Context(), PrincipalKey, SessionPrincipal{Kind: "employee", Role: "employee"}))
	called := false
	handler := AuthorizeLegacy(func(w http.ResponseWriter, r *http.Request) { called = true })
	recorder := httptest.NewRecorder()
	handler(recorder, req)
	if called || recorder.Code != 403 {
		t.Fatal("unauthorized handler was reached")
	}
}

func TestCookieOriginAndPasswordPolicies(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("PUBLIC_SITE_URL", "https://burger.example")
	r := httptest.NewRequest("POST", "https://burger.example/api/auth/refresh", nil)
	r.Header.Set("Origin", "https://evil.example")
	if sameOrigin(r) {
		t.Fatal("cross-origin cookie mutation accepted")
	}
	r.Header.Set("Origin", "https://burger.example")
	if !sameOrigin(r) {
		t.Fatal("same-origin mutation rejected")
	}
	w := httptest.NewRecorder()
	setRefreshCookie(w, "test")
	c := w.Result().Cookies()[0]
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode {
		t.Fatal("refresh cookie is missing protections")
	}
	if validPassword("short") || validPassword("  twelvecharacters") {
		t.Fatal("weak password accepted")
	}
	if !validPassword("una frase larga y segura") {
		t.Fatal("passphrase rejected")
	}
}
