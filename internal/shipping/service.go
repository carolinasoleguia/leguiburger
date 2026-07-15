package shipping

import (
	"context"
	"errors"
	"leguiburger/internal/models"
	"strings"
)

var (
	ErrShippingNotFound          = errors.New("método de envío no encontrado")
	ErrDuplicateShipping         = errors.New("ya existe un método de envío con ese nombre para este comercio")
	ErrInvalidCost               = errors.New("el costo de envío no puede ser negativo")
	ErrTenantNotFoundForShipping = errors.New("el comercio (tenant) especificado no existe")
)

type Service interface {
	CreateMethod(ctx context.Context, tenantID, name, typification, description string, cost float64, estTime string) (*models.ShippingMethod, error)
	GetMethod(ctx context.Context, tenantID, id string) (*models.ShippingMethod, error)
	ListMethods(ctx context.Context, tenantID string) ([]models.ShippingMethod, error)
	UpdateMethod(ctx context.Context, tenantID, id string, name, typification, description string, cost *float64, estTime string, active *bool) (*models.ShippingMethod, error)
	DeleteMethod(ctx context.Context, tenantID, id string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateMethod(ctx context.Context, tenantID, name, typification, description string, cost float64, estTime string) (*models.ShippingMethod, error) {
	if cost < 0 {
		return nil, ErrInvalidCost
	}

	cleanName := strings.TrimSpace(name)
	cleanTyp := strings.ToUpper(strings.TrimSpace(typification)) // Ej: "DELIVERY"

	// 💡 Validamos con el nuevo método del repositorio
	existing, err := s.repo.GetByNameAndTypification(ctx, tenantID, cleanName, cleanTyp)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateShipping
	}

	sm := &models.ShippingMethod{
		TenantID:      tenantID,
		Name:          cleanName,
		Typification:  cleanTyp, // 👈 Se guarda normalizado
		Description:   strings.TrimSpace(description),
		Cost:          cost,
		EstimatedTime: strings.TrimSpace(estTime),
		IsActive:      true,
	}

	if err := s.repo.Create(ctx, sm); err != nil {
		if strings.Contains(err.Error(), "23503") || strings.Contains(err.Error(), "shipping_methods_tenant_id_fkey") {
			return nil, ErrTenantNotFoundForShipping
		}
		return nil, err
	}
	return sm, nil
}

func (s *service) UpdateMethod(ctx context.Context, tenantID, id string, name, typification, description string, cost *float64, estTime string, active *bool) (*models.ShippingMethod, error) {
	sm, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if sm == nil {
		return nil, ErrShippingNotFound
	}

	if cost != nil {
		if *cost < 0 {
			return nil, ErrInvalidCost
		}
		sm.Cost = *cost
	}

	// 💡 Preparamos los valores para validar la colisión cruzada
	tempName := sm.Name
	if name != "" {
		tempName = strings.TrimSpace(name)
	}

	tempTyp := sm.Typification
	if typification != "" {
		tempTyp = strings.ToUpper(strings.TrimSpace(typification))
	}

	// Si cambió el nombre o la tipificación, validamos que no colisione
	if (name != "" && tempName != sm.Name) || (typification != "" && tempTyp != sm.Typification) {
		existing, err := s.repo.GetByNameAndTypification(ctx, tenantID, tempName, tempTyp)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != sm.ID {
			return nil, ErrDuplicateShipping
		}
		sm.Name = tempName
		sm.Typification = tempTyp
	}

	if description != "" {
		sm.Description = strings.TrimSpace(description)
	}
	if estTime != "" {
		sm.EstimatedTime = strings.TrimSpace(estTime)
	}
	if active != nil {
		sm.IsActive = *active
	}

	if err := s.repo.Update(ctx, sm); err != nil {
		return nil, err
	}
	return sm, nil
}

func (s *service) GetMethod(ctx context.Context, tenantID, id string) (*models.ShippingMethod, error) {
	sm, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if sm == nil {
		return nil, ErrShippingNotFound
	}
	return sm, nil
}

func (s *service) ListMethods(ctx context.Context, tenantID string) ([]models.ShippingMethod, error) {
	return s.repo.FetchAll(ctx, tenantID)
}

func (s *service) DeleteMethod(ctx context.Context, tenantID, id string) error {
	sm, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if sm == nil {
		return ErrShippingNotFound
	}

	return s.repo.Delete(ctx, tenantID, id)
}
