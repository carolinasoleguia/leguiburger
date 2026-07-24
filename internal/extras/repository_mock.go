package extras

import (
	"context"
	"leguiburger/internal/models"
)

type mockRepository struct {
	createFunc    func(ctx context.Context, extra *models.Extra) error
	getByIDFunc   func(ctx context.Context, tenantID, id string) (*models.Extra, error)
	getByNameFunc func(ctx context.Context, tenantID, name string) (*models.Extra, error)
	fetchAllFunc  func(ctx context.Context, tenantID string) ([]models.Extra, error)
	updateFunc    func(ctx context.Context, extra *models.Extra) error
	deleteFunc    func(ctx context.Context, tenantID, id string) error
}

func (m *mockRepository) Create(ctx context.Context, extra *models.Extra) error {
	if m.createFunc == nil {
		return nil
	}
	return m.createFunc(ctx, extra)
}

func (m *mockRepository) GetByID(ctx context.Context, tenantID, id string) (*models.Extra, error) {
	if m.getByIDFunc == nil {
		return nil, nil
	}
	return m.getByIDFunc(ctx, tenantID, id)
}

func (m *mockRepository) GetByName(ctx context.Context, tenantID, name string) (*models.Extra, error) {
	if m.getByNameFunc == nil {
		return nil, nil
	}
	return m.getByNameFunc(ctx, tenantID, name)
}

func (m *mockRepository) FetchAll(ctx context.Context, tenantID string) ([]models.Extra, error) {
	if m.fetchAllFunc == nil {
		return nil, nil
	}
	return m.fetchAllFunc(ctx, tenantID)
}

func (m *mockRepository) Update(ctx context.Context, extra *models.Extra) error {
	if m.updateFunc == nil {
		return nil
	}
	return m.updateFunc(ctx, extra)
}

func (m *mockRepository) Delete(ctx context.Context, tenantID, id string) error {
	if m.deleteFunc == nil {
		return nil
	}
	return m.deleteFunc(ctx, tenantID, id)
}
