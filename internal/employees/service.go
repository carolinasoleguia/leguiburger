package employees

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"leguiburger/internal/auth"
	"leguiburger/internal/models"
	"leguiburger/internal/tenants"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmployeeNotFound          = errors.New("empleado no encontrado")
	ErrDuplicateEmployeeEmail    = errors.New("ya existe un empleado con ese email")
	ErrInvalidEmployeeData       = errors.New("nombre, apellido, email y password son obligatorios")
	ErrInvalidEmployeeRole       = errors.New("el rol del empleado no es válido")
	ErrTenantNotFoundForEmployee = errors.New("el comercio (tenant) especificado no existe")
	ErrUnauthorizedAction        = errors.New("no tienes permisos para realizar esta acción sobre este usuario")
)

type Service interface {
	CreateEmployee(ctx context.Context, tenantID, firstName, lastName, email, password, phone, role string) (*models.Employee, error)
	GetEmployee(ctx context.Context, tenantID, id string) (*models.Employee, error)
	ListEmployees(ctx context.Context, tenantID string) ([]models.Employee, error)
	GetAllEmployees(ctx context.Context) ([]models.Employee, error) // <--- Agregado a la interfaz
	UpdateEmployee(ctx context.Context, tenantID, id, firstName, lastName, email, password, phone, role string, isActive *bool) (*models.Employee, error)
	DeleteEmployee(ctx context.Context, tenantID, id string) error
}

type service struct {
	repo       Repository
	tenantRepo tenants.Repository
}

func NewService(repo Repository, tenantRepo tenants.Repository) Service {
	return &service{
		repo:       repo,
		tenantRepo: tenantRepo,
	}
}

func (s *service) CreateEmployee(ctx context.Context, tenantID, firstName, lastName, email, password, phone, role string) (*models.Employee, error) {
	cleanFirstName := strings.TrimSpace(firstName)
	cleanLastName := strings.TrimSpace(lastName)
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	cleanPassword := strings.TrimSpace(password)
	cleanRole := normalizeRole(role)
	cleanTenantID := strings.TrimSpace(tenantID)

	if cleanFirstName == "" || cleanLastName == "" || cleanEmail == "" || cleanPassword == "" {
		return nil, ErrInvalidEmployeeData
	}
	if !isValidRole(cleanRole) {
		return nil, ErrInvalidEmployeeRole
	}

	claims, ok := auth.GetClaimsFromContext(ctx)
	if ok {
		actorRole := strings.ToLower(strings.TrimSpace(claims.Role))
		actorTenantID := strings.TrimSpace(claims.TenantID)
		actorBrandID := ""
		if claims.BrandID != nil {
			actorBrandID = strings.TrimSpace(*claims.BrandID)
		}

		if actorRole != auth.RoleOwner {
			if cleanTenantID == "" {
				cleanTenantID = actorTenantID
			}
			if cleanTenantID == "" {
				return nil, ErrTenantNotFoundForEmployee
			}
			if !s.isAllowedTenantForActor(ctx, cleanTenantID, actorRole, actorTenantID, actorBrandID) {
				return nil, ErrUnauthorizedAction
			}
			if getRoleWeight(actorRole) <= getRoleWeight(cleanRole) {
				return nil, ErrUnauthorizedAction
			}
		}
	} else {
		if cleanTenantID == "" && cleanRole != auth.RoleOwner && cleanRole != auth.RoleSuperAdmin {
			return nil, ErrTenantNotFoundForEmployee
		}
	}

	var tenantPtr *string
	if cleanTenantID != "" {
		tenantPtr = &cleanTenantID
	}

	existing, err := s.repo.GetByEmail(ctx, cleanTenantID, cleanEmail)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateEmployeeEmail
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(cleanPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error al hashear la contraseña: %w", err)
	}

	employee := &models.Employee{
		TenantID:     tenantPtr,
		FirstName:    cleanFirstName,
		LastName:     cleanLastName,
		Email:        cleanEmail,
		PasswordHash: string(hashedBytes),
		Phone:        strings.TrimSpace(phone),
		Role:         cleanRole,
		IsActive:     true,
	}

	if err := s.repo.Create(ctx, employee); err != nil {
		if strings.Contains(err.Error(), "23503") || strings.Contains(err.Error(), "employees_tenant_id_fkey") {
			return nil, ErrTenantNotFoundForEmployee
		}
		return nil, err
	}

	return employee, nil
}

func (s *service) GetEmployee(ctx context.Context, tenantID, id string) (*models.Employee, error) {
	employee, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if employee == nil {
		return nil, ErrEmployeeNotFound
	}
	return employee, nil
}

func (s *service) ListEmployees(ctx context.Context, tenantID string) ([]models.Employee, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFoundForEmployee
	}

	return s.repo.FetchAll(ctx, tenantID)
}

func (s *service) GetAllEmployees(ctx context.Context) ([]models.Employee, error) {
	employees, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return employees, nil
}

