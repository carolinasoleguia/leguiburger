package auth

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

type LoginInput struct {
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) HandleAuthRoutes(w http.ResponseWriter, r *http.Request) {
	trimmedPath := strings.Trim(r.URL.Path, "/")

	if trimmedPath == "api/auth/login" && r.Method == http.MethodPost {
		h.Login(w, r)
		return
	}

	RespondWithError(w, http.StatusNotFound, "NOT_FOUND", "Recurso no encontrado")
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input LoginInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		RespondWithError(w, http.StatusBadRequest, "INVALID_INPUT", "JSON invalido")
		return
	}

	tenantID := strings.TrimSpace(r.Header.Get(TenantHeaderName))
	if tenantID == "" {
		tenantID = strings.TrimSpace(input.TenantID)
	}

	if strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.Password) == "" {
		RespondWithError(w, http.StatusBadRequest, "MISSING_FIELDS", "Email y contrasena son requeridos")
		return
	}

	res, err := h.service.Login(r.Context(), tenantID, input.Email, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			RespondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Credenciales invalidas")
		case errors.Is(err, ErrTenantRequired):
			RespondWithError(w, http.StatusForbidden, "TENANT_REQUIRED", "El comercio es requerido para este usuario")
		case errors.Is(err, ErrForbiddenTenant):
			RespondWithError(w, http.StatusForbidden, "FORBIDDEN", "No autorizado para este comercio")
		case errors.Is(err, ErrTenantNotFoundForAuth):
			RespondWithError(w, http.StatusForbidden, "INVALID_TENANT", "El comercio no existe o esta inactivo")
		default:
			RespondWithError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Error inesperado")
		}
		return
	}

	respondWithJSON(w, http.StatusOK, res)
}

func respondWithJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
