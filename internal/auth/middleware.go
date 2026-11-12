package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"leguiburger/internal/db"
	"leguiburger/internal/models"

	"gorm.io/gorm"
)

type contextKey string

const ClaimsKey contextKey = "user_claims"

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return AuthMiddlewareWithDB(db.DB, next)
}

func AuthMiddlewareWithDB(database *gorm.DB, next http.HandlerFunc) http.HandlerFunc {
	return authMiddleware(database, next, 25*time.Second)
}
func AuthStreamMiddlewareWithDB(database *gorm.DB, next http.HandlerFunc) http.HandlerFunc {
	return authMiddleware(database, next, 3*time.Minute)
}
func authMiddleware(database *gorm.DB, next http.HandlerFunc, duration time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			RespondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Se requiere token de autenticacion")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			RespondWithError(w, http.StatusUnauthorized, "INVALID_TOKEN_FORMAT", "Formato de token invalido")
			return
		}

		claims, err := ValidateToken(parts[1])
		if err != nil {
			RespondWithError(w, http.StatusUnauthorized, "INVALID_OR_EXPIRED_TOKEN", "Token invalido o expirado")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), duration)
		defer cancel()
		principal, err := ResolveSessionPrincipal(ctx, database, claims)
		if err != nil {
			if errors.Is(err, ErrInvalidCredentials) {
				RespondWithError(w, http.StatusUnauthorized, "SESSION_REVOKED", "La cuenta o sus permisos cambiaron; iniciá sesión nuevamente")
			} else {
				RespondWithError(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "No se pudo verificar la sesión")
			}
			return
		}
		ctx = context.WithValue(ctx, ClaimsKey, claims)
		ctx = context.WithValue(ctx, PrincipalKey, principal)
		allowed := principal.Kind == "user" && principal.Role == "admin" && BusinessPathAllowed(r.URL.Path)
		available, businessErr := businessAvailable(ctx, database, principal)
		if businessErr != nil {
			RespondWithError(w, 503, "AUTH_UNAVAILABLE", "No se pudo verificar el estado del restaurante")
			return
		}
		if !allowed && !available {
			RespondWithError(w, 403, "BUSINESS_SUSPENDED", "El restaurante está suspendido. Su dueño puede consultar la membresía y contactar a la administración.")
			return
		}
		ctx = context.WithValue(ctx, suspensionAccessKey, allowed)
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
		if !ok || !isGlobalRole(claims.Role) {
			RespondWithError(w, http.StatusForbidden, "FORBIDDEN", "Acceso denegado: Se requiere rol de Owner")
			return
		}
		next.ServeHTTP(w, r)
	}
}

func TenantIDFromRequest(r *http.Request) (string, error) {
	if p, ok := PrincipalFromContext(r.Context()); ok {
		id := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
		if id == "" {
			id = p.TenantID
		}
		if id == "" {
			return "", ErrMissingTenantID
		}
		if !p.Platform() && !p.Manager() && id != p.TenantID {
			return "", ErrForbiddenTenant
		}
		var brand string
		result := db.ForContext(r.Context()).Raw(`SELECT brand_id::text FROM public.tenants WHERE id=? AND active`, id).Scan(&brand)
		if result.Error != nil || result.RowsAffected != 1 {
			return "", ErrForbiddenTenant
		}
		if !p.Platform() && brand != p.BrandID {
			return "", ErrForbiddenTenant
		}
		return id, nil
	}
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	claims, ok := GetClaimsFromContext(r.Context())
	if !ok {
		return "", errors.New("UNAUTHORIZED")
	}

	if tenantID == "" {
		if !isGlobalRole(claims.Role) {
			tenantID = strings.TrimSpace(claims.TenantID)
		}
	}

	if tenantID == "" {
		return "", ErrMissingTenantID
	}

	if isGlobalRole(claims.Role) {
		return tenantID, nil
	}

	if strings.EqualFold(strings.TrimSpace(claims.Role), "admin") && claims.BrandID != nil {
		if tenantID == strings.TrimSpace(claims.TenantID) {
			return tenantID, nil
		}

		var tenant models.Tenant
		err := db.ForContext(r.Context()).First(&tenant, "id = ?", tenantID).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", ErrForbiddenTenant
			}
			return "", ErrForbiddenTenant
		}
		if strings.TrimSpace(tenant.BrandID) != strings.TrimSpace(*claims.BrandID) {
			return "", ErrForbiddenTenant
		}
		return tenantID, nil
	}

	if tenantID != strings.TrimSpace(claims.TenantID) {
		return "", ErrForbiddenTenant
	}

	return tenantID, nil
}
