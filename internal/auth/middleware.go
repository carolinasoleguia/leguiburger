package auth

import (
	"context"
	"encoding/json"
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
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			RespondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Se requiere token de autenticacion")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			RespondWithError(w, http.StatusUnauthorized, "INVALID_TOKEN_FORMAT", "Formato de token invalido (se esperaba Bearer <token>)")
			return
		}

		claims, err := ValidateToken(parts[1])
		if err != nil {
			RespondWithError(w, http.StatusUnauthorized, "INVALID_OR_EXPIRED_TOKEN", "Token invalido o expirado")
			return
		}

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
