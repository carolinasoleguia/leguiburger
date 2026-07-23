package tenants

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type CreateTenantRequest struct {
	Name      string `json:"name"`
	Subdomain string `json:"subdomain"`
	TaxID     string `json:"tax_id"`
}

type UpdateTenantRequest struct {
	Name      *string `json:"name" validate:"omitempty,gt=0"`
	Subdomain *string `json:"subdomain" validate:"omitempty,gt=0"`
	TaxID     *string `json:"tax_id" validate:"omitempty,gt=0"`
	Active    *bool   `json:"active"`
}

var validate = validator.New()

// HandleTenantRoutes rutea las peticiones según el método y la URL
func (h *Handler) HandleTenantRoutes(w http.ResponseWriter, r *http.Request) {
	// Limpiamos barras extras y dividimos el path en partes
	trimmedPath := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(trimmedPath, "/")

	// CASO A: /api/tenants (Listar - GET / Creación - POST)
	if len(pathParts) == 2 && pathParts[0] == "api" && pathParts[1] == "tenants" {
		switch r.Method {
		case http.MethodGet:
			h.GetTenants(w, r) // <--- Agregamos este método para listar
			return
		case http.MethodPost:
			h.CreateTenant(w, r)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido", nil)
			return
		}
	}

	// CASO B: /api/tenants/{id} (Editar - PUT, Eliminar - DELETE)
	if len(pathParts) == 3 && pathParts[0] == "api" && pathParts[1] == "tenants" {
		id := pathParts[2]

		if id == "" {
			h.respondWithError(w, http.StatusBadRequest, "INVALID_ID", "ID de comercio inválido", nil)
			return
		}

		switch r.Method {
		case http.MethodPut:
			h.UpdateTenant(w, r, id)
			return
		case http.MethodDelete:
			h.DeleteTenant(w, r, id)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido", nil)
			return
		}
	}

	// Si no entró en ninguna regla
	h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", "Recurso no encontrado", nil)
}

func (h *Handler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "El cuerpo de la petición no es un JSON válido", nil)
		return
	}

	if req.Name == "" || req.Subdomain == "" {
		details := map[string]string{}
		if req.Name == "" {
			details["name"] = "El nombre del comercio es requerido"
		}
		if req.Subdomain == "" {
			details["subdomain"] = "El subdominio es requerido"
		}
		if req.TaxID == "" {
			details["tax_id"] = "El tax_id es requerido"
		}
		h.respondWithError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Faltan campos obligatorios", details)
		return
	}

	tenant, err := h.service.RegisterTenant(r.Context(), req.Name, req.Subdomain, req.TaxID)
	if err != nil {

		if err == ErrDuplicateBranch {
			h.respondWithError(w, http.StatusBadRequest, "DUPLICATED_BRANCH", err.Error(), nil)
			return
		}
		h.respondWithError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Error inesperado", nil)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, tenant)
}

func (h *Handler) GetTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.service.GetAllTenants(r.Context())
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Error al obtener los comercios", nil)
		return
	}
	h.respondWithJSON(w, http.StatusOK, tenants)
}

func (h *Handler) UpdateTenant(w http.ResponseWriter, r *http.Request, id string) {
	var req UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "El cuerpo de la petición no es un JSON válido", nil)
		return
	}

	if err := validate.Struct(req); err != nil {
		// Usamos tu helper h.respondWithError para que el cliente de Vue reciba siempre la misma estructura de error
		h.respondWithError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Error de validación: los campos enviados (name, subdomain o tax_id) no pueden estar vacíos", nil)
		return
	}

	// 1. 👇 Desreferenciamos los punteros de manera segura 👇
	var nameVal, subdomainVal, taxIDVal string
	if req.Name != nil {
		nameVal = *req.Name
	}
	if req.Subdomain != nil {
		subdomainVal = *req.Subdomain
	}
	if req.TaxID != nil {
		taxIDVal = *req.TaxID
	}

	// 2. Pasamos los strings limpios y desempaquetados al servicio
	tenant, err := h.service.UpdateTenant(r.Context(), id, nameVal, subdomainVal, taxIDVal, req.Active)
	if err != nil {
		if err == ErrTenantNotFound {
			h.respondWithError(w, http.StatusNotFound, "TENANT_NOT_FOUND", err.Error(), nil)
			return
		}
		if err == ErrDuplicateBranch {
			h.respondWithError(w, http.StatusBadRequest, "DUPLICATED_BRANCH", err.Error(), nil)
			return
		}
		h.respondWithError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Error inesperado", nil)
		return
	}

	h.respondWithJSON(w, http.StatusOK, tenant)
}

func (h *Handler) DeleteTenant(w http.ResponseWriter, r *http.Request, id string) {
	err := h.service.DeleteTenant(r.Context(), id)
	if err != nil {
		if err == ErrTenantNotFound {
			h.respondWithError(w, http.StatusNotFound, "TENANT_NOT_FOUND", err.Error(), nil)
			return
		}
		h.respondWithError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Error inesperado", nil)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Comercio desactivado con éxito",
	})
}

// --- HELPERS DE RESPUESTA ---

func (h *Handler) respondWithError(w http.ResponseWriter, status int, code string, message string, details interface{}) {
	h.respondWithJSON(w, status, ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	})
}

func (h *Handler) respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
