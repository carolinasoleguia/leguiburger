package employees

import (
	"context"
	"errors"
	"testing"

	"leguiburger/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func stringPtr(s string) *string {
	return &s
}

// ---------------- MOCK TENANT REPOSITORY ----------------

type mockTenantRepository struct {
	getByIDFunc func(
		ctx context.Context,
		id string,
	) (*models.Tenant, error)

	getAllFunc func(
		ctx context.Context,
	) ([]models.Tenant, error)
}

func (m *mockTenantRepository) GetByID(
	ctx context.Context,
	id string,
) (*models.Tenant, error) {

	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}

	return &models.Tenant{
		ID:      id,
		BrandID: "brand-id",
	}, nil
}

func (m *mockTenantRepository) GetAll(
	ctx context.Context,
) ([]models.Tenant, error) {

	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}

	return nil, nil
}

func (m *mockTenantRepository) Create(
	ctx context.Context,
	tenant *models.Tenant,
) error {
	return nil
}

func (m *mockTenantRepository) GetByBrandAndSubdomain(
	ctx context.Context,
	brandID string,
	subdomain string,
) (*models.Tenant, error) {

	return nil, nil
}

func (m *mockTenantRepository) Update(
	ctx context.Context,
	tenant *models.Tenant,
) error {

	return nil
}

func (m *mockTenantRepository) Delete(
	ctx context.Context,
	id string,
) error {

	return nil
}

// ---------------- TESTS ----------------

func TestCreateEmployee_Success(t *testing.T) {

	repo := &mockRepository{

		getByEmailFunc: func(
			ctx context.Context,
			tenantID,
			email string,
		) (*models.Employee, error) {

			if email != "admin@email.com" {
				t.Errorf(
					"se esperaba email normalizado, se obtuvo %s",
					email,
				)
			}

			return nil, nil
		},

		createFunc: func(
			ctx context.Context,
			employee *models.Employee,
		) error {

			employee.ID = "generated-id"

			return nil
		},
	}

	tenantRepo := &mockTenantRepository{

		getByIDFunc: func(
			ctx context.Context,
			id string,
		) (*models.Tenant, error) {

			return &models.Tenant{
				ID:      id,
				BrandID: "brand-id",
			}, nil
		},
	}

	service := NewService(
		repo,
		tenantRepo,
	)

	rawPassword := "PasswordSegura123!"

	res, err := service.CreateEmployee(
		context.Background(),
		"tenant-1",
		" Ana ",
		" Eguia ",
		" ADMIN@EMAIL.COM ",
		rawPassword,
		" 2215555555 ",
		"ADMIN",
	)

	if err != nil {
		t.Fatalf(
			"se esperaba exito: %v",
			err,
		)
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(res.PasswordHash),
		[]byte(rawPassword),
	)

	if err != nil {
		t.Errorf(
			"password incorrecto: %v",
			err,
		)
	}

	if res.TenantID == nil ||
		*res.TenantID != "tenant-1" {

		t.Errorf(
			"TenantID incorrecto: %v",
			res.TenantID,
		)
	}

	if res.FirstName != "Ana" ||
		res.LastName != "Eguia" ||
		res.Email != "admin@email.com" ||
		res.Phone != "2215555555" ||
		res.Role != "admin" ||
		!res.IsActive {

		t.Errorf(
			"datos incorrectos: %+v",
			res,
		)
	}

}

func TestCreateEmployee_Owner_Success(t *testing.T) {

	repo := &mockRepository{

		getByEmailFunc: func(
			ctx context.Context,
			tenantID, email string,
		) (*models.Employee, error) {

			return nil, nil
		},

		createFunc: func(
			ctx context.Context,
			employee *models.Employee,
		) error {

			employee.ID = "owner-id"

			return nil
		},
	}

	service := NewService(
		repo,
		&mockTenantRepository{},
	)

	res, err := service.CreateEmployee(
		context.Background(),
		"",
		"Carolina",
		"Eguia",
		"owner@admin.com",
		"Password123!",
		"2214347305",
		"owner",
	)

	if err != nil {
		t.Fatalf(
			"error inesperado %v",
			err,
		)
	}

	if res.TenantID != nil {
		t.Error(
			"owner no debe tener tenant",
		)
	}

	if res.Role != "owner" {
		t.Errorf(
			"rol incorrecto %s",
			res.Role,
		)
	}

}

// Los siguientes tests no requieren cambios,
// solamente reemplaza tu mockTenantRepository viejo
// por el nuevo de arriba.

func TestCreateEmployee_NormalUser_RequiresTenantID(t *testing.T) {

	service := NewService(
		&mockRepository{},
		&mockTenantRepository{},
	)

	_, err := service.CreateEmployee(
		context.Background(),
		"",
		"Ana",
		"Eguia",
		"ana@email.com",
		"PasswordSegura123!",
		"",
		"employee",
	)

	if !errors.Is(err, ErrTenantNotFoundForEmployee) {

		t.Errorf(
			"error esperado %v obtenido %v",
			ErrTenantNotFoundForEmployee,
			err,
		)
	}

}

func TestCreateEmployee_InvalidData(t *testing.T) {

	service := NewService(
		&mockRepository{},
		&mockTenantRepository{},
	)

	_, err := service.CreateEmployee(
		context.Background(),
		"tenant-1",
		"",
		"Eguia",
		"ana@email.com",
		"Password123!",
		"",
		"employee",
	)

	if !errors.Is(err, ErrInvalidEmployeeData) {
		t.Errorf(
			"error incorrecto %v",
			err,
		)
	}

}
