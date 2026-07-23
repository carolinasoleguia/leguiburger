package extras

import (
	"context"
	"errors"
	"leguiburger/internal/db"
	"leguiburger/internal/models"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, extra *models.Extra) error
	GetByID(ctx context.Context, tenantID, id string) (*models.Extra, error)
	GetByName(ctx context.Context, tenantID, name string) (*models.Extra, error)
	FetchAll(ctx context.Context, tenantID string) ([]models.Extra, error)
	Update(ctx context.Context, extra *models.Extra) error
	Delete(ctx context.Context, tenantID, id string) error
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, extra *models.Extra) error {
	return db.DB.WithContext(ctx).Create(extra).Error
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (*models.Extra, error) {
	var extra models.Extra
	err := db.DB.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&extra).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &extra, nil
}

func (r *repository) GetByName(ctx context.Context, tenantID, name string) (*models.Extra, error) {
	var extra models.Extra
	err := db.DB.WithContext(ctx).Where("tenant_id = ? AND name = ?", tenantID, name).First(&extra).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &extra, nil
}

func (r *repository) FetchAll(ctx context.Context, tenantID string) ([]models.Extra, error) {
	var extras []models.Extra
	err := db.DB.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&extras).Error
	return extras, err
}

func (r *repository) Update(ctx context.Context, extra *models.Extra) error {
	return db.DB.WithContext(ctx).Save(extra).Error
}

func (r *repository) Delete(ctx context.Context, tenantID, id string) error {
	return db.DB.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&models.Extra{}).Error
}
