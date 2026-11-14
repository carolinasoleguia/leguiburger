package auth

import (
	"crypto/sha256"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func ValidPassword(value string) bool { return validPassword(value) }

// Both mailboxes must approve. A credential change or a newer request invalidates
// the pending operation. Existing login email stays intact until both approvals.
func (h *SecurityHandler) changeEmail(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFromContext(r.Context())
	if e := h.rate(r, "email-change", p.ID, 3); e != nil {
		return e
	}
	if !MailConfigured() {
		return inputError("El envío de correo todavía no está configurado")
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"current_password"`
	}
	if e := readSecurityJSON(r, &in); e != nil {
		return e
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if !validEmail(in.Email) {
		return inputError("Ingresá un correo válido")
	}
	site, e := publicSite()
	if e != nil {
		return e
	}
	e = h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := LockSession(r.Context(), tx); e != nil {
			return e
		}
		i, e := loadIdentity(r.Context(), tx, p.ID)
		if e != nil {
			return e
		}
		if e = lockIdentity(tx, i); e != nil {
			return e
		}
		if bcrypt.CompareHashAndPassword([]byte(i.PasswordHash), []byte(in.Password)) != nil {
			return ErrInvalidCredentials
		}
		if strings.EqualFold(i.Email, in.Email) {
			return inputError("Elegí un correo diferente al actual")
		}
		var count int64
		if e = tx.Raw(`SELECT (SELECT count(*) FROM public.users WHERE lower(email)=?)+(SELECT count(*) FROM public.employees WHERE lower(email)=?)`, in.Email, in.Email).Scan(&count).Error; e != nil {
			return e
		}
		if count > 0 {
			return inputError("Ese correo no está disponible")
		}
		if e = tx.Exec(`UPDATE legui_private.email_changes SET canceled_at=clock_timestamp() WHERE principal_kind=? AND principal_id=? AND completed_at IS NULL AND canceled_at IS NULL`, p.Kind, p.ID).Error; e != nil {
			return e
		}
		old, newToken := randomToken(), randomToken()
		if e = tx.Exec(`INSERT INTO legui_private.email_changes(principal_kind,principal_id,old_email,new_email,credential_digest,old_digest,new_digest) VALUES(?,?,?,?,?,?,?)`, p.Kind, p.ID, strings.ToLower(i.Email), in.Email, fmt.Sprintf("%x", sha256.Sum256([]byte(i.PasswordHash))), digest(old), digest(newToken)).Error; e != nil {
			return e
		}
		for recipient, token := range map[string]string{i.Email: old, in.Email: newToken} {
			if e = enqueueMail(tx, recipient, "Confirmá el cambio de correo de LeguiBurger", "Confirmá el cambio solicitado. Sólo se aplicará cuando ambos correos lo aprueben. Vence en 30 minutos.\n\n"+site+"/confirm-email#token="+url.QueryEscape(token), time.Now().Add(30*time.Minute)); e != nil {
				return e
			}
		}
		return Audit(r.Context(), tx, p, "auth.email_change_requested", p.ID, nil)
	})
	if e != nil {
		return e
	}
	respondWithJSON(w, 202, map[string]string{"message": "Enviamos una confirmación a ambos correos. Tu acceso actual sigue vigente."})
	return nil
}
func (h *SecurityHandler) confirmEmail(w http.ResponseWriter, r *http.Request) error {
	if e := h.ipRate(r, "email-confirm", 20); e != nil {
		return e
	}
	var in struct {
		Token string `json:"token"`
	}
	if e := readSecurityJSON(r, &in); e != nil {
		return e
	}
	if len(in.Token) < 32 || len(in.Token) > 200 {
		return invalidAction()
	}
	completed := false
	e := h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		var item struct {
			ID, PrincipalKind, PrincipalID, OldEmail, NewEmail, CredentialDigest, OldDigest string
			OldConfirmedAt, NewConfirmedAt                                                  *time.Time
		}
		q := tx.Raw(`SELECT * FROM legui_private.email_changes WHERE (old_digest=? OR new_digest=?) AND completed_at IS NULL AND canceled_at IS NULL AND expires_at>clock_timestamp() FOR UPDATE`, digest(in.Token), digest(in.Token)).Scan(&item)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected != 1 {
			return invalidAction()
		}
		i, e := loadIdentity(r.Context(), tx, item.PrincipalID)
		if e != nil {
			return e
		}
		if i.Kind != item.PrincipalKind {
			return invalidAction()
		}
		if e = lockIdentity(tx, i); e != nil {
			return e
		}
		i, e = loadIdentity(r.Context(), tx, item.PrincipalID)
		if e != nil {
			return e
		}
		if strings.ToLower(i.Email) != item.OldEmail || fmt.Sprintf("%x", sha256.Sum256([]byte(i.PasswordHash))) != item.CredentialDigest {
			return invalidAction()
		}
		column := "new_confirmed_at"
		if digest(in.Token) == item.OldDigest {
			column = "old_confirmed_at"
		}
		if e = tx.Exec(`UPDATE legui_private.email_changes SET `+column+`=coalesce(`+column+`,clock_timestamp()) WHERE id=?`, item.ID).Error; e != nil {
			return e
		}
		if e = tx.Raw(`SELECT old_confirmed_at IS NOT NULL AND new_confirmed_at IS NOT NULL FROM legui_private.email_changes WHERE id=?`, item.ID).Scan(&completed).Error; e != nil {
			return e
		}
		if completed {
			if e = tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "email:"+item.NewEmail).Error; e != nil {
				return e
			}
			var count int64
			if e = tx.Raw(`SELECT (SELECT count(*) FROM public.users WHERE lower(email)=?)+(SELECT count(*) FROM public.employees WHERE lower(email)=?)`, item.NewEmail, item.NewEmail).Scan(&count).Error; e != nil {
				return e
			}
			if count > 0 {
				return inputError("Ese correo ya no está disponible")
			}
			table := "public.users"
			if i.Kind == "employee" {
				table = "public.employees"
			}
			if e = tx.Exec(`UPDATE `+table+` SET email=? WHERE id=?`, item.NewEmail, i.ID).Error; e != nil {
				return e
			}
			if e = tx.Exec(`UPDATE legui_private.email_changes SET completed_at=clock_timestamp() WHERE id=?`, item.ID).Error; e != nil {
				return e
			}
			return Audit(r.Context(), tx, i.principal(), "auth.email_changed", i.ID, nil)
		}
		return nil
	})
	if e != nil {
		return e
	}
	respondWithJSON(w, 200, map[string]any{"completed": completed})
	return nil
}
