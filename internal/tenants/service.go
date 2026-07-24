package tenants

import (
	"context"
	"errors"
	"leguiburger/internal/brands"
	"leguiburger/internal/models"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrTenantNotFound  = errors.New("el comercio especificado no existe")
	ErrDuplicateBranch = errors.New("ya existe un local registrado con esa marca y subdominio")
)

type Service interface {
	RegisterTenant(
		ctx context.Context,
		brandID string,
		subdomain string,
	) (*models.Tenant, error)

	UpdateTenant(
		ctx context.Context,
		id string,
		subdomain string,
		active *bool,
	) (*models.Tenant, error)

	DeleteTenant(
		ctx context.Context,
		id string,
	) error

	GetAllTenants(
		ctx context.Context,
	) ([]models.Tenant, error)
}

type service struct {
	repo      Repository
	brandRepo brands.Repository
}

func NewService(
	r Repository,
	brandRepo brands.Repository,
) Service {

	return &service{
		repo:      r,
		brandRepo: brandRepo,
	}
}

// CREATE

func (s *service) RegisterTenant(
	ctx context.Context,
	brandID string,
	subdomain string,
) (*models.Tenant, error) {

	cleanSubdomain := strings.ToLower(
		strings.TrimSpace(subdomain),
	)

	brand, err := s.brandRepo.GetByID(
		ctx,
		brandID,
	)

	if err != nil {
		return nil, err
	}

	if brand == nil {
		return nil, errors.New("marca inexistente")
	}

	existing, err := s.repo.GetByBrandAndSubdomain(
		ctx,
		brandID,
		cleanSubdomain,
	)

	if err != nil {
		return nil, err
	}

	if existing != nil {
		return nil, ErrDuplicateBranch
	}

	tenant := &models.Tenant{
		ID:        uuid.New().String(),
		BrandID:   brandID,
		Subdomain: cleanSubdomain,
		Active:    true,
	}

	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

// LIST

func (s *service) GetAllTenants(
	ctx context.Context,
) ([]models.Tenant, error) {

	return s.repo.GetAll(ctx)
}

// UPDATE

func (s *service) UpdateTenant(
	ctx context.Context,
	id string,
	subdomain string,
	active *bool,
) (*models.Tenant, error) {

	tenant, err := s.repo.GetByID(ctx, id)

	if err != nil {
		return nil, ErrTenantNotFound
	}

	if tenant == nil {
		return nil, ErrTenantNotFound
	}

	if subdomain != "" {

		cleanSubdomain := strings.ToLower(
			strings.TrimSpace(subdomain),
		)

		if cleanSubdomain != tenant.Subdomain {

			existing, err := s.repo.GetByBrandAndSubdomain(
				ctx,
				tenant.BrandID,
				cleanSubdomain,
			)

			if err != nil {
				return nil, err
			}

			if existing != nil && existing.ID != tenant.ID {
				return nil, ErrDuplicateBranch
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

// DELETE

func (s *service) DeleteTenant(
	ctx context.Context,
	id string,
) error {

	tenant, err := s.repo.GetByID(
		ctx,
		id,
	)

	if err != nil || tenant == nil {
		return ErrTenantNotFound
	}

	return s.repo.Delete(
		ctx,
		id,
	)
}
