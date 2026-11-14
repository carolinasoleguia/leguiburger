package operations

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type verificationTransport func(*http.Request) (*http.Response, error)

func (f verificationTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestCheckoutVerificationBindsOriginActionSessionAndAge(t *testing.T) {
	for _, tt := range []struct {
		name, host, action, cdata string
		age                       time.Duration
		success                   bool
	}{
		{"valid", "shop.example", "checkout", "bound", time.Minute, true},
		{"foreign host", "evil.example", "checkout", "bound", time.Minute, true},
		{"foreign action", "shop.example", "login", "bound", time.Minute, true},
		{"foreign browser", "shop.example", "checkout", "other", time.Minute, true},
		{"expired", "shop.example", "checkout", "bound", 6 * time.Minute, true},
		{"future", "shop.example", "checkout", "bound", -time.Minute, true},
		{"rejected", "shop.example", "checkout", "bound", time.Minute, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := &checkoutSecurity{hostname: "shop.example", secret: "fixture-not-a-provider-key"}
			s.client = &http.Client{Transport: verificationTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != "https://challenges.cloudflare.com/turnstile/v0/siteverify" || r.Method != "POST" {
					t.Fatal("unexpected provider request")
				}
				r.ParseForm()
				if r.Form.Get("response") != "fixture-token" || r.Form.Get("secret") != s.secret {
					t.Fatal("invalid provider payload")
				}
				out, _ := json.Marshal(map[string]any{"success": tt.success, "hostname": tt.host, "action": tt.action, "cdata": tt.cdata, "challenge_ts": time.Now().Add(-tt.age)})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(out)))}, nil
			})}
			err := s.verify(context.Background(), "fixture-token", "bound")
			if (err == nil) != (tt.name == "valid") {
				t.Fatalf("wrong validation result: %v", err)
			}
		})
	}
}
func TestVerificationOutageIsNotBuyerAbuse(t *testing.T) {
	s := &checkoutSecurity{client: &http.Client{Transport: verificationTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })}}
	if !errors.Is(s.verify(context.Background(), "token", "context"), errVerificationUnavailable) {
		t.Fatal("network failure classified as invalid buyer")
	}
}
func TestRiskWindowsAndHumanVerificationConfiguration(t *testing.T) {
	now := time.Now()
	for _, tt := range []struct {
		state  riskState
		reason string
	}{
		{riskState{WindowStart: now, NewSessions: 12}, "session_churn"},
		{riskState{WindowStart: now, InvalidRequests: 8}, "invalid_requests"},
		{riskState{WindowStart: now, Orders: 8}, "repeated_orders"},
		{riskState{WindowStart: now.Add(-16 * time.Minute), InvalidRequests: 100}, ""},
		{riskState{WindowStart: now, NewSessions: 2, Orders: 1}, ""},
	} {
		if riskReason(tt.state) != tt.reason {
			t.Fatal("unexpected risk window")
		}
	}
	s := &checkoutSecurity{siteKey: "fixture-site", secret: "fixture-secret", hostname: "shop.example", hashKey: "fixture-hmac-key"}
	t.Setenv("APP_ENV", "production")
	if !s.configured() {
		t.Fatal("complete config rejected")
	}
	s.siteKey = "1x0000-test"
	if s.configured() {
		t.Fatal("test key enabled in production")
	}
	s.siteKey = "fixture-site"
	s.secret = ""
	if s.configured() {
		t.Fatal("partial configuration accepted")
	}
	if s.hash("ip:192.0.2.1") == (&checkoutSecurity{hashKey: "other-key"}).hash("ip:192.0.2.1") {
		t.Fatal("unsalted subject identity")
	}
}
