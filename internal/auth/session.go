package auth

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"os"
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

type identity struct {
	ID           string
	Kind         string
	Role         string
	BrandID      string
	TenantID     string
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
}

func (i identity) principal() SessionPrincipal {
	return SessionPrincipal{ID: i.ID, Kind: i.Kind, Role: i.Role, BrandID: i.BrandID, TenantID: i.TenantID}
}

const identitySQL = `SELECT u.id::text,'user' kind,u.role,coalesce(u.brand_id::text,'') brand_id,'' tenant_id,u.email,u.password_hash,u.first_name,u.last_name FROM public.users u WHERE u.id=? AND u.is_active
 UNION ALL SELECT e.id::text,'employee',e.role,coalesce(t.brand_id::text,''),coalesce(e.tenant_id::text,''),e.email,e.password_hash,e.first_name,e.last_name FROM public.employees e JOIN public.tenants t ON t.id=e.tenant_id AND t.active WHERE e.id=? AND e.is_active AND e.role IN ('employee','cashier','kitchen') LIMIT 1`

func loadIdentity(ctx context.Context, database *gorm.DB, id string) (identity, error) {
	var i identity
	parsed, err := uuid.Parse(id)
	if err != nil || parsed == uuid.Nil {
		return i, ErrInvalidCredentials
	}
	result := database.WithContext(ctx).Raw(identitySQL, id, id).Scan(&i)
	if result.Error != nil {
		return i, result.Error
	}
	if result.RowsAffected != 1 {
		return i, ErrInvalidCredentials
	}
	if i.Kind == "user" && !(isGlobalRole(i.Role) || i.Role == "admin" && i.BrandID != "") {
		return identity{}, ErrInvalidCredentials
	}
	return i, nil
}

type authSession struct {
	ID               string
	PrincipalKind    string
	PrincipalID      string
	Role             string
	BrandID          string
	TenantID         string
	CredentialDigest string
	MFAVerified      bool
}

func loadSession(ctx context.Context, database *gorm.DB, id string, lock bool) (authSession, error) {
	var s authSession
	if _, err := uuid.Parse(id); err != nil {
		return s, ErrInvalidCredentials
	}
	query := `SELECT id::text,principal_kind,principal_id::text,role,coalesce(brand_id::text,'') brand_id,coalesce(tenant_id::text,'') tenant_id,credential_digest,mfa_verified FROM legui_private.auth_sessions WHERE id=? AND revoked_at IS NULL AND expires_at>clock_timestamp() AND last_seen_at>clock_timestamp()-interval '30 minutes'`
	if lock {
		query += " FOR UPDATE"
	}
	result := database.WithContext(ctx).Raw(query, id).Scan(&s)
	if result.Error != nil {
		return s, result.Error
	}
	if result.RowsAffected != 1 {
		return s, ErrInvalidCredentials
	}
	return s, nil
}
func sessionIdentity(ctx context.Context, database *gorm.DB, s authSession) (identity, error) {
	i, err := loadIdentity(ctx, database, s.PrincipalID)
	if err != nil {
		return i, err
	}
	if i.Kind != s.PrincipalKind || i.Role != s.Role || i.BrandID != s.BrandID || i.TenantID != s.TenantID || digest(i.PasswordHash) != s.CredentialDigest {
		return identity{}, ErrInvalidCredentials
	}
	if i.principal().Platform() && PlatformMFARequired() && !s.MFAVerified {
		return identity{}, ErrInvalidCredentials
	}
	return i, nil
}

// Password login is the current product policy. MFA can be enabled explicitly
// later without changing the role/resource authorization or session validation.
func PlatformMFARequired() bool { return os.Getenv("AUTH_REQUIRE_PLATFORM_MFA") == "true" }
func ResolveSessionPrincipal(ctx context.Context, database *gorm.DB, claims *Claims) (SessionPrincipal, error) {
	if database == nil {
		return SessionPrincipal{}, fmt.Errorf("authentication database unavailable")
	}
	if claims == nil || claims.SessionID == "" || claims.Purpose != "access" || claims.Subject != claims.UserID {
		return SessionPrincipal{}, ErrInvalidCredentials
	}
	s, err := loadSession(ctx, database, claims.SessionID, false)
	if err != nil {
		return SessionPrincipal{}, err
	}
	i, err := sessionIdentity(ctx, database, s)
	if err != nil {
		return SessionPrincipal{}, err
	}
	brand := ""
	if claims.BrandID != nil {
		brand = strings.TrimSpace(*claims.BrandID)
	}
	if i.ID != claims.UserID || i.Role != claims.Role || i.BrandID != brand || i.TenantID != claims.TenantID || s.MFAVerified != claims.MFA {
		return SessionPrincipal{}, ErrInvalidCredentials
	}
	if err = database.WithContext(ctx).Exec(`UPDATE legui_private.auth_sessions SET last_seen_at=clock_timestamp() WHERE id=? AND last_seen_at<clock_timestamp()-interval '1 minute' AND revoked_at IS NULL`, s.ID).Error; err != nil {
		return SessionPrincipal{}, err
	}
	return i.principal(), nil
}
func LockSession(ctx context.Context, tx *gorm.DB) error {
	c, ok := GetClaimsFromContext(ctx)
	if !ok || c.SessionID == "" {
		return ErrInvalidCredentials
	}
	var id string
	result := tx.WithContext(ctx).Raw(`SELECT id::text FROM legui_private.auth_sessions WHERE id=? AND principal_id=? AND revoked_at IS NULL AND expires_at>clock_timestamp() AND last_seen_at>clock_timestamp()-interval '30 minutes' FOR SHARE`, c.SessionID, c.UserID).Scan(&id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrInvalidCredentials
	}
	return nil
}
