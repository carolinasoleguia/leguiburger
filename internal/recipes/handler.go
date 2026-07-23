package recipes

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
	ProductID    string  `json:"product_id"`
	SupplyID     string  `json:"supply_id"`
	QuantityUsed float64 `json:"quantity_used"`
}

type UpdateInput struct {
	QuantityUsed float64 `json:"quantity_used"`
}

type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (h *Handler) HandleRecipeRoutes(w http.ResponseWriter, r *http.Request) {
	trimmedPath := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(trimmedPath, "/")

	if len(pathParts) == 2 && pathParts[0] == "api" && pathParts[1] == "recipes" {
		switch r.Method {
		case http.MethodPost:
			h.CreateRecipe(w, r)
			return
		case http.MethodGet:
			h.ListRecipes(w, r)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido")
			return
		}
	}

	if len(pathParts) == 4 && pathParts[0] == "api" && pathParts[1] == "recipes" {
		productID := pathParts[2]
		supplyID := pathParts[3]
		if productID == "" || supplyID == "" {
			h.respondWithError(w, http.StatusBadRequest, "INVALID_ID", "ID de producto o insumo inválido")
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.GetRecipe(w, r, productID, supplyID)
			return
		case http.MethodPut:
			h.UpdateRecipe(w, r, productID, supplyID)
			return
		case http.MethodDelete:
			h.DeleteRecipe(w, r, productID, supplyID)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido")
			return
		}
	}

	h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", "Recurso no encontrado")
}

func (h *Handler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
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

	recipe, err := h.service.CreateRecipe(r.Context(), tenantID, input.ProductID, input.SupplyID, input.QuantityUsed)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, recipe)
}

func (h *Handler) ListRecipes(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	recipes, err := h.service.ListRecipes(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, recipes)
}

func (h *Handler) GetRecipe(w http.ResponseWriter, r *http.Request, productID, supplyID string) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	recipe, err := h.service.GetRecipe(r.Context(), tenantID, productID, supplyID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, recipe)
}

func (h *Handler) UpdateRecipe(w http.ResponseWriter, r *http.Request, productID, supplyID string) {
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

	recipe, err := h.service.UpdateRecipe(r.Context(), tenantID, productID, supplyID, input.QuantityUsed)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, recipe)
}

func (h *Handler) DeleteRecipe(w http.ResponseWriter, r *http.Request, productID, supplyID string) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_TENANT_ID", "Falta el ID del comercio")
		return
	}

	if err := h.service.DeleteRecipe(r.Context(), tenantID, productID, supplyID); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrDuplicateRecipe) {
		h.respondWithError(w, http.StatusConflict, "DUPLICATE_RECIPE", err.Error())
	} else if errors.Is(err, ErrRecipeNotFound) {
		h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	} else if errors.Is(err, ErrInvalidRecipeData) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_RECIPE_DATA", err.Error())
	} else if errors.Is(err, ErrInvalidRecipeQuantity) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_RECIPE_QUANTITY", err.Error())
	} else if errors.Is(err, ErrProductNotFoundForRecipe) {
		h.respondWithError(w, http.StatusBadRequest, "PRODUCT_NOT_FOUND", err.Error())
	} else if errors.Is(err, ErrSupplyNotFoundForRecipe) {
		h.respondWithError(w, http.StatusBadRequest, "SUPPLY_NOT_FOUND", err.Error())
	} else if errors.Is(err, ErrTenantNotFoundForRecipe) {
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
