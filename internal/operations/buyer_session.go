package operations

import (
	"context"
	"encoding/json"
	"leguiburger/internal/buyeraccess"
	"net/http"
	"time"
)

// A browser cookie is distinct from staff auth and carries no contact information.
func (h *Handler) buyerSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	if !buyeraccess.SameOrigin(r) {
		writeError(w, forbidden())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	if !h.buyerGate(w, r) {
		return
	}
	if h.security != nil {
		if e := h.security.blockCheck(ctx, r); e != nil {
			writeError(w, e)
			return
		}
	}
	tenant := r.PathValue("id")
	if !validID(tenant) {
		writeError(w, notFound())
		return
	}
	var exists bool
	if e := h.repo.DB.WithContext(ctx).Raw(`SELECT EXISTS(SELECT 1 FROM public.tenants t JOIN public.brands b ON b.id=t.brand_id WHERE t.id=? AND t.active AND b.suspended_at IS NULL AND b.closed_at IS NULL)`, tenant).Scan(&exists).Error; e != nil {
		writeError(w, e)
		return
	}
	if !exists {
		writeError(w, notFound())
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	input, e := decode[struct {
		LegacyKey string `json:"legacy_key,omitempty"`
		New       bool   `json:"new,omitempty"`
	}](r)
	if e != nil {
		writeError(w, e)
		return
	}
	session, ok := buyeraccess.CookieSession(r, tenant)
	set := !ok
	if input.New && ok {
		grant, e := buyeraccess.Resolve(ctx, h.repo.DB, session.Secret, false)
		if e != nil {
			writeError(w, notFound())
			return
		}
		var terminal bool
		e = h.repo.DB.WithContext(ctx).Raw(`SELECT (s.code IN ('canceled','rejected') OR (s.code='delivered' AND o.total_paid>=o.total_due)) AND NOT EXISTS(SELECT 1 FROM legui_private.mp_checkouts x WHERE x.order_id=o.id AND x.state IN ('prepared','submitting','review')) FROM public.orders o JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE o.id=? AND o.tenant_id=?`, grant.OrderID, tenant).Scan(&terminal).Error
		if e != nil {
			writeError(w, e)
			return
		}
		if !terminal {
			writeError(w, conflict("Ya tenés un pedido en curso. Esperá a recibirlo antes de hacer otro"))
			return
		}
		set = true
		ok = false
	}
	if !ok {
		session = buyeraccess.NewBrowserSession()
		if raw := r.Header.Get("Authorization"); raw != "" {
			secret := buyeraccess.Token(r)
			if !buyeraccess.ValidToken(secret) || !validID(input.LegacyKey) {
				writeError(w, badInput("Confirmación anterior inválida"))
				return
			}
			var prior struct {
				TenantID, Key string
				Active        bool
			}
			q := h.repo.DB.WithContext(ctx).Raw(`SELECT tenant_id::text,creation_key::text key,expires_at>clock_timestamp() active FROM legui_private.buyer_orders WHERE access_digest=?`, buyeraccess.Digest(secret)).Scan(&prior)
			if q.Error != nil {
				writeError(w, q.Error)
				return
			}
			if q.RowsAffected > 0 && (prior.TenantID != tenant || prior.Key != input.LegacyKey || !prior.Active) {
				writeError(w, notFound())
				return
			}
			session.Secret = secret
			session.Key = input.LegacyKey
		}
	}
	if set {
		if h.security != nil {
			if e := h.security.observe(ctx, r, tenant, "session", ""); e != nil {
				writeError(w, e)
				return
			}
		}
		buyeraccess.SetCookie(w, tenant, session)
	}
	json.NewEncoder(w).Encode(map[string]string{"creation_key": session.Key, "payment_key": buyeraccess.PaymentKey(session)})
}
