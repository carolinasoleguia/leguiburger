package auth

import (
	"context"
	"errors"
	"testing"

	"leguiburger/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type mockAuthRepository struct {
	getEmployeeByEmailAndTenantFn func(ctx context.Context, tenantID, email string) (*models.Employee, error)
	getEmployeeByEmailFn          func(ctx context.Context, email string) (*models.Employee, error)
	getAllEmployeesByEmailFn      func(ctx context.Context, email string) ([]models.Employee, error)
	getUserByEmailFn              func(ctx context.Context, email string) (*models.User, error)
	getAllUsersByEmailFn          func(ctx context.Context, email string) ([]models.User, error)
}

func (m *mockAuthRepository) GetEmployeeByEmailAndTenant(ctx context.Context, tenantID, email string) (*models.Employee, error) {
	if m.getEmployeeByEmailAndTenantFn != nil {
		return m.getEmployeeByEmailAndTenantFn(ctx, tenantID, email)
	}
	return nil, nil
}

func (m *mockAuthRepository) GetEmployeeByEmail(ctx context.Context, email string) (*models.Employee, error) {
	if m.getEmployeeByEmailFn != nil {
		return m.getEmployeeByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *mockAuthRepository) GetAllEmployeesByEmail(ctx context.Context, email string) ([]models.Employee, error) {
	if m.getAllEmployeesByEmailFn != nil {
		return m.getAllEmployeesByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *mockAuthRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *mockAuthRepository) GetAllUsersByEmail(ctx context.Context, email string) ([]models.User, error) {
	if m.getAllUsersByEmailFn != nil {
		return m.getAllUsersByEmailFn(ctx, email)
	}
	return nil, nil
}

type mockTenantRepository struct {
	getByIDFn                  func(ctx context.Context, id string) (*models.Tenant, error)
	getBySubdomainFn           func(ctx context.Context, subdomain string) (*models.Tenant, error)
	getByNameAndSubdomainFn    func(ctx context.Context, name, subdomain string) (*models.Tenant, error)
	getByBrandIDFn             func(ctx context.Context, brandID string) ([]models.Tenant, error)
	getByBrandAndSubdomainFunc func(ctx context.Context, brandID, subdomain string) (*models.Tenant, error)
	createFn                   func(ctx context.Context, tenant *models.Tenant) error
	updateFn                   func(ctx context.Context, tenant *models.Tenant) error
	deleteFn                   func(ctx context.Context, id string) error
	getAllFn                   func(ctx context.Context) ([]models.Tenant, error)
}

func (m *mockTenantRepository) GetByID(
	ctx context.Context,
	id string,
) (*models.Tenant, error) {

	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}

	return &models.Tenant{ID: id, Active: true}, nil
}

func (m *mockTenantRepository) GetAll(
	ctx context.Context,
) ([]models.Tenant, error) {

	if m.getAllFn != nil {
		return m.getAllFn(ctx)
	}

	return nil, nil
}
func (m *mockTenantRepository) GetBySubdomain(
	ctx context.Context,
	subdomain string,
) (*models.Tenant, error) {

	if m.getBySubdomainFn != nil {
		return m.getBySubdomainFn(ctx, subdomain)
	}

	return nil, nil
}

func (m *mockTenantRepository) GetByNameAndSubdomain(
	ctx context.Context,
	name string,
	subdomain string,
) (*models.Tenant, error) {

	if m.getByNameAndSubdomainFn != nil {
		return m.getByNameAndSubdomainFn(ctx, name, subdomain)
	}

	return nil, nil
}

func (m *mockTenantRepository) GetByBrandAndSubdomain(
	ctx context.Context,
	brandID string,
	subdomain string,
) (*models.Tenant, error) {

	if m.getByBrandAndSubdomainFunc != nil {
		return m.getByBrandAndSubdomainFunc(ctx, brandID, subdomain)
	}

	return nil, nil
}

func (m *mockTenantRepository) GetByBrandID(ctx context.Context, brandID string) ([]models.Tenant, error) {
	if m.getByBrandIDFn != nil {
		return m.getByBrandIDFn(ctx, brandID)
	}
	return nil, nil
}

func (m *mockTenantRepository) Create(
	ctx context.Context,
	tenant *models.Tenant,
) error {

	if m.createFn != nil {
		return m.createFn(ctx, tenant)
	}

	return nil
}

func (m *mockTenantRepository) Update(
	ctx context.Context,
	tenant *models.Tenant,
) error {

	if m.updateFn != nil {
		return m.updateFn(ctx, tenant)
	}

	return nil
}

func (m *mockTenantRepository) Delete(
	ctx context.Context,
	id string,
) error {

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

func TestService_LoginWithUsersTable(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	password := "Secret123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("no se pudo hashear password: %v", err)
	}

	repoMock := &mockAuthRepository{
		getUserByEmailFn: func(ctx context.Context, email string) (*models.User, error) {
			return &models.User{
				ID:           "user-1",
				Email:        email,
				PasswordHash: string(hashedPassword),
				Role:         RoleOwner,
				IsActive:     true,
			}, nil
		},
	}

	svc, err := NewService(repoMock, &mockTenantRepository{})
	if err != nil {
		t.Fatalf("no se esperaba error al crear servicio: %v", err)
	}

	res, err := svc.Login(context.Background(), "", "owner@test.com", password)
	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo %v", err)
	}

	if res.Employee != nil {
		t.Fatalf("se esperaba que no venga employee para un login de user, se obtuvo %+v", res.Employee)
	}
	if res.User == nil || res.User.Email != "owner@test.com" || res.User.Role != RoleOwner {
		t.Fatalf("se esperaba payload de usuario para el owner, se obtuvo %+v", res.User)
	}
}

