package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
			name:         "HTTP 200 OK - login exitoso con tenant",
			tenantHeader: "tenant-uuid-1",
			body: map[string]string{
				"email":    "admin@test.com",
				"password": "Password123!",
			},
			mockService: func(ctx context.Context, tenantID, email, password string) (*LoginResponse, error) {
				return &LoginResponse{
					Token: "mock.jwt.token",
					Employee: EmployeeDTO{
						ID:       "emp-1",
						TenantID: &tenantID,
						Email:    email,
						Role:     "admin",
						IsActive: true,
					},
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedCode:   "",
		},
		{
			name:         "HTTP 200 OK - login exitoso owner sin tenant",
			tenantHeader: "",
			body: map[string]string{
				"email":    "owner@test.com",
				"password": "Password123!",
			},
			mockService: func(ctx context.Context, tenantID, email, password string) (*LoginResponse, error) {
				return &LoginResponse{
					Token: "mock.jwt.owner.token",
					Employee: EmployeeDTO{
						ID:       "owner-1",
						Email:    email,
						Role:     RoleOwner,
						IsActive: true,
					},
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedCode:   "",
		},
		{
			name:         "HTTP 400 Bad Request - campos requeridos faltantes",
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
			name:           "HTTP 400 Bad Request - JSON invalido",
			tenantHeader:   "tenant-uuid-1",
			body:           "{",
			mockService:    nil,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_INPUT",
		},
		{
			name:         "HTTP 401 Unauthorized - credenciales invalidas",
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
		{
			name:         "HTTP 403 Forbidden - tenant requerido",
			tenantHeader: "",
			body: map[string]string{
				"email":    "admin@test.com",
				"password": "Password123!",
			},
			mockService: func(ctx context.Context, tenantID, email, password string) (*LoginResponse, error) {
				return nil, ErrTenantRequired
			},
			expectedStatus: http.StatusForbidden,
			expectedCode:   "TENANT_REQUIRED",
		},
		{
			name:         "HTTP 403 Forbidden - tenant no autorizado",
			tenantHeader: "tenant-uuid-2",
			body: map[string]string{
				"email":    "admin@test.com",
				"password": "Password123!",
			},
			mockService: func(ctx context.Context, tenantID, email, password string) (*LoginResponse, error) {
				return nil, ErrForbiddenTenant
			},
			expectedStatus: http.StatusForbidden,
			expectedCode:   "FORBIDDEN",
		},
		{
			name:         "HTTP 403 Forbidden - tenant invalido",
			tenantHeader: "tenant-inactivo",
			body: map[string]string{
				"email":    "admin@test.com",
				"password": "Password123!",
			},
			mockService: func(ctx context.Context, tenantID, email, password string) (*LoginResponse, error) {
				return nil, ErrTenantNotFoundForAuth
			},
			expectedStatus: http.StatusForbidden,
			expectedCode:   "INVALID_TENANT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := &mockAuthService{loginFn: tt.mockService}
			handler := NewHandler(svcMock)

			var bodyBytes []byte
			if raw, ok := tt.body.(string); ok {
				bodyBytes = []byte(raw)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			if tt.tenantHeader != "" {
				req.Header.Set(TenantHeaderName, tt.tenantHeader)
			}

			w := httptest.NewRecorder()
			handler.HandleAuthRoutes(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("se esperaba status HTTP %d, obtenido %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedCode != "" {
				var response map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("respuesta JSON invalida: %v", err)
				}
				if response["code"] != tt.expectedCode {
					t.Errorf("se esperaba codigo de error %q, obtenido %q", tt.expectedCode, response["code"])
				}
				return
			}

			var response LoginResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("respuesta JSON invalida: %v", err)
			}
			if response.Token == "" || response.Employee.ID == "" {
				t.Fatal("se esperaba token y DTO publico de empleado")
			}
		})
	}
}
