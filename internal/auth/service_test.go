package auth

import (
	"context"
	"errors"
	"testing"

	"leguiburger/internal/models"

	"golang.org/x/crypto/bcrypt"
)

// --- MOCK DE AUTH REPOSITORY ---

type mockAuthRepository struct {
	getByEmailAndTenantFn func(ctx context.Context, tenantID, email string) (*models.Employee, error)
	getByEmailFn          func(ctx context.Context, email string) (*models.Employee, error) // 👈 Agregado
}

func (m *mockAuthRepository) GetByEmailAndTenant(ctx context.Context, tenantID, email string) (*models.Employee, error) {
	if m.getByEmailAndTenantFn != nil {
		return m.getByEmailAndTenantFn(ctx, tenantID, email)
	}
	return nil, nil
}

// 👈 Implementación requerida para cumplir con la interfaz Repository
func (m *mockAuthRepository) GetByEmail(ctx context.Context, email string) (*models.Employee, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, nil
}

// --- MOCK COMPLETO DE TENANT REPOSITORY ---

type mockTenantRepository struct {
	getByIDFn               func(ctx context.Context, id string) (*models.Tenant, error)
	getBySubdomainFn        func(ctx context.Context, subdomain string) (*models.Tenant, error)
	getByTaxIDFn            func(ctx context.Context, taxID string) (*models.Tenant, error)
	getByNameAndSubdomainFn func(ctx context.Context, name, subdomain string) (*models.Tenant, error)
	createFn                func(ctx context.Context, tenant *models.Tenant) error
	updateFn                func(ctx context.Context, tenant *models.Tenant) error
	deleteFn                func(ctx context.Context, id string) error
	listFn                  func(ctx context.Context) ([]*models.Tenant, error)
}

func (m *mockTenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockTenantRepository) GetBySubdomain(ctx context.Context, subdomain string) (*models.Tenant, error) {
	if m.getBySubdomainFn != nil {
		return m.getBySubdomainFn(ctx, subdomain)
	}
	return nil, nil
}

func (m *mockTenantRepository) GetByTaxID(ctx context.Context, taxID string) (*models.Tenant, error) {
	if m.getByTaxIDFn != nil {
		return m.getByTaxIDFn(ctx, taxID)
	}
	return nil, nil
}

func (m *mockTenantRepository) GetByNameAndSubdomain(ctx context.Context, name, subdomain string) (*models.Tenant, error) {
	if m.getByNameAndSubdomainFn != nil {
		return m.getByNameAndSubdomainFn(ctx, name, subdomain)
	}
	return nil, nil
}

func (m *mockTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	if m.createFn != nil {
		return m.createFn(ctx, tenant)
	}
	return nil
}

func (m *mockTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenant)
	}
	return nil
}

func (m *mockTenantRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockTenantRepository) List(ctx context.Context) ([]*models.Tenant, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

// --- SUITE DE TESTS DEL SERVICIO ---

func TestService_Login(t *testing.T) {
	password := "Secret123!"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	validTenantID := "tenant-uuid-1"
	validEmail := "admin@test.com"
	ownerEmail := "owner@test.com"

	dummyEmployee := &models.Employee{
		ID:           "employee-uuid-1",
		TenantID:     &validTenantID,
		Email:        validEmail,
		PasswordHash: string(hashedPassword),
		Role:         "admin",
		IsActive:     true,
	}

	dummyOwner := &models.Employee{
		ID:           "owner-uuid-1",
		TenantID:     nil, // Owner global
		Email:        ownerEmail,
		PasswordHash: string(hashedPassword),
		Role:         "owner",
		IsActive:     true,
	}

	dummyTenant := &models.Tenant{
		ID:   validTenantID,
		Name: "LeguiBurger",
	}

	tests := []struct {
		name           string
		tenantID       string
		email          string
		password       string
		mockTenant     func(ctx context.Context, id string) (*models.Tenant, error)
		mockRepo       func(ctx context.Context, tenantID, email string) (*models.Employee, error)
		mockRepoGlobal func(ctx context.Context, email string) (*models.Employee, error) // 👈 Para test de Owner
		expectedErr    error
		expectSuccess  bool
	}{
		{
			name:     "Login exitoso Admin",
			tenantID: validTenantID,
			email:    validEmail,
			password: password,
			mockTenant: func(ctx context.Context, id string) (*models.Tenant, error) {
				return dummyTenant, nil
			},
			mockRepo: func(ctx context.Context, tenantID, email string) (*models.Employee, error) {
				return dummyEmployee, nil
			},
			expectedErr:   nil,
			expectSuccess: true,
		},
		{
			name:       "Login exitoso Owner (sin tenantID)",
			tenantID:   "",
			email:      ownerEmail,
			password:   password,
			mockTenant: nil, // No debería ejecutarse validación de tenant
			mockRepoGlobal: func(ctx context.Context, email string) (*models.Employee, error) {
				return dummyOwner, nil
			},
			expectedErr:   nil,
			expectSuccess: true,
		},
		{
			name:     "Error si el Tenant no existe",
			tenantID: "tenant-inexistente",
			email:    validEmail,
			password: password,
			mockTenant: func(ctx context.Context, id string) (*models.Tenant, error) {
				return nil, nil // Tenant no encontrado
			},
			mockRepo: func(ctx context.Context, tenantID, email string) (*models.Employee, error) {
				return nil, nil
			},
			expectedErr:   ErrTenantNotFoundForAuth,
			expectSuccess: false,
		},
		{
			name:     "Error si el Empleado no existe",
			tenantID: validTenantID,
			email:    "noexiste@test.com",
			password: password,
			mockTenant: func(ctx context.Context, id string) (*models.Tenant, error) {
				return dummyTenant, nil
			},
			mockRepo: func(ctx context.Context, tenantID, email string) (*models.Employee, error) {
				return nil, nil // Empleado no encontrado
			},
			expectedErr:   ErrInvalidCredentials,
			expectSuccess: false,
		},
		{
			name:     "Error por contraseña incorrecta",
			tenantID: validTenantID,
			email:    validEmail,
			password: "WrongPassword!",
			mockTenant: func(ctx context.Context, id string) (*models.Tenant, error) {
				return dummyTenant, nil
			},
			mockRepo: func(ctx context.Context, tenantID, email string) (*models.Employee, error) {
				return dummyEmployee, nil
			},
			expectedErr:   ErrInvalidCredentials,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoMock := &mockAuthRepository{
				getByEmailAndTenantFn: tt.mockRepo,
				getByEmailFn:          tt.mockRepoGlobal,
			}
			tenantMock := &mockTenantRepository{getByIDFn: tt.mockTenant}

			svc := NewService(repoMock, tenantMock)
			res, err := svc.Login(context.Background(), tt.tenantID, tt.email, tt.password)

			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("se esperaba error %v, pero se obtuvo %v", tt.expectedErr, err)
				}
			} else if err != nil {
				t.Fatalf("no se esperaba error pero ocurrió: %v", err)
			}

			if tt.expectSuccess {
				if res == nil || res.Token == "" {
					t.Error("se esperaba un token JWT válido en la respuesta")
				}
				if res.Employee.Email != tt.email {
					t.Errorf("email retornado erróneo. Esperado: %s, obtenido: %s", tt.email, res.Employee.Email)
				}
			}
		})
	}
}
