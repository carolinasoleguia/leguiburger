package shipping

import (
	"context"
	"leguiburger/internal/models"
)

type mockRepository struct {
	createFunc                   func(ctx context.Context, sm *models.ShippingMethod) error
	getByIDFunc                  func(ctx context.Context, tenantID, id string) (*models.ShippingMethod, error)
	getByNameAndTypificationFunc func(ctx context.Context, tenantID, name, typification string) (*models.ShippingMethod, error)
	fetchAllFunc                 func(ctx context.Context, tenantID string) ([]models.ShippingMethod, error)
	updateFunc                   func(ctx context.Context, sm *models.ShippingMethod) error
	deleteFunc                   func(ctx context.Context, tenantID, id string) error
}

func (m *mockRepository) Create(ctx context.Context, sm *models.ShippingMethod) error {
	return m.createFunc(ctx, sm)
}

func (m *mockRepository) GetByID(ctx context.Context, tenantID, id string) (*models.ShippingMethod, error) {
	return m.getByIDFunc(ctx, tenantID, id)
}

func (m *mockRepository) GetByNameAndTypification(ctx context.Context, tenantID, name, typification string) (*models.ShippingMethod, error) {
	return m.getByNameAndTypificationFunc(ctx, tenantID, name, typification)
}

func (m *mockRepository) FetchAll(ctx context.Context, tenantID string) ([]models.ShippingMethod, error) {
	return m.fetchAllFunc(ctx, tenantID)
}

func (m *mockRepository) Update(ctx context.Context, sm *models.ShippingMethod) error {
	return m.updateFunc(ctx, sm)
}

func (m *mockRepository) Delete(ctx context.Context, tenantID, id string) error {
	return m.deleteFunc(ctx, tenantID, id)
}
