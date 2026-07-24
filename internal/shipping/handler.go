package shipping

import (
	"encoding/json"
	"errors"
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
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Typification  string  `json:"typification"`
	Cost          float64 `json:"cost"`
	EstimatedTime string  `json:"estimated_time"`
}

type UpdateInput struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Typification  string   `json:"typification"`
	Cost          *float64 `json:"cost"`
	EstimatedTime string   `json:"estimated_time"`
	IsActive      *bool    `json:"is_active"`
}

type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (h *Handler) HandleShippingRoutes(w http.ResponseWriter, r *http.Request) {
	trimmedPath := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(trimmedPath, "/")

	// CASO A: /api/shipping-methods (Crear - POST, Listar - GET)
	if len(pathParts) == 2 && pathParts[0] == "api" && pathParts[1] == "shipping-methods" {
		switch r.Method {
		case http.MethodPost:
			h.CreateShippingMethod(w, r)
			return
		case http.MethodGet:
			h.ListShippingMethods(w, r)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido")
			return
		}
	}

	// CASO B: /api/shipping-methods/{id} (Obtener - GET, Editar - PUT, Eliminar - DELETE)
	if len(pathParts) == 3 && pathParts[0] == "api" && pathParts[1] == "shipping-methods" {
		id := pathParts[2]

		if id == "" {
			h.respondWithError(w, http.StatusBadRequest, "INVALID_ID", "ID de método de envío inválido")
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.GetShippingMethod(w, r, id)
			return
		case http.MethodPut:
			h.UpdateShippingMethod(w, r, id)
			return
		case http.MethodDelete:
			h.DeleteShippingMethod(w, r, id)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido")
			return
		}
	}

	// Si no coincide con ninguna ruta conocida
	h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", "Recurso no encontrado")
}

// --- MÉTODOS DEL CRUD ADAPTADOS ---

func (h *Handler) CreateShippingMethod(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_INPUT", "JSON inválido")
		return
	}

	method, err := h.service.CreateMethod(
		r.Context(),
		tenantID,
		input.Name,
		input.Typification,
		input.Description,
		input.Cost,
		input.EstimatedTime,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, method)
}

func (h *Handler) ListShippingMethods(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	methods, err := h.service.ListMethods(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, methods)
}

func (h *Handler) GetShippingMethod(w http.ResponseWriter, r *http.Request, id string) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	method, err := h.service.GetMethod(r.Context(), tenantID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, method)
}

func (h *Handler) UpdateShippingMethod(w http.ResponseWriter, r *http.Request, id string) {
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

	method, err := h.service.UpdateMethod(r.Context(), tenantID, id, input.Name, input.Typification, input.Description, input.Cost, input.EstimatedTime, input.IsActive)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, method)
}

func (h *Handler) DeleteShippingMethod(w http.ResponseWriter, r *http.Request, id string) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	if err := h.service.DeleteMethod(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- UTILERÍAS ---

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrDuplicateShipping) {
		h.respondWithError(w, http.StatusConflict, "DUPLICATE_SHIPPING", err.Error())
	} else if errors.Is(err, ErrShippingNotFound) {
		h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	} else if errors.Is(err, ErrInvalidCost) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_COST", err.Error())
	} else if errors.Is(err, ErrTenantNotFoundForShipping) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_TENANT", err.Error())
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
