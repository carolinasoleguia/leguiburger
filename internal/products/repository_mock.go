package products

import (
	"context"
	"leguiburger/internal/models"
)

type mockRepository struct {
	createFunc    func(ctx context.Context, product *models.Product) error
	getByIDFunc   func(ctx context.Context, brandID, id string) (*models.Product, error)
	getByNameFunc func(ctx context.Context, brandID, name string) (*models.Product, error)
	fetchAllFunc  func(ctx context.Context, brandID string) ([]models.Product, error)
	updateFunc    func(ctx context.Context, product *models.Product) error
	deleteFunc    func(ctx context.Context, brandID, id string) error
}

func (m *mockRepository) Create(ctx context.Context, product *models.Product) error {
	if m.createFunc == nil {
		return nil
	}
	return m.createFunc(ctx, product)
}

func (m *mockRepository) GetByID(ctx context.Context, brandID, id string) (*models.Product, error) {
	if m.getByIDFunc == nil {
		return nil, nil
	}
	return m.getByIDFunc(ctx, brandID, id)
}

func (m *mockRepository) GetByName(ctx context.Context, brandID, name string) (*models.Product, error) {
	if m.getByNameFunc == nil {
		return nil, nil
	}
	return m.getByNameFunc(ctx, brandID, name)
}

func (m *mockRepository) FetchAll(ctx context.Context, brandID string) ([]models.Product, error) {
	if m.fetchAllFunc == nil {
		return nil, nil
	}
	return m.fetchAllFunc(ctx, brandID)
}

func (m *mockRepository) Update(ctx context.Context, product *models.Product) error {
	if m.updateFunc == nil {
		return nil
	}
	return m.updateFunc(ctx, product)
}

func (m *mockRepository) Delete(ctx context.Context, brandID, id string) error {
	if m.deleteFunc == nil {
		return nil
	}
	return m.deleteFunc(ctx, brandID, id)
}
