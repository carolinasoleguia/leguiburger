package extras

import (
	"context"
	"errors"
	"leguiburger/internal/models"
	"leguiburger/internal/tenants"
	"strings"
)

var (
	ErrExtraNotFound          = errors.New("extra no encontrado")
	ErrDuplicateExtraName     = errors.New("ya existe un extra con ese nombre para este comercio")
	ErrInvalidExtraData       = errors.New("el nombre del extra es obligatorio")
	ErrInvalidExtraPrice      = errors.New("el precio del extra no puede ser negativo")
	ErrInvalidExtraStock      = errors.New("el stock del extra no puede ser negativo")
	ErrTenantNotFoundForExtra = errors.New("el comercio (tenant) especificado no existe")
)

type Service interface {
	CreateExtra(ctx context.Context, tenantID, name string, currentPrice float64, currentStock int, trackStock *bool) (*models.Extra, error)
	GetExtra(ctx context.Context, tenantID, id string) (*models.Extra, error)
	ListExtras(ctx context.Context, tenantID string) ([]models.Extra, error)
	UpdateExtra(ctx context.Context, tenantID, id, name string, currentPrice *float64, currentStock *int, trackStock, isActive *bool) (*models.Extra, error)
	DeleteExtra(ctx context.Context, tenantID, id string) error
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

func (s *service) CreateExtra(ctx context.Context, tenantID, name string, currentPrice float64, currentStock int, trackStock *bool) (*models.Extra, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return nil, ErrInvalidExtraData
	}
	if currentPrice < 0 {
		return nil, ErrInvalidExtraPrice
	}
	if currentStock < 0 {
		return nil, ErrInvalidExtraStock
	}

	existing, err := s.repo.GetByName(ctx, tenantID, cleanName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateExtraName
	}

	shouldTrackStock := true
	if trackStock != nil {
		shouldTrackStock = *trackStock
	}

	extra := &models.Extra{
		TenantID:     tenantID,
		Name:         cleanName,
		CurrentPrice: currentPrice,
		CurrentStock: currentStock,
		TrackStock:   shouldTrackStock,
		IsActive:     true,
	}

	if err := s.repo.Create(ctx, extra); err != nil {
		if strings.Contains(err.Error(), "23503") || strings.Contains(err.Error(), "extras_tenant_id_fkey") {
			return nil, ErrTenantNotFoundForExtra
		}
		return nil, err
	}

	return extra, nil
}

func (s *service) GetExtra(ctx context.Context, tenantID, id string) (*models.Extra, error) {
	extra, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if extra == nil {
		return nil, ErrExtraNotFound
	}
	return extra, nil
}

func (s *service) ListExtras(ctx context.Context, tenantID string) ([]models.Extra, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFoundForExtra
	}

	return s.repo.FetchAll(ctx, tenantID)
}

func (s *service) UpdateExtra(ctx context.Context, tenantID, id, name string, currentPrice *float64, currentStock *int, trackStock, isActive *bool) (*models.Extra, error) {
	extra, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if extra == nil {
		return nil, ErrExtraNotFound
	}

	if currentPrice != nil {
		if *currentPrice < 0 {
			return nil, ErrInvalidExtraPrice
		}
		extra.CurrentPrice = *currentPrice
	}
	if currentStock != nil {
		if *currentStock < 0 {
			return nil, ErrInvalidExtraStock
		}
		extra.CurrentStock = *currentStock
	}
	if name != "" {
		cleanName := strings.TrimSpace(name)
		if cleanName == "" {
			return nil, ErrInvalidExtraData
		}
		if cleanName != extra.Name {
			existing, err := s.repo.GetByName(ctx, tenantID, cleanName)
			if err != nil {
				return nil, err
			}
			if existing != nil && existing.ID != extra.ID {
				return nil, ErrDuplicateExtraName
			}
			extra.Name = cleanName
		}
	}
	if trackStock != nil {
		extra.TrackStock = *trackStock
	}
	if isActive != nil {
		extra.IsActive = *isActive
	}

	if err := s.repo.Update(ctx, extra); err != nil {
		return nil, err
	}

	return extra, nil
}

func (s *service) DeleteExtra(ctx context.Context, tenantID, id string) error {
	extra, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if extra == nil {
		return ErrExtraNotFound
	}

	return s.repo.Delete(ctx, tenantID, id)
}
