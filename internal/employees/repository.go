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
	FetchByBrandID(ctx context.Context, brandID string) ([]models.Employee, error)
	GetAll(ctx context.Context) ([]models.Employee, error)
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
	query := db.DB.WithContext(ctx).
		Preload("Tenant").
		Preload("Tenant.Brand").
		Where("id = ?", id)
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	err := query.First(&employee).Error
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
	query := db.DB.WithContext(ctx).Where("email = ?", email)
	if tenantID == "" {
		query = query.Where("tenant_id IS NULL")
	} else {
		query = query.Where("tenant_id = ?", tenantID)
	}
	err := query.First(&employee).Error
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
	err := db.DB.WithContext(ctx).
		Preload("Tenant").
		Preload("Tenant.Brand").
		Where("tenant_id = ?", tenantID).
		Find(&employees).
		Error
	return employees, err
}

func (r *repository) FetchByBrandID(ctx context.Context, brandID string) ([]models.Employee, error) {
	var employees []models.Employee

	err := db.DB.WithContext(ctx).
		Model(&models.Employee{}).
		Preload("Tenant", func(tx *gorm.DB) *gorm.DB {
			return tx.Select("id", "subdomain", "brand_id")
		}).
		Preload("Tenant.Brand").
		Joins("JOIN tenants ON tenants.id = employees.tenant_id").
		Where("tenants.brand_id = ?", brandID).
		Find(&employees).
		Error

	return employees, err
}

func (r *repository) GetAll(ctx context.Context) ([]models.Employee, error) {
	var employees []models.Employee

	err := db.DB.WithContext(ctx).
		Preload("Tenant").
		Preload("Tenant.Brand").
		Where("tenant_id IS NOT NULL").
		Find(&employees).
		Error
	if err != nil {
		return nil, err
	}

	return employees, nil
}

func (r *repository) Update(ctx context.Context, employee *models.Employee) error {
	return db.DB.WithContext(ctx).Save(employee).Error
}

func (r *repository) Delete(ctx context.Context, tenantID, id string) error {
	query := db.DB.WithContext(ctx).Model(&models.Employee{}).Where("id = ?", id)
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	return query.Update("is_active", false).Error
}
