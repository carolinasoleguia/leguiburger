package recipes

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
	createRecipeFunc func(ctx context.Context, tenantID, productID, supplyID string, quantityUsed float64) (*models.Recipe, error)
}

func (m *mockService) CreateRecipe(ctx context.Context, tenantID, productID, supplyID string, quantityUsed float64) (*models.Recipe, error) {
	return m.createRecipeFunc(ctx, tenantID, productID, supplyID, quantityUsed)
}

func (m *mockService) GetRecipe(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error) {
	return nil, nil
}

func (m *mockService) ListRecipes(ctx context.Context, tenantID string) ([]models.Recipe, error) {
	return nil, nil
}

func (m *mockService) UpdateRecipe(ctx context.Context, tenantID, productID, supplyID string, quantityUsed float64) (*models.Recipe, error) {
	return nil, nil
}

func (m *mockService) DeleteRecipe(ctx context.Context, tenantID, productID, supplyID string) error {
	return nil
}

func TestHandler_CreateRecipe_Success(t *testing.T) {
	mockService := &mockService{
		createRecipeFunc: func(ctx context.Context, tenantID, productID, supplyID string, quantityUsed float64) (*models.Recipe, error) {
			return &models.Recipe{
				ProductID:    productID,
				SupplyID:     supplyID,
				QuantityUsed: quantityUsed,
			}, nil
		},
	}

	handler := NewHandler(mockService)

	body := []byte(`{
		"product_id": "product-1",
		"supply_id": "supply-1",
		"quantity_used": 0.250
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/recipes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-ok")

	rr := httptest.NewRecorder()
	handler.HandleRecipeRoutes(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("se esperaba status 201 Created, se obtuvo: %d", rr.Code)
	}

	var response models.Recipe
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("error al decodificar la respuesta JSON: %v", err)
	}

	if response.ProductID != "product-1" || response.SupplyID != "supply-1" || response.QuantityUsed != 0.250 {
		t.Errorf("se recibio una respuesta incorrecta: %+v", response)
	}
}

func TestHandler_CreateRecipe_MissingTenantHeader(t *testing.T) {
	handler := NewHandler(&mockService{})

	req := httptest.NewRequest(http.MethodPost, "/api/recipes", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleRecipeRoutes(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("se esperaba status 400 Bad Request por falta de Tenant, se obtuvo: %d", rr.Code)
	}
}
