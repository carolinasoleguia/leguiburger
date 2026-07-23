package products

import (
	"context"
	"leguiburger/internal/models"
)

type MockRepository struct {
	OnCreate    func(ctx context.Context, product *models.Product) error
	OnGetByID   func(ctx context.Context, tenantID, id string) (*models.Product, error)
	OnGetByName func(ctx context.Context, tenantID, name string) (*models.Product, error)
	OnFetchAll  func(ctx context.Context, tenantID string) ([]models.Product, error)
	OnUpdate    func(ctx context.Context, product *models.Product) error
	OnDelete    func(ctx context.Context, tenantID, id string) error
}

func (m *MockRepository) Create(ctx context.Context, product *models.Product) error {
	if m.OnCreate == nil {
		return nil
	}
	return m.OnCreate(ctx, product)
}

func (m *MockRepository) GetByID(ctx context.Context, tenantID, id string) (*models.Product, error) {
	if m.OnGetByID == nil {
		return nil, nil
	}
	return m.OnGetByID(ctx, tenantID, id)
}

func (m *MockRepository) GetByName(ctx context.Context, tenantID, name string) (*models.Product, error) {
	if m.OnGetByName == nil {
		return nil, nil
	}
	return m.OnGetByName(ctx, tenantID, name)
}

func (m *MockRepository) FetchAll(ctx context.Context, tenantID string) ([]models.Product, error) {
	if m.OnFetchAll == nil {
		return nil, nil
	}
	return m.OnFetchAll(ctx, tenantID)
}

func (m *MockRepository) Update(ctx context.Context, product *models.Product) error {
	if m.OnUpdate == nil {
		return nil
	}
	return m.OnUpdate(ctx, product)
}

func (m *MockRepository) Delete(ctx context.Context, tenantID, id string) error {
	if m.OnDelete == nil {
		return nil
	}
	return m.OnDelete(ctx, tenantID, id)
}
