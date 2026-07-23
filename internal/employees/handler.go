package employees

import (
	"encoding/json"
	"errors"
	"leguiburger/internal/auth"
	"net/http"
	"strings"
)

type Handler struct {
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

type CreateInput struct {
	TenantID  string `json:"tenant_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Phone     string `json:"phone"`
	Role      string `json:"role"`
}

type UpdateInput struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Phone     string `json:"phone"`
	Role      string `json:"role"`
	IsActive  *bool  `json:"is_active"`
}

type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (h *Handler) HandleEmployeeRoutes(w http.ResponseWriter, r *http.Request) {
	trimmedPath := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(trimmedPath, "/")

	if len(pathParts) == 2 && pathParts[0] == "api" && pathParts[1] == "employees" {
		switch r.Method {
		case http.MethodPost:
			h.CreateEmployee(w, r)
			return
		case http.MethodGet:
			h.ListEmployees(w, r)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido")
			return
		}
	}

	if len(pathParts) == 3 && pathParts[0] == "api" && pathParts[1] == "employees" {
		id := pathParts[2]
		if id == "" {
			h.respondWithError(w, http.StatusBadRequest, "INVALID_ID", "ID de empleado inválido")
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.GetEmployee(w, r, id)
			return
		case http.MethodPut:
			h.UpdateEmployee(w, r, id)
			return
		case http.MethodDelete:
			h.DeleteEmployee(w, r, id)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido")
			return
		}
	}

	h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", "Recurso no encontrado")
}

func (h *Handler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_INPUT", "JSON inválido")
		return
	}

	tenantID := strings.TrimSpace(input.TenantID)
	if tenantID == "" {
		tenantID = strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	}

	normalizedRole := strings.ToLower(strings.TrimSpace(input.Role))
	isGlobalUser := normalizedRole == "owner" || normalizedRole == "super_admin"

	if tenantID == "" && !isGlobalUser {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	employee, err := h.service.CreateEmployee(
		r.Context(),
		tenantID,
		input.FirstName,
		input.LastName,
		input.Email,
		input.Password,
		input.Phone,
		input.Role,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, employee)
}

func (h *Handler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")

	var employees interface{}
	var err error

	if tenantID == "" {
		// Si no hay tenant_id (es el Owner global), llamamos a un método que liste todo
		employees, err = h.service.GetAllEmployees(r.Context())
	} else {
		// Si viene el header, filtramos por ese tenant específico
		employees, err = h.service.ListEmployees(r.Context(), tenantID)
	}

	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, employees)
}

func (h *Handler) GetEmployee(w http.ResponseWriter, r *http.Request, id string) {
	tenantID := r.Header.Get("X-Tenant-ID")

	if tenantID == "" {
		// Verificamos si el usuario es owner a través de los claims del token
		claims, ok := auth.GetClaimsFromContext(r.Context())
		if !ok || claims.Role != "owner" {
			h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
			return
		}
	}

	// Si es owner y tenantID está vacío, tu service deberá manejarlo
	// (o podés pasarle tenantID vacío según cómo tengas armado el service)
	employee, err := h.service.GetEmployee(r.Context(), tenantID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, employee)
}

func (h *Handler) UpdateEmployee(w http.ResponseWriter, r *http.Request, id string) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_INPUT", "JSON inválido")
		return
	}

	employee, err := h.service.UpdateEmployee(r.Context(), tenantID, id, input.FirstName, input.LastName, input.Email, input.Password, input.Phone, input.Role, input.IsActive)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, employee)
}

func (h *Handler) DeleteEmployee(w http.ResponseWriter, r *http.Request, id string) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	if err := h.service.DeleteEmployee(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrDuplicateEmployeeEmail) {
		h.respondWithError(w, http.StatusConflict, "DUPLICATE_EMPLOYEE_EMAIL", err.Error())
	} else if errors.Is(err, ErrEmployeeNotFound) {
		h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	} else if errors.Is(err, ErrInvalidEmployeeData) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_EMPLOYEE_DATA", err.Error())
	} else if errors.Is(err, ErrInvalidEmployeeRole) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_EMPLOYEE_ROLE", err.Error())
	} else if errors.Is(err, ErrTenantNotFoundForEmployee) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_TENANT", err.Error())
	} else if errors.Is(err, ErrUnauthorizedAction) {
		h.respondWithError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	} else {
		h.respondWithError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Error inesperado")
	}
}

func (h *Handler) respondWithError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"code": code, "message": msg})
}

func (h *Handler) respondWithJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
