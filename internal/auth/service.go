package auth

import (
	"context"
	"errors"
	"os"
	"strings"

	"leguiburger/internal/models"
	"leguiburger/internal/tenants"

	"golang.org/x/crypto/bcrypt"
)

const (
	TenantHeaderName = "X-Tenant-ID"
	RoleOwner        = "owner"
	RoleSuperAdmin   = "super_admin"
)

var (
	ErrInvalidCredentials    = errors.New("credenciales invalidas")
	ErrTenantRequired        = errors.New("el tenant es requerido para este usuario")
	ErrForbiddenTenant       = errors.New("no autorizado para este comercio")
	ErrTenantNotFoundForAuth = errors.New("el comercio especificado no existe")
	ErrJWTSecretRequired     = errors.New("JWT_SECRET no configurado")
)

type LoginResponse struct {
	Token    string      `json:"token"`
	Employee *EmployeeDTO `json:"employee,omitempty"`
	User     *UserDTO    `json:"user,omitempty"`
}

type UserDTO struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
	BrandID   *string `json:"brand_id,omitempty"`
	IsActive  bool    `json:"is_active"`
}

type EmployeeDTO struct {
	ID        string  `json:"id"`
	TenantID  *string `json:"tenant_id,omitempty"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     string  `json:"email"`
	Phone     string  `json:"phone"`
	Role      string  `json:"role"`
	IsActive  bool    `json:"is_active"`
}

type TenantChoice struct {
	TenantID string `json:"tenant_id"`
	Label    string `json:"label"`
}

type Service interface {
	Login(ctx context.Context, tenantID, email, password string) (*LoginResponse, error)
	LookupTenantsForEmail(ctx context.Context, email, password string) ([]TenantChoice, error)
}

type service struct {
	repo       Repository
	tenantRepo tenants.Repository
}

func NewService(repo Repository, tenantRepo tenants.Repository) (Service, error) {
	if err := ConfigureJWTSecret(os.Getenv("JWT_SECRET")); err != nil {
		return nil, err
	}

	return &service{repo: repo, tenantRepo: tenantRepo}, nil
}

func (s *service) Login(ctx context.Context, tenantID, email, password string) (*LoginResponse, error) {
	cleanTenantID := strings.TrimSpace(tenantID)
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	cleanPassword := strings.TrimSpace(password)

	if cleanEmail == "" || cleanPassword == "" {
		return nil, ErrInvalidCredentials
	}

	loginUser, err := s.findLoginPrincipal(ctx, cleanTenantID, cleanEmail)
	if err != nil {
		return nil, err
	}

	if loginUser.employee != nil {
		if cleanTenantID == "" && loginUser.employee.TenantID != nil {
			cleanTenantID = *loginUser.employee.TenantID
		}

		if !isGlobalRole(loginUser.employee.Role) && !isBrandOwnerRole(loginUser.employee.Role) && (loginUser.employee.TenantID == nil || *loginUser.employee.TenantID != cleanTenantID) {
			return nil, ErrForbiddenTenant
		}

		if err := bcrypt.CompareHashAndPassword([]byte(loginUser.employee.PasswordHash), []byte(cleanPassword)); err != nil {
			return nil, ErrInvalidCredentials
		}

		if cleanTenantID == "" && !isGlobalRole(loginUser.employee.Role) && !isBrandOwnerRole(loginUser.employee.Role) {
			return nil, ErrTenantRequired
		}

		token, err := GenerateToken(
			loginUser.employee.ID,
			loginUser.employee.Email,
			loginUser.employee.Role,
			loginUser.employee.TenantID,
			nil,
		)
		if err != nil {
			return nil, err
		}

		employeeDTO := toEmployeeDTO(loginUser.employee)
		return &LoginResponse{
			Token:    token,
			Employee: &employeeDTO,
		}, nil
	}

	if loginUser.user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(loginUser.user.PasswordHash), []byte(cleanPassword)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := GenerateToken(
		loginUser.user.ID,
		loginUser.user.Email,
		loginUser.user.Role,
		nil,
		loginUser.user.BrandID,
	)
	if err != nil {
		return nil, err
	}

	userDTO := &UserDTO{ID: loginUser.user.ID, Email: loginUser.user.Email, Role: loginUser.user.Role, BrandID: loginUser.user.BrandID, IsActive: loginUser.user.IsActive}
	return &LoginResponse{
		Token: token,
		User:  userDTO,
	}, nil
}

func (s *service) LookupTenantsForEmail(ctx context.Context, email, password string) ([]TenantChoice, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	cleanPassword := strings.TrimSpace(password)

	if cleanEmail == "" || cleanPassword == "" {
		return nil, ErrInvalidCredentials
	}

	employees, err := s.repo.GetAllEmployeesByEmail(ctx, cleanEmail)
	if err != nil {
		return nil, err
	}
	users, err := s.repo.GetAllUsersByEmail(ctx, cleanEmail)
	if err != nil {
		return nil, err
	}
	if len(employees) == 0 && len(users) == 0 {
		return nil, ErrInvalidCredentials
	}

	choicesMap := map[string]TenantChoice{}
	matchedGlobal := false

	for _, employee := range employees {
		if employee.PasswordHash == "" {
			continue
		}

		if bcrypt.CompareHashAndPassword([]byte(employee.PasswordHash), []byte(cleanPassword)) != nil {
			continue
		}

		if isGlobalRole(employee.Role) || (isBrandOwnerRole(employee.Role) && employee.TenantID == nil) {
			matchedGlobal = true
			continue
		}

		if employee.TenantID == nil {
			continue
		}

		tenant, err := s.getActiveTenant(ctx, *employee.TenantID)
		if err != nil || tenant == nil {
			continue
		}

		choicesMap[tenant.ID] = TenantChoice{
			TenantID: tenant.ID,
			Label:    tenant.Subdomain,
		}
		if err != nil || tenant == nil {
			continue
		}

		choicesMap[tenant.ID] = TenantChoice{
			TenantID: tenant.ID,
			Label:    tenant.Subdomain,
		}
	}

	for _, user := range users {
		if user.PasswordHash == "" {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(cleanPassword)) != nil {
			continue
		}
		if isGlobalRole(user.Role) || isBrandOwnerRole(user.Role) {
			matchedGlobal = true
		}
	}

	if len(choicesMap) == 0 {
		if matchedGlobal {
			return []TenantChoice{}, nil
		}
		return []TenantChoice{}, nil
	}

	choices := make([]TenantChoice, 0, len(choicesMap))
	for _, choice := range choicesMap {
		choices = append(choices, choice)
	}

	return choices, nil
}

func (s *service) getActiveTenant(ctx context.Context, tenantID string) (*models.Tenant, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil || !tenant.Active {
		return nil, nil
	}
	return tenant, nil
}

type loginPrincipal struct {
	employee *models.Employee
	user     *models.User
}

func (s *service) findLoginPrincipal(ctx context.Context, tenantID, email string) (*loginPrincipal, error) {
	if tenantID == "" {
		user, err := s.repo.GetUserByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		if user != nil {
			return &loginPrincipal{user: user}, nil
		}

		employee, err := s.repo.GetEmployeeByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		if employee == nil {
			return nil, ErrInvalidCredentials
		}
		if employee.TenantID != nil {
			if err := s.validateTenant(ctx, *employee.TenantID); err != nil {
				return nil, err
			}
			return &loginPrincipal{employee: employee}, nil
		}
		if isGlobalRole(employee.Role) || isBrandOwnerRole(employee.Role) {
			return &loginPrincipal{employee: employee}, nil
		}
		return nil, ErrTenantRequired
	}

	if err := s.validateTenant(ctx, tenantID); err != nil {
		return nil, err
	}

	employee, err := s.repo.GetEmployeeByEmailAndTenant(ctx, tenantID, email)
	if err != nil {
		return nil, err
	}
	if employee != nil {
		return &loginPrincipal{employee: employee}, nil
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return &loginPrincipal{user: user}, nil
	}

	globalEmployee, err := s.repo.GetEmployeeByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if globalEmployee == nil {
		return nil, ErrInvalidCredentials
	}
	if isGlobalRole(globalEmployee.Role) {
		return &loginPrincipal{employee: globalEmployee}, nil
	}
	if isBrandOwnerRole(globalEmployee.Role) {
		if globalEmployee.TenantID == nil {
			return nil, ErrInvalidCredentials
		}
		sameBrand, err := s.isTenantInSameBrand(ctx, tenantID, *globalEmployee.TenantID)
		if err != nil {
			return nil, err
		}
		if sameBrand {
			return &loginPrincipal{employee: globalEmployee}, nil
		}
	}

	return nil, ErrForbiddenTenant
}

func (s *service) isTenantInSameBrand(ctx context.Context, tenantID, otherTenantID string) (bool, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return false, err
	}
	if tenant == nil || !tenant.Active {
		return false, ErrTenantNotFoundForAuth
	}

	otherTenant, err := s.tenantRepo.GetByID(ctx, otherTenantID)
	if err != nil {
		return false, err
	}
	if otherTenant == nil || !otherTenant.Active {
		return false, ErrTenantNotFoundForAuth
	}

	return tenant.BrandID == otherTenant.BrandID, nil
}

func (s *service) validateTenant(ctx context.Context, tenantID string) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant == nil || !tenant.Active {
		return ErrTenantNotFoundForAuth
	}
	return nil
}

func isGlobalRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case RoleOwner, RoleSuperAdmin:
		return true
	default:
		return false
	}
}

func isBrandOwnerRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin":
		return true
	default:
		return false
	}
}

func toEmployeeDTO(employee *models.Employee) EmployeeDTO {
	return EmployeeDTO{
		ID:        employee.ID,
		TenantID:  employee.TenantID,
		FirstName: employee.FirstName,
		LastName:  employee.LastName,
		Email:     employee.Email,
		Phone:     employee.Phone,
		Role:      employee.Role,
		IsActive:  employee.IsActive,
	}
}
