package supplies

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
	createSupplyFunc func(ctx context.Context, tenantID, name string, currentWholesaleCost, currentStock float64, measurementUnit string) (*models.Supply, error)
}

func (m *mockService) CreateSupply(ctx context.Context, tenantID, name string, currentWholesaleCost, currentStock float64, measurementUnit string) (*models.Supply, error) {
	if m.createSupplyFunc != nil {
		return m.createSupplyFunc(ctx, tenantID, name, currentWholesaleCost, currentStock, measurementUnit)
	}
	return nil, nil
}

func (m *mockService) GetSupply(ctx context.Context, tenantID, id string) (*models.Supply, error) {
	return nil, nil
}

func (m *mockService) ListSupplies(ctx context.Context, tenantID string) ([]models.Supply, error) {
	return nil, nil
}

func (m *mockService) UpdateSupply(ctx context.Context, tenantID, id, name string, currentWholesaleCost, currentStock *float64, measurementUnit string, isActive *bool) (*models.Supply, error) {
	return nil, nil
}

func (m *mockService) DeleteSupply(ctx context.Context, tenantID, id string) error {
	return nil
}

func TestHandler_CreateSupply_Success(t *testing.T) {
	mockService := &mockService{
		createSupplyFunc: func(ctx context.Context, tenantID, name string, currentWholesaleCost, currentStock float64, measurementUnit string) (*models.Supply, error) {
			return &models.Supply{
				ID:                   "new-id",
				TenantID:             tenantID,
				Name:                 name,
				CurrentWholesaleCost: currentWholesaleCost,
				CurrentStock:         currentStock,
				MeasurementUnit:      measurementUnit,
				IsActive:             true,
			}, nil
		},
	}

	handler := NewHandler(mockService)

	req := testutil.JSONRequest(t, http.MethodPost, "/api/supplies", map[string]interface{}{
		"name":                    "Pan Brioche",
		"current_wholesale_cost": 100.50,
		"current_stock":          25.75,
		"measurement_unit":       "kg",
	})
	req.Header.Set("X-Tenant-ID", "tenant-ok")
	claims := &auth.Claims{Role: auth.RoleSuperAdmin, TenantID: "tenant-ok"}
	req = testutil.WithClaims(t, req, claims)

	rr := httptest.NewRecorder()
	handler.HandleSupplyRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusCreated)

	var response models.Supply
	testutil.DecodeJSONBody(t, rr, &response)

	if response.TenantID != "tenant-ok" || response.Name != "Pan Brioche" || response.CurrentWholesaleCost != 100.50 {
		t.Errorf("respuesta inesperada: %+v", response)
	}
}

func TestHandler_CreateSupply_MissingTenantHeader(t *testing.T) {
	handler := NewHandler(&mockService{})

	req := testutil.JSONRequest(t, http.MethodPost, "/api/supplies", map[string]interface{}{})

	rr := httptest.NewRecorder()
	handler.HandleSupplyRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusUnauthorized)
}
