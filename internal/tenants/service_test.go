package tenants

import (
	"context"
	"errors"
	"leguiburger/internal/models"
	"testing"
)

// 1. Creamos un Mock del repositorio para simular la base de datos
type mockRepository struct {
	OnCreate                func(ctx context.Context, tenant *models.Tenant) error
	OnGetByID               func(ctx context.Context, id string) (*models.Tenant, error)
	OnGetBySubdomain        func(ctx context.Context, subdomain string) (*models.Tenant, error)
	OnGetByNameAndSubdomain func(ctx context.Context, name, subdomain string) (*models.Tenant, error)
	OnGetByTaxID            func(ctx context.Context, taxID string) (*models.Tenant, error)
	OnUpdate                func(ctx context.Context, tenant *models.Tenant) error
	OnDelete                func(ctx context.Context, id string) error
}

func (m *mockRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	return m.OnCreate(ctx, tenant)
}

func (m *mockRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	if m.OnGetByID != nil {
		return m.OnGetByID(ctx, id)
	}
	return nil, nil
}

func (m *mockRepository) GetBySubdomain(ctx context.Context, subdomain string) (*models.Tenant, error) {
	return m.OnGetBySubdomain(ctx, subdomain)
}

func (m *mockRepository) GetByTaxID(ctx context.Context, taxID string) (*models.Tenant, error) {
	if m.OnGetByTaxID != nil {
		return m.OnGetByTaxID(ctx, taxID)
	}
	return nil, nil
}

func (m *mockRepository) GetByNameAndSubdomain(ctx context.Context, name, subdomain string) (*models.Tenant, error) {
	if m.OnGetByNameAndSubdomain != nil {
		return m.OnGetByNameAndSubdomain(ctx, name, subdomain)
	}
	return nil, nil
}

func (m *mockRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	if m.OnUpdate != nil {
		return m.OnUpdate(ctx, tenant)
	}
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	if m.OnDelete != nil {
		return m.OnDelete(ctx, id)
	}
	return nil
}

func TestRegisterTenant_RepoError(t *testing.T) {
	// Simula que la base de datos explota al intentar guardar
	mockRepo := &mockRepository{
		OnGetBySubdomain: func(ctx context.Context, subdomain string) (*models.Tenant, error) {
			return nil, nil
		},
		OnCreate: func(ctx context.Context, tenant *models.Tenant) error {
			return errors.New("error de conexion de base de datos")
		},
	}

	service := NewService(mockRepo)
	// Ajustado a la firma con taxID 👈
	_, err := service.RegisterTenant(context.Background(), "Legui", "legui", "20359486163")

	if err == nil {
		t.Error("Se esperaba un error del repositorio, pero la creación fue exitosa")
	}
}

func TestRegisterTenant_DuplicateSubdomain(t *testing.T) {
	mockRepo := &mockRepository{
		// 1. 💡 Configuramos el mock con el nuevo método combinado:
		OnGetByNameAndSubdomain: func(ctx context.Context, name, subdomain string) (*models.Tenant, error) {
			// Simulamos que ya existe un comercio "Leguiburger" con el subdominio "laplata"
			return &models.Tenant{
				ID:        "un-id-existente",
				Name:      "Leguiburger",
				Subdomain: "laplata",
			}, nil
		},
	}

	service := NewService(mockRepo)

	// 2. Intentamos registrar exactamente el mismo comercio y subdominio
	_, err := service.RegisterTenant(context.Background(), "Leguiburger", "laplata", "20359486163")

	// 3. 💡 Validamos que devuelva nuestro nuevo error específico
	if err == nil {
		t.Error("Se esperaba un error por registro duplicado, pero la operación fue exitosa")
	}

	if !errors.Is(err, ErrDuplicateBranch) {
		t.Errorf("Se esperaba el error %v, pero se obtuvo %v", ErrDuplicateBranch, err)
	}
}

func TestRegisterTenant_NormalizesSubdomain(t *testing.T) {
	var savedSubdomain string

	mockRepo := &mockRepository{
		OnGetBySubdomain: func(ctx context.Context, subdomain string) (*models.Tenant, error) {
			return nil, nil // Simula que está disponible
		},
		OnCreate: func(ctx context.Context, tenant *models.Tenant) error {
			savedSubdomain = tenant.Subdomain // Capturamos lo que se va a guardar en la DB
			return nil
		},
	}

	service := NewService(mockRepo)

	// Mandamos un subdominio con mayúsculas, espacios y taxID 👈
	_, err := service.RegisterTenant(context.Background(), "Burger", "  LeGui-CeNtRo  ", "20359486163")
	if err != nil {
		t.Fatalf("No se esperaba un error, pero ocurrió: %v", err)
	}

	// Verificamos si se normalizó correctamente antes de ir a la DB
	expectedNormalized := "legui-centro"
	if savedSubdomain != expectedNormalized {
		t.Errorf("Se esperaba el subdominio normalizado '%s', pero se guardó '%s'", expectedNormalized, savedSubdomain)
	}
}

func TestUpdateTenant_Success(t *testing.T) {
	mockRepo := &mockRepository{
		OnGetByID: func(ctx context.Context, id string) (*models.Tenant, error) {
			return &models.Tenant{ID: "test-id", Name: "Viejo Nombre", Subdomain: "viejo-sub", TaxID: "11111111"}, nil
		},
		OnGetBySubdomain: func(ctx context.Context, subdomain string) (*models.Tenant, error) {
			return nil, nil // El nuevo subdominio está libre
		},
		OnUpdate: func(ctx context.Context, tenant *models.Tenant) error {
			return nil
		},
	}

	service := NewService(mockRepo)
	nuevoActive := false

	// Actualizado para usar los 6 parámetros de la firma, incluyendo taxID 👈
	updated, err := service.UpdateTenant(context.Background(), "test-id", "Nuevo Nombre", "nuevo-sub", "22222222", &nuevoActive)
	if err != nil {
		t.Fatalf("No se esperaba error, pero se obtuvo: %v", err)
	}

	if updated.Name != "Nuevo Nombre" || updated.Subdomain != "nuevo-sub" || updated.TaxID != "22222222" || updated.Active != false {
		t.Errorf("Los campos no se actualizaron correctamente: %+v", updated)
	}
}

func TestUpdateTenant_NotFound(t *testing.T) {
	mockRepo := &mockRepository{
		OnGetByID: func(ctx context.Context, id string) (*models.Tenant, error) {
			return nil, ErrTenantNotFound // Simula que no existe
		},
	}

	service := NewService(mockRepo)
	// Ajustado a la firma con taxID como string vacío "" 👈
	_, err := service.UpdateTenant(context.Background(), "inexistente", "Nombre", "sub", "", nil)

	if err != ErrTenantNotFound {
		t.Errorf("Se esperaba error ErrTenantNotFound, se obtuvo: %v", err)
	}
}

func TestDeleteTenant_Success(t *testing.T) {
	deleteLlamado := false
	mockRepo := &mockRepository{
		OnGetByID: func(ctx context.Context, id string) (*models.Tenant, error) {
			return &models.Tenant{ID: "test-id"}, nil
		},
		OnDelete: func(ctx context.Context, id string) error {
			deleteLlamado = true
			return nil
		},
	}

	service := NewService(mockRepo)
	err := service.DeleteTenant(context.Background(), "test-id")

	if err != nil {
		t.Fatalf("No se esperaba error al eliminar, se obtuvo: %v", err)
	}

	if !deleteLlamado {
		t.Error("Se esperaba que se llamara al método Delete del repositorio")
	}
}
