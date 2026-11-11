package operations

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"leguiburger/internal/auth"
	"net/http"
	"strings"
	"time"
)

type Principal struct {
	ID       string
	Kind     string
	Role     string
	BrandID  string
	TenantID string
}

func (p Principal) Platform() bool {
	return p.Kind == "user" && (p.Role == "owner" || p.Role == "super_admin")
}
func (p Principal) Manager() bool { return p.Platform() || (p.Kind == "user" && p.Role == "admin") }
func (p Principal) Cashier() bool {
	return p.Manager() || (p.Kind == "employee" && p.Role == "cashier")
}

type Scope struct {
	BrandID  string
	TenantID string
}

func validID(id string) bool     { parsed, err := uuid.Parse(id); return err == nil && parsed != uuid.Nil }
func optionalID(id *string) bool { return id == nil || validID(*id) }
func (h *Handler) principal(ctx context.Context, claims *auth.Claims) (Principal, error) {
	session, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		var err error
		session, err = auth.ResolveSessionPrincipal(ctx, h.repo.DB, claims)
		if err != nil {
			return Principal{}, unauthorized()
		}
	}
	return Principal{ID: session.ID, Kind: session.Kind, Role: session.Role, BrandID: session.BrandID, TenantID: session.TenantID}, nil
}

func (h *Handler) tenantScope(ctx context.Context, p Principal, r *http.Request) (Scope, error) {
	id := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if id == "" {
		id = r.URL.Query().Get("tenant_id")
	}
	if id == "" {
		id = p.TenantID
	}
	if !validID(id) {
		return Scope{}, badInput("Se requiere un X-Tenant-ID válido")
	}
	var scope Scope
	result := h.repo.DB.WithContext(ctx).Raw(`SELECT id::text AS tenant_id,brand_id::text AS brand_id FROM public.tenants WHERE id=? AND active`, id).Scan(&scope)
	if result.Error != nil {
		return scope, result.Error
	}
	if result.RowsAffected == 0 {
		return scope, notFound()
	}
	if !p.Platform() && !(p.Kind == "user" && p.Role == "admin" && p.BrandID == scope.BrandID) && !(p.Kind == "employee" && p.TenantID == scope.TenantID) {
		return Scope{}, forbidden()
	}
	return scope, nil
}
func (h *Handler) businessScope(ctx context.Context, p Principal, r *http.Request) (Scope, error) {
	if !p.Manager() {
		return Scope{}, forbidden()
	}
	if r.Header.Get("X-Tenant-ID") != "" || r.URL.Query().Get("tenant_id") != "" {
		return h.tenantScope(ctx, p, r)
	}
	brand := r.URL.Query().Get("brand_id")
	if brand == "" {
		brand = p.BrandID
	}
	if !validID(brand) {
		return Scope{}, badInput("Se requiere brand_id o X-Tenant-ID")
	}
	if !p.Platform() && brand != p.BrandID {
		return Scope{}, forbidden()
	}
	var n int64
	if err := h.repo.DB.WithContext(ctx).Raw(`SELECT count(*) FROM public.brands WHERE id=?`, brand).Scan(&n).Error; err != nil {
		return Scope{}, err
	}
	if n == 0 {
		return Scope{}, notFound()
	}
	return Scope{BrandID: brand}, nil
}
func readPeriod(r *http.Request, timezone string) (Period, error) {
	if timezone == "" {
		timezone = "America/Argentina/Buenos_Aires"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return Period{}, badInput("Zona horaria inválida")
	}
	now := time.Now().In(loc)
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	until := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
	if raw := r.URL.Query().Get("from"); raw != "" {
		from, err = time.ParseInLocation("2006-01-02", raw, loc)
		if err != nil {
			return Period{}, badInput("from debe tener formato AAAA-MM-DD")
		}
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		until, err = time.ParseInLocation("2006-01-02", raw, loc)
		if err != nil {
			return Period{}, badInput("to debe tener formato AAAA-MM-DD")
		}
		until = until.AddDate(0, 0, 1)
	}
	if !until.After(from) || until.Sub(from) > 367*24*time.Hour {
		return Period{}, badInput("El período debe ser válido y de hasta un año")
	}
	return Period{From: from, Until: until, Timezone: timezone}, nil
}
func lockPrincipal(tx *gorm.DB, p Principal, scope Scope) error {
	table := "public.users"
	if p.Kind == "employee" {
		table = "public.employees"
	}
	var id string
	query := fmt.Sprintf(`SELECT id::text FROM %s WHERE id=? AND role=? AND is_active FOR SHARE`, table)
	result := tx.Raw(query, p.ID, p.Role).Scan(&id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return unauthorized()
	}
	if scope.TenantID != "" {
		var tenant string
		result = tx.Raw(`SELECT id::text FROM public.tenants WHERE id=? AND brand_id=? AND active FOR SHARE`, scope.TenantID, scope.BrandID).Scan(&tenant)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return forbidden()
		}
	}
	// Recheck assignment under the same locks as the financial operation, not just stale JWT claims.
	if !p.Platform() {
		var allowed bool
		if p.Kind == "user" {
			if err := tx.Raw(`SELECT brand_id=?::uuid FROM public.users WHERE id=?`, scope.BrandID, p.ID).Scan(&allowed).Error; err != nil {
				return err
			}
		}
		if p.Kind == "employee" {
			if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM legui_private.employee_locations WHERE tenant_id=? AND employee_id=?)`, scope.TenantID, p.ID).Scan(&allowed).Error; err != nil {
				return err
			}
		}
		if !allowed {
			return forbidden()
		}
	}
	return nil
}
