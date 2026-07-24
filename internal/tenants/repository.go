package tenants

import (
	"context"
	"errors"

	"leguiburger/internal/db"
	"leguiburger/internal/models"

	"gorm.io/gorm"
)

type Repository interface {
	Create(
		ctx context.Context,
		tenant *models.Tenant,
	) error

	GetByID(
		ctx context.Context,
		id string,
	) (*models.Tenant, error)

	GetByBrandAndSubdomain(
		ctx context.Context,
		brandID string,
		subdomain string,
	) (*models.Tenant, error)

	Update(
		ctx context.Context,
		tenant *models.Tenant,
	) error

	Delete(
		ctx context.Context,
		id string,
	) error

	GetAll(
		ctx context.Context,
	) ([]models.Tenant, error)
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

// CREATE

func (r *repository) Create(
	ctx context.Context,
	tenant *models.Tenant,
) error {

	return db.DB.
		WithContext(ctx).
		Create(tenant).
		Error
}

// LIST

func (r *repository) GetAll(
	ctx context.Context,
) ([]models.Tenant, error) {

	var tenants []models.Tenant

	err := db.DB.
		WithContext(ctx).
		Preload("Brand").
		Order("created_at DESC").
		Find(&tenants).
		Error

	if err != nil {
		return nil, err
	}

	return tenants, nil
}

// GET BY ID

func (r *repository) GetByID(
	ctx context.Context,
	id string,
) (*models.Tenant, error) {

	var tenant models.Tenant

	err := db.DB.
		WithContext(ctx).
		Preload("Brand").
		First(
			&tenant,
			"id = ?",
			id,
		).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &tenant, nil
}

// GET BY BRAND + SUBDOMAIN

func (r *repository) GetByBrandAndSubdomain(
	ctx context.Context,
	brandID string,
	subdomain string,
) (*models.Tenant, error) {

	var tenant models.Tenant

	err := db.DB.
		WithContext(ctx).
		Where(
			"brand_id = ? AND subdomain = ?",
			brandID,
			subdomain,
		).
		First(&tenant).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &tenant, nil
}

// UPDATE

func (r *repository) Update(
	ctx context.Context,
	tenant *models.Tenant,
) error {

	return db.DB.
		WithContext(ctx).
		Save(tenant).
		Error
}

// DELETE LOGICO

func (r *repository) Delete(
	ctx context.Context,
	id string,
) error {

	return db.DB.
		WithContext(ctx).
		Model(&models.Tenant{}).
		Where(
			"id = ?",
			id,
		).
		Update(
			"active",
			false,
		).
		Error
}
