package tenants

import (
	"context"
	"errors"
	"leguiburger/internal/models"
	"testing"
)

type mockRepository struct {
	createFunc                func(ctx context.Context, tenant *models.Tenant) error
	getByIDFunc               func(ctx context.Context, id string) (*models.Tenant, error)
	getBySubdomainFunc        func(ctx context.Context, subdomain string) (*models.Tenant, error)
	getByNameAndSubdomainFunc func(ctx context.Context, name, subdomain string) (*models.Tenant, error)
	getByTaxIDFunc            func(ctx context.Context, taxID string) (*models.Tenant, error)
	updateFunc                func(ctx context.Context, tenant *models.Tenant) error
	deleteFunc                func(ctx context.Context, id string) error
	getAllFunc                func(ctx context.Context) ([]models.Tenant, error)
}

func (m *mockRepository) GetAll(ctx context.Context) ([]models.Tenant, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}
	return nil, nil
}

func (m *mockRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	return m.createFunc(ctx, tenant)
}

func (m *mockRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockRepository) GetBySubdomain(ctx context.Context, subdomain string) (*models.Tenant, error) {
	return m.getBySubdomainFunc(ctx, subdomain)
}

func (m *mockRepository) GetByTaxID(ctx context.Context, taxID string) (*models.Tenant, error) {
	if m.getByTaxIDFunc != nil {
		return m.getByTaxIDFunc(ctx, taxID)
	}
	return nil, nil
}

func (m *mockRepository) GetByNameAndSubdomain(ctx context.Context, name, subdomain string) (*models.Tenant, error) {
	if m.getByNameAndSubdomainFunc != nil {
		return m.getByNameAndSubdomainFunc(ctx, name, subdomain)
	}
	return nil, nil
}

func (m *mockRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, tenant)
	}
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestRegisterTenant_RepoError(t *testing.T) {
	mockRepo := &mockRepository{
		getBySubdomainFunc: func(ctx context.Context, subdomain string) (*models.Tenant, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, tenant *models.Tenant) error {
			return errors.New("error de conexion de base de datos")
		},
	}

	service := NewService(mockRepo)
	_, err := service.RegisterTenant(context.Background(), "Legui", "legui", "20359486163")

	if err == nil {
		t.Error("se esperaba un error del repositorio, pero la creacion fue exitosa")
	}
}

func TestRegisterTenant_DuplicateSubdomain(t *testing.T) {
	mockRepo := &mockRepository{
		getByNameAndSubdomainFunc: func(ctx context.Context, name, subdomain string) (*models.Tenant, error) {
			return &models.Tenant{
				ID:        "un-id-existente",
				Name:      "Leguiburger",
				Subdomain: "laplata",
			}, nil
		},
	}

	service := NewService(mockRepo)
	_, err := service.RegisterTenant(context.Background(), "Leguiburger", "laplata", "20359486163")
	if err == nil {
		t.Error("se esperaba un error por registro duplicado, pero la operacion fue exitosa")
	}

	if !errors.Is(err, ErrDuplicateBranch) {
		t.Errorf("se esperaba el error %v, pero se obtuvo %v", ErrDuplicateBranch, err)
	}
}

func TestRegisterTenant_NormalizesSubdomain(t *testing.T) {
	var savedSubdomain string

	mockRepo := &mockRepository{
		getBySubdomainFunc: func(ctx context.Context, subdomain string) (*models.Tenant, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, tenant *models.Tenant) error {
			savedSubdomain = tenant.Subdomain
			return nil
		},
	}

	service := NewService(mockRepo)
	_, err := service.RegisterTenant(context.Background(), "Burger", "  LeGui-CeNtRo  ", "20359486163")
	if err != nil {
		t.Fatalf("no se esperaba un error, pero ocurrio: %v", err)
	}
	expectedNormalized := "legui-centro"
	if savedSubdomain != expectedNormalized {
		t.Errorf("se esperaba el subdominio normalizado '%s', pero se guardo '%s'", expectedNormalized, savedSubdomain)
	}
}

func TestUpdateTenant_Success(t *testing.T) {
	mockRepo := &mockRepository{
		getByIDFunc: func(ctx context.Context, id string) (*models.Tenant, error) {
			return &models.Tenant{ID: "test-id", Name: "Viejo Nombre", Subdomain: "viejo-sub", TaxID: "11111111"}, nil
		},
		getBySubdomainFunc: func(ctx context.Context, subdomain string) (*models.Tenant, error) {
			return nil, nil
		},
		updateFunc: func(ctx context.Context, tenant *models.Tenant) error {
			return nil
		},
	}

	service := NewService(mockRepo)
	nuevoActive := false
	updated, err := service.UpdateTenant(context.Background(), "test-id", "Nuevo Nombre", "nuevo-sub", "22222222", &nuevoActive)
	if err != nil {
		t.Fatalf("no se esperaba error, pero se obtuvo: %v", err)
	}

	if updated.Name != "Nuevo Nombre" || updated.Subdomain != "nuevo-sub" || updated.TaxID != "22222222" || updated.Active != false {
		t.Errorf("los campos no se actualizaron correctamente: %+v", updated)
	}
}

func TestUpdateTenant_NotFound(t *testing.T) {
	mockRepo := &mockRepository{
		getByIDFunc: func(ctx context.Context, id string) (*models.Tenant, error) {
			return nil, ErrTenantNotFound
		},
	}

	service := NewService(mockRepo)
	_, err := service.UpdateTenant(context.Background(), "inexistente", "Nombre", "sub", "", nil)

	if err != ErrTenantNotFound {
		t.Errorf("se esperaba error ErrTenantNotFound, se obtuvo: %v", err)
	}
}

func TestDeleteTenant_Success(t *testing.T) {
	deleteLlamado := false
	mockRepo := &mockRepository{
		getByIDFunc: func(ctx context.Context, id string) (*models.Tenant, error) {
			return &models.Tenant{ID: "test-id"}, nil
		},
		deleteFunc: func(ctx context.Context, id string) error {
			deleteLlamado = true
			return nil
		},
	}

	service := NewService(mockRepo)
	err := service.DeleteTenant(context.Background(), "test-id")

	if err != nil {
		t.Fatalf("no se esperaba error al eliminar, se obtuvo: %v", err)
	}

	if !deleteLlamado {
		t.Error("se esperaba que se llamara al metodo Delete del repositorio")
	}
}
