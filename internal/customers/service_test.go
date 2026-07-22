package customers

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

func TestCreateCustomer_Success(t *testing.T) {
	repo := &MockRepository{
		OnGetByEmail: func(ctx context.Context, tenantID, email string) (*models.Customer, error) {
			if email != "juan@email.com" {
				t.Errorf("se esperaba email normalizado, se obtuvo: %s", email)
			}
			return nil, nil
		},
		OnCreate: func(ctx context.Context, customer *models.Customer) error {
			customer.ID = "generated-id"
			return nil
		},
	}

	tenantRepo := &MockTenantRepository{
		OnGetByID: func(ctx context.Context, id string) (*models.Tenant, error) {
			return &models.Tenant{ID: id}, nil
		},
	}

	service := NewService(repo, tenantRepo)
	res, err := service.CreateCustomer(context.Background(), "tenant-1", " Juan ", " Perez ", "  JUAN@EMAIL.COM ", " 2215555555 ")
	if err != nil {
		t.Fatalf("se esperaba éxito, se obtuvo error: %v", err)
	}

	if res.FirstName != "Juan" || res.LastName != "Perez" || res.Email != "juan@email.com" || res.Phone != "2215555555" {
		t.Errorf("los datos no se normalizaron correctamente: %+v", res)
	}
}

func TestCreateCustomer_InvalidData(t *testing.T) {
	service := NewService(&MockRepository{}, &MockTenantRepository{})

	_, err := service.CreateCustomer(context.Background(), "tenant-1", "", "Perez", "juan@email.com", "")
	if !errors.Is(err, ErrInvalidCustomerData) {
		t.Errorf("se esperaba ErrInvalidCustomerData, se obtuvo: %v", err)
	}
}

func TestCreateCustomer_DuplicateEmail(t *testing.T) {
	repo := &MockRepository{
		OnGetByEmail: func(ctx context.Context, tenantID, email string) (*models.Customer, error) {
			return &models.Customer{ID: "existing-id", Email: email}, nil
		},
	}

	service := NewService(repo, &MockTenantRepository{})
	_, err := service.CreateCustomer(context.Background(), "tenant-1", "Juan", "Perez", "juan@email.com", "")

	if !errors.Is(err, ErrDuplicateCustomerEmail) {
		t.Errorf("se esperaba ErrDuplicateCustomerEmail, se obtuvo: %v", err)
	}
}

func TestListCustomers_TenantNotFound(t *testing.T) {
	repo := &MockRepository{}
	tenantRepo := &MockTenantRepository{
		OnGetByID: func(ctx context.Context, id string) (*models.Tenant, error) {
			return nil, nil
		},
	}

	service := NewService(repo, tenantRepo)
	_, err := service.ListCustomers(context.Background(), "tenant-fantasma")

	if !errors.Is(err, ErrTenantNotFoundForCustomer) {
		t.Errorf("se esperaba ErrTenantNotFoundForCustomer, se obtuvo: %v", err)
	}
}

func TestUpdateCustomer_Success(t *testing.T) {
	repo := &MockRepository{
		OnGetByID: func(ctx context.Context, tenantID, id string) (*models.Customer, error) {
			return &models.Customer{ID: id, TenantID: tenantID, FirstName: "Juan", LastName: "Perez", Email: "juan@email.com"}, nil
		},
		OnGetByEmail: func(ctx context.Context, tenantID, email string) (*models.Customer, error) {
			return nil, nil
		},
		OnUpdate: func(ctx context.Context, customer *models.Customer) error {
			return nil
		},
	}

	service := NewService(repo, &MockTenantRepository{})
	res, err := service.UpdateCustomer(context.Background(), "tenant-1", "customer-1", "Juana", "", "JUANA@EMAIL.COM", "")
	if err != nil {
		t.Fatalf("se esperaba éxito, se obtuvo error: %v", err)
	}

	if res.FirstName != "Juana" || res.LastName != "Perez" || res.Email != "juana@email.com" {
		t.Errorf("los datos no se actualizaron correctamente: %+v", res)
	}
}

func TestDeleteCustomer_NotFound(t *testing.T) {
	repo := &MockRepository{
		OnGetByID: func(ctx context.Context, tenantID, id string) (*models.Customer, error) {
			return nil, nil
		},
	}

	service := NewService(repo, &MockTenantRepository{})
	err := service.DeleteCustomer(context.Background(), "tenant-1", "missing")

	if !errors.Is(err, ErrCustomerNotFound) {
		t.Errorf("se esperaba ErrCustomerNotFound, se obtuvo: %v", err)
	}
}
