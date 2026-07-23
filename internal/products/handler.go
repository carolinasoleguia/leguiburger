package products

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
	Description  string  `json:"description"`
	CurrentPrice float64 `json:"current_price"`
	CurrentStock int     `json:"current_stock"`
	TrackStock   *bool   `json:"track_stock"`
	ImageURL     string  `json:"image_url"`
}

type UpdateInput struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	CurrentPrice *float64 `json:"current_price"`
	CurrentStock *int     `json:"current_stock"`
	TrackStock   *bool    `json:"track_stock"`
	ImageURL     string   `json:"image_url"`
	IsActive     *bool    `json:"is_active"`
}

type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (h *Handler) HandleProductRoutes(w http.ResponseWriter, r *http.Request) {
	trimmedPath := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(trimmedPath, "/")

	if len(pathParts) == 2 && pathParts[0] == "api" && pathParts[1] == "products" {
		switch r.Method {
		case http.MethodPost:
			h.CreateProduct(w, r)
			return
		case http.MethodGet:
			h.ListProducts(w, r)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido")
			return
		}
	}

	if len(pathParts) == 3 && pathParts[0] == "api" && pathParts[1] == "products" {
		id := pathParts[2]
		if id == "" {
			h.respondWithError(w, http.StatusBadRequest, "INVALID_ID", "ID de producto inválido")
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.GetProduct(w, r, id)
			return
		case http.MethodPut:
			h.UpdateProduct(w, r, id)
			return
		case http.MethodDelete:
			h.DeleteProduct(w, r, id)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido")
			return
		}
	}

	h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", "Recurso no encontrado")
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
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

	product, err := h.service.CreateProduct(r.Context(), tenantID, input.Name, input.Description, input.CurrentPrice, input.CurrentStock, input.TrackStock, input.ImageURL)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, product)
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	products, err := h.service.ListProducts(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, products)
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request, id string) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	product, err := h.service.GetProduct(r.Context(), tenantID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, product)
}

func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request, id string) {
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

	product, err := h.service.UpdateProduct(r.Context(), tenantID, id, input.Name, input.Description, input.CurrentPrice, input.CurrentStock, input.TrackStock, input.ImageURL, input.IsActive)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, product)
}

func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request, id string) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	if err := h.service.DeleteProduct(r.Context(), tenantID, id); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrDuplicateProductName) {
		h.respondWithError(w, http.StatusConflict, "DUPLICATE_PRODUCT_NAME", err.Error())
	} else if errors.Is(err, ErrProductNotFound) {
		h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	} else if errors.Is(err, ErrInvalidProductData) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_PRODUCT_DATA", err.Error())
	} else if errors.Is(err, ErrInvalidProductPrice) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_PRODUCT_PRICE", err.Error())
	} else if errors.Is(err, ErrInvalidProductStock) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_PRODUCT_STOCK", err.Error())
	} else if errors.Is(err, ErrTenantNotFoundForProduct) {
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
