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
	createProductFunc func(ctx context.Context, brandID, name, description string, basePrice float64, imageURL string) (*models.Product, error)
}

func (m *mockService) CreateProduct(ctx context.Context, brandID, name, description string, basePrice float64, imageURL string) (*models.Product, error) {
	return m.createProductFunc(ctx, brandID, name, description, basePrice, imageURL)
}

func (m *mockService) GetProduct(ctx context.Context, brandID, id string) (*models.Product, error) {
	return nil, nil
}

func (m *mockService) ListProducts(ctx context.Context, brandID string) ([]models.Product, error) {
	return nil, nil
}

func (m *mockService) UpdateProduct(ctx context.Context, brandID, id, name, description string, basePrice *float64, imageURL string, isActive *bool) (*models.Product, error) {
	return nil, nil
}

func (m *mockService) DeleteProduct(ctx context.Context, brandID, id string) error {
	return nil
}

func TestHandler_CreateProduct_Success(t *testing.T) {
	brandID := "brand-ok"
	mockService := &mockService{
		createProductFunc: func(ctx context.Context, receivedBrandID, name, description string, basePrice float64, imageURL string) (*models.Product, error) {
			if receivedBrandID != brandID {
				t.Fatalf("se esperaba brand %s, se obtuvo %s", brandID, receivedBrandID)
			}
			return &models.Product{
				ID:          "new-id",
				BrandID:     receivedBrandID,
				Name:        name,
				Description: description,
				BasePrice:   basePrice,
				ImageURL:    imageURL,
				IsActive:    true,
			}, nil
		},
	}

	handler := NewHandler(mockService)

	req := testutil.JSONRequest(t, http.MethodPost, "/api/products", map[string]interface{}{
		"name":        "Doble Cheddar",
		"description": "Burger con doble cheddar",
		"base_price":  4500.00,
		"image_url":   "https://example.com/burger.jpg",
	})
	req = testutil.WithClaims(t, req, &auth.Claims{Role: "admin", BrandID: &brandID})

	rr := httptest.NewRecorder()
	handler.HandleProductRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusCreated)

	var response models.Product
	testutil.DecodeJSONBody(t, rr, &response)

	if response.BrandID != brandID || response.Name != "Doble Cheddar" || response.BasePrice != 4500 {
		t.Errorf("se recibio una respuesta incorrecta: %+v", response)
	}
}

func TestHandler_CreateProduct_MissingBrand(t *testing.T) {
	handler := NewHandler(&mockService{})

	req := testutil.JSONRequest(t, http.MethodPost, "/api/products", map[string]interface{}{})
	req = testutil.WithClaims(t, req, &auth.Claims{Role: auth.RoleOwner})

	rr := httptest.NewRecorder()
	handler.HandleProductRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusBadRequest)
}
