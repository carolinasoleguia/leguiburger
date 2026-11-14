package buyeraccess

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuyerCookieIsolationAndFlags(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("PUBLIC_SITE_URL", "https://burger.example")
	tenant := "10000000-0000-4000-8000-000000000001"
	other := "10000000-0000-4000-8000-000000000002"
	session := NewBrowserSession()
	w := httptest.NewRecorder()
	SetCookie(w, tenant, session)
	cookie := w.Result().Cookies()[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/api/public" || cookie.MaxAge != 86400 || cookie.Domain != "" {
		t.Fatal("unsafe cookie attributes")
	}
	r := httptest.NewRequest("GET", "https://burger.example/api/public/buyer-order", nil)
	r.AddCookie(cookie)
	r.Header.Set("X-Buyer-Tenant", tenant)
	if Token(r) != session.Secret {
		t.Fatal("cookie not resolved")
	}
	r.Header.Set("X-Buyer-Tenant", other)
	if Token(r) != "" {
		t.Fatal("cross tenant cookie")
	}
	r.Header.Set("X-Buyer-Tenant", tenant)
	r.Header.Set("Origin", "https://evil.example")
	if Token(r) != "" {
		t.Fatal("cross-origin cookie accepted")
	}
	r.Header.Del("Origin")
	r.Header.Set("Sec-Fetch-Site", "same-site")
	if Token(r) != "" {
		t.Fatal("cross-subdomain cookie accepted")
	}
	r.Header.Del("Sec-Fetch-Site")
	r.Header.Set("Authorization", "Bearer invalid")
	if Token(r) != "invalid" {
		t.Fatal("invalid bearer fell back to cookie")
	}
	if PaymentKey(session) != PaymentKey(session) {
		t.Fatal("unstable payment idempotency")
	}
}
