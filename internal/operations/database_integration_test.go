package operations

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"leguiburger/internal/auth"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// Opt-in remote integration runs in ONE outer transaction. Nested endpoint transactions
// use savepoints; every fixture, receipt, sale and payment is rolled back at the end.
var databaseIntegration = flag.Bool("db-integration", false, "Run rollback-only tests against the pinned project")

func TestDatabaseIntegration(t *testing.T) {
	if !*databaseIntegration && os.Getenv("LEGUi_DB_INTEGRATION") != "1" {
		t.Skip("set LEGUi_DB_INTEGRATION=1 to run rollback-only database/API checks")
	}
	_ = godotenv.Load("../../.env")
	cfg, err := pgx.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil || cfg.User != "postgres.ljdebnvzdrlqdxnligvh" {
		t.Fatal("integration target is not the explicitly selected LeguiBurger project")
	}
	database, err := gorm.Open(postgres.New(postgres.Config{DSN: os.Getenv("DATABASE_URL"), PreferSimpleProtocol: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("database connection unavailable")
	}
	if err = VerifyDatabase(database, os.Getenv("DATABASE_URL")); err != nil {
		t.Fatal(err)
	}
	pool, _ := database.DB()
	defer pool.Close()
	tx := database.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	must := func(q string, args ...any) {
		t.Helper()
		if err := tx.Exec(q, args...).Error; err != nil {
			t.Fatal(err)
		}
	}
	var migrations int64
	if err = tx.Raw(`SELECT count(*) FROM legui_private.schema_migrations WHERE version='202611100005_backend_workflow_guards'`).Scan(&migrations).Error; err != nil || migrations != 1 {
		t.Fatal("apply the reviewed workflow migration before integration checks")
	}
	b1, b2, t1, t2, product, extra, supply, shipping := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	owner, manager, other, employee := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	must(`INSERT INTO public.brands(id,name,data_classification) VALUES(?,'__api_fixture_a','production'),(?,'__api_fixture_b','test')`, b1, b2)
	must(`INSERT INTO public.tenants(id,brand_id,subdomain) VALUES(?,?,'__api_fixture_a'),(?,?,'__api_fixture_b')`, t1, b1, t2, b2)
	must(`INSERT INTO public.users(id,email,password_hash,role,brand_id,is_active) VALUES(?,?,'fixture-not-a-password','owner',NULL,true),(?,?,'fixture-not-a-password','admin',?,true),(?,?,'fixture-not-a-password','admin',?,true)`, owner, owner+"@example.invalid", manager, manager+"@example.invalid", b1, other, other+"@example.invalid", b2)
	must(`INSERT INTO public.employees(id,tenant_id,first_name,last_name,email,password_hash,role,is_active) VALUES(?,?,'Fixture','Employee',?,'fixture-not-a-password','employee',true)`, employee, t1, employee+"@example.invalid")
	must(`INSERT INTO public.products(id,brand_id,name,base_price,is_active) VALUES(?,?,'__api_burger',100,true)`, product, b1)
	must(`INSERT INTO public.tenant_products(tenant_id,product_id,current_stock,track_stock,is_available,is_active) VALUES(?,?,5,true,true,true)`, t1, product)
	must(`INSERT INTO public.extras(id,tenant_id,name,current_price,current_stock,current_unit_cost,track_stock,is_active) VALUES(?,?,'__api_extra',10,10,5,true,true)`, extra, t1)
	must(`INSERT INTO public.supplies(id,tenant_id,name,current_wholesale_cost,current_stock,measurement_unit,is_active) VALUES(?,?,'__api_supply',20,100,'unit',true)`, supply, t1)
	must(`INSERT INTO public.recipes(product_id,supply_id,quantity_used) VALUES(?,?,2)`, product, supply)
	must(`INSERT INTO public.shipping_methods(id,tenant_id,name,cost,is_active) VALUES(?,?,'__api_shipping',15,true)`, shipping, t1)
	if err = auth.ConfigureJWTSecret("isolated-integration-fixture-secret-not-used-by-running-server"); err != nil {
		t.Fatal(err)
	}
	token := func(id, role string, tenant, brand *string) string {
		t.Helper()
		kind, tenantID, brandID := "user", "", ""
		if tenant != nil {
			tenantID = *tenant
			kind = "employee"
			brandID = b1
		}
		if brand != nil {
			brandID = *brand
		}
		sid := uuid.NewString()
		must(`INSERT INTO legui_private.auth_sessions(id,principal_kind,principal_id,role,brand_id,tenant_id,credential_digest,mfa_verified) VALUES(?,?,?,?,nullif(?,'')::uuid,nullif(?,'')::uuid,?,?)`, sid, kind, id, role, brandID, tenantID, fmt.Sprintf("%x", sha256.Sum256([]byte("fixture-not-a-password"))), role == "owner")
		value, e := auth.SessionToken(auth.SessionPrincipal{ID: id, Kind: kind, Role: role, BrandID: brandID, TenantID: tenantID}, sid, role == "owner", "access")
		if e != nil {
			t.Fatal(e)
		}
		return value
	}
	ownerToken, managerToken, otherToken, employeeToken := token(owner, "owner", nil, nil), token(manager, "admin", nil, &b1), token(other, "admin", nil, &b2), token(employee, "employee", &t1, nil)
	mux := http.NewServeMux()
	RegisterRoutes(mux, tx)
	request := func(method, path, authorization, tenant, key string, body any, expected int) *httptest.ResponseRecorder {
		t.Helper()
		var encoded []byte
		if body != nil {
			encoded, err = json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
		}
		r := httptest.NewRequest(method, path, bytes.NewReader(encoded))
		if authorization != "" {
			r.Header.Set("Authorization", "Bearer "+authorization)
		}
		if tenant != "" {
			r.Header.Set("X-Tenant-ID", tenant)
		}
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != expected {
			t.Fatalf("%s %s expected %d got %d: %s", method, path, expected, w.Code, w.Body.String())
		}
		return w
	}
	object := func(w *httptest.ResponseRecorder) map[string]any {
		t.Helper()
		var value map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	idOf := func(w *httptest.ResponseRecorder) string {
		t.Helper()
		value, ok := object(w)["id"].(string)
		if !ok || !validID(value) {
			t.Fatal("missing ID", w.Body.String())
		}
		return value
	}
	checkNumeric := func(q string, want int, args ...any) {
		t.Helper()
		var got string
		if err := tx.Raw(q, args...).Scan(&got).Error; err != nil {
			t.Fatal(err)
		}
		actual, ok := new(big.Rat).SetString(got)
		if !ok || actual.Cmp(big.NewRat(int64(want), 1)) != 0 {
			t.Fatalf("expected %d got %s for %s", want, got, q)
		}
	}
	request("GET", "/api/analytics/platform", "", "", "", nil, 401)
	request("GET", "/api/analytics/platform", managerToken, "", "", nil, 403)
	request("GET", "/api/analytics/restaurant", employeeToken, t1, "", nil, 403)
	request("GET", "/api/orders", otherToken, t1, "", nil, 403)
	operational := request("GET", "/api/analytics/operations", employeeToken, t1, "", nil, 200)
	if strings.Contains(operational.Body.String(), "gross_margin") || strings.Contains(operational.Body.String(), "collected") {
		t.Fatal("employee sees financial analytics")
	}
	request("POST", "/api/orders", ownerToken, t1, "", map[string]any{"items": []any{map[string]any{"product_id": product, "quantity": 1}}}, 400)
	body := map[string]any{"channel": "delivery", "shipping_method_id": shipping, "shipping_address": "Fixture", "items": []any{map[string]any{"product_id": product, "quantity": 2, "extras": []any{map[string]any{"extra_id": extra, "quantity": 1}}}}}
	created := request("POST", "/api/orders", ownerToken, t1, "fixture-order", body, 201)
	orderID := idOf(created)
	repeated := request("POST", "/api/orders", ownerToken, t1, "fixture-order", body, 201)
	if repeated.Header().Get("Idempotent-Replayed") != "true" || !bytes.Equal(created.Body.Bytes(), repeated.Body.Bytes()) {
		t.Fatal("order replay changes response")
	}
	if object(created)["total_due"].(float64) != 235 {
		t.Fatal("order total ignores quantities, extras or shipping", created.Body.String())
	}
	checkNumeric(`SELECT count(*) FROM public.orders WHERE tenant_id=?`, 1, t1)
	version := int(object(created)["version"].(float64))
	accepted := request("PATCH", "/api/orders/"+orderID+"/status", ownerToken, t1, "fixture-accept", map[string]any{"status": "accepted", "expected_version": version}, 200)
	checkNumeric(`SELECT current_stock FROM public.tenant_products WHERE tenant_id=? AND product_id=?`, 3, t1, product)
	checkNumeric(`SELECT current_stock FROM public.extras WHERE id=?`, 8, extra)
	checkNumeric(`SELECT frozen_cost_unit FROM public.sales WHERE order_id=?`, 45, orderID)
	checkNumeric(`SELECT count(*) FROM public.sales WHERE order_id=?`, 1, orderID)
	paid := request("POST", "/api/orders/"+orderID+"/payments", ownerToken, t1, "fixture-order-payment", map[string]any{"kind": "charge", "amount": "235.00", "method": "cash"}, 201)
	paymentID := idOf(paid)
	request("POST", "/api/orders/"+orderID+"/payments", ownerToken, t1, "fixture-order-payment", map[string]any{"kind": "charge", "amount": "234.00", "method": "cash"}, 409)
	request("POST", "/api/orders/"+orderID+"/payments", ownerToken, t1, "fixture-overpayment", map[string]any{"kind": "charge", "amount": "1", "method": "cash"}, 422)
	request("PATCH", "/api/orders/"+orderID+"/status", ownerToken, t1, "fixture-stale-status", map[string]any{"status": "preparing", "expected_version": version}, 409)
	businessStats := object(request("GET", "/api/analytics/restaurant", managerToken, t1, "", nil, 200))
	money := businessStats["money"].([]any)[0].(map[string]any)
	if money["sales"].(float64) != 235 || money["collected"].(float64) != 235 || money["gross_margin"].(float64) != 145 {
		t.Fatal("wrong financial definitions", money)
	}
	nowOrder := object(request("GET", "/api/orders/"+orderID, ownerToken, t1, "", nil, 200))
	request("PATCH", "/api/orders/"+orderID+"/status", ownerToken, t1, "fixture-cancel-before-refund", map[string]any{"status": "canceled", "reason": "Fixture", "expected_version": nowOrder["version"]}, 409)
	request("POST", "/api/orders/"+orderID+"/payments", ownerToken, t1, "fixture-refund", map[string]any{"kind": "refund", "original_payment_id": paymentID, "amount": "235.00", "method": "cash"}, 201)
	nowOrder = object(request("GET", "/api/orders/"+orderID, ownerToken, t1, "", nil, 200))
	request("PATCH", "/api/orders/"+orderID+"/status", ownerToken, t1, "fixture-cancel", map[string]any{"status": "canceled", "reason": "Fixture cancellation", "expected_version": nowOrder["version"]}, 200)
	checkNumeric(`SELECT current_stock FROM public.tenant_products WHERE tenant_id=? AND product_id=?`, 5, t1, product)
	sandboxOrder := object(request("POST", "/api/orders", ownerToken, t1, "fixture-sandbox-order", map[string]any{"environment": "sandbox", "items": []any{map[string]any{"product_id": product, "quantity": 1}}}, 201))
	request("PATCH", "/api/orders/"+sandboxOrder["id"].(string)+"/status", ownerToken, t1, "fixture-sandbox-accept", map[string]any{"status": "accepted", "expected_version": sandboxOrder["version"]}, 200)
	checkNumeric(`SELECT current_stock FROM public.tenant_products WHERE tenant_id=? AND product_id=?`, 5, t1, product)
	checkNumeric(`SELECT count(*) FROM public.activity_events WHERE tenant_id=? AND event_name='orders.status' AND environment='sandbox'`, 1, t1)
	checkNumeric(`SELECT current_stock FROM public.extras WHERE id=?`, 10, extra)
	checkNumeric(`SELECT count(*) FROM public.sales WHERE order_id=? AND voided_at IS NOT NULL`, 1, orderID)
	_ = accepted
	var stockVersion int64
	if err = tx.Raw(`SELECT stock_version FROM public.tenant_products WHERE tenant_id=? AND product_id=?`, t1, product).Scan(&stockVersion).Error; err != nil {
		t.Fatal(err)
	}
	movement := map[string]any{"item_kind": "product", "item_id": product, "quantity_delta": "2", "expected_version": stockVersion, "reason": "purchase"}
	request("POST", "/api/inventory/movements", managerToken, t1, "fixture-stock", movement, 201)
	request("POST", "/api/inventory/movements", managerToken, t1, "fixture-stock", movement, 201)
	request("POST", "/api/inventory/movements", managerToken, t1, "fixture-stock-stale", movement, 409)
	checkNumeric(`SELECT current_stock FROM public.tenant_products WHERE tenant_id=? AND product_id=?`, 7, t1, product)
	request("PATCH", "/api/inventory/extra/"+extra+"/settings", managerToken, t1, "fixture-cost", map[string]any{"unit_cost": "6.00", "minimum_stock": "2"}, 200)
	checkNumeric(`SELECT frozen_cost_unit FROM public.sales WHERE order_id=?`, 45, orderID)
	expense := request("POST", "/api/expenses", managerToken, t1, "fixture-expense", map[string]any{"category": "rent", "amount": "100.00", "currency": "ARS", "incurred_at": time.Now().UTC()}, 201)
	request("POST", "/api/expenses/"+idOf(expense)+"/void", managerToken, t1, "fixture-expense-void", map[string]any{"reason": "Fixture correction"}, 200)
	request("GET", "/api/inventory", employeeToken, t1, "", nil, 200)
	stockView := request("GET", "/api/inventory", employeeToken, t1, "", nil, 200)
	if strings.Contains(stockView.Body.String(), "unit_cost") {
		t.Fatal("inventory leaks costs to collaborator")
	}
	request("GET", "/api/inventory/movements", managerToken, t1, "", nil, 200)
	request("GET", "/api/expenses", managerToken, t1, "", nil, 200)
	// Unknown costs must remain unknown; multiple cart lines consume each stock balance only once.
	must(`SELECT set_config('leguiburger.inventory_reason','',true),set_config('leguiburger.source_order_id','',true),set_config('leguiburger.operation_key','',true),set_config('leguiburger.request_digest','',true),set_config('leguiburger.expected_stock_version','',true)`)
	unknownProduct := uuid.NewString()
	must(`INSERT INTO public.products(id,brand_id,name,base_price,is_active) VALUES(?,?,'__api_unknown_cost',50,true)`, unknownProduct, b1)
	must(`INSERT INTO public.tenant_products(tenant_id,product_id,current_stock,track_stock,is_available,is_active) VALUES(?,?,2,true,true,true)`, t1, unknownProduct)
	unknownOrder := object(request("POST", "/api/orders", ownerToken, t1, "fixture-unknown-order", map[string]any{"items": []any{map[string]any{"product_id": unknownProduct, "quantity": 1}}}, 201))
	unknownID := unknownOrder["id"].(string)
	request("PATCH", "/api/orders/"+unknownID+"/status", ownerToken, t1, "fixture-unknown-accept", map[string]any{"status": "accepted", "expected_version": unknownOrder["version"]}, 200)
	checkNumeric(`SELECT count(*) FROM public.sales WHERE order_id=? AND frozen_cost_unit IS NULL`, 1, unknownID)
	unknownStats := object(request("GET", "/api/analytics/restaurant", managerToken, t1, "", nil, 200))
	if unknownStats["money"].([]any)[0].(map[string]any)["gross_margin"] != nil {
		t.Fatal("unknown costs must not become an invented profit")
	}
	multiOrder := object(request("POST", "/api/orders", ownerToken, t1, "fixture-multi-line-order", map[string]any{"items": []any{map[string]any{"product_id": product, "quantity": 1}, map[string]any{"product_id": product, "quantity": 1}}}, 201))
	request("PATCH", "/api/orders/"+multiOrder["id"].(string)+"/status", ownerToken, t1, "fixture-multi-line-accept", map[string]any{"status": "accepted", "expected_version": multiOrder["version"]}, 200)
	checkNumeric(`SELECT current_stock FROM public.tenant_products WHERE tenant_id=? AND product_id=?`, 5, t1, product)

	plan := request("POST", "/api/platform/billing/plans", ownerToken, "", "fixture-plan", map[string]any{"code": "fixture_api_plan", "name": "Fixture API plan"}, 201)
	planID := idOf(plan)
	price := request("POST", "/api/platform/billing/plans/"+planID+"/prices", ownerToken, "", "fixture-price", map[string]any{"currency": "ARS", "amount": "1000.00", "interval_months": 1, "billing_unit": "brand"}, 201)
	priceID := idOf(price)
	account := request("POST", "/api/platform/billing/accounts", ownerToken, "", "fixture-account", map[string]any{"brand_id": b1, "currency": "ARS"}, 201)
	accountID := idOf(account)
	starts, ends := time.Now().UTC().Add(-time.Hour), time.Now().UTC().AddDate(0, 1, 0)
	subscription := request("POST", "/api/platform/billing/subscriptions", ownerToken, "", "fixture-subscription", map[string]any{"account_id": accountID, "plan_price_id": priceID, "environment": "production", "quantity": 1, "status": "active", "period_start": starts, "period_end": ends}, 201)
	subscriptionID := idOf(subscription)
	invoice := request("POST", "/api/platform/billing/invoices", ownerToken, "", "fixture-invoice", map[string]any{"subscription_id": subscriptionID, "period_start": starts, "period_end": ends, "due_at": time.Now().UTC().AddDate(0, 0, 7)}, 201)
	invoiceID := idOf(invoice)
	membershipPayment := request("POST", "/api/platform/billing/invoices/"+invoiceID+"/payments", ownerToken, "", "fixture-membership-payment", map[string]any{"kind": "charge", "amount": "1000.00", "method": "transfer"}, 201)
	membershipPaymentID := idOf(membershipPayment)
	stats := object(request("GET", "/api/analytics/platform", ownerToken, "", "", nil, 200))
	platformMoney := stats["money"].([]any)[0].(map[string]any)
	if platformMoney["active_mrr"].(float64) != 1000 || platformMoney["collected"].(float64) != 1000 {
		t.Fatal("platform analytics mix restaurant money or cash/MRR", platformMoney)
	}
	request("POST", "/api/platform/billing/invoices/"+invoiceID+"/credits", ownerToken, "", "fixture-membership-credit", map[string]any{"amount": "500.00", "reason": "Fixture correction"}, 201)
	request("POST", "/api/platform/billing/invoices/"+invoiceID+"/payments", ownerToken, "", "fixture-membership-refund", map[string]any{"kind": "refund", "original_payment_id": membershipPaymentID, "amount": "500.00", "method": "transfer"}, 201)
	checkNumeric(`SELECT balance_due FROM legui_private.billing_invoices WHERE id=?`, 0, invoiceID)
	request("PATCH", "/api/platform/billing/subscriptions/"+subscriptionID, ownerToken, "", "fixture-subscription-cancel", map[string]any{"expected_version": 1, "status": "canceled"}, 200)
	request("PATCH", "/api/platform/billing/subscriptions/"+subscriptionID, ownerToken, "", "fixture-subscription-stale", map[string]any{"expected_version": 1, "status": "active"}, 409)
	for _, path := range []string{"/api/platform/businesses", "/api/platform/billing/plans", "/api/platform/billing/accounts", "/api/platform/billing/subscriptions", "/api/platform/billing/invoices", "/api/platform/billing/webhook-events"} {
		request("GET", path, ownerToken, "", "", nil, 200)
		request("GET", path, managerToken, "", "", nil, 403)
	}
	request("GET", "/api/platform/billing/invoices/"+invoiceID+"/payments", ownerToken, "", "", nil, 200)
	// Workspace flows reuse the same rollback transaction and real session middleware.
	request("GET", "/api/kitchen", otherToken, t1, "", nil, 403)
	request("GET", "/api/pos/catalog", employeeToken, t1, "", nil, 403)
	request("GET", "/api/crm/customers", employeeToken, t1, "", nil, 403)
	request("GET", "/api/onboarding", employeeToken, t1, "", nil, 403)
	request("GET", "/api/platform/health", managerToken, "", "", nil, 403)
	request("GET", "/api/platform/health", ownerToken, "", "", nil, 200)
	request("GET", "/api/pos/catalog", ownerToken, t1, "", nil, 200)
	request("GET", "/api/crm/customers?search=fixture", managerToken, t1, "", nil, 200)
	request("GET", "/api/onboarding", managerToken, t1, "", nil, 200)
	request("POST", "/api/onboarding/complete", managerToken, t1, "fixture-onboarding", map[string]any{}, 200)
	request("POST", "/api/onboarding/complete", otherToken, t1, "fixture-onboarding-wrong", map[string]any{}, 403)
	queue := request("GET", "/api/kitchen", employeeToken, t1, "", nil, 200)
	for _, secret := range []string{"shipping_address", "total_paid", "customer_id", "final_price_unit"} {
		if strings.Contains(queue.Body.String(), secret) {
			t.Fatalf("kitchen leaks %s", secret)
		}
	}
	prepID := multiOrder["id"].(string)
	prep := object(request("GET", "/api/orders/"+prepID, ownerToken, t1, "", nil, 200))
	request("PATCH", "/api/kitchen/"+prepID+"/status", employeeToken, t1, "fixture-kitchen-invalid", map[string]any{"status": "canceled", "expected_version": prep["version"]}, 400)
	prepared := object(request("PATCH", "/api/kitchen/"+prepID+"/status", employeeToken, t1, "fixture-kitchen-start", map[string]any{"status": "preparing", "expected_version": prep["version"]}, 200))
	request("PATCH", "/api/kitchen/"+prepID+"/status", employeeToken, t1, "fixture-kitchen-stale", map[string]any{"status": "ready", "expected_version": prep["version"]}, 404)
	request("PATCH", "/api/kitchen/"+prepID+"/status", employeeToken, t1, "fixture-kitchen-done", map[string]any{"status": "ready", "expected_version": prepared["version"]}, 200)
	link := object(request("POST", "/api/orders/"+prepID+"/link", ownerToken, t1, "fixture-public-link", map[string]any{}, 201))
	capability := strings.TrimPrefix(link["path"].(string), "/order#token=")
	public := request("GET", "/api/public/order", capability, "", "", nil, 200)
	for _, secret := range []string{"shipping_address", "customer_id", "tenant_id", "provider", "password", "@"} {
		if strings.Contains(public.Body.String(), secret) {
			t.Fatalf("public order leaks %s", secret)
		}
	}
	request("GET", "/api/public/order", managerToken, "", "", nil, 404)
	request("POST", "/api/orders/"+prepID+"/link", otherToken, t1, "fixture-forbidden-link", map[string]any{}, 403)
	request("DELETE", "/api/orders/"+prepID+"/link", ownerToken, t1, "fixture-revoke-link", map[string]any{}, 200)
	request("GET", "/api/public/order", capability, "", "", nil, 404)
	link = object(request("POST", "/api/orders/"+prepID+"/link", ownerToken, t1, "fixture-expiring-link", map[string]any{}, 201))
	capability = strings.TrimPrefix(link["path"].(string), "/order#token=")
	must(`UPDATE legui_private.order_links SET created_at=clock_timestamp()-interval '8 days',expires_at=clock_timestamp()-interval '1 day' WHERE digest=?`, capabilityDigest(capability))
	request("GET", "/api/public/order", capability, "", "", nil, 404)
	if err = GenerateNotifications(t.Context(), tx); err != nil {
		t.Fatal(err)
	}
	var before int64
	tx.Raw(`SELECT count(*) FROM legui_private.notifications`).Scan(&before)
	if err = GenerateNotifications(t.Context(), tx); err != nil {
		t.Fatal(err)
	}
	checkNumeric(`SELECT count(*) FROM legui_private.notifications`, int(before))
	must(`INSERT INTO legui_private.notifications(principal_kind,principal_id,event_key,title,message,destination,brand_id,tenant_id) VALUES('user',?,'fixture-forbidden-notification','SECRET_FOREIGN_NOTIFICATION','scope fixture','/inventory',?,?)`, manager, b2, t2)
	inbox := request("GET", "/api/notifications", managerToken, "", "", nil, 200)
	if strings.Contains(inbox.Body.String(), "SECRET_FOREIGN_NOTIFICATION") {
		t.Fatal("old brand notification disclosed")
	}
	request("POST", "/api/notifications/read", managerToken, "", "", map[string]any{}, 200)
	checkNumeric(`SELECT count(*) FROM legui_private.notifications WHERE principal_id=? AND brand_id=? AND read_at IS NULL`, 0, manager, b1)
	checkNumeric(`SELECT count(*) FROM legui_private.notifications WHERE principal_id=? AND brand_id=? AND read_at IS NULL`, 1, manager, b2)
	request("GET", "/api/notifications", employeeToken, "", "", nil, 200)
	must(`UPDATE public.users SET is_active=false WHERE id=?`, manager)
	request("GET", "/api/orders", managerToken, t1, "", nil, 401)
	t.Log("API workflow, replay, tenant isolation, collaborator permissions, stock restoration, receipts, subscriptions and analytics passed; outer transaction will roll back all fixtures")
}
