package tenants

import (
	"context"
	"errors"
	"leguiburger/internal/models"
	"strings"
)

var ErrSubdomainAlreadyExists = errors.New("este subdominio ya está registrado por otro comercio")
var ErrTaxIdAlreadyExists = errors.New("este CUIT ya está registrado por otro comercio")
var ErrTenantNotFound = errors.New("el comercio especificado no existe")

type Service interface {
	RegisterTenant(ctx context.Context, name, subdomain string, taxId string) (*models.Tenant, error)
	UpdateTenant(ctx context.Context, id string, name string, subdomain string, tax_id string, active *bool) (*models.Tenant, error)
	DeleteTenant(ctx context.Context, id string) error
}

type service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return &service{repo: r}
}

func (s *service) RegisterTenant(ctx context.Context, name, subdomain string, taxID string) (*models.Tenant, error) {
	cleanSubdomain := strings.ToLower(strings.TrimSpace(subdomain))
	cleanTaxID := strings.ToLower(strings.TrimSpace(taxID))
	if cleanSubdomain == "" {
		return nil, errors.New("el subdominio no puede estar vacío")
	}

	if cleanTaxID == "" {
		return nil, errors.New("el tax ID no puede estar vacío")
	}

	existingSubdomain, _ := s.repo.GetBySubdomain(ctx, cleanSubdomain)
	if existingSubdomain != nil {
		return nil, ErrSubdomainAlreadyExists
	}

	existingTaxId, _ := s.repo.GetByTaxID(ctx, cleanSubdomain)
	if existingTaxId != nil {
		return nil, ErrTaxIdAlreadyExists
	}

	tenant := &models.Tenant{
		Name:      name,
		Subdomain: cleanSubdomain,
		TaxID:     cleanTaxID,
		Active:    true,
	}

	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (s *service) UpdateTenant(ctx context.Context, id string, name string, subdomain string, taxID string, active *bool) (*models.Tenant, error) {
	// 1. Buscamos el Tenant actual en la DB
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTenantNotFound
	}

	// 2. Si mandan un nombre no vacío, lo actualizamos
	if name != "" {
		tenant.Name = name
	}

	// 3. Si mandan un subdominio no vacío, validamos la unicidad global
	if subdomain != "" {
		cleanSubdomain := strings.ToLower(strings.TrimSpace(subdomain))

		// Validamos que si el subdominio realmente cambia, no esté duplicado
		if cleanSubdomain != tenant.Subdomain {
			existing, _ := s.repo.GetBySubdomain(ctx, cleanSubdomain)
			if existing != nil {
				return nil, ErrSubdomainAlreadyExists
			}
			tenant.Subdomain = cleanSubdomain
		}
	}

	// 4. Si mandan un Tax ID no vacío, simplemente lo limpiamos y lo asignamos.
	if taxID != "" {
		cleanTaxId := strings.ToLower(strings.TrimSpace(taxID))
		tenant.TaxID = cleanTaxId
	}

	// 5. Si mandan el estado active (puntero), lo actualizamos
	if active != nil {
		tenant.Active = *active
	}

	// 6. Guardamos los cambios en la DB
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
