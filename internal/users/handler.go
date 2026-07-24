package users

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"leguiburger/internal/auth"
)

type Handler struct {
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

type CreateInput struct {
	BrandID   string `json:"brand_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Phone     string `json:"phone"`
}

type UpdateInput struct {
	BrandID   *string `json:"brand_id"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     string  `json:"email"`
	Password  string  `json:"password"`
	Phone     string  `json:"phone"`
	Role      string  `json:"role"`
	IsActive  *bool   `json:"is_active"`
}

func (h *Handler) HandleUserRoutes(w http.ResponseWriter, r *http.Request) {
	trimmedPath := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(trimmedPath, "/")

	if len(pathParts) == 2 && pathParts[0] == "api" && pathParts[1] == "users" {
		switch r.Method {
		case http.MethodPost:
			h.CreateUser(w, r)
			return
		case http.MethodGet:
			h.ListUsers(w, r)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido")
			return
		}
	}

	if len(pathParts) == 3 && pathParts[0] == "api" && pathParts[1] == "users" {
		id := pathParts[2]
		if id == "" {
			h.respondWithError(w, http.StatusBadRequest, "INVALID_ID", "ID de usuario inválido")
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.GetUser(w, r, id)
			return
		case http.MethodPut:
			h.UpdateUser(w, r, id)
			return
		case http.MethodDelete:
			h.DeleteUser(w, r, id)
			return
		default:
			h.respondWithError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido")
			return
		}
	}

	h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", "Recurso no encontrado")
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_INPUT", "JSON inválido")
		return
	}

	claims, ok := auth.GetClaimsFromContext(r.Context())
	if !ok || strings.ToLower(strings.TrimSpace(claims.Role)) != auth.RoleOwner {
		h.respondWithError(w, http.StatusForbidden, "FORBIDDEN", "No autorizado")
		return
	}

	user, err := h.service.CreateUser(r.Context(), input.FirstName, input.LastName, input.BrandID, input.Email, input.Password, "admin")
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, user)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaimsFromContext(r.Context())
	if !ok || strings.ToLower(strings.TrimSpace(claims.Role)) != auth.RoleOwner {
		h.respondWithError(w, http.StatusForbidden, "FORBIDDEN", "No autorizado")
		return
	}

	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, users)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request, id string) {
	claims, ok := auth.GetClaimsFromContext(r.Context())
	if !ok || strings.ToLower(strings.TrimSpace(claims.Role)) != auth.RoleOwner {
		h.respondWithError(w, http.StatusForbidden, "FORBIDDEN", "No autorizado")
		return
	}

	user, err := h.service.GetUser(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, user)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request, id string) {
	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_INPUT", "JSON inválido")
		return
	}

	claims, ok := auth.GetClaimsFromContext(r.Context())
	if !ok || strings.ToLower(strings.TrimSpace(claims.Role)) != auth.RoleOwner {
		h.respondWithError(w, http.StatusForbidden, "FORBIDDEN", "No autorizado")
		return
	}

	user, err := h.service.UpdateUser(r.Context(), id, input.FirstName, input.LastName, input.Email, input.Password, input.Role, input.BrandID, input.IsActive)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, user)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request, id string) {
	claims, ok := auth.GetClaimsFromContext(r.Context())
	if !ok || strings.ToLower(strings.TrimSpace(claims.Role)) != auth.RoleOwner {
		h.respondWithError(w, http.StatusForbidden, "FORBIDDEN", "No autorizado")
		return
	}

	if err := h.service.DeleteUser(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Usuario desactivado con éxito"})
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrDuplicateUserEmail):
		h.respondWithError(w, http.StatusConflict, "DUPLICATE_USER_EMAIL", err.Error())
	case errors.Is(err, ErrUserNotFound):
		h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, ErrInvalidUserData):
		h.respondWithError(w, http.StatusBadRequest, "INVALID_USER_DATA", err.Error())
	case errors.Is(err, ErrInvalidUserRole):
		h.respondWithError(w, http.StatusBadRequest, "INVALID_USER_ROLE", err.Error())
	case errors.Is(err, ErrBrandNotFound):
		h.respondWithError(w, http.StatusBadRequest, "INVALID_BRAND", err.Error())
	case errors.Is(err, ErrUnauthorizedAction):
		h.respondWithError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	default:
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
