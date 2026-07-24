package shipping

import (
	"context"
	"errors"
	"testing"

	"leguiburger/internal/models"
)

type mockTenantRepository struct {
	getByIDFunc               func(ctx context.Context, id string) (*models.Tenant, error)
	createFunc                func(ctx context.Context, tenant *models.Tenant) error
	getByTaxIDFunc            func(ctx context.Context, taxId string) (*models.Tenant, error)
	getBySubdomainFunc        func(ctx context.Context, subdomain string) (*models.Tenant, error)
	getByNameAndSubdomainFunc func(ctx context.Context, name string, subdomain string) (*models.Tenant, error)
	updateFunc                func(ctx context.Context, tenant *models.Tenant) error
	deleteFunc                func(ctx context.Context, id string) error
	getAllFunc                func(ctx context.Context) ([]models.Tenant, error)
}

func (m *mockTenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	return m.getByIDFunc(ctx, id)
}
func (m *mockTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error { return nil }
func (m *mockTenantRepository) GetByTaxID(ctx context.Context, taxId string) (*models.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) GetAll(ctx context.Context) ([]models.Tenant, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}
	return nil, nil
}
func (m *mockTenantRepository) GetBySubdomain(ctx context.Context, subdomain string) (*models.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) GetByNameAndSubdomain(ctx context.Context, name string, subdomain string) (*models.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error { return nil }
func (m *mockTenantRepository) Delete(ctx context.Context, id string) error             { return nil }

func TestCreateMethod_Success(t *testing.T) {
	repo := &mockRepository{
		getByNameAndTypificationFunc: func(ctx context.Context, tenantID, name, typification string) (*models.ShippingMethod, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, sm *models.ShippingMethod) error {
			sm.ID = "generated-uuid-123"
			return nil
		},
	}
	tenantRepo := &mockTenantRepository{
		getByIDFunc: func(ctx context.Context, id string) (*models.Tenant, error) {
			return &models.Tenant{ID: id}, nil
		},
	}

	service := NewService(repo, tenantRepo)
	ctx := context.Background()

	res, err := service.CreateMethod(ctx, "tenant-1", "Moto Express", "delivery", "Envio rapido", 150.0, "30m")
	if err != nil {
		t.Fatalf("se esperaba exito, se obtuvo error: %v", err)
	}

	if res.Name != "Moto Express" {
		t.Errorf("se esperaba Name 'Moto Express', se obtuvo: %s", res.Name)
	}
	if res.Typification != "DELIVERY" {
		t.Errorf("se esperaba Typification 'DELIVERY', se obtuvo: %s", res.Typification)
	}
}

func TestCreateMethod_InvalidCost(t *testing.T) {
	repo := &mockRepository{}
	tenantRepo := &mockTenantRepository{}
	service := NewService(repo, tenantRepo)

	_, err := service.CreateMethod(context.Background(), "tenant-1", "Test", "DELIVERY", "Desc", -50.0, "10m")
	if !errors.Is(err, ErrInvalidCost) {
		t.Errorf("se esperaba ErrInvalidCost, se obtuvo: %v", err)
	}
}

func TestCreateMethod_Duplicate(t *testing.T) {
	repo := &mockRepository{
		getByNameAndTypificationFunc: func(ctx context.Context, tenantID, name, typification string) (*models.ShippingMethod, error) {
			return &models.ShippingMethod{ID: "existente", Name: name, Typification: typification}, nil
		},
	}
	tenantRepo := &mockTenantRepository{}

	service := NewService(repo, tenantRepo)
	_, err := service.CreateMethod(context.Background(), "tenant-1", "Moto Express", "DELIVERY", "Desc", 150.0, "30m")

	if !errors.Is(err, ErrDuplicateShipping) {
		t.Errorf("se esperaba ErrDuplicateShipping, se obtuvo: %v", err)
	}
}

func TestCreateMethod_TenantNotFound(t *testing.T) {
	repo := &mockRepository{
		getByNameAndTypificationFunc: func(ctx context.Context, tenantID, name, typification string) (*models.ShippingMethod, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, sm *models.ShippingMethod) error {
			return errors.New("ERROR: insert violates foreign key constraint \"shipping_methods_tenant_id_fkey\" (SQLSTATE 23503)")
		},
	}
	tenantRepo := &mockTenantRepository{}

	service := NewService(repo, tenantRepo)
	_, err := service.CreateMethod(context.Background(), "fake-tenant", "Moto Express", "DELIVERY", "Desc", 150.0, "30m")

	if !errors.Is(err, ErrTenantNotFoundForShipping) {
		t.Errorf("se esperaba ErrTenantNotFoundForShipping, se obtuvo: %v", err)
	}
}
func TestListMethods_TenantNotFound(t *testing.T) {
	repo := &mockRepository{}
	tenantRepo := &mockTenantRepository{
		getByIDFunc: func(ctx context.Context, id string) (*models.Tenant, error) {
			return nil, nil
		},
	}

	service := NewService(repo, tenantRepo)
	_, err := service.ListMethods(context.Background(), "id-fantasma")

	if !errors.Is(err, ErrTenantNotFoundForShipping) {
		t.Errorf("se esperaba ErrTenantNotFoundForShipping, se obtuvo: %v", err)
	}
}
