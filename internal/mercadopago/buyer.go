package mercadopago

import (
	"context"
	"leguiburger/internal/auth"
	"leguiburger/internal/buyeraccess"
	"net/http"
	"strings"
)

type buyerGrantKey struct{}

// BuyerPaymentAvailable reveals a boolean, never account IDs or credentials.
func (h *Handler) BuyerPaymentAvailable(ctx context.Context, tenant string) (bool, error) {
	if h.cfg.Environment != "production" || !h.cfg.ready() {
		return false, nil
	}
	if _, e := h.cfg.seal("readiness", "buyer-checkout"); e != nil {
		return false, nil
	}
	var ready bool
	e := h.db.WithContext(ctx).Raw(`SELECT EXISTS(SELECT 1 FROM public.tenants t JOIN public.brands b ON b.id=t.brand_id JOIN legui_private.mp_connections c ON c.brand_id=t.brand_id AND (c.tenant_id=t.id OR c.tenant_id IS NULL) WHERE t.id=? AND t.active AND b.suspended_at IS NULL AND b.closed_at IS NULL AND c.scope='business' AND c.state='connected' AND c.environment='production' AND c.credential_cipher IS NOT NULL)`, tenant).Scan(&ready).Error
	return ready, e
}
func (h *Handler) buyerOrderCheckout(w http.ResponseWriter, r *http.Request) error {
	if h.cfg.Environment != "production" {
		return conflict("El pago online no está disponible para este local")
	}
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return denied()
	}
	grant, e := buyeraccess.ResolveRequest(r.Context(), h.db, r, false)
	if e == nil && r.Header.Get("Authorization") == "" && r.Header.Get("X-Shopper-Order") == "" && grant.TenantID != r.Header.Get("X-Buyer-Tenant") {
		return missing()
	}
	if e != nil {
		return missing()
	}
	if e = buyeraccess.Rate(r.Context(), h.db, r, "payment:"+grant.TenantID, 60); e != nil {
		return &apiError{429, "TOO_MANY_REQUESTS", "Esperá unos minutos antes de reintentar el pago"}
	}
	r.SetPathValue("id", grant.OrderID)
	r.Header.Set("X-Tenant-ID", grant.TenantID)
	r = r.WithContext(context.WithValue(r.Context(), buyerGrantKey{}, grant))
	return h.beginCheckout(w, r, auth.SessionPrincipal{ID: grant.OrderID, Kind: "buyer", Role: "buyer", BrandID: grant.BrandID, TenantID: grant.TenantID}, "order")
}
func (h *Handler) lockBuyer(r *http.Request, p auth.SessionPrincipal) error {
	grant, ok := r.Context().Value(buyerGrantKey{}).(buyeraccess.Grant)
	if !ok || p.Kind != "buyer" || p.ID != grant.OrderID || p.BrandID != grant.BrandID || p.TenantID != grant.TenantID || strings.TrimSpace(r.Header.Get("X-Tenant-ID")) != grant.TenantID {
		return denied()
	}
	return nil
}
