package employees

import (
	"context"

	"leguiburger/internal/models"
)

type mockRepository struct {
	createFunc     func(ctx context.Context, employee *models.Employee) error
	getByIDFunc    func(ctx context.Context, tenantID, id string) (*models.Employee, error)
	getByEmailFunc func(ctx context.Context, tenantID, email string) (*models.Employee, error)
	fetchAllFunc   func(ctx context.Context, tenantID string) ([]models.Employee, error)
	getAllFunc     func(ctx context.Context) ([]models.Employee, error)
	updateFunc     func(ctx context.Context, employee *models.Employee) error
	deleteFunc     func(ctx context.Context, tenantID, id string) error
}

func (m *mockRepository) Create(ctx context.Context, employee *models.Employee) error {
	if m.createFunc == nil {
		return nil
	}
	return m.createFunc(ctx, employee)
}

func (m *mockRepository) GetByID(ctx context.Context, tenantID, id string) (*models.Employee, error) {
	if m.getByIDFunc == nil {
		return nil, nil
	}
	return m.getByIDFunc(ctx, tenantID, id)
}

func (m *mockRepository) GetByEmail(ctx context.Context, tenantID, email string) (*models.Employee, error) {
	if m.getByEmailFunc == nil {
		return nil, nil
	}
	return m.getByEmailFunc(ctx, tenantID, email)
}

func (m *mockRepository) GetAll(ctx context.Context) ([]models.Employee, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}
	return nil, nil
}

func (m *mockRepository) FetchAll(ctx context.Context, tenantID string) ([]models.Employee, error) {
	if m.fetchAllFunc == nil {
		return nil, nil
	}
	return m.fetchAllFunc(ctx, tenantID)
}

func (m *mockRepository) Update(ctx context.Context, employee *models.Employee) error {
	if m.updateFunc == nil {
		return nil
	}
	return m.updateFunc(ctx, employee)
}

func (m *mockRepository) Delete(ctx context.Context, tenantID, id string) error {
	if m.deleteFunc == nil {
		return nil
	}
	return m.deleteFunc(ctx, tenantID, id)
}
