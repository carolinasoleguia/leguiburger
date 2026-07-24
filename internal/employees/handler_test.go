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

type mockService struct {
	createEmployeeFunc  func(ctx context.Context, tenantID, firstName, lastName, email, passwordHash, phone, role string) (*models.Employee, error)
	getEmployeeFunc     func(ctx context.Context, tenantID, id string) (*models.Employee, error)
	listEmployeesFunc   func(ctx context.Context, tenantID string) ([]models.Employee, error)
	getAllEmployeesFunc func(ctx context.Context) ([]models.Employee, error)
	updateEmployeeFunc  func(ctx context.Context, tenantID, id, firstName, lastName, email, passwordHash, phone, role string, isActive *bool) (*models.Employee, error)
	deleteEmployeeFunc  func(ctx context.Context, tenantID, id string) error
}

func (m *mockService) CreateEmployee(ctx context.Context, tenantID, firstName, lastName, email, passwordHash, phone, role string) (*models.Employee, error) {
	if m.createEmployeeFunc != nil {
		return m.createEmployeeFunc(ctx, tenantID, firstName, lastName, email, passwordHash, phone, role)
	}
	return nil, nil
}

func (m *mockService) GetEmployee(ctx context.Context, tenantID, id string) (*models.Employee, error) {
	if m.getEmployeeFunc != nil {
		return m.getEmployeeFunc(ctx, tenantID, id)
	}
	return nil, nil
}

func (m *mockService) ListEmployees(ctx context.Context, tenantID string) ([]models.Employee, error) {
	if m.listEmployeesFunc != nil {
		return m.listEmployeesFunc(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockService) GetAllEmployees(ctx context.Context) ([]models.Employee, error) {
	if m.getAllEmployeesFunc != nil {
		return m.getAllEmployeesFunc(ctx)
	}
	return nil, nil
}

func (m *mockService) UpdateEmployee(ctx context.Context, tenantID, id, firstName, lastName, email, passwordHash, phone, role string, isActive *bool) (*models.Employee, error) {
	if m.updateEmployeeFunc != nil {
		return m.updateEmployeeFunc(ctx, tenantID, id, firstName, lastName, email, passwordHash, phone, role, isActive)
	}
	return nil, nil
}

func (m *mockService) DeleteEmployee(ctx context.Context, tenantID, id string) error {
	if m.deleteEmployeeFunc != nil {
		return m.deleteEmployeeFunc(ctx, tenantID, id)
	}
	return nil
}

func TestHandler_CreateEmployee_Success(t *testing.T) {
	mockService := &mockService{
		createEmployeeFunc: func(ctx context.Context, tenantID, firstName, lastName, email, passwordHash, phone, role string) (*models.Employee, error) {
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
		"password": "hash123",
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
	handler := NewHandler(&mockService{})

	req := httptest.NewRequest(http.MethodPost, "/api/employees", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleEmployeeRoutes(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("se esperaba status 400 Bad Request por falta de Tenant, se obtuvo: %d", rr.Code)
	}
}
