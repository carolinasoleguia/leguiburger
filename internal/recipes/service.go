package recipes

import (
	"context"
	"errors"
	"leguiburger/internal/models"
	"leguiburger/internal/tenants"
	"strings"
)

var (
	ErrRecipeNotFound           = errors.New("receta no encontrada")
	ErrDuplicateRecipe          = errors.New("ya existe una receta para ese producto e insumo")
	ErrInvalidRecipeData        = errors.New("product_id y supply_id son obligatorios")
	ErrInvalidRecipeQuantity    = errors.New("la cantidad usada debe ser mayor a cero")
	ErrProductNotFoundForRecipe = errors.New("el producto especificado no existe para este comercio")
	ErrSupplyNotFoundForRecipe  = errors.New("el insumo especificado no existe para este comercio")
	ErrTenantNotFoundForRecipe  = errors.New("el comercio (tenant) especificado no existe")
)

type Service interface {
	CreateRecipe(ctx context.Context, tenantID, productID, supplyID string, quantityUsed float64) (*models.Recipe, error)
	GetRecipe(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error)
	ListRecipes(ctx context.Context, tenantID string) ([]models.Recipe, error)
	UpdateRecipe(ctx context.Context, tenantID, productID, supplyID string, quantityUsed float64) (*models.Recipe, error)
	DeleteRecipe(ctx context.Context, tenantID, productID, supplyID string) error
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

func (s *service) CreateRecipe(ctx context.Context, tenantID, productID, supplyID string, quantityUsed float64) (*models.Recipe, error) {
	cleanProductID := strings.TrimSpace(productID)
	cleanSupplyID := strings.TrimSpace(supplyID)

	if cleanProductID == "" || cleanSupplyID == "" {
		return nil, ErrInvalidRecipeData
	}
	if quantityUsed <= 0 {
		return nil, ErrInvalidRecipeQuantity
	}

	if err := s.validateProductAndSupply(ctx, tenantID, cleanProductID, cleanSupplyID); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByID(ctx, tenantID, cleanProductID, cleanSupplyID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateRecipe
	}

	recipe := &models.Recipe{
		ProductID:    cleanProductID,
		SupplyID:     cleanSupplyID,
		QuantityUsed: quantityUsed,
	}

	if err := s.repo.Create(ctx, recipe); err != nil {
		return nil, err
	}

	return recipe, nil
}

func (s *service) GetRecipe(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error) {
	recipe, err := s.repo.GetByID(ctx, tenantID, strings.TrimSpace(productID), strings.TrimSpace(supplyID))
	if err != nil {
		return nil, err
	}
	if recipe == nil {
		return nil, ErrRecipeNotFound
	}
	return recipe, nil
}

func (s *service) ListRecipes(ctx context.Context, tenantID string) ([]models.Recipe, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFoundForRecipe
	}

	return s.repo.FetchAll(ctx, tenantID)
}

func (s *service) UpdateRecipe(ctx context.Context, tenantID, productID, supplyID string, quantityUsed float64) (*models.Recipe, error) {
	if quantityUsed <= 0 {
		return nil, ErrInvalidRecipeQuantity
	}

	recipe, err := s.repo.GetByID(ctx, tenantID, strings.TrimSpace(productID), strings.TrimSpace(supplyID))
	if err != nil {
		return nil, err
	}
	if recipe == nil {
		return nil, ErrRecipeNotFound
	}

	recipe.QuantityUsed = quantityUsed

	if err := s.repo.Update(ctx, recipe); err != nil {
		return nil, err
	}

	return recipe, nil
}

func (s *service) DeleteRecipe(ctx context.Context, tenantID, productID, supplyID string) error {
	recipe, err := s.repo.GetByID(ctx, tenantID, strings.TrimSpace(productID), strings.TrimSpace(supplyID))
	if err != nil {
		return err
	}
	if recipe == nil {
		return ErrRecipeNotFound
	}

	return s.repo.Delete(ctx, tenantID, recipe.ProductID, recipe.SupplyID)
}

func (s *service) validateProductAndSupply(ctx context.Context, tenantID, productID, supplyID string) error {
	productExists, err := s.repo.ProductExistsForTenant(ctx, tenantID, productID)
	if err != nil {
		return err
	}
	if !productExists {
		return ErrProductNotFoundForRecipe
	}

	supplyExists, err := s.repo.SupplyExistsForTenant(ctx, tenantID, supplyID)
	if err != nil {
		return err
	}
	if !supplyExists {
		return ErrSupplyNotFoundForRecipe
	}

	return nil
}
