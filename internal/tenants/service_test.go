package tenants

import (
	"context"
	"leguiburger/internal/models"
	"testing"
)

// 1. Creamos un Mock del repositorio para simular la base de datos
type mockRepository struct {
	OnCreate         func(ctx context.Context, tenant *models.Tenant) error
	OnGetByID        func(ctx context.Context, id string) (*models.Tenant, error)
	OnGetBySubdomain func(ctx context.Context, subdomain string) (*models.Tenant, error)
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

	// Ejecutamos la función que queremos testear
	_, err := service.RegisterTenant(context.Background(), "Legui Centro", "legui-centro")

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

	// Mandamos un subdominio con mayúsculas y espacios
	_, err := service.RegisterTenant(context.Background(), "Burger", "  LeGui-CeNtRo  ")
	if err != nil {
		t.Fatalf("No se esperaba un error, pero ocurrió: %v", err)
	}

	// Verificamos si se normalizó correctamente antes de ir a la DB
	expectedNormalized := "legui-centro"
	if savedSubdomain != expectedNormalized {
		t.Errorf("Se esperaba el subdominio normalizado '%s', pero se guardó '%s'", expectedNormalized, savedSubdomain)
	}
}
