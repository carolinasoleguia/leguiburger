package products

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"leguiburger/internal/models"
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

	body := []byte(`{
		"name": "Doble Cheddar",
		"description": "Burger con doble cheddar",
		"current_price": 4500.00,
		"current_stock": 20,
		"track_stock": true,
		"image_url": "https://example.com/burger.jpg"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-ok")

	rr := httptest.NewRecorder()
	handler.HandleProductRoutes(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("se esperaba status 201 Created, se obtuvo: %d", rr.Code)
	}

	var response models.Product
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("error al decodificar la respuesta JSON: %v", err)
	}

	if response.TenantID != "tenant-ok" || response.Name != "Doble Cheddar" || response.CurrentPrice != 4500 {
		t.Errorf("se recibio una respuesta incorrecta: %+v", response)
	}
}

func TestHandler_CreateProduct_MissingTenantHeader(t *testing.T) {
	handler := NewHandler(&mockService{})

	req := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleProductRoutes(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("se esperaba status 400 Bad Request por falta de Tenant, se obtuvo: %d", rr.Code)
	}
}
