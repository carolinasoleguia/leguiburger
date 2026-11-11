package mercadopago

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"leguiburger/internal/auth"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	db       *gorm.DB
	cfg      Config
	provider *Provider
}

func NewHandler(database *gorm.DB, cfg Config, p *Provider) *Handler {
	if p == nil {
		p = NewProvider()
	}
	return &Handler{db: database.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}), cfg: cfg, provider: p}
}

type apiError struct {
	status        int
	code, message string
}

func (e *apiError) Error() string { return e.message }
func bad(message string) error    { return &apiError{400, "INVALID_INPUT", message} }
func denied() error {
	return &apiError{403, "FORBIDDEN", "No tenés permiso para operar sobre este recurso"}
}
func missing() error                { return &apiError{404, "NOT_FOUND", "No se encontró el recurso"} }
func conflict(message string) error { return &apiError{409, "CONFLICT", message} }
func notConfigured() error {
	return &apiError{503, "MP_NOT_CONFIGURED", "Mercado Pago todavía no está habilitado para este entorno"}
}
func (h *Handler) Register(mux *http.ServeMux) {
	routes := map[string]func(http.ResponseWriter, *http.Request, auth.SessionPrincipal) error{
		"GET /api/payments/mercadopago/config":                          h.configuration,
		"GET /api/integrations/mercadopago/connections":                 h.connections,
		"POST /api/integrations/mercadopago/connections":                h.createConnection,
		"POST /api/integrations/mercadopago/connections/{id}/authorize": h.authorize,
		"DELETE /api/integrations/mercadopago/connections/{id}":         h.disconnect,
		"POST /api/orders/{id}/checkout":                                h.orderCheckout,
		"POST /api/billing/invoices/{id}/checkout":                      h.invoiceCheckout,
		"POST /api/billing/subscriptions/{id}/checkout":                 h.subscriptionCheckout,
		"POST /api/billing/subscriptions/{id}/cancel-renewal":           h.cancelRenewal,
		"GET /api/payments/mercadopago/checkouts":                       h.listCheckouts,
		"GET /api/payments/mercadopago/transactions":                    h.transactions,
		"GET /api/payments/mercadopago/reviews":                         h.reviews,
		"POST /api/payments/mercadopago/reviews/{id}/resolve":           h.resolveReview,
		"POST /api/payments/mercadopago/transactions/{id}/refunds":      h.refund,
		"POST /api/payments/mercadopago/reconcile":                      h.reconcileHTTP,
		"GET /api/billing/my-account":                                   h.myBilling,
		"GET /api/billing/plans":                                        h.publicPlans,
		"POST /api/billing/subscribe":                                   h.subscribe,
	}
	for route, fn := range routes {
		mux.HandleFunc(route, auth.AuthMiddlewareWithDB(h.db, h.wrap(fn)))
	}
	mux.HandleFunc("GET /api/integrations/mercadopago/callback", h.wrapPublic(h.callback))
	mux.HandleFunc("POST /api/webhooks/mercadopago/{id}", h.wrapPublic(h.webhook))
	mux.HandleFunc("POST /api/webhooks/mercadopago/platform", h.wrapPublic(h.platformWebhook))
}
func (h *Handler) wrap(fn func(http.ResponseWriter, *http.Request, auth.SessionPrincipal) error) http.HandlerFunc {
	return h.wrapPublic(func(w http.ResponseWriter, r *http.Request) error {
		p, ok := auth.PrincipalFromContext(r.Context())
		if !ok {
			return denied()
		}
		return fn(w, r, p)
	})
}
func (h *Handler) wrapPublic(fn func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		if err := fn(w, r.WithContext(ctx)); err != nil {
			var api *apiError
			if errors.As(err, &api) {
				auth.RespondWithError(w, api.status, api.code, api.message)
			} else {
				auth.RespondWithError(w, 503, "MP_UNAVAILABLE", "No se pudo confirmar la operación. Revisá su estado antes de reintentar")
			}
		}
	}
}
func respond(w http.ResponseWriter, status int, data any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
func decode(r *http.Request, v any) error {
	d := json.NewDecoder(io.LimitReader(r.Body, 16384))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return bad("Revisá los datos de la solicitud")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return bad("Se requiere un único objeto JSON")
	}
	return nil
}
func validID(id string) bool { v, e := uuid.Parse(id); return e == nil && v != uuid.Nil }
func routeID(r *http.Request) (string, error) {
	id := r.PathValue("id")
	if !validID(id) {
		return "", bad("Identificador inválido")
	}
	return id, nil
}
func envValid(env string) bool { return env == "sandbox" || env == "production" }
func (h *Handler) enabled(env string) error {
	if !h.cfg.ready() || h.cfg.Environment != env {
		return notConfigured()
	}
	return nil
}
func (h *Handler) limitAction(r *http.Request, scope, actor string, max int) error {
	var attempts int
	e := h.db.WithContext(r.Context()).Raw(`INSERT INTO legui_private.auth_rate_limits(bucket,attempts) VALUES(?,1) ON CONFLICT(bucket) DO UPDATE SET attempts=CASE WHEN auth_rate_limits.window_start<clock_timestamp()-interval '15 minutes' THEN 1 ELSE auth_rate_limits.attempts+1 END,window_start=CASE WHEN auth_rate_limits.window_start<clock_timestamp()-interval '15 minutes' THEN clock_timestamp() ELSE auth_rate_limits.window_start END RETURNING attempts`, "mp:"+scope+":"+hash(actor)).Scan(&attempts).Error
	if e != nil {
		return e
	}
	if attempts > max {
		return &apiError{429, "TOO_MANY_REQUESTS", "Demasiadas operaciones. Esperá unos minutos antes de reintentar"}
	}
	return nil
}
func key(r *http.Request) (string, error) {
	k := r.Header.Get("Idempotency-Key")
	if !operationPattern.MatchString(k) {
		return "", bad("Se requiere Idempotency-Key")
	}
	return k, nil
}
func (h *Handler) scope(r *http.Request, p auth.SessionPrincipal, tenant string) (string, error) {
	if tenant == "" {
		tenant = strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	}
	if tenant == "" {
		tenant = p.TenantID
	}
	if !validID(tenant) {
		return "", bad("Elegí un local")
	}
	var brand string
	result := h.db.WithContext(r.Context()).Raw(`SELECT brand_id::text FROM public.tenants WHERE id=? AND active`, tenant).Scan(&brand)
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected != 1 {
		return "", missing()
	}
	if !p.Platform() && !(p.Manager() && p.BrandID == brand) && !(p.Kind == "employee" && p.TenantID == tenant && p.BrandID == brand) {
		return "", denied()
	}
	return brand, nil
}
func manager(p auth.SessionPrincipal) error {
	if !p.Manager() {
		return denied()
	}
	return nil
}
func (h *Handler) configuration(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if !p.Manager() && p.Role != "cashier" {
		return denied()
	}
	keyReady := false
	if b, e := h.cfg.seal("check", "configuration"); e == nil && b != "" {
		keyReady = true
	}
	respond(w, 200, map[string]any{"enabled": h.cfg.ready(), "environment": h.cfg.Environment, "live_enabled": h.cfg.LiveEnabled, "oauth_configured": h.cfg.ready() && h.cfg.ClientID != "" && h.cfg.ClientSecret != "" && keyReady, "platform_configured": h.cfg.ready() && h.cfg.PlatformToken != "" && h.cfg.PlatformCollector != "", "encryption_configured": keyReady})
	return nil
}

