package products

import (
	"encoding/json"
	"errors"
	"fmt"
	"leguiburger/internal/auth"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	service    Service
	imageStore ProductImageStore
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s, imageStore: NewProductImageStoreFromEnv()}
}

func NewHandlerWithImageStore(s Service, imageStore ProductImageStore) *Handler {
	if imageStore == nil {
		imageStore = NewProductImageStoreFromEnv()
	}
	return &Handler{service: s, imageStore: imageStore}
}

type CreateInput struct {
	Name         string  `json:"name"`
	BrandID      string  `json:"brand_id"`
	Description  string  `json:"description"`
	BasePrice    float64 `json:"base_price"`
	CurrentPrice float64 `json:"current_price"`
	ImageURL     string  `json:"image_url"`
}

type UpdateInput struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	BasePrice    *float64 `json:"base_price"`
	CurrentPrice *float64 `json:"current_price"`
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
	input, uploadedImageURL, err := h.decodeCreateInput(r)
	if err != nil {
		h.handleDecodeError(w, err)
		return
	}

	brandID, err := h.brandIDFromRequest(r, input.BrandID)
	if err != nil {
		h.handleBrandRequestError(w, err)
		return
	}

	basePrice := input.BasePrice
	if basePrice == 0 && input.CurrentPrice > 0 {
		basePrice = input.CurrentPrice
	}
	if uploadedImageURL != "" {
		input.ImageURL = uploadedImageURL
	}

	product, err := h.service.CreateProduct(r.Context(), brandID, input.Name, input.Description, basePrice, input.ImageURL)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, product)
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	brandID, err := h.brandIDFromRequest(r, r.URL.Query().Get("brand_id"))
	if err != nil {
		h.handleBrandRequestError(w, err)
		return
	}

	products, err := h.service.ListProducts(r.Context(), brandID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, products)
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request, id string) {
	brandID, err := h.brandIDFromRequest(r, r.URL.Query().Get("brand_id"))
	if err != nil {
		h.handleBrandRequestError(w, err)
		return
	}

	product, err := h.service.GetProduct(r.Context(), brandID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, product)
}

func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request, id string) {
	input, uploadedImageURL, err := h.decodeUpdateInput(r)
	if err != nil {
		h.handleDecodeError(w, err)
		return
	}

	brandID, err := h.brandIDFromRequest(r, r.URL.Query().Get("brand_id"))
	if err != nil {
		h.handleBrandRequestError(w, err)
		return
	}

	basePrice := input.BasePrice
	if basePrice == nil {
		basePrice = input.CurrentPrice
	}
	if uploadedImageURL != "" {
		input.ImageURL = uploadedImageURL
	}

	product, err := h.service.UpdateProduct(r.Context(), brandID, id, input.Name, input.Description, basePrice, input.ImageURL, input.IsActive)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, product)
}

func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request, id string) {
	brandID, err := h.brandIDFromRequest(r, r.URL.Query().Get("brand_id"))
	if err != nil {
		h.handleBrandRequestError(w, err)
		return
	}

	if err := h.service.DeleteProduct(r.Context(), brandID, id); err != nil {
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
	} else if errors.Is(err, ErrBrandNotFoundForProduct) {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_BRAND", err.Error())
	} else {
		h.respondWithError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Error inesperado")
	}
}

func (h *Handler) handleDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrProductImageUploadFailed) {
		log.Printf("product image upload failed: %v", err)
		h.respondWithError(w, http.StatusBadRequest, "IMAGE_UPLOAD_FAILED", "No se pudo subir la imagen del producto")
		return
	}

	h.respondWithError(w, http.StatusBadRequest, "INVALID_INPUT", "Datos inválidos")
}

func (h *Handler) brandIDFromRequest(r *http.Request, explicitBrandID string) (string, error) {
	claims, ok := auth.GetClaimsFromContext(r.Context())
	if !ok {
		return "", errors.New("UNAUTHORIZED")
	}

	brandID := strings.TrimSpace(explicitBrandID)
	if brandID == "" {
		brandID = strings.TrimSpace(r.Header.Get("X-Brand-ID"))
	}
	if brandID == "" && claims.BrandID != nil {
		brandID = strings.TrimSpace(*claims.BrandID)
	}
	if brandID == "" {
		return "", ErrBrandNotFoundForProduct
	}
	if claims.Role != auth.RoleOwner && claims.BrandID != nil && brandID != strings.TrimSpace(*claims.BrandID) {
		return "", auth.ErrForbiddenTenant
	}

	return brandID, nil
}

