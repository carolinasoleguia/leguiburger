package tenantproducts

import (
	"context"
	"errors"
	"leguiburger/internal/models"
	"leguiburger/internal/tenants"
	"strings"
)

var (
	ErrTenantProductNotFound    = errors.New("producto no encontrado para este comercio")
	ErrDuplicateTenantProduct   = errors.New("este producto ya está asignado a este comercio")
	ErrInvalidTenantProductData = errors.New("product_id es obligatorio")
	ErrInvalidPriceOverride     = errors.New("el precio local no puede ser negativo")
	ErrInvalidTenantStock       = errors.New("el stock local no puede ser negativo")
	ErrProductBrandMismatch     = errors.New("el producto no pertenece a la marca de este comercio")
	ErrInactiveBaseProduct      = errors.New("el producto está inactivo en el catálogo general")
	ErrTenantNotFound           = errors.New("el comercio (tenant) especificado no existe")
)

type Service interface {
	CreateTenantProduct(ctx context.Context, tenantID, productID string, priceOverride *float64, currentStock int, trackStock, isAvailable *bool) (*models.TenantProduct, error)
	GetTenantProduct(ctx context.Context, tenantID, productID string) (*models.TenantProduct, error)
	ListTenantProducts(ctx context.Context, tenantID string) ([]models.TenantProduct, error)
	UpdateTenantProduct(ctx context.Context, tenantID, productID string, priceOverride *float64, currentStock *int, trackStock, isAvailable, isActive *bool) (*models.TenantProduct, error)
	DeleteTenantProduct(ctx context.Context, tenantID, productID string) error
}

type service struct {
	repo       Repository
	tenantRepo tenants.Repository
}

func NewService(repo Repository, tenantRepo tenants.Repository) Service {
	return &service{repo: repo, tenantRepo: tenantRepo}
}

func (s *service) CreateTenantProduct(ctx context.Context, tenantID, productID string, priceOverride *float64, currentStock int, trackStock, isAvailable *bool) (*models.TenantProduct, error) {
	cleanProductID := strings.TrimSpace(productID)
	if cleanProductID == "" {
		return nil, ErrInvalidTenantProductData
	}
	if priceOverride != nil && *priceOverride < 0 {
		return nil, ErrInvalidPriceOverride
	}
	if currentStock < 0 {
		return nil, ErrInvalidTenantStock
	}
	if err := s.validateTenantAndProduct(ctx, tenantID, cleanProductID); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByID(ctx, tenantID, cleanProductID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateTenantProduct
	}

	shouldTrackStock := true
	if trackStock != nil {
		shouldTrackStock = *trackStock
	}
	available := true
	if isAvailable != nil {
		available = *isAvailable
	}

	tenantProduct := &models.TenantProduct{
		TenantID:      tenantID,
		ProductID:     cleanProductID,
		PriceOverride: priceOverride,
		CurrentStock:  currentStock,
		TrackStock:    shouldTrackStock,
		IsAvailable:   available,
		IsActive:      true,
	}

	if err := s.repo.Create(ctx, tenantProduct); err != nil {
		return nil, err
	}
	return tenantProduct, nil
}

func (s *service) GetTenantProduct(ctx context.Context, tenantID, productID string) (*models.TenantProduct, error) {
	tenantProduct, err := s.repo.GetByID(ctx, tenantID, strings.TrimSpace(productID))
	if err != nil {
		return nil, err
	}
	if tenantProduct == nil {
		return nil, ErrTenantProductNotFound
	}
	return tenantProduct, nil
}

func (s *service) ListTenantProducts(ctx context.Context, tenantID string) ([]models.TenantProduct, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}
	return s.repo.FetchAll(ctx, tenantID)
}

func (s *service) UpdateTenantProduct(ctx context.Context, tenantID, productID string, priceOverride *float64, currentStock *int, trackStock, isAvailable, isActive *bool) (*models.TenantProduct, error) {
	tenantProduct, err := s.repo.GetByID(ctx, tenantID, strings.TrimSpace(productID))
	if err != nil {
		return nil, err
	}
	if tenantProduct == nil {
		return nil, ErrTenantProductNotFound
	}
	if priceOverride != nil {
		if *priceOverride < 0 {
			return nil, ErrInvalidPriceOverride
		}
		tenantProduct.PriceOverride = priceOverride
	}
	if currentStock != nil {
		if *currentStock < 0 {
			return nil, ErrInvalidTenantStock
		}
		tenantProduct.CurrentStock = *currentStock
	}
	if trackStock != nil {
		tenantProduct.TrackStock = *trackStock
	}
	if isAvailable != nil {
		tenantProduct.IsAvailable = *isAvailable
	}
	if isActive != nil {
		tenantProduct.IsActive = *isActive
	}

	if err := s.repo.Update(ctx, tenantProduct); err != nil {
		return nil, err
	}
	return tenantProduct, nil
}

func (s *service) DeleteTenantProduct(ctx context.Context, tenantID, productID string) error {
	tenantProduct, err := s.repo.GetByID(ctx, tenantID, strings.TrimSpace(productID))
	if err != nil {
		return err
	}
	if tenantProduct == nil {
		return ErrTenantProductNotFound
	}
	return s.repo.Delete(ctx, tenantID, tenantProduct.ProductID)
}

func (s *service) validateTenantAndProduct(ctx context.Context, tenantID, productID string) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant == nil {
		return ErrTenantNotFound
	}
	product, err := s.repo.GetProductForTenantBrand(ctx, tenantID, productID)
	if err != nil {
		return err
	}
	if product == nil {
		return ErrProductBrandMismatch
	}
	if !product.IsActive {
		return ErrInactiveBaseProduct
	}
	return nil
}
