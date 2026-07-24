package supplies

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

	body := []byte(`{
		"name": "Pan Brioche",
		"current_wholesale_cost": 100.50,
		"current_stock": 25.75,
		"measurement_unit": "kg"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/supplies", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-ok")

	rr := httptest.NewRecorder()
	handler.HandleSupplyRoutes(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("se esperaba status 201 Created, se obtuvo: %d", rr.Code)
	}

	var response models.Supply
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("error al decodificar la respuesta JSON: %v", err)
	}

	if response.TenantID != "tenant-ok" || response.Name != "Pan Brioche" || response.CurrentWholesaleCost != 100.50 {
		t.Errorf("respuesta inesperada: %+v", response)
	}
}

func TestHandler_CreateSupply_MissingTenantHeader(t *testing.T) {
	handler := NewHandler(&mockService{})

	req := httptest.NewRequest(http.MethodPost, "/api/supplies", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleSupplyRoutes(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("se esperaba status 400 Bad Request por falta de Tenant, se obtuvo: %d", rr.Code)
	}
}