func (h *Handler) handleBrandRequestError(w http.ResponseWriter, err error) {
	if errors.Is(err, auth.ErrForbiddenTenant) {
		h.respondWithError(w, http.StatusForbidden, "FORBIDDEN", "No puedes operar en esta marca")
		return
	}
	if errors.Is(err, ErrBrandNotFoundForProduct) {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_BRAND_ID", "Falta el ID de la marca")
		return
	}
	h.respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "No autorizado")
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

func (h *Handler) decodeCreateInput(r *http.Request) (CreateInput, string, error) {
	if isMultipartRequest(r) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			return CreateInput{}, "", err
		}

		input := CreateInput{
			Name:        strings.TrimSpace(r.FormValue("name")),
			BrandID:     strings.TrimSpace(r.FormValue("brand_id")),
			Description: r.FormValue("description"),
			ImageURL:    strings.TrimSpace(r.FormValue("image_url")),
		}
		if basePrice, err := parseOptionalFloat(r.FormValue("base_price")); err == nil {
			input.BasePrice = basePrice
		}
		if currentPrice, err := parseOptionalFloat(r.FormValue("current_price")); err == nil {
			input.CurrentPrice = currentPrice
		}

		fileHeader, err := fileHeaderFromForm(r, "image_file")
		if err != nil {
			return CreateInput{}, "", err
		}
		if fileHeader != nil {
			imageURL, err := h.storeProductImage(r, fileHeader)
			if err != nil {
				return CreateInput{}, "", fmt.Errorf("%w: %v", ErrProductImageUploadFailed, err)
			}
			return input, imageURL, nil
		}

		return input, "", nil
	}

	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return CreateInput{}, "", err
	}
	return input, "", nil
}

func (h *Handler) decodeUpdateInput(r *http.Request) (UpdateInput, string, error) {
	if isMultipartRequest(r) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			return UpdateInput{}, "", err
		}

		input := UpdateInput{
			Name:        strings.TrimSpace(r.FormValue("name")),
			Description: r.FormValue("description"),
			ImageURL:    "",
		}
		if basePriceStr := strings.TrimSpace(r.FormValue("base_price")); basePriceStr != "" {
			if basePrice, err := strconv.ParseFloat(basePriceStr, 64); err == nil {
				input.BasePrice = &basePrice
			}
		}
		if currentPriceStr := strings.TrimSpace(r.FormValue("current_price")); currentPriceStr != "" {
			if currentPrice, err := strconv.ParseFloat(currentPriceStr, 64); err == nil {
				input.CurrentPrice = &currentPrice
			}
		}
		if isActiveStr := strings.TrimSpace(r.FormValue("is_active")); isActiveStr != "" {
			if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
				input.IsActive = &isActive
			}
		}

		imageURL := strings.TrimSpace(r.FormValue("image_url"))
		fileHeader, err := fileHeaderFromForm(r, "image_file")
		if err != nil {
			return UpdateInput{}, "", err
		}
		if fileHeader != nil {
			savedImageURL, err := h.storeProductImage(r, fileHeader)
			if err != nil {
				return UpdateInput{}, "", fmt.Errorf("%w: %v", ErrProductImageUploadFailed, err)
			}
			imageURL = savedImageURL
		}
		if imageURL != "" {
			input.ImageURL = imageURL
		}

		return input, imageURL, nil
	}

	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return UpdateInput{}, "", err
	}
	return input, input.ImageURL, nil
}

func (h *Handler) storeProductImage(r *http.Request, fileHeader *multipart.FileHeader) (string, error) {
	if fileHeader == nil {
		return "", nil
	}

	return h.imageStore.SaveProductImage(r.Context(), fileHeader)
}

func parseOptionalFloat(value string) (float64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	return strconv.ParseFloat(trimmed, 64)
}

func fileHeaderFromForm(r *http.Request, fieldName string) (*multipart.FileHeader, error) {
	_, fileHeader, err := r.FormFile(fieldName)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return nil, nil
		}
		return nil, err
	}
	return fileHeader, nil
}

func isMultipartRequest(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data")
}
