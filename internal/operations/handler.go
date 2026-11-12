package operations

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"io"
	"leguiburger/internal/auth"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type apiError struct {
	Status  int
	Code    string
	Message string
}

func (e *apiError) Error() string   { return e.Message }
func badInput(message string) error { return &apiError{400, "INVALID_INPUT", message} }
func forbidden() error {
	return &apiError{403, "FORBIDDEN", "No tenés permiso para realizar esta operación"}
}
func unauthorized() error {
	return &apiError{401, "UNAUTHORIZED", "La cuenta o sesión no está habilitada"}
}
func notFound() error               { return &apiError{404, "NOT_FOUND", "Recurso no encontrado"} }
func conflict(message string) error { return &apiError{409, "CONFLICT", message} }

type Handler struct {
	repo  *Repository
	buyer BuyerOptions
}
type endpoint func(*http.Request, Principal) (Result, error)

func RegisterRoutes(mux *http.ServeMux, db *gorm.DB, options ...BuyerOptions) {
	h := &Handler{repo: NewRepository(db)}
	if len(options) > 0 {
		h.buyer = options[0]
	}
	mux.HandleFunc("POST /api/public/menu/{id}/buyer-session", h.buyerSession)
	mux.HandleFunc("GET /api/shopper/orders", h.buyerWrap(h.shopperOrders))
	mux.HandleFunc("POST /api/shopper/claim", h.buyerWrap(h.claimOrder))
	mux.HandleFunc("GET /api/shopper/coupons", h.buyerWrap(h.shopperCoupons))
	mux.HandleFunc("GET /api/shopper/preferences", h.buyerWrap(h.shopperPreferences))
	mux.HandleFunc("PATCH /api/shopper/preferences", h.buyerWrap(h.shopperPreferences))
	mux.HandleFunc("GET /api/public/order/events", func(w http.ResponseWriter, r *http.Request) { h.orderStream(w, r, true) })
	mux.HandleFunc("GET /api/public/buyer-order/events", func(w http.ResponseWriter, r *http.Request) { h.orderStream(w, r, false) })
	mux.HandleFunc("GET /api/public/order", h.publicOrder)
	mux.HandleFunc("GET /api/public/restaurants", h.buyerWrap(h.publicRestaurants))
	mux.HandleFunc("GET /api/public/menu/{id}", h.publicMenu)
	mux.HandleFunc("POST /api/public/menu/{id}/quote", h.buyerWrap(h.buyerQuote))
	mux.HandleFunc("POST /api/public/menu/{id}/orders", h.buyerWrap(h.buyerCreate))
	mux.HandleFunc("GET /api/public/buyer-order", h.buyerWrap(h.buyerReceipt))
	routes := map[string]endpoint{
		"GET /api/marketing/customers": h.marketing, "GET /api/marketing/coupons": h.marketing, "POST /api/marketing/coupons": h.marketing, "PATCH /api/marketing/coupons/{id}": h.marketing,
		"GET /api/menu/settings": h.menuSettings, "PATCH /api/menu/settings": h.menuSettings,
		"GET /api/cash/shifts": h.cashShifts, "POST /api/cash/shifts": h.openCash, "GET /api/cash/shifts/{id}": h.cashShift, "GET /api/cash/shifts/{id}/entries": h.cashEntries, "POST /api/cash/shifts/{id}/entries": h.moveCash, "POST /api/cash/shifts/{id}/close": h.closeCash,
		"GET /api/pos/catalog": h.posCatalog, "GET /api/crm/customers": h.customerDirectory,
		"GET /api/kitchen": h.kitchenOrders, "PATCH /api/kitchen/{id}/status": h.kitchenStatus,
		"GET /api/notifications": h.notifications, "POST /api/notifications/read": h.readNotifications,
		"GET /api/onboarding": h.onboarding, "POST /api/onboarding/complete": h.completeOnboarding,
		"POST /api/orders/{id}/link": h.orderLink, "DELETE /api/orders/{id}/link": h.orderLink,
		"GET /api/platform/health":      h.workerHealth,
		"GET /api/analytics/restaurant": h.restaurantAnalytics, "GET /api/analytics/operations": h.operationalAnalytics, "GET /api/analytics/platform": h.platformAnalytics,
		"GET /api/orders": h.listOrders, "POST /api/orders": h.createOrder, "GET /api/orders/{id}": h.getOrder,
		"PATCH /api/orders/{id}/status": h.changeOrderStatus, "POST /api/orders/{id}/payments": h.createOrderPayment, "GET /api/orders/{id}/payments": h.listOrderPayments,
		"GET /api/inventory": h.listInventory, "GET /api/inventory/movements": h.listInventoryMovements, "POST /api/inventory/movements": h.recordInventoryMovement,
		"PATCH /api/inventory/{kind}/{id}/settings": h.updateInventorySettings,
		"GET /api/expenses":                         h.listExpenses, "POST /api/expenses": h.createExpense,
		"POST /api/expenses/{id}/void": h.voidExpense,
		"GET /api/platform/businesses": h.listBusinesses, "PATCH /api/platform/businesses/{id}": h.updateBusiness,
		"GET /api/platform/billing/plans": h.listPlans, "POST /api/platform/billing/plans": h.createPlan, "POST /api/platform/billing/plans/{id}/prices": h.createPlanPrice,
		"GET /api/platform/billing/accounts": h.listBillingAccounts, "POST /api/platform/billing/accounts": h.createBillingAccount,
		"GET /api/platform/billing/subscriptions": h.listSubscriptions, "POST /api/platform/billing/subscriptions": h.createSubscription, "PATCH /api/platform/billing/subscriptions/{id}": h.updateSubscription,
		"GET /api/platform/billing/invoices": h.listInvoices, "POST /api/platform/billing/invoices": h.createInvoice,
		"GET /api/platform/billing/invoices/{id}/payments": h.listBillingPayments, "POST /api/platform/billing/invoices/{id}/payments": h.createBillingPayment,
		"POST /api/platform/billing/invoices/{id}/credits": h.createCreditNote, "GET /api/platform/billing/webhook-events": h.listWebhookEvents,
	}
	for pattern, fn := range routes {
		mux.HandleFunc(pattern, auth.AuthMiddlewareWithDB(h.repo.DB, h.wrap(fn)))
	}
}
func (h *Handler) wrap(fn endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		claims, ok := auth.GetClaimsFromContext(ctx)
		if !ok {
			writeError(w, unauthorized())
			return
		}
		p, err := h.principal(ctx, claims)
		if err != nil {
			writeError(w, err)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/platform/") || r.URL.Path == "/api/analytics/platform" {
			if !p.Platform() {
				writeError(w, forbidden())
				return
			}
		}
		result, err := fn(r, p)
		if err != nil {
			writeError(w, err)
			return
		}
		if result.Replayed {
			w.Header().Set("Idempotent-Replayed", "true")
		}
		if result.Status == 0 {
			result.Status = 200
		}
		w.WriteHeader(result.Status)
		if len(result.Data) == 0 {
			result.Data = json.RawMessage(`{}`)
		}
		_, _ = w.Write(result.Data)
	}
}
func writeError(w http.ResponseWriter, err error) {
	var api *apiError
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		api = &apiError{504, "OPERATION_TIMEOUT", "La operación excedió el tiempo disponible; reintentá con la misma clave"}
	}
	if api == nil && !errors.As(err, &api) {
		var pg *pgconn.PgError
		if errors.As(err, &pg) {
			switch pg.Code {
			case "23505":
				api = &apiError{409, "DUPLICATE", "La operación o el recurso ya existen"}
			case "23503":
				api = &apiError{409, "INVALID_REFERENCE", "Una relación del recurso no es válida para este negocio"}
			case "23514", "23502", "22P02", "22003", "22007":
				api = &apiError{422, "INVALID_OPERATION", "La operación no cumple las reglas de importes, estado o integridad"}
			case "40001", "40P01", "55P03":
				api = &apiError{409, "RETRY_OPERATION", "Hubo un cambio concurrente; reintentá con la misma clave de operación"}
			case "42501":
				api = &apiError{403, "FORBIDDEN", "Operación no permitida"}
			}
			if api == nil {
				log.Printf("operations database failure: code=%s message=%s", pg.Code, pg.Message)
			}
		}
		if api == nil {
			api = &apiError{500, "INTERNAL_ERROR", "No se pudo completar la operación"}
			if pg == nil {
				log.Printf("operations internal failure: %T", err)
			}
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(api.Status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": api.Code, "message": api.Message})
}
func decode[T any](r *http.Request) (T, error) {
	var input T
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return input, badInput("JSON inválido o campos no reconocidos")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return input, badInput("Se requiere un único objeto JSON")
	}
	return input, nil
}
func resourceID(r *http.Request) (string, error) {
	id := r.PathValue("id")
	if !validID(id) {
		return "", badInput("Identificador inválido")
	}
	return id, nil
}
func pagination(r *http.Request) (int, int, error) {
	limit, offset := 50, 0
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, badInput("limit inválido")
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, badInput("offset inválido")
		}
	}
	if limit < 1 || limit > 100 || offset < 0 || offset > 10000 {
		return 0, 0, badInput("limit debe ser 1–100 y offset 0–10000")
	}
	return limit, offset, nil
}
func resultJSON(data json.RawMessage, err error) (Result, error) {
	return Result{Data: data, Status: 200}, err
}
