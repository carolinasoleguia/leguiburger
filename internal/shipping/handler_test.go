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

// Mock de Servicio para el Handler
type MockService struct {
	OnCreateMethod func(ctx context.Context, tenantID, name, typification, description string, cost float64, estTime string) (*models.ShippingMethod, error)
}

func (m *MockService) CreateMethod(ctx context.Context, tenantID, name, typification, description string, cost float64, estTime string) (*models.ShippingMethod, error) {
	return m.OnCreateMethod(ctx, tenantID, name, typification, description, cost, estTime)
}
func (m *MockService) GetMethod(ctx context.Context, tenantID, id string) (*models.ShippingMethod, error) {
	return nil, nil
}
func (m *MockService) ListMethods(ctx context.Context, tenantID string) ([]models.ShippingMethod, error) {
	return nil, nil
}
func (m *MockService) UpdateMethod(ctx context.Context, tenantID, id string, name, typification, description string, cost *float64, estTime string, active *bool) (*models.ShippingMethod, error) {
	return nil, nil
}
func (m *MockService) DeleteMethod(ctx context.Context, tenantID, id string) error {
	return nil
}

func TestHandler_CreateShippingMethod_Success(t *testing.T) {
	mockService := &MockService{
		OnCreateMethod: func(ctx context.Context, tenantID, name, typification, description string, cost float64, estTime string) (*models.ShippingMethod, error) {
			// El test verifica que la descripción mantenga minúsculas y mayúsculas normales
			if description != "Entrega en menos de 45 minutos" {
				t.Errorf("la descripción se deformó, recibida: %s", description)
			}
			if typification != "DELIVERY" {
				t.Errorf("la tipificación se recibió mal: %s", typification)
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
		"name": "Envío Moto Express",
		"description": "Entrega en menos de 45 minutos",
		"cost": 1500.00,
		"estimated_time": "30-45 min"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/shipping-methods", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-ok")

	rr := httptest.NewRecorder()

	// Llamamos al ruteador del handler
	handler.HandleShippingRoutes(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("se esperaba status 201 Created, se obtuvo: %d", rr.Code)
	}

	var response models.ShippingMethod
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("error al decodificar la respuesta JSON: %v", err)
	}

	if response.Description != "Entrega en menos de 45 minutos" {
		t.Errorf("se guardó la descripción incorrectamente: %s", response.Description)
	}
}

func TestHandler_CreateShippingMethod_MissingTenantHeader(t *testing.T) {
	handler := NewHandler(&MockService{})

	req := httptest.NewRequest(http.MethodPost, "/api/shipping-methods", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	// No agregamos el Header "X-Tenant-ID"

	rr := httptest.NewRecorder()
	handler.HandleShippingRoutes(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("se esperaba status 400 Bad Request por falta de Tenant, se obtuvo: %d", rr.Code)
	}
}
