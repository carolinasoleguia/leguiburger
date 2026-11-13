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
func TestStorefrontOperationsIntegration(t *testing.T) {
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
	makeOrder := func(tenant, product string, headers map[string]string) string {
		basket := map[string]any{"fulfillment": "pickup", "items": []any{map[string]any{"product_id": product, "quantity": 1}}}
		q := parse(request("POST", "/api/public/menu/"+tenant+"/quote", "", "", headers, basket, 200))
		basket["quote_digest"] = q["digest"]
		basket["name"] = "Shopper Fixture"
		basket["phone"] = "1155550123"
		basket["payment_method"] = "at_local"
		h := map[string]string{"Authorization": "Bearer " + base64.RawURLEncoding.EncodeToString([]byte(uuid.NewString()[:32]))}
		for k, v := range headers {
			h[k] = v
		}
		return parse(request("POST", "/api/public/menu/"+tenant+"/orders", "", "", h, basket, 201))["id"].(string)
	}
	settings := map[string]any{"timezone": "America/Argentina/Buenos_Aires", "schedule_enabled": false, "paused": false, "pause_minutes": 0, "pending_minutes": 10, "prep_min": 15, "prep_max": 25, "periods": []any{}}
	request("GET", "/api/ordering/settings", "admin_b", t1, nil, nil, 403)
	request("GET", "/api/ordering/settings", "cashier", t1, nil, nil, 403)
	request("PATCH", "/api/ordering/settings", "admin_a", t1, nil, settings, 200)
	request("GET", "/api/orders/inbox", "kitchen", t1, nil, nil, 403)
	request("GET", "/api/orders/events", "admin_b", t1, nil, nil, 403)
	request("GET", "/api/orders/events", "cashier", t2, nil, nil, 403)
	request("GET", "/api/orders/events", "", "", nil, nil, 401)
	settings["paused"] = true
	request("PATCH", "/api/ordering/settings", "admin_a", t1, nil, settings, 200)
	menu := parse(request("GET", "/api/public/menu/"+t1, "", "", nil, nil, 200))
	if menu["ordering_ready"] != false {
		t.Fatal("paused menu still receiving")
	}
	basket := map[string]any{"fulfillment": "pickup", "items": []any{map[string]any{"product_id": p1, "quantity": 1}}}
	request("POST", "/api/public/menu/"+t1+"/quote", "", "", ca, basket, 409)
	settings["paused"] = false
	settings["schedule_enabled"] = true
	settings["periods"] = []any{map[string]any{"day": 1, "opens": "19:00", "closes": "01:00"}}
	request("PATCH", "/api/ordering/settings", "admin_a", t1, nil, settings, 200)
	var open, closed bool
	if e := tx.Raw(`SELECT legui_private.tenant_receiving(?,'2026-11-10 00:30:00-03'::timestamptz),legui_private.tenant_receiving(?,'2026-11-10 02:00:00-03'::timestamptz)`, t1, t1).Row().Scan(&open, &closed); e != nil {
		t.Fatal(e)
	}
	if !open || closed {
		t.Fatal("overnight schedule incorrect")
	}
	settings["periods"] = []any{map[string]any{"day": 1, "opens": "19:00", "closes": "22:00"}, map[string]any{"day": 1, "opens": "21:00", "closes": "23:00"}}
	request("PATCH", "/api/ordering/settings", "admin_a", t1, nil, settings, 400)
	settings["schedule_enabled"] = false
	settings["periods"] = []any{}
	request("PATCH", "/api/ordering/settings", "admin_a", t1, nil, settings, 200)
	first := makeOrder(t1, p1, ca)
	own := map[string]string{"Cookie": ca["Cookie"], "X-Shopper-Order": first}
	other := map[string]string{"Cookie": cb["Cookie"], "X-Shopper-Order": first}
	r := parse(request("GET", "/api/public/buyer-order", "", "", own, nil, 200))
	if r["prep_min"] != float64(15) || r["pending_deadline"] == nil {
		t.Fatal("missing expectations snapshot")
	}
	request("POST", "/api/public/buyer-order/cancel", "", "", other, map[string]any{}, 404)
	request("POST", "/api/public/buyer-order/cancel", "", "", nil, map[string]any{}, 404)
	r = parse(request("POST", "/api/public/buyer-order/cancel", "", "", own, map[string]any{}, 200))
	if r["status"] != "canceled" || r["ended_reason"] == "" {
		t.Fatal("buyer cancellation missing")
	}
	request("POST", "/api/public/buyer-order/cancel", "", "", own, map[string]any{}, 200)
	second := makeOrder(t1, p1, ca)
	must(`UPDATE public.orders SET pending_deadline=clock_timestamp()-interval '1 minute' WHERE id=?`, second)
	d := parse(request("GET", "/api/orders/"+second, "cashier", t1, nil, nil, 200))
	request("PATCH", "/api/orders/"+second+"/status", "cashier", t1, nil, map[string]any{"status": "accepted", "expected_version": d["version"]}, 409)
	must(`SELECT legui_private.expire_pending_orders()`)
	if parse(request("GET", "/api/orders/"+second, "cashier", t1, nil, nil, 200))["status"] != "canceled" {
		t.Fatal("pending not expired")
	}
	guestRaw := base64.RawURLEncoding.EncodeToString([]byte(uuid.NewString()[:32]))
	guestHeaders := map[string]string{"Authorization": "Bearer " + guestRaw}
	guestID := makeOrder(t1, p1, guestHeaders)
	if parse(request("POST", "/api/public/buyer-order/cancel", "", "", guestHeaders, map[string]any{}, 200))["id"] != guestID {
		t.Fatal("guest capability did not cancel its order")
	}
	protected := makeOrder(t1, p1, ca)
	cid, cv := uuid.NewString(), uuid.NewString()
	must(`INSERT INTO legui_private.mp_connections(id,scope,brand_id,environment) VALUES(?,'business',?,'production')`, cid, b1)
	checkout := uuid.NewString()
	must(`INSERT INTO legui_private.mp_checkouts(id,purpose,connection_id,connection_version,brand_id,tenant_id,order_id,environment,currency,amount,target_version,reference,actor_kind,actor_id,operation_key,request_digest,state,expires_at) SELECT ?,'order',?,?,?,tenant_id,id,'production','ARS',total_due,version,?,'user',?,?,?,'review',clock_timestamp()-interval '1 day' FROM public.orders WHERE id=?`, checkout, cid, cv, b1, uuid.NewString(), a1, uuid.NewString(), strings.Repeat("a", 64), protected)
	protectedHeaders := map[string]string{"Cookie": ca["Cookie"], "X-Shopper-Order": protected}
	must(`UPDATE public.orders SET pending_deadline=clock_timestamp()-interval '1 minute' WHERE id=?`, protected)
	request("POST", "/api/public/buyer-order/cancel", "", "", protectedHeaders, map[string]any{}, 409)
	must(`SELECT legui_private.expire_pending_orders()`)
	if parse(request("GET", "/api/orders/"+protected, "cashier", t1, nil, nil, 200))["status"] != "pending" {
		t.Fatal("uncertain expired checkout lost protection")
	}
	must(`UPDATE legui_private.mp_checkouts SET state='failed' WHERE id=?`, checkout)
	request("POST", "/api/orders/"+protected+"/payments", "cashier", t1, nil, map[string]any{"kind": "charge", "method": "transfer", "amount": "100.00"}, 201)
	request("POST", "/api/public/buyer-order/cancel", "", "", protectedHeaders, map[string]any{}, 409)
	must(`SELECT legui_private.expire_pending_orders()`)
	if parse(request("GET", "/api/orders/"+protected, "cashier", t1, nil, nil, 200))["status"] != "pending" {
		t.Fatal("paid pending order auto canceled")
	}
	third := makeOrder(t1, p1, ca)
	must(`UPDATE legui_private.shoppers SET email_verified=true WHERE id=?`, a)
	d = parse(request("GET", "/api/orders/"+third, "cashier", t1, nil, nil, 200))
	d = parse(request("PATCH", "/api/orders/"+third+"/status", "cashier", t1, nil, map[string]any{"status": "accepted", "expected_version": d["version"]}, 200))
	if d["estimated_ready_at"] == nil {
		t.Fatal("acceptance ETA missing")
	}
	request("POST", "/api/public/buyer-order/cancel", "", "", map[string]string{"Cookie": ca["Cookie"], "X-Shopper-Order": third}, map[string]any{}, 409)
	for i := 0; i < 20; i++ {
		changed, e := queueOrderMail(tx)
		if e != nil {
			t.Fatal(e)
		}
		if !changed {
			break
		}
	}
	var n int64
	tx.Raw(`SELECT count(*) FROM legui_private.mail_outbox m JOIN legui_private.order_mail_events e ON e.id=m.order_event_id WHERE e.order_id=?`, third).Scan(&n)
	if n != 1 {
		t.Fatalf("obsolete or duplicate transactional mail: %d", n)
	}
	_, e = queueOrderMail(tx)
	if e != nil {
		t.Fatal(e)
	}
	tx.Raw(`SELECT count(*) FROM legui_private.mail_outbox m JOIN legui_private.order_mail_events e ON e.id=m.order_event_id WHERE e.order_id=?`, third).Scan(&n)
	if n != 1 {
		t.Fatal("event mail duplicated")
	}
	request("POST", "/api/marketing/campaigns", "cashier", t1, nil, map[string]any{}, 403)
	request("POST", "/api/marketing/campaigns", "admin_b", t1, nil, map[string]any{}, 403)
	campaign := parse(request("POST", "/api/marketing/campaigns", "admin_a", t1, nil, map[string]any{"subject": "Nueva burger", "body": "Probá la nueva burger de esta semana", "segment": "all"}, 201))
	id := campaign["id"].(string)
	request("GET", "/api/marketing/campaigns/"+id, "admin_b", t2, nil, nil, 404)
	request("POST", "/api/marketing/campaigns/"+id+"/send", "admin_b", t2, nil, map[string]any{}, 404)
	request("PATCH", "/api/shopper/preferences", "", "", ca, map[string]any{"brand_id": b1, "newsletter": true, "personalization": false}, 200)
	request("PATCH", "/api/shopper/preferences", "", "", cb, map[string]any{"brand_id": b1, "newsletter": true, "personalization": true}, 200)
	preview := parse(request("GET", "/api/marketing/campaigns/"+id, "admin_a", t1, nil, nil, 200))
	if preview["eligible_count"] != float64(1) {
		t.Fatal("unconfirmed subscriber eligible")
	}
	segmented := parse(request("POST", "/api/marketing/campaigns", "admin_a", t1, nil, map[string]any{"subject": "Para los nuevos", "body": "Un beneficio para tu próxima compra", "segment": "new"}, 201))["id"].(string)
	if parse(request("GET", "/api/marketing/campaigns/"+segmented, "admin_a", t1, nil, nil, 200))["eligible_count"] != float64(0) {
		t.Fatal("segmented campaign ignored personalization opt-out")
	}
	request("PATCH", "/api/shopper/preferences", "", "", ca, map[string]any{"brand_id": b2, "newsletter": true, "personalization": true}, 200)
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_FROM", "")
	request("POST", "/api/marketing/campaigns/"+id+"/send", "admin_a", t1, nil, map[string]any{}, 409)
	t.Setenv("SMTP_HOST", "smtp.example.invalid")
	t.Setenv("SMTP_FROM", "sender@example.invalid")
	request("POST", "/api/marketing/campaigns/"+id+"/send", "admin_a", t1, nil, map[string]any{}, 200)
	request("POST", "/api/marketing/campaigns/"+id+"/send", "admin_a", t1, nil, map[string]any{}, 409)
	changed, e := queueCampaignMail(tx)
	if e != nil || !changed {
		t.Fatalf("fanout failed: %v", e)
	}
	_, e = queueCampaignMail(tx)
	if e != nil {
		t.Fatal(e)
	}
	tx.Raw(`SELECT count(*) FROM legui_private.mail_outbox WHERE campaign_id=?`, id).Scan(&n)
	if n != 1 {
		t.Fatal("campaign duplicated or unverified recipient")
	}
	var mid string
	tx.Raw(`SELECT id::text FROM legui_private.mail_outbox WHERE campaign_id=?`, id).Scan(&mid)
	var eligible bool
	tx.Raw(`SELECT legui_private.mail_eligible(?)`, mid).Scan(&eligible)
	if !eligible {
		t.Fatal("confirmed subscriber not eligible")
	}
	raw, e := unsubscribeToken()
	if e != nil {
		t.Fatal(e)
	}
	must(`INSERT INTO legui_private.newsletter_unsubscribes(digest,brand_id,shopper_id) VALUES(?,?,?)`, fmt.Sprintf("%x", sha256.Sum256([]byte(raw))), b1, a)
	request("POST", "/api/public/newsletter/unsubscribe", "", "", nil, map[string]any{"token": raw}, 200)
	request("POST", "/api/public/newsletter/unsubscribe", "", "", nil, map[string]any{"token": raw}, 200)
	tx.Raw(`SELECT legui_private.mail_eligible(?)`, mid).Scan(&eligible)
	if eligible {
		t.Fatal("unsubscribe does not prevent queued send")
	}
	var subscribed bool
	tx.Raw(`SELECT newsletter FROM legui_private.shopper_preferences WHERE shopper_id=? AND brand_id=?`, b, b1).Scan(&subscribed)
	if !subscribed {
		t.Fatal("unsubscribe altered another shopper")
	}
	request("POST", "/api/marketing/campaigns/"+id+"/cancel", "admin_a", t1, nil, map[string]any{}, 200)
	tx.Raw(`SELECT newsletter FROM legui_private.shopper_preferences WHERE shopper_id=? AND brand_id=?`, a, b2).Scan(&subscribed)
	if !subscribed {
		t.Fatal("unsubscribe altered another brand")
	}
	var recipients int64
	tx.Raw(`SELECT count(*) FROM legui_private.newsletter_recipients WHERE campaign_id=?`, id).Scan(&recipients)
	if recipients != 1 {
		t.Fatal("audience not snapshotted")
	}
	t.Logf("Storefront operations: %d HTTP checks; schedule/cancellation/expiry/ETA/tenant scopes/mail dedupe/consent verified; fixtures rolled back, SMTP never called", checks)
}
