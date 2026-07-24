package shipping

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
	createMethodFunc func(ctx context.Context, tenantID, name, typification, description string, cost float64, estTime string) (*models.ShippingMethod, error)
}

func (m *mockService) CreateMethod(ctx context.Context, tenantID, name, typification, description string, cost float64, estTime string) (*models.ShippingMethod, error) {
	return m.createMethodFunc(ctx, tenantID, name, typification, description, cost, estTime)
}
func (m *mockService) GetMethod(ctx context.Context, tenantID, id string) (*models.ShippingMethod, error) {
	return nil, nil
}
func (m *mockService) ListMethods(ctx context.Context, tenantID string) ([]models.ShippingMethod, error) {
	return nil, nil
}
func (m *mockService) UpdateMethod(ctx context.Context, tenantID, id string, name, typification, description string, cost *float64, estTime string, active *bool) (*models.ShippingMethod, error) {
	return nil, nil
}
func (m *mockService) DeleteMethod(ctx context.Context, tenantID, id string) error {
	return nil
}

func TestHandler_CreateShippingMethod_Success(t *testing.T) {
	mockService := &mockService{
		createMethodFunc: func(ctx context.Context, tenantID, name, typification, description string, cost float64, estTime string) (*models.ShippingMethod, error) {
			if description != "Entrega en menos de 45 minutos" {
				t.Errorf("la descripcion se deformo, recibida: %s", description)
			}
			if typification != "DELIVERY" {
				t.Errorf("la tipificacion se recibio mal: %s", typification)
			}
			return &models.ShippingMethod{
				ID:            "new-id",
				TenantID:      tenantID,
				Name:          name,
				Typification:  typification,
				Description:   description,
				Cost:          cost,
				EstimatedTime: estTime,
			}, nil
		},
	}

	handler := NewHandler(mockService)

	req := testutil.JSONRequest(t, http.MethodPost, "/api/shipping-methods", map[string]interface{}{
		"typification":   "DELIVERY",
		"name":           "Envio Moto Express",
		"description":    "Entrega en menos de 45 minutos",
		"cost":           1500.00,
		"estimated_time": "30-45 min",
	})
	req.Header.Set("X-Tenant-ID", "tenant-ok")
	claims := &auth.Claims{Role: auth.RoleSuperAdmin, TenantID: "tenant-ok"}
	req = testutil.WithClaims(t, req, claims)

	rr := httptest.NewRecorder()
	handler.HandleShippingRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusCreated)

	var response models.ShippingMethod
	testutil.DecodeJSONBody(t, rr, &response)

	if response.Description != "Entrega en menos de 45 minutos" {
		t.Errorf("se guardo la descripcion incorrectamente: %s", response.Description)
	}
}

func TestHandler_CreateShippingMethod_MissingTenantHeader(t *testing.T) {
	handler := NewHandler(&mockService{})

	req := testutil.JSONRequest(t, http.MethodPost, "/api/shipping-methods", map[string]interface{}{})

	rr := httptest.NewRecorder()
	handler.HandleShippingRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusUnauthorized)
}
