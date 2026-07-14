package tenants

import (
	"context"
	"leguiburger/internal/db"
	"leguiburger/internal/models"
)

type Repository interface {
	Create(ctx context.Context, tenant *models.Tenant) error
	GetByID(ctx context.Context, id string) (*models.Tenant, error)
	GetBySubdomain(ctx context.Context, subdomain string) (*models.Tenant, error)
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

func (r *repository) Update(ctx context.Context, tenant *models.Tenant) error {
	return db.DB.WithContext(ctx).Save(tenant).Error
}

func (r *repository) Delete(ctx context.Context, id string) error {
	// Hacemos una eliminación lógica (Soft Delete) pasando active a false
	// para no perder la integridad referencial de los empleados/pedidos históricos.
	return db.DB.WithContext(ctx).Model(&models.Tenant{}).Where("id = ?", id).Update("active", false).Error
}
