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
	"leguiburger/internal/auth"
	"leguiburger/internal/brands"
	"leguiburger/internal/businesscare"
	"leguiburger/internal/db"
	"leguiburger/internal/mercadopago"
	"leguiburger/internal/products"
	"leguiburger/internal/recipes"
	"leguiburger/internal/supplies"
	"leguiburger/internal/tenantproducts"
	"leguiburger/internal/tenants"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Real handlers/repositories, two unrelated brands and a second local of brand A.
// Everything (including sessions, financial ledgers and receipts) is rolled back.
func TestTenantIsolationIntegration(t *testing.T) {
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
		if e := tx.Exec(q, args...).Error; e != nil {
			t.Fatal(e)
		}
	}
	b1, b2, t1, t2, t3 := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	p1, p2, s1, s2 := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	owner, a1, a2, cashier, kitchen := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	must(`INSERT INTO public.brands(id,name,base_location_limit,base_collaborator_limit) VALUES(?,'__isolation_brand_a',3,5),(?,'__isolation_brand_b',3,5)`, b1, b2)
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
	RegisterRoutes(mux, tx)
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
	for _, actor := range []string{"admin_a", "cashier", "kitchen"} {
		request("GET", "/api/products", actor, "", nil, nil, 200)
		request("GET", "/api/products?brand_id="+b2, actor, "", nil, nil, 403)
		request("GET", "/api/products", actor, "", map[string]string{"X-Brand-ID": b2}, nil, 403)
		request("GET", "/api/products/"+p2, actor, "", nil, nil, 404)
		for _, path := range []string{"/api/tenant-products", "/api/supplies", "/api/recipes", "/api/inventory", "/api/inventory/movements", "/api/kitchen"} {
			request("GET", path, actor, t1, nil, nil, 200)
			request("GET", path, actor, t2, nil, nil, 403)
			request("GET", path, actor, t3, nil, nil, map[bool]int{true: 200, false: 403}[actor == "admin_a"])
		}
		for _, path := range []string{"/api/tenant-products/" + p2, "/api/supplies/" + s2, "/api/recipes/" + p2 + "/" + s2} {
			request("GET", path, actor, t1, nil, nil, 404)
		}
		for _, path := range []string{"/api/analytics/platform", "/api/platform/businesses", "/api/platform/billing/accounts", "/api/platform/restaurants"} {
			request("GET", path, actor, t1, nil, nil, 403)
		}
	}
	for _, path := range []string{"/api/orders", "/api/cash/shifts", "/api/pos/catalog"} {
		request("GET", path, "cashier", t2, nil, nil, 403)
		request("GET", path, "kitchen", t1, nil, nil, 403)
		request("GET", path, "admin_a", t2, nil, nil, 403)
	}
	request("PUT", "/api/products/"+p2, "admin_a", "", nil, map[string]any{"name": "tampered"}, 404)
	request("DELETE", "/api/products/"+p2, "admin_a", "", nil, nil, 404)
	request("PUT", "/api/supplies/"+s2, "admin_a", t1, nil, map[string]any{"name": "tampered"}, 404)
	request("DELETE", "/api/recipes/"+p2+"/"+s2, "admin_a", t1, nil, nil, 404)
	request("POST", "/api/tenant-products", "admin_a", t1, nil, map[string]any{"product_id": p2}, 400)
	request("POST", "/api/recipes", "admin_a", t1, nil, map[string]any{"product_id": p1, "supply_id": s2, "quantity_used": 1}, 400)
	for _, actor := range []string{"cashier", "kitchen"} {
		request("POST", "/api/products", actor, "", nil, map[string]any{"name": "tampered", "brand_id": b2}, 403)
		request("GET", "/api/menu/settings", actor, t1, nil, nil, 403)
	}
	request("PATCH", "/api/menu/settings", "admin_a", t2, nil, map[string]any{"published": true}, 403)
	request("GET", "/api/menu/settings", "admin_a", t2, nil, nil, 403)
	for _, path := range []string{"/api/care/backups", "/api/care/closure"} {
		request("GET", path+"?brand_id="+b2, "cashier", t1, nil, nil, 403)
	}
	request("GET", "/api/products?brand_id="+b2, "owner", "", nil, nil, 200)
	request("GET", "/api/inventory", "owner", t2, nil, nil, 200)
	request("GET", "/api/products?brand_id="+b2, "admin_b", "", nil, nil, 200)
	var name string
	tx.Raw(`SELECT name FROM public.products WHERE id=?`, p2).Scan(&name)
	if name != "__isolation_product_b" {
		t.Fatal("cross-brand mutation changed the target")
	}

	object := func(w *httptest.ResponseRecorder) map[string]any {
		t.Helper()
		var out map[string]any
		if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil {
			t.Fatal(e)
		}
		return out
	}
	request("GET", "/api/public/menu/"+t1, "", "", nil, nil, 404)
	request("PATCH", "/api/menu/settings", "admin_a", t1, nil, map[string]any{"published": true}, 200)
	public := request("GET", "/api/public/menu/"+t1, "", "", nil, nil, 200)
	for _, private := range []string{"unit_cost", "current_stock", "minimum_stock", "recipe", "brand_id", "tenant_id", "password", "actor_"} {
		if strings.Contains(public.Body.String(), private) {
			t.Fatalf("public menu contains private field %s", private)
		}
	}
	request("GET", "/api/public/menu/"+t2, "admin_a", t1, nil, nil, 404)
	request("PUT", "/api/products/"+p1, "admin_a", "", nil, map[string]any{"category": "Bebidas"}, 200)
	filtered := object(request("GET", "/api/public/menu/"+t1+"?category=Bebidas", "", "", nil, nil, 200))
	if filtered["total"] != float64(1) {
		t.Fatal("published category did not use existing catalogue")
	}
	request("PATCH", "/api/menu/settings", "admin_a", t1, nil, map[string]any{"published": false}, 200)
	request("GET", "/api/public/menu/"+t1, "", "", nil, nil, 404)
	// Search finds a row beyond the first page and treats SQL syntax as literal input.
	for i := 0; i < 30; i++ {
		id := uuid.NewString()
		must(`INSERT INTO public.products(id,brand_id,name,base_price,is_active) VALUES(?,?,?,100,true)`, id, b1, fmt.Sprintf("__isolation_search_%02d", i))
		must(`INSERT INTO public.tenant_products(tenant_id,product_id,current_stock,track_stock,is_available,is_active) VALUES(?,?,10,true,true,true)`, t1, id)
	}
	searchResult := object(request("GET", "/api/inventory?limit=25&search=search_29", "admin_a", t1, nil, nil, 200))
	if searchResult["total"] != float64(1) {
		t.Fatal("search did not find a row outside the first page")
	}
	must(`UPDATE public.tenant_products SET minimum_stock=20 WHERE tenant_id=? AND product_id=?`, t1, p1)
	lowResult := object(request("GET", "/api/inventory?low=true", "admin_a", t1, nil, nil, 200))
	if lowResult["total"] != float64(1) {
		t.Fatal("minimum stock filter is not applied before pagination")
	}
	empty := object(request("GET", "/api/inventory?search=%27%20OR%201%3D1--", "cashier", t1, nil, nil, 200))
	if empty["total"] != float64(0) {
		t.Fatal("search escaped its scope")
	}
	request("GET", "/api/inventory?kind=unknown", "admin_a", t1, nil, nil, 400)
	request("GET", "/api/pos/catalog?search=search_29", "cashier", t1, nil, nil, 200)
	// One drawer per local/currency, retry-safe opening and immutable reconciled closure.
	openBody := map[string]any{"opening_balance": "100.00", "currency": "ARS"}
	key := map[string]string{"Idempotency-Key": "cash-open-fixture"}
	opened := object(request("POST", "/api/cash/shifts", "cashier", t1, key, openBody, 201))
	shift := opened["id"].(string)
	replay := request("POST", "/api/cash/shifts", "cashier", t1, key, openBody, 201)
	if replay.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatal("opening retry was not replayed")
	}
	request("POST", "/api/cash/shifts", "cashier", t1, nil, openBody, 409)
	otherShift := object(request("POST", "/api/cash/shifts", "admin_b", t2, nil, openBody, 201))["id"].(string)
	for _, suffix := range []string{"", "/entries"} {
		request("GET", "/api/cash/shifts/"+otherShift+suffix, "admin_a", t1, nil, nil, 404)
	}
	request("POST", "/api/cash/shifts/"+otherShift+"/close", "admin_a", t1, nil, map[string]any{"counted_balance": "100.00", "note": "forged"}, 404)
	created := object(request("POST", "/api/orders", "cashier", t1, nil, map[string]any{"channel": "counter", "environment": "production", "items": []map[string]any{{"product_id": p1, "quantity": 1}}}, 201))
	order := created["id"].(string)
	paid := object(request("POST", "/api/orders/"+order+"/payments", "cashier", t1, nil, map[string]any{"kind": "charge", "method": "cash", "amount": "60.00"}, 201))
	payment := paid["id"].(string)
	request("POST", "/api/orders/"+order+"/payments", "cashier", t1, nil, map[string]any{"kind": "refund", "method": "cash", "amount": "10.00", "original_payment_id": payment}, 201)
	request("POST", "/api/cash/shifts/"+shift+"/entries", "cashier", t1, nil, map[string]any{"kind": "cash_in", "amount": "20.00", "note": "Cambio para el turno"}, 201)
	request("POST", "/api/cash/shifts/"+shift+"/entries", "cashier", t1, nil, map[string]any{"kind": "cash_out", "amount": "10.00", "note": "Retiro a caja fuerte"}, 201)
	request("POST", "/api/cash/shifts/"+shift+"/entries", "cashier", t1, nil, map[string]any{"kind": "cash_out", "amount": "9999.00", "note": "Retiro imposible"}, 409)
	request("POST", "/api/cash/shifts/"+shift+"/close", "cashier", t1, nil, map[string]any{"counted_balance": "155.00", "expected_balance": "155.00"}, 400)
	request("POST", "/api/cash/shifts/"+shift+"/close", "cashier", t1, nil, map[string]any{"counted_balance": "155.00"}, 400)
	closed := object(request("POST", "/api/cash/shifts/"+shift+"/close", "cashier", t1, nil, map[string]any{"counted_balance": "155.00", "note": "Faltante detectado al contar"}, 200))
	if closed["expected_balance"] != "160.00" || closed["difference"] != "-5.00" {
		t.Fatalf("cash ledger mismatch: expected=%v difference=%v", closed["expected_balance"], closed["difference"])
	}
	request("POST", "/api/cash/shifts/"+shift+"/entries", "cashier", t1, nil, map[string]any{"kind": "cash_in", "amount": "1.00", "note": "Fuera de turno"}, 409)
	request("POST", "/api/orders/"+order+"/payments", "cashier", t1, nil, map[string]any{"kind": "charge", "method": "cash", "amount": "1.00"}, 409)
	request("GET", "/api/cash/shifts/"+otherShift, "owner", t2, nil, nil, 200)
	request("PATCH", "/api/menu/settings", "admin_a", t1, nil, map[string]any{"published": true}, 200)
	must(`UPDATE public.brands SET suspended_at=clock_timestamp() WHERE id=?`, b1)
	request("GET", "/api/public/menu/"+t1, "", "", nil, nil, 404)
	t.Logf("%d real HTTP isolation, publication, search and cash-ledger checks passed; all fixtures roll back", checks)

}
