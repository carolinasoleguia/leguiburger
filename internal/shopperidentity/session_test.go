package shopperidentity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShopperCookieAndOrigin(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("PUBLIC_SITE_URL", "https://burger.example")
	raw := Random()
	w := httptest.NewRecorder()
	SetCookie(w, raw)
	c := w.Result().Cookies()[0]
	if c.Name != Cookie || !c.HttpOnly || !c.Secure || c.Path != "/api" || c.MaxAge != 604800 || c.SameSite != http.SameSiteLaxMode || c.Domain != "" {
		t.Fatal("unsafe shopper cookie")
	}
	if Digest(raw) == raw || len(Digest(raw)) != 64 || len(raw) != 43 {
		t.Fatal("invalid capability representation")
	}
	for _, origin := range []string{"https://evil.example", "http://burger.example", "https://burger.example.evil.test"} {
		r := httptest.NewRequest("POST", "https://burger.example/api/shopper/login", nil)
		r.Header.Set("Origin", origin)
		if Origin(r) {
			t.Fatal("untrusted origin accepted")
		}
	}
	r := httptest.NewRequest("GET", "https://burger.example/api/shopper/me", nil)
	if _, e := Current(context.Background(), nil, r); e != ErrSession {
		t.Fatal("anonymous became shopper")
	}
	r.Header.Set("Origin", "https://burger.example")
	if !Origin(r) {
		t.Fatal("trusted origin rejected")
	}
	r.Header.Set("Sec-Fetch-Site", "same-site")
	if Origin(r) {
		t.Fatal("cross-subdomain request accepted")
	}
	cleared := httptest.NewRecorder()
	SetCookie(cleared, "")
	if cleared.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("logout cookie retained")
	}
}
