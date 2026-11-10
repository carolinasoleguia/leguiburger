package auth

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"strings"
)

type SessionPrincipal struct {
	ID       string
	Kind     string
	Role     string
	BrandID  string
	TenantID string
}

const PrincipalKey contextKey = "session_principal"

func PrincipalFromContext(ctx context.Context) (SessionPrincipal, bool) {
	p, ok := ctx.Value(PrincipalKey).(SessionPrincipal)
	return p, ok
}
func ResolveSessionPrincipal(ctx context.Context, database *gorm.DB, claims *Claims) (SessionPrincipal, error) {
	var p SessionPrincipal
	if database == nil {
		return p, fmt.Errorf("authentication database unavailable")
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil || id == uuid.Nil {
		return p, ErrInvalidCredentials
	}
	result := database.WithContext(ctx).Raw(`SELECT id::text,'user' AS kind,role,coalesce(brand_id::text,'') brand_id,'' AS tenant_id FROM public.users WHERE id=? AND is_active AND lower(email)=lower(?)
 UNION ALL SELECT e.id::text,'employee',e.role,coalesce(t.brand_id::text,''),coalesce(e.tenant_id::text,'') FROM public.employees e LEFT JOIN public.tenants t ON t.id=e.tenant_id WHERE e.id=? AND e.is_active AND lower(e.email)=lower(?) LIMIT 1`, claims.UserID, claims.Email, claims.UserID, claims.Email).Scan(&p)
	if result.Error != nil {
		return p, result.Error
	}
	if result.RowsAffected == 0 || p.Role != claims.Role {
		return p, ErrInvalidCredentials
	}
	if !isGlobalRole(p.Role) {
		if p.Kind == "user" && p.Role == "admin" && (claims.BrandID == nil || p.BrandID != strings.TrimSpace(*claims.BrandID)) {
			return p, ErrInvalidCredentials
		}
		if p.Kind == "employee" && p.TenantID != strings.TrimSpace(claims.TenantID) {
			return p, ErrInvalidCredentials
		}
	}
	return p, nil
}
