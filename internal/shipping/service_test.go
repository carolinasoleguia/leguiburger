package shipping

import (
	"context"
	"errors"
	"testing"

	"leguiburger/internal/models"
)

func TestCreateMethod_Success(t *testing.T) {
	repo := &MockRepository{
		OnGetByNameAndTypification: func(ctx context.Context, tenantID, name, typification string) (*models.ShippingMethod, error) {
			return nil, nil // No existe duplicado
		},
		OnCreate: func(ctx context.Context, sm *models.ShippingMethod) error {
			sm.ID = "generated-uuid-123"
			return nil
		},
	}

	service := NewService(repo)
	ctx := context.Background()

	res, err := service.CreateMethod(ctx, "tenant-1", "Moto Express", "delivery", "Envío rápido", 150.0, "30m")
	if err != nil {
		t.Fatalf("se esperaba éxito, se obtuvo error: %v", err)
	}

	if res.Name != "Moto Express" {
		t.Errorf("se esperaba Name 'Moto Express', se obtuvo: %s", res.Name)
	}
	if res.Typification != "DELIVERY" { // Debe estar normalizado a mayúsculas
		t.Errorf("se esperaba Typification 'DELIVERY', se obtuvo: %s", res.Typification)
	}
}

func TestCreateMethod_InvalidCost(t *testing.T) {
	repo := &MockRepository{}
	service := NewService(repo)

	_, err := service.CreateMethod(context.Background(), "tenant-1", "Test", "DELIVERY", "Desc", -50.0, "10m")
	if !errors.Is(err, ErrInvalidCost) {
		t.Errorf("se esperaba ErrInvalidCost, se obtuvo: %v", err)
	}
}

func TestCreateMethod_Duplicate(t *testing.T) {
	repo := &MockRepository{
		OnGetByNameAndTypification: func(ctx context.Context, tenantID, name, typification string) (*models.ShippingMethod, error) {
			// Simulamos que ya existe un método con el mismo Name + Typification
			return &models.ShippingMethod{ID: "existente", Name: name, Typification: typification}, nil
		},
	}

	service := NewService(repo)
	_, err := service.CreateMethod(context.Background(), "tenant-1", "Moto Express", "DELIVERY", "Desc", 150.0, "30m")

	if !errors.Is(err, ErrDuplicateShipping) {
		t.Errorf("se esperaba ErrDuplicateShipping, se obtuvo: %v", err)
	}
}

func TestCreateMethod_TenantNotFound(t *testing.T) {
	repo := &MockRepository{
		OnGetByNameAndTypification: func(ctx context.Context, tenantID, name, typification string) (*models.ShippingMethod, error) {
			return nil, nil
		},
		OnCreate: func(ctx context.Context, sm *models.ShippingMethod) error {
			// Simulamos violación de clave foránea de PostgreSQL (error 23503)
			return errors.New("ERROR: insert violates foreign key constraint \"shipping_methods_tenant_id_fkey\" (SQLSTATE 23503)")
		},
	}

	service := NewService(repo)
	_, err := service.CreateMethod(context.Background(), "fake-tenant", "Moto Express", "DELIVERY", "Desc", 150.0, "30m")

	if !errors.Is(err, ErrTenantNotFoundForShipping) {
		t.Errorf("se esperaba ErrTenantNotFoundForShipping, se obtuvo: %v", err)
	}
}
