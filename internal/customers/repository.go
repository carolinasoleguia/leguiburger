package customers

import (
	"context"
	"errors"
	"leguiburger/internal/db"
	"leguiburger/internal/models"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, customer *models.Customer) error
	GetByID(ctx context.Context, tenantID, id string) (*models.Customer, error)
	GetByEmail(ctx context.Context, tenantID, email string) (*models.Customer, error)
	FetchAll(ctx context.Context, tenantID string) ([]models.Customer, error)
	Update(ctx context.Context, customer *models.Customer) error
	Delete(ctx context.Context, tenantID, id string) error
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, customer *models.Customer) error {
	return db.DB.WithContext(ctx).Create(customer).Error
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (*models.Customer, error) {
	var customer models.Customer
	err := db.DB.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&customer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}

func (r *repository) GetByEmail(ctx context.Context, tenantID, email string) (*models.Customer, error) {
	var customer models.Customer
	err := db.DB.WithContext(ctx).Where("tenant_id = ? AND email = ?", tenantID, email).First(&customer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}

func (r *repository) FetchAll(ctx context.Context, tenantID string) ([]models.Customer, error) {
	var customers []models.Customer
	err := db.DB.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&customers).Error
	return customers, err
}

func (r *repository) Update(ctx context.Context, customer *models.Customer) error {
	return db.DB.WithContext(ctx).Save(customer).Error
}

func (r *repository) Delete(ctx context.Context, tenantID, id string) error {
	return db.DB.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&models.Customer{}).Error
}
