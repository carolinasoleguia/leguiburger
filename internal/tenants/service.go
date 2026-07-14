package tenants

import (
	"context"
	"errors"
	"leguiburger/internal/models"
	"strings"
)

var ErrSubdomainAlreadyExists = errors.New("este subdominio ya está registrado por otro comercio")
var ErrTenantNotFound = errors.New("el comercio especificado no existe")

type Service interface {
	RegisterTenant(ctx context.Context, name, subdomain string) (*models.Tenant, error)
	UpdateTenant(ctx context.Context, id string, name string, subdomain string, active *bool) (*models.Tenant, error)
	DeleteTenant(ctx context.Context, id string) error
}

type service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return &service{repo: r}
}

func (s *service) RegisterTenant(ctx context.Context, name, subdomain string) (*models.Tenant, error) {
	cleanSubdomain := strings.ToLower(strings.TrimSpace(subdomain))
	if cleanSubdomain == "" {
		return nil, errors.New("el subdominio no puede estar vacío")
	}

	existing, _ := s.repo.GetBySubdomain(ctx, cleanSubdomain)
	if existing != nil {
		return nil, ErrSubdomainAlreadyExists
	}

	tenant := &models.Tenant{
		Name:      name,
		Subdomain: cleanSubdomain,
		Active:    true,
	}

	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (s *service) UpdateTenant(ctx context.Context, id string, name string, subdomain string, active *bool) (*models.Tenant, error) {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTenantNotFound
	}

	if name != "" {
		tenant.Name = name
	}

	if subdomain != "" {
		cleanSubdomain := strings.ToLower(strings.TrimSpace(subdomain))
		// Validar que si cambia el subdominio, no exista ya en otro tenant
		if cleanSubdomain != tenant.Subdomain {
			existing, _ := s.repo.GetBySubdomain(ctx, cleanSubdomain)
			if existing != nil {
				return nil, ErrSubdomainAlreadyExists
			}
			tenant.Subdomain = cleanSubdomain
		}
	}

	if active != nil {
		tenant.Active = *active
	}

	if err := s.repo.Update(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (s *service) DeleteTenant(ctx context.Context, id string) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrTenantNotFound
	}
	return s.repo.Delete(ctx, id)
}
