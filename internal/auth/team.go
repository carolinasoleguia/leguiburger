package auth

import (
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func managerTenant(tx *gorm.DB, p SessionPrincipal, tenant string) (string, error) {
	if !p.Manager() {
		return "", forbiddenAction()
	}
	if _, err := uuid.Parse(tenant); err != nil {
		return "", inputError("Elegí un local válido")
	}
	var brand string
	result := tx.Raw(`SELECT brand_id::text FROM public.tenants WHERE id=? AND active FOR SHARE`, tenant).Scan(&brand)
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected != 1 || !p.Platform() && p.BrandID != brand {
		return "", forbiddenAction()
	}
	return brand, nil
}
func (h *SecurityHandler) invite(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	if !p.Manager() {
		return forbiddenAction()
	}
	if err := h.rate(r, "team-invite", p.ID, 30); err != nil {
		return err
	}
	var input struct {
		Email    string `json:"email"`
		TenantID string `json:"tenant_id"`
		Role     string `json:"role"`
	}
	if err := readSecurityJSON(r, &input); err != nil {
		return err
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if !validEmail(input.Email) || !collaboratorRole(input.Role) {
		return inputError("Ingresá un correo y un rol de colaborador válidos")
	}
	site, err := publicSite()
	if err != nil {
		return err
	}
	raw := randomToken()
	id := uuid.NewString()
	err = h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := LockSession(r.Context(), tx); e != nil {
			return e
		}
		brand, e := managerTenant(tx, p, input.TenantID)
		if e != nil {
			return e
		}
		var exists bool
		if e = tx.Raw(`SELECT EXISTS(SELECT 1 FROM public.employees WHERE lower(email)=? UNION ALL SELECT 1 FROM public.users WHERE lower(email)=?)`, input.Email, input.Email).Scan(&exists).Error; e != nil {
			return e
		}
		if exists {
			return &securityError{409, "ACCOUNT_EXISTS", "Ese correo ya tiene una cuenta. Administrá sus permisos desde Equipo"}
		}
		if e = tx.Exec(`UPDATE legui_private.auth_action_tokens SET used_at=clock_timestamp() WHERE purpose='team_invitation' AND tenant_id=? AND email=? AND used_at IS NULL`, input.TenantID, input.Email).Error; e != nil {
			return e
		}
		if e = tx.Exec(`INSERT INTO legui_private.auth_action_tokens(id,digest,purpose,email,tenant_id,role,invited_by,expires_at) VALUES(?,?,'team_invitation',?,?,?,?,clock_timestamp()+interval '48 hours')`, id, digest(raw), input.Email, input.TenantID, input.Role, p.ID).Error; e != nil {
			return e
		}
		if e = enqueueMail(tx, input.Email, "Te invitaron al equipo de LeguiBurger", "Te invitaron a trabajar con tu equipo. Elegí tu contraseña desde este enlace, válido por 48 horas:\n\n"+site+"/accept-invitation#token="+url.QueryEscape(raw), time.Now().Add(48*time.Hour)); e != nil {
			return e
		}
		actor := p
		actor.BrandID = brand
		actor.TenantID = input.TenantID
		return Audit(r.Context(), tx, actor, "team.invited", id, map[string]any{"role": input.Role})
	})
	if err != nil {
		return err
	}
	respondWithJSON(w, 201, map[string]any{"id": id, "invitation_url": site + "/accept-invitation#token=" + url.QueryEscape(raw), "message": "Invitación creada. También podés compartir el enlace directamente."})
	return nil
}
func collaboratorRole(role string) bool {
	return role == "employee" || role == "cashier" || role == "kitchen"
}
func (h *SecurityHandler) invitations(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	if !p.Manager() {
		return forbiddenAction()
	}
	limit, offset, err := limitOffset(r)
	if err != nil {
		return err
	}
	var rows []map[string]any
	err = h.database.WithContext(r.Context()).Raw(`SELECT a.id,a.email,a.role,a.tenant_id,t.subdomain,a.created_at,a.expires_at,a.used_at FROM legui_private.auth_action_tokens a JOIN public.tenants t ON t.id=a.tenant_id WHERE a.purpose='team_invitation' AND (? OR t.brand_id=nullif(?,'')::uuid) ORDER BY a.created_at DESC LIMIT ? OFFSET ?`, p.Platform(), p.BrandID, limit, offset).Scan(&rows).Error
	if err != nil {
		return err
	}
	respondWithJSON(w, 200, rows)
	return nil
}
func (h *SecurityHandler) revokeInvitation(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	if !p.Manager() {
		return forbiddenAction()
	}
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		return inputError("Invitación inválida")
	}
	err := h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		var tenant string
		result := tx.Raw(`SELECT tenant_id::text FROM legui_private.auth_action_tokens WHERE id=? AND purpose='team_invitation' FOR UPDATE`, id).Scan(&tenant)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return invalidAction()
		}
		brand, e := managerTenant(tx, p, tenant)
		if e != nil {
			return e
		}
		if e = tx.Exec(`UPDATE legui_private.auth_action_tokens SET used_at=clock_timestamp() WHERE id=? AND used_at IS NULL`, id).Error; e != nil {
			return e
		}
		actor := p
		actor.BrandID = brand
		actor.TenantID = tenant
		return Audit(r.Context(), tx, actor, "team.invitation_revoked", id, nil)
	})
	if err != nil {
		return err
	}
	respondWithJSON(w, 200, map[string]bool{"revoked": true})
	return nil
}
func (h *SecurityHandler) acceptInvitation(w http.ResponseWriter, r *http.Request) error {
	if err := h.ipRate(r, "team-accept-ip", 20); err != nil {
		return err
	}
	var input struct {
		Token     string `json:"token"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Password  string `json:"password"`
	}
	if err := readSecurityJSON(r, &input); err != nil {
		return err
	}
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	if input.FirstName == "" || input.LastName == "" || len(input.FirstName) > 50 || len(input.LastName) > 50 || !validPassword(input.Password) {
		return inputError("Ingresá nombre, apellido y una contraseña de 12 a 72 bytes")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		action, e := actionByToken(tx, input.Token, "team_invitation")
		if e != nil {
			return e
		}
		inviter, e := loadIdentity(r.Context(), tx, action.InvitedBy)
		if e != nil {
			return invalidAction()
		}
		if e = lockIdentity(tx, inviter); e != nil {
			return e
		}
		inviter, e = loadIdentity(r.Context(), tx, action.InvitedBy)
		if e != nil {
			return invalidAction()
		}
		brand, e := managerTenant(tx, inviter.principal(), action.TenantID)
		if e != nil {
			return invalidAction()
		}
		if !collaboratorRole(action.Role) {
			return invalidAction()
		}
		var exists bool
		if e = tx.Raw(`SELECT EXISTS(SELECT 1 FROM public.users WHERE lower(email)=? UNION ALL SELECT 1 FROM public.employees WHERE lower(email)=?)`, action.Email, action.Email).Scan(&exists).Error; e != nil {
			return e
		}
		if exists {
			return &securityError{409, "ACCOUNT_EXISTS", "Este correo ya tiene una cuenta. Iniciá sesión o pedí una nueva invitación"}
		}
		id := uuid.NewString()
		if e = tx.Exec(`INSERT INTO public.employees(id,tenant_id,first_name,last_name,email,password_hash,role,is_active) VALUES(?,?,?,?,?,?,?,true)`, id, action.TenantID, input.FirstName, input.LastName, action.Email, string(hashed), action.Role).Error; e != nil {
			return e
		}
		if e = tx.Exec(`UPDATE legui_private.auth_action_tokens SET used_at=clock_timestamp() WHERE digest=?`, action.Digest).Error; e != nil {
			return e
		}
		return Audit(r.Context(), tx, SessionPrincipal{ID: id, Kind: "employee", Role: action.Role, BrandID: brand, TenantID: action.TenantID}, "team.invitation_accepted", id, nil)
	})
	if err != nil {
		return err
	}
	respondWithJSON(w, 201, map[string]string{"message": "Tu cuenta está lista. Ya podés iniciar sesión."})
	return nil
}
func (h *SecurityHandler) audit(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	if !p.Manager() {
		return forbiddenAction()
	}
	limit, offset, err := limitOffset(r)
	if err != nil {
		return err
	}
	tenant := r.URL.Query().Get("tenant_id")
	if tenant != "" {
		if _, err = managerTenant(h.database.WithContext(r.Context()), p, tenant); err != nil {
			return err
		}
	}
	var rows []map[string]any
	err = h.database.WithContext(r.Context()).Raw(`WITH events AS (
 SELECT id,occurred_at,event_name,actor_kind,actor_id,brand_id,tenant_id,target_id,facts FROM legui_private.security_events
 UNION ALL SELECT id,occurred_at,event_name,CASE WHEN actor_user_id IS NOT NULL THEN 'user' ELSE 'employee' END,coalesce(actor_user_id,actor_employee_id),brand_id,tenant_id,facts->>'entity_id',facts FROM public.activity_events
 UNION ALL SELECT id,occurred_at,event_name,'user',actor_user_id,target_brand_id,NULL::uuid,NULL::text,facts FROM legui_private.platform_admin_events WHERE ?)
 SELECT e.id,e.occurred_at,e.event_name,e.actor_kind,e.actor_id,e.brand_id,e.tenant_id,e.target_id,e.facts,coalesce(u.first_name||' '||u.last_name,em.first_name||' '||em.last_name,'Sistema') AS actor_name
 FROM events e LEFT JOIN public.users u ON e.actor_kind='user' AND u.id=e.actor_id LEFT JOIN public.employees em ON e.actor_kind='employee' AND em.id=e.actor_id
 WHERE (? OR e.brand_id=nullif(?,'')::uuid) AND (?='' OR e.tenant_id=nullif(?,'')::uuid) ORDER BY e.occurred_at DESC,e.id DESC LIMIT ? OFFSET ?`, p.Platform(), p.Platform(), p.BrandID, tenant, tenant, limit, offset).Scan(&rows).Error
	if err != nil {
		return err
	}
	respondWithJSON(w, 200, map[string]any{"items": rows, "limit": limit, "offset": offset})
	return nil
}
