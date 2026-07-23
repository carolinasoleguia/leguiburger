package products

import (
	"context"
	"errors"
	"leguiburger/internal/models"
	"leguiburger/internal/tenants"
	"strings"
)

var (
	ErrProductNotFound          = errors.New("producto no encontrado")
	ErrDuplicateProductName     = errors.New("ya existe un producto con ese nombre para este comercio")
	ErrInvalidProductData       = errors.New("el nombre del producto es obligatorio")
	ErrInvalidProductPrice      = errors.New("el precio del producto no puede ser negativo")
	ErrInvalidProductStock      = errors.New("el stock del producto no puede ser negativo")
	ErrTenantNotFoundForProduct = errors.New("el comercio (tenant) especificado no existe")
)

type Service interface {
	CreateProduct(ctx context.Context, tenantID, name, description string, currentPrice float64, currentStock int, trackStock *bool, imageURL string) (*models.Product, error)
	GetProduct(ctx context.Context, tenantID, id string) (*models.Product, error)
	ListProducts(ctx context.Context, tenantID string) ([]models.Product, error)
	UpdateProduct(ctx context.Context, tenantID, id, name, description string, currentPrice *float64, currentStock *int, trackStock *bool, imageURL string, isActive *bool) (*models.Product, error)
	DeleteProduct(ctx context.Context, tenantID, id string) error
}

type service struct {
	repo       Repository
	tenantRepo tenants.Repository
}

func NewService(repo Repository, tenantRepo tenants.Repository) Service {
	return &service{
		repo:       repo,
		tenantRepo: tenantRepo,
	}
}

func (s *service) CreateProduct(ctx context.Context, tenantID, name, description string, currentPrice float64, currentStock int, trackStock *bool, imageURL string) (*models.Product, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return nil, ErrInvalidProductData
	}
	if currentPrice < 0 {
		return nil, ErrInvalidProductPrice
	}
	if currentStock < 0 {
		return nil, ErrInvalidProductStock
	}

	existing, err := s.repo.GetByName(ctx, tenantID, cleanName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateProductName
	}

	shouldTrackStock := true
	if trackStock != nil {
		shouldTrackStock = *trackStock
	}

	product := &models.Product{
		TenantID:     tenantID,
		Name:         cleanName,
		Description:  strings.TrimSpace(description),
		CurrentPrice: currentPrice,
		CurrentStock: currentStock,
		TrackStock:   shouldTrackStock,
		ImageURL:     strings.TrimSpace(imageURL),
		IsActive:     true,
	}

	if err := s.repo.Create(ctx, product); err != nil {
		if strings.Contains(err.Error(), "23503") || strings.Contains(err.Error(), "products_tenant_id_fkey") {
			return nil, ErrTenantNotFoundForProduct
		}
		return nil, err
	}

	return product, nil
}

func (s *service) GetProduct(ctx context.Context, tenantID, id string) (*models.Product, error) {
	product, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}
	return product, nil
}

func (s *service) ListProducts(ctx context.Context, tenantID string) ([]models.Product, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFoundForProduct
	}

	return s.repo.FetchAll(ctx, tenantID)
}

func (s *service) UpdateProduct(ctx context.Context, tenantID, id, name, description string, currentPrice *float64, currentStock *int, trackStock *bool, imageURL string, isActive *bool) (*models.Product, error) {
	product, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	if currentPrice != nil {
		if *currentPrice < 0 {
			return nil, ErrInvalidProductPrice
		}
		product.CurrentPrice = *currentPrice
	}
	if currentStock != nil {
		if *currentStock < 0 {
			return nil, ErrInvalidProductStock
		}
		product.CurrentStock = *currentStock
	}
	if name != "" {
		cleanName := strings.TrimSpace(name)
		if cleanName == "" {
			return nil, ErrInvalidProductData
		}
		if cleanName != product.Name {
			existing, err := s.repo.GetByName(ctx, tenantID, cleanName)
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
	if trackStock != nil {
		product.TrackStock = *trackStock
	}
	if isActive != nil {
		product.IsActive = *isActive
	}

	if err := s.repo.Update(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (s *service) DeleteProduct(ctx context.Context, tenantID, id string) error {
	product, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if product == nil {
		return ErrProductNotFound
	}

	return s.repo.Delete(ctx, tenantID, id)
}
