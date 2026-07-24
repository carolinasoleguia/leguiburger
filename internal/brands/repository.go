package brands

import (
	"context"
	"errors"
	"leguiburger/internal/db"
	"leguiburger/internal/models"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, brand *models.Brand) error
	GetByID(ctx context.Context, id string) (*models.Brand, error)
	GetAll(ctx context.Context) ([]models.Brand, error)
	GetByName(ctx context.Context, name string) (*models.Brand, error)
	Update(ctx context.Context, brand *models.Brand) error
	Delete(ctx context.Context, id string) error
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, brand *models.Brand) error {
	return db.DB.WithContext(ctx).Create(brand).Error
}

func (r *repository) GetByID(ctx context.Context, id string) (*models.Brand, error) {
	var brand models.Brand

	err := db.DB.WithContext(ctx).
		Where("id = ?", id).
		First(&brand).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &brand, nil
}

func (r *repository) GetAll(ctx context.Context) ([]models.Brand, error) {
	var brands []models.Brand

	err := db.DB.WithContext(ctx).
		Order("name ASC").
		Find(&brands).Error

	return brands, err
}

func (r *repository) GetByName(ctx context.Context, name string) (*models.Brand, error) {
	var brand models.Brand

	err := db.DB.WithContext(ctx).
		Where("name = ?", name).
		First(&brand).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &brand, nil
}

func (r *repository) Update(ctx context.Context, brand *models.Brand) error {
	return db.DB.WithContext(ctx).Save(brand).Error
}

func (r *repository) Delete(ctx context.Context, id string) error {
	return db.DB.WithContext(ctx).
		Delete(&models.Brand{}, id).Error
}
