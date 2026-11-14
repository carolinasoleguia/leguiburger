package mercadopago

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestExactMoneyAndEncryptedCredentials(t *testing.T) {
	if value, e := plus("999999999999.99", "0.01"); e != nil || value != "1000000000000.00" {
		t.Fatal("money precision lost")
	}
	for _, value := range []string{"NaN", "1e3", "-1", "0.001", "Infinity"} {
		if positiveMoney(value) {
			t.Fatalf("invalid money accepted: %s", value)
		}
	}
	cfg := Config{KeyID: "v1", Keys: map[string]string{"v1": base64.StdEncoding.EncodeToString(make([]byte, 32))}}
	encrypted, e := cfg.seal("private-refresh-token", "connection/a")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = cfg.open(encrypted, "connection/b"); e == nil {
		t.Fatal("credential moved across merchants")
	}
	value, e := cfg.open(encrypted, "connection/a")
	if e != nil || value != "private-refresh-token" {
		t.Fatal("credential encryption round trip failed")
	}
}
func signForTest(id, request, secret string, at time.Time) string {
	ts := strconv.FormatInt(at.UnixMilli(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "id:%s;request-id:%s;ts:%s;", id, request, ts)
	return "ts=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}
func TestSignedIngressCannotBeForgedOrReplayedOutsideWindow(t *testing.T) {
	now := time.Now()
	signature := signForTest("1001", "fixture-request", "fixture-secret", now)
	if !signatureValid(signature, "fixture-request", "1001", "fixture-secret", now) {
		t.Fatal("valid signature rejected")
	}
	if signatureValid(signature, "fixture-request", "1002", "fixture-secret", now) || signatureValid(signature, "fixture-request", "1001", "wrong", now) || signatureValid(signature, "fixture-request", "1001", "fixture-secret", now.Add(6*time.Minute)) {
		t.Fatal("forged or stale notification accepted")
	}
	if signatureValid(signature+",v1=x", "fixture-request", "1001", "fixture-secret", now) {
		t.Fatal("duplicate signature accepted")
	}
}
func TestProviderNoRedirectAndAmbiguousWrites(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Path == "/failure" {
			w.WriteHeader(503)
			return
		}
		http.Redirect(w, r, "http://127.0.0.1:1/steal", 302)
	}))
	defer server.Close()
	p := NewProvider()
	p.base = server.URL
	if _, e := p.call(context.Background(), "GET", "/redirect", "private-token", "", nil); e == nil {
		t.Fatal("provider redirect followed")
	}
	_, e := p.call(context.Background(), "POST", "/failure", "private-token", "", map[string]string{"test": "value"})
	failure, ok := e.(*providerFailure)
	if !ok || !failure.Ambiguous {
		t.Fatal("uncertain POST was treated as safe to repeat")
	}
	if hits != 2 {
		t.Fatal("provider request retried automatically")
	}
}
func TestPaymentProofBindsCollectorEnvironmentAndUnknownCosts(t *testing.T) {
	c := connection{CollectorID: "seller-1", Environment: "sandbox"}
	raw := map[string]any{"id": "1", "collector_id": "seller-1", "live_mode": false, "status": "approved", "currency_id": "ARS", "transaction_amount": json.Number("10.00"), "date_approved": time.Now().Format(time.RFC3339), "date_last_updated": time.Now().Format(time.RFC3339)}
	p, e := parsePayment(raw, "1", c)
	if e != nil {
		t.Fatal(e)
	}
	if p.Fee != nil || p.Net != nil {
		t.Fatal("unknown fees fabricated as zero")
	}
	raw["live_mode"] = true
	if _, e = parsePayment(raw, "1", c); e == nil {
		t.Fatal("live payment accepted into sandbox")
	}
	raw["live_mode"] = false
	raw["collector_id"] = "other-seller"
	if _, e = parsePayment(raw, "1", c); e == nil {
		t.Fatal("another seller's payment accepted")
	}
}
