package tenants

import (
	"context"
	"errors"
	"testing"

	"leguiburger/internal/models"
)

// ---------------- MOCK TENANT REPOSITORY ----------------

type mockRepository struct {
	createFunc                 func(ctx context.Context, tenant *models.Tenant) error
	getByIDFunc                func(ctx context.Context, id string) (*models.Tenant, error)
	getByBrandAndSubdomainFunc func(ctx context.Context, brandID, subdomain string) (*models.Tenant, error)
	updateFunc                 func(ctx context.Context, tenant *models.Tenant) error
	deleteFunc                 func(ctx context.Context, id string) error
	getAllFunc                 func(ctx context.Context) ([]models.Tenant, error)
}

func (m *mockRepository) Create(
	ctx context.Context,
	tenant *models.Tenant,
) error {

	if m.createFunc != nil {
		return m.createFunc(ctx, tenant)
	}

	return nil
}

func (m *mockRepository) GetByID(
	ctx context.Context,
	id string,
) (*models.Tenant, error) {

	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}

	return nil, nil
}

func (m *mockRepository) GetByBrandAndSubdomain(
	ctx context.Context,
	brandID string,
	subdomain string,
) (*models.Tenant, error) {

	if m.getByBrandAndSubdomainFunc != nil {
		return m.getByBrandAndSubdomainFunc(
			ctx,
			brandID,
			subdomain,
		)
	}

	return nil, nil
}

func (m *mockRepository) Update(
	ctx context.Context,
	tenant *models.Tenant,
) error {

	if m.updateFunc != nil {
		return m.updateFunc(ctx, tenant)
	}

	return nil
}

func (m *mockRepository) Delete(
	ctx context.Context,
	id string,
) error {

	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}

	return nil
}

func (m *mockRepository) GetAll(
	ctx context.Context,
) ([]models.Tenant, error) {

	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}

	return nil, nil
}

// ---------------- MOCK BRAND REPOSITORY ----------------

type mockBrandRepository struct {
	getByIDFunc func(
		ctx context.Context,
		id string,
	) (*models.Brand, error)
}

func (m *mockBrandRepository) GetByID(
	ctx context.Context,
	id string,
) (*models.Brand, error) {

	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}

	return &models.Brand{
		ID: id,
	}, nil
}

func (m *mockBrandRepository) Create(
	ctx context.Context,
	brand *models.Brand,
) error {
	return nil
}

func (m *mockBrandRepository) GetByName(
	ctx context.Context,
	name string,
) (*models.Brand, error) {
	return nil, nil
}

func (m *mockBrandRepository) Update(
	ctx context.Context,
	brand *models.Brand,
) error {
	return nil
}

func (m *mockBrandRepository) Delete(
	ctx context.Context,
	id string,
) error {
	return nil
}

func (m *mockBrandRepository) GetAll(
	ctx context.Context,
) ([]models.Brand, error) {
	return nil, nil
}

// ---------------- TESTS ----------------

func TestRegisterTenant_RepoError(t *testing.T) {

	mockRepo := &mockRepository{
		createFunc: func(
			ctx context.Context,
			tenant *models.Tenant,
		) error {

			return errors.New("error conexion DB")
		},
	}

	service := NewService(
		mockRepo,
		&mockBrandRepository{},
	)

	_, err := service.RegisterTenant(
		context.Background(),
		"brand-id",
		"laplata",
	)

	if err == nil {
		t.Error("se esperaba error del repositorio")
	}
}

func TestRegisterTenant_DuplicateBranch(t *testing.T) {

	mockRepo := &mockRepository{

		getByBrandAndSubdomainFunc: func(
			ctx context.Context,
			brandID string,
			subdomain string,
		) (*models.Tenant, error) {

			return &models.Tenant{
				ID:        "tenant-existente",
				Subdomain: "laplata",
			}, nil
		},
	}

	service := NewService(
		mockRepo,
		&mockBrandRepository{},
	)

	_, err := service.RegisterTenant(
		context.Background(),
		"brand-id",
		"laplata",
	)

	if !errors.Is(err, ErrDuplicateBranch) {

		t.Errorf(
			"se esperaba ErrDuplicateBranch, se obtuvo %v",
			err,
		)
	}
}

func TestRegisterTenant_NormalizesSubdomain(t *testing.T) {

	var savedSubdomain string

	mockRepo := &mockRepository{

		createFunc: func(
			ctx context.Context,
			tenant *models.Tenant,
		) error {

			savedSubdomain = tenant.Subdomain

			return nil
		},
	}

	service := NewService(
		mockRepo,
		&mockBrandRepository{},
	)

	_, err := service.RegisterTenant(
		context.Background(),
		"brand-id",
		"  LeGui-CeNtRo ",
	)

	if err != nil {

		t.Fatalf(
			"no se esperaba error: %v",
			err,
		)
	}

	if savedSubdomain != "legui-centro" {

		t.Errorf(
			"se esperaba legui-centro pero fue %s",
			savedSubdomain,
		)
	}
}

func TestUpdateTenant_Success(t *testing.T) {

	mockRepo := &mockRepository{

		getByIDFunc: func(
			ctx context.Context,
			id string,
		) (*models.Tenant, error) {

			return &models.Tenant{
				ID:        "tenant-id",
				Subdomain: "viejo",
				Active:    true,
			}, nil
		},

		updateFunc: func(
			ctx context.Context,
			tenant *models.Tenant,
		) error {

			return nil
		},
	}

	service := NewService(
		mockRepo,
		&mockBrandRepository{},
	)

	active := false

	tenant, err := service.UpdateTenant(
		context.Background(),
		"tenant-id",
		"nuevo",
		&active,
	)

	if err != nil {

		t.Fatalf(
			"no se esperaba error: %v",
			err,
		)
	}

	if tenant.Subdomain != "nuevo" ||
		tenant.Active != false {

		t.Errorf(
			"tenant incorrecto %+v",
			tenant,
		)
	}
}

func TestUpdateTenant_NotFound(t *testing.T) {

	mockRepo := &mockRepository{

		getByIDFunc: func(
			ctx context.Context,
			id string,
		) (*models.Tenant, error) {

			return nil, nil
		},
	}

	service := NewService(
		mockRepo,
		&mockBrandRepository{},
	)

	_, err := service.UpdateTenant(
		context.Background(),
		"fake",
		"sub",
		nil,
	)

	if !errors.Is(err, ErrTenantNotFound) {

		t.Errorf(
			"se esperaba ErrTenantNotFound: %v",
			err,
		)
	}
}

func TestDeleteTenant_Success(t *testing.T) {

	deleted := false

	mockRepo := &mockRepository{

		getByIDFunc: func(
			ctx context.Context,
			id string,
		) (*models.Tenant, error) {

			return &models.Tenant{
				ID: id,
			}, nil
		},

		deleteFunc: func(
			ctx context.Context,
			id string,
		) error {

			deleted = true

			return nil
		},
	}

	service := NewService(
		mockRepo,
		&mockBrandRepository{},
	)

	err := service.DeleteTenant(
		context.Background(),
		"tenant-id",
	)

	if err != nil {

		t.Fatalf(
			"no se esperaba error: %v",
			err,
		)
	}

	if !deleted {
		t.Error(
			"no se llamo Delete",
		)
	}
}
