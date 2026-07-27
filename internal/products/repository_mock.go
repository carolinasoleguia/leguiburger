package products

import (
	"context"
	"leguiburger/internal/models"
)

type mockRepository struct {
	createFunc    func(ctx context.Context, product *models.Product) error
	getByIDFunc   func(ctx context.Context, tenantID, id string) (*models.Product, error)
	getByNameFunc func(ctx context.Context, tenantID, name string) (*models.Product, error)
	fetchAllFunc  func(ctx context.Context, tenantID string) ([]models.Product, error)
	updateFunc    func(ctx context.Context, product *models.Product) error
	deleteFunc    func(ctx context.Context, tenantID, id string) error
}

func (m *mockRepository) Create(ctx context.Context, product *models.Product) error {
	if m.createFunc == nil {
		return nil
	}
	return m.createFunc(ctx, product)
}

func (m *mockRepository) GetByID(ctx context.Context, tenantID, id string) (*models.Product, error) {
	if m.getByIDFunc == nil {
		return nil, nil
	}
	return m.getByIDFunc(ctx, tenantID, id)
}

func (m *mockRepository) GetByName(ctx context.Context, tenantID, name string) (*models.Product, error) {
	if m.getByNameFunc == nil {
		return nil, nil
	}
	return m.getByNameFunc(ctx, tenantID, name)
}

func (m *mockRepository) FetchAll(ctx context.Context, tenantID string) ([]models.Product, error) {
	if m.fetchAllFunc == nil {
		return nil, nil
	}
	return m.fetchAllFunc(ctx, tenantID)
}

func (m *mockRepository) Update(ctx context.Context, product *models.Product) error {
	if m.updateFunc == nil {
		return nil
	}
	return m.updateFunc(ctx, product)
}

func (m *mockRepository) Delete(ctx context.Context, tenantID, id string) error {
	if m.deleteFunc == nil {
		return nil
	}
	return m.deleteFunc(ctx, tenantID, id)
}
