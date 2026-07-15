package tenants

import (
	"context"
	"errors"
	"leguiburger/internal/models"
	"strings"

	"github.com/google/uuid"
)

var ErrTenantNotFound = errors.New("el comercio especificado no existe")
var ErrDuplicateBranch = errors.New("ya existe un registro idéntico (mismo comercio y subdominio) en la base de datos")

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

func (s *service) RegisterTenant(ctx context.Context, name, subdomain, taxID string) (*models.Tenant, error) {
	// 1. Normalizamos datos de entrada
	cleanSubdomain := strings.ToLower(strings.TrimSpace(subdomain))
	cleanName := strings.TrimSpace(name)

	// 2. Validamos que no exista la combinación de NAME + SUBDOMAIN
	existing, err := s.repo.GetByNameAndSubdomain(ctx, cleanName, cleanSubdomain)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		return nil, ErrDuplicateBranch
	}

	// 3. Si está libre, creamos el nuevo Tenant
	tenant := &models.Tenant{
		ID:        uuid.New().String(),
		Name:      cleanName,
		Subdomain: cleanSubdomain,
		TaxID:     strings.TrimSpace(taxID),
		Active:    true,
	}

	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (s *service) UpdateTenant(ctx context.Context, id string, name string, subdomain string, taxID string, active *bool) (*models.Tenant, error) {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTenantNotFound
	}

	// 1. Preparamos el nombre temporalmente para la validación cruzada
	tempName := tenant.Name
	if name != "" {
		tempName = strings.TrimSpace(name) // Normalizamos de paso
	}

	// 2. Validamos la unicidad del subdominio dentro del mismo comercio ("name")
	if subdomain != "" {
		cleanSubdomain := strings.ToLower(strings.TrimSpace(subdomain))

		if cleanSubdomain != tenant.Subdomain {
			// 💡 Ahora SÍ manejamos el error de forma segura en vez de usar "_"
			existing, err := s.repo.GetByNameAndSubdomain(ctx, tempName, cleanSubdomain)
			if err != nil {
				return nil, err
			}

			// 💡 Devolvemos ErrDuplicateBranch para que el Handler responda 409 Conflict
			if existing != nil && existing.ID != tenant.ID {
				return nil, ErrDuplicateBranch
			}
			tenant.Subdomain = cleanSubdomain
		}
	}

	// 3. Aplicamos los cambios restantes una vez superadas las validaciones
	if name != "" {
		tenant.Name = strings.TrimSpace(name)
	}

	if taxID != "" {
		tenant.TaxID = strings.ToLower(strings.TrimSpace(taxID))
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