func (s *service) UpdateEmployee(ctx context.Context, tenantID, id, firstName, lastName, email, password, phone, role string, isActive *bool) (*models.Employee, error) {
	actorRole := ""
	actorTenantID := ""
	actorBrandID := ""
	claims, ok := auth.GetClaimsFromContext(ctx)
	if ok {
		actorRole = strings.ToLower(strings.TrimSpace(claims.Role))
		actorTenantID = strings.TrimSpace(claims.TenantID)
		if claims.BrandID != nil {
			actorBrandID = strings.TrimSpace(*claims.BrandID)
		}

		if actorRole != auth.RoleOwner {
			if tenantID == "" {
				tenantID = actorTenantID
			}
			if tenantID == "" {
				return nil, ErrUnauthorizedAction
			}
			if !s.isAllowedTenantForActor(ctx, tenantID, actorRole, actorTenantID, actorBrandID) {
				return nil, ErrUnauthorizedAction
			}
		}
	}

	employee, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if employee == nil {
		return nil, ErrEmployeeNotFound
	}

	if actorRole != auth.RoleOwner && getRoleWeight(actorRole) <= getRoleWeight(employee.Role) {
		if actorRole != employee.Role {
			return nil, ErrUnauthorizedAction
		}
	}

	if firstName != "" {
		employee.FirstName = strings.TrimSpace(firstName)
	}
	if lastName != "" {
		employee.LastName = strings.TrimSpace(lastName)
	}

	cleanPassword := strings.TrimSpace(password)
	if cleanPassword != "" {
		hashedBytes, err := bcrypt.GenerateFromPassword([]byte(cleanPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("error al hashear la contraseña: %w", err)
		}
		employee.PasswordHash = string(hashedBytes)
	}

	if phone != "" {
		employee.Phone = strings.TrimSpace(phone)
	}
	if role != "" {
		cleanRole := normalizeRole(role)
		if !isValidRole(cleanRole) {
			return nil, ErrInvalidEmployeeRole
		}
		employee.Role = cleanRole
	}
	if email != "" {
		cleanEmail := strings.ToLower(strings.TrimSpace(email))
		if cleanEmail == "" {
			return nil, ErrInvalidEmployeeData
		}
		if cleanEmail != employee.Email {
			existing, err := s.repo.GetByEmail(ctx, tenantID, cleanEmail)
			if err != nil {
				return nil, err
			}
			if existing != nil && existing.ID != employee.ID {
				return nil, ErrDuplicateEmployeeEmail
			}
			employee.Email = cleanEmail
		}
	}
	if isActive != nil {
		employee.IsActive = *isActive
	}

	if err := s.repo.Update(ctx, employee); err != nil {
		return nil, err
	}

	return employee, nil
}

func (s *service) DeleteEmployee(ctx context.Context, tenantID, id string) error {
	actorRole := ""
	actorTenantID := ""
	actorBrandID := ""
	claims, ok := auth.GetClaimsFromContext(ctx)
	if ok {
		actorRole = strings.ToLower(strings.TrimSpace(claims.Role))
		actorTenantID = strings.TrimSpace(claims.TenantID)
		if claims.BrandID != nil {
			actorBrandID = strings.TrimSpace(*claims.BrandID)
		}

		if actorRole != auth.RoleOwner {
			if tenantID == "" {
				tenantID = actorTenantID
			}
			if tenantID == "" {
				return ErrUnauthorizedAction
			}
			if !s.isAllowedTenantForActor(ctx, tenantID, actorRole, actorTenantID, actorBrandID) {
				return ErrUnauthorizedAction
			}
		}
	}

	employee, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if employee == nil {
		return ErrEmployeeNotFound
	}

	if actorRole != auth.RoleOwner && getRoleWeight(actorRole) <= getRoleWeight(employee.Role) {
		return ErrUnauthorizedAction
	}

	return s.repo.Delete(ctx, tenantID, id)
}

func (s *service) isAllowedTenantForActor(ctx context.Context, tenantID, actorRole, actorTenantID, actorBrandID string) bool {
	if actorRole == auth.RoleOwner {
		return true
	}
	if strings.TrimSpace(tenantID) == "" {
		return false
	}
	if strings.TrimSpace(actorTenantID) != "" && strings.TrimSpace(tenantID) == strings.TrimSpace(actorTenantID) {
		return true
	}
	if strings.EqualFold(actorRole, "admin") {
		tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
		if err != nil || tenant == nil {
			return false
		}
		if strings.TrimSpace(actorBrandID) != "" {
			return strings.TrimSpace(tenant.BrandID) == strings.TrimSpace(actorBrandID)
		}
		if strings.TrimSpace(actorTenantID) == "" {
			return false
		}
		actorTenant, err := s.tenantRepo.GetByID(ctx, actorTenantID)
		if err != nil || actorTenant == nil {
			return false
		}
		return tenant.BrandID == actorTenant.BrandID
	}
	return false
}

func normalizeRole(role string) string {
	r := strings.ToLower(strings.TrimSpace(role))
	if r == "" {
		return "employee"
	}
	return r
}

func isValidRole(role string) bool {
	switch role {
	case "employee", "cashier", "kitchen", "admin", "owner", "super_admin":
		return true
	default:
		return false
	}
}

func getRoleWeight(role string) int {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "owner", "super_admin":
		return 3
	case "admin":
		return 2
	default:
		return 1
	}
}