func TestService_Login_AutoUsesEmployeeTenantWhenTenantMissing(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	password := "Secret123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("no se pudo hashear password: %v", err)
	}

	tenantID := "tenant-uuid-1"
	repoMock := &mockAuthRepository{
		getEmployeeByEmailFn: func(ctx context.Context, email string) (*models.Employee, error) {
			return &models.Employee{
				ID:           "employee-uuid-1",
				TenantID:     &tenantID,
				Email:        email,
				PasswordHash: string(hashedPassword),
				Role:         "employee",
				IsActive:     true,
			}, nil
		},
	}

	svc, err := NewService(repoMock, &mockTenantRepository{})
	if err != nil {
		t.Fatalf("no se esperaba error al crear servicio: %v", err)
	}

	res, err := svc.Login(context.Background(), "", "employee@test.com", password)
	if err != nil {
		t.Fatalf("no se esperaba error para un empleado con tenant asociado, se obtuvo %v", err)
	}
	if res.Employee == nil || res.Employee.TenantID == nil || *res.Employee.TenantID != tenantID {
		t.Fatalf("se esperaba que el tenant del empleado se use automaticamente, se obtuvo %+v", res.Employee)
	}
}

func TestService_LookupTenantsForEmail_AllowsAdminUsers(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	password := "Secret123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("no se pudo hashear password: %v", err)
	}

	repoMock := &mockAuthRepository{
		getAllEmployeesByEmailFn: func(ctx context.Context, email string) ([]models.Employee, error) {
			return nil, nil
		},
		getAllUsersByEmailFn: func(ctx context.Context, email string) ([]models.User, error) {
			return []models.User{{
				ID:           "user-1",
				Email:        "admin@test.com",
				PasswordHash: string(hashedPassword),
				Role:         "admin",
				IsActive:     true,
			}}, nil
		},
	}

	svc, err := NewService(repoMock, &mockTenantRepository{})
	if err != nil {
		t.Fatalf("no se esperaba error al crear servicio: %v", err)
	}

	choices, err := svc.LookupTenantsForEmail(context.Background(), "admin@test.com", password)
	if err != nil {
		t.Fatalf("no se esperaba error para un admin user, se obtuvo %v", err)
	}
	if len(choices) != 0 {
		t.Fatalf("se esperaba una lista vacia de choices para un admin global, se obtuvo %+v", choices)
	}
}

