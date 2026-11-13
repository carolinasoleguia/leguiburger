package operations

import (
	"bytes"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"leguiburger/internal/auth"
	"leguiburger/internal/brands"
	"leguiburger/internal/businesscare"
	"leguiburger/internal/db"
	"leguiburger/internal/mercadopago"
	"leguiburger/internal/tenants"
	"leguiburger/internal/users"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestBusinessHierarchyIntegration(t *testing.T) {
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
		t.Fatal(e)
	}
	pool, _ := database.DB()
	defer pool.Close()
	if e = VerifyDatabase(database, os.Getenv("DATABASE_URL")); e != nil {
		t.Fatal(e)
	}
	tx := database.Begin()
	defer tx.Rollback()
	previous := db.DB
	db.DB = tx
	defer func() { db.DB = previous }()
	t.Setenv("JWT_SECRET", "hierarchy-integration-secret-local-fixture")
	t.Setenv("AUTH_ENCRYPTION_KEY", "hierarchy-integration-encrypted-mail")
	t.Setenv("PUBLIC_SITE_URL", "http://127.0.0.1:5173")
	t.Setenv("APP_ENV", "development")
	t.Setenv("AUTH_REQUIRE_PLATFORM_MFA", "false")
	must := func(q string, args ...any) {
		t.Helper()
		if e := tx.Exec(q, args...).Error; e != nil {
			t.Fatal(e)
		}
	}
	password := "hierarchy-fixture-password-123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	platform := uuid.NewString()
	must(`INSERT INTO public.users(id,email,password_hash,role,is_active) VALUES(?,?,?,'super_admin',true)`, platform, platform+"@example.invalid", string(hash))
	svc, e := auth.NewService(auth.NewRepository(), tenants.NewRepository())
	if e != nil {
		t.Fatal(e)
	}
	mux := http.NewServeMux()
	auth.RegisterSecurityRoutes(mux, tx, svc)
	RegisterRoutes(mux, tx)
	businesscare.Register(mux, tx)
	mercadopago.NewHandler(tx, mercadopago.Config{Environment: "sandbox"}, nil).Register(mux)
	uh := users.NewHandler(users.NewService(users.NewRepository(), brands.NewRepository()))
	mux.HandleFunc("/api/users", auth.AuthMiddlewareWithDB(tx, auth.AuthorizeLegacy(uh.HandleUserRoutes)))
	request := func(method, path, token, key string, body any, want int) *httptest.ResponseRecorder {
		t.Helper()
		encoded, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(encoded))
		r.RemoteAddr = "192.0.2.71:1234"
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: want %d got %d: %s", method, path, want, w.Code, w.Body.String())
		}
		return w
	}
	obj := func(w *httptest.ResponseRecorder) map[string]any {
		t.Helper()
		var out map[string]any
		if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil {
			t.Fatal(e)
		}
		return out
	}
	login := func(email string) string {
		t.Helper()
		return obj(request("POST", "/api/auth/login", "", "", map[string]any{"email": email, "password": password}, 200))["token"].(string)
	}
	capability := func(link string) string { u, _ := url.Parse(link); return strings.TrimPrefix(u.Fragment, "token=") }
	platformToken := login(platform + "@example.invalid")
	email := uuid.NewString() + "@example.invalid"
	body := map[string]any{"name": "__hierarchy_fixture", "email": email, "local": "h-" + uuid.NewString()[:18], "base_location_limit": 2, "base_collaborator_limit": 1}
	created := obj(request("POST", "/api/platform/restaurants", platformToken, "hierarchy-create", body, 200))
	brand, home := created["brand_id"].(string), created["tenant_id"].(string)
	fixtureAccount, fixtureSubscription := paidLegacyFixture(t, tx, brand)
	replay := obj(request("POST", "/api/platform/restaurants", platformToken, "hierarchy-create", body, 200))
	if replay["brand_id"] != brand {
		t.Fatal("business replay duplicated identity")
	}
	changed := map[string]any{"name": "changed", "email": email, "local": "changed"}
	request("POST", "/api/platform/restaurants", platformToken, "hierarchy-create", changed, 400)
	token := capability(created["invitation"].(map[string]any)["invitation_url"].(string))
	accept := map[string]any{"token": token, "first_name": "Owner", "last_name": "Fixture", "password": password}
	request("POST", "/api/auth/business-invitations/accept", "", "", accept, 200)
	request("POST", "/api/auth/business-invitations/accept", "", "", accept, 400)
	ownerToken := login(email)
	request("GET", "/api/business", ownerToken, "", nil, 200)
	request("POST", "/api/platform/restaurants", ownerToken, "forbidden-create", body, 403)
	request("GET", "/api/platform/restaurants", ownerToken, "", nil, 403)
	// super_admin must share the same legacy owner-management permissions.
	request("GET", "/api/users", platformToken, "", nil, 200)
	locationBody := map[string]any{"subdomain": "h-" + uuid.NewString()[:18], "display_name": "Second local", "active": true}
	second := obj(request("POST", "/api/business/locations", ownerToken, "hierarchy-location", locationBody, 200))["id"].(string)
	locationBody["subdomain"] = "h-" + uuid.NewString()[:18]
	request("POST", "/api/business/locations", ownerToken, "hierarchy-third", locationBody, 400)
	personEmail := uuid.NewString() + "@example.invalid"
	invitation := obj(request("POST", "/api/team/invitations", ownerToken, "", map[string]any{"email": personEmail, "tenant_id": home, "role": "cashier"}, 201))
	request("POST", "/api/team/invitations", ownerToken, "", map[string]any{"email": uuid.NewString() + "@example.invalid", "tenant_id": home, "role": "cashier"}, 400)
	request("POST", "/api/auth/invitations/accept", "", "", map[string]any{"token": capability(invitation["invitation_url"].(string)), "first_name": "Cashier", "last_name": "Fixture", "password": password}, 201)
	staffToken := login(personEmail)
	request("POST", "/api/auth/location", staffToken, "", map[string]string{"tenant_id": second}, 403)
	var employee string
	tx.Raw(`SELECT id::text FROM public.employees WHERE email=?`, personEmail).Scan(&employee)
	update := map[string]any{"first_name": "Cashier", "last_name": "Fixture", "role": "cashier", "is_active": true, "tenant_id": home, "location_ids": []string{home, second}}
	request("PATCH", "/api/business/team/"+employee, ownerToken, "hierarchy-assign", update, 200)
	request("GET", "/api/business/team", staffToken, "", nil, 403)
	switched := obj(request("POST", "/api/auth/location", staffToken, "", map[string]string{"tenant_id": second}, 200))
	secondToken := switched["token"].(string)
	if switched["user"].(map[string]any)["tenant_id"] != second {
		t.Fatal("wrong active context")
	}
	request("GET", "/api/auth/me", staffToken, "", nil, 401)
	request("GET", "/api/kitchen", secondToken, "", nil, 200)
	// Real writes in a secondary location must retain employee attribution even
	// after that assignment is removed or the employee changes their home.
	product := uuid.NewString()
	must(`INSERT INTO public.products(id,brand_id,name,base_price,is_active) VALUES(?,?,'__hierarchy_burger',100,true)`, product, brand)
	must(`INSERT INTO public.tenant_products(tenant_id,product_id,current_stock,track_stock,is_available,is_active) VALUES(?,?,10,true,true,true)`, second, product)
	order := obj(request("POST", "/api/orders", secondToken, "hierarchy-secondary-order", map[string]any{"items": []any{map[string]any{"product_id": product, "quantity": 1}}}, 201))
	orderID := order["id"].(string)
	request("PATCH", "/api/orders/"+orderID+"/status", secondToken, "hierarchy-secondary-accept", map[string]any{"status": "accepted", "expected_version": order["version"]}, 200)
	update["location_ids"] = []string{home}
	request("PATCH", "/api/business/team/"+employee, ownerToken, "hierarchy-revoke-assignment", update, 200)
	request("GET", "/api/auth/me", secondToken, "", nil, 401)
	var preserved int64
	tx.Raw(`SELECT count(*) FROM public.order_status_events WHERE order_id=? AND actor_employee_id=?`, orderID, employee).Scan(&preserved)
	if preserved < 2 {
		t.Fatal("secondary location attribution lost")
	}
	update["role"] = "admin"
	request("PATCH", "/api/business/team/"+employee, ownerToken, "hierarchy-role-escalation", update, 400)
	update["role"] = "cashier"
	otherBrand, otherLocal := uuid.NewString(), uuid.NewString()
	must(`INSERT INTO public.brands(id,name) VALUES(?,'__hierarchy_other')`, otherBrand)
	must(`INSERT INTO public.tenants(id,brand_id,subdomain) VALUES(?,?,?)`, otherLocal, otherBrand, "h-"+uuid.NewString()[:18])
	update["location_ids"] = []string{home, otherLocal}
	request("PATCH", "/api/business/team/"+employee, ownerToken, "hierarchy-cross-business", update, 403)
	// Owner transfer keeps the current account active until acceptance.
	successorEmail := uuid.NewString() + "@example.invalid"
	transfer := obj(request("POST", "/api/business/"+brand+"/owner-invitation", ownerToken, "hierarchy-transfer", map[string]any{"email": successorEmail, "current_password": password}, 200))
	request("GET", "/api/auth/me", ownerToken, "", nil, 200)
	request("POST", "/api/auth/business-invitations/accept", "", "", map[string]any{"token": capability(transfer["invitation_url"].(string)), "first_name": "Successor", "last_name": "Fixture", "password": password}, 200)
	request("GET", "/api/auth/me", ownerToken, "", nil, 401)
	successor := login(successorEmail)
	staffToken = login(personEmail)
	status := map[string]any{"suspended": true, "reason": "Hierarchy fixture suspension", "base_location_limit": 2, "base_collaborator_limit": 1}
	request("PATCH", "/api/platform/restaurants/"+brand+"/status", platformToken, "hierarchy-suspend", status, 200)
	request("GET", "/api/auth/me", staffToken, "", nil, 401)
	request("GET", "/api/business", successor, "", nil, 200)
	request("GET", "/api/billing/my-account", successor, "", nil, 200)
	request("GET", "/api/business/locations", successor, "", nil, 403)
	request("GET", "/api/analytics/restaurant?brand_id="+brand, successor, "", nil, 403)
	request("POST", "/api/auth/login", "", "", map[string]any{"email": personEmail, "password": password}, 401)
	status["suspended"] = false
	status["reason"] = ""
	request("PATCH", "/api/platform/restaurants/"+brand+"/status", platformToken, "hierarchy-resume", status, 200)
	request("GET", "/api/business/locations", successor, "", nil, 200)
	request("GET", "/api/auth/me", staffToken, "", nil, 401)
	login(personEmail)
	// Existing accounts prove their password; no duplicate identity or forced reset.
	attach := obj(request("POST", "/api/team/invitations", successor, "", map[string]any{"email": personEmail, "tenant_id": second, "role": "cashier"}, 201))
	existingAccept := map[string]any{"token": capability(attach["invitation_url"].(string)), "first_name": "Cashier", "last_name": "Fixture", "current_password": "wrong"}
	request("POST", "/api/auth/invitations/accept", "", "", existingAccept, 401)
	existingAccept["current_password"] = password
	request("POST", "/api/auth/invitations/accept", "", "", existingAccept, 201)
	var identities int64
	tx.Raw(`SELECT count(*) FROM public.employees WHERE email=?`, personEmail).Scan(&identities)
	if identities != 1 {
		t.Fatal("multi-location invitation duplicated identity")
	}
	returnInvite := obj(request("POST", "/api/business/"+brand+"/owner-invitation", successor, "hierarchy-return-owner", map[string]any{"email": email, "current_password": password}, 200))
	request("POST", "/api/auth/business-invitations/accept", "", "", map[string]any{"token": capability(returnInvite["invitation_url"].(string)), "first_name": "Owner", "last_name": "Fixture", "current_password": password}, 200)
	request("GET", "/api/auth/me", successor, "", nil, 401)
	successor = login(email)
	// A real current contract overrides base capacity; sandbox contracts do not.
	plan := obj(request("POST", "/api/platform/billing/plans", platformToken, "hierarchy-plan", map[string]any{"code": "hierarchy_" + strings.ReplaceAll(uuid.NewString(), "-", ""), "name": "Hierarchy plan"}, 201))
	price := obj(request("POST", "/api/platform/billing/plans/"+plan["id"].(string)+"/prices", platformToken, "hierarchy-price", map[string]any{"amount": "1000", "currency": "ARS", "interval_months": 1, "billing_unit": "brand", "maximum_locations": 3, "maximum_collaborators": 2}, 201))
	must(`UPDATE legui_private.subscriptions SET status='canceled' WHERE id=?`, fixtureSubscription)
	account := map[string]any{"id": fixtureAccount}
	request("POST", "/api/platform/billing/subscriptions", platformToken, "hierarchy-contract", map[string]any{"account_id": account["id"], "plan_price_id": price["id"], "environment": "production", "quantity": 1, "status": "active", "period_start": time.Now().Add(-time.Hour), "period_end": time.Now().AddDate(0, 1, 0)}, 201)
	locationBody["subdomain"] = "h-" + uuid.NewString()[:18]
	request("POST", "/api/business/locations", successor, "hierarchy-upgraded-local", locationBody, 200)
	locationBody["subdomain"] = "h-" + uuid.NewString()[:18]
	request("POST", "/api/business/locations", successor, "hierarchy-contracted-limit", locationBody, 400)
	// Care endpoints share the same live session and resource boundaries.
	request("GET", "/api/care/support", "", "", nil, 401)
	request("GET", "/api/care/errors", successor, "", nil, 403)
	request("POST", "/api/care/errors", successor, "", map[string]any{"code": "render", "url": "private-fragment"}, 400)
	request("POST", "/api/care/errors", successor, "", map[string]any{"code": "render"}, 200)
	ticketBody := map[string]any{"subject": "Fixture support", "body": "Please inspect this fixture"}
	ticket := obj(request("POST", "/api/care/support", successor, "care-ticket", ticketBody, 200))["id"].(string)
	if replay := obj(request("POST", "/api/care/support", successor, "care-ticket", ticketBody, 200)); replay["id"] != ticket {
		t.Fatal("support retry duplicated ticket")
	}
	request("GET", "/api/care/support/"+uuid.NewString(), successor, "", nil, 403)
	request("GET", "/api/care/support/"+ticket, platformToken, "", nil, 200)
	platformTicket := obj(request("POST", "/api/care/support", platformToken, "care-platform-ticket", ticketBody, 200))["id"].(string)
	request("GET", "/api/care/support/"+platformTicket, successor, "", nil, 403)
	request("POST", "/api/care/support/"+ticket+"/messages", platformToken, "care-reply", map[string]any{"body": "We are reviewing your request"}, 200)
	request("POST", "/api/care/support/"+ticket+"/close", successor, "care-resolve", map[string]any{}, 200)
	t.Setenv("BACKUP_ENCRYPTION_KEY", "care-integration-private-key-at-least-32-bytes")
	backup := obj(request("POST", "/api/care/backups", successor, "care-backup", map[string]any{}, 200))
	if backup["state"] != "pending" || backup["report"].(map[string]any)["verified"] != true {
		t.Fatal("backup bypassed restore validation")
	}
	request("GET", "/api/care/backups/"+backup["id"].(string)+"/download", successor, "", nil, 403)
	var cleartext bool
	must(`SELECT EXISTS(SELECT 1 FROM legui_private.business_backups WHERE id=? AND position('fixture'::bytea IN payload_cipher)>0)`, backup["id"])
	tx.Raw(`SELECT position('fixture'::bytea IN payload_cipher)>0 FROM legui_private.business_backups WHERE id=?`, backup["id"]).Scan(&cleartext)
	if cleartext {
		t.Fatal("backup contains plaintext")
	}
	closeBody := map[string]any{"password": "wrong", "confirmation": "CERRAR RESTAURANTE", "reason": "Fixture closure"}
	request("POST", "/api/care/closure", successor, "care-wrong-password", closeBody, 403)
	closeBody["password"] = password
	request("POST", "/api/care/closure", successor, "care-close", closeBody, 200)
	request("GET", "/api/care/closure", successor, "", nil, 200)
	request("GET", "/api/care/backups", successor, "", nil, 200)
	request("GET", "/api/business/locations", successor, "", nil, 403)
	var closed, noRenewal bool
	tx.Raw(`SELECT closed_at IS NOT NULL FROM public.brands WHERE id=?`, brand).Scan(&closed)
	tx.Raw(`SELECT bool_and(s.cancel_at_period_end) FROM legui_private.subscriptions s JOIN legui_private.billing_accounts a ON a.id=s.account_id WHERE a.brand_id=? AND s.status<>'canceled'`, brand).Scan(&noRenewal)
	if !closed || !noRenewal {
		t.Fatal("closure did not stop local renewals")
	}
	// The primary account can deactivate only after archiving/transferring.
	request("POST", "/api/care/account-close", successor, "care-account-close", map[string]any{"password": password, "confirmation": "CERRAR CUENTA"}, 200)
	request("GET", "/api/auth/me", successor, "", nil, 401)
	var primaryGone bool
	tx.Raw(`SELECT primary_owner_id IS NULL FROM public.brands WHERE id=?`, brand).Scan(&primaryGone)
	if !primaryGone {
		t.Fatal("archived primary responsibility was not released")
	}
	t.Log("Hierarchy and care integration passed: onboarding, transfers, quotas, context, support/reply/replay, sanitized monitoring, encrypted verified snapshot, reauthenticated archival and retained history; all fixtures roll back")
}
