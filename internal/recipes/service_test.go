package recipes

import (
	"context"
	"errors"
	"testing"

	"leguiburger/internal/models"
)

type mockTenantRepository struct {
	getByIDFunc func(ctx context.Context, id string) (*models.Tenant, error)
	getAllFunc  func(ctx context.Context) ([]models.Tenant, error)
}

func (m *mockTenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	return m.getByIDFunc(ctx, id)
}

func (m *mockTenantRepository) GetAll(ctx context.Context) ([]models.Tenant, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}
	return nil, nil
}
func (m *mockTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	return nil
}
func (m *mockTenantRepository) GetByTaxID(ctx context.Context, taxId string) (*models.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) GetBySubdomain(ctx context.Context, subdomain string) (*models.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) GetByNameAndSubdomain(ctx context.Context, name string, subdomain string) (*models.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	return nil
}
func (m *mockTenantRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func TestCreateRecipe_Success(t *testing.T) {
	repo := &mockRepository{
		productExistsForTenantFunc: func(ctx context.Context, tenantID, productID string) (bool, error) {
			return true, nil
		},
		supplyExistsForTenantFunc: func(ctx context.Context, tenantID, supplyID string) (bool, error) {
			return true, nil
		},
		getByIDFunc: func(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, recipe *models.Recipe) error {
			return nil
		},
	}

	tenantRepo := &mockTenantRepository{
		getByIDFunc: func(ctx context.Context, id string) (*models.Tenant, error) {
			return &models.Tenant{ID: id}, nil
		},
	}

	service := NewService(repo, tenantRepo)
	res, err := service.CreateRecipe(context.Background(), "tenant-1", " product-1 ", " supply-1 ", 0.250)
	if err != nil {
		t.Fatalf("se esperaba exito, se obtuvo error: %v", err)
	}

	if res.ProductID != "product-1" || res.SupplyID != "supply-1" || res.QuantityUsed != 0.250 {
		t.Errorf("los datos no se normalizaron correctamente: %+v", res)
	}
}

func TestCreateRecipe_InvalidData(t *testing.T) {
	service := NewService(&mockRepository{}, &mockTenantRepository{})

	_, err := service.CreateRecipe(context.Background(), "tenant-1", "", "supply-1", 1)
	if !errors.Is(err, ErrInvalidRecipeData) {
		t.Errorf("se esperaba ErrInvalidRecipeData, se obtuvo: %v", err)
	}
}

func TestCreateRecipe_InvalidQuantity(t *testing.T) {
	service := NewService(&mockRepository{}, &mockTenantRepository{})

	_, err := service.CreateRecipe(context.Background(), "tenant-1", "product-1", "supply-1", 0)
	if !errors.Is(err, ErrInvalidRecipeQuantity) {
		t.Errorf("se esperaba ErrInvalidRecipeQuantity, se obtuvo: %v", err)
	}
}

func TestCreateRecipe_ProductNotFound(t *testing.T) {
	repo := &mockRepository{
		productExistsForTenantFunc: func(ctx context.Context, tenantID, productID string) (bool, error) {
			return false, nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	_, err := service.CreateRecipe(context.Background(), "tenant-1", "missing-product", "supply-1", 1)

	if !errors.Is(err, ErrProductNotFoundForRecipe) {
		t.Errorf("se esperaba ErrProductNotFoundForRecipe, se obtuvo: %v", err)
	}
}

func TestCreateRecipe_Duplicate(t *testing.T) {
	repo := &mockRepository{
		productExistsForTenantFunc: func(ctx context.Context, tenantID, productID string) (bool, error) {
			return true, nil
		},
		supplyExistsForTenantFunc: func(ctx context.Context, tenantID, supplyID string) (bool, error) {
			return true, nil
		},
		getByIDFunc: func(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error) {
			return &models.Recipe{ProductID: productID, SupplyID: supplyID, QuantityUsed: 1}, nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	_, err := service.CreateRecipe(context.Background(), "tenant-1", "product-1", "supply-1", 1)

	if !errors.Is(err, ErrDuplicateRecipe) {
		t.Errorf("se esperaba ErrDuplicateRecipe, se obtuvo: %v", err)
	}
}

func TestListRecipes_TenantNotFound(t *testing.T) {
	repo := &mockRepository{}
	tenantRepo := &mockTenantRepository{
		getByIDFunc: func(ctx context.Context, id string) (*models.Tenant, error) {
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
	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error) {
			return &models.Recipe{ProductID: productID, SupplyID: supplyID, QuantityUsed: 1}, nil
		},
		updateFunc: func(ctx context.Context, recipe *models.Recipe) error {
			return nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	res, err := service.UpdateRecipe(context.Background(), "tenant-1", "product-1", "supply-1", 2.5)
	if err != nil {
		t.Fatalf("se esperaba exito, se obtuvo error: %v", err)
	}

	if res.QuantityUsed != 2.5 {
		t.Errorf("la cantidad no se actualizo correctamente: %+v", res)
	}
}

func TestDeleteRecipe_NotFound(t *testing.T) {
	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error) {
			return nil, nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	err := service.DeleteRecipe(context.Background(), "tenant-1", "product-1", "missing")

	if !errors.Is(err, ErrRecipeNotFound) {
		t.Errorf("se esperaba ErrRecipeNotFound, se obtuvo: %v", err)
	}
}
