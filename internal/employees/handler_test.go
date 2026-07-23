package employees

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
	OnCreateEmployee func(ctx context.Context, tenantID, firstName, lastName, email, passwordHash, phone, role string) (*models.Employee, error)
}

func (m *MockService) CreateEmployee(ctx context.Context, tenantID, firstName, lastName, email, passwordHash, phone, role string) (*models.Employee, error) {
	return m.OnCreateEmployee(ctx, tenantID, firstName, lastName, email, passwordHash, phone, role)
}

func (m *MockService) GetEmployee(ctx context.Context, tenantID, id string) (*models.Employee, error) {
	return nil, nil
}

func (m *MockService) ListEmployees(ctx context.Context, tenantID string) ([]models.Employee, error) {
	return nil, nil
}

func (m *MockService) UpdateEmployee(ctx context.Context, tenantID, id, firstName, lastName, email, passwordHash, phone, role string, isActive *bool) (*models.Employee, error) {
	return nil, nil
}

func (m *MockService) DeleteEmployee(ctx context.Context, tenantID, id string) error {
	return nil
}

func TestHandler_CreateEmployee_Success(t *testing.T) {
	mockService := &MockService{
		OnCreateEmployee: func(ctx context.Context, tenantID, firstName, lastName, email, passwordHash, phone, role string) (*models.Employee, error) {
			return &models.Employee{
				ID:           "new-id",
				TenantID:     &tenantID,
				FirstName:    firstName,
				LastName:     lastName,
				Email:        email,
				PasswordHash: passwordHash,
				Phone:        phone,
				Role:         role,
				IsActive:     true,
			}, nil
		},
	}

	handler := NewHandler(mockService)

	body := []byte(`{
		"first_name": "Ana",
		"last_name": "Eguia",
		"email": "ana@email.com",
		"password_hash": "hash123",
		"phone": "2215555555",
		"role": "admin"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/employees", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-ok")

	rr := httptest.NewRecorder()
	handler.HandleEmployeeRoutes(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("se esperaba status 201 Created, se obtuvo: %d", rr.Code)
	}

	var response models.Employee
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("error al decodificar la respuesta JSON: %v", err)
	}

	if response.TenantID == nil || *response.TenantID != "tenant-ok" || response.Email != "ana@email.com" || response.Role != "admin" {
		t.Errorf("respuesta inesperada: %+v", response)
	}
}

func TestHandler_CreateEmployee_MissingTenantHeader(t *testing.T) {
	handler := NewHandler(&MockService{})

	req := httptest.NewRequest(http.MethodPost, "/api/employees", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleEmployeeRoutes(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("se esperaba status 400 Bad Request por falta de Tenant, se obtuvo: %d", rr.Code)
	}
}
