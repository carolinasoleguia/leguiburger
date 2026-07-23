package extras

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
	Name         string  `json:"name"`
	CurrentPrice float64 `json:"current_price"`
	CurrentStock int     `json:"current_stock"`
	TrackStock   *bool   `json:"track_stock"`
}

type UpdateInput struct {
	Name         string   `json:"name"`
	CurrentPrice *float64 `json:"current_price"`
	CurrentStock *int     `json:"current_stock"`
	TrackStock   *bool    `json:"track_stock"`
	IsActive     *bool    `json:"is_active"`
}

type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (h *Handler) HandleExtraRoutes(w http.ResponseWriter, r *http.Request) {
	trimmedPath := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(trimmedPath, "/")

	if len(pathParts) == 2 && pathParts[0] == "api" && pathParts[1] == "extras" {
		switch r.Method {
		case http.MethodPost:
			h.CreateExtra(w, r)
			return
		case http.MethodGet:
			h.ListExtras(w, r)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido")
			return
		}
	}

	if len(pathParts) == 3 && pathParts[0] == "api" && pathParts[1] == "extras" {
		id := pathParts[2]
		if id == "" {
			h.respondWithError(w, http.StatusBadRequest, "INVALID_ID", "ID de extra inválido")
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.GetExtra(w, r, id)
			return
		case http.MethodPut:
			h.UpdateExtra(w, r, id)
			return
		case http.MethodDelete:
			h.DeleteExtra(w, r, id)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido")
			return
		}
	}

	h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", "Recurso no encontrado")
}

func (h *Handler) CreateExtra(w http.ResponseWriter, r *http.Request) {
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

	extra, err := h.service.CreateExtra(r.Context(), tenantID, input.Name, input.CurrentPrice, input.CurrentStock, input.TrackStock)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, extra)
}

func (h *Handler) ListExtras(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	extras, err := h.service.ListExtras(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, extras)
}

func (h *Handler) GetExtra(w http.ResponseWriter, r *http.Request, id string) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	extra, err := h.service.GetExtra(r.Context(), tenantID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, extra)
}

func (h *Handler) UpdateExtra(w http.ResponseWriter, r *http.Request, id string) {
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

	extra, err := h.service.UpdateExtra(r.Context(), tenantID, id, input.Name, input.CurrentPrice, input.CurrentStock, input.TrackStock, input.IsActive)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, extra)
}

func (h *Handler) DeleteExtra(w http.ResponseWriter, r *http.Request, id string) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	if err := h.service.DeleteExtra(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrDuplicateExtraName) {
		h.respondWithError(w, http.StatusConflict, "DUPLICATE_EXTRA_NAME", err.Error())
	} else if errors.Is(err, ErrExtraNotFound) {
		h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	} else if errors.Is(err, ErrInvalidExtraData) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_EXTRA_DATA", err.Error())
	} else if errors.Is(err, ErrInvalidExtraPrice) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_EXTRA_PRICE", err.Error())
	} else if errors.Is(err, ErrInvalidExtraStock) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_EXTRA_STOCK", err.Error())
	} else if errors.Is(err, ErrTenantNotFoundForExtra) {
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
