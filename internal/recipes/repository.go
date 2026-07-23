package recipes

import (
	"context"
	"errors"
	"leguiburger/internal/db"
	"leguiburger/internal/models"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, recipe *models.Recipe) error
	GetByID(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error)
	FetchAll(ctx context.Context, tenantID string) ([]models.Recipe, error)
	Update(ctx context.Context, recipe *models.Recipe) error
	Delete(ctx context.Context, tenantID, productID, supplyID string) error
	ProductExistsForTenant(ctx context.Context, tenantID, productID string) (bool, error)
	SupplyExistsForTenant(ctx context.Context, tenantID, supplyID string) (bool, error)
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, recipe *models.Recipe) error {
	return db.DB.WithContext(ctx).Create(recipe).Error
}

func (r *repository) GetByID(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error) {
	var recipe models.Recipe
	err := db.DB.WithContext(ctx).
		Table("recipes").
		Select("recipes.*").
		Joins("JOIN products ON products.id = recipes.product_id").
		Joins("JOIN supplies ON supplies.id = recipes.supply_id").
		Where("products.tenant_id = ? AND supplies.tenant_id = ? AND recipes.product_id = ? AND recipes.supply_id = ?", tenantID, tenantID, productID, supplyID).
		First(&recipe).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &recipe, nil
}

func (r *repository) FetchAll(ctx context.Context, tenantID string) ([]models.Recipe, error) {
	var recipes []models.Recipe
	err := db.DB.WithContext(ctx).
		Table("recipes").
		Select("recipes.*").
		Joins("JOIN products ON products.id = recipes.product_id").
		Joins("JOIN supplies ON supplies.id = recipes.supply_id").
		Where("products.tenant_id = ? AND supplies.tenant_id = ?", tenantID, tenantID).
		Find(&recipes).Error
	return recipes, err
}

func (r *repository) Update(ctx context.Context, recipe *models.Recipe) error {
	return db.DB.WithContext(ctx).Save(recipe).Error
}

func (r *repository) Delete(ctx context.Context, tenantID, productID, supplyID string) error {
	return db.DB.WithContext(ctx).
		Where("product_id = ? AND supply_id = ?", productID, supplyID).
		Delete(&models.Recipe{}).Error
}

func (r *repository) ProductExistsForTenant(ctx context.Context, tenantID, productID string) (bool, error) {
	var count int64
	err := db.DB.WithContext(ctx).
		Model(&models.Product{}).
		Where("tenant_id = ? AND id = ?", tenantID, productID).
		Count(&count).Error
	return count > 0, err
}

func (r *repository) SupplyExistsForTenant(ctx context.Context, tenantID, supplyID string) (bool, error) {
	var count int64
	err := db.DB.WithContext(ctx).
		Model(&models.Supply{}).
		Where("tenant_id = ? AND id = ?", tenantID, supplyID).
		Count(&count).Error
	return count > 0, err
}
