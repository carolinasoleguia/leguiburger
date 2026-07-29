package tenantproducts

import (
	"context"
	"leguiburger/internal/models"
)

type mockRepository struct {
	createFunc                      func(ctx context.Context, tenantProduct *models.TenantProduct) error
	getByIDFunc                     func(ctx context.Context, tenantID, productID string) (*models.TenantProduct, error)
	fetchAllFunc                    func(ctx context.Context, tenantID string) ([]models.TenantProduct, error)
	updateFunc                      func(ctx context.Context, tenantProduct *models.TenantProduct) error
	deleteFunc                      func(ctx context.Context, tenantID, productID string) error
	productBelongsToTenantBrandFunc func(ctx context.Context, tenantID, productID string) (bool, error)
}

func (m *mockRepository) Create(ctx context.Context, tenantProduct *models.TenantProduct) error {
	if m.createFunc == nil {
		return nil
	}
	return m.createFunc(ctx, tenantProduct)
}

func (m *mockRepository) GetByID(ctx context.Context, tenantID, productID string) (*models.TenantProduct, error) {
	if m.getByIDFunc == nil {
		return nil, nil
	}
	return m.getByIDFunc(ctx, tenantID, productID)
}

func (m *mockRepository) FetchAll(ctx context.Context, tenantID string) ([]models.TenantProduct, error) {
	if m.fetchAllFunc == nil {
		return nil, nil
	}
	return m.fetchAllFunc(ctx, tenantID)
}

func (m *mockRepository) Update(ctx context.Context, tenantProduct *models.TenantProduct) error {
	if m.updateFunc == nil {
		return nil
	}
	return m.updateFunc(ctx, tenantProduct)
}

func (m *mockRepository) Delete(ctx context.Context, tenantID, productID string) error {
	if m.deleteFunc == nil {
		return nil
	}
	return m.deleteFunc(ctx, tenantID, productID)
}

func (m *mockRepository) ProductBelongsToTenantBrand(ctx context.Context, tenantID, productID string) (bool, error) {
	if m.productBelongsToTenantBrandFunc == nil {
		return true, nil
	}
	return m.productBelongsToTenantBrandFunc(ctx, tenantID, productID)
}
