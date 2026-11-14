package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MembershipAccess struct {
	Tier                 string     `json:"tier"`
	State                string     `json:"state"`
	Operational          bool       `json:"operational"`
	Premium              bool       `json:"premium"`
	Locations            *int       `json:"locations"`
	TerminalsPerLocation *int       `json:"terminals_per_location"`
	ValidUntil           *time.Time `json:"valid_until"`
}

const membershipKey contextKey = "membership_access"
const membershipAccountKey contextKey = "membership_account_access"

func MembershipFromContext(ctx context.Context) (MembershipAccess, bool) {
	a, ok := ctx.Value(membershipKey).(MembershipAccess)
	return a, ok
}
func PremiumRestricted(ctx context.Context) bool {
	a, ok := MembershipFromContext(ctx)
	return ok && !a.Premium
}

// Commercial restrictions also apply to nested legacy and order detail DTOs.
func RedactMembershipData(ctx context.Context, data json.RawMessage) json.RawMessage {
	if !PremiumRestricted(ctx) || len(data) == 0 {
		return data
	}
	var value any
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if d.Decode(&value) != nil {
		return json.RawMessage(`{}`)
	}
	var clean func(any)
	clean = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, v := range x {
				switch strings.ToLower(k) {
				case "current_stock", "currentstock", "minimum_stock", "track_stock", "trackstock", "unit_cost", "current_unit_cost", "current_wholesale_cost", "frozen_cost_unit", "gross_margin", "operating_expenses", "estimated_operating_result", "missing_cost_lines":
					delete(x, k)
				default:
					clean(v)
				}
			}
		case []any:
			for _, v := range x {
				clean(v)
			}
		}
	}
	clean(value)
	out, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return out
}

func ReadMembership(ctx context.Context, database *gorm.DB, brand string) (MembershipAccess, error) {
	var a MembershipAccess
	q := database.WithContext(ctx).Raw(`SELECT * FROM legui_private.membership_access(?::uuid)`, brand).Scan(&a)
	return a, q.Error
}

// Account security and renewal remain available, including for expired staff sessions.
func membershipAccountPath(path string) bool {
	return path == "/api/membership" || strings.HasPrefix(path, "/api/auth/")
}

func premiumPath(path string) bool {
	for _, prefix := range []string{"/api/inventory", "/api/expenses", "/api/supplies", "/api/recipes", "/api/production", "/api/analytics/operations"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func (h *SecurityHandler) membership(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	if p.Platform() {
		respondWithJSON(w, 200, map[string]any{"platform": true})
		return nil
	}
	a, err := ReadMembership(r.Context(), h.database, p.BrandID)
	if err != nil {
		return err
	}
	respondWithJSON(w, 200, a)
	return nil
}

func (h *SecurityHandler) operatingTerminals(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	if !p.Manager() || p.BrandID == "" {
		return forbiddenAction()
	}
	var items []map[string]any
	err := h.database.WithContext(r.Context()).Raw(`SELECT s.id,s.tenant_id,coalesce(nullif(t.display_name,''),t.subdomain) location,e.first_name,e.last_name,s.user_agent,s.created_at,s.last_seen_at FROM legui_private.auth_sessions s JOIN public.tenants t ON t.id=s.tenant_id JOIN public.employees e ON e.id=s.principal_id WHERE s.principal_kind='employee' AND t.brand_id=? AND s.brand_id=? AND s.revoked_at IS NULL AND s.expires_at>clock_timestamp() AND s.last_seen_at>clock_timestamp()-interval '30 minutes' ORDER BY s.last_seen_at DESC LIMIT 100`, p.BrandID, p.BrandID).Scan(&items).Error
	if err != nil {
		return err
	}
	respondWithJSON(w, 200, map[string]any{"items": items})
	return nil
}

func (h *SecurityHandler) revokeOperatingTerminal(w http.ResponseWriter, r *http.Request) error {
	return h.hierarchyMutation(w, r, struct{}{}, func(tx *gorm.DB, p SessionPrincipal) (any, error) {
		if !p.Manager() || p.BrandID == "" {
			return nil, forbiddenAction()
		}
		id := r.PathValue("id")
		if parsed, err := uuid.Parse(id); err != nil || parsed == uuid.Nil {
			return nil, inputError("Terminal inválida")
		}
		q := tx.Exec(`UPDATE legui_private.auth_sessions s SET revoked_at=coalesce(s.revoked_at,clock_timestamp()) FROM public.tenants t WHERE s.id=? AND s.principal_kind='employee' AND t.id=s.tenant_id AND t.brand_id=? AND s.brand_id=?`, id, p.BrandID, p.BrandID)
		if q.Error != nil {
			return nil, q.Error
		}
		if q.RowsAffected != 1 {
			return nil, forbiddenAction()
		}
		if err := Audit(r.Context(), tx, p, "membership.terminal_released", id, nil); err != nil {
			return nil, err
		}
		return map[string]bool{"released": true}, nil
	})
}
