package recipes

import (
	"context"
	"leguiburger/internal/models"
)

type MockRepository struct {
	OnCreate                 func(ctx context.Context, recipe *models.Recipe) error
	OnGetByID                func(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error)
	OnFetchAll               func(ctx context.Context, tenantID string) ([]models.Recipe, error)
	OnUpdate                 func(ctx context.Context, recipe *models.Recipe) error
	OnDelete                 func(ctx context.Context, tenantID, productID, supplyID string) error
	OnProductExistsForTenant func(ctx context.Context, tenantID, productID string) (bool, error)
	OnSupplyExistsForTenant  func(ctx context.Context, tenantID, supplyID string) (bool, error)
}

func (m *MockRepository) Create(ctx context.Context, recipe *models.Recipe) error {
	if m.OnCreate == nil {
		return nil
	}
	return m.OnCreate(ctx, recipe)
}

func (m *MockRepository) GetByID(ctx context.Context, tenantID, productID, supplyID string) (*models.Recipe, error) {
	if m.OnGetByID == nil {
		return nil, nil
	}
	return m.OnGetByID(ctx, tenantID, productID, supplyID)
}

func (m *MockRepository) FetchAll(ctx context.Context, tenantID string) ([]models.Recipe, error) {
	if m.OnFetchAll == nil {
		return nil, nil
	}
	return m.OnFetchAll(ctx, tenantID)
}

func (m *MockRepository) Update(ctx context.Context, recipe *models.Recipe) error {
	if m.OnUpdate == nil {
		return nil
	}
	return m.OnUpdate(ctx, recipe)
}

func (m *MockRepository) Delete(ctx context.Context, tenantID, productID, supplyID string) error {
	if m.OnDelete == nil {
		return nil
	}
	return m.OnDelete(ctx, tenantID, productID, supplyID)
}

func (m *MockRepository) ProductExistsForTenant(ctx context.Context, tenantID, productID string) (bool, error) {
	if m.OnProductExistsForTenant == nil {
		return true, nil
	}
	return m.OnProductExistsForTenant(ctx, tenantID, productID)
}

func (m *MockRepository) SupplyExistsForTenant(ctx context.Context, tenantID, supplyID string) (bool, error) {
	if m.OnSupplyExistsForTenant == nil {
		return true, nil
	}
	return m.OnSupplyExistsForTenant(ctx, tenantID, supplyID)
}
