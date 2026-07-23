package products

import (
	"context"
	"errors"
	"leguiburger/internal/db"
	"leguiburger/internal/models"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, product *models.Product) error
	GetByID(ctx context.Context, tenantID, id string) (*models.Product, error)
	GetByName(ctx context.Context, tenantID, name string) (*models.Product, error)
	FetchAll(ctx context.Context, tenantID string) ([]models.Product, error)
	Update(ctx context.Context, product *models.Product) error
	Delete(ctx context.Context, tenantID, id string) error
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, product *models.Product) error {
	return db.DB.WithContext(ctx).Create(product).Error
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (*models.Product, error) {
	var product models.Product
	err := db.DB.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&product).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

func (r *repository) GetByName(ctx context.Context, tenantID, name string) (*models.Product, error) {
	var product models.Product
	err := db.DB.WithContext(ctx).Where("tenant_id = ? AND name = ?", tenantID, name).First(&product).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

func (r *repository) FetchAll(ctx context.Context, tenantID string) ([]models.Product, error) {
	var products []models.Product
	err := db.DB.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&products).Error
	return products, err
}

func (r *repository) Update(ctx context.Context, product *models.Product) error {
	return db.DB.WithContext(ctx).Save(product).Error
}

func (r *repository) Delete(ctx context.Context, tenantID, id string) error {
	return db.DB.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&models.Product{}).Error
}
