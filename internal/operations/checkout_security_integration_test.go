package operations

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"leguiburger/internal/auth"
	"leguiburger/internal/buyeraccess"
	"leguiburger/internal/db"
	"leguiburger/internal/requestguard"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCheckoutSecurityIntegration(t *testing.T) {
	if !*securityIntegration {
		t.Skip("explicit -security-integration required")
	}
	_ = godotenv.Load("../../.env")
	cfg, e := pgx.ParseConfig(os.Getenv("DATABASE_URL"))
	if e != nil || cfg.User != "postgres.ljdebnvzdrlqdxnligvh" {
		t.Fatal("unexpected integration target")
	}
	database, e := gorm.Open(postgres.New(postgres.Config{DSN: os.Getenv("DATABASE_URL"), PreferSimpleProtocol: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal("database unavailable")
	}
	pool, _ := database.DB()
	defer pool.Close()
	if e = VerifyDatabase(database, os.Getenv("DATABASE_URL")); e != nil {
		t.Fatal(e)
	}
	tx := database.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	previous := db.DB
	db.DB = tx
	defer func() { db.DB = previous }()
	t.Setenv("PUBLIC_SITE_URL", "http://127.0.0.1:5173")
	t.Setenv("APP_ENV", "development")
	t.Setenv("AUTH_REQUIRE_PLATFORM_MFA", "false")
	t.Setenv("RENDER", "")
	t.Setenv("TURNSTILE_SITE_KEY", "fixture-public-widget")
	t.Setenv("TURNSTILE_SECRET_KEY", "fixture-provider-secret")
	t.Setenv("ABUSE_HASH_KEY", "fixture-stable-private-hash-key")
	if e = auth.ConfigureJWTSecret("fixture-security-access-secret-for-rollback-only"); e != nil {
		t.Fatal(e)
	}
	must := func(q string, args ...any) {
		t.Helper()
		if e := tx.Exec(q, args...).Error; e != nil {
			t.Fatal(e)
		}
	}
	brand, tenant, product, owner, admin, cashier := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	must(`INSERT INTO public.brands(id,name) VALUES(?,'__abuse_security_fixture')`, brand)
	paidLegacyFixture(t, tx, brand)
	must(`INSERT INTO public.tenants(id,brand_id,subdomain,menu_published,ordering_enabled) VALUES(?,?,'abuse-'||?::text,true,true)`, tenant, brand, tenant)
	must(`INSERT INTO public.products(id,brand_id,name,base_price,is_active) VALUES(?,?,'Fixture burger',100,true)`, product, brand)
	must(`INSERT INTO public.tenant_products(tenant_id,product_id,current_stock,is_available,is_active,track_stock) VALUES(?,?,100,true,true,true)`, tenant, product)
	must(`INSERT INTO public.users(id,email,password_hash,role,brand_id,is_active) VALUES(?,?,'fixture','owner',NULL,true),(?,?,'fixture','admin',?,true)`, owner, owner+"@example.invalid", admin, admin+"@example.invalid", brand)
	must(`INSERT INTO public.employees(id,tenant_id,email,first_name,last_name,password_hash,role,is_active) VALUES(?, ?,?,'Fixture','Cashier','fixture','cashier',true)`, cashier, tenant, cashier+"@example.invalid")
	token := func(id, kind, role, b, local string) string {
		sid := uuid.NewString()
		must(`INSERT INTO legui_private.auth_sessions(id,principal_kind,principal_id,role,brand_id,tenant_id,credential_digest,mfa_verified) VALUES(?,?,?,?,nullif(?,'')::uuid,nullif(?,'')::uuid,?,false)`, sid, kind, id, role, b, local, fmt.Sprintf("%x", sha256.Sum256([]byte("fixture"))))
		value, e := auth.SessionToken(auth.SessionPrincipal{ID: id, Kind: kind, Role: role, BrandID: b, TenantID: local}, sid, false, "access")
		if e != nil {
			t.Fatal(e)
		}
		return value
	}
	ownerToken, adminToken, cashierToken := token(owner, "user", "owner", "", ""), token(admin, "user", "admin", brand, ""), token(cashier, "employee", "cashier", brand, tenant)
	mux := http.NewServeMux()
	h := registerRoutes(mux, tx)
	checkoutBursts = requestguard.New()
	defer func() { checkoutBursts = requestguard.New() }()
	providerContext := ""
	h.security.client = &http.Client{Transport: verificationTransport(func(r *http.Request) (*http.Response, error) {
		r.ParseForm()
		value := r.Form.Get("response")
		host, action, cdata, at, ok := "127.0.0.1", "checkout", providerContext, time.Now(), true
		switch value {
		case "wrong-host":
			host = "other.example"
		case "wrong-context":
			cdata = "another-session"
		case "expired":
			at = at.Add(-6 * time.Minute)
		case "outage":
			return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("{}"))}, nil
		case "invalid":
			ok = false
		}
		payload, _ := json.Marshal(map[string]any{"success": ok, "hostname": host, "action": action, "cdata": cdata, "challenge_ts": at})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(payload))}, nil
	})}
	type reply struct {
		payload  map[string]any
		cookie   *http.Cookie
		recorder *httptest.ResponseRecorder
	}
	checks := 0
	request := func(method, path, actor, ip, key string, cookie *http.Cookie, body any, want int) reply {
		t.Helper()
		encoded, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(encoded))
		r.RemoteAddr = ip + ":4321"
		if actor != "" {
			r.Header.Set("Authorization", "Bearer "+actor)
		}
		if cookie != nil {
			r.AddCookie(cookie)
			r.Header.Set("X-Buyer-Tenant", tenant)
		}
		if key == "" {
			key = uuid.NewString()
		}
		r.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		checks++
		var payload map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &payload)
		if w.Code != want {
			t.Fatalf("%s %s want %d got %d code=%v", method, path, want, w.Code, payload["code"])
		}
		var updated *http.Cookie
		for _, c := range w.Result().Cookies() {
			if c.Name == buyeraccess.CookieName(tenant) {
				updated = c
			}
		}
		return reply{payload, updated, w}
	}
	ip := "198.51.100.73"
	path := "/api/public/menu/" + tenant
	session := func() *http.Cookie {
		t.Helper()
		c := request("POST", path+"/buyer-session", "", ip, "", nil, map[string]any{}, 200).cookie
		if c == nil {
			t.Fatal("missing buyer cookie")
		}
		return c
	}
	cookieKey := func(c *http.Cookie) string { return strings.Split(c.Value, ".")[1] }
	basket := map[string]any{"fulfillment": "pickup", "items": []any{map[string]any{"product_id": product, "quantity": 1}}}
	quote := request("POST", path+"/quote", "", ip, "", nil, basket, 200).payload
	body := map[string]any{"fulfillment": "pickup", "items": basket["items"], "quote_digest": quote["digest"], "name": "Fixture buyer", "phone": "1155550123", "payment_method": "at_local"}
	a := session()
	request("POST", path+"/orders", "", ip, cookieKey(a), a, body, 201)
	var b *http.Cookie
	for i := 0; i < 12; i++ {
		b = session()
	}
	c := session()
	required := request("POST", path+"/orders", "", ip, cookieKey(b), b, body, 403)
	if required.payload["code"] != "HUMAN_VERIFICATION_REQUIRED" {
		t.Fatal("risk did not require human verification")
	}
	config := request("GET", path+"/human-check", "", ip, "", b, nil, 200).payload
	if strings.Contains(fmt.Sprint(config), h.security.secret) {
		t.Fatal("provider secret exposed")
	}
	providerContext = config["cdata"].(string)
	for _, value := range []string{"wrong-host", "wrong-context", "expired"} {
		request("POST", path+"/human-check", "", ip, "", b, map[string]any{"token": value}, 403)
	}
	riskRequest := httptest.NewRequest("GET", "/", nil)
	riskRequest.RemoteAddr = ip + ":4321"
	before, _ := h.security.state(t.Context(), riskRequest)
	request("POST", path+"/human-check", "", ip, "", b, map[string]any{"token": "outage"}, 503)
	after, _ := h.security.state(t.Context(), riskRequest)
	if after.InvalidRequests != before.InvalidRequests {
		t.Fatal("provider outage penalized the buyer")
	}
	request("POST", path+"/human-check", "", ip, "", b, map[string]any{"token": "valid-b"}, 200)
	request("POST", path+"/human-check", "", ip, "", b, map[string]any{"token": "valid-b"}, 409)
	request("POST", path+"/human-check", "", ip, "", c, map[string]any{"token": "other-browser"}, 403)
	request("POST", path+"/human-check", "", "198.51.100.74", "", b, map[string]any{"token": "other-ip"}, 403)
	request("POST", path+"/orders", "", ip, cookieKey(b), b, body, 201)
	request("POST", path+"/orders", "", ip, cookieKey(b), b, body, 201) // Same receipt, no second grant/order.
	forged := *b
	forged.Value = strings.Split(b.Value, ".")[0] + "." + uuid.NewString()
	request("POST", path+"/orders", "", ip, cookieKey(&forged), &forged, body, 403)
	for _, actor := range []string{adminToken, cashierToken} {
		request("GET", "/api/platform/security/incidents", actor, ip, "", nil, nil, 403)
	}
	list := request("GET", "/api/platform/security/incidents?status=all", ownerToken, ip, "", nil, nil, 200).payload
	rows := list["items"].([]any)
	if len(rows) == 0 {
		t.Fatal("incidents missing")
	}
	incident := ""
	for _, row := range rows {
		item := row.(map[string]any)
		if item["reference"] == h.security.subject(riskRequest)[:12] {
			incident = item["id"].(string)
			break
		}
	}
	if incident == "" {
		t.Fatal("target origin incident missing")
	}
	actionPath := "/api/platform/security/incidents/" + incident + "/actions"
	action := map[string]any{"action": "block", "minutes": 60, "note": "Fixture owner reviewed this connection"}
	request("POST", actionPath, adminToken, ip, "", nil, action, 403)
	request("POST", actionPath, ownerToken, ip, "", nil, map[string]any{"action": "block", "minutes": 60, "note": "short"}, 400)
	request("POST", actionPath, ownerToken, ip, "fixture-block", nil, action, 200)
	replay := request("POST", actionPath, ownerToken, ip, "fixture-block", nil, action, 200)
	if replay.recorder.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatal("administrative retry not idempotent")
	}
	request("POST", path+"/orders", "", ip, cookieKey(c), c, body, 429)
	request("GET", path, "", ip, "", nil, nil, 200)
	request("GET", "/api/public/buyer-order", "", ip, "", a, nil, 200)
	request("POST", path+"/orders", "", ip, cookieKey(b), b, body, 201) // A manual block cannot hide a committed receipt.
	// Force a local burst block, then ensure the persisted release overrides it.
	checkoutBursts.Allow("orders:"+ip, 1, time.Now())
	checkoutBursts.Allow("orders:"+ip, 1, time.Now())
	request("POST", actionPath, ownerToken, ip, "", nil, map[string]any{"action": "unblock", "note": "Fixture false positive was checked"}, 200)
	released, releaseErr := h.security.state(t.Context(), riskRequest)
	if releaseErr != nil || released.ReleasedUntil == nil || blockedRisk(released) != nil {
		t.Fatalf("release not effective: err=%v released=%v blocked=%v", releaseErr, released.ReleasedUntil, released.BlockedUntil)
	}
	request("POST", path+"/orders", "", ip, cookieKey(c), c, body, 201)
	var auditCount int64
	tx.Raw(`SELECT count(*) FROM legui_private.security_events WHERE actor_id=? AND event_name IN ('checkout_abuse.block','checkout_abuse.unblock')`, owner).Scan(&auditCount)
	if auditCount != 2 {
		t.Fatal("missing or duplicate actor-attributed audit")
	}
	var orders int64
	tx.Raw(`SELECT count(*) FROM public.orders WHERE tenant_id=?`, tenant).Scan(&orders)
	if orders != 3 {
		t.Fatal("verification/retries duplicated orders")
	}
	t.Logf("%d security HTTP checks: risk, provider binding/replay, one-order grant, owner-only review/unblock, audit and purchase replay; all fixtures roll back", checks)
}
