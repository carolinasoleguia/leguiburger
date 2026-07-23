package products

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

func TestCreateProduct_Success(t *testing.T) {
	repo := &MockRepository{
		OnGetByName: func(ctx context.Context, tenantID, name string) (*models.Product, error) {
			if name != "Doble Cheddar" {
				t.Errorf("se esperaba nombre normalizado, se obtuvo: %s", name)
			}
			return nil, nil
		},
		OnCreate: func(ctx context.Context, product *models.Product) error {
			product.ID = "generated-id"
			return nil
		},
	}

	tenantRepo := &MockTenantRepository{
		OnGetByID: func(ctx context.Context, id string) (*models.Tenant, error) {
			return &models.Tenant{ID: id}, nil
		},
	}

	service := NewService(repo, tenantRepo)
	trackStock := false

	res, err := service.CreateProduct(context.Background(), "tenant-1", " Doble Cheddar ", " Burger con cheddar ", 4500, 20, &trackStock, " https://example.com/burger.jpg ")
	if err != nil {
		t.Fatalf("se esperaba éxito, se obtuvo error: %v", err)
	}

	if res.Name != "Doble Cheddar" || res.Description != "Burger con cheddar" || res.CurrentPrice != 4500 || res.CurrentStock != 20 || res.TrackStock != false || res.ImageURL != "https://example.com/burger.jpg" || res.IsActive != true {
		t.Errorf("los datos no se normalizaron correctamente: %+v", res)
	}
}

func TestCreateProduct_InvalidName(t *testing.T) {
	service := NewService(&MockRepository{}, &MockTenantRepository{})

	_, err := service.CreateProduct(context.Background(), "tenant-1", "", "Desc", 100, 1, nil, "")
	if !errors.Is(err, ErrInvalidProductData) {
		t.Errorf("se esperaba ErrInvalidProductData, se obtuvo: %v", err)
	}
}

func TestCreateProduct_InvalidPrice(t *testing.T) {
	service := NewService(&MockRepository{}, &MockTenantRepository{})

	_, err := service.CreateProduct(context.Background(), "tenant-1", "Burger", "Desc", -1, 1, nil, "")
	if !errors.Is(err, ErrInvalidProductPrice) {
		t.Errorf("se esperaba ErrInvalidProductPrice, se obtuvo: %v", err)
	}
}

func TestCreateProduct_DuplicateName(t *testing.T) {
	repo := &MockRepository{
		OnGetByName: func(ctx context.Context, tenantID, name string) (*models.Product, error) {
			return &models.Product{ID: "existing-id", Name: name}, nil
		},
	}

	service := NewService(repo, &MockTenantRepository{})
	_, err := service.CreateProduct(context.Background(), "tenant-1", "Burger", "Desc", 100, 1, nil, "")

	if !errors.Is(err, ErrDuplicateProductName) {
		t.Errorf("se esperaba ErrDuplicateProductName, se obtuvo: %v", err)
	}
}

func TestListProducts_TenantNotFound(t *testing.T) {
	repo := &MockRepository{}
	tenantRepo := &MockTenantRepository{
		OnGetByID: func(ctx context.Context, id string) (*models.Tenant, error) {
			return nil, nil
		},
	}

	service := NewService(repo, tenantRepo)
	_, err := service.ListProducts(context.Background(), "tenant-fantasma")

	if !errors.Is(err, ErrTenantNotFoundForProduct) {
		t.Errorf("se esperaba ErrTenantNotFoundForProduct, se obtuvo: %v", err)
	}
}

func TestUpdateProduct_Success(t *testing.T) {
	repo := &MockRepository{
		OnGetByID: func(ctx context.Context, tenantID, id string) (*models.Product, error) {
			return &models.Product{ID: id, TenantID: tenantID, Name: "Burger", Description: "Vieja", CurrentPrice: 100, CurrentStock: 5, TrackStock: true, IsActive: true}, nil
		},
		OnGetByName: func(ctx context.Context, tenantID, name string) (*models.Product, error) {
			return nil, nil
		},
		OnUpdate: func(ctx context.Context, product *models.Product) error {
			return nil
		},
	}

	service := NewService(repo, &MockTenantRepository{})
	newPrice := 150.0
	newStock := 8
	newActive := false

	res, err := service.UpdateProduct(context.Background(), "tenant-1", "product-1", "Doble Burger", "Nueva", &newPrice, &newStock, nil, "https://example.com/new.jpg", &newActive)
	if err != nil {
		t.Fatalf("se esperaba éxito, se obtuvo error: %v", err)
	}

	if res.Name != "Doble Burger" || res.Description != "Nueva" || res.CurrentPrice != 150 || res.CurrentStock != 8 || res.ImageURL != "https://example.com/new.jpg" || res.IsActive != false {
		t.Errorf("los datos no se actualizaron correctamente: %+v", res)
	}
}

func TestDeleteProduct_NotFound(t *testing.T) {
	repo := &MockRepository{
		OnGetByID: func(ctx context.Context, tenantID, id string) (*models.Product, error) {
			return nil, nil
		},
	}

	service := NewService(repo, &MockTenantRepository{})
	err := service.DeleteProduct(context.Background(), "tenant-1", "missing")

	if !errors.Is(err, ErrProductNotFound) {
		t.Errorf("se esperaba ErrProductNotFound, se obtuvo: %v", err)
	}
}
