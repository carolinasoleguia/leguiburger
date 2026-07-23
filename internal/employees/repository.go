package employees

import (
	"context"
	"errors"
	"leguiburger/internal/db"
	"leguiburger/internal/models"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, employee *models.Employee) error
	GetByID(ctx context.Context, tenantID, id string) (*models.Employee, error)
	GetByEmail(ctx context.Context, tenantID, email string) (*models.Employee, error)
	FetchAll(ctx context.Context, tenantID string) ([]models.Employee, error)
	GetAll(ctx context.Context) ([]models.Employee, error) // <--- Agregado a la interfaz
	Update(ctx context.Context, employee *models.Employee) error
	Delete(ctx context.Context, tenantID, id string) error
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, employee *models.Employee) error {
	return db.DB.WithContext(ctx).Create(employee).Error
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (*models.Employee, error) {
	var employee models.Employee
	err := db.DB.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&employee).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &employee, nil
}

func (r *repository) GetByEmail(ctx context.Context, tenantID, email string) (*models.Employee, error) {
	var employee models.Employee
	err := db.DB.WithContext(ctx).Where("email = ?", email).First(&employee).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &employee, nil
}

func (r *repository) FetchAll(ctx context.Context, tenantID string) ([]models.Employee, error) {
	var employees []models.Employee
	err := db.DB.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&employees).Error
	return employees, err
}

func (r *repository) GetAll(ctx context.Context) ([]models.Employee, error) {
	var employees []models.Employee

	err := db.DB.WithContext(ctx).
		Where("tenant_id IS NOT NULL").
		Find(&employees).Error
	if err != nil {
		return nil, err
	}

	return employees, nil
}

func (r *repository) Update(ctx context.Context, employee *models.Employee) error {
	return db.DB.WithContext(ctx).Save(employee).Error
}

func (r *repository) Delete(ctx context.Context, tenantID, id string) error {
	return db.DB.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&models.Employee{}).Error
}
