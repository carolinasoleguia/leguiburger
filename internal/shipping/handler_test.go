package shipping

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

	body := []byte(`{
		"typification": "DELIVERY",
		"name": "Envio Moto Express",
		"description": "Entrega en menos de 45 minutos",
		"cost": 1500.00,
		"estimated_time": "30-45 min"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/shipping-methods", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-ok")

	rr := httptest.NewRecorder()
	handler.HandleShippingRoutes(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("se esperaba status 201 Created, se obtuvo: %d", rr.Code)
	}

	var response models.ShippingMethod
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("error al decodificar la respuesta JSON: %v", err)
	}

	if response.Description != "Entrega en menos de 45 minutos" {
		t.Errorf("se guardo la descripcion incorrectamente: %s", response.Description)
	}
}

func TestHandler_CreateShippingMethod_MissingTenantHeader(t *testing.T) {
	handler := NewHandler(&mockService{})

	req := httptest.NewRequest(http.MethodPost, "/api/shipping-methods", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleShippingRoutes(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("se esperaba status 400 Bad Request por falta de Tenant, se obtuvo: %d", rr.Code)
	}
}
