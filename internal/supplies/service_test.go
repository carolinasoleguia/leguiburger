package supplies

import (
	"context"
	"errors"
	"testing"

	"leguiburger/internal/models"
)

type mockTenantRepository struct {
	getByIDFunc               func(ctx context.Context, id string) (*models.Tenant, error)
	getAllFunc                func(ctx context.Context) ([]models.Tenant, error)
	createFunc                func(ctx context.Context, tenant *models.Tenant) error
	getByTaxIDFunc            func(ctx context.Context, taxID string) (*models.Tenant, error)
	getBySubdomainFunc        func(ctx context.Context, subdomain string) (*models.Tenant, error)
	getByNameAndSubdomainFunc func(ctx context.Context, name, subdomain string) (*models.Tenant, error)
	updateFunc                func(ctx context.Context, tenant *models.Tenant) error
	deleteFunc                func(ctx context.Context, id string) error
}

func (m *mockTenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockTenantRepository) GetAll(ctx context.Context) ([]models.Tenant, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}
	return nil, nil
}

func (m *mockTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, tenant)
	}
	return nil
}

func (m *mockTenantRepository) GetByTaxID(ctx context.Context, taxID string) (*models.Tenant, error) {
	if m.getByTaxIDFunc != nil {
		return m.getByTaxIDFunc(ctx, taxID)
	}
	return nil, nil
}

func (m *mockTenantRepository) GetBySubdomain(ctx context.Context, subdomain string) (*models.Tenant, error) {
	if m.getBySubdomainFunc != nil {
		return m.getBySubdomainFunc(ctx, subdomain)
	}
	return nil, nil
}

func (m *mockTenantRepository) GetByNameAndSubdomain(ctx context.Context, name, subdomain string) (*models.Tenant, error) {
	if m.getByNameAndSubdomainFunc != nil {
		return m.getByNameAndSubdomainFunc(ctx, name, subdomain)
	}
	return nil, nil
}

func (m *mockTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, tenant)
	}
	return nil
}

func (m *mockTenantRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestCreateSupply_Success(t *testing.T) {
	repo := &mockRepository{
		getByNameFunc: func(ctx context.Context, tenantID, name string) (*models.Supply, error) {
			if name != "Pan Brioche" {
				t.Errorf("se esperaba nombre normalizado, se obtuvo: %s", name)
			}
			return nil, nil
		},
		createFunc: func(ctx context.Context, supply *models.Supply) error {
			supply.ID = "generated-id"
			return nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	res, err := service.CreateSupply(context.Background(), "tenant-1", " Pan Brioche ", 100.5, 25.75, " KG ")
	if err != nil {
		t.Fatalf("se esperaba exito, se obtuvo error: %v", err)
	}

	if res.Name != "Pan Brioche" || res.CurrentWholesaleCost != 100.5 || res.CurrentStock != 25.75 || res.MeasurementUnit != "kg" || !res.IsActive {
		t.Errorf("los datos no se normalizaron correctamente: %+v", res)
	}
}

func TestCreateSupply_InvalidData(t *testing.T) {
	service := NewService(&mockRepository{}, &mockTenantRepository{})

	_, err := service.CreateSupply(context.Background(), "tenant-1", "", 10, 1, "kg")
	if !errors.Is(err, ErrInvalidSupplyData) {
		t.Errorf("se esperaba ErrInvalidSupplyData, se obtuvo: %v", err)
	}
}

func TestCreateSupply_InvalidCost(t *testing.T) {
	service := NewService(&mockRepository{}, &mockTenantRepository{})

	_, err := service.CreateSupply(context.Background(), "tenant-1", "Pan", -1, 1, "kg")
	if !errors.Is(err, ErrInvalidSupplyCost) {
		t.Errorf("se esperaba ErrInvalidSupplyCost, se obtuvo: %v", err)
	}
}

func TestCreateSupply_InvalidStock(t *testing.T) {
	service := NewService(&mockRepository{}, &mockTenantRepository{})

	_, err := service.CreateSupply(context.Background(), "tenant-1", "Pan", 1, -1, "kg")
	if !errors.Is(err, ErrInvalidSupplyStock) {
		t.Errorf("se esperaba ErrInvalidSupplyStock, se obtuvo: %v", err)
	}
}

func TestCreateSupply_DuplicateName(t *testing.T) {
	repo := &mockRepository{
		getByNameFunc: func(ctx context.Context, tenantID, name string) (*models.Supply, error) {
			return &models.Supply{ID: "existing-id", Name: name}, nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	_, err := service.CreateSupply(context.Background(), "tenant-1", "Pan", 1, 1, "kg")

	if !errors.Is(err, ErrDuplicateSupplyName) {
		t.Errorf("se esperaba ErrDuplicateSupplyName, se obtuvo: %v", err)
	}
}

func TestListSupplies_TenantNotFound(t *testing.T) {
	repo := &mockRepository{}
	tenantRepo := &mockTenantRepository{
		getByIDFunc: func(ctx context.Context, id string) (*models.Tenant, error) {
			return nil, nil
		},
	}

	service := NewService(repo, tenantRepo)
	_, err := service.ListSupplies(context.Background(), "tenant-fantasma")

	if !errors.Is(err, ErrTenantNotFoundForSupply) {
		t.Errorf("se esperaba ErrTenantNotFoundForSupply, se obtuvo: %v", err)
	}
}

func TestUpdateSupply_Success(t *testing.T) {
	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, tenantID, id string) (*models.Supply, error) {
			return &models.Supply{ID: id, TenantID: tenantID, Name: "Pan", CurrentWholesaleCost: 100, CurrentStock: 5, MeasurementUnit: "kg", IsActive: true}, nil
		},
		getByNameFunc: func(ctx context.Context, tenantID, name string) (*models.Supply, error) {
			return nil, nil
		},
		updateFunc: func(ctx context.Context, supply *models.Supply) error {
			return nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	newCost := 150.0
	newStock := 8.5
	newActive := false

	res, err := service.UpdateSupply(context.Background(), "tenant-1", "supply-1", "Queso", &newCost, &newStock, "UN", &newActive)
	if err != nil {
		t.Fatalf("se esperaba exito, se obtuvo error: %v", err)
	}

	if res.Name != "Queso" || res.CurrentWholesaleCost != 150 || res.CurrentStock != 8.5 || res.MeasurementUnit != "un" || res.IsActive {
		t.Errorf("los datos no se actualizaron correctamente: %+v", res)
	}
}

func TestDeleteSupply_NotFound(t *testing.T) {
	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, tenantID, id string) (*models.Supply, error) {
			return nil, nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	err := service.DeleteSupply(context.Background(), "tenant-1", "missing")

	if !errors.Is(err, ErrSupplyNotFound) {
		t.Errorf("se esperaba ErrSupplyNotFound, se obtuvo: %v", err)
	}
}
