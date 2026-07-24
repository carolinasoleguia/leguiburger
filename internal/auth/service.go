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
	Employee EmployeeDTO `json:"employee"`
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

type Service interface {
	Login(ctx context.Context, tenantID, email, password string) (*LoginResponse, error)
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

	employee, err := s.findEmployeeForLogin(ctx, cleanTenantID, cleanEmail)
	if err != nil {
		return nil, err
	}

	if !isGlobalRole(employee.Role) && (employee.TenantID == nil || *employee.TenantID != cleanTenantID) {
		return nil, ErrForbiddenTenant
	}

	if err := bcrypt.CompareHashAndPassword([]byte(employee.PasswordHash), []byte(cleanPassword)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := GenerateToken(
		employee.ID,
		employee.Email,
		employee.Role,
		employee.TenantID,
	)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:    token,
		Employee: toEmployeeDTO(employee),
	}, nil
}

func (s *service) findEmployeeForLogin(ctx context.Context, tenantID, email string) (*models.Employee, error) {
	if tenantID == "" {
		employee, err := s.repo.GetByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		if employee == nil {
			return nil, ErrInvalidCredentials
		}
		if !isGlobalRole(employee.Role) {
			return nil, ErrTenantRequired
		}
		return employee, nil
	}

	if err := s.validateTenant(ctx, tenantID); err != nil {
		return nil, err
	}

	employee, err := s.repo.GetByEmailAndTenant(ctx, tenantID, email)
	if err != nil {
		return nil, err
	}
	if employee != nil {
		return employee, nil
	}

	globalEmployee, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if globalEmployee == nil {
		return nil, ErrInvalidCredentials
	}
	if !isGlobalRole(globalEmployee.Role) {
		return nil, ErrForbiddenTenant
	}

	return globalEmployee, nil
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
