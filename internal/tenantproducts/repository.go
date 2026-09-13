package tenantproducts

import (
	"context"
	"errors"
	"leguiburger/internal/db"
	"leguiburger/internal/models"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, tenantProduct *models.TenantProduct) error
	GetByID(ctx context.Context, tenantID, productID string) (*models.TenantProduct, error)
	FetchAll(ctx context.Context, tenantID string) ([]models.TenantProduct, error)
	Update(ctx context.Context, tenantProduct *models.TenantProduct) error
	Delete(ctx context.Context, tenantID, productID string) error
	GetProductForTenantBrand(ctx context.Context, tenantID, productID string) (*models.Product, error)
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, tenantProduct *models.TenantProduct) error {
	return db.DB.WithContext(ctx).Create(tenantProduct).Error
}

func (r *repository) GetByID(ctx context.Context, tenantID, productID string) (*models.TenantProduct, error) {
	var tenantProduct models.TenantProduct
	err := db.DB.WithContext(ctx).
		Preload("Product").
		Where("tenant_id = ? AND product_id = ?", tenantID, productID).
		First(&tenantProduct).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tenantProduct, nil
}

func (r *repository) FetchAll(ctx context.Context, tenantID string) ([]models.TenantProduct, error) {
	var tenantProducts []models.TenantProduct
	err := db.DB.WithContext(ctx).
		Preload("Product").
		Where("tenant_id = ?", tenantID).
		Find(&tenantProducts).Error
	return tenantProducts, err
}

func (r *repository) Update(ctx context.Context, tenantProduct *models.TenantProduct) error {
	return db.DB.WithContext(ctx).Save(tenantProduct).Error
}

func (r *repository) Delete(ctx context.Context, tenantID, productID string) error {
	return db.DB.WithContext(ctx).
		Where("tenant_id = ? AND product_id = ?", tenantID, productID).
		Delete(&models.TenantProduct{}).Error
}

func (r *repository) GetProductForTenantBrand(ctx context.Context, tenantID, productID string) (*models.Product, error) {
	var product models.Product
	err := db.DB.WithContext(ctx).
		Table("products").
		Joins("JOIN tenants ON tenants.brand_id = products.brand_id").
		Where("tenants.id = ? AND products.id = ?", tenantID, productID).
		First(&product).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}
