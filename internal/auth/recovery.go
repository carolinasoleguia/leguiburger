package auth

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"time"
)

func publicSite() (string, error) {
	value := strings.TrimRight(os.Getenv("PUBLIC_SITE_URL"), "/")
	if value == "" && os.Getenv("APP_ENV") != "production" {
		value = "http://127.0.0.1:5173"
	}
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return "", errors.New("PUBLIC_SITE_URL is required")
	}
	if u.Scheme != "https" && !(os.Getenv("APP_ENV") != "production" && u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1")) {
		return "", errors.New("PUBLIC_SITE_URL must use HTTPS")
	}
	return value, nil
}
func validEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value && len(value) <= 254
}
func (h *SecurityHandler) forgotPassword(w http.ResponseWriter, r *http.Request) error {
	if err := h.ipRate(r, "password-forgot-ip", 20); err != nil {
		return err
	}
	var input struct {
		Email string `json:"email"`
	}
	if err := readSecurityJSON(r, &input); err != nil {
		return err
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if !validEmail(email) {
		return inputError("Ingresá un correo válido")
	}
	if err := h.rate(r, "password-forgot-account", email, 3); err != nil {
		return err
	}
	site, err := publicSite()
	if err != nil {
		return err
	}
	err = h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		var matches []struct {
			ID   string
			Kind string
		}
		if e := tx.Raw(`SELECT id::text,'user' kind FROM public.users WHERE lower(email)=? AND is_active UNION ALL SELECT id::text,'employee' kind FROM public.employees WHERE lower(email)=? AND is_active AND role IN ('employee','cashier','kitchen') LIMIT 10`, email, email).Scan(&matches).Error; e != nil {
			return e
		}
		for _, match := range matches {
			raw := randomToken()
			if e := tx.Exec(`INSERT INTO legui_private.auth_action_tokens(digest,purpose,principal_kind,principal_id,email,expires_at) VALUES(?,'password_reset',?,?,?,clock_timestamp()+interval '30 minutes')`, digest(raw), match.Kind, match.ID, email).Error; e != nil {
				return e
			}
			// URL fragments keep the credential out of HTTP access logs and referrers.
			link := site + "/reset-password#token=" + url.QueryEscape(raw)
			if e := enqueueMail(tx, email, "Restablecé tu contraseña de LeguiBurger", "Abrí este enlace para elegir una contraseña nueva. Vence en 30 minutos y funciona una sola vez.\n\n"+link+"\n\nSi no lo pediste, podés ignorar este correo.", time.Now().Add(30*time.Minute)); e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	respondWithJSON(w, 200, map[string]string{"message": "Si existe una cuenta con ese correo, recibirás un enlace para restablecer tu contraseña."})
	return nil
}

type actionToken struct {
	Digest        string
	Purpose       string
	PrincipalKind string
	PrincipalID   string
	Email         string
	TenantID      string
	Role          string
	InvitedBy     string
}

func actionByToken(tx *gorm.DB, raw, purpose string) (actionToken, error) {
	var action actionToken
	if len(raw) < 32 || len(raw) > 200 {
		return action, invalidAction()
	}
	result := tx.Raw(`SELECT digest,purpose,coalesce(principal_kind,'') principal_kind,coalesce(principal_id::text,'') principal_id,email,coalesce(tenant_id::text,'') tenant_id,coalesce(role,'') role,coalesce(invited_by::text,'') invited_by FROM legui_private.auth_action_tokens WHERE digest=? AND purpose=? AND used_at IS NULL AND expires_at>clock_timestamp() FOR UPDATE`, digest(raw), purpose).Scan(&action)
	if result.Error != nil {
		return action, result.Error
	}
	if result.RowsAffected != 1 {
		return action, invalidAction()
	}
	return action, nil
}
func (h *SecurityHandler) resetPassword(w http.ResponseWriter, r *http.Request) error {
	if err := h.ipRate(r, "password-reset-ip", 20); err != nil {
		return err
	}
	var input struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := readSecurityJSON(r, &input); err != nil {
		return err
	}
	if !validPassword(input.Password) {
		return inputError("Usá una contraseña de 12 a 72 bytes, sin espacios al principio ni al final")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		action, e := actionByToken(tx, input.Token, "password_reset")
		if e != nil {
			return e
		}
		table := "public.users"
		if action.PrincipalKind == "employee" {
			table = "public.employees"
		}
		result := tx.Exec(`UPDATE `+table+` SET password_hash=? WHERE id=? AND lower(email)=? AND is_active`, string(hashed), action.PrincipalID, action.Email)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return invalidAction()
		}
		if e = tx.Exec(`UPDATE legui_private.auth_action_tokens SET used_at=clock_timestamp() WHERE principal_kind=? AND principal_id=? AND purpose='password_reset' AND used_at IS NULL`, action.PrincipalKind, action.PrincipalID).Error; e != nil {
			return e
		}
		// The identity trigger revokes all sessions, including stolen refresh/access tokens.
		return Audit(r.Context(), tx, SessionPrincipal{ID: action.PrincipalID, Kind: action.PrincipalKind}, "auth.password_reset", action.PrincipalID, nil)
	})
	if err != nil {
		return err
	}
	setRefreshCookie(w, "")
	respondWithJSON(w, 200, map[string]string{"message": "Contraseña actualizada. Iniciá sesión nuevamente."})
	return nil
}
func (h *SecurityHandler) changePassword(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	if err := h.rate(r, "password-change", p.ID, 10); err != nil {
		return err
	}
	var input struct {
		CurrentPassword string `json:"current_password"`
		Password        string `json:"password"`
	}
	if err := readSecurityJSON(r, &input); err != nil {
		return err
	}
	if !validPassword(input.Password) {
		return inputError("Usá una contraseña de 12 a 72 bytes, sin espacios al principio ni al final")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		i, e := loadIdentity(r.Context(), tx, p.ID)
		if e != nil {
			return e
		}
		if e = lockIdentity(tx, i); e != nil {
			return e
		}
		i, e = loadIdentity(r.Context(), tx, p.ID)
		if e != nil {
			return e
		}
		if bcrypt.CompareHashAndPassword([]byte(i.PasswordHash), []byte(input.CurrentPassword)) != nil {
			return ErrInvalidCredentials
		}
		table := "public.users"
		if p.Kind == "employee" {
			table = "public.employees"
		}
		if e = tx.Exec(`UPDATE `+table+` SET password_hash=? WHERE id=?`, string(hashed), p.ID).Error; e != nil {
			return e
		}
		return Audit(r.Context(), tx, p, "auth.password_changed", p.ID, nil)
	})
	if err != nil {
		return err
	}
	setRefreshCookie(w, "")
	respondWithJSON(w, 200, map[string]string{"message": "Contraseña actualizada. Iniciá sesión nuevamente."})
	return nil
}