func TestService_LookupTenantsForEmail_SkipsTenantSelectionForTenantEmployee(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	password := "Secret123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("no se pudo hashear password: %v", err)
	}

	tenantID := "tenant-uuid-1"
	repoMock := &mockAuthRepository{
		getAllEmployeesByEmailFn: func(ctx context.Context, email string) ([]models.Employee, error) {
			return []models.Employee{{
				ID:           "employee-uuid-1",
				TenantID:     &tenantID,
				Email:        email,
				PasswordHash: string(hashedPassword),
				Role:         "employee",
				IsActive:     true,
			}}, nil
		},
		getAllUsersByEmailFn: func(ctx context.Context, email string) ([]models.User, error) {
			return nil, nil
		},
	}

	svc, err := NewService(repoMock, &mockTenantRepository{})
	if err != nil {
		t.Fatalf("no se esperaba error al crear servicio: %v", err)
	}

	choices, err := svc.LookupTenantsForEmail(context.Background(), "employee@test.com", password)
	if err != nil {
		t.Fatalf("no se esperaba error para un empleado con tenant asociado, se obtuvo %v", err)
	}
	if len(choices) != 0 {
		t.Fatalf("se esperaba que no haya opciones de tienda para un empleado con tenant asociado, se obtuvo %+v", choices)
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
		ID:      validTenantID,
		BrandID: "brand-uuid-1",
		Active:  true,
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
			name:     "empleado sin tenant puede iniciar sesion como admin",
			tenantID: "",
			email:    validEmail,
			password: password,
			mockRepoGlobal: func(ctx context.Context, email string) (*models.Employee, error) {
				return dummyEmployee, nil
			},
			expectedErr:   nil,
			expectSuccess: true,
		},
		{
			name:     "empleado con tenant distinto puede iniciar sesion si es admin de la misma marca",
			tenantID: "tenant-uuid-2",
			email:    validEmail,
			password: password,
			mockTenant: func(ctx context.Context, id string) (*models.Tenant, error) {
				return &models.Tenant{ID: id, BrandID: "brand-uuid-1", Active: true}, nil
			},
			mockRepo: func(ctx context.Context, tenantID, email string) (*models.Employee, error) {
				return nil, nil
			},
			mockRepoGlobal: func(ctx context.Context, email string) (*models.Employee, error) {
				return dummyEmployee, nil
			},
			expectedErr:   nil,
			expectSuccess: true,
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
				getEmployeeByEmailAndTenantFn: tt.mockRepo,
				getEmployeeByEmailFn:          tt.mockRepoGlobal,
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
	brandID := "brand-1"
	token, err := GenerateToken("user-1", "user@test.com", "admin", &tenantID, &brandID)
	if err != nil {
		t.Fatalf("no se esperaba error generando token: %v", err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("no se esperaba error validando token: %v", err)
	}

	if claims.UserID != "user-1" || claims.Email != "user@test.com" || claims.Role != "admin" || claims.TenantID != tenantID || claims.BrandID == nil || *claims.BrandID != brandID {
		t.Fatalf("claims inesperados: %+v", claims)
	}
}

func TestGenerateToken_MissingSecret(t *testing.T) {
	previousSecret := jwtSecret
	t.Cleanup(func() {
		jwtSecret = previousSecret
	})

	jwtSecret = nil

	_, err := GenerateToken("user-1", "user@test.com", "admin", nil, nil)
	if !errors.Is(err, ErrJWTSecretRequired) {
		t.Fatalf("se esperaba ErrJWTSecretRequired generando token, se obtuvo %v", err)
	}
}
