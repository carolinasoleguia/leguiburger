package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var branchName = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

// All hierarchy mutations serialize by actor/key, validate the live session and
// commit their response with the write so a lost reply cannot create a second business.
func (h *SecurityHandler) hierarchyMutation(w http.ResponseWriter, r *http.Request, input any, fn func(*gorm.DB, SessionPrincipal) (any, error)) error {
	p, _ := PrincipalFromContext(r.Context())
	key := r.Header.Get("Idempotency-Key")
	if !regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,160}$`).MatchString(key) {
		return inputError("Se requiere Idempotency-Key")
	}
	raw, _ := json.Marshal(input)
	sum := sha256.Sum256(append([]byte(r.Method+" "+r.URL.Path+"\n"), raw...))
	hash := hex.EncodeToString(sum[:])
	var output json.RawMessage
	e := h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := LockSession(r.Context(), tx); e != nil {
			return e
		}
		if e := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "hierarchy:"+p.ID+":"+key).Error; e != nil {
			return e
		}
		var receipt struct{ Digest, Response string }
		q := tx.Raw(`SELECT digest,response::text FROM legui_private.hierarchy_receipts WHERE actor_id=? AND operation_key=?`, p.ID, key).Scan(&receipt)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected > 0 {
			if receipt.Digest != hash {
				return inputError("Esa clave ya se usó con otros datos")
			}
			output = json.RawMessage(receipt.Response)
			return nil
		}
		result, e := fn(tx, p)
		if e != nil {
			return e
		}
		output, e = json.Marshal(result)
		if e != nil {
			return e
		}
		if e = tx.Exec(`INSERT INTO legui_private.hierarchy_receipts(actor_id,operation_key,digest,response) VALUES(?,?,?,?::jsonb)`, p.ID, key, hash, string(output)).Error; e != nil {
			return e
		}
		return Audit(r.Context(), tx, p, "business."+r.Method, r.PathValue("id"), nil)
	})
	if e != nil {
		return businessError(e)
	}
	w.WriteHeader(200)
	_, _ = w.Write(output)
	return nil
}
func businessError(e error) error {
	var pg *pgconn.PgError
	if errors.As(e, &pg) {
		if pg.Code == "23514" {
			return inputError("Revisá los límites del plan, la asignación al negocio y el responsable principal")
		}
		if pg.Code == "23505" {
			return inputError("El correo o el local ya están registrados")
		}
		if pg.Code == "23503" {
			return inputError("La relación con el negocio no es válida")
		}
	}
	return e
}
func hierarchyBrand(tx *gorm.DB, p SessionPrincipal, id string, allowSuspended bool) error {
	if _, e := uuid.Parse(id); e != nil {
		return inputError("Negocio inválido")
	}
	if !p.Platform() && (!p.Manager() || id != p.BrandID) {
		return forbiddenAction()
	}
	var active bool
	q := tx.Raw(`SELECT suspended_at IS NULL FROM public.brands WHERE id=? FOR SHARE`, id).Scan(&active)
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected != 1 || !active && !allowSuspended {
		return forbiddenAction()
	}
	return nil
}
func ownerInvitation(tx *gorm.DB, p SessionPrincipal, brand, email, purpose string) (map[string]string, error) {
	site, e := publicSite()
	if e != nil {
		return nil, e
	}
	token := randomToken()
	id := uuid.NewString()
	if e = tx.Exec(`UPDATE legui_private.business_invitations SET revoked_at=clock_timestamp() WHERE brand_id=? AND used_at IS NULL AND revoked_at IS NULL`, brand).Error; e != nil {
		return nil, e
	}
	if e = tx.Exec(`INSERT INTO legui_private.business_invitations(id,digest,brand_id,email,purpose,previous_owner_id,invited_by) SELECT ?,?,?,?, ?,primary_owner_id,? FROM public.brands WHERE id=?`, id, digest(token), brand, email, purpose, p.ID, brand).Error; e != nil {
		return nil, e
	}
	link := site + "/accept-business-invitation#token=" + url.QueryEscape(token)
	if e = enqueueMail(tx, email, "Tu restaurante en LeguiBurger", "Te invitaron a administrar tu restaurante. Abrí este enlace y elegí tu acceso. Vence en 48 horas. Si ya tenés cuenta en este negocio, confirmá con tu contraseña actual.\n\n"+link, time.Now().Add(48*time.Hour)); e != nil {
		return nil, e
	}
	return map[string]string{"id": id, "invitation_url": link}, nil
}
func (h *SecurityHandler) createRestaurant(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	if !p.Platform() {
		return forbiddenAction()
	}
	var in struct {
		Name          string `json:"name"`
		TaxID         string `json:"tax_id"`
		Local         string `json:"local"`
		Email         string `json:"email"`
		Locations     int    `json:"base_location_limit"`
		Collaborators int    `json:"base_collaborator_limit"`
	}
	if e := readSecurityJSON(r, &in); e != nil {
		return e
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.Locations == 0 {
		in.Locations = 1
	}
	if in.Collaborators == 0 {
		in.Collaborators = 5
	}
	if len(in.Name) < 2 || len(in.Name) > 100 || len(in.TaxID) > 20 || !branchName.MatchString(in.Local) || !validEmail(in.Email) || in.Locations < 1 || in.Locations > 1000 || in.Collaborators < 1 || in.Collaborators > 10000 {
		return inputError("Revisá nombre, correo, local y límites del negocio")
	}
	return h.hierarchyMutation(w, r, in, func(tx *gorm.DB, p SessionPrincipal) (any, error) {
		var count int64
		if e := tx.Raw(`SELECT (SELECT count(*) FROM public.users WHERE lower(email)=?)+(SELECT count(*) FROM public.employees WHERE lower(email)=?)`, in.Email, in.Email).Scan(&count).Error; e != nil {
			return nil, e
		}
		if count > 0 {
			return nil, inputError("Ese correo ya tiene una cuenta; usá una invitación para el negocio existente")
		}
		brand, tenant := uuid.NewString(), uuid.NewString()
		if e := tx.Exec(`INSERT INTO public.brands(id,name,tax_id,base_location_limit,base_collaborator_limit) VALUES(?,?,?,?,?)`, brand, in.Name, in.TaxID, in.Locations, in.Collaborators).Error; e != nil {
			return nil, e
		}
		if e := tx.Exec(`INSERT INTO public.tenants(id,brand_id,subdomain,display_name,active) VALUES(?,?,?, ?,true)`, tenant, brand, in.Local, in.Local).Error; e != nil {
			return nil, e
		}
		invite, e := ownerInvitation(tx, p, brand, in.Email, "onboarding")
		if e != nil {
			return nil, e
		}
		return map[string]any{"brand_id": brand, "tenant_id": tenant, "invitation": invite}, nil
	})
}
func (h *SecurityHandler) businesses(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	if !p.Manager() || strings.HasPrefix(r.URL.Path, "/api/platform/") && !p.Platform() {
		return forbiddenAction()
	}
	limit, offset, e := limitOffset(r)
	if e != nil {
		return e
	}
	var payload string
	e = h.database.WithContext(r.Context()).Raw(`SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb))::text FROM(
 SELECT b.id,b.name,b.tax_id,b.primary_owner_id,u.email owner_email,b.closed_at,b.suspended_at,b.suspension_reason,b.base_location_limit,b.base_collaborator_limit,c.locations maximum_locations,c.collaborators maximum_collaborators,c.source capacity_source,
 (SELECT count(*) FROM public.tenants WHERE brand_id=b.id AND active) locations,
 (SELECT count(*) FROM public.employees e JOIN public.tenants t ON t.id=e.tenant_id WHERE t.brand_id=b.id AND e.is_active) collaborators,
 coalesce((SELECT jsonb_agg(jsonb_build_object('id',x.id,'email',x.email,'purpose',x.purpose,'expires_at',x.expires_at)) FROM legui_private.business_invitations x WHERE x.brand_id=b.id AND x.used_at IS NULL AND x.revoked_at IS NULL AND x.expires_at>now()),'[]'::jsonb) invitations
 FROM public.brands b LEFT JOIN public.users u ON u.id=b.primary_owner_id CROSS JOIN LATERAL legui_private.business_capacity(b.id)c WHERE (? OR b.id=nullif(?,'')::uuid) ORDER BY b.created_at DESC,b.id LIMIT ? OFFSET ?)v`, p.Platform(), p.BrandID, limit, offset).Scan(&payload).Error
	if e != nil {
		return e
	}
	_, _ = w.Write([]byte(payload))
	return nil
}
func (h *SecurityHandler) businessInvitation(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Email           string `json:"email"`
		CurrentPassword string `json:"current_password"`
	}
	if e := readSecurityJSON(r, &in); e != nil {
		return e
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if !validEmail(in.Email) {
		return inputError("Correo inválido")
	}
	return h.hierarchyMutation(w, r, in, func(tx *gorm.DB, p SessionPrincipal) (any, error) {
		brand := r.PathValue("id")
		if e := hierarchyBrand(tx, p, brand, false); e != nil {
			return nil, e
		}
		var owner string
		if e := tx.Raw(`SELECT coalesce(primary_owner_id::text,'') FROM public.brands WHERE id=? FOR UPDATE`, brand).Scan(&owner).Error; e != nil {
			return nil, e
		}
		if !p.Platform() {
			if p.ID != owner {
				return nil, forbiddenAction()
			}
			i, e := loadIdentity(r.Context(), tx, p.ID)
			if e != nil {
				return nil, e
			}
			if bcrypt.CompareHashAndPassword([]byte(i.PasswordHash), []byte(in.CurrentPassword)) != nil {
				return nil, ErrInvalidCredentials
			}
		}
		var elsewhere int64
		if e := tx.Raw(`SELECT (SELECT count(*) FROM public.users WHERE lower(email)=? AND (brand_id IS DISTINCT FROM ?::uuid OR role<>'admin'))+(SELECT count(*) FROM public.employees WHERE lower(email)=?)`, in.Email, brand, in.Email).Scan(&elsewhere).Error; e != nil {
			return nil, e
		}
		if elsewhere > 0 {
			return nil, inputError("Ese correo no puede administrar este negocio")
		}
		purpose := "onboarding"
		if owner != "" {
			purpose = "transfer"
		}
		return ownerInvitation(tx, p, brand, in.Email, purpose)
	})
}
func (h *SecurityHandler) businessStatus(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	if !p.Platform() {
		return forbiddenAction()
	}
	var in struct {
		Suspended     bool   `json:"suspended"`
		Reason        string `json:"reason"`
		Locations     int    `json:"base_location_limit"`
		Collaborators int    `json:"base_collaborator_limit"`
	}
	if e := readSecurityJSON(r, &in); e != nil {
		return e
	}
	if len(in.Reason) > 500 || in.Suspended && len(strings.TrimSpace(in.Reason)) < 3 || in.Locations < 1 || in.Collaborators < 1 || in.Locations > 1000 || in.Collaborators > 10000 {
		return inputError("Revisá el motivo y los límites")
	}
	return h.hierarchyMutation(w, r, in, func(tx *gorm.DB, p SessionPrincipal) (any, error) {
		id := r.PathValue("id")
		if e := hierarchyBrand(tx, p, id, true); e != nil {
			return nil, e
		}
		e := tx.Exec(`UPDATE public.brands SET suspended_at=CASE WHEN ? THEN coalesce(suspended_at,clock_timestamp()) ELSE NULL END,suspension_reason=?,base_location_limit=?,base_collaborator_limit=? WHERE id=?`, in.Suspended, in.Reason, in.Locations, in.Collaborators, id).Error
		return map[string]bool{"ok": true}, e
	})
}
func (h *SecurityHandler) revokeBusinessInvitation(w http.ResponseWriter, r *http.Request) error {
	return h.hierarchyMutation(w, r, struct{}{}, func(tx *gorm.DB, p SessionPrincipal) (any, error) {
		brand := r.PathValue("id")
		if e := hierarchyBrand(tx, p, brand, false); e != nil {
			return nil, e
		}
		var owner string
		if e := tx.Raw(`SELECT coalesce(primary_owner_id::text,'') FROM public.brands WHERE id=?`, brand).Scan(&owner).Error; e != nil {
			return nil, e
		}
		if !p.Platform() && p.ID != owner {
			return nil, forbiddenAction()
		}
		e := tx.Exec(`UPDATE legui_private.business_invitations SET revoked_at=clock_timestamp() WHERE brand_id=? AND revoked_at IS NULL AND used_at IS NULL`, brand).Error
		return map[string]bool{"revoked": true}, e
	})
}
func (h *SecurityHandler) acceptBusinessInvitation(w http.ResponseWriter, r *http.Request) error {
	if e := h.ipRate(r, "business-accept", 20); e != nil {
		return e
	}
	var in struct {
		Token           string `json:"token"`
		First           string `json:"first_name"`
		Last            string `json:"last_name"`
		Password        string `json:"password"`
		CurrentPassword string `json:"current_password"`
	}
	if e := readSecurityJSON(r, &in); e != nil {
		return e
	}
	if len(in.Token) < 32 || len(in.Token) > 200 || strings.TrimSpace(in.First) == "" || strings.TrimSpace(in.Last) == "" || len(in.First) > 50 || len(in.Last) > 50 || (in.CurrentPassword == "" && !validPassword(in.Password)) {
		return inputError("Revisá el enlace, nombre y contraseña")
	}
	var hashed []byte
	var e error
	if validPassword(in.Password) {
		hashed, e = bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if e != nil {
			return e
		}
	}
	e = h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		var invite struct{ ID, BrandID, Email, Purpose, PreviousOwnerID, InvitedBy string }
		q := tx.Raw(`SELECT id::text,brand_id::text,email,purpose,coalesce(previous_owner_id::text,'') previous_owner_id,invited_by::text FROM legui_private.business_invitations WHERE digest=? AND used_at IS NULL AND revoked_at IS NULL AND expires_at>now() FOR UPDATE`, digest(in.Token)).Scan(&invite)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected != 1 {
			return invalidAction()
		}
		inviter, e := loadIdentity(r.Context(), tx, invite.InvitedBy)
		if e != nil {
			return invalidAction()
		}
		if !inviter.principal().Platform() && inviter.ID != invite.PreviousOwnerID {
			return invalidAction()
		}
		var owner string
		q = tx.Raw(`SELECT coalesce(primary_owner_id::text,'') FROM public.brands WHERE id=? AND suspended_at IS NULL FOR UPDATE`, invite.BrandID).Scan(&owner)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected != 1 || owner != invite.PreviousOwnerID {
			return invalidAction()
		}
		if e = tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "identity-email:"+invite.Email).Error; e != nil {
			return e
		}
		var user struct {
			ID, BrandID, Role, PasswordHash string
			IsActive                        bool
		}
		q = tx.Raw(`SELECT id::text,coalesce(brand_id::text,'') brand_id,role,password_hash,is_active FROM public.users WHERE lower(email)=? FOR UPDATE`, invite.Email).Scan(&user)
		if q.Error != nil {
			return q.Error
		}
		var employeeCount int64
		tx.Raw(`SELECT count(*) FROM public.employees WHERE lower(email)=?`, invite.Email).Scan(&employeeCount)
		if employeeCount > 0 {
			return inputError("El correo ya corresponde a un colaborador")
		}
		if q.RowsAffected > 0 {
			if user.BrandID != invite.BrandID || user.Role != "admin" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.CurrentPassword)) != nil {
				return ErrInvalidCredentials
			}
		}
		id := user.ID
		if id == "" {
			if !validPassword(in.Password) {
				return inputError("Elegí una contraseña de 12 a 72 bytes para tu cuenta nueva")
			}
			id = uuid.NewString()
			if e = tx.Exec(`INSERT INTO public.users(id,brand_id,role,email,first_name,last_name,password_hash,is_active) VALUES(?,?,'admin',?,?,?,?,true)`, id, invite.BrandID, invite.Email, in.First, in.Last, string(hashed)).Error; e != nil {
				return e
			}
		}
		if !user.IsActive && user.ID != "" {
			if e = tx.Exec(`UPDATE public.users SET is_active=true WHERE id=?`, user.ID).Error; e != nil {
				return e
			}
		}
		if e = tx.Exec(`UPDATE public.brands SET primary_owner_id=? WHERE id=?`, id, invite.BrandID).Error; e != nil {
			return e
		}
		if owner != "" && owner != id {
			if e = tx.Exec(`UPDATE public.users SET is_active=false WHERE id=? AND brand_id=? AND role='admin'`, owner, invite.BrandID).Error; e != nil {
				return e
			}
		}
		if e = tx.Exec(`UPDATE legui_private.business_invitations SET used_at=clock_timestamp() WHERE id=?`, invite.ID).Error; e != nil {
			return e
		}
		return Audit(r.Context(), tx, SessionPrincipal{ID: id, Kind: "user", Role: "admin", BrandID: invite.BrandID}, "business.owner_accepted", invite.BrandID, nil)
	})
	if e != nil {
		return businessError(e)
	}
	respondWithJSON(w, 200, map[string]string{"message": "Tu restaurante está listo. Ya podés iniciar sesión con tu correo y contraseña."})
	return nil
}

// A suspended owner retains access only to account security, membership and the
// business status. Provider webhooks/workers continue reconciling existing money.
func BusinessPathAllowed(path string) bool {
	return strings.HasPrefix(path, "/api/auth/") || strings.HasPrefix(path,"/api/care/") || path == "/api/business" || path == "/api/tenants" || strings.HasPrefix(path, "/api/billing/") || path == "/api/payments/mercadopago/config"
}
func businessAvailable(ctx context.Context, database *gorm.DB, p SessionPrincipal) (bool, error) {
	if p.Platform() || p.BrandID == "" {
		return true, nil
	}
	var active bool
	q := database.WithContext(ctx).Raw(`SELECT suspended_at IS NULL FROM public.brands WHERE id=?`, p.BrandID).Scan(&active)
	return q.RowsAffected == 1 && active, q.Error
}

const suspensionAccessKey contextKey = "suspended_business_account_access"
