package customers

import (
	"context"
	"leguiburger/internal/models"
)

type mockRepository struct {
	createFunc     func(ctx context.Context, customer *models.Customer) error
	getByIDFunc    func(ctx context.Context, tenantID, id string) (*models.Customer, error)
	getByEmailFunc func(ctx context.Context, tenantID, email string) (*models.Customer, error)
	fetchAllFunc   func(ctx context.Context, tenantID string) ([]models.Customer, error)
	updateFunc     func(ctx context.Context, customer *models.Customer) error
	deleteFunc     func(ctx context.Context, tenantID, id string) error
}

func (m *mockRepository) Create(ctx context.Context, customer *models.Customer) error {
	if m.createFunc == nil {
		return nil
	}
	return m.createFunc(ctx, customer)
}

func (m *mockRepository) GetByID(ctx context.Context, tenantID, id string) (*models.Customer, error) {
	if m.getByIDFunc == nil {
		return nil, nil
	}
	return m.getByIDFunc(ctx, tenantID, id)
}

func (m *mockRepository) GetByEmail(ctx context.Context, tenantID, email string) (*models.Customer, error) {
	if m.getByEmailFunc == nil {
		return nil, nil
	}
	return m.getByEmailFunc(ctx, tenantID, email)
}

func (m *mockRepository) FetchAll(ctx context.Context, tenantID string) ([]models.Customer, error) {
	if m.fetchAllFunc == nil {
		return nil, nil
	}
	return m.fetchAllFunc(ctx, tenantID)
}

func (m *mockRepository) Update(ctx context.Context, customer *models.Customer) error {
	if m.updateFunc == nil {
		return nil
	}
	return m.updateFunc(ctx, customer)
}

func (m *mockRepository) Delete(ctx context.Context, tenantID, id string) error {
	if m.deleteFunc == nil {
		return nil
	}
	return m.deleteFunc(ctx, tenantID, id)
}
