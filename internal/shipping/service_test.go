package shipping

import (
	"context"
	"errors"
	"testing"

	"leguiburger/internal/models"
)

// Mock manual para el repositorio de Tenants para aislar las pruebas de integración
type MockTenantRepository struct {
	OnGetByID               func(ctx context.Context, id string) (*models.Tenant, error)
	OnCreate                func(ctx context.Context, tenant *models.Tenant) error
	OnGetByTaxID            func(ctx context.Context, taxId string) (*models.Tenant, error)
	OnGetBySubdomain        func(ctx context.Context, subdomain string) (*models.Tenant, error)
	OnGetByNameAndSubdomain func(ctx context.Context, name string, subdomain string) (*models.Tenant, error)
	OnUpdate                func(ctx context.Context, tenant *models.Tenant) error
	OnDelete                func(ctx context.Context, id string) error
	OnGetAll                func(ctx context.Context) ([]models.Tenant, error)
}

func (m *MockTenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	return m.OnGetByID(ctx, id)
}
func (m *MockTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error { return nil }
func (m *MockTenantRepository) GetByTaxID(ctx context.Context, taxId string) (*models.Tenant, error) {
	return nil, nil
}
func (m *MockTenantRepository) GetAll(ctx context.Context) ([]models.Tenant, error) {
	if m.OnGetAll != nil {
		return m.OnGetAll(ctx)
	}
	return nil, nil
}
func (m *MockTenantRepository) GetBySubdomain(ctx context.Context, subdomain string) (*models.Tenant, error) {
	return nil, nil
}
func (m *MockTenantRepository) GetByNameAndSubdomain(ctx context.Context, name string, subdomain string) (*models.Tenant, error) {
	return nil, nil
}
func (m *MockTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error { return nil }
func (m *MockTenantRepository) Delete(ctx context.Context, id string) error             { return nil }

func TestCreateMethod_Success(t *testing.T) {
	repo := &MockRepository{
		OnGetByNameAndTypification: func(ctx context.Context, tenantID, name, typification string) (*models.ShippingMethod, error) {
			return nil, nil // No existe duplicado
		},
		OnCreate: func(ctx context.Context, sm *models.ShippingMethod) error {
			sm.ID = "generated-uuid-123"
			return nil
		},
	}

	// Mock del tenant que devuelve que sí existe
	tenantRepo := &MockTenantRepository{
		OnGetByID: func(ctx context.Context, id string) (*models.Tenant, error) {
			return &models.Tenant{ID: id}, nil
		},
	}

	service := NewService(repo, tenantRepo) // 👈 Pasamos ambos mocks
	ctx := context.Background()

	res, err := service.CreateMethod(ctx, "tenant-1", "Moto Express", "delivery", "Envío rápido", 150.0, "30m")
	if err != nil {
		t.Fatalf("se esperaba éxito, se obtuvo error: %v", err)
	}

	if res.Name != "Moto Express" {
		t.Errorf("se esperaba Name 'Moto Express', se obtuvo: %s", res.Name)
	}
	if res.Typification != "DELIVERY" { // Debe estar normalizado a mayúsculas
		t.Errorf("se esperaba Typification 'DELIVERY', se obtuvo: %s", res.Typification)
	}
}

func TestCreateMethod_InvalidCost(t *testing.T) {
	repo := &MockRepository{}
	tenantRepo := &MockTenantRepository{}
	service := NewService(repo, tenantRepo)

	_, err := service.CreateMethod(context.Background(), "tenant-1", "Test", "DELIVERY", "Desc", -50.0, "10m")
	if !errors.Is(err, ErrInvalidCost) {
		t.Errorf("se esperaba ErrInvalidCost, se obtuvo: %v", err)
	}
}

func TestCreateMethod_Duplicate(t *testing.T) {
	repo := &MockRepository{
		OnGetByNameAndTypification: func(ctx context.Context, tenantID, name, typification string) (*models.ShippingMethod, error) {
			// Simulamos que ya existe un método con el mismo Name + Typification
			return &models.ShippingMethod{ID: "existente", Name: name, Typification: typification}, nil
		},
	}
	tenantRepo := &MockTenantRepository{}

	service := NewService(repo, tenantRepo)
	_, err := service.CreateMethod(context.Background(), "tenant-1", "Moto Express", "DELIVERY", "Desc", 150.0, "30m")

	if !errors.Is(err, ErrDuplicateShipping) {
		t.Errorf("se esperaba ErrDuplicateShipping, se obtuvo: %v", err)
	}
}

func TestCreateMethod_TenantNotFound(t *testing.T) {
	repo := &MockRepository{
		OnGetByNameAndTypification: func(ctx context.Context, tenantID, name, typification string) (*models.ShippingMethod, error) {
			return nil, nil
		},
		OnCreate: func(ctx context.Context, sm *models.ShippingMethod) error {
			// Simulamos violación de clave foránea de PostgreSQL (error 23503)
			return errors.New("ERROR: insert violates foreign key constraint \"shipping_methods_tenant_id_fkey\" (SQLSTATE 23503)")
		},
	}
	tenantRepo := &MockTenantRepository{}

	service := NewService(repo, tenantRepo)
	_, err := service.CreateMethod(context.Background(), "fake-tenant", "Moto Express", "DELIVERY", "Desc", 150.0, "30m")

	if !errors.Is(err, ErrTenantNotFoundForShipping) {
		t.Errorf("se esperaba ErrTenantNotFoundForShipping, se obtuvo: %v", err)
	}
}

// 🧪 NUEVO TEST: Validamos la lógica de buscar un tenant inexistente al listar
func TestListMethods_TenantNotFound(t *testing.T) {
	repo := &MockRepository{}

	// El mock de tenant devuelve nil (No existe el tenant en la DB)
	tenantRepo := &MockTenantRepository{
		OnGetByID: func(ctx context.Context, id string) (*models.Tenant, error) {
			return nil, nil
		},
	}

	service := NewService(repo, tenantRepo)
	_, err := service.ListMethods(context.Background(), "id-fantasma")

	if !errors.Is(err, ErrTenantNotFoundForShipping) {
		t.Errorf("se esperaba ErrTenantNotFoundForShipping, se obtuvo: %v", err)
	}
}
