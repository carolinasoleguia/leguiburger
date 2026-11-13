package auth

import (
	"bytes"
	"context"
	"gorm.io/gorm"
	"leguiburger/internal/db"
	"net/http"
	"strings"
)

func (p SessionPrincipal) Platform() bool { return p.Kind == "user" && isGlobalRole(p.Role) }
func (p SessionPrincipal) Manager() bool {
	return p.Platform() || p.Kind == "user" && p.Role == "admin" && p.BrandID != ""
}

// Administrative restaurant ownership does not grant counter or kitchen access.
func (p SessionPrincipal) Cashier() bool {
	return p.Platform() || p.Kind == "employee" && p.Role == "cashier"
}
func (p SessionPrincipal) Operator() bool {
	return p.Platform() || p.Kind == "employee" && (p.Role == "cashier" || p.Role == "employee" || p.Role == "kitchen")
}

// CanLegacy is the complete allowlist for the original API; new modules must opt in.
func CanLegacy(p SessionPrincipal, group, method string) bool {
	read := method == http.MethodGet
	if !read && method != http.MethodPost && method != http.MethodPut && method != http.MethodDelete {
		return false
	}
	switch group {
	case "brands", "users":
		return p.Platform()
	case "tenants":
		return read || p.Platform()
	case "employees":
		return p.Manager()
	case "products", "extras", "supplies", "shipping-methods", "tenant-products", "recipes":
		return read || p.Manager()
	case "customers":
		return p.Manager() || p.Kind == "employee" && p.Role == "cashier"
	case "production":
		return p.Manager() || p.Kind == "employee" && (p.Role == "employee" || p.Role == "kitchen") && method != http.MethodDelete
	}
	return false
}

// AuthorizeLegacy shares one transaction across all repositories for mutations,
// so actor attribution, permission locks, writes and the audit event commit together.
func AuthorizeLegacy(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 12<<20)
		p, ok := PrincipalFromContext(r.Context())
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if !ok || len(parts) < 2 || !CanLegacy(p, parts[1], r.Method) {
			RespondWithError(w, 403, "FORBIDDEN", "No tenés permiso para realizar esta operación")
			return
		}
		if r.Method == http.MethodGet {
			next(w, r)
			return
		}
		capture := &bufferedResponse{header: make(http.Header)}
		err := db.ForContext(r.Context()).Transaction(func(tx *gorm.DB) error {
			ctx := db.WithTransaction(r.Context(), tx)
			if err := LockSession(ctx, tx); err != nil {
				return err
			}
			user, employee := "", ""
			if p.Kind == "user" {
				user = p.ID
			} else {
				employee = p.ID
			}
			if err := tx.Exec(`SELECT set_config('leguiburger.actor_user_id',?,true),set_config('leguiburger.actor_employee_id',?,true)`, user, employee).Error; err != nil {
				return err
			}
			next(capture, r.WithContext(ctx))
			if capture.status >= 400 {
				return errHandlerRejected
			}
			target := ""
			if len(parts) > 2 {
				target = parts[2]
			}
			return Audit(ctx, tx, p, parts[1]+"."+strings.ToLower(r.Method), target, nil)
		})
		if err != nil && err != errHandlerRejected {
			RespondWithError(w, 503, "OPERATION_UNAVAILABLE", "No se pudo completar la operación")
			return
		}
		for k, values := range capture.header {
			w.Header()[k] = values
		}
		if capture.status == 0 {
			capture.status = 200
		}
		w.WriteHeader(capture.status)
		_, _ = w.Write(capture.body.Bytes())
	}
}

type handlerRejected struct{}

func (handlerRejected) Error() string { return "handler rejected mutation" }

var errHandlerRejected error = handlerRejected{}

type bufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (b *bufferedResponse) Header() http.Header { return b.header }
func (b *bufferedResponse) WriteHeader(status int) {
	if b.status == 0 {
		b.status = status
	}
}
func (b *bufferedResponse) Write(data []byte) (int, error) {
	if b.status == 0 {
		b.status = 200
	}
	return b.body.Write(data)
}

func BrandIDFromRequest(r *http.Request, explicit string) (string, error) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		// Direct handler unit tests use claims; production always resolves the DB principal.
		c, exists := GetClaimsFromContext(r.Context())
		if !exists {
			return "", ErrUnauthorized
		}
		p = SessionPrincipal{ID: c.UserID, Kind: "employee", Role: c.Role, TenantID: c.TenantID}
		if c.BrandID != nil {
			p.BrandID = strings.TrimSpace(*c.BrandID)
			p.Kind = "user"
		}
		if isGlobalRole(c.Role) {
			p.Kind = "user"
		}
	}
	requested := strings.TrimSpace(explicit)
	if requested == "" {
		requested = strings.TrimSpace(r.Header.Get("X-Brand-ID"))
	}
	if p.BrandID == "" && !p.Platform() {
		return "", ErrForbiddenTenant
	}
	if requested == "" {
		requested = p.BrandID
	}
	if requested == "" {
		return "", ErrMissingTenantID
	}
	if !p.Platform() && requested != p.BrandID {
		return "", ErrForbiddenTenant
	}
	return requested, nil
}

func Audit(ctx context.Context, tx *gorm.DB, p SessionPrincipal, event, target string, facts any) error {
	if facts == nil {
		facts = map[string]any{}
	}
	return tx.WithContext(ctx).Exec(`INSERT INTO legui_private.security_events(actor_kind,actor_id,brand_id,tenant_id,event_name,target_id,facts) VALUES(?,nullif(?,'')::uuid,nullif(?,'')::uuid,nullif(?,'')::uuid,?,?,?::jsonb)`, p.Kind, p.ID, p.BrandID, p.TenantID, event, target, jsonText(facts)).Error
}
