package tenantproducts

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
	ProductID     string   `json:"product_id"`
	PriceOverride *float64 `json:"price_override"`
	CurrentStock  int      `json:"current_stock"`
	TrackStock    *bool    `json:"track_stock"`
	IsAvailable   *bool    `json:"is_available"`
}

type UpdateInput struct {
	PriceOverride *float64 `json:"price_override"`
	CurrentStock  *int     `json:"current_stock"`
	TrackStock    *bool    `json:"track_stock"`
	IsAvailable   *bool    `json:"is_available"`
	IsActive      *bool    `json:"is_active"`
}

func (h *Handler) HandleTenantProductRoutes(w http.ResponseWriter, r *http.Request) {
	trimmedPath := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(trimmedPath, "/")

	if len(pathParts) == 2 && pathParts[0] == "api" && pathParts[1] == "tenant-products" {
		switch r.Method {
		case http.MethodPost:
			h.CreateTenantProduct(w, r)
			return
		case http.MethodGet:
			h.ListTenantProducts(w, r)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido")
			return
		}
	}

	if len(pathParts) == 3 && pathParts[0] == "api" && pathParts[1] == "tenant-products" {
		productID := pathParts[2]
		switch r.Method {
		case http.MethodGet:
			h.GetTenantProduct(w, r, productID)
			return
		case http.MethodPut:
			h.UpdateTenantProduct(w, r, productID)
			return
		case http.MethodDelete:
			h.DeleteTenantProduct(w, r, productID)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido")
			return
		}
	}

	h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", "Recurso no encontrado")
}

func (h *Handler) CreateTenantProduct(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}

	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_INPUT", "JSON inválido")
		return
	}

	tenantProduct, err := h.service.CreateTenantProduct(r.Context(), tenantID, input.ProductID, input.PriceOverride, input.CurrentStock, input.TrackStock, input.IsAvailable)
	if err != nil {
		h.handleError(w, err)
		return
	}
	h.respondWithJSON(w, http.StatusCreated, tenantProduct)
}

func (h *Handler) ListTenantProducts(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}

	tenantProducts, err := h.service.ListTenantProducts(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	h.respondWithJSON(w, http.StatusOK, tenantProducts)
}

func (h *Handler) GetTenantProduct(w http.ResponseWriter, r *http.Request, productID string) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}

	tenantProduct, err := h.service.GetTenantProduct(r.Context(), tenantID, productID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	h.respondWithJSON(w, http.StatusOK, tenantProduct)
}

func (h *Handler) UpdateTenantProduct(w http.ResponseWriter, r *http.Request, productID string) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}

	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_INPUT", "JSON inválido")
		return
	}

	tenantProduct, err := h.service.UpdateTenantProduct(r.Context(), tenantID, productID, input.PriceOverride, input.CurrentStock, input.TrackStock, input.IsAvailable, input.IsActive)
	if err != nil {
		h.handleError(w, err)
		return
	}
	h.respondWithJSON(w, http.StatusOK, tenantProduct)
}

func (h *Handler) DeleteTenantProduct(w http.ResponseWriter, r *http.Request, productID string) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}

	if err := h.service.DeleteTenantProduct(r.Context(), tenantID, productID); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) tenantID(w http.ResponseWriter, r *http.Request) (string, bool) {
	tenantID, err := auth.TenantIDFromRequest(r)
	if err == nil {
		return tenantID, true
	}
	if errors.Is(err, auth.ErrMissingTenantID) {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return "", false
	}
	if errors.Is(err, auth.ErrForbiddenTenant) {
		h.respondWithError(w, http.StatusForbidden, "FORBIDDEN", "No puedes operar en este comercio")
		return "", false
	}
	h.respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "No autorizado")
	return "", false
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrDuplicateTenantProduct) {
		h.respondWithError(w, http.StatusConflict, "DUPLICATE_TENANT_PRODUCT", err.Error())
	} else if errors.Is(err, ErrTenantProductNotFound) {
		h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	} else if errors.Is(err, ErrInvalidTenantProductData) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_TENANT_PRODUCT_DATA", err.Error())
	} else if errors.Is(err, ErrInvalidPriceOverride) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_PRICE_OVERRIDE", err.Error())
	} else if errors.Is(err, ErrInvalidTenantStock) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_TENANT_STOCK", err.Error())
	} else if errors.Is(err, ErrProductBrandMismatch) {
		h.respondWithError(w, http.StatusBadRequest, "PRODUCT_BRAND_MISMATCH", err.Error())
	} else if errors.Is(err, ErrInactiveBaseProduct) {
		h.respondWithError(w, http.StatusBadRequest, "INACTIVE_BASE_PRODUCT", err.Error())
	} else if errors.Is(err, ErrTenantNotFound) {
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
