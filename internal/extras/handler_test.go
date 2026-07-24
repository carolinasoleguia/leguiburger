package extras

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
	createExtraFunc func(ctx context.Context, tenantID, name string, currentPrice float64, currentStock int, trackStock *bool) (*models.Extra, error)
}

func (m *mockService) CreateExtra(ctx context.Context, tenantID, name string, currentPrice float64, currentStock int, trackStock *bool) (*models.Extra, error) {
	return m.createExtraFunc(ctx, tenantID, name, currentPrice, currentStock, trackStock)
}

func (m *mockService) GetExtra(ctx context.Context, tenantID, id string) (*models.Extra, error) {
	return nil, nil
}

func (m *mockService) ListExtras(ctx context.Context, tenantID string) ([]models.Extra, error) {
	return nil, nil
}

func (m *mockService) UpdateExtra(ctx context.Context, tenantID, id, name string, currentPrice *float64, currentStock *int, trackStock, isActive *bool) (*models.Extra, error) {
	return nil, nil
}

func (m *mockService) DeleteExtra(ctx context.Context, tenantID, id string) error {
	return nil
}

func TestHandler_CreateExtra_Success(t *testing.T) {
	mockService := &mockService{
		createExtraFunc: func(ctx context.Context, tenantID, name string, currentPrice float64, currentStock int, trackStock *bool) (*models.Extra, error) {
			return &models.Extra{
				ID:           "new-id",
				TenantID:     tenantID,
				Name:         name,
				CurrentPrice: currentPrice,
				CurrentStock: currentStock,
				TrackStock:   *trackStock,
				IsActive:     true,
			}, nil
		},
	}

	handler := NewHandler(mockService)

	body := []byte(`{
		"name": "Cheddar",
		"current_price": 250.00,
		"current_stock": 10,
		"track_stock": true
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/extras", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-ok")

	rr := httptest.NewRecorder()
	handler.HandleExtraRoutes(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("se esperaba status 201 Created, se obtuvo: %d", rr.Code)
	}

	var response models.Extra
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("error al decodificar la respuesta JSON: %v", err)
	}

	if response.TenantID != "tenant-ok" || response.Name != "Cheddar" || response.CurrentPrice != 250 {
		t.Errorf("se recibio una respuesta incorrecta: %+v", response)
	}
}

func TestHandler_CreateExtra_MissingTenantHeader(t *testing.T) {
	handler := NewHandler(&mockService{})

	req := httptest.NewRequest(http.MethodPost, "/api/extras", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleExtraRoutes(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("se esperaba status 400 Bad Request por falta de Tenant, se obtuvo: %d", rr.Code)
	}
}
