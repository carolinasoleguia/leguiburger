package recipes

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

	req := testutil.JSONRequest(t, http.MethodPost, "/api/recipes", map[string]interface{}{
		"product_id":    "product-1",
		"supply_id":     "supply-1",
		"quantity_used": 0.250,
	})
	req.Header.Set("X-Tenant-ID", "tenant-ok")
	claims := &auth.Claims{Role: auth.RoleSuperAdmin, TenantID: "tenant-ok"}
	req = testutil.WithClaims(t, req, claims)

	rr := httptest.NewRecorder()
	handler.HandleRecipeRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusCreated)

	var response models.Recipe
	testutil.DecodeJSONBody(t, rr, &response)

	if response.ProductID != "product-1" || response.SupplyID != "supply-1" || response.QuantityUsed != 0.250 {
		t.Errorf("se recibio una respuesta incorrecta: %+v", response)
	}
}

func TestHandler_CreateRecipe_MissingTenantHeader(t *testing.T) {
	handler := NewHandler(&mockService{})

	req := testutil.JSONRequest(t, http.MethodPost, "/api/recipes", map[string]interface{}{})

	rr := httptest.NewRecorder()
	handler.HandleRecipeRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusUnauthorized)
}
