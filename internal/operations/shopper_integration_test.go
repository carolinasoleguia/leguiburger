package operations

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"leguiburger/internal/auth"
	"leguiburger/internal/brands"
	"leguiburger/internal/businesscare"
	"leguiburger/internal/db"
	"leguiburger/internal/mercadopago"
	"leguiburger/internal/orderstream"
	"leguiburger/internal/products"
	"leguiburger/internal/recipes"
	"leguiburger/internal/shopperidentity"
	"leguiburger/internal/supplies"
	"leguiburger/internal/tenantproducts"
	"leguiburger/internal/tenants"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// Guest purchase through real handlers against two unrelated restaurant fixtures.
// Everything (including sessions, financial ledgers and receipts) is rolled back.
func TestShopperAccountsIntegration(t *testing.T) {
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
	t.Setenv("AUTH_REQUIRE_PLATFORM_MFA", "false")
	t.Setenv("PUBLIC_SITE_URL", "http://127.0.0.1:5173")
	t.Setenv("APP_ENV", "development")
	if e = auth.ConfigureJWTSecret("tenant-isolation-fixture-secret-for-rollback-tests"); e != nil {
		t.Fatal(e)
	}
	must := func(q string, args ...any) {
		t.Helper()
		// Fixture writes are outside an HTTP actor; never inherit the last
		// counter session across this rollback transaction's savepoints.
		if e := tx.Exec(`SELECT set_config('leguiburger.actor_user_id','',true),set_config('leguiburger.actor_employee_id','',true),set_config('leguiburger.operation_key','',true),set_config('leguiburger.request_digest','',true),set_config('leguiburger.source_order_id','',true)`).Error; e != nil {
			t.Fatal(e)
		}

		if e := tx.Exec(q, args...).Error; e != nil {
			t.Fatal(e)
		}
	}
	b1, b2, t1, t2, t3 := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	p1, p2, s1, s2 := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	owner, a1, a2, cashier, kitchen := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	must(`INSERT INTO public.brands(id,name,base_location_limit,base_collaborator_limit) VALUES(?,'__isolation_brand_a',3,5),(?,'__isolation_brand_b',3,5)`, b1, b2)
	paidLegacyFixture(t, tx, b1)
	paidLegacyFixture(t, tx, b2)
	must(`INSERT INTO public.tenants(id,brand_id,subdomain) VALUES(?,?,'__isolation_a'),(?,?,'__isolation_b'),(?,?,'__isolation_a2')`, t1, b1, t2, b2, t3, b1)
	must(`INSERT INTO public.users(id,email,password_hash,role,brand_id,is_active) VALUES(?,?,'fixture','owner',NULL,true),(?,?,'fixture','admin',?,true),(?,?,'fixture','admin',?,true)`, owner, owner+"@example.invalid", a1, a1+"@example.invalid", b1, a2, a2+"@example.invalid", b2)
	must(`INSERT INTO public.employees(id,tenant_id,first_name,last_name,email,password_hash,role,is_active) VALUES(?,?,'Fixture','Cashier',?,'fixture','cashier',true),(?,?,'Fixture','Kitchen',?,'fixture','kitchen',true)`, cashier, t1, cashier+"@example.invalid", kitchen, t1, kitchen+"@example.invalid")
	must(`INSERT INTO public.products(id,brand_id,name,base_price,is_active) VALUES(?,?,'__isolation_product_a',100,true),(?,?,'__isolation_product_b',100,true)`, p1, b1, p2, b2)
	must(`INSERT INTO public.tenant_products(tenant_id,product_id,current_stock,track_stock,is_available,is_active) VALUES(?,?,10,true,true,true),(?,?,10,true,true,true)`, t1, p1, t2, p2)
	must(`INSERT INTO public.supplies(id,tenant_id,name,current_wholesale_cost,current_stock,measurement_unit,is_active) VALUES(?,?,'__isolation_supply_a',10,100,'unit',true),(?,?,'__isolation_supply_b',10,100,'unit',true)`, s1, t1, s2, t2)
	must(`INSERT INTO public.recipes(product_id,supply_id,quantity_used) VALUES(?,?,1),(?,?,1)`, p1, s1, p2, s2)
	token := func(id, kind, role, brand, tenant string) string {
		t.Helper()
		sid := uuid.NewString()
		must(`INSERT INTO legui_private.auth_sessions(id,principal_kind,principal_id,role,brand_id,tenant_id,credential_digest,mfa_verified) VALUES(?,?,?,?,nullif(?,'')::uuid,nullif(?,'')::uuid,?,false)`, sid, kind, id, role, brand, tenant, fmt.Sprintf("%x", sha256.Sum256([]byte("fixture"))))
		v, e := auth.SessionToken(auth.SessionPrincipal{ID: id, Kind: kind, Role: role, BrandID: brand, TenantID: tenant}, sid, false, "access")
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	tokens := map[string]string{"owner": token(owner, "user", "owner", "", ""), "admin_a": token(a1, "user", "admin", b1, ""), "admin_b": token(a2, "user", "admin", b2, ""), "cashier": token(cashier, "employee", "cashier", b1, t1), "kitchen": token(kitchen, "employee", "kitchen", b1, t1)}
	mux := http.NewServeMux()
	hub := orderstream.New()
	streamCtx, stopStream := context.WithCancel(context.Background())
	defer stopStream()
	go hub.Run(streamCtx, os.Getenv("DATABASE_URL"))
	streamWait := time.Now().Add(15 * time.Second)
	for !hub.Ready() && time.Now().Before(streamWait) {
		time.Sleep(100 * time.Millisecond)
	}
	if !hub.Ready() {
		t.Fatal("dedicated SSE listener unavailable")
	}
	RegisterRoutes(mux, tx, BuyerOptions{Stream: hub})
	shopperidentity.Register(mux, tx)
	businesscare.Register(mux, tx)
	mercadopago.NewHandler(tx, mercadopago.Config{}, nil).Register(mux)
	auth.RegisterSecurityRoutes(mux, tx, nil)
	for prefix, handler := range map[string]http.HandlerFunc{
		"/api/products":        products.NewHandler(products.NewService(products.NewRepository(), brands.NewRepository())).HandleProductRoutes,
		"/api/tenant-products": tenantproducts.NewHandler(tenantproducts.NewService(tenantproducts.NewRepository(), tenants.NewRepository())).HandleTenantProductRoutes,
		"/api/supplies":        supplies.NewHandler(supplies.NewService(supplies.NewRepository(), tenants.NewRepository())).HandleSupplyRoutes,
		"/api/recipes":         recipes.NewHandler(recipes.NewService(recipes.NewRepository(), tenants.NewRepository())).HandleRecipeRoutes,
		"/api/tenants":         tenants.NewHandler(tenants.NewService(tenants.NewRepository(), brands.NewRepository())).HandleTenantRoutes,
	} {
		wrapped := auth.AuthMiddlewareWithDB(tx, auth.AuthorizeLegacy(handler))
		mux.HandleFunc(prefix, wrapped)
		mux.HandleFunc(prefix+"/", wrapped)
	}
	checks := 0
	request := func(method, path, actor, tenant string, headers map[string]string, body any, want int) *httptest.ResponseRecorder {
		t.Helper()
		encoded, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(encoded))
		if actor != "" {
			r.Header.Set("Authorization", "Bearer "+tokens[actor])
		}
		if tenant != "" {
			r.Header.Set("X-Tenant-ID", tenant)
		}
		r.Header.Set("Idempotency-Key", uuid.NewString())
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		checks++
		if w.Code != want {
			t.Fatalf("%s %s actor=%s want=%d got=%d body=%s", method, path, actor, want, w.Code, w.Body.String())
		}
		if want >= 400 && strings.Contains(w.Body.String(), "__isolation_") {
			t.Fatal("denied response leaks private fixture data")
		}
		return w
	}

	must(`UPDATE public.tenants SET menu_published=true,ordering_enabled=true WHERE id IN (?,?)`, t1, t2)
	parse := func(w *httptest.ResponseRecorder) map[string]any {
		t.Helper()
		var v map[string]any
		if e := json.Unmarshal(w.Body.Bytes(), &v); e != nil {
			t.Fatal(e)
		}
		return v
	}
	makeAccount := func(label string) (string, map[string]string) {
		t.Helper()
		created := request("POST", "/api/shopper/register", "", "", nil, map[string]any{"name": "Buyer " + label, "email": uuid.NewString() + "@example.invalid", "password": "Buyer-Fixture-Pass123!"}, 200)
		profile := parse(created)
		c := created.Result().Cookies()
		if len(c) != 1 || !c[0].HttpOnly || c[0].MaxAge != 7*86400 {
			t.Fatal("shopper cookie attributes")
		}
		return profile["id"].(string), map[string]string{"Cookie": c[0].Name + "=" + c[0].Value}
	}
	a, ca := makeAccount("A")
	b, cb := makeAccount("B")
	request("GET", "/api/shopper/me", "", "", nil, nil, 401)
	if parse(request("GET", "/api/shopper/me", "", "", ca, nil, 200))["id"] != a {
		t.Fatal("wrong profile")
	}
	request("GET", "/api/orders", "", t1, ca, nil, 401)
	request("GET", "/api/shopper/orders", "", "", nil, nil, 401)
	quoteAndCreate := func(tenant, product, code string, cookie map[string]string, want string) string {
		t.Helper()
		basket := map[string]any{"fulfillment": "pickup", "items": []any{map[string]any{"product_id": product, "quantity": 1}}}
		if code != "" {
			basket["coupon_code"] = code
		}
		q := parse(request("POST", "/api/public/menu/"+tenant+"/quote", "", "", cookie, basket, 200))
		if q["total"] != want {
			t.Fatalf("wrong discounted total %v", q)
		}
		basket["quote_digest"] = q["digest"]
		basket["name"] = "Shopper Fixture"
		basket["phone"] = "1155550123"
		basket["payment_method"] = "at_local"
		headers := map[string]string{"Authorization": "Bearer " + base64.RawURLEncoding.EncodeToString([]byte(uuid.NewString()[:32]))}
		for k, v := range cookie {
			headers[k] = v
		}
		return parse(request("POST", "/api/public/menu/"+tenant+"/orders", "", "", headers, basket, 201))["id"].(string)
	}
	request("PATCH", "/api/shopper/profile", "", "", nil, map[string]any{"name": "Buyer A", "phone": "1155554444", "delivery_address": "Calle habitual 123"}, 401)
	request("PATCH", "/api/shopper/profile", "", "", ca, map[string]any{"name": "Buyer A", "phone": "1155554444", "delivery_address": "Calle habitual 123"}, 200)
	request("PATCH", "/api/shopper/profile", "", "", ca, map[string]any{"name": "Buyer A", "phone": "invalid", "delivery_address": ""}, 400)
	if parse(request("GET", "/api/shopper/me", "", "", ca, nil, 200))["phone"] != "1155554444" {
		t.Fatal("saved shopper contact missing")
	}
	if parse(request("GET", "/api/shopper/me", "", "", cb, nil, 200))["phone"] != "" {
		t.Fatal("another buyer inherited contact")
	}
	first := quoteAndCreate(t1, p1, "", ca, "100.00")
	request("POST", "/api/marketing/coupons", "admin_a", t2, nil, map[string]any{}, 403)
	request("POST", "/api/marketing/coupons", "cashier", t1, nil, map[string]any{}, 403)
	input := map[string]any{"code": "VOLVE10", "percent": 10, "minimum": "100.00", "max_uses": 2, "days": 7, "shopper_id": a, "all_locations": false}
	created := parse(request("POST", "/api/marketing/coupons", "admin_a", t1, nil, input, 201))
	id := created["id"].(string)
	request("PATCH", "/api/marketing/coupons/"+id, "admin_b", t2, nil, map[string]any{"active": false}, 404)
	input["code"] = "AJENO"
	input["shopper_id"] = b
	request("POST", "/api/marketing/coupons", "admin_a", t1, nil, input, 404)
	basket := map[string]any{"fulfillment": "pickup", "items": []any{map[string]any{"product_id": p1, "quantity": 1}}, "coupon_code": "VOLVE10"}
	request("POST", "/api/public/menu/"+t1+"/quote", "", "", nil, basket, 401)
	request("POST", "/api/public/menu/"+t1+"/quote", "", "", cb, basket, 409)
	discounted := quoteAndCreate(t1, p1, "VOLVE10", ca, "90.00")
	request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, basket, 409)
	third := quoteAndCreate(t2, p2, "", ca, "100.00")
	history := parse(request("GET", "/api/shopper/orders", "", "", ca, nil, 200))["items"].([]any)
	if len(history) != 3 {
		t.Fatalf("global shopper history wrong %d", len(history))
	}
	if len(parse(request("GET", "/api/shopper/orders?shopper_id="+a, "", "", cb, nil, 200))["items"].([]any)) != 0 {
		t.Fatal("another buyer's history leaked")
	}
	ownerHeaders := map[string]string{"Cookie": ca["Cookie"], "X-Shopper-Order": discounted}
	receipt := parse(request("GET", "/api/public/buyer-order", "", "", ownerHeaders, nil, 200))
	if receipt["id"] != discounted || receipt["total_due"] != "90.00" || receipt["account_payment_key"] == nil {
		t.Fatalf("account receipt invalid %v", receipt)
	}
	thief := map[string]string{"Cookie": cb["Cookie"], "X-Shopper-Order": discounted}
	request("GET", "/api/public/buyer-order", "", "", thief, nil, 404)
	current := parse(request("GET", "/api/orders/"+discounted, "cashier", t1, nil, nil, 200))
	request("PATCH", "/api/orders/"+discounted+"/status", "cashier", t1, nil, map[string]any{"status": "accepted", "expected_version": current["version"]}, 200)
	var net, discount string
	if e := tx.Raw(`SELECT sum(total_net_revenue)::text,sum(total_discount_applied)::text FROM public.sales WHERE order_id=?`, discounted).Row().Scan(&net, &discount); e != nil {
		t.Fatal(e)
	}
	if net != "90.00" || discount != "10.00" {
		t.Fatalf("coupon ledger inconsistent net=%s discount=%s", net, discount)
	}
	request("PATCH", "/api/shopper/preferences", "", "", ca, map[string]any{"brand_id": b1, "newsletter": true}, 200)
	prefs := request("GET", "/api/shopper/preferences", "", "", ca, nil, 200).Body.String()
	if !strings.Contains(prefs, `"newsletter": true`) {
		t.Fatal("newsletter preference missing")
	}
	customers := parse(request("GET", "/api/marketing/customers", "admin_a", t1, nil, nil, 200))["items"].([]any)
	if len(customers) != 1 || customers[0].(map[string]any)["orders"] != float64(2) || customers[0].(map[string]any)["newsletter_confirmed"] != false {
		t.Fatalf("brand customer privacy or consent failed %v", customers)
	}
	// Secret possession, never matching email, is required to attach guest orders.
	request("POST", "/api/shopper/claim", "", "", cb, map[string]any{"tenant_id": t1}, 404)

	if customers[0].(map[string]any)["spent"] != nil || customers[0].(map[string]any)["favorite_product"] != nil {
		t.Fatal("personalization without consent")
	}
	request("PATCH", "/api/shopper/preferences", "", "", ca, map[string]any{"brand_id": b1, "personalization": true}, 200)
	request("PATCH", "/api/shopper/preferences", "", "", cb, map[string]any{"brand_id": b2, "personalization": true}, 200)
	enabled := parse(request("GET", "/api/marketing/customers", "admin_a", t1, nil, nil, 200))["items"].([]any)[0].(map[string]any)
	if enabled["spent"] != "0.00" || enabled["segment"] != "Primera compra" || enabled["orders"] != float64(2) {
		t.Fatalf("scoped shopper metrics invalid %v", enabled)
	}
	request("POST", "/api/orders/"+discounted+"/payments", "cashier", t1, nil, map[string]any{"kind": "charge", "method": "transfer", "amount": "90.00"}, 201)
	enabled = parse(request("GET", "/api/marketing/customers", "admin_a", t1, nil, nil, 200))["items"].([]any)[0].(map[string]any)
	if enabled["spent"] != "90.00" || enabled["favorite_product"] != "__isolation_product_a" || enabled["segment"] != "En crecimiento" {
		t.Fatalf("purchase profiling invalid %v", enabled)
	}
	request("PATCH", "/api/shopper/preferences", "", "", ca, map[string]any{"brand_id": b1, "personalization": false}, 200)
	disabled := parse(request("GET", "/api/marketing/customers", "admin_a", t1, nil, nil, 200))["items"].([]any)[0].(map[string]any)
	if disabled["spent"] != nil || disabled["segment"] != nil {
		t.Fatal("withdrawn personalization still exposed")
	}
	if !strings.Contains(request("GET", "/api/shopper/preferences", "", "", ca, nil, 200).Body.String(), `"newsletter": true`) {
		t.Fatal("personalization toggle erased newsletter consent")
	}
	var count int64
	tx.Raw(`SELECT count(*) FROM public.sales s JOIN legui_private.shopper_orders a ON a.order_id=s.order_id WHERE a.shopper_id=?`, a).Scan(&count)
	if count != 1 {
		t.Fatal("sales cannot be joined to shopper ID")
	}
	// Match PostgreSQL's per-unit currency rounding, including half-cent discounts.
	must(`UPDATE public.products SET base_price=0.05 WHERE id=?`, p1)
	request("POST", "/api/marketing/coupons", "admin_a", t1, nil, map[string]any{"code": "CENTAVOS", "percent": 10, "minimum": "0", "max_uses": 5, "days": 7, "shopper_id": a, "all_locations": false}, 201)
	tiny := map[string]any{"fulfillment": "pickup", "coupon_code": "CENTAVOS", "items": []any{map[string]any{"product_id": p1, "quantity": 3}}}
	tinyQuote := parse(request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, tiny, 200))
	if tinyQuote["total"] != "0.12" || tinyQuote["discount"] != "0.03" {
		t.Fatalf("wrong currency rounding %v", tinyQuote)
	}
	tiny["quote_digest"] = tinyQuote["digest"]
	tiny["name"] = "Rounding Fixture"
	tiny["phone"] = "1155550123"
	tiny["payment_method"] = "at_local"
	tiny["remember_contact"] = true
	tinyHeaders := map[string]string{"Cookie": ca["Cookie"], "Authorization": "Bearer " + base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("z", 32)))}
	tinyReceipt := parse(request("POST", "/api/public/menu/"+t1+"/orders", "", "", tinyHeaders, tiny, 201))
	if tinyReceipt["total_due"] != "0.12" || tinyReceipt["discount"] != "0.03" {
		t.Fatalf("persisted currency rounding differs %v", tinyReceipt)
	}
	stored := parse(request("GET", "/api/shopper/me", "", "", ca, nil, 200))
	if stored["phone"] != "1155550123" || stored["delivery_address"] != "Calle habitual 123" {
		t.Fatalf("opt-in contact save failed %v", stored)
	}
	profile := parse(request("GET", "/api/shopper/me", "", "", ca, nil, 200))
	request("POST", "/api/shopper/login", "", "", nil, map[string]any{"email": profile["email"], "password": "Incorrect-Fixture-Pass123!"}, 401)
	login := request("POST", "/api/shopper/login", "", "", nil, map[string]any{"email": profile["email"], "password": "Buyer-Fixture-Pass123!"}, 200)
	if len(login.Result().Cookies()) != 1 {
		t.Fatal("buyer login did not issue a cookie")
	}
	verifyRaw := shopperidentity.Random()
	must(`INSERT INTO legui_private.shopper_actions(digest,shopper_id,purpose) VALUES(?,?,'verify')`, shopperidentity.Digest(verifyRaw), a)
	request("POST", "/api/shopper/verify", "", "", nil, map[string]any{"token": verifyRaw}, 200)
	request("POST", "/api/shopper/verify", "", "", nil, map[string]any{"token": verifyRaw}, 400)
	if parse(request("GET", "/api/shopper/me", "", "", ca, nil, 200))["email_verified"] != true {
		t.Fatal("verification not reflected")
	}
	reset := shopperidentity.Random()
	must(`INSERT INTO legui_private.shopper_actions(digest,shopper_id,purpose) VALUES(?,?,'reset')`, shopperidentity.Digest(reset), b)
	request("POST", "/api/shopper/reset", "", "", nil, map[string]any{"token": reset, "password": "Changed-Fixture-Pass123!"}, 200)
	request("GET", "/api/shopper/me", "", "", cb, nil, 401)
	request("POST", "/api/shopper/reset", "", "", nil, map[string]any{"token": reset, "password": "Changed-Fixture-Pass123!"}, 400)
	request("POST", "/api/shopper/password", "", "", ca, map[string]any{"current_password": "Buyer-Fixture-Pass123!", "password": "Another-Fixture-Pass123!"}, 200)
	request("GET", "/api/shopper/me", "", "", ca, nil, 401)
	fresh := request("POST", "/api/shopper/login", "", "", nil, map[string]any{"email": profile["email"], "password": "Another-Fixture-Pass123!"}, 200)
	ca["Cookie"] = fresh.Result().Cookies()[0].Name + "=" + fresh.Result().Cookies()[0].Value
	ownerHeaders["Cookie"] = ca["Cookie"]
	request("POST", "/api/shopper/logout", "", "", ca, map[string]any{}, 200)
	request("GET", "/api/shopper/me", "", "", ca, nil, 401)
	request("GET", "/api/public/buyer-order", "", "", ownerHeaders, nil, 404)
	_ = first
	_ = third
	t.Logf("%d shopper account, cross-brand privacy, targeting, exact coupon totals, sales ledger, consent and logout checks; fixtures rolled back", checks)
}
