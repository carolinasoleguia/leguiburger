package employees

import (
	"context"
	"errors"
	"leguiburger/internal/models"
	"leguiburger/internal/tenants"
	"strings"
)

var (
	ErrEmployeeNotFound          = errors.New("empleado no encontrado")
	ErrDuplicateEmployeeEmail    = errors.New("ya existe un empleado con ese email")
	ErrInvalidEmployeeData       = errors.New("nombre, apellido, email y password_hash son obligatorios")
	ErrInvalidEmployeeRole       = errors.New("el rol del empleado no es válido")
	ErrTenantNotFoundForEmployee = errors.New("el comercio (tenant) especificado no existe")
)

type Service interface {
	CreateEmployee(ctx context.Context, tenantID, firstName, lastName, email, passwordHash, phone, role string) (*models.Employee, error)
	GetEmployee(ctx context.Context, tenantID, id string) (*models.Employee, error)
	ListEmployees(ctx context.Context, tenantID string) ([]models.Employee, error)
	UpdateEmployee(ctx context.Context, tenantID, id, firstName, lastName, email, passwordHash, phone, role string, isActive *bool) (*models.Employee, error)
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

func (s *service) CreateEmployee(ctx context.Context, tenantID, firstName, lastName, email, passwordHash, phone, role string) (*models.Employee, error) {
	cleanFirstName := strings.TrimSpace(firstName)
	cleanLastName := strings.TrimSpace(lastName)
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	cleanPasswordHash := strings.TrimSpace(passwordHash)
	cleanRole := normalizeRole(role)

	if cleanFirstName == "" || cleanLastName == "" || cleanEmail == "" || cleanPasswordHash == "" {
		return nil, ErrInvalidEmployeeData
	}
	if !isValidRole(cleanRole) {
		return nil, ErrInvalidEmployeeRole
	}

	existing, err := s.repo.GetByEmail(ctx, cleanEmail)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateEmployeeEmail
	}

	employee := &models.Employee{
		TenantID:     tenantID,
		FirstName:    cleanFirstName,
		LastName:     cleanLastName,
		Email:        cleanEmail,
		PasswordHash: cleanPasswordHash,
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

func (s *service) UpdateEmployee(ctx context.Context, tenantID, id, firstName, lastName, email, passwordHash, phone, role string, isActive *bool) (*models.Employee, error) {
	employee, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if employee == nil {
		return nil, ErrEmployeeNotFound
	}

	if firstName != "" {
		employee.FirstName = strings.TrimSpace(firstName)
	}
	if lastName != "" {
		employee.LastName = strings.TrimSpace(lastName)
	}
	if passwordHash != "" {
		employee.PasswordHash = strings.TrimSpace(passwordHash)
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
			existing, err := s.repo.GetByEmail(ctx, cleanEmail)
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
	employee, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if employee == nil {
		return ErrEmployeeNotFound
	}

	return s.repo.Delete(ctx, tenantID, id)
}

func normalizeRole(role string) string {
	cleanRole := strings.ToLower(strings.TrimSpace(role))
	if cleanRole == "" {
		return "employee"
	}
	return cleanRole
}

func isValidRole(role string) bool {
	switch role {
	case "admin", "cashier", "kitchen", "employee":
		return true
	default:
		return false
	}
}
