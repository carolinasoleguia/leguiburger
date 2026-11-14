package operations

import (
	"context"
	"fmt"
	"leguiburger/internal/auth"
	"leguiburger/internal/buyeraccess"
	"net/http"
	"strings"
	"time"
)

// No query credentials: fetch streaming carries the existing capability header.
func (h *Handler) orderStream(w http.ResponseWriter, r *http.Request, tracking bool) {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		writeError(w, forbidden())
		return
	}
	hub := h.buyer.Stream
	if hub == nil || !hub.Ready() {
		w.Header().Set("Retry-After", "5")
		http.Error(w, "stream unavailable", 503)
		return
	}
	raw := buyeraccess.Token(r)
	if !buyeraccess.ValidToken(raw) && r.Header.Get("X-Shopper-Order") == "" {
		writeError(w, notFound())
		return
	}
	resolve := func() (string, error) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if !tracking {
			grant, e := buyeraccess.ResolveRequest(ctx, h.repo.DB, r, false)
			if e == nil && r.Header.Get("Authorization") == "" && r.Header.Get("X-Shopper-Order") == "" && grant.TenantID != r.Header.Get("X-Buyer-Tenant") {
				return "", notFound()
			}
			return grant.OrderID, e
		}
		var id string
		q := h.repo.DB.WithContext(ctx).Raw(`SELECT l.order_id::text FROM legui_private.order_links l JOIN public.orders o ON o.id=l.order_id AND o.tenant_id=l.tenant_id JOIN public.tenants t ON t.id=o.tenant_id AND t.active JOIN public.brands b ON b.id=t.brand_id AND b.suspended_at IS NULL AND b.closed_at IS NULL WHERE l.digest=? AND l.expires_at>clock_timestamp() AND l.revoked_at IS NULL`, buyeraccess.Digest(raw)).Scan(&id)
		if q.Error != nil {
			return "", q.Error
		}
		if q.RowsAffected != 1 {
			return "", notFound()
		}
		return id, nil
	}
	id, e := resolve()
	if e != nil {
		writeError(w, notFound())
		return
	}
	streamKey := buyeraccess.Digest(raw)
	if r.Header.Get("X-Shopper-Order") != "" {
		if c, e := r.Cookie("legui_shopper"); e == nil {
			streamKey = buyeraccess.Digest(c.Value)
		}
	}
	h.serveOrderEvents(w, r, id, streamKey, resolve)
}
func (h *Handler) staffOrderStream(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		writeError(w, forbidden())
		return
	}
	raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	resolve := func() (string, error) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		claims, e := auth.ValidateToken(raw)
		if e != nil {
			return "", unauthorized()
		}
		session, e := auth.ResolveSessionPrincipal(ctx, h.repo.DB, claims)
		if e != nil {
			return "", unauthorized()
		}
		p := Principal{ID: session.ID, Kind: session.Kind, Role: session.Role, BrandID: session.BrandID, TenantID: session.TenantID}
		if !p.Operator() {
			return "", forbidden()
		}
		scope, e := h.tenantScope(ctx, p, r)
		if e != nil {
			return "", e
		}
		var available bool
		e = h.repo.DB.WithContext(ctx).Raw(`SELECT EXISTS(SELECT 1 FROM public.brands WHERE id=? AND suspended_at IS NULL AND closed_at IS NULL)`, scope.BrandID).Scan(&available).Error
		if e != nil {
			return "", e
		}
		if !available {
			return "", forbidden()
		}
		if !p.Platform() {
			a, err := auth.ReadMembership(ctx, h.repo.DB, scope.BrandID)
			if err != nil {
				return "", err
			}
			if !a.Operational {
				return "", forbidden()
			}
		}
		return "tenant:" + scope.TenantID, nil
	}
	id, e := resolve()
	if e != nil {
		writeError(w, e)
		return
	}
	h.serveOrderEvents(w, r, id, "staff:"+buyeraccess.Digest(raw), resolve)
}
func (h *Handler) serveOrderEvents(w http.ResponseWriter, r *http.Request, id, key string, resolve func() (string, error)) {
	hub := h.buyer.Stream
	if hub == nil || !hub.Ready() {
		w.Header().Set("Retry-After", "5")
		http.Error(w, "stream unavailable", 503)
		return
	}
	changed, release, ok := hub.Subscribe(id, key)
	if !ok {
		http.Error(w, "stream capacity unavailable", 503)
		return
	}
	defer release()
	rc := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	send := func(event string) bool {
		if rc.SetWriteDeadline(time.Now().Add(10*time.Second)) != nil {
			return false
		}
		if _, e := fmt.Fprintf(w, "event: %s\ndata: {}\n\n", event); e != nil {
			return false
		}
		return rc.Flush() == nil
	}
	if !send("ready") {
		return
	}
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	authCheck := time.NewTicker(45 * time.Second)
	defer authCheck.Stop()
	lifetime := time.NewTimer(2 * time.Minute)
	defer lifetime.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-lifetime.C:
			return
		case <-changed:
			if !hub.Ready() || !send("changed") {
				return
			}
		case <-heartbeat.C:
			if !hub.Ready() || !send("heartbeat") {
				return
			}
		case <-authCheck.C:
			if current, e := resolve(); e != nil || current != id {
				return
			}
		}
	}
}
