package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"leguiburger/internal/models"
)

// Mock del Servicio blindado contra llamadas no configuradas
type mockAuthService struct {
	loginFn func(ctx context.Context, tenantID, email, password string) (*LoginResponse, error)
}

func (m *mockAuthService) Login(ctx context.Context, tenantID, email, password string) (*LoginResponse, error) {
	if m.loginFn == nil {
		return nil, ErrInvalidCredentials
	}
	return m.loginFn(ctx, tenantID, email, password)
}

func TestHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		tenantHeader   string
		body           interface{}
		mockService    func(ctx context.Context, tenantID, email, password string) (*LoginResponse, error)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:         "HTTP 200 OK - Login Exitoso con Tenant",
			tenantHeader: "tenant-uuid-1",
			body: map[string]string{
				"email":    "admin@test.com",
				"password": "Password123!",
			},
			mockService: func(ctx context.Context, tenantID, email, password string) (*LoginResponse, error) {
				return &LoginResponse{
					Token: "mock.jwt.token",
					Employee: &models.Employee{
						ID:       "emp-1",
						TenantID: &tenantID,
						Email:    email,
					},
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedCode:   "",
		},
		{
			name:         "HTTP 200 OK - Login Exitoso OWNER (Sin X-Tenant-ID)",
			tenantHeader: "", // Header vacío permitido para Owner
			body: map[string]string{
				"email":    "owner@test.com",
				"password": "Password123!",
			},
			mockService: func(ctx context.Context, tenantID, email, password string) (*LoginResponse, error) {
				return &LoginResponse{
					Token: "mock.jwt.owner.token",
					Employee: &models.Employee{
						ID:       "owner-1",
						TenantID: nil,
						Email:    email,
					},
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedCode:   "",
		},
		{
			name:         "HTTP 400 Bad Request - Campos requeridos faltantes",
			tenantHeader: "tenant-uuid-1",
			body: map[string]string{
				"email":    "",
				"password": "Password123!",
			},
			mockService:    nil,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "MISSING_FIELDS",
		},
		{
			name:         "HTTP 401 Unauthorized - Credenciales inválidas",
			tenantHeader: "tenant-uuid-1",
			body: map[string]string{
				"email":    "admin@test.com",
				"password": "WrongPassword",
			},
			mockService: func(ctx context.Context, tenantID, email, password string) (*LoginResponse, error) {
				return nil, ErrInvalidCredentials
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := &mockAuthService{loginFn: tt.mockService}
			handler := NewHandler(svcMock)

			// Serializar el body
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			if tt.tenantHeader != "" {
				req.Header.Set("X-Tenant-ID", tt.tenantHeader)
			}

			w := httptest.NewRecorder()
			handler.HandleAuthRoutes(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("se esperaba status HTTP %d, obtenido %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedCode != "" {
				var response map[string]string
				json.Unmarshal(w.Body.Bytes(), &response)
				if response["code"] != tt.expectedCode {
					t.Errorf("se esperaba el código de error '%s', obtenido '%s'", tt.expectedCode, response["code"])
				}
			}
		})
	}
}
