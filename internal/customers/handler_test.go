package customers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"leguiburger/internal/models"
)

type MockService struct {
	OnCreateCustomer func(ctx context.Context, tenantID, firstName, lastName, email, phone string) (*models.Customer, error)
}

func (m *MockService) CreateCustomer(ctx context.Context, tenantID, firstName, lastName, email, phone string) (*models.Customer, error) {
	return m.OnCreateCustomer(ctx, tenantID, firstName, lastName, email, phone)
}

func (m *MockService) GetCustomer(ctx context.Context, tenantID, id string) (*models.Customer, error) {
	return nil, nil
}

func (m *MockService) ListCustomers(ctx context.Context, tenantID string) ([]models.Customer, error) {
	return nil, nil
}

func (m *MockService) UpdateCustomer(ctx context.Context, tenantID, id, firstName, lastName, email, phone string) (*models.Customer, error) {
	return nil, nil
}

func (m *MockService) DeleteCustomer(ctx context.Context, tenantID, id string) error {
	return nil
}

func TestHandler_CreateCustomer_Success(t *testing.T) {
	mockService := &MockService{
		OnCreateCustomer: func(ctx context.Context, tenantID, firstName, lastName, email, phone string) (*models.Customer, error) {
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

	body := []byte(`{
		"first_name": "Juan",
		"last_name": "Perez",
		"email": "juan@email.com",
		"phone": "2215555555"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-ok")

	rr := httptest.NewRecorder()
	handler.HandleCustomerRoutes(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("se esperaba status 201 Created, se obtuvo: %d", rr.Code)
	}

	var response models.Customer
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("error al decodificar la respuesta JSON: %v", err)
	}

	if response.TenantID != "tenant-ok" || response.Email != "juan@email.com" {
		t.Errorf("se recibió una respuesta incorrecta: %+v", response)
	}
}

func TestHandler_CreateCustomer_MissingTenantHeader(t *testing.T) {
	handler := NewHandler(&MockService{})

	req := httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleCustomerRoutes(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("se esperaba status 400 Bad Request por falta de Tenant, se obtuvo: %d", rr.Code)
	}
}
