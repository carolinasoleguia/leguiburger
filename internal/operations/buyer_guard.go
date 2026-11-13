package operations

import (
	"errors"
	"leguiburger/internal/requestguard"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var checkoutBursts = requestguard.New()

func buyerWait(r *http.Request) time.Duration {
	if r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, "/api/public/menu/") {
		return 0
	}
	limit := 120
	if strings.HasSuffix(r.URL.Path, "/orders") {
		limit = 20
	}
	if strings.HasSuffix(r.URL.Path, "/buyer-session") {
		limit = 60
	}
	action := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	return checkoutBursts.Allow(action+":"+requestguard.ClientIP(r), limit, time.Now())
}
func denyBuyerBurst(w http.ResponseWriter, wait time.Duration) bool {
	if wait <= 0 {
		return true
	}
	w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
	writeError(w, &apiError{429, "CHECKOUT_RATE_LIMITED", "Detectamos demasiados intentos de compra. Esperá unos minutos y volvé a intentar; podés seguir consultando la carta."})
	return false
}
func guardBuyerRequest(w http.ResponseWriter, r *http.Request) bool {
	return denyBuyerBurst(w, buyerWait(r))
}

// The release lookup lets an owner undo a false positive across all API replicas.
// Blocked requests perform only an indexed read, never expensive checkout writes.
func (h *Handler) buyerGate(w http.ResponseWriter, r *http.Request) bool {
	wait := buyerWait(r)
	if wait <= 0 {
		return true
	}
	if h.security != nil {
		st, e := h.security.state(r.Context(), r)
		if e == nil && st.ReleasedUntil != nil && st.ReleasedUntil.After(st.now()) && blockedRisk(st) == nil {
			return true
		}
		if wait == 5*time.Minute {
			tenant := r.PathValue("id")
			if !validID(tenant) {
				tenant = ""
			}
			if e := h.security.observe(r.Context(), r, tenant, "burst", "rate_limit"); e != nil {
				log.Printf("abuse incident unavailable (%T)", e)
			}
		}
	}
	return denyBuyerBurst(w, wait)
}
func (h *Handler) observeBuyerFailure(r *http.Request, e error) {
	if h.security == nil || r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, "/api/public/menu/") || strings.HasSuffix(r.URL.Path, "/human-check") {
		return
	}
	var api *apiError
	if errors.As(e, &api) && (api.Status == 400 || api.Status == 404 || api.Status == 403 && api.Code == "FORBIDDEN") {
		tenant := r.PathValue("id")
		if !validID(tenant) {
			tenant = ""
		}
		if err := h.security.observe(r.Context(), r, tenant, "invalid", ""); err != nil {
			log.Printf("abuse signal unavailable (%T)", err)
		}
	}
}
