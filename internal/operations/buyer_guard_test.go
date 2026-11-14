package operations

import (
	"leguiburger/internal/requestguard"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuyerBurstRejectsBeforeDBAndKeepsMenuReadable(t *testing.T) {
	t.Setenv("RENDER", "")
	checkoutBursts = requestguard.New()
	h := &Handler{}
	for i := 0; i < 20; i++ {
		r := httptest.NewRequest("POST", "/api/public/menu/forged/orders", nil)
		guardBuyerRequest(httptest.NewRecorder(), r)
	}
	invoked := false
	wrapped := h.buyerWrap(func(r *http.Request) (Result, error) { invoked = true; return Result{}, nil })
	w := httptest.NewRecorder()
	wrapped(w, httptest.NewRequest("POST", "/api/public/menu/other/orders", nil))
	if w.Code != 429 || invoked || w.Header().Get("Retry-After") == "" {
		t.Fatal("burst reached checkout/DB")
	}
	if !guardBuyerRequest(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/public/menu/any", nil)) {
		t.Fatal("menu was blocked")
	}
	checkoutBursts = requestguard.New()
}
