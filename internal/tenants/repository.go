package tenants

import (
	"context"
	"errors"
	"leguiburger/internal/db"
	"leguiburger/internal/models"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, tenant *models.Tenant) error
	GetByID(ctx context.Context, id string) (*models.Tenant, error)
	GetByTaxID(ctx context.Context, taxId string) (*models.Tenant, error)
	GetBySubdomain(ctx context.Context, subdomain string) (*models.Tenant, error)
	GetByNameAndSubdomain(ctx context.Context, name string, subdomain string) (*models.Tenant, error)
	Update(ctx context.Context, tenant *models.Tenant) error
	Delete(ctx context.Context, id string) error
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, tenant *models.Tenant) error {
	return db.DB.WithContext(ctx).Create(tenant).Error
}

func (r *repository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	var tenant models.Tenant
	err := db.DB.WithContext(ctx).First(&tenant, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tenant, nil
}

func (r *repository) GetBySubdomain(ctx context.Context, subdomain string) (*models.Tenant, error) {
	var tenant models.Tenant
	err := db.DB.WithContext(ctx).First(&tenant, "subdomain = ?", subdomain).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *repository) GetByTaxID(ctx context.Context, taxID string) (*models.Tenant, error) {
	var tenant models.Tenant
	err := db.DB.WithContext(ctx).First(&tenant, "tax_id = ?", taxID).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *repository) GetByNameAndSubdomain(ctx context.Context, name string, subdomain string) (*models.Tenant, error) {
	var tenant models.Tenant

	// Buscamos usando "name" que ya existe en tu DB 🎉
	err := db.DB.WithContext(ctx).
		Where("name = ? AND subdomain = ?", name, subdomain).
		First(&tenant).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &tenant, nil
}

func (r *repository) Update(ctx context.Context, tenant *models.Tenant) error {
	return db.DB.WithContext(ctx).Save(tenant).Error
}

func (r *repository) Delete(ctx context.Context, id string) error {
	// Hacemos una eliminación lógica (Soft Delete) pasando active a false
	// para no perder la integridad referencial de los empleados/pedidos históricos.
	return db.DB.WithContext(ctx).Model(&models.Tenant{}).Where("id = ?", id).Update("active", false).Error
}
