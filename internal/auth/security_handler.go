package auth

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type SecurityHandler struct {
	database    *gorm.DB
	credentials Service
}
type securityError struct {
	status        int
	code, message string
}

func (e *securityError) Error() string { return e.message }
func invalidAction() error {
	return &securityError{400, "INVALID_ACTION", "El enlace venció o ya fue utilizado"}
}
func forbiddenAction() error {
	return &securityError{403, "FORBIDDEN", "No tenés permiso para realizar esta operación"}
}
func inputError(message string) error { return &securityError{400, "INVALID_INPUT", message} }
func unauthorizedAction() error {
	return &securityError{401, "UNAUTHORIZED", "La sesión venció o fue revocada"}
}

func RegisterSecurityRoutes(mux *http.ServeMux, database *gorm.DB, credentials Service) {
	h := &SecurityHandler{database: database, credentials: credentials}
	public := map[string]func(http.ResponseWriter, *http.Request) error{
		"POST /api/auth/business-invitations/accept": h.acceptBusinessInvitation,
		"POST /api/auth/email/confirm":               h.confirmEmail,
		"POST /api/auth/login":                       h.login, "POST /api/auth/refresh": h.refresh, "POST /api/auth/logout": h.logout,
		"POST /api/auth/password/forgot": h.forgotPassword, "POST /api/auth/password/reset": h.resetPassword,
		"POST /api/auth/mfa/enroll": h.enrollMFA, "POST /api/auth/mfa/verify": h.verifyMFA,
		"POST /api/auth/invitations/accept": h.acceptInvitation,
	}
	for pattern, fn := range public {
		mux.HandleFunc(pattern, h.handle(fn))
	}
	protected := map[string]func(http.ResponseWriter, *http.Request) error{
		"GET /api/membership":                h.membership,
		"GET /api/billing/terminals":         h.operatingTerminals,
		"DELETE /api/billing/terminals/{id}": h.revokeOperatingTerminal,
		"GET /api/business":                  h.businesses, "GET /api/platform/restaurants": h.businesses,
		"POST /api/platform/restaurants":              h.createRestaurant,
		"POST /api/business/{id}/owner-invitation":    h.businessInvitation,
		"DELETE /api/business/{id}/owner-invitation":  h.revokeBusinessInvitation,
		"PATCH /api/platform/restaurants/{id}/status": h.businessStatus,
		"GET /api/business/locations":                 h.locations, "POST /api/business/locations": h.saveLocation, "PATCH /api/business/locations/{id}": h.saveLocation,
		"GET /api/business/team": h.teamDirectory, "PATCH /api/business/team/{id}": h.updateTeamMember,
		"POST /api/auth/location":     h.switchLocation,
		"POST /api/auth/email/change": h.changeEmail,
		"GET /api/auth/me":            h.me, "GET /api/auth/sessions": h.sessions, "DELETE /api/auth/sessions/{id}": h.revokeSession,
		"POST /api/auth/logout-all": h.logoutAll, "POST /api/auth/password/change": h.changePassword,
		"POST /api/team/invitations": h.invite, "GET /api/team/invitations": h.invitations, "DELETE /api/team/invitations/{id}": h.revokeInvitation,
		"GET /api/audit": h.audit,
	}
	for pattern, fn := range protected {
		mux.HandleFunc(pattern, AuthMiddlewareWithDB(database, h.handle(fn)))
	}
}
func (h *SecurityHandler) handle(fn func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if r.Method != "GET" && !sameOrigin(r) {
			RespondWithError(w, 403, "INVALID_ORIGIN", "Origen de la solicitud no permitido")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		err := fn(w, r.WithContext(ctx))
		if err == nil {
			return
		}
		err = businessError(err)
		var api *securityError
		if errors.As(err, &api) {
			RespondWithError(w, api.status, api.code, api.message)
			return
		}
		if errors.Is(err, ErrInvalidCredentials) {
			RespondWithError(w, 401, "UNAUTHORIZED", "Credenciales o sesión inválidas")
			return
		}
		RespondWithError(w, 503, "AUTH_UNAVAILABLE", "No se pudo completar la operación. Reintentá en unos minutos")
	}
}
func readSecurityJSON(r *http.Request, v any) error {
	d := json.NewDecoder(io.LimitReader(r.Body, 16*1024))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return inputError("Revisá los datos ingresados")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return inputError("Se requiere un único objeto JSON")
	}
	return nil
}
func sameOrigin(r *http.Request) bool {
	if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	if configured := os.Getenv("PUBLIC_SITE_URL"); configured != "" {
		return strings.TrimRight(origin, "/") == strings.TrimRight(configured, "/")
	}
	if os.Getenv("APP_ENV") == "production" {
		return false
	}
	return u.Scheme == "http" && (u.Host == r.Host || u.Host == "127.0.0.1:5173" || u.Host == "localhost:5173")
}
func secureCookie() bool {
	return os.Getenv("APP_ENV") == "production" || strings.HasPrefix(os.Getenv("PUBLIC_SITE_URL"), "https://")
}

