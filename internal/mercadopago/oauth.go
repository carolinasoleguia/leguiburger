package mercadopago

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"leguiburger/internal/auth"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type credentials struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if e := h.limitAction(r, "authorize", p.ID, 10); e != nil {
		return e
	}
	if e := manager(p); e != nil {
		return e
	}
	id, e := routeID(r)
	if e != nil {
		return e
	}
	c, e := h.connection(r, p, id)
	if e != nil {
		return e
	}
	if c.Scope != "business" {
		return denied()
	}
	if e = h.enabled(c.Environment); e != nil {
		return e
	}
	if h.cfg.ClientID == "" || h.cfg.ClientSecret == "" {
		return notConfigured()
	}
	state, binding, verifier := random(), random(), random()
	ciphertext, e := h.cfg.seal(verifier, "oauth/"+id)
	if e != nil {
		return notConfigured()
	}
	attempt := uuid.NewString()
	newVersion := uuid.NewString()
	claims, _ := auth.GetClaimsFromContext(r.Context())
	e = h.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := auth.LockSession(r.Context(), tx); e != nil {
			return e
		}
		result := tx.Exec(`UPDATE legui_private.mp_connections SET state='connecting',version=?,credential_version=gen_random_uuid(),credential_cipher=NULL,refresh_claim=NULL,refresh_lease_until=NULL,updated_at=clock_timestamp() WHERE id=? AND version=?`, newVersion, id, c.Version)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return conflict("La conexión cambió; actualizá la pantalla")
		}
		if e := tx.Exec(`UPDATE legui_private.mp_oauth_attempts SET status='superseded',verifier_cipher=NULL WHERE connection_id=? AND status IN ('pending','submitting')`, id).Error; e != nil {
			return e
		}
		if e := tx.Exec(`INSERT INTO legui_private.mp_oauth_attempts(id,connection_id,connection_version,actor_id,session_id,state_digest,binding_digest,verifier_cipher) VALUES(?,?,?,?,?,?,?,?)`, attempt, id, newVersion, p.ID, claims.SessionID, hash(state), hash(binding), ciphertext).Error; e != nil {
			return e
		}
		return auth.Audit(r.Context(), tx, p, "payments.oauth_started", id, nil)
	})
	if e != nil {
		return e
	}
	http.SetCookie(w, &http.Cookie{Name: "legui_mp_oauth", Value: binding, Path: "/api/integrations/mercadopago/callback", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 600})
	challenge := sha256.Sum256([]byte(verifier))
	query := url.Values{"client_id": {h.cfg.ClientID}, "response_type": {"code"}, "redirect_uri": {h.cfg.APIOrigin + "/api/integrations/mercadopago/callback"}, "state": {state}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}, "code_challenge_method": {"S256"}}
	respond(w, 200, map[string]string{"authorization_url": "https://auth.mercadopago.com/authorization?" + query.Encode()})
	return nil
}
func (h *Handler) callback(w http.ResponseWriter, r *http.Request) error {
	if !h.cfg.ready() {
		return notConfigured()
	}
	query := r.URL.Query()
	for _, name := range []string{"state", "code", "error"} {
		if len(query[name]) > 1 {
			return bad("Autorización ambigua")
		}
	}
	state, code := query.Get("state"), query.Get("code")
	cookie, e := r.Cookie("legui_mp_oauth")
	if e != nil || len(state) > 200 || len(code) > 2048 || code == "" || query.Get("error") != "" {
		return denied()
	}
	var attempt struct {
		ID                string
		ConnectionID      string
		ConnectionVersion string
		VerifierCipher    string
		Environment       string
		ActorID           string
	}
	e = h.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		result := tx.Raw(`SELECT a.id::text,a.connection_id::text,a.connection_version::text,a.verifier_cipher,c.environment,a.actor_id::text FROM legui_private.mp_oauth_attempts a JOIN legui_private.mp_connections c ON c.id=a.connection_id JOIN legui_private.auth_sessions s ON s.id=a.session_id JOIN public.users u ON u.id=a.actor_id WHERE a.state_digest=? AND a.binding_digest=? AND a.status='pending' AND a.expires_at>clock_timestamp() AND c.version=a.connection_version AND c.state='connecting' AND s.revoked_at IS NULL AND s.expires_at>clock_timestamp() AND s.last_seen_at>clock_timestamp()-interval '30 minutes' AND u.is_active AND u.role=s.role AND (u.role IN ('owner','super_admin') AND s.mfa_verified OR u.role='admin' AND u.brand_id=c.brand_id AND s.brand_id=u.brand_id) FOR UPDATE OF a,c`, hash(state), hash(cookie.Value)).Scan(&attempt)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return denied()
		}
		return tx.Exec(`UPDATE legui_private.mp_oauth_attempts SET status='submitting' WHERE id=?`, attempt.ID).Error
	})
	if e != nil {
		return e
	}
	verifier, e := h.cfg.open(attempt.VerifierCipher, "oauth/"+attempt.ConnectionID)
	if e != nil {
		return e
	}
	remote, e := h.provider.call(r.Context(), "POST", "/oauth/token", "", "", map[string]any{"client_id": h.cfg.ClientID, "client_secret": h.cfg.ClientSecret, "grant_type": "authorization_code", "code": code, "redirect_uri": h.cfg.APIOrigin + "/api/integrations/mercadopago/callback", "code_verifier": verifier, "test_token": attempt.Environment == "sandbox"})
	if e != nil {
		_ = h.failOAuth(r.Context(), attempt.ID, attempt.ConnectionID, attempt.ConnectionVersion)
		return e
	}
	token, refresh, collector := text(remote["access_token"]), text(remote["refresh_token"]), text(remote["user_id"])
	expires, e := strconv.ParseInt(text(remote["expires_in"]), 10, 64)
	live, ok := remote["live_mode"].(bool)
	if e != nil || expires < 60 || expires > 366*24*60*60 || !ok || live != (attempt.Environment == "production") || token == "" || refresh == "" || !resourcePattern.MatchString(collector) {
		_ = h.failOAuth(r.Context(), attempt.ID, attempt.ConnectionID, attempt.ConnectionVersion)
		return conflict("No se pudo verificar la cuenta de Mercado Pago")
	}
	me, e := h.provider.call(r.Context(), "GET", "/users/me", token, "", nil)
	if e != nil || text(me["id"]) != collector {
		_ = h.failOAuth(r.Context(), attempt.ID, attempt.ConnectionID, attempt.ConnectionVersion)
		return conflict("No se pudo verificar el titular de la cuenta")
	}
	encrypted, e := h.cfg.seal(jsonString(credentials{AccessToken: token, RefreshToken: refresh}), "connection/"+attempt.ConnectionID)
	if e != nil {
		return e
	}
	e = h.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(`UPDATE legui_private.mp_connections c SET state='connected',collector_id=?,credential_cipher=?,credential_version=gen_random_uuid(),expires_at=?,updated_at=clock_timestamp() WHERE c.id=? AND c.version=? AND c.state='connecting' AND EXISTS(SELECT 1 FROM legui_private.mp_oauth_attempts a JOIN legui_private.auth_sessions s ON s.id=a.session_id JOIN public.users u ON u.id=a.actor_id WHERE a.id=? AND a.status='submitting' AND a.expires_at>clock_timestamp() AND s.revoked_at IS NULL AND s.expires_at>clock_timestamp() AND s.last_seen_at>clock_timestamp()-interval '30 minutes' AND u.is_active AND u.role=s.role AND (u.role IN ('owner','super_admin') AND s.mfa_verified OR u.role='admin' AND u.brand_id=c.brand_id AND s.brand_id=u.brand_id))`, collector, encrypted, time.Now().Add(time.Duration(expires)*time.Second), attempt.ConnectionID, attempt.ConnectionVersion, attempt.ID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return conflict("La conexión fue reemplazada durante la autorización")
		}
		if e := tx.Exec(`UPDATE legui_private.mp_oauth_attempts SET status='completed',verifier_cipher=NULL WHERE id=? AND status='submitting'`, attempt.ID).Error; e != nil {
			return e
		}
		return auth.Audit(r.Context(), tx, auth.SessionPrincipal{Kind: "user", ID: attempt.ActorID}, "payments.oauth_connected", attempt.ConnectionID, nil)
	})
	if e != nil {
		return e
	}
	http.SetCookie(w, &http.Cookie{Name: "legui_mp_oauth", Value: "", Path: "/api/integrations/mercadopago/callback", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	http.Redirect(w, r, h.cfg.PublicOrigin+"/payments?connected=1", http.StatusSeeOther)
	return nil
}
func (h *Handler) failOAuth(ctx context.Context, attempt, id, version string) error {
	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec(`UPDATE legui_private.mp_oauth_attempts SET status='review',verifier_cipher=NULL WHERE id=? AND status='submitting'`, attempt).Error; e != nil {
			return e
		}
		return tx.Exec(`UPDATE legui_private.mp_connections SET state='needs_reconnect',updated_at=clock_timestamp() WHERE id=? AND version=? AND state='connecting'`, id, version).Error
	})
}
func (h *Handler) account(ctx context.Context, id string) (connection, string, error) {
	var c connection
	result := h.db.WithContext(ctx).Raw(`SELECT `+connectionColumns+` FROM legui_private.mp_connections WHERE id=? AND state='connected'`, id).Scan(&c)
	if result.Error != nil {
		return c, "", result.Error
	}
	if result.RowsAffected != 1 {
		return c, "", conflict("La cuenta de cobro necesita reconectarse")
	}
	if e := h.enabled(c.Environment); e != nil {
		return c, "", e
	}
	if c.Scope == "platform" {
		if h.cfg.PlatformToken == "" || c.CollectorID != h.cfg.PlatformCollector {
			return c, "", notConfigured()
		}
		return c, h.cfg.PlatformToken, nil
	}
	raw, e := h.cfg.open(c.CredentialCipher, "connection/"+c.ID)
	if e != nil {
		return c, "", e
	}
	var keys credentials
	if json.Unmarshal([]byte(raw), &keys) != nil || keys.AccessToken == "" || keys.RefreshToken == "" {
		return c, "", notConfigured()
	}
	if c.ExpiresAt != nil && c.ExpiresAt.After(time.Now().Add(time.Minute)) {
		return c, keys.AccessToken, nil
	}
	claim := uuid.NewString()
	result = h.db.WithContext(ctx).Exec(`UPDATE legui_private.mp_connections SET refresh_claim=?,refresh_lease_until=clock_timestamp()+interval '1 minute' WHERE id=? AND version=? AND credential_version=? AND state='connected' AND refresh_claim IS NULL`, claim, c.ID, c.Version, c.CredentialVersion)
	if result.Error != nil {
		return c, "", result.Error
	}
	if result.RowsAffected != 1 {
		_ = h.db.WithContext(ctx).Exec(`UPDATE legui_private.mp_connections SET state='needs_reconnect',updated_at=clock_timestamp() WHERE id=? AND refresh_claim IS NOT NULL AND refresh_lease_until<clock_timestamp()`, c.ID).Error
		return c, "", conflict("La renovación de la cuenta está en curso o requiere reconexión")
	}
	remote, e := h.provider.call(ctx, "POST", "/oauth/token", "", "", map[string]any{"client_id": h.cfg.ClientID, "client_secret": h.cfg.ClientSecret, "grant_type": "refresh_token", "refresh_token": keys.RefreshToken, "test_token": c.Environment == "sandbox"})
	if e != nil {
		_ = h.db.WithContext(context.WithoutCancel(ctx)).Exec(`UPDATE legui_private.mp_connections SET state='needs_reconnect',updated_at=clock_timestamp() WHERE id=? AND refresh_claim=?`, c.ID, claim).Error
		return c, "", e
	}
	expires, e := strconv.ParseInt(text(remote["expires_in"]), 10, 64)
	live, valid := remote["live_mode"].(bool)
	if e != nil || expires < 60 || expires > 366*86400 || !valid || live != (c.Environment == "production") || text(remote["user_id"]) != c.CollectorID || text(remote["access_token"]) == "" || text(remote["refresh_token"]) == "" {
		_ = h.db.WithContext(ctx).Exec(`UPDATE legui_private.mp_connections SET state='needs_reconnect' WHERE id=? AND refresh_claim=?`, c.ID, claim).Error
		return c, "", errors.New("refresh ownership unconfirmed")
	}
	keys = credentials{AccessToken: text(remote["access_token"]), RefreshToken: text(remote["refresh_token"])}
	encrypted, e := h.cfg.seal(jsonString(keys), "connection/"+c.ID)
	if e != nil {
		return c, "", e
	}
	version := uuid.NewString()
	result = h.db.WithContext(ctx).Exec(`UPDATE legui_private.mp_connections SET credential_cipher=?,credential_version=?,expires_at=?,refresh_claim=NULL,refresh_lease_until=NULL,updated_at=clock_timestamp() WHERE id=? AND version=? AND credential_version=? AND refresh_claim=? AND state='connected'`, encrypted, version, time.Now().Add(time.Duration(expires)*time.Second), c.ID, c.Version, c.CredentialVersion, claim)
	if result.Error != nil {
		return c, "", result.Error
	}
	if result.RowsAffected != 1 {
		return c, "", conflict("La conexión cambió durante la renovación")
	}
	c.CredentialVersion = version
	return c, keys.AccessToken, nil
}
func (h *Handler) platform(ctx context.Context) (connection, error) {
	var c connection
	if e := h.enabled(h.cfg.Environment); e != nil {
		return c, e
	}
	if h.cfg.PlatformToken == "" || !resourcePattern.MatchString(h.cfg.PlatformCollector) {
		return c, notConfigured()
	}
	remote, e := h.provider.call(ctx, "GET", "/users/me", h.cfg.PlatformToken, "", nil)
	if e != nil {
		return c, e
	}
	if text(remote["id"]) != h.cfg.PlatformCollector {
		return c, conflict("La cuenta de la plataforma no coincide con la configurada")
	}
	e = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "mp-platform:"+h.cfg.Environment).Error; e != nil {
			return e
		}
		result := tx.Raw(`SELECT `+connectionColumns+` FROM legui_private.mp_connections WHERE scope='platform' AND environment=? FOR UPDATE`, h.cfg.Environment).Scan(&c)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return tx.Raw(`INSERT INTO legui_private.mp_connections(scope,environment,state,collector_id) VALUES('platform',?,'connected',?) RETURNING `+connectionColumns, h.cfg.Environment, h.cfg.PlatformCollector).Scan(&c).Error
		}
		if c.CollectorID != h.cfg.PlatformCollector {
			return conflict("La cuenta receptora de la plataforma cambió; requiere revisión")
		}
		return nil
	})
	return c, e
}
