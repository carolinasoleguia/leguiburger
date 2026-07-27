package customers

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
	createCustomerFunc func(ctx context.Context, tenantID, firstName, lastName, email, phone string) (*models.Customer, error)
}

func (m *mockService) CreateCustomer(ctx context.Context, tenantID, firstName, lastName, email, phone string) (*models.Customer, error) {
	return m.createCustomerFunc(ctx, tenantID, firstName, lastName, email, phone)
}

func (m *mockService) GetCustomer(ctx context.Context, tenantID, id string) (*models.Customer, error) {
	return nil, nil
}

func (m *mockService) ListCustomers(ctx context.Context, tenantID string) ([]models.Customer, error) {
	return nil, nil
}

func (m *mockService) UpdateCustomer(ctx context.Context, tenantID, id, firstName, lastName, email, phone string) (*models.Customer, error) {
	return nil, nil
}

func (m *mockService) DeleteCustomer(ctx context.Context, tenantID, id string) error {
	return nil
}

func TestHandler_CreateCustomer_Success(t *testing.T) {
	mockService := &mockService{
		createCustomerFunc: func(ctx context.Context, tenantID, firstName, lastName, email, phone string) (*models.Customer, error) {
			return &models.Customer{
				ID:        "new-id",
				TenantID:  tenantID,
				FirstName: firstName,
				LastName:  lastName,
				Email:     email,
				Phone:     phone,
			}, nil
		},
	}

	handler := NewHandler(mockService)

	req := testutil.JSONRequest(t, http.MethodPost, "/api/customers", map[string]interface{}{
		"first_name": "Juan",
		"last_name":  "Perez",
		"email":      "juan@email.com",
		"phone":      "2215555555",
	})
	req.Header.Set("X-Tenant-ID", "tenant-ok")
	claims := &auth.Claims{Role: auth.RoleSuperAdmin, TenantID: "tenant-ok"}
	req = testutil.WithClaims(t, req, claims)

	rr := httptest.NewRecorder()
	handler.HandleCustomerRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusCreated)

	var response models.Customer
	testutil.DecodeJSONBody(t, rr, &response)

	if response.TenantID != "tenant-ok" || response.Email != "juan@email.com" {
		t.Errorf("se recibio una respuesta incorrecta: %+v", response)
	}
}

func TestHandler_CreateCustomer_MissingTenantHeader(t *testing.T) {
	handler := NewHandler(&mockService{})

	req := testutil.JSONRequest(t, http.MethodPost, "/api/customers", map[string]interface{}{})

	rr := httptest.NewRecorder()
	handler.HandleCustomerRoutes(rr, req)

	testutil.AssertStatus(t, rr, http.StatusUnauthorized)
}
