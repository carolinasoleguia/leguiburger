package customers

import (
	"context"
	"errors"
	"leguiburger/internal/models"
	"leguiburger/internal/tenants"
	"strings"
)

var (
	ErrCustomerNotFound          = errors.New("cliente no encontrado")
	ErrDuplicateCustomerEmail    = errors.New("ya existe un cliente con ese email para este comercio")
	ErrInvalidCustomerData       = errors.New("nombre, apellido y email son obligatorios")
	ErrTenantNotFoundForCustomer = errors.New("el comercio (tenant) especificado no existe")
)

type Service interface {
	CreateCustomer(ctx context.Context, tenantID, firstName, lastName, email, phone string) (*models.Customer, error)
	GetCustomer(ctx context.Context, tenantID, id string) (*models.Customer, error)
	ListCustomers(ctx context.Context, tenantID string) ([]models.Customer, error)
	UpdateCustomer(ctx context.Context, tenantID, id, firstName, lastName, email, phone string) (*models.Customer, error)
	DeleteCustomer(ctx context.Context, tenantID, id string) error
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

func (s *service) CreateCustomer(ctx context.Context, tenantID, firstName, lastName, email, phone string) (*models.Customer, error) {
	cleanFirstName := strings.TrimSpace(firstName)
	cleanLastName := strings.TrimSpace(lastName)
	cleanEmail := strings.ToLower(strings.TrimSpace(email))

	if cleanFirstName == "" || cleanLastName == "" || cleanEmail == "" {
		return nil, ErrInvalidCustomerData
	}

	existing, err := s.repo.GetByEmail(ctx, tenantID, cleanEmail)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateCustomerEmail
	}

	customer := &models.Customer{
		TenantID:  tenantID,
		FirstName: cleanFirstName,
		LastName:  cleanLastName,
		Email:     cleanEmail,
		Phone:     strings.TrimSpace(phone),
	}

	if err := s.repo.Create(ctx, customer); err != nil {
		if strings.Contains(err.Error(), "23503") || strings.Contains(err.Error(), "customers_tenant_id_fkey") {
			return nil, ErrTenantNotFoundForCustomer
		}
		return nil, err
	}

	return customer, nil
}

func (s *service) GetCustomer(ctx context.Context, tenantID, id string) (*models.Customer, error) {
	customer, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, ErrCustomerNotFound
	}
	return customer, nil
}

func (s *service) ListCustomers(ctx context.Context, tenantID string) ([]models.Customer, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFoundForCustomer
	}

	return s.repo.FetchAll(ctx, tenantID)
}

func (s *service) UpdateCustomer(ctx context.Context, tenantID, id, firstName, lastName, email, phone string) (*models.Customer, error) {
	customer, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, ErrCustomerNotFound
	}

	if firstName != "" {
		customer.FirstName = strings.TrimSpace(firstName)
	}
	if lastName != "" {
		customer.LastName = strings.TrimSpace(lastName)
	}
	if phone != "" {
		customer.Phone = strings.TrimSpace(phone)
	}
	if email != "" {
		cleanEmail := strings.ToLower(strings.TrimSpace(email))
		if cleanEmail == "" {
			return nil, ErrInvalidCustomerData
		}
		if cleanEmail != customer.Email {
			existing, err := s.repo.GetByEmail(ctx, tenantID, cleanEmail)
			if err != nil {
				return nil, err
			}
			if existing != nil && existing.ID != customer.ID {
				return nil, ErrDuplicateCustomerEmail
			}
			customer.Email = cleanEmail
		}
	}

	if err := s.repo.Update(ctx, customer); err != nil {
		return nil, err
	}

	return customer, nil
}

func (s *service) DeleteCustomer(ctx context.Context, tenantID, id string) error {
	customer, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if customer == nil {
		return ErrCustomerNotFound
	}

	return s.repo.Delete(ctx, tenantID, id)
}
