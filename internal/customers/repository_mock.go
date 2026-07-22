package customers

import (
	"context"
	"leguiburger/internal/models"
)

type MockRepository struct {
	OnCreate     func(ctx context.Context, customer *models.Customer) error
	OnGetByID    func(ctx context.Context, tenantID, id string) (*models.Customer, error)
	OnGetByEmail func(ctx context.Context, tenantID, email string) (*models.Customer, error)
	OnFetchAll   func(ctx context.Context, tenantID string) ([]models.Customer, error)
	OnUpdate     func(ctx context.Context, customer *models.Customer) error
	OnDelete     func(ctx context.Context, tenantID, id string) error
}

func (m *MockRepository) Create(ctx context.Context, customer *models.Customer) error {
	if m.OnCreate == nil {
		return nil
	}
	return m.OnCreate(ctx, customer)
}

func (m *MockRepository) GetByID(ctx context.Context, tenantID, id string) (*models.Customer, error) {
	if m.OnGetByID == nil {
		return nil, nil
	}
	return m.OnGetByID(ctx, tenantID, id)
}

func (m *MockRepository) GetByEmail(ctx context.Context, tenantID, email string) (*models.Customer, error) {
	if m.OnGetByEmail == nil {
		return nil, nil
	}
	return m.OnGetByEmail(ctx, tenantID, email)
}

func (m *MockRepository) FetchAll(ctx context.Context, tenantID string) ([]models.Customer, error) {
	if m.OnFetchAll == nil {
		return nil, nil
	}
	return m.OnFetchAll(ctx, tenantID)
}

func (m *MockRepository) Update(ctx context.Context, customer *models.Customer) error {
	if m.OnUpdate == nil {
		return nil
	}
	return m.OnUpdate(ctx, customer)
}

func (m *MockRepository) Delete(ctx context.Context, tenantID, id string) error {
	if m.OnDelete == nil {
		return nil
	}
	return m.OnDelete(ctx, tenantID, id)
}
