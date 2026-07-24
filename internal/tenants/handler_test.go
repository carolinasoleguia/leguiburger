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

// ---------------- MOCK SERVICE ----------------

type mockService struct {
	registerTenantFunc func(
		ctx context.Context,
		brandID string,
		subdomain string,
	) (*models.Tenant, error)

	updateTenantFunc func(
		ctx context.Context,
		id string,
		subdomain string,
		active *bool,
	) (*models.Tenant, error)

	deleteTenantFunc func(
		ctx context.Context,
		id string,
	) error

	getAllTenantsFunc func(
		ctx context.Context,
	) ([]models.Tenant, error)
}

func (m *mockService) RegisterTenant(
	ctx context.Context,
	brandID string,
	subdomain string,
) (*models.Tenant, error) {

	if m.registerTenantFunc != nil {
		return m.registerTenantFunc(
			ctx,
			brandID,
			subdomain,
		)
	}

	return nil, nil
}

func (m *mockService) UpdateTenant(
	ctx context.Context,
	id string,
	subdomain string,
	active *bool,
) (*models.Tenant, error) {

	if m.updateTenantFunc != nil {
		return m.updateTenantFunc(
			ctx,
			id,
			subdomain,
			active,
		)
	}

	return nil, nil
}

func (m *mockService) DeleteTenant(
	ctx context.Context,
	id string,
) error {

	if m.deleteTenantFunc != nil {
		return m.deleteTenantFunc(ctx, id)
	}

	return nil
}

func (m *mockService) GetAllTenants(
	ctx context.Context,
) ([]models.Tenant, error) {

	if m.getAllTenantsFunc != nil {
		return m.getAllTenantsFunc(ctx)
	}

	return nil, nil
}

// ---------------- TESTS ----------------

func TestHandler_RegisterTenant_Success(t *testing.T) {

	mockSvc := &mockService{

		registerTenantFunc: func(
			ctx context.Context,
			brandID string,
			subdomain string,
		) (*models.Tenant, error) {

			return &models.Tenant{

				ID: "tenant-id",

				BrandID: brandID,

				Subdomain: subdomain,

				Active: true,
			}, nil
		},
	}

	handler := NewHandler(mockSvc)

	payload := map[string]string{

		"brand_id": "brand-id",

		"subdomain": "legui-centro",
	}

	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/tenants",
		bytes.NewBuffer(body),
	)

	rec := httptest.NewRecorder()

	handler.CreateTenant(
		rec,
		req,
	)

	if rec.Code != http.StatusCreated {

		t.Errorf(
			"se esperaba codigo %d, se obtuvo %d",
			http.StatusCreated,
			rec.Code,
		)
	}

}

func TestHandler_UpdateTenant_Success(t *testing.T) {

	mockSvc := &mockService{

		updateTenantFunc: func(
			ctx context.Context,
			id string,
			subdomain string,
			active *bool,
		) (*models.Tenant, error) {

			return &models.Tenant{

				ID: id,

				BrandID: "brand-id",

				Subdomain: subdomain,

				Active: *active,
			}, nil
		},
	}

	handler := NewHandler(mockSvc)

	active := false

	reqStruct := UpdateTenantRequest{

		Subdomain: ptr("nuevo-sub"),

		Active: &active,
	}

	body, _ := json.Marshal(reqStruct)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/tenants/test-id",
		bytes.NewBuffer(body),
	)

	rec := httptest.NewRecorder()

	handler.UpdateTenant(
		rec,
		req,
		"test-id",
	)

	if rec.Code != http.StatusOK {

		t.Errorf(
			"se esperaba codigo %d, se obtuvo %d",
			http.StatusOK,
			rec.Code,
		)
	}

}

func TestHandler_UpdateTenant_ValidationError(t *testing.T) {

	mockSvc := &mockService{}

	handler := NewHandler(mockSvc)

	empty := ptr("")

	reqStruct := UpdateTenantRequest{

		Subdomain: empty,
	}

	body, _ := json.Marshal(reqStruct)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/tenants/test-id",
		bytes.NewBuffer(body),
	)

	rec := httptest.NewRecorder()

	handler.UpdateTenant(
		rec,
		req,
		"test-id",
	)

	if rec.Code != http.StatusBadRequest {

		t.Errorf(
			"se esperaba codigo %d, se obtuvo %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}

}

func TestHandler_DeleteTenant_Success(t *testing.T) {

	deleteCalled := false

	mockSvc := &mockService{

		deleteTenantFunc: func(
			ctx context.Context,
			id string,
		) error {

			deleteCalled = true

			return nil
		},
	}

	handler := NewHandler(mockSvc)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/tenants/test-id",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.DeleteTenant(
		rec,
		req,
		"test-id",
	)

	if rec.Code != http.StatusOK {

		t.Errorf(
			"se esperaba codigo %d, se obtuvo %d",
			http.StatusOK,
			rec.Code,
		)
	}

	if !deleteCalled {

		t.Error(
			"se esperaba llamar DeleteTenant",
		)
	}

}
