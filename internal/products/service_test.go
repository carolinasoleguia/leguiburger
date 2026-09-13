package products

import (
	"context"
	"errors"
	"testing"

	"leguiburger/internal/models"
)

type mockBrandRepository struct {
	getByIDFunc func(ctx context.Context, id string) (*models.Brand, error)
}

func (m *mockBrandRepository) GetByID(ctx context.Context, id string) (*models.Brand, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return &models.Brand{ID: id}, nil
}
func (m *mockBrandRepository) Create(ctx context.Context, brand *models.Brand) error { return nil }
func (m *mockBrandRepository) GetAll(ctx context.Context) ([]models.Brand, error)    { return nil, nil }
func (m *mockBrandRepository) GetByName(ctx context.Context, name string) (*models.Brand, error) {
	return nil, nil
}
func (m *mockBrandRepository) Update(ctx context.Context, brand *models.Brand) error { return nil }
func (m *mockBrandRepository) Delete(ctx context.Context, id string) error           { return nil }

func TestCreateProduct_Success(t *testing.T) {
	repo := &mockRepository{
		getByNameFunc: func(ctx context.Context, brandID, name string) (*models.Product, error) {
			if brandID != "brand-1" {
				t.Errorf("se esperaba brand-1, se obtuvo: %s", brandID)
			}
			if name != "Doble Cheddar" {
				t.Errorf("se esperaba nombre normalizado, se obtuvo: %s", name)
			}
			return nil, nil
		},
		createFunc: func(ctx context.Context, product *models.Product) error {
			product.ID = "generated-id"
			return nil
		},
	}

	service := NewService(repo, &mockBrandRepository{})

	res, err := service.CreateProduct(context.Background(), "brand-1", " Doble Cheddar ", " Burger con cheddar ", 4500, " https://example.com/burger.jpg ")
	if err != nil {
		t.Fatalf("se esperaba exito, se obtuvo error: %v", err)
	}

	if res.BrandID != "brand-1" || res.Name != "Doble Cheddar" || res.Description != "Burger con cheddar" || res.BasePrice != 4500 || res.ImageURL != "https://example.com/burger.jpg" || res.IsActive != true {
		t.Errorf("los datos no se normalizaron correctamente: %+v", res)
	}
}

func TestCreateProduct_InvalidName(t *testing.T) {
	service := NewService(&mockRepository{}, &mockBrandRepository{})

	_, err := service.CreateProduct(context.Background(), "brand-1", "", "Desc", 100, "")
	if !errors.Is(err, ErrInvalidProductData) {
		t.Errorf("se esperaba ErrInvalidProductData, se obtuvo: %v", err)
	}
}

func TestCreateProduct_InvalidPrice(t *testing.T) {
	service := NewService(&mockRepository{}, &mockBrandRepository{})

	_, err := service.CreateProduct(context.Background(), "brand-1", "Burger", "Desc", -1, "")
	if !errors.Is(err, ErrInvalidProductPrice) {
		t.Errorf("se esperaba ErrInvalidProductPrice, se obtuvo: %v", err)
	}
}

func TestCreateProduct_BrandNotFound(t *testing.T) {
	brandRepo := &mockBrandRepository{
		getByIDFunc: func(ctx context.Context, id string) (*models.Brand, error) {
			return nil, nil
		},
	}

	service := NewService(&mockRepository{}, brandRepo)
	_, err := service.CreateProduct(context.Background(), "missing-brand", "Burger", "Desc", 100, "")

	if !errors.Is(err, ErrBrandNotFoundForProduct) {
		t.Errorf("se esperaba ErrBrandNotFoundForProduct, se obtuvo: %v", err)
	}
}

func TestCreateProduct_DuplicateName(t *testing.T) {
	repo := &mockRepository{
		getByNameFunc: func(ctx context.Context, brandID, name string) (*models.Product, error) {
			return &models.Product{ID: "existing-id", BrandID: brandID, Name: name}, nil
		},
	}

	service := NewService(repo, &mockBrandRepository{})
	_, err := service.CreateProduct(context.Background(), "brand-1", "Burger", "Desc", 100, "")

	if !errors.Is(err, ErrDuplicateProductName) {
		t.Errorf("se esperaba ErrDuplicateProductName, se obtuvo: %v", err)
	}
}

func TestListProducts_BrandNotFound(t *testing.T) {
	brandRepo := &mockBrandRepository{
		getByIDFunc: func(ctx context.Context, id string) (*models.Brand, error) {
			return nil, nil
		},
	}

	service := NewService(&mockRepository{}, brandRepo)
	_, err := service.ListProducts(context.Background(), "brand-fantasma")

	if !errors.Is(err, ErrBrandNotFoundForProduct) {
		t.Errorf("se esperaba ErrBrandNotFoundForProduct, se obtuvo: %v", err)
	}
}

func TestUpdateProduct_Success(t *testing.T) {
	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, brandID, id string) (*models.Product, error) {
			return &models.Product{ID: id, BrandID: brandID, Name: "Burger", Description: "Vieja", BasePrice: 100, IsActive: true}, nil
		},
		getByNameFunc: func(ctx context.Context, brandID, name string) (*models.Product, error) {
			return nil, nil
		},
		updateFunc: func(ctx context.Context, product *models.Product) error {
			return nil
		},
	}

	service := NewService(repo, &mockBrandRepository{})
	newPrice := 150.0
	newActive := false

	res, err := service.UpdateProduct(context.Background(), "brand-1", "product-1", "Doble Burger", "Nueva", &newPrice, "https://example.com/new.jpg", &newActive)
	if err != nil {
		t.Fatalf("se esperaba exito, se obtuvo error: %v", err)
	}

	if res.Name != "Doble Burger" || res.Description != "Nueva" || res.BasePrice != 150 || res.ImageURL != "https://example.com/new.jpg" || res.IsActive != false {
		t.Errorf("los datos no se actualizaron correctamente: %+v", res)
	}
}

func TestDeleteProduct_NotFound(t *testing.T) {
	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, brandID, id string) (*models.Product, error) {
			return nil, nil
		},
	}

	service := NewService(repo, &mockBrandRepository{})
	err := service.DeleteProduct(context.Background(), "brand-1", "missing")

	if !errors.Is(err, ErrProductNotFound) {
		t.Errorf("se esperaba ErrProductNotFound, se obtuvo: %v", err)
	}
}
