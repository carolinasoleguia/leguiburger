package tenants

import (
	"encoding/json"
	"net/http"
	"strings"
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
}

type UpdateTenantRequest struct {
	Name      string `json:"name"`
	Subdomain string `json:"subdomain"`
	Active    *bool  `json:"active"`
}

// HandleTenantRoutes rutea las peticiones según el método y la URL
func (h *Handler) HandleTenantRoutes(w http.ResponseWriter, r *http.Request) {
	// Limpiamos barras extras y dividimos el path en partes
	trimmedPath := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(trimmedPath, "/")

	// CASO A: /api/tenants (Creación - POST)
	if len(pathParts) == 2 && pathParts[0] == "api" && pathParts[1] == "tenants" {
		if r.Method == http.MethodPost {
			h.CreateTenant(w, r)
			return
		}
		h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido", nil)
		return
	}

	// CASO B: /api/tenants/{id} (Editar - PUT, Eliminar - DELETE)
	if len(pathParts) == 3 && pathParts[0] == "api" && pathParts[1] == "tenants" {
		id := pathParts[2]

		// Pequeña validación: si el ID viene vacío
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
		h.respondWithError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Faltan campos obligatorios", details)
		return
	}

	tenant, err := h.service.RegisterTenant(r.Context(), req.Name, req.Subdomain)
	if err != nil {
		if err == ErrSubdomainAlreadyExists {
			h.respondWithError(w, http.StatusBadRequest, "SUBDOMAIN_ALREADY_EXISTS", err.Error(), nil)
			return
		}
		h.respondWithError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Error inesperado", nil)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, tenant)
}

func (h *Handler) UpdateTenant(w http.ResponseWriter, r *http.Request, id string) {
	var req UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "El cuerpo de la petición no es un JSON válido", nil)
		return
	}

	tenant, err := h.service.UpdateTenant(r.Context(), id, req.Name, req.Subdomain, req.Active)
	if err != nil {
		if err == ErrTenantNotFound {
			h.respondWithError(w, http.StatusNotFound, "TENANT_NOT_FOUND", err.Error(), nil)
			return
		}
		if err == ErrSubdomainAlreadyExists {
			h.respondWithError(w, http.StatusBadRequest, "SUBDOMAIN_ALREADY_EXISTS", err.Error(), nil)
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
