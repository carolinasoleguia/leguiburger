package shipping

import (
	"context"
	"errors"
	"leguiburger/internal/db"
	"leguiburger/internal/models"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, sm *models.ShippingMethod) error
	GetByID(ctx context.Context, tenantID, id string) (*models.ShippingMethod, error)
	GetByNameAndTypification(ctx context.Context, tenantID, name, typification string) (*models.ShippingMethod, error)
	FetchAll(ctx context.Context, tenantID string) ([]models.ShippingMethod, error)
	Update(ctx context.Context, sm *models.ShippingMethod) error
	Delete(ctx context.Context, tenantID, id string) error
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, sm *models.ShippingMethod) error {
	return db.DB.WithContext(ctx).Create(sm).Error
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (*models.ShippingMethod, error) {
	var sm models.ShippingMethod
	err := db.DB.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&sm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 💡 Retornamos nil seguro si no existe
		}
		return nil, err
	}
	return &sm, nil
}

func (r *repository) GetByName(ctx context.Context, tenantID, name string) (*models.ShippingMethod, error) {
	var sm models.ShippingMethod
	err := db.DB.WithContext(ctx).Where("tenant_id = ? AND name = ?", tenantID, name).First(&sm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 💡 Retornamos nil seguro si no existe
		}
		return nil, err
	}
	return &sm, nil
}

func (r *repository) GetByNameAndTypification(ctx context.Context, tenantID, name, typification string) (*models.ShippingMethod, error) {
	var sm models.ShippingMethod
	err := db.DB.WithContext(ctx).
		Where("tenant_id = ? AND name = ? AND typification = ?", tenantID, name, typification).
		First(&sm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sm, nil
}

func (r *repository) FetchAll(ctx context.Context, tenantID string) ([]models.ShippingMethod, error) {
	var methods []models.ShippingMethod
	err := db.DB.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&methods).Error
	return methods, err
}

func (r *repository) Update(ctx context.Context, sm *models.ShippingMethod) error {
	return db.DB.WithContext(ctx).Save(sm).Error
}

func (r *repository) Delete(ctx context.Context, tenantID, id string) error {
	return db.DB.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&models.ShippingMethod{}).Error
}
