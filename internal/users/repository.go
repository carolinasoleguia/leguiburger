package users

import (
	"context"
	"errors"
	"leguiburger/internal/db"
	"leguiburger/internal/models"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetAll(ctx context.Context) ([]models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id string) error
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, user *models.User) error {
	return db.DB.WithContext(ctx).Create(user).Error
}

func (r *repository) GetByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := db.DB.WithContext(ctx).First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := db.DB.WithContext(ctx).Where("LOWER(email) = LOWER(?)", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *repository) GetAll(ctx context.Context) ([]models.User, error) {
	var users []models.User
	err := db.DB.WithContext(ctx).Order("created_at DESC").Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *repository) Update(ctx context.Context, user *models.User) error {
	return db.DB.WithContext(ctx).Save(user).Error
}

func (r *repository) Delete(ctx context.Context, id string) error {
	return db.DB.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Update("is_active", false).Error
}
