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
	BrandID   string `json:"brand_id"`
	Subdomain string `json:"subdomain"`
}

type UpdateTenantRequest struct {
	Subdomain *string `json:"subdomain" validate:"omitempty,gt=0"`
	Active    *bool   `json:"active"`
}

var validate = validator.New()

func (h *Handler) HandleTenantRoutes(w http.ResponseWriter, r *http.Request) {

	trimmedPath := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(trimmedPath, "/")

	// /api/tenants
	if len(pathParts) == 2 &&
		pathParts[0] == "api" &&
		pathParts[1] == "tenants" {

		switch r.Method {

		case http.MethodGet:
			h.GetTenants(w, r)
			return

		case http.MethodPost:
			h.CreateTenant(w, r)
			return

		default:
			h.respondWithError(
				w,
				http.StatusMethodNotAllowed,
				"METHOD_NOT_ALLOWED",
				"Método no permitido",
				nil,
			)
			return
		}
	}

	// /api/tenants/{id}
	if len(pathParts) == 3 &&
		pathParts[0] == "api" &&
		pathParts[1] == "tenants" {

		id := pathParts[2]

		if id == "" {
			h.respondWithError(
				w,
				http.StatusBadRequest,
				"INVALID_ID",
				"ID de comercio inválido",
				nil,
			)
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
			h.respondWithError(
				w,
				http.StatusMethodNotAllowed,
				"METHOD_NOT_ALLOWED",
				"Método no permitido",
				nil,
			)
			return
		}
	}

	h.respondWithError(
		w,
		http.StatusNotFound,
		"NOT_FOUND",
		"Recurso no encontrado",
		nil,
	)
}

// CREATE

func (h *Handler) CreateTenant(w http.ResponseWriter, r *http.Request) {

	var req CreateTenantRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		h.respondWithError(
			w,
			http.StatusBadRequest,
			"INVALID_PAYLOAD",
			"El cuerpo de la petición no es un JSON válido",
			nil,
		)

		return
	}

	details := map[string]string{}

	if req.BrandID == "" {
		details["brand_id"] = "La marca es requerida"
	}

	if req.Subdomain == "" {
		details["subdomain"] = "El subdominio es requerido"
	}

	if len(details) > 0 {

		h.respondWithError(
			w,
			http.StatusBadRequest,
			"VALIDATION_FAILED",
			"Faltan campos obligatorios",
			details,
		)

		return
	}

	tenant, err := h.service.RegisterTenant(
		r.Context(),
		req.BrandID,
		req.Subdomain,
	)

	if err != nil {

		if err == ErrDuplicateBranch {

			h.respondWithError(
				w,
				http.StatusConflict,
				"DUPLICATED_BRANCH",
				err.Error(),
				nil,
			)

			return
		}

		h.respondWithError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_SERVER_ERROR",
			"Error inesperado",
			nil,
		)

		return
	}

	h.respondWithJSON(
		w,
		http.StatusCreated,
		tenant,
	)
}

// LIST

func (h *Handler) GetTenants(w http.ResponseWriter, r *http.Request) {

	tenants, err := h.service.GetAllTenants(r.Context())

	if err != nil {

		h.respondWithError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_SERVER_ERROR",
			"Error al obtener los comercios",
			nil,
		)

		return
	}

	h.respondWithJSON(
		w,
		http.StatusOK,
		tenants,
	)
}

// UPDATE

func (h *Handler) UpdateTenant(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {

	var req UpdateTenantRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		h.respondWithError(
			w,
			http.StatusBadRequest,
			"INVALID_PAYLOAD",
			"El cuerpo de la petición no es un JSON válido",
			nil,
		)

		return
	}

	if err := validate.Struct(req); err != nil {

		h.respondWithError(
			w,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"Error de validación",
			nil,
		)

		return
	}

	var subdomain string

	if req.Subdomain != nil {
		subdomain = *req.Subdomain
	}

	tenant, err := h.service.UpdateTenant(
		r.Context(),
		id,
		subdomain,
		req.Active,
	)

	if err != nil {

		switch err {

		case ErrTenantNotFound:

			h.respondWithError(
				w,
				http.StatusNotFound,
				"TENANT_NOT_FOUND",
				err.Error(),
				nil,
			)

		case ErrDuplicateBranch:

			h.respondWithError(
				w,
				http.StatusConflict,
				"DUPLICATED_BRANCH",
				err.Error(),
				nil,
			)

		default:

			h.respondWithError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_SERVER_ERROR",
				"Error inesperado",
				nil,
			)
		}

		return
	}

	h.respondWithJSON(
		w,
		http.StatusOK,
		tenant,
	)
}

// DELETE

func (h *Handler) DeleteTenant(w http.ResponseWriter, r *http.Request, id string) {

	err := h.service.DeleteTenant(
		r.Context(),
		id,
	)

	if err != nil {

		if err == ErrTenantNotFound {

			h.respondWithError(
				w,
				http.StatusNotFound,
				"TENANT_NOT_FOUND",
				err.Error(),
				nil,
			)

			return
		}

		h.respondWithError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_SERVER_ERROR",
			"Error inesperado",
			nil,
		)

		return
	}

	h.respondWithJSON(
		w,
		http.StatusOK,
		map[string]string{
			"message": "Comercio desactivado con éxito",
		},
	)
}

// HELPERS

func (h *Handler) respondWithError(
	w http.ResponseWriter,
	status int,
	code string,
	message string,
	details interface{},
) {

	h.respondWithJSON(
		w,
		status,
		ErrorResponse{
			Code:    code,
			Message: message,
			Details: details,
		},
	)
}

func (h *Handler) respondWithJSON(
	w http.ResponseWriter,
	status int,
	payload interface{},
) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(payload)
}
