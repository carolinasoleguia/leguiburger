package auth

import (
	"context"
	"errors"
	"testing"

	"leguiburger/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type mockAuthRepository struct {
	getByEmailAndTenantFn func(ctx context.Context, tenantID, email string) (*models.Employee, error)
	getByEmailFn          func(ctx context.Context, email string) (*models.Employee, error)
}

func (m *mockAuthRepository) GetByEmailAndTenant(ctx context.Context, tenantID, email string) (*models.Employee, error) {
	if m.getByEmailAndTenantFn != nil {
		return m.getByEmailAndTenantFn(ctx, tenantID, email)
	}
	return nil, nil
}

func (m *mockAuthRepository) GetByEmail(ctx context.Context, email string) (*models.Employee, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, nil
}

type mockTenantRepository struct {
	getByIDFn               func(ctx context.Context, id string) (*models.Tenant, error)
	getBySubdomainFn        func(ctx context.Context, subdomain string) (*models.Tenant, error)
	getByTaxIDFn            func(ctx context.Context, taxID string) (*models.Tenant, error)
	getByNameAndSubdomainFn func(ctx context.Context, name, subdomain string) (*models.Tenant, error)
	createFn                func(ctx context.Context, tenant *models.Tenant) error
	updateFn                func(ctx context.Context, tenant *models.Tenant) error
	deleteFn                func(ctx context.Context, id string) error
	getAllFn                func(ctx context.Context) ([]models.Tenant, error)
}

