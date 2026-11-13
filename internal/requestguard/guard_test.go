package requestguard

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestBurstIsolationAndExpiry(t *testing.T) {
	g := New()
	now := time.Now()
	for i := 0; i < 20; i++ {
		if g.Allow("orders:ip-a", 20, now) != 0 {
			t.Fatal("early block")
		}
	}
	if g.Allow("orders:ip-a", 20, now) != 5*time.Minute {
		t.Fatal("missing cooling off")
	}
	if g.Allow("orders:ip-b", 20, now) != 0 || g.Allow("quote:ip-a", 20, now) != 0 {
		t.Fatal("unrelated client/action blocked")
	}
	if g.Allow("orders:ip-a", 20, now.Add(time.Minute)) != 4*time.Minute {
		t.Fatal("block extended by retry")
	}
	if g.Allow("orders:ip-a", 20, now.Add(5*time.Minute)) != 0 {
		t.Fatal("block did not expire")
	}
}
func TestClientIPTrustAndNormalization(t *testing.T) {
	r := httptest.NewRequest("POST", "/", nil)
	r.RemoteAddr = "192.0.2.3:3456"
	r.Header.Set("CF-Connecting-IP", "198.51.100.1")
	r.Header.Set("X-Forwarded-For", "198.51.100.2")
	t.Setenv("RENDER", "")
	if ClientIP(r) != "192.0.2.3" {
		t.Fatal("spoofed local IP")
	}
	t.Setenv("RENDER", "true")
	if ClientIP(r) != "198.51.100.1" {
		t.Fatal("Render visitor IP not resolved")
	}
	r.Header.Set("CF-Connecting-IP", "garbage")
	if ClientIP(r) != "192.0.2.3" {
		t.Fatal("invalid header accepted")
	}
	r.Header.Set("CF-Connecting-IP", "::ffff:198.51.100.1")
	if ClientIP(r) != "198.51.100.1" {
		t.Fatal("mapped IPv4 bypass")
	}
}
