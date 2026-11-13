package auth

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

func (h *SecurityHandler) locations(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	var rows []map[string]any
	e := h.database.WithContext(r.Context()).Raw(`SELECT t.id,t.brand_id,b.name business_name,t.subdomain,t.display_name,t.address,t.contact_phone,t.service_hours,t.active FROM public.tenants t JOIN public.brands b ON b.id=t.brand_id WHERE (? OR (?='user' AND t.brand_id=nullif(?,'')::uuid) OR (?='employee' AND t.active AND EXISTS(SELECT 1 FROM legui_private.employee_locations a WHERE a.employee_id=? AND a.tenant_id=t.id))) ORDER BY t.subdomain,t.id`, p.Platform(), p.Kind, p.BrandID, p.Kind, p.ID).Scan(&rows).Error
	if e != nil {
		return e
	}
	respondWithJSON(w, 200, map[string]any{"items": rows})
	return nil
}
func (h *SecurityHandler) saveLocation(w http.ResponseWriter, r *http.Request) error {
	if id := r.PathValue("id"); id != "" {
		if _, e := uuid.Parse(id); e != nil {
			return inputError("Local inválido")
		}
	}
	var in struct {
		BrandID   string `json:"brand_id"`
		Subdomain string `json:"subdomain"`
		Name      string `json:"display_name"`
		Address   string `json:"address"`
		Phone     string `json:"contact_phone"`
		Hours     string `json:"service_hours"`
		Active    bool   `json:"active"`
	}
	if e := readSecurityJSON(r, &in); e != nil {
		return e
	}
	if !branchName.MatchString(in.Subdomain) || len(in.Name) > 100 || len(in.Address) > 500 || len(in.Phone) > 30 || len(in.Hours) > 1000 {
		return inputError("Revisá los datos del local")
	}
	return h.hierarchyMutation(w, r, in, func(tx *gorm.DB, p SessionPrincipal) (any, error) {
		if !p.Manager() {
			return nil, forbiddenAction()
		}
		brand := p.BrandID
		if p.Platform() {
			brand = in.BrandID
		}
		if e := hierarchyBrand(tx, p, brand, false); e != nil {
			return nil, e
		}
		id := r.PathValue("id")
		if id == "" {
			id = uuid.NewString()
			e := tx.Exec(`INSERT INTO public.tenants(id,brand_id,subdomain,display_name,address,contact_phone,service_hours,active) VALUES(?,?,?,?,?,?,?,?)`, id, brand, in.Subdomain, in.Name, in.Address, in.Phone, in.Hours, in.Active).Error
			return map[string]string{"id": id}, e
		}
		q := tx.Exec(`UPDATE public.tenants SET subdomain=?,display_name=?,address=?,contact_phone=?,service_hours=?,active=? WHERE id=? AND brand_id=?`, in.Subdomain, in.Name, in.Address, in.Phone, in.Hours, in.Active, id, brand)
		if q.Error != nil {
			return nil, q.Error
		}
		if q.RowsAffected != 1 {
			return nil, forbiddenAction()
		}
		if !in.Active {
			if e := tx.Exec(`UPDATE legui_private.auth_sessions SET revoked_at=clock_timestamp() WHERE tenant_id=? AND principal_kind='employee' AND revoked_at IS NULL`, id).Error; e != nil {
				return nil, e
			}
		}
		return map[string]string{"id": id}, nil
	})
}
func (h *SecurityHandler) teamDirectory(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	if !p.Manager() {
		return forbiddenAction()
	}
	limit, offset, e := limitOffset(r)
	if e != nil {
		return e
	}
	var payload string
	e = h.database.WithContext(r.Context()).Raw(`SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb))::text FROM(SELECT e.id,e.first_name,e.last_name,e.email,e.phone,e.role,e.is_active,e.tenant_id,t.brand_id,
 coalesce((SELECT jsonb_agg(a.tenant_id ORDER BY a.tenant_id) FROM legui_private.employee_locations a WHERE a.employee_id=e.id),'[]'::jsonb) location_ids
 FROM public.employees e JOIN public.tenants t ON t.id=e.tenant_id WHERE (? OR t.brand_id=nullif(?,'')::uuid) ORDER BY e.created_at DESC,e.id LIMIT ? OFFSET ?)v`, p.Platform(), p.BrandID, limit, offset).Scan(&payload).Error
	if e != nil {
		return e
	}
	_, _ = w.Write([]byte(payload))
	return nil
}
func (h *SecurityHandler) updateTeamMember(w http.ResponseWriter, r *http.Request) error {
	if _, e := uuid.Parse(r.PathValue("id")); e != nil {
		return inputError("Colaborador inválido")
	}
	var in struct {
		First     string   `json:"first_name"`
		Last      string   `json:"last_name"`
		Role      string   `json:"role"`
		Active    bool     `json:"is_active"`
		Locations []string `json:"location_ids"`
		Home      string   `json:"tenant_id"`
	}
	if e := readSecurityJSON(r, &in); e != nil {
		return e
	}
	if !collaboratorRole(in.Role) || strings.TrimSpace(in.First) == "" || strings.TrimSpace(in.Last) == "" || len(in.First) > 50 || len(in.Last) > 50 || len(in.Locations) < 1 || len(in.Locations) > 1000 {
		return inputError("Revisá nombre, rol y locales del colaborador")
	}
	return h.hierarchyMutation(w, r, in, func(tx *gorm.DB, p SessionPrincipal) (any, error) {
		if !p.Manager() {
			return nil, forbiddenAction()
		}
		id := r.PathValue("id")
		var brand string
		q := tx.Raw(`SELECT t.brand_id::text FROM public.employees e JOIN public.tenants t ON t.id=e.tenant_id WHERE e.id=? FOR UPDATE OF e`, id).Scan(&brand)
		if q.Error != nil {
			return nil, q.Error
		}
		if q.RowsAffected != 1 {
			return nil, forbiddenAction()
		}
		if e := hierarchyBrand(tx, p, brand, false); e != nil {
			return nil, e
		}
		seen := map[string]bool{}
		for _, tenant := range in.Locations {
			if _, e := uuid.Parse(tenant); e != nil || seen[tenant] {
				return nil, inputError("Locales inválidos o repetidos")
			}
			seen[tenant] = true
			var allowed bool
			if e := tx.Raw(`SELECT EXISTS(SELECT 1 FROM public.tenants WHERE id=? AND brand_id=?)`, tenant, brand).Scan(&allowed).Error; e != nil {
				return nil, e
			}
			if !allowed {
				return nil, forbiddenAction()
			}
		}
		if !seen[in.Home] {
			return nil, inputError("El local principal debe estar entre los asignados")
		}
		if e := tx.Exec(`UPDATE public.employees SET first_name=?,last_name=?,role=?,is_active=?,tenant_id=? WHERE id=?`, in.First, in.Last, in.Role, in.Active, in.Home, id).Error; e != nil {
			return nil, e
		}
		if e := tx.Exec(`DELETE FROM legui_private.employee_locations WHERE employee_id=? AND tenant_id NOT IN ?`, id, in.Locations).Error; e != nil {
			return nil, e
		}
		for _, tenant := range in.Locations {
			if e := tx.Exec(`INSERT INTO legui_private.employee_locations(employee_id,tenant_id) VALUES(?,?) ON CONFLICT DO NOTHING`, id, tenant).Error; e != nil {
				return nil, e
			}
		}
		return map[string]bool{"ok": true}, nil
	})
}
func (h *SecurityHandler) switchLocation(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	if p.Kind != "employee" {
		return forbiddenAction()
	}
	var in struct {
		TenantID string `json:"tenant_id"`
	}
	if e := readSecurityJSON(r, &in); e != nil {
		return e
	}
	if _, e := uuid.Parse(in.TenantID); e != nil {
		return inputError("Local inválido")
	}
	var output map[string]any
	var refresh string
	e := h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := LockSession(r.Context(), tx); e != nil {
			return e
		}
		i, e := loadIdentity(r.Context(), tx, p.ID)
		if e != nil {
			return e
		}
		var brand string
		q := tx.Raw(`SELECT t.brand_id::text FROM legui_private.employee_locations a JOIN public.tenants t ON t.id=a.tenant_id AND t.active JOIN public.brands b ON b.id=t.brand_id AND b.suspended_at IS NULL WHERE a.employee_id=? AND a.tenant_id=? FOR SHARE OF a,t`, p.ID, in.TenantID).Scan(&brand)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected != 1 || brand != p.BrandID {
			return forbiddenAction()
		}
		i.TenantID = in.TenantID
		i.BrandID = brand
		sid := uuid.NewString()
		c, _ := GetClaimsFromContext(r.Context())
		if e = tx.Exec(`UPDATE legui_private.auth_sessions SET revoked_at=clock_timestamp() WHERE id=?`, c.SessionID).Error; e != nil {
			return e
		}
		if e = tx.Exec(`INSERT INTO legui_private.auth_sessions(id,principal_kind,principal_id,role,brand_id,tenant_id,credential_digest,user_agent) VALUES(?,'employee',?,?,?,?,?,?)`, sid, p.ID, i.Role, i.BrandID, i.TenantID, digest(i.PasswordHash), truncate(r.UserAgent(), 200)).Error; e != nil {
			return e
		}
		output, refresh, e = issueSession(tx, i, sid, false)
		return e
	})
	if e != nil {
		return e
	}
	setRefreshCookie(w, refresh)
	respondWithJSON(w, 200, output)
	return nil
}
