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
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// Guest purchase through real handlers against two unrelated restaurant fixtures.
// Everything (including sessions, financial ledgers and receipts) is rolled back.
func TestCommerceExperienceIntegration(t *testing.T) {
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
		// Handler savepoints share this outer rollback transaction. Direct fixture
		// writes must not inherit the last HTTP operation's idempotency metadata.
		if e := tx.Exec(`SELECT set_config('leguiburger.operation_key','',true),set_config('leguiburger.request_digest','',true),set_config('leguiburger.source_order_id','',true)`).Error; e != nil {
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
	_, cb := makeAccount("B")
	must(`UPDATE public.tenant_products SET track_stock=false WHERE tenant_id=? AND product_id=?`, t1, p1)
	p3 := uuid.NewString()
	must(`INSERT INTO public.products(id,brand_id,name,base_price,is_active) VALUES(?,?,'__second_burger',100,true)`, p3, b1)
	must(`INSERT INTO public.tenant_products(tenant_id,product_id,current_stock,track_stock,is_available,is_active) VALUES(?,?,10,false,true,true)`, t1, p3)
	must(`INSERT INTO public.recipes(product_id,supply_id,quantity_used) VALUES(?,?,1)`, p3, s1)
	must(`UPDATE public.supplies SET current_stock=1 WHERE id=?`, s1)
	basket := map[string]any{"fulfillment": "pickup", "items": []any{map[string]any{"product_id": p1, "quantity": 1}, map[string]any{"product_id": p3, "quantity": 1}}}
	request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, basket, 409)
	basket["items"] = []any{map[string]any{"product_id": p1, "quantity": 2}}
	request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, basket, 409)
	must(`UPDATE public.supplies SET current_stock=0 WHERE id=?`, s1)
	menu := parse(request("GET", "/api/public/menu/"+t1, "", "", nil, nil, 200))
	for _, item := range menu["items"].([]any) {
		if item.(map[string]any)["available"] != false {
			t.Fatal("depleted local recipe remained available")
		}
	}
	untracked := uuid.NewString()
	must(`INSERT INTO public.products(id,brand_id,name,base_price,is_active) VALUES(?,?,'__manual_availability',90,true)`, untracked, b1)
	must(`INSERT INTO public.tenant_products(tenant_id,product_id,current_stock,track_stock,is_available,is_active) VALUES(?,?,0,false,true,true)`, t1, untracked)
	manualMenu := parse(request("GET", "/api/public/menu/"+t1, "", "", nil, nil, 200))
	foundManual := false
	for _, raw := range manualMenu["items"].([]any) {
		item := raw.(map[string]any)
		if item["id"] == untracked {
			foundManual = item["available"] == true
		}
	}
	if !foundManual {
		t.Fatal("untracked product without local recipe lost manual availability")
	}
	must(`UPDATE public.supplies SET current_stock=100 WHERE id=?`, s1)
	request("GET", "/api/menu/availability", "kitchen", t1, nil, nil, 403)
	request("PATCH", "/api/menu/products/"+p1+"/availability", "admin_b", t1, nil, map[string]any{"enabled": false}, 403)
	request("PATCH", "/api/menu/products/"+p2+"/availability", "admin_a", t1, nil, map[string]any{"enabled": false}, 404)
	request("PATCH", "/api/menu/products/"+p1+"/availability", "cashier", t1, nil, map[string]any{"enabled": false}, 200)
	basket["items"] = []any{map[string]any{"product_id": p1, "quantity": 1}}
	request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, basket, 409)
	request("PATCH", "/api/menu/products/"+p1+"/availability", "cashier", t1, nil, map[string]any{"enabled": true}, 200)
	favorite := "/api/shopper/favorites/" + t1 + "/" + p1
	request("PUT", favorite, "", "", nil, map[string]any{}, 401)
	request("PUT", favorite, "", "", ca, map[string]any{}, 200)
	request("PUT", favorite, "", "", ca, map[string]any{}, 200)
	if len(parse(request("GET", "/api/shopper/favorites", "", "", ca, nil, 200))["items"].([]any)) != 1 {
		t.Fatal("favorite not idempotent")
	}
	request("DELETE", favorite, "", "", cb, map[string]any{}, 200)
	if len(parse(request("GET", "/api/shopper/favorites", "", "", cb, nil, 200))["items"].([]any)) != 0 {
		t.Fatal("another shopper's favorites leaked")
	}
	if len(parse(request("GET", "/api/shopper/favorites", "", "", ca, nil, 200))["items"].([]any)) != 1 {
		t.Fatal("another shopper removed favorite")
	}
	request("PUT", "/api/shopper/favorites/"+t1+"/"+p2, "", "", ca, map[string]any{}, 404)
	request("PUT", "/api/shopper/favorites/"+t2+"/"+p2, "", "", ca, map[string]any{}, 200)
	must(`UPDATE public.tenants SET menu_published=false WHERE id=?`, t2)
	if len(parse(request("GET", "/api/shopper/favorites", "", "", ca, nil, 200))["items"].([]any)) != 1 {
		t.Fatal("unpublished menu leaked through favorites")
	}
	request("DELETE", "/api/shopper/favorites/"+t2+"/"+p2, "", "", ca, map[string]any{}, 200)
	request("GET", "/api/delivery/zones", "cashier", t1, nil, nil, 403)
	request("POST", "/api/delivery/zones", "admin_b", t1, nil, map[string]any{}, 403)
	request("PATCH", "/api/delivery/settings", "admin_a", t1, nil, map[string]any{"zones_enabled": true}, 409)
	zoneInput := map[string]any{"name": "Centro", "description": "Zona central", "cost": "20.00", "minimum": "150.00", "postal_codes": []string{"1896"}, "active": true}
	zone := parse(request("POST", "/api/delivery/zones", "admin_a", t1, nil, zoneInput, 201))["id"].(string)
	zoneInput["name"] = "Otra zona"
	request("POST", "/api/delivery/zones", "admin_a", t1, nil, zoneInput, 409)
	request("PATCH", "/api/delivery/zones/"+zone, "admin_b", t2, nil, zoneInput, 404)
	zoneInput["name"] = "Centro"
	request("PATCH", "/api/delivery/settings", "admin_a", t1, nil, map[string]any{"zones_enabled": true}, 200)
	basket["fulfillment"] = "delivery"
	basket["delivery_zone_id"] = zone
	basket["postal_code"] = "9999"
	request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, basket, 409)
	basket["postal_code"] = "1896"
	request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, basket, 409)
	basket["items"] = []any{map[string]any{"product_id": p1, "quantity": 1}, map[string]any{"product_id": p3, "quantity": 1}}
	basket["postal_code"] = "B1896ABC"
	quote := parse(request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, basket, 200))
	if quote["total"] != "220.00" || quote["postal_code"] != "1896" {
		t.Fatalf("wrong zoned quote %v", quote)
	}
	input := map[string]any{}
	for k, v := range basket {
		input[k] = v
	}
	input["quote_digest"] = quote["digest"]
	input["name"] = "Shopper Fixture"
	input["phone"] = "1155550123"
	input["payment_method"] = "at_local"
	input["address"] = "Calle del local 123"
	secret := base64.RawURLEncoding.EncodeToString([]byte(uuid.NewString()[:32]))
	headers := map[string]string{"Cookie": ca["Cookie"], "Authorization": "Bearer " + secret}
	order := parse(request("POST", "/api/public/menu/"+t1+"/orders", "", "", headers, input, 201))["id"].(string)
	// Existing orders retain the delivered zone and shipping snapshots.
	zoneInput["name"] = "Centro nuevo"
	zoneInput["cost"] = "40.00"
	request("PATCH", "/api/delivery/zones/"+zone, "admin_a", t1, nil, zoneInput, 200)
	full := parse(request("GET", "/api/orders/"+order, "cashier", t1, nil, nil, 200))
	if full["delivery_zone_name"] != "Centro" || fmt.Sprint(full["shipping_cost"]) != "20" {
		t.Fatal("historical delivery snapshot changed")
	}
	request("POST", "/api/shopper/orders/"+order+"/repeat", "", "", ca, map[string]any{}, 409)
	request("POST", "/api/shopper/orders/"+order+"/repeat", "", "", cb, map[string]any{}, 404)
	request("GET", "/api/orders/"+order+"/ticket?kind=receipt", "kitchen", t1, nil, nil, 403)
	request("GET", "/api/orders/"+order+"/ticket?kind=kitchen", "cashier", t2, nil, nil, 403)
	request("GET", "/api/orders/"+order+"/ticket?kind=receipt", "admin_b", t2, nil, nil, 403)
	kt := parse(request("GET", "/api/orders/"+order+"/ticket?kind=kitchen", "kitchen", t1, nil, nil, 200))
	for _, field := range []string{"phone", "address", "name", "total_due", "total_paid", "currency"} {
		if _, ok := kt[field]; ok {
			t.Fatal("kitchen ticket exposed buyer or financial data")
		}
	}
	receipt := parse(request("GET", "/api/orders/"+order+"/ticket?kind=receipt", "cashier", t1, nil, nil, 200))
	if receipt["total_due"] != "220.00" || receipt["name"] != "Shopper Fixture" {
		t.Fatal("receipt projection wrong")
	}
	must(`UPDATE public.orders SET order_status_id=(SELECT id FROM public.order_statuses_lookup WHERE code='canceled') WHERE id=?`, order)
	must(`UPDATE public.products SET name='__new_product_name',base_price=120 WHERE id=?`, p1)
	must(`UPDATE public.tenant_products SET is_available=false WHERE tenant_id=? AND product_id=?`, t1, p3)
	var before, after int64
	tx.Raw(`SELECT count(*) FROM public.orders WHERE tenant_id=?`, t1).Scan(&before)
	repeat := parse(request("POST", "/api/shopper/orders/"+order+"/repeat", "", "", ca, map[string]any{}, 200))
	if len(repeat["items"].([]any)) != 1 || len(repeat["changes"].([]any)) < 2 {
		t.Fatalf("repeat did not report changes %v", repeat)
	}
	item := repeat["items"].([]any)[0].(map[string]any)
	if item["price"] != "120.00" || item["product_id"] != p1 {
		t.Fatalf("repeat used unexpected price or identity: %#v", item)
	}
	tx.Raw(`SELECT count(*) FROM public.orders WHERE tenant_id=?`, t1).Scan(&after)
	if before != after {
		t.Fatal("repeat created an order without checkout confirmation")
	}
	ticket := parse(request("GET", "/api/orders/"+order+"/ticket?kind=kitchen", "kitchen", t1, nil, nil, 200))
	if strings.Contains(fmt.Sprint(ticket["items"]), "__new_product_name") {
		t.Fatal("ticket changed historic product snapshots")
	}
	var linked bool
	tx.Raw(`SELECT EXISTS(SELECT 1 FROM legui_private.shopper_orders WHERE order_id=? AND shopper_id=?)`, order, a).Scan(&linked)
	if !linked {
		t.Fatal("purchase shopper relation lost")
	}
	request("DELETE", favorite, "", "", ca, map[string]any{}, 200)
	counter := parse(request("POST", "/api/orders", "cashier", t1, nil, map[string]any{"channel": "counter", "environment": "production", "items": []any{map[string]any{"product_id": p1, "quantity": 1}}}, 201))
	counterID := counter["id"].(string)
	version := counter["version"]
	must(`UPDATE public.supplies SET current_stock=0 WHERE id=?`, s1)
	request("PATCH", "/api/orders/"+counterID+"/status", "cashier", t1, nil, map[string]any{"status": "accepted", "expected_version": version}, 409)
	must(`UPDATE public.supplies SET current_stock=3 WHERE id=?`, s1)
	request("PATCH", "/api/orders/"+counterID+"/status", "cashier", t1, nil, map[string]any{"status": "accepted", "expected_version": version}, 200)
	var remaining string
	tx.Raw(`SELECT current_stock::text FROM public.supplies WHERE id=?`, s1).Scan(&remaining)
	amount, _ := priceRat(remaining)
	if amount == nil || amount.Cmp(big.NewRat(2, 1)) != 0 {
		t.Fatal("acceptance did not consume exactly one recipe unit")
	}
	t.Logf("Commerce experience: %d HTTP checks; recipe/shared stock, pause, scoped favorites, postal coverage/minimum/snapshots, print permissions and repeat-price checks passed; all fixtures rolled back", checks)
}
