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

	claims, ok := auth.GetClaimsFromContext(r.Context())
	if !ok {
		h.respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "No autorizado")
		return
	}

	tenantID := strings.TrimSpace(input.TenantID)
	if tenantID == "" {
		tenantID = strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	}

	normalizedRole := strings.ToLower(strings.TrimSpace(input.Role))
	isGlobalUser := normalizedRole == "owner" || normalizedRole == "super_admin"

	if tenantID == "" && claims.Role != "owner" {
		tenantID = strings.TrimSpace(claims.TenantID)
	}

	if tenantID == "" && !isGlobalUser {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	if claims.Role != "owner" && tenantID != strings.TrimSpace(claims.TenantID) {
		h.respondWithError(w, http.StatusForbidden, "FORBIDDEN", "No puedes crear empleados fuera de tu comercio")
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
	claims, ok := auth.GetClaimsFromContext(r.Context())
	if !ok {
		h.respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "No autorizado")
		return
	}

	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		tenantID = strings.TrimSpace(claims.TenantID)
	}

	var employees interface{}
	var err error

	if claims.Role == "owner" && tenantID == "" {
		employees, err = h.service.GetAllEmployees(r.Context())
	} else {
		employees, err = h.service.ListEmployees(r.Context(), tenantID)
	}

	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, employees)
}

func (h *Handler) GetEmployee(w http.ResponseWriter, r *http.Request, id string) {
	claims, ok := auth.GetClaimsFromContext(r.Context())
	if !ok {
		h.respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "No autorizado")
		return
	}

	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		tenantID = strings.TrimSpace(claims.TenantID)
	}

	if claims.Role != "owner" && tenantID != strings.TrimSpace(claims.TenantID) {
		h.respondWithError(w, http.StatusForbidden, "FORBIDDEN", "No puedes ver empleados de otro comercio")
		return
	}

	employee, err := h.service.GetEmployee(r.Context(), tenantID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, employee)
}

func (h *Handler) UpdateEmployee(w http.ResponseWriter, r *http.Request, id string) {
	claims, ok := auth.GetClaimsFromContext(r.Context())
	if !ok {
		h.respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "No autorizado")
		return
	}

	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		tenantID = strings.TrimSpace(claims.TenantID)
	}

	if claims.Role != "owner" && tenantID != strings.TrimSpace(claims.TenantID) {
		h.respondWithError(w, http.StatusForbidden, "FORBIDDEN", "No puedes editar empleados de otro comercio")
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
	claims, ok := auth.GetClaimsFromContext(r.Context())
	if !ok {
		h.respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "No autorizado")
		return
	}

	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		tenantID = strings.TrimSpace(claims.TenantID)
	}

	if claims.Role != "owner" && tenantID != strings.TrimSpace(claims.TenantID) {
		h.respondWithError(w, http.StatusForbidden, "FORBIDDEN", "No puedes borrar empleados de otro comercio")
		return
	}

	if err := h.service.DeleteEmployee(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Empleado desactivado con éxito"})
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
