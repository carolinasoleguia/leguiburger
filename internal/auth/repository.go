package auth

import (
	"context"
	"errors"

	"leguiburger/internal/db"
	"leguiburger/internal/models"

	"gorm.io/gorm"
)

type Repository interface {
	GetEmployeeByEmailAndTenant(ctx context.Context, tenantID, email string) (*models.Employee, error)
	GetEmployeeByEmail(ctx context.Context, email string) (*models.Employee, error)
	GetAllEmployeesByEmail(ctx context.Context, email string) ([]models.Employee, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetAllUsersByEmail(ctx context.Context, email string) ([]models.User, error)
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) GetEmployeeByEmailAndTenant(ctx context.Context, tenantID, email string) (*models.Employee, error) {
	var emp models.Employee
	err := db.DB.WithContext(ctx).
		Where("tenant_id = ? AND LOWER(email) = LOWER(?) AND is_active = true", tenantID, email).
		First(&emp).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &emp, nil
}

// GetByEmail busca un usuario en toda la base sin filtrar por tenant_id.
func (r *repository) GetEmployeeByEmail(ctx context.Context, email string) (*models.Employee, error) {
	var emp models.Employee
	err := db.DB.WithContext(ctx).
		Where("LOWER(email) = LOWER(?) AND is_active = true", email).
		First(&emp).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &emp, nil
}

func (r *repository) GetAllEmployeesByEmail(ctx context.Context, email string) ([]models.Employee, error) {
	var employees []models.Employee
	err := db.DB.WithContext(ctx).
		Where("LOWER(email) = LOWER(?) AND is_active = true", email).
		Find(&employees).Error

	if err != nil {
		return nil, err
	}
	return employees, nil
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := db.DB.WithContext(ctx).
		Where("LOWER(email) = LOWER(?) AND is_active = true", email).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *repository) GetAllUsersByEmail(ctx context.Context, email string) ([]models.User, error) {
	var users []models.User
	err := db.DB.WithContext(ctx).
		Where("LOWER(email) = LOWER(?) AND is_active = true", email).
		Find(&users).Error

	if err != nil {
		return nil, err
	}
	return users, nil
}
