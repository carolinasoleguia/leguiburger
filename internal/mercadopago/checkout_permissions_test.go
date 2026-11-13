package mercadopago

import (
	"leguiburger/internal/auth"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRestaurantAdminCannotCollectOrder(t *testing.T) {
	h := &Handler{}
	err := h.orderCheckout(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/orders/forged/checkout", nil), auth.SessionPrincipal{Kind: "user", Role: "admin", BrandID: "fixture"})
	api, ok := err.(*apiError)
	if !ok || api.status != http.StatusForbidden {
		t.Fatalf("wanted forbidden before reading order/provider, got %v", err)
	}
}
