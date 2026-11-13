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
func TestProductOptionsCapacityIntegration(t *testing.T) {
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
	_ = a
	x1, x2, x3, foreign := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	must(`INSERT INTO public.extras(id,tenant_id,name,current_price,current_stock,track_stock,is_active,current_unit_cost) VALUES(?,?,'Cheddar',5,10,true,true,2),(?,?,'Simple',0,0,false,true,0),(?,?,'Bebida',100,3,true,true,30),(?,?,'Ajeno',9,0,false,true,1)`, x1, t1, x2, t1, x3, t1, foreign, t2)
	basket := func(choices []any, qty int) map[string]any {
		return map[string]any{"fulfillment": "pickup", "items": []any{map[string]any{"product_id": p1, "quantity": qty, "extras": choices}}}
	}
	selection := []any{map[string]any{"extra_id": x1, "quantity": 2}}
	if parse(request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, basket(selection, 1), 200))["total"] != "110.00" {
		t.Fatal("legacy extras price changed")
	}
	groups := []any{
		map[string]any{"name": "Tamaño", "min": 1, "max": 1, "options": []any{map[string]any{"extra_id": x2, "price_override": "0"}}},
		map[string]any{"name": "Bebida", "min": 1, "max": 1, "options": []any{map[string]any{"extra_id": x3, "price_override": "0"}}},
		map[string]any{"name": "Adicionales", "min": 0, "max": 2, "options": []any{map[string]any{"extra_id": x1, "price_override": "10.00"}}},
	}
	optionConfig := map[string]any{"use_product_options": true, "groups": groups}
	path := "/api/menu/products/" + p1 + "/options"
	request("PUT", path, "cashier", t1, nil, optionConfig, 403)
	request("PUT", path, "admin_b", t1, nil, optionConfig, 403)
	request("PUT", path, "admin_b", t2, nil, optionConfig, 404)
	request("PUT", path, "admin_a", t1, nil, optionConfig, 200)
	one := parse(request("GET", "/api/public/menu/"+t1+"?product_id="+p1+"&limit=1", "", "", nil, nil, 200))["items"].([]any)
	if len(one) != 1 || len(one[0].(map[string]any)["option_groups"].([]any)) != 3 {
		t.Fatal("single-product options projection invalid")
	}
	if len(parse(request("GET", "/api/public/menu/"+t1+"?product_id="+p2, "", "", nil, nil, 200))["items"].([]any)) != 0 {
		t.Fatal("product filter exposed another restaurant")
	}
	request("GET", "/api/public/menu/"+t1+"?product_id=invalid", "", "", nil, nil, 400)

	bad := map[string]any{"use_product_options": true, "groups": []any{map[string]any{"name": "Ajeno", "min": 0, "max": 1, "options": []any{map[string]any{"extra_id": foreign, "price_override": nil}}}}}
	request("PUT", path, "admin_a", t1, nil, bad, 404)
	if len(parse(request("GET", path, "admin_a", t1, nil, nil, 200))["groups"].([]any)) != 3 {
		t.Fatal("failed configuration discarded valid groups")
	}
	request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, basket(nil, 1), 409)
	request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, basket(selection, 1), 409)
	valid := []any{map[string]any{"extra_id": x2, "quantity": 1}, map[string]any{"extra_id": x3, "quantity": 1}, map[string]any{"extra_id": x1, "quantity": 2}}
	quote := parse(request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, basket(valid, 2), 200))
	if quote["total"] != "240.00" {
		t.Fatalf("wrong option override total: %v", quote["total"])
	}
	tooMany := []any{map[string]any{"extra_id": x2, "quantity": 1}, map[string]any{"extra_id": x3, "quantity": 1}, map[string]any{"extra_id": x1, "quantity": 3}}
	request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, basket(tooMany, 1), 409)
	must(`UPDATE public.extras SET current_stock=0 WHERE id=?`, x3)
	menu := parse(request("GET", "/api/public/menu/"+t1, "", "", nil, nil, 200))
	for _, raw := range menu["items"].([]any) {
		if raw.(map[string]any)["id"] == p1 && raw.(map[string]any)["available"] != false {
			t.Fatal("required unavailable choice did not hide product")
		}
	}
	request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, basket(valid, 1), 409)
	must(`UPDATE public.extras SET current_stock=3 WHERE id=?`, x3)
	create := func(qty int) map[string]any {
		b := basket(valid, qty)
		q := parse(request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, b, 200))
		b["quote_digest"] = q["digest"]
		b["name"] = "Options Fixture"
		b["phone"] = "1155550123"
		b["payment_method"] = "at_local"
		h := map[string]string{"Cookie": ca["Cookie"], "Authorization": "Bearer " + base64.RawURLEncoding.EncodeToString([]byte(uuid.NewString()[:32]))}
		return parse(request("POST", "/api/public/menu/"+t1+"/orders", "", "", h, b, 201))
	}
	order := create(2)
	id := order["id"].(string)
	// Editing future choices must not rewrite prices, labels, stock or costs already agreed.
	groups[1].(map[string]any)["name"] = "Tu bebida"
	groups[2].(map[string]any)["options"] = []any{map[string]any{"extra_id": x1, "price_override": "20.00"}}
	request("PUT", path, "admin_a", t1, nil, optionConfig, 200)
	detail := parse(request("GET", "/api/orders/"+id, "admin_a", t1, nil, nil, 200))
	request("PATCH", "/api/orders/"+id+"/status", "admin_a", t1, nil, map[string]any{"status": "accepted", "expected_version": detail["version"]}, 200)
	var stock int
	tx.Raw(`SELECT current_stock FROM public.extras WHERE id=?`, x1).Scan(&stock)
	if stock != 6 {
		t.Fatal("selected extra quantity did not consume stock")
	}
	tx.Raw(`SELECT current_stock FROM public.extras WHERE id=?`, x3).Scan(&stock)
	if stock != 1 {
		t.Fatal("included extra did not consume stock")
	}
	var revenue string
	tx.Raw(`SELECT sum(total_net_revenue)::text FROM public.sales WHERE order_id=?`, id).Scan(&revenue)
	if revenue != "240.00" {
		t.Fatalf("sales snapshot changed: %s", revenue)
	}
	var label string
	tx.Raw(`SELECT option_group_name FROM public.order_detail_extras WHERE extra_id=? AND order_detail_id IN(SELECT id FROM public.order_details WHERE order_id=?)`, x3, id).Scan(&label)
	if label != "Bebida" {
		t.Fatal("historical choice group label changed")
	}
	receipt := parse(request("GET", "/api/public/buyer-order", "", "", map[string]string{"Cookie": ca["Cookie"], "X-Shopper-Order": id}, nil, 200))
	if receipt["total_due"] != "240.00" {
		t.Fatal("buyer quote and persisted choice charge disagree")
	}
	detail = parse(request("GET", "/api/orders/"+id, "admin_a", t1, nil, nil, 200))
	request("PATCH", "/api/orders/"+id+"/status", "admin_a", t1, nil, map[string]any{"status": "canceled", "reason": "Fixture", "expected_version": detail["version"]}, 200)
	request("POST", "/api/orders", "cashier", t1, nil, map[string]any{"channel": "counter", "environment": "production", "items": []any{map[string]any{"product_id": p1, "quantity": 1}}}, 409)
	settings := map[string]any{"timezone": "America/Argentina/Buenos_Aires", "schedule_enabled": false, "paused": false, "pause_minutes": 0, "pending_minutes": 15, "prep_min": 20, "prep_max": 35, "periods": []any{}, "max_active_orders": 2, "queue_step_minutes": 3, "max_queue_minutes": 6}
	request("PATCH", "/api/ordering/settings", "admin_a", t1, nil, settings, 200)
	first := create(1)
	second := create(1)
	if first["prep_min"] != float64(20) || second["prep_min"] != float64(23) {
		t.Fatalf("queue delay not snapshotted: %v %v", first["prep_min"], second["prep_min"])
	}
	request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, basket(valid, 1), 409)
	request("POST", "/api/orders", "cashier", t1, nil, map[string]any{"channel": "counter", "environment": "production", "items": []any{map[string]any{"product_id": p1, "quantity": 1, "extras": valid}}}, 409)
	menu = parse(request("GET", "/api/public/menu/"+t1, "", "", nil, nil, 200))
	if menu["ordering_ready"] != false || menu["capacity_full"] != true {
		t.Fatal("full kitchen still publicly receiving")
	}
	load := parse(request("GET", "/api/kitchen/load", "kitchen", t1, nil, nil, 200))
	if load["active_count"] != float64(2) || load["full"] != true {
		t.Fatal("wrong private queue projection")
	}
	request("GET", "/api/kitchen/load", "cashier", t2, nil, nil, 403)
	var locked bool
	if e := database.Raw(`SELECT pg_try_advisory_xact_lock(hashtextextended(?,0))`, "kitchen-capacity:"+t1).Scan(&locked).Error; e != nil {
		t.Fatal(e)
	}
	if locked {
		t.Fatal("capacity reservation did not retain transaction serialization lock")
	}
	e = tx.Transaction(func(inner *gorm.DB) error {
		return inner.Exec(`INSERT INTO public.orders(tenant_id,channel,payment_environment,idempotency_key) VALUES(?,'counter','production',?)`, t1, uuid.NewString()).Error
	})
	if e == nil || !strings.Contains(e.Error(), "23514") {
		t.Fatal("database capacity backstop absent")
	}
	request("POST", "/api/public/buyer-order/cancel", "", "", map[string]string{"Cookie": ca["Cookie"], "X-Shopper-Order": first["id"].(string)}, map[string]any{}, 200)
	menu = parse(request("GET", "/api/public/menu/"+t1, "", "", nil, nil, 200))
	if menu["ordering_ready"] != true {
		t.Fatal("cancel did not free capacity")
	}
	// Old clients saving hours must preserve new capacity fields they do not send.
	delete(settings, "max_active_orders")
	delete(settings, "queue_step_minutes")
	delete(settings, "max_queue_minutes")
	request("PATCH", "/api/ordering/settings", "admin_a", t1, nil, settings, 200)
	current := parse(request("GET", "/api/ordering/settings", "admin_a", t1, nil, nil, 200))
	if current["max_active_orders"] != float64(2) {
		t.Fatal("legacy settings update erased capacity")
	}
	bid := second["id"].(string)
	for _, next := range []string{"accepted", "preparing", "ready"} {
		d := parse(request("GET", "/api/orders/"+bid, "admin_a", t1, nil, nil, 200))
		request("PATCH", "/api/orders/"+bid+"/status", "admin_a", t1, nil, map[string]any{"status": next, "expected_version": d["version"]}, 200)
	}
	if parse(request("GET", "/api/kitchen/load", "kitchen", t1, nil, nil, 200))["active_count"] != float64(0) {
		t.Fatal("ready orders still occupy kitchen slot")
	}
	request("PUT", "/api/menu/products/"+p2+"/options", "admin_b", t2, nil, map[string]any{"use_product_options": true, "groups": []any{}}, 200)
	request("POST", "/api/public/menu/"+t2+"/quote", "", "", nil, map[string]any{"fulfillment": "pickup", "items": []any{map[string]any{"product_id": p2, "quantity": 1, "extras": []any{map[string]any{"extra_id": foreign, "quantity": 1}}}}}, 409)
	t.Logf("Options/capacity: %d HTTP checks; existing extras, selection constraints/prices, snapshots, sale/stock, role/tenant scopes, kitchen limits/serialization/reopening and legacy settings verified; all fixtures rolled back", checks)
}
