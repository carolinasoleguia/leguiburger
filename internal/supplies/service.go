package supplies

import (
	"context"
	"errors"
	"strings"

	"leguiburger/internal/models"
	"leguiburger/internal/tenants"
)

var (
	ErrSupplyNotFound          = errors.New("insumo no encontrado")
	ErrDuplicateSupplyName     = errors.New("ya existe un insumo con ese nombre para este comercio")
	ErrInvalidSupplyData       = errors.New("el nombre y la unidad de medida son obligatorios")
	ErrInvalidSupplyCost       = errors.New("el costo mayorista no puede ser negativo")
	ErrInvalidSupplyStock      = errors.New("el stock del insumo no puede ser negativo")
	ErrTenantNotFoundForSupply = errors.New("el comercio (tenant) especificado no existe")
)

type Service interface {
	CreateSupply(ctx context.Context, tenantID, name string, currentWholesaleCost, currentStock float64, measurementUnit string) (*models.Supply, error)
	GetSupply(ctx context.Context, tenantID, id string) (*models.Supply, error)
	ListSupplies(ctx context.Context, tenantID string) ([]models.Supply, error)
	UpdateSupply(ctx context.Context, tenantID, id, name string, currentWholesaleCost, currentStock *float64, measurementUnit string, isActive *bool) (*models.Supply, error)
	DeleteSupply(ctx context.Context, tenantID, id string) error
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

func (s *service) CreateSupply(ctx context.Context, tenantID, name string, currentWholesaleCost, currentStock float64, measurementUnit string) (*models.Supply, error) {
	cleanName := strings.TrimSpace(name)
	cleanUnit := strings.ToLower(strings.TrimSpace(measurementUnit))
	if cleanName == "" || cleanUnit == "" {
		return nil, ErrInvalidSupplyData
	}
	if currentWholesaleCost < 0 {
		return nil, ErrInvalidSupplyCost
	}
	if currentStock < 0 {
		return nil, ErrInvalidSupplyStock
	}

	existing, err := s.repo.GetByName(ctx, tenantID, cleanName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateSupplyName
	}

	supply := &models.Supply{
		TenantID:             tenantID,
		Name:                 cleanName,
		CurrentWholesaleCost: currentWholesaleCost,
		CurrentStock:         currentStock,
		MeasurementUnit:      cleanUnit,
		IsActive:             true,
	}

	if err := s.repo.Create(ctx, supply); err != nil {
		if strings.Contains(err.Error(), "23503") || strings.Contains(err.Error(), "supplies_tenant_id_fkey") {
			return nil, ErrTenantNotFoundForSupply
		}
		return nil, err
	}

	return supply, nil
}

func (s *service) GetSupply(ctx context.Context, tenantID, id string) (*models.Supply, error) {
	supply, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if supply == nil {
		return nil, ErrSupplyNotFound
	}
	return supply, nil
}

func (s *service) ListSupplies(ctx context.Context, tenantID string) ([]models.Supply, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFoundForSupply
	}

	return s.repo.FetchAll(ctx, tenantID)
}

func (s *service) UpdateSupply(ctx context.Context, tenantID, id, name string, currentWholesaleCost, currentStock *float64, measurementUnit string, isActive *bool) (*models.Supply, error) {
	supply, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if supply == nil {
		return nil, ErrSupplyNotFound
	}

	if currentWholesaleCost != nil {
		if *currentWholesaleCost < 0 {
			return nil, ErrInvalidSupplyCost
		}
		supply.CurrentWholesaleCost = *currentWholesaleCost
	}
	if currentStock != nil {
		if *currentStock < 0 {
			return nil, ErrInvalidSupplyStock
		}
		supply.CurrentStock = *currentStock
	}
	if name != "" {
		cleanName := strings.TrimSpace(name)
		if cleanName == "" {
			return nil, ErrInvalidSupplyData
		}
		if cleanName != supply.Name {
			existing, err := s.repo.GetByName(ctx, tenantID, cleanName)
			if err != nil {
				return nil, err
			}
			if existing != nil && existing.ID != supply.ID {
				return nil, ErrDuplicateSupplyName
			}
			supply.Name = cleanName
		}
	}
	if measurementUnit != "" {
		cleanUnit := strings.ToLower(strings.TrimSpace(measurementUnit))
		if cleanUnit == "" {
			return nil, ErrInvalidSupplyData
		}
		supply.MeasurementUnit = cleanUnit
	}
	if isActive != nil {
		supply.IsActive = *isActive
	}

	if err := s.repo.Update(ctx, supply); err != nil {
		return nil, err
	}

	return supply, nil
}

func (s *service) DeleteSupply(ctx context.Context, tenantID, id string) error {
	supply, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if supply == nil {
		return ErrSupplyNotFound
	}

	return s.repo.Delete(ctx, tenantID, id)
}
