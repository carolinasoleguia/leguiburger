package extras

import (
	"context"
	"errors"
	"testing"

	"leguiburger/internal/models"
)

type mockTenantRepository struct {
	getByIDFunc func(ctx context.Context, id string) (*models.Tenant, error)
	getAllFunc  func(ctx context.Context) ([]models.Tenant, error)
}

func (m *mockTenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	return m.getByIDFunc(ctx, id)
}
func (m *mockTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	return nil
}
func (m *mockTenantRepository) GetAll(ctx context.Context) ([]models.Tenant, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}
	return nil, nil
}
func (m *mockTenantRepository) GetByTaxID(ctx context.Context, taxId string) (*models.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) GetBySubdomain(ctx context.Context, subdomain string) (*models.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) GetByNameAndSubdomain(ctx context.Context, name string, subdomain string) (*models.Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	return nil
}
func (m *mockTenantRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func TestCreateExtra_Success(t *testing.T) {
	repo := &mockRepository{
		getByNameFunc: func(ctx context.Context, tenantID, name string) (*models.Extra, error) {
			if name != "Cheddar" {
				t.Errorf("se esperaba nombre normalizado, se obtuvo: %s", name)
			}
			return nil, nil
		},
		createFunc: func(ctx context.Context, extra *models.Extra) error {
			extra.ID = "generated-id"
			return nil
		},
	}

	tenantRepo := &mockTenantRepository{
		getByIDFunc: func(ctx context.Context, id string) (*models.Tenant, error) {
			return &models.Tenant{ID: id}, nil
		},
	}

	service := NewService(repo, tenantRepo)
	trackStock := false

	res, err := service.CreateExtra(context.Background(), "tenant-1", " Cheddar ", 250.5, 10, &trackStock)
	if err != nil {
		t.Fatalf("se esperaba exito, se obtuvo error: %v", err)
	}

	if res.Name != "Cheddar" || res.CurrentPrice != 250.5 || res.CurrentStock != 10 || res.TrackStock != false || res.IsActive != true {
		t.Errorf("los datos no se normalizaron correctamente: %+v", res)
	}
}

func TestCreateExtra_InvalidName(t *testing.T) {
	service := NewService(&mockRepository{}, &mockTenantRepository{})

	_, err := service.CreateExtra(context.Background(), "tenant-1", "", 100, 1, nil)
	if !errors.Is(err, ErrInvalidExtraData) {
		t.Errorf("se esperaba ErrInvalidExtraData, se obtuvo: %v", err)
	}
}

func TestCreateExtra_InvalidPrice(t *testing.T) {
	service := NewService(&mockRepository{}, &mockTenantRepository{})

	_, err := service.CreateExtra(context.Background(), "tenant-1", "Cheddar", -1, 1, nil)
	if !errors.Is(err, ErrInvalidExtraPrice) {
		t.Errorf("se esperaba ErrInvalidExtraPrice, se obtuvo: %v", err)
	}
}

func TestCreateExtra_DuplicateName(t *testing.T) {
	repo := &mockRepository{
		getByNameFunc: func(ctx context.Context, tenantID, name string) (*models.Extra, error) {
			return &models.Extra{ID: "existing-id", Name: name}, nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	_, err := service.CreateExtra(context.Background(), "tenant-1", "Cheddar", 100, 1, nil)

	if !errors.Is(err, ErrDuplicateExtraName) {
		t.Errorf("se esperaba ErrDuplicateExtraName, se obtuvo: %v", err)
	}
}

func TestListExtras_TenantNotFound(t *testing.T) {
	repo := &mockRepository{}
	tenantRepo := &mockTenantRepository{
		getByIDFunc: func(ctx context.Context, id string) (*models.Tenant, error) {
			return nil, nil
		},
	}

	service := NewService(repo, tenantRepo)
	_, err := service.ListExtras(context.Background(), "tenant-fantasma")

	if !errors.Is(err, ErrTenantNotFoundForExtra) {
		t.Errorf("se esperaba ErrTenantNotFoundForExtra, se obtuvo: %v", err)
	}
}

func TestUpdateExtra_Success(t *testing.T) {
	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, tenantID, id string) (*models.Extra, error) {
			return &models.Extra{ID: id, TenantID: tenantID, Name: "Cheddar", CurrentPrice: 100, CurrentStock: 5, TrackStock: true, IsActive: true}, nil
		},
		getByNameFunc: func(ctx context.Context, tenantID, name string) (*models.Extra, error) {
			return nil, nil
		},
		updateFunc: func(ctx context.Context, extra *models.Extra) error {
			return nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	newPrice := 150.0
	newStock := 8
	newActive := false

	res, err := service.UpdateExtra(context.Background(), "tenant-1", "extra-1", "Panceta", &newPrice, &newStock, nil, &newActive)
	if err != nil {
		t.Fatalf("se esperaba exito, se obtuvo error: %v", err)
	}

	if res.Name != "Panceta" || res.CurrentPrice != 150 || res.CurrentStock != 8 || res.IsActive != false {
		t.Errorf("los datos no se actualizaron correctamente: %+v", res)
	}
}

func TestDeleteExtra_NotFound(t *testing.T) {
	repo := &mockRepository{
		getByIDFunc: func(ctx context.Context, tenantID, id string) (*models.Extra, error) {
			return nil, nil
		},
	}

	service := NewService(repo, &mockTenantRepository{})
	err := service.DeleteExtra(context.Background(), "tenant-1", "missing")

	if !errors.Is(err, ErrExtraNotFound) {
		t.Errorf("se esperaba ErrExtraNotFound, se obtuvo: %v", err)
	}
}
