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
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) HandleAuthRoutes(w http.ResponseWriter, r *http.Request) {
	trimmedPath := strings.Trim(r.URL.Path, "/")

	if trimmedPath == "api/auth/login" && r.Method == http.MethodPost {
		h.Login(w, r)
		return
	}

	h.respondWithError(w, http.StatusNotFound, "NOT_FOUND", "Recurso no encontrado")
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	// 1. Decodificar el JSON primero
	var input struct {
		TenantID string `json:"tenant_id"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "INVALID_INPUT", "JSON inválido")
		return
	}

	// 2. Determinar el tenantID: Prioridad al Header, sino se usa el del Body
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		tenantID = strings.TrimSpace(input.TenantID)
	}

	// 3. Solo requerir Email y Password obligatorios
	if strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.Password) == "" {
		h.respondWithError(w, http.StatusBadRequest, "MISSING_FIELDS", "Email y contraseña son requeridos")
		return
	}

	// 4. Ejecutar el login (tenantID puede ser "" para el OWNER global)
	res, err := h.service.Login(r.Context(), tenantID, input.Email, input.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			h.respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Credenciales inválidas")
		} else if errors.Is(err, ErrTenantNotFoundForAuth) {
			h.respondWithError(w, http.StatusBadRequest, "INVALID_TENANT", "El comercio no existe")
		} else {
			h.respondWithError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Error inesperado")
		}
		return
	}

	h.respondWithJSON(w, http.StatusOK, res)
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
