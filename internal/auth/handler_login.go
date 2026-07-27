package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type LoginLookupInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginLookupResponse struct {
	Choices []TenantChoice `json:"choices"`
}

func (h *Handler) LoginLookup(w http.ResponseWriter, r *http.Request) {
	var input LoginLookupInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		RespondWithError(w, http.StatusBadRequest, "INVALID_INPUT", "JSON invalido")
		return
	}

	if strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.Password) == "" {
		RespondWithError(w, http.StatusBadRequest, "MISSING_FIELDS", "Email y contrasena son requeridos")
		return
	}

	choices, err := h.service.LookupTenantsForEmail(r.Context(), input.Email, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			RespondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Credenciales invalidas")
		default:
			RespondWithError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Error inesperado")
		}
		return
	}

	respondWithJSON(w, http.StatusOK, LoginLookupResponse{Choices: choices})
}
