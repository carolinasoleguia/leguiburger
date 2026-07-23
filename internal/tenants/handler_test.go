package tenants

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"leguiburger/internal/models"
)

// Helper para crear punteros a string de forma sencilla en los tests
func ptr(s string) *string {
	return &s
}

// 1. Creamos un Mock del Servicio
type mockService struct {
	OnRegisterTenant func(ctx context.Context, name, subdomain, taxID string) (*models.Tenant, error)
	OnUpdateTenant   func(ctx context.Context, id string, name, subdomain, taxID string, active *bool) (*models.Tenant, error)
	OnGetByID        func(ctx context.Context, id string) (*models.Tenant, error)
	OnDeleteTenant   func(ctx context.Context, id string) error
	OnGetAllTenants  func(ctx context.Context) ([]models.Tenant, error)
}

func (m *mockService) RegisterTenant(ctx context.Context, name, subdomain, taxID string) (*models.Tenant, error) {
	if m.OnRegisterTenant != nil {
		return m.OnRegisterTenant(ctx, name, subdomain, taxID)
	}
	return nil, nil
}

func (m *mockService) GetAllTenants(ctx context.Context) ([]models.Tenant, error) {
	if m.OnGetAllTenants != nil {
		return m.OnGetAllTenants(ctx)
	}
	return nil, nil
}

func (m *mockService) UpdateTenant(ctx context.Context, id string, name, subdomain, taxID string, active *bool) (*models.Tenant, error) {
	if m.OnUpdateTenant != nil {
		return m.OnUpdateTenant(ctx, id, name, subdomain, taxID, active)
	}
	return nil, nil
}

func (m *mockService) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	if m.OnGetByID != nil {
		return m.OnGetByID(ctx, id)
	}
	return nil, nil
}

func (m *mockService) DeleteTenant(ctx context.Context, id string) error {
	if m.OnDeleteTenant != nil {
		return m.OnDeleteTenant(ctx, id)
	}
	return nil
}

// 2. Tests para el Handler

func TestHandler_RegisterTenant_Success(t *testing.T) {
	mockSvc := &mockService{
		OnRegisterTenant: func(ctx context.Context, name, subdomain, taxID string) (*models.Tenant, error) {
			return &models.Tenant{
				ID:        "algun-uuid",
				Name:      name,
				Subdomain: subdomain,
				TaxID:     taxID,
				Active:    true,
			}, nil
		},
	}

	handler := NewHandler(mockSvc)

	payload := map[string]string{
		"name":      "Leguiburger",
		"subdomain": "legui-centro",
		"tax_id":    "20359486163",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/tenants", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	handler.CreateTenant(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Se esperaba código %d, se obtuvo %d", http.StatusCreated, rec.Code)
	}
}

func TestHandler_UpdateTenant_Success(t *testing.T) {
	mockSvc := &mockService{
		OnUpdateTenant: func(ctx context.Context, id, name, subdomain, taxID string, active *bool) (*models.Tenant, error) {
			return &models.Tenant{
				ID:        id,
				Name:      name,
				Subdomain: subdomain,
				TaxID:     taxID,
				Active:    *active,
			}, nil
		},
	}

	handler := NewHandler(mockSvc)
	nuevoActive := false

	// Usamos el helper ptr() para crear los punteros requeridos por el Request Struct
	reqStruct := UpdateTenantRequest{
		Name:      ptr("Nuevo Nombre"),
		Subdomain: ptr("nuevo-sub"),
		TaxID:     ptr("20359486163"),
		Active:    &nuevoActive,
	}
	body, _ := json.Marshal(reqStruct)

	req := httptest.NewRequest(http.MethodPut, "/api/tenants/test-id", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	handler.UpdateTenant(rec, req, "test-id")

	if rec.Code != http.StatusOK {
		t.Errorf("Se esperaba código %d, se obtuvo %d", http.StatusOK, rec.Code)
	}
}

func TestHandler_UpdateTenant_ValidationError(t *testing.T) {
	mockSvc := &mockService{}
	handler := NewHandler(mockSvc)

	// Mandamos campos vacíos explícitos para forzar el fallo de validación
	reqStruct := UpdateTenantRequest{
		Name:      ptr(""),
		Subdomain: ptr("nuevo-sub"),
	}
	body, _ := json.Marshal(reqStruct)

	req := httptest.NewRequest(http.MethodPut, "/api/tenants/test-id", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	handler.UpdateTenant(rec, req, "test-id")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Se esperaba código de validación %d, se obtuvo %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandler_DeleteTenant_Success(t *testing.T) {
	deleteCalled := false
	mockSvc := &mockService{
		OnDeleteTenant: func(ctx context.Context, id string) error {
			deleteCalled = true // 1. Se escribe acá
			return nil
		},
	}

	handler := NewHandler(mockSvc)

	req := httptest.NewRequest(http.MethodDelete, "/api/tenants/test-id", nil)
	rec := httptest.NewRecorder()

	handler.DeleteTenant(rec, req, "test-id")

	if rec.Code != http.StatusNoContent && rec.Code != http.StatusOK {
		t.Errorf("Se esperaba código de éxito, se obtuvo %d", rec.Code)
	}

	if !deleteCalled {
		t.Error("Se esperaba que se llamara al método DeleteTenant del servicio")
	}
}
