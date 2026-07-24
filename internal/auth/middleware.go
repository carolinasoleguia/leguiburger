package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

type contextKey string

const ClaimsKey contextKey = "user_claims"

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("--- [DEBUG AuthMiddleware] Entró una petición a:", r.Method, r.URL.Path)

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Println("DEBUG: No vino el header Authorization")
			RespondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Se requiere token de autenticacion")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Println("DEBUG: Formato de token inválido")
			RespondWithError(w, http.StatusUnauthorized, "INVALID_TOKEN_FORMAT", "Formato de token invalido")
			return
		}

		log.Println("DEBUG: Token recibido, intentando validar...")
		claims, err := ValidateToken(parts[1])
		if err != nil {
			log.Println("DEBUG: Error al validar token:", err)
			RespondWithError(w, http.StatusUnauthorized, "INVALID_OR_EXPIRED_TOKEN", "Token invalido o expirado")
			return
		}

		log.Println("DEBUG: ¡Token VÁLIDO! Permitiendo acceso al usuario:", claims.Email)
		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func GetClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsKey).(*Claims)
	return claims, ok
}

func RespondWithError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Code:    code,
		Message: message,
	})
}

var (
	ErrMissingTenantID = errors.New("missing tenant id")
	ErrUnauthorized    = errors.New("unauthorized")
)

func RequireOwnerMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetClaimsFromContext(r.Context())
		if !ok || claims.Role != "owner" {
			RespondWithError(w, http.StatusForbidden, "FORBIDDEN", "Acceso denegado: Se requiere rol de Owner")
			return
		}
		next.ServeHTTP(w, r)
	}
}

func TenantIDFromRequest(r *http.Request) (string, error) {
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	claims, ok := GetClaimsFromContext(r.Context())
	if !ok {
		return "", errors.New("UNAUTHORIZED")
	}

	if tenantID == "" {
		if claims.Role != RoleOwner {
			tenantID = strings.TrimSpace(claims.TenantID)
		}
	}

	if tenantID == "" {
		return "", ErrMissingTenantID
	}

	if claims.Role != RoleOwner && tenantID != strings.TrimSpace(claims.TenantID) {
		return "", ErrForbiddenTenant
	}

	return tenantID, nil
}
