package auth

import (
	"context"
	"errors"

	"leguiburger/internal/db"
	"leguiburger/internal/models"

	"gorm.io/gorm"
)

type Repository interface {
	GetByEmailAndTenant(ctx context.Context, tenantID, email string) (*models.Employee, error)
	GetByEmail(ctx context.Context, email string) (*models.Employee, error)
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) GetByEmailAndTenant(ctx context.Context, tenantID, email string) (*models.Employee, error) {
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
func (r *repository) GetByEmail(ctx context.Context, email string) (*models.Employee, error) {
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
