package tenantproducts

import (
	"context"
	"errors"
	"testing"

	"leguiburger/internal/models"
)

type mockTenantRepository struct {
	getByIDFunc                func(ctx context.Context, id string) (*models.Tenant, error)
	getAllFunc                 func(ctx context.Context) ([]models.Tenant, error)
	getByBrandAndSubdomainFunc func(ctx context.Context, brandID, subdomain string) (*models.Tenant, error)
	getByBrandIDFunc           func(ctx context.Context, brandID string) ([]models.Tenant, error)
}

func (m *mockTenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return &models.Tenant{ID: id, BrandID: "brand-1"}, nil
}
func (m *mockTenantRepository) GetAll(ctx context.Context) ([]models.Tenant, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}
	return nil, nil
}
func (m *mockTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error { return nil }
func (m *mockTenantRepository) GetByTaxID(ctx context.Context, taxId string) (*models.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) GetBySubdomain(ctx context.Context, subdomain string) (*models.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) GetByNameAndSubdomain(ctx context.Context, name string, subdomain string) (*models.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) GetByBrandAndSubdomain(ctx context.Context, brandID, subdomain string) (*models.Tenant, error) {
	if m.getByBrandAndSubdomainFunc != nil {
		return m.getByBrandAndSubdomainFunc(ctx, brandID, subdomain)
	}
	return nil, nil
}
func (m *mockTenantRepository) GetByBrandID(ctx context.Context, brandID string) ([]models.Tenant, error) {
	if m.getByBrandIDFunc != nil {
		return m.getByBrandIDFunc(ctx, brandID)
	}
	return nil, nil
}
func (m *mockTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error { return nil }
func (m *mockTenantRepository) Delete(ctx context.Context, id string) error             { return nil }

func TestCreateTenantProduct_Success(t *testing.T) {
	price := 4800.0
	trackStock := false
	isAvailable := true
	repo := &mockRepository{
		productBelongsToTenantBrandFunc: func(ctx context.Context, tenantID, productID string) (bool, error) {
			if tenantID != "tenant-1" || productID != "product-1" {
				t.Fatalf("ids inesperados: tenant=%s product=%s", tenantID, productID)
			}
			return true, nil
		},
		getByIDFunc: func(ctx context.Context, tenantID, productID string) (*models.TenantProduct, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, tenantProduct *models.TenantProduct) error {
			return nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	res, err := service.CreateTenantProduct(context.Background(), "tenant-1", " product-1 ", &price, 10, &trackStock, &isAvailable)
	if err != nil {
		t.Fatalf("se esperaba exito, se obtuvo: %v", err)
	}

	if res.TenantID != "tenant-1" || res.ProductID != "product-1" || res.PriceOverride == nil || *res.PriceOverride != 4800 || res.CurrentStock != 10 || res.TrackStock != false || res.IsAvailable != true || res.IsActive != true {
		t.Errorf("tenant product incorrecto: %+v", res)
	}
}

func TestCreateTenantProduct_Duplicate(t *testing.T) {
	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, tenantID, productID string) (*models.TenantProduct, error) {
			return &models.TenantProduct{TenantID: tenantID, ProductID: productID}, nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	_, err := service.CreateTenantProduct(context.Background(), "tenant-1", "product-1", nil, 0, nil, nil)

	if !errors.Is(err, ErrDuplicateTenantProduct) {
		t.Errorf("se esperaba ErrDuplicateTenantProduct, se obtuvo: %v", err)
	}
}

func TestCreateTenantProduct_ProductBrandMismatch(t *testing.T) {
	repo := &mockRepository{
		productBelongsToTenantBrandFunc: func(ctx context.Context, tenantID, productID string) (bool, error) {
			return false, nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	_, err := service.CreateTenantProduct(context.Background(), "tenant-1", "product-other-brand", nil, 0, nil, nil)

	if !errors.Is(err, ErrProductBrandMismatch) {
		t.Errorf("se esperaba ErrProductBrandMismatch, se obtuvo: %v", err)
	}
}

func TestCreateTenantProduct_InvalidStock(t *testing.T) {
	service := NewService(&mockRepository{}, &mockTenantRepository{})

	_, err := service.CreateTenantProduct(context.Background(), "tenant-1", "product-1", nil, -1, nil, nil)
	if !errors.Is(err, ErrInvalidTenantStock) {
		t.Errorf("se esperaba ErrInvalidTenantStock, se obtuvo: %v", err)
	}
}

func TestUpdateTenantProduct_Success(t *testing.T) {
	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, tenantID, productID string) (*models.TenantProduct, error) {
			return &models.TenantProduct{TenantID: tenantID, ProductID: productID, CurrentStock: 5, TrackStock: true, IsAvailable: true, IsActive: true}, nil
		},
		updateFunc: func(ctx context.Context, tenantProduct *models.TenantProduct) error {
			return nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	price := 5000.0
	stock := 12
	isAvailable := false

	res, err := service.UpdateTenantProduct(context.Background(), "tenant-1", "product-1", &price, &stock, nil, &isAvailable, nil)
	if err != nil {
		t.Fatalf("se esperaba exito, se obtuvo: %v", err)
	}

	if res.PriceOverride == nil || *res.PriceOverride != 5000 || res.CurrentStock != 12 || res.IsAvailable != false {
		t.Errorf("tenant product no actualizado: %+v", res)
	}
}

func TestDeleteTenantProduct_NotFound(t *testing.T) {
	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, tenantID, productID string) (*models.TenantProduct, error) {
			return nil, nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	err := service.DeleteTenantProduct(context.Background(), "tenant-1", "missing")

	if !errors.Is(err, ErrTenantProductNotFound) {
		t.Errorf("se esperaba ErrTenantProductNotFound, se obtuvo: %v", err)
	}
}
