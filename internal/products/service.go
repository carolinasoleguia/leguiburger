package products

import (
	"context"
	"errors"
	"leguiburger/internal/brands"
	"leguiburger/internal/models"
	"strings"
)

var (
	ErrProductNotFound         = errors.New("producto no encontrado")
	ErrDuplicateProductName    = errors.New("ya existe un producto con ese nombre para esta marca")
	ErrInvalidProductData      = errors.New("el nombre del producto es obligatorio")
	ErrInvalidProductPrice     = errors.New("el precio del producto no puede ser negativo")
	ErrBrandNotFoundForProduct = errors.New("la marca especificada no existe")
)

type Service interface {
	CreateProduct(ctx context.Context, brandID, name, description string, basePrice float64, imageURL string) (*models.Product, error)
	GetProduct(ctx context.Context, brandID, id string) (*models.Product, error)
	ListProducts(ctx context.Context, brandID string) ([]models.Product, error)
	UpdateProduct(ctx context.Context, brandID, id, name, description string, basePrice *float64, imageURL string, isActive *bool) (*models.Product, error)
	DeleteProduct(ctx context.Context, brandID, id string) error
}

type service struct {
	repo      Repository
	brandRepo brands.Repository
}

func NewService(repo Repository, brandRepo brands.Repository) Service {
	return &service{
		repo:      repo,
		brandRepo: brandRepo,
	}
}

func (s *service) CreateProduct(ctx context.Context, brandID, name, description string, basePrice float64, imageURL string) (*models.Product, error) {
	cleanName := strings.TrimSpace(name)
	cleanBrandID := strings.TrimSpace(brandID)
	if cleanBrandID == "" || cleanName == "" {
		return nil, ErrInvalidProductData
	}
	if basePrice < 0 {
		return nil, ErrInvalidProductPrice
	}

	brand, err := s.brandRepo.GetByID(ctx, cleanBrandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, ErrBrandNotFoundForProduct
	}

	existing, err := s.repo.GetByName(ctx, cleanBrandID, cleanName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateProductName
	}

	product := &models.Product{
		BrandID:     cleanBrandID,
		Name:        cleanName,
		Description: strings.TrimSpace(description),
		BasePrice:   basePrice,
		ImageURL:    strings.TrimSpace(imageURL),
		IsActive:    true,
	}

	if err := s.repo.Create(ctx, product); err != nil {
		if strings.Contains(err.Error(), "23503") || strings.Contains(err.Error(), "products_brand_id_fkey") {
			return nil, ErrBrandNotFoundForProduct
		}
		return nil, err
	}

	return product, nil
}

func (s *service) GetProduct(ctx context.Context, brandID, id string) (*models.Product, error) {
	product, err := s.repo.GetByID(ctx, strings.TrimSpace(brandID), id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}
	return product, nil
}

func (s *service) ListProducts(ctx context.Context, brandID string) ([]models.Product, error) {
	cleanBrandID := strings.TrimSpace(brandID)
	brand, err := s.brandRepo.GetByID(ctx, cleanBrandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, ErrBrandNotFoundForProduct
	}

	return s.repo.FetchAll(ctx, cleanBrandID)
}

func (s *service) UpdateProduct(ctx context.Context, brandID, id, name, description string, basePrice *float64, imageURL string, isActive *bool) (*models.Product, error) {
	cleanBrandID := strings.TrimSpace(brandID)
	product, err := s.repo.GetByID(ctx, cleanBrandID, id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	if basePrice != nil {
		if *basePrice < 0 {
			return nil, ErrInvalidProductPrice
		}
		product.BasePrice = *basePrice
	}
	if name != "" {
		cleanName := strings.TrimSpace(name)
		if cleanName == "" {
			return nil, ErrInvalidProductData
		}
		if cleanName != product.Name {
			existing, err := s.repo.GetByName(ctx, cleanBrandID, cleanName)
			if err != nil {
				return nil, err
			}
			if existing != nil && existing.ID != product.ID {
				return nil, ErrDuplicateProductName
			}
			product.Name = cleanName
		}
	}
	if description != "" {
		product.Description = strings.TrimSpace(description)
	}
	if imageURL != "" {
		product.ImageURL = strings.TrimSpace(imageURL)
	}
	if isActive != nil {
		product.IsActive = *isActive
	}

	if err := s.repo.Update(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (s *service) DeleteProduct(ctx context.Context, brandID, id string) error {
	product, err := s.repo.GetByID(ctx, strings.TrimSpace(brandID), id)
	if err != nil {
		return err
	}
	if product == nil {
		return ErrProductNotFound
	}

	return s.repo.Delete(ctx, strings.TrimSpace(brandID), id)
}
