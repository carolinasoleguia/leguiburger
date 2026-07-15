package shipping

import (
	"context"
	"leguiburger/internal/models"
)

type MockRepository struct {
	OnCreate                   func(ctx context.Context, sm *models.ShippingMethod) error
	OnGetByID                  func(ctx context.Context, tenantID, id string) (*models.ShippingMethod, error)
	OnGetByNameAndTypification func(ctx context.Context, tenantID, name, typification string) (*models.ShippingMethod, error)
	OnFetchAll                 func(ctx context.Context, tenantID string) ([]models.ShippingMethod, error)
	OnUpdate                   func(ctx context.Context, sm *models.ShippingMethod) error
	OnDelete                   func(ctx context.Context, tenantID, id string) error
}

func (m *MockRepository) Create(ctx context.Context, sm *models.ShippingMethod) error {
	return m.OnCreate(ctx, sm)
}

func (m *MockRepository) GetByID(ctx context.Context, tenantID, id string) (*models.ShippingMethod, error) {
	return m.OnGetByID(ctx, tenantID, id)
}

func (m *MockRepository) GetByNameAndTypification(ctx context.Context, tenantID, name, typification string) (*models.ShippingMethod, error) {
	return m.OnGetByNameAndTypification(ctx, tenantID, name, typification)
}

func (m *MockRepository) FetchAll(ctx context.Context, tenantID string) ([]models.ShippingMethod, error) {
	return m.OnFetchAll(ctx, tenantID)
}

func (m *MockRepository) Update(ctx context.Context, sm *models.ShippingMethod) error {
	return m.OnUpdate(ctx, sm)
}

func (m *MockRepository) Delete(ctx context.Context, tenantID, id string) error {
	return m.OnDelete(ctx, tenantID, id)
}
