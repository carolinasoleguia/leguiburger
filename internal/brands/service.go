package brands

import (
	"context"
	"errors"
	"leguiburger/internal/models"
	"strings"
)

var (
	ErrBrandNotFound  = errors.New("marca no encontrada")
	ErrDuplicateBrand = errors.New("ya existe una marca con ese nombre")
)

type Service interface {
	Create(ctx context.Context, name, taxID string) (*models.Brand, error)
	List(ctx context.Context) ([]models.Brand, error)
	Get(ctx context.Context, id string) (*models.Brand, error)
	Update(ctx context.Context, id, name, taxID string) (*models.Brand, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) Create(ctx context.Context, name, taxID string) (*models.Brand, error) {

	name = strings.TrimSpace(name)

	existing, err := s.repo.GetByName(ctx, name)

	if err != nil {
		return nil, err
	}

	if existing != nil {
		return nil, ErrDuplicateBrand
	}

	brand := &models.Brand{
		Name:  name,
		TaxID: strings.TrimSpace(taxID),
	}

	err = s.repo.Create(ctx, brand)

	return brand, err
}

func (s *service) List(ctx context.Context) ([]models.Brand, error) {
	return s.repo.GetAll(ctx)
}

func (s *service) Get(ctx context.Context, id string) (*models.Brand, error) {

	brand, err := s.repo.GetByID(ctx, id)

	if err != nil {
		return nil, err
	}

	if brand == nil {
		return nil, ErrBrandNotFound
	}

	return brand, nil
}

func (s *service) Update(ctx context.Context, id, name, taxID string) (*models.Brand, error) {

	brand, err := s.Get(ctx, id)

	if err != nil {
		return nil, err
	}

	if name != "" {
		name = strings.TrimSpace(name)

		existing, _ := s.repo.GetByName(ctx, name)

		if existing != nil && existing.ID != brand.ID {
			return nil, ErrDuplicateBrand
		}

		brand.Name = name
	}

	if taxID != "" {
		brand.TaxID = strings.TrimSpace(taxID)
	}

	err = s.repo.Update(ctx, brand)

	return brand, err
}

func (s *service) Delete(ctx context.Context, id string) error {

	brand, err := s.Get(ctx, id)

	if err != nil {
		return err
	}

	if brand == nil {
		return ErrBrandNotFound
	}

	return s.repo.Delete(ctx, id)
}
