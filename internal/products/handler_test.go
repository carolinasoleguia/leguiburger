package products

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
	createProductFunc func(ctx context.Context, tenantID, name, description string, currentPrice float64, currentStock int, trackStock *bool, imageURL string) (*models.Product, error)
}

func (m *mockService) CreateProduct(ctx context.Context, tenantID, name, description string, currentPrice float64, currentStock int, trackStock *bool, imageURL string) (*models.Product, error) {
	return m.createProductFunc(ctx, tenantID, name, description, currentPrice, currentStock, trackStock, imageURL)
}

func (m *mockService) GetProduct(ctx context.Context, tenantID, id string) (*models.Product, error) {
	return nil, nil
}

func (m *mockService) ListProducts(ctx context.Context, tenantID string) ([]models.Product, error) {
	return nil, nil
}

func (m *mockService) UpdateProduct(ctx context.Context, tenantID, id, name, description string, currentPrice *float64, currentStock *int, trackStock *bool, imageURL string, isActive *bool) (*models.Product, error) {
	return nil, nil
}

func (m *mockService) DeleteProduct(ctx context.Context, tenantID, id string) error {
	return nil
}

func TestHandler_CreateProduct_Success(t *testing.T) {
	mockService := &mockService{
		createProductFunc: func(ctx context.Context, tenantID, name, description string, currentPrice float64, currentStock int, trackStock *bool, imageURL string) (*models.Product, error) {
			return &models.Product{
				ID:           "new-id",
				TenantID:     tenantID,
				Name:         name,
				Description:  description,
				CurrentPrice: currentPrice,
				CurrentStock: currentStock,
				TrackStock:   *trackStock,
				ImageURL:     imageURL,
				IsActive:     true,
			}, nil
		},
	}

	handler := NewHandler(mockService)

	req := testutil.JSONRequest(t, http.MethodPost, "/api/products", map[string]interface{}{
		"name":          "Doble Cheddar",
		"description":   "Burger con doble cheddar",
		"current_price": 4500.00,
		"current_stock": 20,
		"track_stock":   true,
		"image_url":     "https://example.com/burger.jpg",
	})
	req.Header.Set("X-Tenant-ID", "tenant-ok")
	claims := &auth.Claims{Role: auth.RoleSuperAdmin, TenantID: "tenant-ok"}
	req = testutil.WithClaims(t, req, claims)

	rr := httptest.NewRecorder()
	handler.HandleProductRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusCreated)

	var response models.Product
	testutil.DecodeJSONBody(t, rr, &response)

	if response.TenantID != "tenant-ok" || response.Name != "Doble Cheddar" || response.CurrentPrice != 4500 {
		t.Errorf("se recibio una respuesta incorrecta: %+v", response)
	}
}

func TestHandler_CreateProduct_MissingTenantHeader(t *testing.T) {
	handler := NewHandler(&mockService{})

	req := testutil.JSONRequest(t, http.MethodPost, "/api/products", map[string]interface{}{})

	rr := httptest.NewRecorder()
	handler.HandleProductRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusUnauthorized)
}
