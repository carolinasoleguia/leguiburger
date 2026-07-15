package tenants

import (
	"context"
	"errors"
	"leguiburger/internal/models"
	"testing"
)

// 1. Creamos un Mock del repositorio para simular la base de datos
type mockRepository struct {
	OnCreate         func(ctx context.Context, tenant *models.Tenant) error
	OnGetByID        func(ctx context.Context, id string) (*models.Tenant, error)
	OnGetBySubdomain func(ctx context.Context, subdomain string) (*models.Tenant, error)
	OnGetByTaxID     func(ctx context.Context, taxID string) (*models.Tenant, error) // 👈 Agregado a la interfaz
	OnUpdate         func(ctx context.Context, tenant *models.Tenant) error
	OnDelete         func(ctx context.Context, id string) error
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

// 👈 Implementación del nuevo método en el mock
func (m *mockRepository) GetByTaxID(ctx context.Context, taxID string) (*models.Tenant, error) {
	if m.OnGetByTaxID != nil {
		return m.OnGetByTaxID(ctx, taxID)
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

// 2. El Test Unitario para la regla de negocio del subdominio duplicado
func TestRegisterTenant_DuplicateSubdomain(t *testing.T) {
	// Configuramos el mock para que simule que el subdominio "legui-centro" YA EXISTE
	mockRepo := &mockRepository{
		OnGetBySubdomain: func(ctx context.Context, subdomain string) (*models.Tenant, error) {
			return &models.Tenant{ID: "algun-uuid", Subdomain: "legui-centro"}, nil
		},
		OnCreate: func(ctx context.Context, tenant *models.Tenant) error {
			return nil
		},
	}

	// Inyectamos el mock en el servicio real
	service := NewService(mockRepo)

	// Ejecutamos la función pasando el nuevo parámetro taxID: "20359486163" 👈
	_, err := service.RegisterTenant(context.Background(), "Legui Centro", "legui-centro", "20359486163")

	// Validamos que el servicio falle con el error esperado
	if err == nil {
		t.Fatal("Se esperaba un error por subdominio duplicado, pero la operación fue exitosa")
	}

	expectedError := "este subdominio ya está registrado por otro comercio"
	if err.Error() != expectedError {
		t.Errorf("Se esperaba el error '%s', pero se obtuvo '%s'", expectedError, err.Error())
	}
}

// 3. Test para verificar que el subdominio se limpie y normalice a minúsculas
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

func TestUpdateTenant_DuplicateSubdomain(t *testing.T) {
	mockRepo := &mockRepository{
		OnGetByID: func(ctx context.Context, id string) (*models.Tenant, error) {
			return &models.Tenant{ID: "mi-id", Name: "Burger", Subdomain: "sub-actual"}, nil
		},
		OnGetBySubdomain: func(ctx context.Context, subdomain string) (*models.Tenant, error) {
			// Simula que el nuevo subdominio ya lo tiene otro local con otra ID
			return &models.Tenant{ID: "otro-id", Subdomain: "sub-ocupado"}, nil
		},
	}

	service := NewService(mockRepo)
	// Ajustado a la firma con taxID vacío y sin validaciones de unicidad de tax_id 👈
	_, err := service.UpdateTenant(context.Background(), "mi-id", "Burger", "sub-ocupado", "", nil)

	if err == nil || err.Error() != "este subdominio ya está registrado por otro comercio" {
		t.Errorf("Se esperaba error de subdominio duplicado al actualizar, se obtuvo: %v", err)
	}
}
