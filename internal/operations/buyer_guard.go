package operations

import (
	"leguiburger/internal/requestguard"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var checkoutBursts = requestguard.New()

func guardBuyerRequest(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, "/api/public/menu/") {
		return true
	}
	limit := 120
	if strings.HasSuffix(r.URL.Path, "/orders") {
		limit = 20
	}
	if strings.HasSuffix(r.URL.Path, "/buyer-session") {
		limit = 60
	}
	// Global by operation and IP: changing the requested local cannot reset a burst.
	action := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	wait := checkoutBursts.Allow(action+":"+requestguard.ClientIP(r), limit, time.Now())
	if wait <= 0 {
		return true
	}
	w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
	writeError(w, &apiError{429, "CHECKOUT_RATE_LIMITED", "Detectamos demasiados intentos de compra. El acceso a pedidos está pausado temporalmente. Esperá unos minutos y volvé a intentar; podés seguir consultando la carta."})
	return false
}
