package tenants

import (
	"bytes"
	"context"
	"encoding/json"
	"leguiburger/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock de la capa de servicio para no depender de la lógica real en el test del handler
type mockService struct {
	OnRegisterTenant func(ctx context.Context, name, subdomain string) (*models.Tenant, error)
	OnUpdateTenant   func(ctx context.Context, id string, name string, subdomain string, active *bool) (*models.Tenant, error)
	OnDeleteTenant   func(ctx context.Context, id string) error
}

func (m *mockService) RegisterTenant(ctx context.Context, name, subdomain string) (*models.Tenant, error) {
	return m.OnRegisterTenant(ctx, name, subdomain)
}

func (m *mockService) UpdateTenant(ctx context.Context, id string, name string, subdomain string, active *bool) (*models.Tenant, error) {
	if m.OnUpdateTenant != nil {
		return m.OnUpdateTenant(ctx, id, name, subdomain, active)
	}
	return nil, nil
}

func (m *mockService) DeleteTenant(ctx context.Context, id string) error {
	if m.OnDeleteTenant != nil {
		return m.OnDeleteTenant(ctx, id)
	}
	return nil
}

func TestCreateTenantHandler_Success(t *testing.T) {
	// 1. Configuramos el mock de servicio para simular un registro exitoso
	mockSvc := &mockService{
		OnRegisterTenant: func(ctx context.Context, name, subdomain string) (*models.Tenant, error) {
			return &models.Tenant{
				ID:        "uuid-de-prueba",
				Name:      name,
				Subdomain: subdomain,
				Active:    true,
			}, nil
		},
	}

	handler := NewHandler(mockSvc)

	// 2. Preparamos el payload JSON que le enviaríamos al endpoint
	payload := CreateTenantRequest{
		Name:      "Legui Burger Centro",
		Subdomain: "legui-centro",
	}
	body, _ := json.Marshal(payload)

	// 3. Creamos la petición HTTP de prueba (POST /api/tenants)
	req, err := http.NewRequest(http.MethodPost, "/api/tenants", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	// 4. Creamos un grabador de respuestas (ResponseRecorder) para capturar el resultado
	rr := httptest.NewRecorder()

	// 5. Ejecutamos el handler pasando nuestro request y el capturador
	handler.CreateTenant(rr, req)

	// 6. Validamos el código de estado HTTP esperado (201 Created)
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("Se esperaba código %v, pero se obtuvo %v", http.StatusCreated, status)
	}

	// 7. Validamos que la respuesta JSON contenga el ID y subdominio correctos
	var response models.Tenant
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("No se pudo parsear el JSON de respuesta: %v", err)
	}

	if response.ID != "uuid-de-prueba" || response.Subdomain != "legui-centro" {
		t.Errorf("Respuesta inesperada: %+v", response)
	}
}