const refreshCookie = "legui_refresh"

func setRefreshCookie(w http.ResponseWriter, value string) {
	maxAge := 7 * 24 * 60 * 60
	if value == "" {
		maxAge = -1
	}
	http.SetCookie(w, &http.Cookie{Name: refreshCookie, Value: value, Path: "/api/auth", HttpOnly: true, Secure: secureCookie(), SameSite: http.SameSiteStrictMode, MaxAge: maxAge})
}
func (h *SecurityHandler) rate(r *http.Request, scope, key string, max int) error {
	var count int
	result := h.database.WithContext(r.Context()).Raw(`INSERT INTO legui_private.auth_rate_limits(bucket,attempts) VALUES(?,1) ON CONFLICT(bucket) DO UPDATE SET attempts=CASE WHEN auth_rate_limits.window_start<clock_timestamp()-interval '15 minutes' THEN 1 ELSE auth_rate_limits.attempts+1 END,window_start=CASE WHEN auth_rate_limits.window_start<clock_timestamp()-interval '15 minutes' THEN clock_timestamp() ELSE auth_rate_limits.window_start END RETURNING attempts`, scope+":"+digest(string(jwtSecret)+"/rate/"+strings.ToLower(strings.TrimSpace(key)))).Scan(&count)
	if result.Error != nil {
		return result.Error
	}
	if count > max {
		return &securityError{429, "TOO_MANY_ATTEMPTS", "Demasiados intentos. Esperá 15 minutos antes de volver a intentar"}
	}
	return nil
}
func (h *SecurityHandler) ipRate(r *http.Request, scope string, max int) error {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	return h.rate(r, scope, ip, max)
}
func (h *SecurityHandler) login(w http.ResponseWriter, r *http.Request) error {
	if err := h.ipRate(r, "login-ip", 60); err != nil {
		return err
	}
	var input LoginInput
	if err := readSecurityJSON(r, &input); err != nil {
		return err
	}
	if len(input.Email) > 254 || len(input.Password) > 72 {
		return ErrInvalidCredentials
	}
	if err := h.rate(r, "login-account", input.Email, 10); err != nil {
		return err
	}
	response, err := h.credentials.Login(r.Context(), input.TenantID, input.Email, input.Password)
	if err != nil {
		// Equalize the inexpensive unknown-user branch with a real bcrypt check.
		if errors.Is(err, ErrInvalidCredentials) {
			_ = bcrypt.CompareHashAndPassword([]byte(dummyPasswordHash), []byte(input.Password))
		}
		return ErrInvalidCredentials
	}
	c, err := ValidateToken(response.Token)
	if err != nil {
		return err
	}
	var output map[string]any
	var refresh string
	err = h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		i, e := loadIdentity(r.Context(), tx, c.UserID)
		if e != nil {
			return e
		}
		// Recheck credentials under a row lock before creating the new session.
		if e = lockIdentity(tx, i); e != nil {
			return e
		}
		i, e = loadIdentity(r.Context(), tx, c.UserID)
		if e != nil {
			return e
		}
		if bcrypt.CompareHashAndPassword([]byte(i.PasswordHash), []byte(strings.TrimSpace(input.Password))) != nil {
			return ErrInvalidCredentials
		}
		if i.Kind == "employee" && input.TenantID != "" {
			var allowed bool
			if e = tx.Raw(`SELECT EXISTS(SELECT 1 FROM legui_private.employee_locations a JOIN public.tenants t ON t.id=a.tenant_id AND t.active WHERE a.employee_id=? AND a.tenant_id=? AND t.brand_id=?)`, i.ID, input.TenantID, i.BrandID).Scan(&allowed).Error; e != nil {
				return e
			}
			if !allowed {
				return ErrInvalidCredentials
			}
			i.TenantID = input.TenantID
		}
		sid := uuid.NewString()
		// Replacing a login in the same browser must not consume a second terminal.
		// Credentials have already been checked; only this identity's cookie session is released.
		if cookie, err := r.Cookie(refreshCookie); err == nil && len(cookie.Value) <= 200 {
			if e = tx.Exec(`UPDATE legui_private.auth_sessions s SET revoked_at=coalesce(s.revoked_at,clock_timestamp()) FROM legui_private.auth_refresh_tokens t WHERE t.digest=? AND t.session_id=s.id AND s.principal_id=? AND s.principal_kind=?`, digest(cookie.Value), i.ID, i.Kind).Error; e != nil {
				return e
			}
		}
		platform := i.principal().Platform() && PlatformMFARequired()
		lifetime := "7 days"
		if platform {
			lifetime = "5 minutes"
		}
		if e = tx.Exec(`INSERT INTO legui_private.auth_sessions(id,principal_kind,principal_id,role,brand_id,tenant_id,credential_digest,expires_at,user_agent) VALUES(?,?,?,?,nullif(?,'')::uuid,nullif(?,'')::uuid,?,clock_timestamp()+?::interval,?)`, sid, i.Kind, i.ID, i.Role, i.BrandID, i.TenantID, digest(i.PasswordHash), lifetime, truncate(r.UserAgent(), 200)).Error; e != nil {
			return e
		}
		if platform {
			var enrolled bool
			if e = tx.Raw(`SELECT EXISTS(SELECT 1 FROM legui_private.auth_factors WHERE principal_id=? AND verified_at IS NOT NULL)`, i.ID).Scan(&enrolled).Error; e != nil {
				return e
			}
			challenge, e := SessionToken(i.principal(), sid, false, "mfa")
			if e != nil {
				return e
			}
			output = map[string]any{"mfa_required": true, "enrollment_required": !enrolled, "challenge": challenge}
		} else {
			output, refresh, e = issueSession(tx, i, sid, false)
			if e != nil {
				return e
			}
		}
		return Audit(r.Context(), tx, i.principal(), "auth.password_verified", sid, nil)
	})
	if err != nil {
		return err
	}
	setRefreshCookie(w, refresh)
	respondWithJSON(w, 200, output)
	return nil
}

const dummyPasswordHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

func truncate(value string, max int) string {
	if len(value) > max {
		return value[:max]
	}
	return value
}
func lockIdentity(tx *gorm.DB, i identity) error {
	table := "public.users"
	if i.Kind == "employee" {
		table = "public.employees"
	}
	var id string
	result := tx.Raw(`SELECT id::text FROM `+table+` WHERE id=? AND is_active FOR SHARE`, i.ID).Scan(&id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrInvalidCredentials
	}
	return nil
}
func issueSession(tx *gorm.DB, i identity, sid string, mfa bool) (map[string]any, string, error) {
	raw := randomToken()
	if err := tx.Exec(`INSERT INTO legui_private.auth_refresh_tokens(digest,session_id) VALUES(?,?)`, digest(raw), sid).Error; err != nil {
		return nil, "", err
	}
	token, err := SessionToken(i.principal(), sid, mfa, "access")
	if err != nil {
		return nil, "", err
	}
	user := map[string]any{"id": i.ID, "role": i.Role, "email": i.Email, "brand_id": i.BrandID, "tenant_id": i.TenantID, "first_name": i.FirstName, "last_name": i.LastName, "is_active": true}
	return map[string]any{"token": token, "user": user, "session_id": sid, "expires_in": 900}, raw, nil
}
func (h *SecurityHandler) refresh(w http.ResponseWriter, r *http.Request) error {
	if err := h.ipRate(r, "refresh-ip", 120); err != nil {
		return err
	}
	cookie, err := r.Cookie(refreshCookie)
	if err != nil || len(cookie.Value) > 200 {
		return unauthorizedAction()
	}
	var output map[string]any
	var raw string
	var rejected bool
	err = h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		var token struct {
			SessionID  string
			ConsumedAt *time.Time
		}
		result := tx.Raw(`SELECT session_id::text,consumed_at FROM legui_private.auth_refresh_tokens WHERE digest=?`, digest(cookie.Value)).Scan(&token)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			rejected = true
			return nil
		}
		// All rotation and replay detection serialize on the session row.
		s, e := loadSession(r.Context(), tx, token.SessionID, true)
		if e != nil {
			if errors.Is(e, ErrInvalidCredentials) {
				rejected = true
				return nil
			}
			return e
		}
		result = tx.Raw(`SELECT session_id::text,consumed_at FROM legui_private.auth_refresh_tokens WHERE digest=? FOR UPDATE`, digest(cookie.Value)).Scan(&token)
		if result.Error != nil {
			return result.Error
		}
		if token.ConsumedAt != nil {
			rejected = true
			return tx.Exec(`UPDATE legui_private.auth_sessions SET revoked_at=clock_timestamp() WHERE id=?`, s.ID).Error
		}
		i, e := sessionIdentity(r.Context(), tx, s)
		if e != nil {
			if errors.Is(e, ErrInvalidCredentials) {
				rejected = true
				return nil
			}
			return e
		}
		if expected := r.Header.Get("X-Legui-Account"); expected != "" && expected != i.ID {
			return &securityError{409, "ACCOUNT_CHANGED", "La cuenta de esta pestaña cambió. Iniciá sesión nuevamente"}
		}
		if e = tx.Exec(`UPDATE legui_private.auth_refresh_tokens SET consumed_at=clock_timestamp() WHERE digest=?`, digest(cookie.Value)).Error; e != nil {
			return e
		}
		if e = tx.Exec(`UPDATE legui_private.auth_sessions SET last_seen_at=clock_timestamp() WHERE id=?`, s.ID).Error; e != nil {
			return e
		}
		output, raw, e = issueSession(tx, i, s.ID, s.MFAVerified)
		return e
	})
	if err != nil {
		return err
	}
	if rejected {
		setRefreshCookie(w, "")
		return unauthorizedAction()
	}
	setRefreshCookie(w, raw)
	respondWithJSON(w, 200, output)
	return nil
}
func (h *SecurityHandler) logout(w http.ResponseWriter, r *http.Request) error {
	if err := h.ipRate(r, "logout-ip", 120); err != nil {
		return err
	}
	if cookie, err := r.Cookie(refreshCookie); err == nil && len(cookie.Value) <= 200 {
		if err = h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
			var sid string
			result := tx.Raw(`SELECT session_id::text FROM legui_private.auth_refresh_tokens WHERE digest=?`, digest(cookie.Value)).Scan(&sid)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return nil
			}
			return tx.Exec(`UPDATE legui_private.auth_sessions SET revoked_at=coalesce(revoked_at,clock_timestamp()) WHERE id=?`, sid).Error
		}); err != nil {
			return err
		}
	}
	setRefreshCookie(w, "")
	respondWithJSON(w, 200, map[string]bool{"revoked": true})
	return nil
}
func (h *SecurityHandler) me(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	i, err := loadIdentity(r.Context(), h.database, p.ID)
	if err != nil {
		return err
	}
	respondWithJSON(w, 200, map[string]any{"id": i.ID, "email": i.Email, "role": i.Role, "brand_id": i.BrandID, "tenant_id": i.TenantID, "first_name": i.FirstName, "last_name": i.LastName})
	return nil
}
func (h *SecurityHandler) sessions(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	c, _ := GetClaimsFromContext(r.Context())
	var rows []map[string]any
	if err := h.database.WithContext(r.Context()).Raw(`SELECT id,created_at,last_seen_at,expires_at,user_agent,(id=?::uuid) AS current FROM legui_private.auth_sessions WHERE principal_kind=? AND principal_id=? AND revoked_at IS NULL AND expires_at>clock_timestamp() AND last_seen_at>clock_timestamp()-interval '30 minutes' ORDER BY created_at DESC LIMIT 100`, c.SessionID, p.Kind, p.ID).Scan(&rows).Error; err != nil {
		return err
	}
	respondWithJSON(w, 200, rows)
	return nil
}
func (h *SecurityHandler) revokeSession(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		return inputError("Sesión inválida")
	}
	if err := h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(`UPDATE legui_private.auth_sessions SET revoked_at=coalesce(revoked_at,clock_timestamp()) WHERE id=? AND principal_id=? AND principal_kind=?`, id, p.ID, p.Kind)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return forbiddenAction()
		}
		return Audit(r.Context(), tx, p, "auth.session_revoked", id, nil)
	}); err != nil {
		return err
	}
	c, _ := GetClaimsFromContext(r.Context())
	if c.SessionID == id {
		setRefreshCookie(w, "")
	}
	respondWithJSON(w, 200, map[string]bool{"revoked": true})
	return nil
}
func (h *SecurityHandler) logoutAll(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	if err := h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec(`UPDATE legui_private.auth_sessions SET revoked_at=clock_timestamp() WHERE principal_kind=? AND principal_id=? AND revoked_at IS NULL`, p.Kind, p.ID).Error; e != nil {
			return e
		}
		return Audit(r.Context(), tx, p, "auth.all_sessions_revoked", p.ID, nil)
	}); err != nil {
		return err
	}
	setRefreshCookie(w, "")
	respondWithJSON(w, 200, map[string]bool{"revoked": true})
	return nil
}
func limitOffset(r *http.Request) (int, int, error) {
	limit, offset := 50, 0
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, inputError("Paginación inválida")
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, inputError("Paginación inválida")
		}
	}
	if limit < 1 || limit > 100 || offset < 0 || offset > 100000 {
		return 0, 0, inputError("Paginación inválida")
	}
	return limit, offset, nil
}
