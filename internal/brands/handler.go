package brands

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
	Name  string `json:"name"`
	TaxID string `json:"tax_id"`
}

type UpdateInput struct {
	Name  string `json:"name"`
	TaxID string `json:"tax_id"`
}

func (h *Handler) HandleBrandRoutes(w http.ResponseWriter, r *http.Request) {

	trimmedPath := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(trimmedPath, "/")

	// /api/brands
	// POST crear
	// GET listar
	if len(pathParts) == 2 &&
		pathParts[0] == "api" &&
		pathParts[1] == "brands" {

		switch r.Method {

		case http.MethodPost:
			h.CreateBrand(w, r)
			return

		case http.MethodGet:
			h.ListBrands(w, r)
			return

		default:
			h.respondWithError(
				w,
				http.StatusMethodNotAllowed,
				"METHOD_NOT_ALLOWED",
				"Método no permitido",
			)
			return
		}
	}

	// /api/brands/{id}
	if len(pathParts) == 3 &&
		pathParts[0] == "api" &&
		pathParts[1] == "brands" {

		id := pathParts[2]

		if id == "" {
			h.respondWithError(
				w,
				http.StatusBadRequest,
				"INVALID_ID",
				"ID de marca inválido",
			)
			return
		}

		switch r.Method {

		case http.MethodGet:
			h.GetBrand(w, r, id)
			return

		case http.MethodPut:
			h.UpdateBrand(w, r, id)
			return

		case http.MethodDelete:
			h.DeleteBrand(w, r, id)
			return

		default:
			h.respondWithError(
				w,
				http.StatusMethodNotAllowed,
				"METHOD_NOT_ALLOWED",
				"Método no permitido",
			)
			return
		}
	}

	h.respondWithError(
		w,
		http.StatusNotFound,
		"NOT_FOUND",
		"Recurso no encontrado",
	)
}

// CREATE

func (h *Handler) CreateBrand(w http.ResponseWriter, r *http.Request) {

	var input CreateInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.respondWithError(
			w,
			http.StatusBadRequest,
			"INVALID_INPUT",
			"JSON inválido",
		)
		return
	}

	brand, err := h.service.Create(
		r.Context(),
		input.Name,
		input.TaxID,
	)

	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(
		w,
		http.StatusCreated,
		brand,
	)
}

// LIST

func (h *Handler) ListBrands(w http.ResponseWriter, r *http.Request) {

	brands, err := h.service.List(r.Context())

	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(
		w,
		http.StatusOK,
		brands,
	)
}

// GET

func (h *Handler) GetBrand(w http.ResponseWriter, r *http.Request, id string) {

	brand, err := h.service.Get(
		r.Context(),
		id,
	)

	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(
		w,
		http.StatusOK,
		brand,
	)
}

// UPDATE

func (h *Handler) UpdateBrand(w http.ResponseWriter, r *http.Request, id string) {

	var input UpdateInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {

		h.respondWithError(
			w,
			http.StatusBadRequest,
			"INVALID_INPUT",
			"JSON inválido",
		)
		return
	}

	brand, err := h.service.Update(
		r.Context(),
		id,
		input.Name,
		input.TaxID,
	)

	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(
		w,
		http.StatusOK,
		brand,
	)
}

// DELETE

func (h *Handler) DeleteBrand(w http.ResponseWriter, r *http.Request, id string) {

	err := h.service.Delete(
		r.Context(),
		id,
	)

	if err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ERROR HANDLING

func (h *Handler) handleError(w http.ResponseWriter, err error) {

	switch {

	case errors.Is(err, ErrBrandNotFound):

		h.respondWithError(
			w,
			http.StatusNotFound,
			"NOT_FOUND",
			err.Error(),
		)

	case errors.Is(err, ErrDuplicateBrand):

		h.respondWithError(
			w,
			http.StatusConflict,
			"DUPLICATE_BRAND",
			err.Error(),
		)

	default:

		h.respondWithError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"Error inesperado",
		)
	}

}

func (h *Handler) respondWithError(
	w http.ResponseWriter,
	status int,
	code string,
	msg string,
) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(
		map[string]string{
			"code":    code,
			"message": msg,
		},
	)
}

func (h *Handler) respondWithJSON(
	w http.ResponseWriter,
	status int,
	data interface{},
) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(data)
}
