package recipes

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

func TestCreateRecipe_Success(t *testing.T) {
	repo := &MockRepository{
		OnProductExistsForTenant: func(ctx context.Context, tenantID, productID string) (bool, error) {
			return true, nil
		},
		OnSupplyExistsForTenant: func(ctx context.Context, tenantID, supplyID string) (bool, error) {
			return true, nil
		},
		OnGetByID: func(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error) {
			return nil, nil
		},
		OnCreate: func(ctx context.Context, recipe *models.Recipe) error {
			return nil
		},
	}

	tenantRepo := &MockTenantRepository{
		OnGetByID: func(ctx context.Context, id string) (*models.Tenant, error) {
			return &models.Tenant{ID: id}, nil
		},
	}

	service := NewService(repo, tenantRepo)
	res, err := service.CreateRecipe(context.Background(), "tenant-1", " product-1 ", " supply-1 ", 0.250)
	if err != nil {
		t.Fatalf("se esperaba éxito, se obtuvo error: %v", err)
	}

	if res.ProductID != "product-1" || res.SupplyID != "supply-1" || res.QuantityUsed != 0.250 {
		t.Errorf("los datos no se normalizaron correctamente: %+v", res)
	}
}

func TestCreateRecipe_InvalidData(t *testing.T) {
	service := NewService(&MockRepository{}, &MockTenantRepository{})

	_, err := service.CreateRecipe(context.Background(), "tenant-1", "", "supply-1", 1)
	if !errors.Is(err, ErrInvalidRecipeData) {
		t.Errorf("se esperaba ErrInvalidRecipeData, se obtuvo: %v", err)
	}
}

func TestCreateRecipe_InvalidQuantity(t *testing.T) {
	service := NewService(&MockRepository{}, &MockTenantRepository{})

	_, err := service.CreateRecipe(context.Background(), "tenant-1", "product-1", "supply-1", 0)
	if !errors.Is(err, ErrInvalidRecipeQuantity) {
		t.Errorf("se esperaba ErrInvalidRecipeQuantity, se obtuvo: %v", err)
	}
}

func TestCreateRecipe_ProductNotFound(t *testing.T) {
	repo := &MockRepository{
		OnProductExistsForTenant: func(ctx context.Context, tenantID, productID string) (bool, error) {
			return false, nil
		},
	}

	service := NewService(repo, &MockTenantRepository{})
	_, err := service.CreateRecipe(context.Background(), "tenant-1", "missing-product", "supply-1", 1)

	if !errors.Is(err, ErrProductNotFoundForRecipe) {
		t.Errorf("se esperaba ErrProductNotFoundForRecipe, se obtuvo: %v", err)
	}
}

func TestCreateRecipe_Duplicate(t *testing.T) {
	repo := &MockRepository{
		OnProductExistsForTenant: func(ctx context.Context, tenantID, productID string) (bool, error) {
			return true, nil
		},
		OnSupplyExistsForTenant: func(ctx context.Context, tenantID, supplyID string) (bool, error) {
			return true, nil
		},
		OnGetByID: func(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error) {
			return &models.Recipe{ProductID: productID, SupplyID: supplyID, QuantityUsed: 1}, nil
		},
	}

	service := NewService(repo, &MockTenantRepository{})
	_, err := service.CreateRecipe(context.Background(), "tenant-1", "product-1", "supply-1", 1)

	if !errors.Is(err, ErrDuplicateRecipe) {
		t.Errorf("se esperaba ErrDuplicateRecipe, se obtuvo: %v", err)
	}
}

func TestListRecipes_TenantNotFound(t *testing.T) {
	repo := &MockRepository{}
	tenantRepo := &MockTenantRepository{
		OnGetByID: func(ctx context.Context, id string) (*models.Tenant, error) {
			return nil, nil
		},
	}

	service := NewService(repo, tenantRepo)
	_, err := service.ListRecipes(context.Background(), "tenant-fantasma")

	if !errors.Is(err, ErrTenantNotFoundForRecipe) {
		t.Errorf("se esperaba ErrTenantNotFoundForRecipe, se obtuvo: %v", err)
	}
}

func TestUpdateRecipe_Success(t *testing.T) {
	repo := &MockRepository{
		OnGetByID: func(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error) {
			return &models.Recipe{ProductID: productID, SupplyID: supplyID, QuantityUsed: 1}, nil
		},
		OnUpdate: func(ctx context.Context, recipe *models.Recipe) error {
			return nil
		},
	}

	service := NewService(repo, &MockTenantRepository{})
	res, err := service.UpdateRecipe(context.Background(), "tenant-1", "product-1", "supply-1", 2.5)
	if err != nil {
		t.Fatalf("se esperaba éxito, se obtuvo error: %v", err)
	}

	if res.QuantityUsed != 2.5 {
		t.Errorf("la cantidad no se actualizó correctamente: %+v", res)
	}
}

func TestDeleteRecipe_NotFound(t *testing.T) {
	repo := &MockRepository{
		OnGetByID: func(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error) {
			return nil, nil
		},
	}

	service := NewService(repo, &MockTenantRepository{})
	err := service.DeleteRecipe(context.Background(), "tenant-1", "product-1", "missing")

	if !errors.Is(err, ErrRecipeNotFound) {
		t.Errorf("se esperaba ErrRecipeNotFound, se obtuvo: %v", err)
	}
}
