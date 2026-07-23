package employees

import (
	"context"

	"leguiburger/internal/models"
)

type MockRepository struct {
	OnCreate     func(ctx context.Context, employee *models.Employee) error
	OnGetByID    func(ctx context.Context, tenantID, id string) (*models.Employee, error)
	OnGetByEmail func(ctx context.Context, tenantID, email string) (*models.Employee, error) // 🔑 Recibe tenantID y email
	OnFetchAll   func(ctx context.Context, tenantID string) ([]models.Employee, error)
	OnGetAll     func(ctx context.Context) ([]models.Employee, error) // 🔑 Agregado aquí
	OnUpdate     func(ctx context.Context, employee *models.Employee) error
	OnDelete     func(ctx context.Context, tenantID, id string) error
}

func (m *MockRepository) Create(ctx context.Context, employee *models.Employee) error {
	if m.OnCreate == nil {
		return nil
	}
	return m.OnCreate(ctx, employee)
}

func (m *MockRepository) GetByID(ctx context.Context, tenantID, id string) (*models.Employee, error) {
	if m.OnGetByID == nil {
		return nil, nil
	}
	return m.OnGetByID(ctx, tenantID, id)
}

func (m *MockRepository) GetByEmail(ctx context.Context, tenantID, email string) (*models.Employee, error) {
	if m.OnGetByEmail == nil {
		return nil, nil
	}
	return m.OnGetByEmail(ctx, tenantID, email)
}

func (m *MockRepository) GetAll(ctx context.Context) ([]models.Employee, error) {
	if m.OnGetAll != nil {
		return m.OnGetAll(ctx)
	}
	return nil, nil
}

func (m *MockRepository) FetchAll(ctx context.Context, tenantID string) ([]models.Employee, error) {
	if m.OnFetchAll == nil {
		return nil, nil
	}
	return m.OnFetchAll(ctx, tenantID)
}

func (m *MockRepository) Update(ctx context.Context, employee *models.Employee) error {
	if m.OnUpdate == nil {
		return nil
	}
	return m.OnUpdate(ctx, employee)
}

func (m *MockRepository) Delete(ctx context.Context, tenantID, id string) error {
	if m.OnDelete == nil {
		return nil
	}
	return m.OnDelete(ctx, tenantID, id)
}
