package shopperidentity

import (
	"context"
	"encoding/json"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"io"
	"leguiburger/internal/auth"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"time"
)

type Handler struct{ db *gorm.DB }
type problem struct {
	status  int
	message string
}

func (e *problem) Error() string { return e.message }
func Register(mux *http.ServeMux, db *gorm.DB) {
	h := &Handler{db}
	mux.HandleFunc("POST /api/public/shopper/logout", func(w http.ResponseWriter, r *http.Request) { h.handle(w, r, "logout") })
	for _, action := range []string{"register", "login", "logout", "me", "verify", "resend", "forgot", "reset", "password"} {
		a := action
		method := "POST"
		if a == "me" {
			method = "GET"
		}
		mux.HandleFunc(method+" /api/shopper/"+a, func(w http.ResponseWriter, r *http.Request) { h.handle(w, r, a) })
	}
}
func (h *Handler) handle(w http.ResponseWriter, r *http.Request, action string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	if !Origin(r) {
		auth.RespondWithError(w, 403, "ORIGIN", "Origen no permitido")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	result, e := h.run(w, r, action)
	if e != nil {
		var p *problem
		if errors.As(e, &p) {
			auth.RespondWithError(w, p.status, "SHOPPER_ERROR", p.message)
		} else if errors.Is(e, ErrSession) {
			auth.RespondWithError(w, 401, "UNAUTHORIZED", "Ingresá a tu cuenta de comprador")
		} else {
			auth.RespondWithError(w, 503, "SHOPPER_UNAVAILABLE", "No pudimos completar la solicitud")
		}
		return
	}
	json.NewEncoder(w).Encode(result)
}
func (h *Handler) rate(r *http.Request, scope string, max int) error {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	var count int
	e := h.db.WithContext(r.Context()).Raw(`INSERT INTO legui_private.auth_rate_limits(bucket,attempts) VALUES(?,1) ON CONFLICT(bucket) DO UPDATE SET attempts=CASE WHEN auth_rate_limits.window_start<clock_timestamp()-interval '15 minutes' THEN 1 ELSE auth_rate_limits.attempts+1 END,window_start=CASE WHEN auth_rate_limits.window_start<clock_timestamp()-interval '15 minutes' THEN clock_timestamp() ELSE auth_rate_limits.window_start END RETURNING attempts`, "shopper:"+scope+":"+Digest(ip)).Scan(&count).Error
	if e != nil {
		return e
	}
	if count > max {
		return &problem{429, "Esperá unos minutos antes de reintentar"}
	}
	return nil
}
func (h *Handler) issue(w http.ResponseWriter, r *http.Request, id string) error {
	raw := Random()
	e := h.db.WithContext(r.Context()).Exec(`INSERT INTO legui_private.shopper_sessions(digest,shopper_id) VALUES(?,?)`, Digest(raw), id).Error
	if e == nil {
		SetCookie(w, raw)
	}
	return e
}
func (h *Handler) queue(r *http.Request, id, email, purpose string) error {
	if !auth.MailConfigured() {
		return nil
	}
	site := strings.TrimRight(os.Getenv("PUBLIC_SITE_URL"), "/")
	u, e := url.Parse(site)
	if e != nil || u.Host == "" || (u.Scheme != "https" && os.Getenv("APP_ENV") == "production") {
		return &problem{503, "El correo de cuentas todavía no está configurado"}
	}
	raw := Random()
	return h.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec(`INSERT INTO legui_private.shopper_actions(digest,shopper_id,purpose) VALUES(?,?,?)`, Digest(raw), id, purpose).Error; e != nil {
			return e
		}
		return auth.QueueShopperMail(tx, email, "Tu cuenta de comprador de LeguiBurger", site+"/mi-cuenta?accion="+purpose+"#token="+raw)
	})
}
func (h *Handler) run(w http.ResponseWriter, r *http.Request, action string) (any, error) {
	ctx := r.Context()
	if action == "me" {
		return Current(ctx, h.db, r)
	}
	if action == "logout" {
		if c, e := r.Cookie(Cookie); e == nil {
			if e = h.db.WithContext(ctx).Exec(`UPDATE legui_private.shopper_sessions SET revoked_at=clock_timestamp() WHERE digest=? AND revoked_at IS NULL`, Digest(c.Value)).Error; e != nil {
				return nil, e
			}
		}
		for _, c := range r.Cookies() {
			if strings.HasPrefix(c.Name, "legui_buyer_") && len(c.Name) == 44 {
				http.SetCookie(w, &http.Cookie{Name: c.Name, Value: "", Path: "/api/public", HttpOnly: true, Secure: os.Getenv("APP_ENV") == "production" || strings.HasPrefix(os.Getenv("PUBLIC_SITE_URL"), "https://"), SameSite: http.SameSiteLaxMode, MaxAge: -1})
			}
		}
		SetCookie(w, "")
		return map[string]bool{"ok": true}, nil
	}
	var in struct {
		Name            string `json:"name"`
		Email           string `json:"email"`
		Password        string `json:"password"`
		CurrentPassword string `json:"current_password"`
		Token           string `json:"token"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&in) != nil {
		return nil, &problem{400, "Revisá los datos ingresados"}
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return nil, &problem{400, "Solicitud inválida"}
	}
	if e := h.rate(r, action, 20); e != nil {
		return nil, e
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.TrimSpace(in.Name)
	if action == "verify" || action == "reset" {
		if len(in.Token) != 43 {
			return nil, &problem{400, "El enlace no es válido o venció"}
		}
		if action == "reset" && !auth.ValidPassword(in.Password) {
			return nil, &problem{400, "Usá una contraseña de al menos 12 caracteres"}
		}
		e := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var id string
			q := tx.Raw(`UPDATE legui_private.shopper_actions SET used_at=clock_timestamp() WHERE digest=? AND purpose=? AND used_at IS NULL AND expires_at>clock_timestamp() RETURNING shopper_id::text`, Digest(in.Token), map[string]string{"verify": "verify", "reset": "reset"}[action]).Scan(&id)
			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected != 1 {
				return &problem{400, "El enlace no es válido o venció"}
			}
			if action == "verify" {
				return tx.Exec(`UPDATE legui_private.shoppers SET email_verified=true WHERE id=? AND active`, id).Error
			}
			hash, e := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
			if e != nil {
				return e
			}
			if e = tx.Exec(`UPDATE legui_private.shoppers SET password_hash=? WHERE id=? AND active`, string(hash), id).Error; e != nil {
				return e
			}
			if e := tx.Exec(`UPDATE legui_private.shopper_actions SET used_at=clock_timestamp() WHERE shopper_id=? AND purpose='reset' AND used_at IS NULL`, id).Error; e != nil {
				return e
			}
			return tx.Exec(`UPDATE legui_private.shopper_sessions SET revoked_at=clock_timestamp() WHERE shopper_id=? AND revoked_at IS NULL`, id).Error
		})
		return map[string]bool{"ok": e == nil}, e
	}
	if action == "resend" || action == "password" {
		p, e := Current(ctx, h.db, r)
		if e != nil {
			return nil, e
		}
		if action == "resend" {
			if !auth.MailConfigured() {
				return nil, &problem{409, "El envío de correo todavía no está configurado"}
			}
			return map[string]bool{"queued": true}, h.queue(r, p.ID, p.Email, "verify")
		}
		if !auth.ValidPassword(in.Password) {
			return nil, &problem{400, "Usá una contraseña segura de al menos 12 caracteres"}
		}
		var hash string
		h.db.WithContext(ctx).Raw(`SELECT password_hash FROM legui_private.shoppers WHERE id=?`, p.ID).Scan(&hash)
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.CurrentPassword)) != nil {
			return nil, &problem{400, "La contraseña actual no es válida"}
		}
		newHash, e := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if e != nil {
			return nil, e
		}
		e = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if e := tx.Exec(`UPDATE legui_private.shoppers SET password_hash=? WHERE id=?`, string(newHash), p.ID).Error; e != nil {
				return e
			}
			if e := tx.Exec(`UPDATE legui_private.shopper_actions SET used_at=clock_timestamp() WHERE shopper_id=? AND purpose='reset' AND used_at IS NULL`, p.ID).Error; e != nil {
				return e
			}
			return tx.Exec(`UPDATE legui_private.shopper_sessions SET revoked_at=clock_timestamp() WHERE shopper_id=?`, p.ID).Error
		})
		if e == nil {
			e = h.issue(w, r, p.ID)
		}
		return map[string]bool{"ok": e == nil}, e
	}
	email, e := mail.ParseAddress(in.Email)
	if e != nil || email.Address != in.Email || len(in.Email) > 254 {
		return nil, &problem{400, "Ingresá un correo válido"}
	}
	if action == "forgot" {
		var p Profile
		h.db.WithContext(ctx).Raw(`SELECT id::text,email FROM legui_private.shoppers WHERE email=? AND active`, in.Email).Scan(&p)
		if p.ID != "" {
			if e := h.queue(r, p.ID, p.Email, "reset"); e != nil {
				return nil, e
			}
		}
		return map[string]string{"message": "Si existe una cuenta y el correo está habilitado, recibirás un enlace de recuperación."}, nil
	}
	if action == "register" {
		if !auth.ValidPassword(in.Password) || len([]rune(in.Name)) < 2 || len([]rune(in.Name)) > 100 {
			return nil, &problem{400, "Ingresá tu nombre y una contraseña de al menos 12 caracteres"}
		}
		hash, e := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if e != nil {
			return nil, e
		}
		var id string
		q := h.db.WithContext(ctx).Raw(`INSERT INTO legui_private.shoppers(name,email,password_hash) VALUES(?,?,?) ON CONFLICT(email) DO NOTHING RETURNING id::text`, in.Name, in.Email, string(hash)).Scan(&id)
		if q.Error != nil {
			return nil, q.Error
		}
		if q.RowsAffected != 1 {
			return nil, &problem{409, "No pudimos crear una cuenta con esos datos. Probá ingresar o recuperar la contraseña"}
		}
		if e = h.issue(w, r, id); e != nil {
			return nil, e
		}
		_ = h.queue(r, id, in.Email, "verify")
		return Profile{ID: id, Name: in.Name, Email: in.Email}, nil
	}
	var row struct {
		Profile
		PasswordHash string
	}
	q := h.db.WithContext(ctx).Raw(`SELECT id::text,name,email,email_verified,phone,default_address address,password_hash FROM legui_private.shoppers WHERE email=? AND active`, in.Email).Scan(&row)
	if q.Error != nil {
		return nil, q.Error
	}
	hash := row.PasswordHash
	if hash == "" {
		hash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil || row.ID == "" {
		return nil, &problem{401, "Correo o contraseña incorrectos"}
	}
	if e = h.issue(w, r, row.ID); e != nil {
		return nil, e
	}
	return row.Profile, nil
}