func (m *mockTenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockTenantRepository) GetAll(ctx context.Context) ([]models.Tenant, error) {
	if m.getAllFn != nil {
		return m.getAllFn(ctx)
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

func TestNewService_RequiresJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	_, err := NewService(&mockAuthRepository{}, &mockTenantRepository{})
	if !errors.Is(err, ErrJWTSecretRequired) {
		t.Fatalf("se esperaba ErrJWTSecretRequired, se obtuvo %v", err)
	}
}

func TestService_Login(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	password := "Secret123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("no se pudo hashear password: %v", err)
	}

	validTenantID := "tenant-uuid-1"
	validEmail := "admin@test.com"
	ownerEmail := "owner@test.com"

	dummyEmployee := &models.Employee{
		ID:           "employee-uuid-1",
		TenantID:     &validTenantID,
		FirstName:    "Ana",
		LastName:     "Admin",
		Email:        validEmail,
		PasswordHash: string(hashedPassword),
		Phone:        "2215555555",
		Role:         "admin",
		IsActive:     true,
	}

	dummyOwner := &models.Employee{
		ID:           "owner-uuid-1",
		TenantID:     nil,
		FirstName:    "Carolina",
		LastName:     "Owner",
		Email:        ownerEmail,
		PasswordHash: string(hashedPassword),
		Role:         RoleOwner,
		IsActive:     true,
	}

	activeTenant := &models.Tenant{
		ID:     validTenantID,
		Name:   "LeguiBurger",
		Active: true,
	}

	tests := []struct {
		name           string
		tenantID       string
		email          string
		password       string
		mockTenant     func(ctx context.Context, id string) (*models.Tenant, error)
		mockRepo       func(ctx context.Context, tenantID, email string) (*models.Employee, error)
		mockRepoGlobal func(ctx context.Context, email string) (*models.Employee, error)
		expectedErr    error
		expectSuccess  bool
	}{
		{
			name:     "login exitoso empleado con tenant",
			tenantID: validTenantID,
			email:    " ADMIN@Test.com ",
			password: password,
			mockTenant: func(ctx context.Context, id string) (*models.Tenant, error) {
				return activeTenant, nil
			},
			mockRepo: func(ctx context.Context, tenantID, email string) (*models.Employee, error) {
				if tenantID != validTenantID || email != validEmail {
					t.Fatalf("se esperaban tenant/email normalizados, se obtuvo %q/%q", tenantID, email)
				}
				return dummyEmployee, nil
			},
			expectedErr:   nil,
			expectSuccess: true,
		},
		{
			name:     "login exitoso owner sin tenant",
			tenantID: "",
			email:    ownerEmail,
			password: password,
			mockRepoGlobal: func(ctx context.Context, email string) (*models.Employee, error) {
				return dummyOwner, nil
			},
			expectedErr:   nil,
			expectSuccess: true,
		},
		{
			name:     "empleado sin tenant devuelve tenant requerido",
			tenantID: "",
			email:    validEmail,
			password: password,
			mockRepoGlobal: func(ctx context.Context, email string) (*models.Employee, error) {
				return dummyEmployee, nil
			},
			expectedErr:   ErrTenantRequired,
			expectSuccess: false,
		},
		{
			name:     "empleado con tenant incorrecto devuelve forbidden",
			tenantID: "tenant-uuid-2",
			email:    validEmail,
			password: password,
			mockTenant: func(ctx context.Context, id string) (*models.Tenant, error) {
				return &models.Tenant{ID: id, Active: true}, nil
			},
			mockRepo: func(ctx context.Context, tenantID, email string) (*models.Employee, error) {
				return nil, nil
			},
			mockRepoGlobal: func(ctx context.Context, email string) (*models.Employee, error) {
				return dummyEmployee, nil
			},
			expectedErr:   ErrForbiddenTenant,
			expectSuccess: false,
		},
		{
			name:     "tenant inexistente devuelve tenant invalido",
			tenantID: "tenant-inexistente",
			email:    validEmail,
			password: password,
			mockTenant: func(ctx context.Context, id string) (*models.Tenant, error) {
				return nil, nil
			},
			expectedErr:   ErrTenantNotFoundForAuth,
			expectSuccess: false,
		},
		{
			name:     "tenant inactivo devuelve tenant invalido",
			tenantID: validTenantID,
			email:    validEmail,
			password: password,
			mockTenant: func(ctx context.Context, id string) (*models.Tenant, error) {
				return &models.Tenant{ID: id, Active: false}, nil
			},
			expectedErr:   ErrTenantNotFoundForAuth,
			expectSuccess: false,
		},
		{
			name:     "empleado inexistente devuelve credenciales invalidas",
			tenantID: validTenantID,
			email:    "noexiste@test.com",
			password: password,
			mockTenant: func(ctx context.Context, id string) (*models.Tenant, error) {
				return activeTenant, nil
			},
			mockRepo: func(ctx context.Context, tenantID, email string) (*models.Employee, error) {
				return nil, nil
			},
			mockRepoGlobal: func(ctx context.Context, email string) (*models.Employee, error) {
				return nil, nil
			},
			expectedErr:   ErrInvalidCredentials,
			expectSuccess: false,
		},
		{
			name:     "password incorrecta devuelve credenciales invalidas",
			tenantID: validTenantID,
			email:    validEmail,
			password: "WrongPassword!",
			mockTenant: func(ctx context.Context, id string) (*models.Tenant, error) {
				return activeTenant, nil
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

			svc, err := NewService(repoMock, tenantMock)
			if err != nil {
				t.Fatalf("no se esperaba error al crear servicio: %v", err)
			}

			res, err := svc.Login(context.Background(), tt.tenantID, tt.email, tt.password)

			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("se esperaba error %v, se obtuvo %v", tt.expectedErr, err)
				}
			} else if err != nil {
				t.Fatalf("no se esperaba error, se obtuvo %v", err)
			}

			if tt.expectSuccess {
				if res == nil || res.Token == "" {
					t.Fatal("se esperaba un token JWT valido")
				}
				if res.Employee.Email == "" || res.Employee.ID == "" {
					t.Fatal("se esperaba DTO publico de empleado")
				}
			}
		})
	}
}

func TestGenerateToken_ConfiguredSecret(t *testing.T) {
	previousSecret := jwtSecret
	t.Cleanup(func() {
		jwtSecret = previousSecret
	})

	if err := ConfigureJWTSecret("test-secret"); err != nil {
		t.Fatalf("no se esperaba error configurando JWT_SECRET: %v", err)
	}

	tenantID := "tenant-1"
	token, err := GenerateToken("user-1", "user@test.com", "admin", &tenantID)
	if err != nil {
		t.Fatalf("no se esperaba error generando token: %v", err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("no se esperaba error validando token: %v", err)
	}

	if claims.UserID != "user-1" || claims.Email != "user@test.com" || claims.Role != "admin" || claims.TenantID != tenantID {
		t.Fatalf("claims inesperados: %+v", claims)
	}
}

func TestGenerateToken_MissingSecret(t *testing.T) {
	previousSecret := jwtSecret
	t.Cleanup(func() {
		jwtSecret = previousSecret
	})

	jwtSecret = nil

	_, err := GenerateToken("user-1", "user@test.com", "admin", nil)
	if !errors.Is(err, ErrJWTSecretRequired) {
		t.Fatalf("se esperaba ErrJWTSecretRequired generando token, se obtuvo %v", err)
	}
}
