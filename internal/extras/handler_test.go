package extras

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

	req := testutil.JSONRequest(t, http.MethodPost, "/api/extras", map[string]interface{}{
		"name":          "Cheddar",
		"current_price": 250.00,
		"current_stock": 10,
		"track_stock":   true,
	})
	req.Header.Set("X-Tenant-ID", "tenant-ok")
	claims := &auth.Claims{Role: auth.RoleSuperAdmin, TenantID: "tenant-ok"}
	req = testutil.WithClaims(t, req, claims)

	rr := httptest.NewRecorder()
	handler.HandleExtraRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusCreated)

	var response models.Extra
	testutil.DecodeJSONBody(t, rr, &response)

	if response.TenantID != "tenant-ok" || response.Name != "Cheddar" || response.CurrentPrice != 250 {
		t.Errorf("se recibio una respuesta incorrecta: %+v", response)
	}
}

func TestHandler_CreateExtra_MissingTenantHeader(t *testing.T) {
	handler := NewHandler(&mockService{})

	req := testutil.JSONRequest(t, http.MethodPost, "/api/extras", map[string]interface{}{})

	rr := httptest.NewRecorder()
	handler.HandleExtraRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusUnauthorized)
}
