package mercadopago

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"leguiburger/internal/auth"
	"leguiburger/internal/buyeraccess"
	"leguiburger/internal/operations"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

var mpIntegration = flag.Bool("mp-db-integration", false, "Run rollback-only Mercado Pago integration with a simulated upstream")

func TestDatabaseIntegration(t *testing.T) {
	if !*mpIntegration {
		t.Skip("opt-in DB integration")
	}
	_ = godotenv.Load("../../.env")
	database, e := gorm.Open(postgres.New(postgres.Config{DSN: os.Getenv("DATABASE_URL"), PreferSimpleProtocol: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal("database unavailable")
	}
	pool, _ := database.DB()
	defer pool.Close()
	var project string
	database.Raw(`SELECT project_ref FROM legui_private.database_contract WHERE singleton AND application='leguiburger'`).Scan(&project)
	if project != "ljdebnvzdrlqdxnligvh" {
		t.Fatal("unexpected DB identity")
	}
	tx := database.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	must := func(q string, args ...any) {
		t.Helper()
		if e := tx.Exec(q, args...).Error; e != nil {
			t.Fatal(e)
		}
	}
	var exists bool
	tx.Raw(`SELECT to_regclass('legui_private.mp_connections') IS NOT NULL`).Scan(&exists)
	if !exists {
		sql, e := os.ReadFile("../../database/migrations/202611110001_mercadopago_integration.sql")
		if e != nil {
			t.Fatal(e)
		}
		must(string(sql))
	}
	brand, otherBrand, tenant, otherTenant, admin, otherAdmin, owner, employee, product := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	must(`INSERT INTO public.brands(id,name) VALUES(?,'__mp_fixture'),(?,'__mp_other')`, brand, otherBrand)
	must(`INSERT INTO public.tenants(id,brand_id,subdomain) VALUES(?,?,'__mp_fixture'),(?,?,'__mp_other')`, tenant, brand, otherTenant, otherBrand)
	must(`INSERT INTO public.users(id,email,password_hash,role,brand_id,is_active) VALUES(?,?,'fixture','admin',?,true),(?,?,'fixture','admin',?,true),(?,?,'fixture','owner',NULL,true)`, admin, admin+"@example.invalid", brand, otherAdmin, otherAdmin+"@example.invalid", otherBrand, owner, owner+"@example.invalid")
	must(`INSERT INTO public.employees(id,tenant_id,first_name,last_name,email,password_hash,role,is_active) VALUES(?,?,'Fixture','Worker',?,'fixture','employee',true)`, employee, tenant, employee+"@example.invalid")
	must(`INSERT INTO public.products(id,brand_id,name,base_price,is_active) VALUES(?,?,'__mp_burger',100,true)`, product, brand)
	must(`INSERT INTO public.tenant_products(tenant_id,product_id,current_stock,is_active,is_available) VALUES(?,?,100,true,true)`, tenant, product)
	if e = auth.ConfigureJWTSecret("mp-integration-jwt-fixture-not-for-running-app"); e != nil {
		t.Fatal(e)
	}
	token := func(id, role, kind, b, tID string) string {
		t.Helper()
		sid := uuid.NewString()
		must(`INSERT INTO legui_private.auth_sessions(id,principal_kind,principal_id,role,brand_id,tenant_id,credential_digest,mfa_verified) VALUES(?,?,?,?,nullif(?,'')::uuid,nullif(?,'')::uuid,?,?)`, sid, kind, id, role, b, tID, fmt.Sprintf("%x", sha256.Sum256([]byte("fixture"))), role == "owner")
		value, e := auth.SessionToken(auth.SessionPrincipal{ID: id, Kind: kind, Role: role, BrandID: b, TenantID: tID}, sid, role == "owner", "access")
		if e != nil {
			t.Fatal(e)
		}
		return value
	}
	adminToken, otherToken, ownerToken, employeeToken := token(admin, "admin", "user", brand, ""), token(otherAdmin, "admin", "user", otherBrand, ""), token(owner, "owner", "user", "", ""), token(employee, "employee", "employee", brand, tenant)
	cfg := Config{Enabled: true, Environment: "sandbox", ClientID: "123", ClientSecret: "fixture-client", PlatformToken: "platform-token", PlatformCollector: "999", WebhookSecret: "fixture-webhook", APIOrigin: "https://leguiburger.example", PublicOrigin: "https://leguiburger.example", KeyID: "v1", Keys: map[string]string{"v1": base64.StdEncoding.EncodeToString(make([]byte, 32))}}
	var mutex sync.Mutex
	preferences := map[string]map[string]any{}
	payments := map[string]map[string]any{}
	mandates := map[string]map[string]any{}
	refunds := map[string][]any{}
	creates := 0
	cancelWrites := 0
	ambiguousNext := false
	providerCounter := 0
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		collector := "111"
		if r.Header.Get("Authorization") == "Bearer platform-token" {
			collector = "999"
		}
		var body map[string]any
		if r.Body != nil {
			d := json.NewDecoder(r.Body)
			d.UseNumber()
			_ = d.Decode(&body)
		}
		switch {
		case r.URL.Path == "/users/me":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": collector})
		case r.URL.Path == "/oauth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "business-token", "refresh_token": "business-refresh", "expires_in": json.Number("3600"), "user_id": "111", "live_mode": false})
		case r.Method == "POST" && r.URL.Path == "/checkout/preferences":
			creates++
			providerCounter++
			id := fmt.Sprintf("preference-%d", providerCounter)
			body["id"] = id
			body["collector_id"] = collector
			body["init_point"] = "https://www.mercadopago.com.ar/checkout?pref_id=" + id
			preferences[id] = body
			if ambiguousNext {
				ambiguousNext = false
				w.WriteHeader(503)
				return
			}
			_ = json.NewEncoder(w).Encode(body)
		case r.URL.Path == "/checkout/preferences/search":
			items := []any{}
			for _, v := range preferences {
				if text(v["external_reference"]) == r.URL.Query().Get("external_reference") {
					items = append(items, v)
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"elements": items, "total": len(items)})
		case strings.HasPrefix(r.URL.Path, "/checkout/preferences/"):
			v := preferences[strings.TrimPrefix(r.URL.Path, "/checkout/preferences/")]
			if v == nil {
				w.WriteHeader(404)
				return
			}
			_ = json.NewEncoder(w).Encode(v)
		case r.Method == "POST" && r.URL.Path == "/preapproval":
			creates++
			providerCounter++
			id := fmt.Sprintf("mandate-%d", providerCounter)
			body["id"] = id
			body["collector_id"] = collector
			body["init_point"] = "https://www.mercadopago.com.ar/subscriptions/checkout?preapproval_id=" + id
			body["last_modified"] = time.Now().UTC().Format(time.RFC3339Nano)
			mandates[id] = body
			_ = json.NewEncoder(w).Encode(body)
		case strings.HasPrefix(r.URL.Path, "/preapproval/"):
			id := strings.TrimPrefix(r.URL.Path, "/preapproval/")
			m := mandates[id]
			if m == nil {
				w.WriteHeader(404)
				return
			}
			if r.Method == "PUT" {
				cancelWrites++
				m["status"] = body["status"]
				m["last_modified"] = time.Now().UTC().Add(time.Second).Format(time.RFC3339Nano)
			}
			_ = json.NewEncoder(w).Encode(m)
		case r.URL.Path == "/v1/payments/search":
			items := []any{}
			for _, p := range payments {
				if r.URL.Query().Get("external_reference") == "" || r.URL.Query().Get("external_reference") == text(p["external_reference"]) {
					items = append(items, p)
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"results": items})
		case strings.HasPrefix(r.URL.Path, "/v1/payments/"):
			path := strings.TrimPrefix(r.URL.Path, "/v1/payments/")
			parts := strings.Split(path, "/")
			p := payments[parts[0]]
			if p == nil {
				w.WriteHeader(404)
				return
			}
			if len(parts) > 1 && parts[1] == "refunds" {
				if r.Method == "POST" {
					refund := map[string]any{"id": "refund-1", "payment_id": parts[0], "amount": body["amount"], "status": "approved", "date_created": time.Now().UTC().Format(time.RFC3339Nano)}
					refunds[parts[0]] = append(refunds[parts[0]], refund)
					p["transaction_amount_refunded"] = body["amount"]
					p["date_last_updated"] = time.Now().UTC().Add(time.Second).Format(time.RFC3339Nano)
					_ = json.NewEncoder(w).Encode(refund)
				} else {
					_ = json.NewEncoder(w).Encode(refunds[parts[0]])
				}
				return
			}
			_ = json.NewEncoder(w).Encode(p)
		default:
			w.WriteHeader(404)
		}
	}))
	defer fake.Close()
	provider := NewProvider()
	provider.base = fake.URL
	h := NewHandler(tx, cfg, provider)
	mux := http.NewServeMux()
	h.Register(mux)
	operations.RegisterRoutes(mux, tx)
	type reply struct {
		Body    map[string]any
		Cookies []*http.Cookie
		Status  int
	}
	request := func(method, path, token, op, local string, body any, want int, cookies ...*http.Cookie) reply {
		t.Helper()
		encoded, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(encoded))
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if op != "" {
			r.Header.Set("Idempotency-Key", op)
		}
		if local != "" {
			r.Header.Set("X-Tenant-ID", local)
		}
		for _, c := range cookies {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		var result map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &result)
		if w.Code != want {
			t.Fatalf("%s %s expected %d got %d code=%v message=%v", method, path, want, w.Code, result["code"], result["message"])
		}
		return reply{Body: result, Cookies: w.Result().Cookies(), Status: w.Code}
	}
	count := func(q string, want int, args ...any) {
		t.Helper()
		var value int64
		if e := tx.Raw(q, args...).Scan(&value).Error; e != nil {
			t.Fatal(e)
		}
		if value != int64(want) {
			t.Fatalf("expected %d got %d for %s", want, value, q)
		}
	}
	request("GET", "/api/payments/mercadopago/transactions", "", "", "", nil, 401)
	request("POST", "/api/integrations/mercadopago/connections", employeeToken, "", "", map[string]any{}, 403)
	conn := request("POST", "/api/integrations/mercadopago/connections", adminToken, "", "", map[string]any{"brand_id": brand, "environment": "sandbox"}, 201).Body["id"].(string)
	request("POST", "/api/integrations/mercadopago/connections/"+conn+"/authorize", otherToken, "", "", map[string]any{}, 403)
	authorization := request("POST", "/api/integrations/mercadopago/connections/"+conn+"/authorize", adminToken, "", "", map[string]any{}, 200)
	redirect, _ := url.Parse(authorization.Body["authorization_url"].(string))
	callback := "/api/integrations/mercadopago/callback?code=fixture-code&state=" + url.QueryEscape(redirect.Query().Get("state"))
	request("GET", callback, "", "", "", nil, 303, authorization.Cookies...)
	count(`SELECT count(*) FROM legui_private.mp_connections WHERE id=? AND state='connected' AND credential_cipher NOT LIKE '%business-token%'`, 1, conn)
	order := request("POST", "/api/orders", adminToken, "order-create", tenant, map[string]any{"channel": "web", "environment": "sandbox", "items": []any{map[string]any{"product_id": product, "quantity": 2}}}, 201).Body["id"].(string)
	request("POST", "/api/orders/"+order+"/checkout", otherToken, "cross", otherTenant, map[string]any{}, 404)
	coResult := request("POST", "/api/orders/"+order+"/checkout", adminToken, "order-checkout", tenant, map[string]any{}, 200)
	checkoutID := coResult.Body["id"].(string)
	if coResult.Body["state"] != "created" {
		t.Fatalf("checkout not created: state=%v reason=%v", coResult.Body["state"], coResult.Body["reason"])
	}
	request("POST", "/api/orders/"+order+"/checkout", adminToken, "order-checkout", tenant, map[string]any{}, 200)
	if creates != 1 {
		t.Fatal("checkout retry created another remote checkout")
	}
	var co checkout
	tx.Raw(`SELECT `+checkoutColumns+` FROM legui_private.mp_checkouts WHERE id=?`, checkoutID).Scan(&co)
	newPayment := func(id string, checkout checkout, collector, amount string) map[string]any {
		at := time.Now().UTC().Format(time.RFC3339Nano)
		return map[string]any{"id": id, "collector_id": collector, "live_mode": false, "external_reference": checkout.Reference, "transaction_amount": json.Number(amount), "transaction_amount_refunded": json.Number("0"), "currency_id": "ARS", "status": "approved", "status_detail": "accredited", "date_approved": at, "date_last_updated": at, "fee_details": []any{map[string]any{"amount": json.Number("10.00")}}, "transaction_details": map[string]any{"net_received_amount": json.Number("190.00")}}
	}
	payments["1001"] = newPayment("1001", co, "111", "200.00")
	enqueue := func(id, connection string) {
		t.Helper()
		r := httptest.NewRequest("POST", "/api/webhooks/mercadopago/"+connection+"?data.id="+id+"&type=payment", strings.NewReader(`{"type":"payment","data":{"id":"`+id+`"}}`))
		r.Header.Set("X-Request-ID", "fixture-request")
		r.Header.Set("X-Signature", signForTest(id, "fixture-request", cfg.WebhookSecret, time.Now()))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != 202 {
			t.Fatalf("webhook rejected %d", w.Code)
		}
	}
	enqueue("1001", conn)
	enqueue("1001", conn)
	count(`SELECT count(*) FROM legui_private.mp_webhook_inbox WHERE connection_id=?`, 1, conn)
	if found, e := h.processInbox(context.Background()); e != nil || !found {
		t.Fatalf("inbox failed: %v", e)
	}
	count(`SELECT count(*) FROM public.order_payments WHERE order_id=? AND status='succeeded'`, 1, order)
	var total string
	tx.Raw(`SELECT total_paid::text FROM public.orders WHERE id=?`, order).Scan(&total)
	if !sameMoney(total, "200.00") {
		t.Fatalf("order balance not applied: %s", total)
	}
	if state, e := h.observePayment(context.Background(), conn, "1001"); e != nil || state != "replayed" {
		t.Fatalf("payment replay failed: %s %v", state, e)
	}
	var paymentID string
	tx.Raw(`SELECT id::text FROM legui_private.mp_provider_payments WHERE payment_id='1001'`).Scan(&paymentID)
	request("POST", "/api/payments/mercadopago/transactions/"+paymentID+"/refunds", otherToken, "refund-cross", "", map[string]any{"amount": "20.00"}, 403)
	refund := request("POST", "/api/payments/mercadopago/transactions/"+paymentID+"/refunds", adminToken, "refund-one", "", map[string]any{"amount": "20.00"}, 200)
	if refund.Body["state"] != "completed" {
		t.Fatal("refund not completed")
	}
	request("POST", "/api/payments/mercadopago/transactions/"+paymentID+"/refunds", adminToken, "refund-one", "", map[string]any{"amount": "20.00"}, 200)
	tx.Raw(`SELECT total_paid::text FROM public.orders WHERE id=?`, order).Scan(&total)
	if !sameMoney(total, "180.00") {
		t.Fatalf("refund ledger not applied: %s", total)
	}
	// A different payment with the same checkout cannot settle it again.
	payments["1002"] = newPayment("1002", co, "111", "200.00")
	if state, e := h.observePayment(context.Background(), conn, "1002"); e != nil || state != "review" {
		t.Fatalf("duplicate checkout payment was not held: %s %v", state, e)
	}
	count(`SELECT count(*) FROM legui_private.mp_reviews WHERE resource_id='1002' AND resolved_at IS NULL`, 1)
	var reviewID string
	tx.Raw(`SELECT id::text FROM legui_private.mp_reviews WHERE connection_id=? AND resource_id='1002' AND resolved_at IS NULL`, conn).Scan(&reviewID)
	request("POST", "/api/payments/mercadopago/reviews/"+reviewID+"/resolve", otherToken, "review-cross", "", map[string]any{"resolution": "reprocess", "note": "Revisar nuevamente el pago verificado."}, 403)
	request("POST", "/api/payments/mercadopago/reviews/"+reviewID+"/resolve", adminToken, "review-reprocess", "", map[string]any{"resolution": "reprocess", "note": "Revisar nuevamente el pago verificado."}, 200)
	if found, e := h.processInbox(context.Background()); e != nil || !found {
		t.Fatalf("review reprocessing failed: %v", e)
	}
	count(`SELECT count(*) FROM public.order_payments WHERE order_id=? AND kind='charge'`, 1, order)
	// A platform subscription authorizes billing but does not grant a paid period.
	plan, price := uuid.NewString(), uuid.NewString()
	must(`INSERT INTO legui_private.billing_plans(id,code,name) VALUES(?,?,'Fixture membership')`, plan, "fixture_"+strings.ReplaceAll(plan, "-", ""))
	must(`INSERT INTO legui_private.billing_plan_prices(id,plan_id,currency,amount,interval_months,version,billing_unit) VALUES(?,?,'ARS',1000,1,1,'brand')`, price, plan)
	selection := request("POST", "/api/billing/subscribe", adminToken, "select-plan", "", map[string]any{"price_id": price, "brand_id": brand, "payer_email": admin + "@example.invalid"}, 201)
	subscriptionID := selection.Body["subscription_id"].(string)
	subCheckout := request("POST", "/api/billing/subscriptions/"+subscriptionID+"/checkout", adminToken, "sub-checkout", "", map[string]any{"payer_email": admin + "@example.invalid"}, 200)
	if subCheckout.Body["state"] != "created" {
		t.Fatalf("subscription checkout not created: %v", subCheckout.Body)
	}
	var sub checkout
	tx.Raw(`SELECT `+checkoutColumns+` FROM legui_private.mp_checkouts WHERE id=?`, subCheckout.Body["id"]).Scan(&sub)
	count(`SELECT count(*) FROM legui_private.subscriptions WHERE id=? AND status='incomplete' AND current_period_end IS NULL`, 1, subscriptionID)
	payments["2001"] = newPayment("2001", sub, "999", "1000.00")
	payments["2001"]["point_of_interaction"] = map[string]any{"transaction_data": map[string]any{"subscription_id": sub.RemoteID}}
	if state, e := h.observePayment(context.Background(), sub.ConnectionID, "2001"); e != nil || state != "applied" {
		t.Fatalf("membership charge failed: %s %v", state, e)
	}
	count(`SELECT count(*) FROM legui_private.subscriptions WHERE id=? AND status='active' AND current_period_end IS NOT NULL`, 1, subscriptionID)
	count(`SELECT count(*) FROM legui_private.billing_payments p JOIN legui_private.billing_invoices i ON i.id=p.invoice_id WHERE i.subscription_id=? AND p.status='succeeded'`, 1, subscriptionID)
	request("POST", "/api/billing/subscriptions/"+subscriptionID+"/cancel-renewal", adminToken, "cancel-sub", "", map[string]any{}, 200)
	count(`SELECT count(*) FROM legui_private.subscriptions WHERE id=? AND status='active' AND cancel_at_period_end`, 1, subscriptionID)
	request("GET", "/api/payments/mercadopago/transactions", ownerToken, "", "", nil, 200)
	request("GET", "/api/payments/mercadopago/reviews", adminToken, "", "", nil, 200)
	request("GET", "/api/billing/my-account", adminToken, "", "", nil, 200)
	// An uncertain POST is recovered with GET/search, never a second creation.
	uncertainOrder := request("POST", "/api/orders", adminToken, "uncertain-order", tenant, map[string]any{"channel": "web", "environment": "sandbox", "items": []any{map[string]any{"product_id": product, "quantity": 1}}}, 201).Body["id"].(string)
	mutex.Lock()
	ambiguousNext = true
	beforeCreates := creates
	mutex.Unlock()
	uncertain := request("POST", "/api/orders/"+uncertainOrder+"/checkout", adminToken, "uncertain-checkout", tenant, map[string]any{}, 200)
	if uncertain.Body["state"] != "review" {
		t.Fatal("uncertain provider outcome was not held")
	}
	request("POST", "/api/orders/"+uncertainOrder+"/checkout", adminToken, "uncertain-checkout", tenant, map[string]any{}, 200)
	if creates != beforeCreates+1 {
		t.Fatal("uncertain creation repeated POST")
	}
	must(`UPDATE legui_private.mp_checkouts SET reconcile_at=clock_timestamp()+interval '1 day' WHERE connection_id=? AND id<>?`, conn, uncertain.Body["id"])
	if e = h.reconcileOneCheckout(context.Background(), conn); e != nil {
		t.Fatal(e)
	}
	count(`SELECT count(*) FROM legui_private.mp_checkouts WHERE id=? AND state='created' AND checkout_url IS NOT NULL`, 1, uncertain.Body["id"])
	if creates != beforeCreates+1 {
		t.Fatal("reconciliation recreated checkout")
	}
	if e = h.scanOne(context.Background(), conn); e != nil {
		t.Fatal(e)
	}
	count(`SELECT count(*) FROM legui_private.mp_scan_cursors WHERE connection_id=? AND claim IS NULL AND page_offset=0`, 1, conn)
	// Late pending snapshots cannot undo a verified settlement.
	mutex.Lock()
	stale := newPayment("1001", co, "111", "200.00")
	stale["status"] = "pending"
	stale["date_last_updated"] = time.Now().Add(-time.Hour).Format(time.RFC3339Nano)
	payments["1001"] = stale
	mutex.Unlock()
	if state, e := h.observePayment(context.Background(), conn, "1001"); e != nil || state != "stale" {
		t.Fatalf("late status regressed payment: %s %v", state, e)
	}
	// Authentication and browser roles do not grant direct access to provider state.
	count(`SELECT CASE WHEN has_table_privilege('anon','legui_private.mp_connections','SELECT') OR has_table_privilege('authenticated','legui_private.mp_provider_payments','SELECT') THEN 1 ELSE 0 END`, 0)

	// Production guest flow still uses only the simulated provider and rollback DB.
	liveCfg := cfg
	liveCfg.Environment = "production"
	liveCfg.LiveEnabled = true
	liveHandler := NewHandler(tx, liveCfg, provider)
	oldMux := mux
	mux = http.NewServeMux()
	liveHandler.Register(mux)
	operations.RegisterRoutes(mux, tx, operations.BuyerOptions{OnlinePaymentAvailable: liveHandler.BuyerPaymentAvailable})
	liveConnection := uuid.NewString()
	encrypted, e := liveCfg.seal(jsonString(credentials{AccessToken: "business-token", RefreshToken: "business-refresh"}), "connection/"+liveConnection)
	if e != nil {
		t.Fatal(e)
	}
	must(`INSERT INTO legui_private.mp_connections(id,scope,brand_id,tenant_id,environment,state,collector_id,credential_cipher,expires_at) VALUES(?,'business',?,?,'production','connected','111',?,clock_timestamp()+interval '1 day')`, liveConnection, brand, tenant, encrypted)
	must(`UPDATE public.tenants SET menu_published=true,ordering_enabled=true WHERE id=?`, tenant)
	guest := base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("g", 32)))
	guestBasket := map[string]any{"fulfillment": "pickup", "items": []any{map[string]any{"product_id": product, "quantity": 1}}}
	quote := request("POST", "/api/public/menu/"+tenant+"/quote", "", "", "", guestBasket, 200)
	guestPayload := map[string]any{"fulfillment": "pickup", "items": guestBasket["items"], "quote_digest": quote.Body["digest"], "name": "Guest buyer", "phone": "1155550123", "payment_method": "mercadopago"}
	guestOrder := request("POST", "/api/public/menu/"+tenant+"/orders", guest, uuid.NewString(), "", guestPayload, 201).Body["id"].(string)
	guestKey := uuid.NewString()
	request("POST", "/api/public/buyer-order/mercadopago", guest, guestKey, otherTenant, map[string]any{}, 409)
	privateOrder := request("GET", "/api/orders/"+guestOrder, adminToken, "", tenant, nil, 200)
	request("PATCH", "/api/orders/"+guestOrder+"/status", adminToken, uuid.NewString(), tenant, map[string]any{"status": "accepted", "expected_version": privateOrder.Body["version"]}, 200)
	request("POST", "/api/public/buyer-order/mercadopago", guest, guestKey, otherTenant, map[string]any{"amount": "0.01"}, 400)
	request("POST", "/api/public/buyer-order/mercadopago", buyeraccess.TrackingToken(guest), guestKey, tenant, map[string]any{}, 404)
	beforeGuestCreates := creates
	guestCheckout := request("POST", "/api/public/buyer-order/mercadopago", guest, guestKey, otherTenant, map[string]any{}, 200)
	request("POST", "/api/public/buyer-order/mercadopago", guest, guestKey, otherTenant, map[string]any{}, 200)
	if creates != beforeGuestCreates+1 || guestCheckout.Body["state"] != "created" || guestCheckout.Body["amount"] != "100.00" {
		t.Fatalf("guest checkout failed %v", guestCheckout.Body)
	}
	var gc checkout
	tx.Raw(`SELECT `+checkoutColumns+` FROM legui_private.mp_checkouts WHERE id=?`, guestCheckout.Body["id"]).Scan(&gc)
	if gc.OrderID != guestOrder || gc.TenantID != tenant || gc.ActorKind != "buyer" {
		t.Fatal("guest checkout escaped capability scope")
	}
	mutex.Lock()
	preference := preferences[gc.RemoteID]
	back := preference["back_urls"].(map[string]any)["success"].(string)
	mutex.Unlock()
	if !strings.Contains(back, "/catalogo/"+tenant) || strings.Contains(back, guest) {
		t.Fatal("unsafe buyer payment return")
	}
	payments["3001"] = newPayment("3001", gc, "111", "100.00")
	payments["3001"]["live_mode"] = true
	if state, e := liveHandler.observePayment(context.Background(), liveConnection, "3001"); e != nil || state != "applied" {
		t.Fatalf("guest payment settlement %s %v", state, e)
	}
	guestReceipt := request("GET", "/api/public/buyer-order", guest, "", "", nil, 200)
	if guestReceipt.Body["total_paid"] != "100.00" {
		t.Fatal("verified guest payment missing from receipt")
	}
	request("GET", "/api/orders/"+guestOrder, guest, "", tenant, nil, 401)
	mux = oldMux
	// An archived business submits only prepared cancellation commands through
	// the normal journal. Re-running a worker cannot send the command again.
	must(`INSERT INTO legui_private.business_closures(brand_id,actor_id,reason) VALUES(?,?,'Fixture closure worker')`, brand, admin)
	must(`UPDATE public.brands SET closed_at=now(),suspended_at=now(),suspension_reason='Fixture closure' WHERE id=?`, brand)
	closureOperation := uuid.NewString()
	must(`INSERT INTO legui_private.mp_operations(id,connection_id,kind,target_id,resource_id,operation_key,request_digest,actor_kind,actor_id) VALUES(?,?,'cancel_subscription',?,?,?,?,'user',?)`, closureOperation, sub.ConnectionID, subscriptionID, sub.RemoteID, "closure:"+subscriptionID, hash("cancel:"+subscriptionID), admin)
	beforeCancel := cancelWrites
	if e := h.submitClosureCancellation(context.Background()); e != nil {
		t.Fatal(e)
	}
	count(`SELECT count(*) FROM legui_private.mp_operations WHERE id=? AND state='completed'`, 1, closureOperation)
	if e := h.submitClosureCancellation(context.Background()); e != nil {
		t.Fatal(e)
	}
	if cancelWrites != beforeCancel+1 {
		t.Fatal("closure cancellation was lost or submitted twice")
	}
	count(`SELECT count(*) FROM legui_private.subscriptions WHERE id=? AND status='active' AND cancel_at_period_end`, 1, subscriptionID)
	t.Log("Mercado Pago integration passed: OAuth encryption, ownership, durable checkout/replay, signed inbox, verified charge, refunds, reviews, membership settlement/cancellation, closure worker replay and role boundaries. All DB fixtures roll back; upstream is simulated.")
}