type connection struct {
	ID                string
	Scope             string
	BrandID           string
	TenantID          string
	Environment       string
	State             string
	Version           string
	CredentialVersion string
	CollectorID       string
	CredentialCipher  string
	ExpiresAt         *time.Time
}

const connectionColumns = `id::text,scope,coalesce(brand_id::text,'') brand_id,coalesce(tenant_id::text,'') tenant_id,environment,state,version::text,credential_version::text,coalesce(collector_id,'') collector_id,coalesce(credential_cipher,'') credential_cipher,expires_at`

func (h *Handler) connection(r *http.Request, p auth.SessionPrincipal, id string) (connection, error) {
	var c connection
	result := h.db.WithContext(r.Context()).Raw(`SELECT `+connectionColumns+` FROM legui_private.mp_connections WHERE id=?`, id).Scan(&c)
	if result.Error != nil {
		return c, result.Error
	}
	if result.RowsAffected != 1 {
		return c, missing()
	}
	if !p.Platform() && (c.Scope == "platform" || !p.Manager() || c.BrandID != p.BrandID) {
		return c, denied()
	}
	return c, nil
}
func (h *Handler) connections(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if err := manager(p); err != nil {
		return err
	}
	var rows []map[string]any
	err := h.db.WithContext(r.Context()).Raw(`SELECT id,scope,brand_id,tenant_id,environment,state,collector_id,expires_at,updated_at FROM legui_private.mp_connections WHERE (? OR brand_id=nullif(?,'')::uuid) ORDER BY created_at DESC LIMIT 100`, p.Platform(), p.BrandID).Scan(&rows).Error
	if err != nil {
		return err
	}
	respond(w, 200, map[string]any{"items": rows})
	return nil
}
func (h *Handler) createConnection(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if err := manager(p); err != nil {
		return err
	}
	var input struct {
		BrandID     string `json:"brand_id"`
		TenantID    string `json:"tenant_id"`
		Environment string `json:"environment"`
	}
	if e := decode(r, &input); e != nil {
		return e
	}
	if input.Environment == "" {
		input.Environment = h.cfg.Environment
	}
	if !envValid(input.Environment) {
		return bad("Entorno inválido")
	}
	if input.BrandID == "" {
		input.BrandID = p.BrandID
	}
	if input.TenantID != "" {
		brand, e := h.scope(r, p, input.TenantID)
		if e != nil {
			return e
		}
		if input.BrandID != "" && input.BrandID != brand {
			return denied()
		}
		input.BrandID = brand
	}
	if !validID(input.BrandID) || !p.Platform() && input.BrandID != p.BrandID {
		return denied()
	}
	var id string
	err := h.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := auth.LockSession(r.Context(), tx); e != nil {
			return e
		}
		if e := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "mp-connection:"+input.BrandID+":"+input.TenantID+":"+input.Environment).Error; e != nil {
			return e
		}
		result := tx.Raw(`SELECT id::text FROM legui_private.mp_connections WHERE scope='business' AND brand_id=? AND coalesce(tenant_id::text,'')=? AND environment=?`, input.BrandID, input.TenantID, input.Environment).Scan(&id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return nil
		}
		if e := tx.Raw(`INSERT INTO legui_private.mp_connections(scope,brand_id,tenant_id,environment) VALUES('business',?,nullif(?,'')::uuid,?) RETURNING id::text`, input.BrandID, input.TenantID, input.Environment).Scan(&id).Error; e != nil {
			return e
		}
		return auth.Audit(r.Context(), tx, p, "payments.connection_created", id, nil)
	})
	if err != nil {
		return err
	}
	respond(w, 201, map[string]string{"id": id})
	return nil
}
func (h *Handler) disconnect(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	id, e := routeID(r)
	if e != nil {
		return e
	}
	c, e := h.connection(r, p, id)
	if e != nil {
		return e
	}
	if c.Scope != "business" {
		return denied()
	}
	e = h.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := auth.LockSession(r.Context(), tx); e != nil {
			return e
		}
		if e := tx.Exec(`UPDATE legui_private.mp_connections SET state='disconnected',version=gen_random_uuid(),credential_version=gen_random_uuid(),credential_cipher=NULL,refresh_claim=NULL,refresh_lease_until=NULL,updated_at=clock_timestamp() WHERE id=?`, id).Error; e != nil {
			return e
		}
		if e := tx.Exec(`UPDATE legui_private.mp_oauth_attempts SET status='superseded',verifier_cipher=NULL WHERE connection_id=? AND status IN ('pending','submitting')`, id).Error; e != nil {
			return e
		}
		return auth.Audit(r.Context(), tx, p, "payments.connection_disconnected", id, nil)
	})
	if e != nil {
		return e
	}
	respond(w, 200, map[string]bool{"disconnected": true})
	return nil
}
func page(r *http.Request) (int, int, error) {
	limit, offset := 50, 0
	var e error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, e = strconv.Atoi(raw)
		if e != nil {
			return 0, 0, bad("Paginación inválida")
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, e = strconv.Atoi(raw)
		if e != nil {
			return 0, 0, bad("Paginación inválida")
		}
	}
	if limit < 1 || limit > 100 || offset < 0 || offset > 100000 {
		return 0, 0, bad("Paginación inválida")
	}
	return limit, offset, nil
}
func validEmail(email string) bool {
	a, e := mail.ParseAddress(email)
	return e == nil && a.Address == email && len(email) <= 254
}
