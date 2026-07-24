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

func ptr(s string) *string {
	return &s
}

type mockService struct {
	registerTenantFunc func(ctx context.Context, name, subdomain, taxID string) (*models.Tenant, error)
	updateTenantFunc   func(ctx context.Context, id string, name, subdomain, taxID string, active *bool) (*models.Tenant, error)
	getByIDFunc        func(ctx context.Context, id string) (*models.Tenant, error)
	deleteTenantFunc   func(ctx context.Context, id string) error
	getAllTenantsFunc  func(ctx context.Context) ([]models.Tenant, error)
}

func (m *mockService) RegisterTenant(ctx context.Context, name, subdomain, taxID string) (*models.Tenant, error) {
	if m.registerTenantFunc != nil {
		return m.registerTenantFunc(ctx, name, subdomain, taxID)
	}
	return nil, nil
}

func (m *mockService) GetAllTenants(ctx context.Context) ([]models.Tenant, error) {
	if m.getAllTenantsFunc != nil {
		return m.getAllTenantsFunc(ctx)
	}
	return nil, nil
}

func (m *mockService) UpdateTenant(ctx context.Context, id string, name, subdomain, taxID string, active *bool) (*models.Tenant, error) {
	if m.updateTenantFunc != nil {
		return m.updateTenantFunc(ctx, id, name, subdomain, taxID, active)
	}
	return nil, nil
}

func (m *mockService) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockService) DeleteTenant(ctx context.Context, id string) error {
	if m.deleteTenantFunc != nil {
		return m.deleteTenantFunc(ctx, id)
	}
	return nil
}

func TestHandler_RegisterTenant_Success(t *testing.T) {
	mockSvc := &mockService{
		registerTenantFunc: func(ctx context.Context, name, subdomain, taxID string) (*models.Tenant, error) {
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
		t.Errorf("se esperaba codigo %d, se obtuvo %d", http.StatusCreated, rec.Code)
	}
}

func TestHandler_UpdateTenant_Success(t *testing.T) {
	mockSvc := &mockService{
		updateTenantFunc: func(ctx context.Context, id, name, subdomain, taxID string, active *bool) (*models.Tenant, error) {
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
		t.Errorf("se esperaba codigo %d, se obtuvo %d", http.StatusOK, rec.Code)
	}
}

func TestHandler_UpdateTenant_ValidationError(t *testing.T) {
	mockSvc := &mockService{}
	handler := NewHandler(mockSvc)
	reqStruct := UpdateTenantRequest{
		Name:      ptr(""),
		Subdomain: ptr("nuevo-sub"),
	}
	body, _ := json.Marshal(reqStruct)

	req := httptest.NewRequest(http.MethodPut, "/api/tenants/test-id", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	handler.UpdateTenant(rec, req, "test-id")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("se esperaba codigo de validación %d, se obtuvo %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandler_DeleteTenant_Success(t *testing.T) {
	deleteCalled := false
	mockSvc := &mockService{
		deleteTenantFunc: func(ctx context.Context, id string) error {
			deleteCalled = true
			return nil
		},
	}

	handler := NewHandler(mockSvc)

	req := httptest.NewRequest(http.MethodDelete, "/api/tenants/test-id", nil)
	rec := httptest.NewRecorder()

	handler.DeleteTenant(rec, req, "test-id")

	if rec.Code != http.StatusNoContent && rec.Code != http.StatusOK {
		t.Errorf("se esperaba codigo de exito, se obtuvo %d", rec.Code)
	}

	if !deleteCalled {
		t.Error("se esperaba que se llamara al metodo DeleteTenant del servicio")
	}
}
