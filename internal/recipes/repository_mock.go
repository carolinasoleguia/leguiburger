package recipes

import (
	"context"
	"leguiburger/internal/models"
)

type mockRepository struct {
	createFunc                 func(ctx context.Context, recipe *models.Recipe) error
	getByIDFunc                func(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error)
	fetchAllFunc               func(ctx context.Context, tenantID string) ([]models.Recipe, error)
	updateFunc                 func(ctx context.Context, recipe *models.Recipe) error
	deleteFunc                 func(ctx context.Context, tenantID, productID, supplyID string) error
	productExistsForTenantFunc func(ctx context.Context, tenantID, productID string) (bool, error)
	supplyExistsForTenantFunc  func(ctx context.Context, tenantID, supplyID string) (bool, error)
}

func (m *mockRepository) Create(ctx context.Context, recipe *models.Recipe) error {
	if m.createFunc == nil {
		return nil
	}
	return m.createFunc(ctx, recipe)
}

func (m *mockRepository) GetByID(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error) {
	if m.getByIDFunc == nil {
		return nil, nil
	}
	return m.getByIDFunc(ctx, tenantID, productID, supplyID)
}

func (m *mockRepository) FetchAll(ctx context.Context, tenantID string) ([]models.Recipe, error) {
	if m.fetchAllFunc == nil {
		return nil, nil
	}
	return m.fetchAllFunc(ctx, tenantID)
}

func (m *mockRepository) Update(ctx context.Context, recipe *models.Recipe) error {
	if m.updateFunc == nil {
		return nil
	}
	return m.updateFunc(ctx, recipe)
}

func (m *mockRepository) Delete(ctx context.Context, tenantID, productID, supplyID string) error {
	if m.deleteFunc == nil {
		return nil
	}
	return m.deleteFunc(ctx, tenantID, productID, supplyID)
}

func (m *mockRepository) ProductExistsForTenant(ctx context.Context, tenantID, productID string) (bool, error) {
	if m.productExistsForTenantFunc == nil {
		return true, nil
	}
	return m.productExistsForTenantFunc(ctx, tenantID, productID)
}

func (m *mockRepository) SupplyExistsForTenant(ctx context.Context, tenantID, supplyID string) (bool, error) {
	if m.supplyExistsForTenantFunc == nil {
		return true, nil
	}
	return m.supplyExistsForTenantFunc(ctx, tenantID, supplyID)
}
