package operations

import (
	"bufio"
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
	"leguiburger/internal/buyeraccess"
	"leguiburger/internal/db"
	"leguiburger/internal/mercadopago"
	"leguiburger/internal/orderstream"
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
	"time"
)

// Guest purchase through real handlers against two unrelated restaurant fixtures.
// Everything (including sessions, financial ledgers and receipts) is rolled back.
func TestBuyerCheckoutIntegration(t *testing.T) {
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
	x1, x2, shipping := uuid.NewString(), uuid.NewString(), uuid.NewString()
	must(`INSERT INTO public.extras(id,tenant_id,name,current_price,current_stock,track_stock,is_active) VALUES(?,?,'Buyer cheddar',10.25,10,true,true),(?,?,'Foreign cheddar',10.25,10,true,true)`, x1, t1, x2, t2)
	must(`INSERT INTO public.shipping_methods(id,tenant_id,name,cost,is_active) VALUES(?,?,'Zona centro',20.50,true)`, shipping, t1)
	guest := base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("a", 32)))
	other := base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("b", 32)))
	if !buyeraccess.ValidToken(guest) || !buyeraccess.ValidToken(other) {
		t.Fatal("invalid test capability")
	}
	key := uuid.NewString()
	headers := map[string]string{"Authorization": "Bearer " + guest, "Idempotency-Key": key}
	basket := map[string]any{"fulfillment": "pickup", "items": []map[string]any{{"product_id": p1, "quantity": 2, "extras": []map[string]any{{"extra_id": x1, "quantity": 1}}, "note": "Sin cebolla"}}}
	path := "/api/public/menu/" + t1
	parse := func(w *httptest.ResponseRecorder) map[string]any {
		t.Helper()
		var v map[string]any
		if e := json.Unmarshal(w.Body.Bytes(), &v); e != nil {
			t.Fatal(e)
		}
		return v
	}
	quote := parse(request("POST", path+"/quote", "", "", nil, basket, 200))
	if quote["total"] != "220.50" {
		t.Fatalf("quote total %v", quote["total"])
	}
	foreign := map[string]any{"fulfillment": "pickup", "items": []map[string]any{{"product_id": p2, "quantity": 1}}}
	request("POST", path+"/quote", "", "", nil, foreign, 409)
	foreign["items"] = []map[string]any{{"product_id": p1, "quantity": 1, "extras": []map[string]any{{"extra_id": x2, "quantity": 1}}}}
	request("POST", path+"/quote", "", "", nil, foreign, 409)
	foreign["items"] = []map[string]any{{"product_id": p1, "quantity": 11}}
	request("POST", path+"/quote", "", "", nil, foreign, 409)
	foreign["items"] = []map[string]any{{"product_id": p1, "quantity": 1, "price": "0.01"}}
	request("POST", path+"/quote", "", "", nil, foreign, 400)
	delivery := map[string]any{"fulfillment": "delivery", "shipping_method_id": shipping, "items": basket["items"]}
	dq := parse(request("POST", path+"/quote", "", "", nil, delivery, 200))
	if dq["total"] != "241.00" {
		t.Fatalf("delivery total %v", dq["total"])
	}
	request("POST", "/api/public/menu/"+t2+"/quote", "", "", nil, delivery, 409)
	payload := map[string]any{"fulfillment": "pickup", "items": basket["items"], "quote_digest": quote["digest"], "name": "Buyer Fixture", "phone": "11 5555 0123", "email": "buyer@example.invalid", "payment_method": "at_local"}
	request("POST", path+"/orders", "", "", nil, payload, 400)
	payload["payment_method"] = "mercadopago"
	request("POST", path+"/orders", "", "", headers, payload, 409)
	payload["payment_method"] = "at_local"
	must(`UPDATE public.products SET base_price=101 WHERE id=?`, p1)
	request("POST", path+"/orders", "", "", headers, payload, 409)
	must(`UPDATE public.products SET base_price=100 WHERE id=?`, p1)
	order := parse(request("POST", path+"/orders", "", "", headers, payload, 201))
	oid := order["id"].(string)

	server := httptest.NewServer(mux)
	defer server.Close()
	client := &http.Client{Timeout: 10 * time.Second}
	openStream := func(path, cap string, want int) *http.Response {
		t.Helper()
		r, _ := http.NewRequest("GET", server.URL+path, nil)
		r.Header.Set("Authorization", "Bearer "+cap)
		response, e := client.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		if response.StatusCode != want {
			t.Fatalf("SSE status want %d got %d", want, response.StatusCode)
		}
		return response
	}
	bad := openStream("/api/public/buyer-order/events", other, 404)
	bad.Body.Close()
	denied := openStream("/api/public/buyer-order/events", buyeraccess.TrackingToken(guest), 404)
	denied.Body.Close()
	stream := openStream("/api/public/buyer-order/events", guest, 200)
	reader := bufio.NewReader(stream.Body)
	readEvent := func(want string) {
		t.Helper()
		for {
			line, e := reader.ReadString('\n')
			if e != nil {
				t.Fatal(e)
			}
			if strings.HasPrefix(line, "event:") {
				if strings.TrimSpace(line) != "event: "+want {
					t.Fatalf("wrong SSE event %q", line)
				}
				return
			}
		}
	}
	readEvent("ready")
	// NOTIFY commits without modifying any production rows. The order fixture stays
	// inside its rollback transaction; notification carries only the opaque ID.
	rolled := database.Begin()
	rolled.Exec(`SELECT pg_notify('legui_order_changed',?)`, oid)
	rolled.Rollback()
	if e := database.Exec(`SELECT pg_notify('legui_order_changed',?)`, oid).Error; e != nil {
		t.Fatal(e)
	}
	readEvent("changed")
	stream.Body.Close()
	trackingStream := openStream("/api/public/order/events", buyeraccess.TrackingToken(guest), 200)
	trackingStream.Body.Close()
	if order["status"] != "pending" || order["total_due"] != "220.50" || order["total_paid"] != "0.00" {
		t.Fatalf("wrong buyer receipt: %v", order)
	}
	for _, field := range []string{"buyer_name", "buyer_phone", "buyer_email", "shipping_address", "access_digest"} {
		if _, ok := order[field]; ok {
			t.Fatal("unsafe receipt field " + field)
		}
	}
	replay := request("POST", path+"/orders", "", "", headers, payload, 201)
	if replay.Header().Get("Idempotent-Replayed") != "true" || parse(replay)["id"] != oid {
		t.Fatal("non-idempotent guest order")
	}
	changed := map[string]string{"Authorization": "Bearer " + other, "Idempotency-Key": key}
	request("POST", path+"/orders", "", "", changed, payload, 404)
	payload["name"] = "Changed buyer"
	request("POST", path+"/orders", "", "", headers, payload, 409)
	payload["name"] = "Buyer Fixture"
	request("GET", "/api/public/buyer-order", "", "", nil, nil, 404)
	request("GET", "/api/public/buyer-order", "", "", changed, nil, 404)
	request("GET", "/api/public/buyer-order", "", "", headers, nil, 200)
	request("GET", "/api/orders/"+oid, "", t1, headers, nil, 401)
	tracking := map[string]string{"Authorization": "Bearer " + buyeraccess.TrackingToken(guest)}
	request("GET", "/api/public/order", "", "", tracking, nil, 200)
	request("GET", "/api/public/buyer-order", "", "", tracking, nil, 404)
	request("POST", "/api/public/buyer-order/mercadopago", "", "", tracking, map[string]any{}, 409) // provider disabled
	private := parse(request("GET", "/api/orders/"+oid, "admin_a", t1, nil, nil, 200))
	if private["buyer_name"] != "Buyer Fixture" || private["buyer_phone"] != "11 5555 0123" {
		t.Fatal("restaurant did not receive buyer contact")
	}
	request("GET", "/api/orders/"+oid, "admin_b", t2, nil, nil, 404)
	var stock, sales int64
	tx.Raw(`SELECT current_stock FROM public.tenant_products WHERE tenant_id=? AND product_id=?`, t1, p1).Scan(&stock)
	tx.Raw(`SELECT count(*) FROM public.sales WHERE order_id=?`, oid).Scan(&sales)
	if stock != 10 || sales != 0 {
		t.Fatal("unaccepted guest order consumed stock or posted revenue")
	}
	request("PATCH", "/api/orders/"+oid+"/status", "admin_a", t1, nil, map[string]any{"status": "accepted", "expected_version": private["version"]}, 200)
	tx.Raw(`SELECT current_stock FROM public.tenant_products WHERE tenant_id=? AND product_id=?`, t1, p1).Scan(&stock)
	if stock != 8 {
		t.Fatal("accepted guest order did not consume stock")
	}
	kitchenData := request("GET", "/api/kitchen", "kitchen", t1, nil, nil, 200).Body.String()
	if !strings.Contains(kitchenData, "Sin cebolla") || strings.Contains(kitchenData, "buyer@example.invalid") || strings.Contains(kitchenData, "11 5555 0123") {
		t.Fatal("kitchen buyer projection invalid")
	}
	must(`UPDATE public.tenants SET ordering_enabled=false,menu_published=false WHERE id=?`, t1)
	request("POST", path+"/quote", "", "", nil, basket, 409)
	request("GET", "/api/public/buyer-order", "", "", headers, nil, 200)
	must(`UPDATE public.brands SET suspended_at=clock_timestamp() WHERE id=?`, b1)
	request("GET", "/api/public/buyer-order", "", "", headers, nil, 404)
	t.Logf("%d buyer checks; all fixtures rolled back", checks)
}
