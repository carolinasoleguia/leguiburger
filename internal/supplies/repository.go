package supplies

import (
	"context"
	"errors"

	"leguiburger/internal/db"
	"leguiburger/internal/models"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, supply *models.Supply) error
	GetByID(ctx context.Context, tenantID, id string) (*models.Supply, error)
	GetByName(ctx context.Context, tenantID, name string) (*models.Supply, error)
	FetchAll(ctx context.Context, tenantID string) ([]models.Supply, error)
	Update(ctx context.Context, supply *models.Supply) error
	Delete(ctx context.Context, tenantID, id string) error
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, supply *models.Supply) error {
	return db.DB.WithContext(ctx).Create(supply).Error
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (*models.Supply, error) {
	var supply models.Supply
	err := db.DB.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&supply).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &supply, nil
}

func (r *repository) GetByName(ctx context.Context, tenantID, name string) (*models.Supply, error) {
	var supply models.Supply
	err := db.DB.WithContext(ctx).Where("tenant_id = ? AND name = ?", tenantID, name).First(&supply).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &supply, nil
}

func (r *repository) FetchAll(ctx context.Context, tenantID string) ([]models.Supply, error) {
	var supplies []models.Supply
	err := db.DB.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&supplies).Error
	return supplies, err
}

func (r *repository) Update(ctx context.Context, supply *models.Supply) error {
	return db.DB.WithContext(ctx).Save(supply).Error
}

func (r *repository) Delete(ctx context.Context, tenantID, id string) error {
	return db.DB.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&models.Supply{}).Error
}
