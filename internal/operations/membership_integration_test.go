package operations

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"leguiburger/internal/auth"
	"leguiburger/internal/db"
	"leguiburger/internal/mercadopago"
	"leguiburger/internal/tenantproducts"
	"leguiburger/internal/tenants"
)

var membershipIntegration = flag.Bool("membership-db-integration", false, "Rollback-only membership, access and terminal integration")

func TestMembershipIntegration(t *testing.T) {
	if !*membershipIntegration {
		t.Skip("opt-in DB integration")
	}
	_ = godotenv.Load("../../.env")
	database, err := gorm.Open(postgres.New(postgres.Config{DSN: os.Getenv("DATABASE_URL"), PreferSimpleProtocol: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("database unavailable")
	}
	pool, _ := database.DB()
	defer pool.Close()
	if err = VerifyDatabase(database, os.Getenv("DATABASE_URL")); err != nil {
		t.Fatal(err)
	}
	tx := database.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	previous := db.DB
	db.DB = tx
	defer func() { db.DB = previous }()
	t.Setenv("JWT_SECRET", "membership-integration-fixture-secret")
	t.Setenv("APP_ENV", "development")
	t.Setenv("AUTH_REQUIRE_PLATFORM_MFA", "false")
	must := func(q string, args ...any) {
		t.Helper()
		if e := tx.Exec(q, args...).Error; e != nil {
			t.Fatal(e)
		}
	}
	b, expiredBrand, local, expiredLocal, admin, expiredAdmin, owner, employee, expiredEmployee, product := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	must(`INSERT INTO public.brands(id,name) VALUES(?,'__membership_http'),(?,'__membership_expired')`, b, expiredBrand)
	must(`INSERT INTO public.tenants(id,brand_id,subdomain,menu_published,ordering_enabled) VALUES(?,?,'__membership_http',true,true),(?,?,'__membership_expired',true,true)`, local, b, expiredLocal, expiredBrand)
	must(`INSERT INTO legui_private.membership_trials(brand_id,started_at) VALUES(?,now()-interval '8 days')`, expiredBrand)
	password := "membership-fixture-password-123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	must(`INSERT INTO public.users(id,email,password_hash,role,brand_id,is_active) VALUES(?,?,?,'admin',?,true),(?,?,?,'admin',?,true),(?,?,?,'owner',NULL,true)`, admin, admin+"@example.invalid", string(hash), b, expiredAdmin, expiredAdmin+"@example.invalid", string(hash), expiredBrand, owner, owner+"@example.invalid", string(hash))
	must(`INSERT INTO public.employees(id,tenant_id,first_name,last_name,email,password_hash,role,is_active) VALUES(?,?,'Fixture','Cashier',?,?,'cashier',true),(?,?,'Expired','Cashier',?,?,'cashier',true)`, employee, local, employee+"@example.invalid", string(hash), expiredEmployee, expiredLocal, expiredEmployee+"@example.invalid", string(hash))
	must(`INSERT INTO public.products(id,brand_id,name,base_price,is_active) VALUES(?,?,'Fixture burger',100,true)`, product, b)
	must(`INSERT INTO public.tenant_products(tenant_id,product_id,track_stock,current_stock) VALUES(?,?,true,0)`, local, product)
	svc, e := auth.NewService(auth.NewRepository(), tenants.NewRepository())
	if e != nil {
		t.Fatal(e)
	}
	mux := http.NewServeMux()
	auth.RegisterSecurityRoutes(mux, tx, svc)
	RegisterRoutes(mux, tx)
	mercadopago.NewHandler(tx, mercadopago.Config{Environment: "sandbox"}, nil).Register(mux)
	legacy := tenantproducts.NewHandler(tenantproducts.NewService(tenantproducts.NewRepository(), tenants.NewRepository()))
	mux.HandleFunc("/api/tenant-products", auth.AuthMiddlewareWithDB(tx, auth.AuthorizeLegacy(legacy.HandleTenantProductRoutes)))
	mux.HandleFunc("/api/tenant-products/", auth.AuthMiddlewareWithDB(tx, auth.AuthorizeLegacy(legacy.HandleTenantProductRoutes)))
	checks := 0
	request := func(method, path, token, tenant string, body any, status int, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		raw, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(raw))
		r.RemoteAddr = "192.0.2.81:1234"
		r.Header.Set("Idempotency-Key", uuid.NewString())
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if tenant != "" {
			r.Header.Set("X-Tenant-ID", tenant)
		}
		for _, c := range cookies {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		checks++
		if w.Code != status {
			t.Fatalf("%s %s expected %d got %d: %s", method, path, status, w.Code, w.Body.String())
		}
		return w
	}
	obj := func(w *httptest.ResponseRecorder) map[string]any {
		t.Helper()
		var v map[string]any
		if e := json.Unmarshal(w.Body.Bytes(), &v); e != nil {
			t.Fatal(e)
		}
		return v
	}
	login := func(id string, status int, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		return request("POST", "/api/auth/login", "", "", map[string]any{"email": id + "@example.invalid", "password": password}, status, cookies...)
	}
	adminToken := obj(login(admin, 200))["token"].(string)
	ownerToken := obj(login(owner, 200))["token"].(string)
	first := login(employee, 200)
	staffToken := obj(first)["token"].(string)
	login(employee, 200)
	login(employee, 409)
	replacement := login(employee, 200, first.Result().Cookies()...)
	staffToken = obj(replacement)["token"].(string)
	if access := obj(request("GET", "/api/membership", adminToken, "", nil, 200)); access["tier"] != "trial" || access["premium"] != false {
		t.Fatal("Wrong trial access")
	}
	request("GET", "/api/inventory", adminToken, local, nil, 403)
	request("GET", "/api/expenses", adminToken, local, nil, 403)
	request("GET", "/api/billing/terminals", staffToken, "", nil, 403)
	live := obj(request("GET", "/api/billing/terminals", adminToken, "", nil, 200))["items"].([]any)
	if len(live) != 2 {
		t.Fatalf("Expected two terminals, got %d", len(live))
	}
	sessionID := live[0].(map[string]any)["id"].(string)
	request("DELETE", "/api/billing/terminals/"+sessionID, adminToken, "", map[string]any{}, 200)
	login(employee, 200)
	request("PUT", "/api/tenant-products/"+product, adminToken, local, map[string]any{"current_stock": 99}, 403)
	request("PUT", "/api/tenant-products/"+product, adminToken, local, map[string]any{"is_available": true}, 200)
	data := request("GET", "/api/tenant-products", adminToken, local, nil, 200).Body.String()
	if bytes.Contains([]byte(data), []byte("current_stock")) {
		t.Fatal("Stock leaked to basic plan")
	}
	report := request("GET", "/api/analytics/restaurant", adminToken, local, nil, 200).Body.String()
	if bytes.Contains([]byte(report), []byte("operating_expenses")) {
		t.Fatal("Premium metrics leaked")
	}
	menu := obj(request("GET", "/api/public/menu/"+local, "", "", nil, 200))
	if menu["ordering_ready"] != true {
		t.Fatal("Trial menu unexpectedly closed")
	}
	if menu["items"].([]any)[0].(map[string]any)["available"] != true {
		t.Fatal("Basic menu was blocked by retained ERP stock")
	}
	expiredToken := obj(login(expiredAdmin, 200))["token"].(string)
	expiredStaff := obj(login(expiredEmployee, 200))["token"].(string)
	request("GET", "/api/analytics/restaurant", expiredToken, expiredLocal, nil, 403)
	request("GET", "/api/orders", expiredStaff, expiredLocal, nil, 403)
	request("GET", "/api/membership", expiredStaff, "", nil, 200)
	request("GET", "/api/billing/my-account", expiredToken, "", nil, 200)
	request("GET", "/api/billing/plans", expiredStaff, "", nil, 403)
	request("DELETE", "/api/billing/terminals/"+sessionID, expiredToken, "", map[string]any{}, 403)
	if obj(request("GET", "/api/public/menu/"+expiredLocal, "", "", nil, 200))["ordering_ready"] != false {
		t.Fatal("Expired brand accepts new orders")
	}
	plan := obj(request("POST", "/api/platform/billing/plans", ownerToken, "", map[string]any{"code": "fixture_" + uuid.NewString()[:8], "name": "Fixture Premium"}, 201))["id"].(string)
	draft := obj(request("POST", "/api/platform/billing/plans/"+plan+"/prices", ownerToken, "", map[string]any{"currency": "ARS", "amount": "0", "interval_months": 1, "billing_unit": "brand", "membership_tier": "premium", "extra_location_amount": "0"}, 201))["id"].(string)
	request("PATCH", "/api/platform/billing/prices/"+draft+"/publication", adminToken, "", map[string]any{"published": true}, 403)
	request("PATCH", "/api/platform/billing/prices/"+draft+"/publication", ownerToken, "", map[string]any{"published": true}, 422)
	price := obj(request("POST", "/api/platform/billing/plans/"+plan+"/prices", ownerToken, "", map[string]any{"currency": "ARS", "amount": "1000", "interval_months": 1, "billing_unit": "brand", "membership_tier": "premium", "extra_location_amount": "250", "published": true}, 201))["id"].(string)
	account := obj(request("POST", "/api/platform/billing/accounts", ownerToken, "", map[string]any{"brand_id": b, "currency": "ARS"}, 201))["id"].(string)
	sub := obj(request("POST", "/api/platform/billing/subscriptions", ownerToken, "", map[string]any{"account_id": account, "plan_price_id": price, "environment": "production", "status": "incomplete", "purchased_locations": 3}, 201))["id"].(string)
	request("GET", "/api/inventory", adminToken, local, nil, 403)
	must(`UPDATE legui_private.subscriptions SET status='active',current_period_start=now()-interval '1 hour',current_period_end=now()+interval '1 month' WHERE id=?`, sub)
	request("GET", "/api/inventory", adminToken, local, nil, 200)
	request("GET", "/api/expenses", adminToken, local, nil, 200)
	var amount string
	tx.Raw(`SELECT contracted_amount::text FROM legui_private.subscriptions WHERE id=?`, sub).Scan(&amount)
	if amount != "1500.00" {
		t.Fatal("Additional stores were not included exactly: " + amount)
	}
	// An invalid publication cannot retire the current price; a valid new version
	// replaces only the offer, never the amount already contracted by this brand.
	request("PATCH", "/api/platform/billing/prices/"+draft+"/publication", ownerToken, "", map[string]any{"published": true}, 422)
	var published bool
	must(`SELECT is_active FROM legui_private.billing_plan_prices WHERE id=?`, price)
	if e := tx.Raw(`SELECT is_active FROM legui_private.billing_plan_prices WHERE id=?`, price).Scan(&published).Error; e != nil || !published {
		t.Fatal("Failed publication retired a valid price")
	}
	newPrice := obj(request("POST", "/api/platform/billing/plans/"+plan+"/prices", ownerToken, "", map[string]any{"currency": "ARS", "amount": "2000", "interval_months": 1, "billing_unit": "brand", "membership_tier": "premium", "extra_location_amount": "500", "published": true}, 201))["id"].(string)
	if e := tx.Raw(`SELECT is_active FROM legui_private.billing_plan_prices WHERE id=?`, price).Scan(&published).Error; e != nil || published {
		t.Fatal("Superseded commercial offer remained published")
	}
	offers := obj(request("GET", "/api/billing/plans", adminToken, "", nil, 200))["items"].([]any)
	if len(offers) != 1 || offers[0].(map[string]any)["id"] != newPrice {
		t.Fatal("Plan listing did not select the current published version")
	}
	request("GET", "/api/inventory", adminToken, local, nil, 200)
	if e := tx.Raw(`SELECT contracted_amount::text FROM legui_private.subscriptions WHERE id=?`, sub).Scan(&amount).Error; e != nil || amount != "1500.00" {
		t.Fatal("Publishing a price changed a contracted total")
	}
	must(`UPDATE legui_private.subscriptions SET status='canceled' WHERE id=?`, sub)
	if obj(request("GET", "/api/membership", adminToken, "", nil, 200))["operational"] != true {
		t.Fatal("Canceled renewal removed already paid access")
	}
	fmt.Printf("Membership integration: %d HTTP checks; all fixtures roll back\n", checks)
}
