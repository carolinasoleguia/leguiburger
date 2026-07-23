package employees

import (
	"context"
	"errors"
	"testing"

	"leguiburger/internal/models"
)

type MockTenantRepository struct {
	OnGetByID func(ctx context.Context, id string) (*models.Tenant, error)
}

func (m *MockTenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	return m.OnGetByID(ctx, id)
}
func (m *MockTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	return nil
}
func (m *MockTenantRepository) GetByTaxID(ctx context.Context, taxId string) (*models.Tenant, error) {
	return nil, nil
}
func (m *MockTenantRepository) GetBySubdomain(ctx context.Context, subdomain string) (*models.Tenant, error) {
	return nil, nil
}
func (m *MockTenantRepository) GetByNameAndSubdomain(ctx context.Context, name string, subdomain string) (*models.Tenant, error) {
	return nil, nil
}
func (m *MockTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	return nil
}
func (m *MockTenantRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func TestCreateEmployee_Success(t *testing.T) {
	repo := &MockRepository{
		OnGetByEmail: func(ctx context.Context, email string) (*models.Employee, error) {
			if email != "admin@email.com" {
				t.Errorf("se esperaba email normalizado, se obtuvo: %s", email)
			}
			return nil, nil
		},
		OnCreate: func(ctx context.Context, employee *models.Employee) error {
			employee.ID = "generated-id"
			return nil
		},
	}

	tenantRepo := &MockTenantRepository{
		OnGetByID: func(ctx context.Context, id string) (*models.Tenant, error) {
			return &models.Tenant{ID: id}, nil
		},
	}

	service := NewService(repo, tenantRepo)
	res, err := service.CreateEmployee(context.Background(), "tenant-1", " Ana ", " Eguia ", " ADMIN@EMAIL.COM ", " hash123 ", " 2215555555 ", "ADMIN")
	if err != nil {
		t.Fatalf("se esperaba éxito, se obtuvo error: %v", err)
	}

	if res.FirstName != "Ana" || res.LastName != "Eguia" || res.Email != "admin@email.com" || res.PasswordHash != "hash123" || res.Phone != "2215555555" || res.Role != "admin" || res.IsActive != true {
		t.Errorf("los datos no se normalizaron correctamente: %+v", res)
	}
}

func TestCreateEmployee_DefaultRole(t *testing.T) {
	repo := &MockRepository{
		OnGetByEmail: func(ctx context.Context, email string) (*models.Employee, error) {
			return nil, nil
		},
	}

	service := NewService(repo, &MockTenantRepository{})
	res, err := service.CreateEmployee(context.Background(), "tenant-1", "Ana", "Eguia", "ana@email.com", "hash123", "", "")
	if err != nil {
		t.Fatalf("se esperaba éxito, se obtuvo error: %v", err)
	}

	if res.Role != "employee" {
		t.Errorf("se esperaba rol employee por defecto, se obtuvo: %s", res.Role)
	}
}

func TestCreateEmployee_InvalidData(t *testing.T) {
	service := NewService(&MockRepository{}, &MockTenantRepository{})

	_, err := service.CreateEmployee(context.Background(), "tenant-1", "", "Eguia", "ana@email.com", "hash123", "", "employee")
	if !errors.Is(err, ErrInvalidEmployeeData) {
		t.Errorf("se esperaba ErrInvalidEmployeeData, se obtuvo: %v", err)
	}
}

func TestCreateEmployee_InvalidRole(t *testing.T) {
	service := NewService(&MockRepository{}, &MockTenantRepository{})

	_, err := service.CreateEmployee(context.Background(), "tenant-1", "Ana", "Eguia", "ana@email.com", "hash123", "", "owner")
	if !errors.Is(err, ErrInvalidEmployeeRole) {
		t.Errorf("se esperaba ErrInvalidEmployeeRole, se obtuvo: %v", err)
	}
}

func TestCreateEmployee_DuplicateEmail(t *testing.T) {
	repo := &MockRepository{
		OnGetByEmail: func(ctx context.Context, email string) (*models.Employee, error) {
			return &models.Employee{ID: "existing-id", Email: email}, nil
		},
	}

	service := NewService(repo, &MockTenantRepository{})
	_, err := service.CreateEmployee(context.Background(), "tenant-1", "Ana", "Eguia", "ana@email.com", "hash123", "", "employee")

	if !errors.Is(err, ErrDuplicateEmployeeEmail) {
		t.Errorf("se esperaba ErrDuplicateEmployeeEmail, se obtuvo: %v", err)
	}
}

func TestListEmployees_TenantNotFound(t *testing.T) {
	repo := &MockRepository{}
	tenantRepo := &MockTenantRepository{
		OnGetByID: func(ctx context.Context, id string) (*models.Tenant, error) {
			return nil, nil
		},
	}

	service := NewService(repo, tenantRepo)
	_, err := service.ListEmployees(context.Background(), "tenant-fantasma")

	if !errors.Is(err, ErrTenantNotFoundForEmployee) {
		t.Errorf("se esperaba ErrTenantNotFoundForEmployee, se obtuvo: %v", err)
	}
}

func TestUpdateEmployee_Success(t *testing.T) {
	repo := &MockRepository{
		OnGetByID: func(ctx context.Context, tenantID, id string) (*models.Employee, error) {
			return &models.Employee{ID: id, TenantID: tenantID, FirstName: "Ana", LastName: "Eguia", Email: "ana@email.com", PasswordHash: "oldhash", Role: "employee", IsActive: true}, nil
		},
		OnGetByEmail: func(ctx context.Context, email string) (*models.Employee, error) {
			return nil, nil
		},
		OnUpdate: func(ctx context.Context, employee *models.Employee) error {
			return nil
		},
	}

	service := NewService(repo, &MockTenantRepository{})
	newActive := false

	res, err := service.UpdateEmployee(context.Background(), "tenant-1", "employee-1", "Juana", "", "JUANA@EMAIL.COM", "", "2219999999", "cashier", &newActive)
	if err != nil {
		t.Fatalf("se esperaba éxito, se obtuvo error: %v", err)
	}

	if res.FirstName != "Juana" || res.LastName != "Eguia" || res.Email != "juana@email.com" || res.Phone != "2219999999" || res.Role != "cashier" || res.IsActive != false {
		t.Errorf("los datos no se actualizaron correctamente: %+v", res)
	}
}

func TestDeleteEmployee_NotFound(t *testing.T) {
	repo := &MockRepository{
		OnGetByID: func(ctx context.Context, tenantID, id string) (*models.Employee, error) {
			return nil, nil
		},
	}

	service := NewService(repo, &MockTenantRepository{})
	err := service.DeleteEmployee(context.Background(), "tenant-1", "missing")

	if !errors.Is(err, ErrEmployeeNotFound) {
		t.Errorf("se esperaba ErrEmployeeNotFound, se obtuvo: %v", err)
	}
}
