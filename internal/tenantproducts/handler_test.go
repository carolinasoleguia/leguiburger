package tenantproducts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"leguiburger/internal/auth"
	"leguiburger/internal/models"
	"leguiburger/internal/testutil"
)

type mockService struct {
	createTenantProductFunc func(ctx context.Context, tenantID, productID string, priceOverride *float64, currentStock int, trackStock, isAvailable *bool) (*models.TenantProduct, error)
}

func (m *mockService) CreateTenantProduct(ctx context.Context, tenantID, productID string, priceOverride *float64, currentStock int, trackStock, isAvailable *bool) (*models.TenantProduct, error) {
	return m.createTenantProductFunc(ctx, tenantID, productID, priceOverride, currentStock, trackStock, isAvailable)
}
func (m *mockService) GetTenantProduct(ctx context.Context, tenantID, productID string) (*models.TenantProduct, error) {
	return nil, nil
}
func (m *mockService) ListTenantProducts(ctx context.Context, tenantID string) ([]models.TenantProduct, error) {
	return nil, nil
}
func (m *mockService) UpdateTenantProduct(ctx context.Context, tenantID, productID string, priceOverride *float64, currentStock *int, trackStock, isAvailable, isActive *bool) (*models.TenantProduct, error) {
	return nil, nil
}
func (m *mockService) DeleteTenantProduct(ctx context.Context, tenantID, productID string) error {
	return nil
}

func TestHandler_CreateTenantProduct_Success(t *testing.T) {
	mockService := &mockService{
		createTenantProductFunc: func(ctx context.Context, tenantID, productID string, priceOverride *float64, currentStock int, trackStock, isAvailable *bool) (*models.TenantProduct, error) {
			return &models.TenantProduct{
				TenantID:      tenantID,
				ProductID:     productID,
				PriceOverride: priceOverride,
				CurrentStock:  currentStock,
				TrackStock:    *trackStock,
				IsAvailable:   *isAvailable,
				IsActive:      true,
			}, nil
		},
	}

	handler := NewHandler(mockService)
	price := 4800.0
	req := testutil.JSONRequest(t, http.MethodPost, "/api/tenant-products", map[string]interface{}{
		"product_id":     "product-1",
		"price_override": price,
		"current_stock":  10,
		"track_stock":    true,
		"is_available":   true,
	})
	req.Header.Set("X-Tenant-ID", "tenant-ok")
	req = testutil.WithClaims(t, req, &auth.Claims{Role: auth.RoleSuperAdmin, TenantID: "tenant-ok"})

	rr := httptest.NewRecorder()
	handler.HandleTenantProductRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusCreated)

	var response models.TenantProduct
	testutil.DecodeJSONBody(t, rr, &response)

	if response.TenantID != "tenant-ok" || response.ProductID != "product-1" || response.CurrentStock != 10 {
		t.Errorf("respuesta incorrecta: %+v", response)
	}
}

func TestHandler_CreateTenantProduct_MissingTenant(t *testing.T) {
	handler := NewHandler(&mockService{})

	req := testutil.JSONRequest(t, http.MethodPost, "/api/tenant-products", map[string]interface{}{})
	req = testutil.WithClaims(t, req, &auth.Claims{Role: auth.RoleOwner})

	rr := httptest.NewRecorder()
	handler.HandleTenantProductRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusBadRequest)
}
