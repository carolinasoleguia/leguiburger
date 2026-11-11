package businesscare

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"io"
	"leguiburger/internal/auth"
	"net/http"
	"net/mail"
	"os"
	"regexp"
	"strings"
)

type Handler struct{ DB *gorm.DB }
type problem struct {
	status  int
	message string
}

func (e *problem) Error() string { return e.message }
func invalid(s string) error     { return &problem{400, s} }
func denied() error {
	return &problem{403, "No tenés permiso para consultar o modificar este recurso"}
}

type endpoint func(*http.Request, auth.SessionPrincipal) (any, error)

func Register(mux *http.ServeMux, db *gorm.DB) {
	h := &Handler{db}
	routes := map[string]endpoint{
		"GET /api/care/support": h.tickets, "POST /api/care/support": h.createTicket,
		"GET /api/care/support/{id}": h.messages, "POST /api/care/support/{id}/messages": h.reply,
		"POST /api/care/support/{id}/close": h.closeTicket,
		"GET /api/care/backups":             h.backups, "POST /api/care/backups": h.createBackup,
		"GET /api/care/closure": h.closure, "POST /api/care/closure": h.closeBusiness,
		"POST /api/care/account-close": h.closeAccount,
		"GET /api/care/errors":         h.runtimeErrors, "POST /api/care/errors": h.reportError,
	}
	for pattern, fn := range routes {
		mux.HandleFunc(pattern, auth.AuthMiddlewareWithDB(db, h.wrap(fn)))
	}
	mux.HandleFunc("GET /api/care/backups/{id}/download", auth.AuthMiddlewareWithDB(db, h.download))
}
func (h *Handler) wrap(fn endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, _ := auth.PrincipalFromContext(r.Context())
		value, err := fn(r, p)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err != nil {
			status, message := 500, "No se pudo completar la solicitud. Reintentá en unos momentos."
			var e *problem
			if errors.As(err, &e) {
				status, message = e.status, e.message
			}
			if errors.Is(err, auth.ErrInvalidCredentials) {
				status, message = 403, "La contraseña actual no es correcta"
			}
			auth.RespondWithError(w, status, "CARE_ERROR", message)
			return
		}
		json.NewEncoder(w).Encode(value)
	}
}
func read(r *http.Request, v any) error {
	d := json.NewDecoder(io.LimitReader(r.Body, 16385))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return invalid("Revisá los datos del formulario")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return invalid("Solicitud inválida")
	}
	return nil
}
func (h *Handler) mutate(r *http.Request, p auth.SessionPrincipal, input any, fn func(*gorm.DB) (any, error)) (any, error) {
	key := r.Header.Get("Idempotency-Key")
	if !regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,160}$`).MatchString(key) {
		return nil, invalid("Falta la clave de la operación")
	}
	raw, _ := json.Marshal(input)
	digest := sha256.Sum256(append([]byte(r.Method+" "+r.URL.Path+"?brand_id="+selected(r, p)+"\n"), raw...))
	fingerprint := hex.EncodeToString(digest[:])
	var result json.RawMessage
	err := h.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := auth.LockSession(r.Context(), tx); e != nil {
			return e
		}
		if e := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "care:"+p.Kind+":"+p.ID+":"+key).Error; e != nil {
			return e
		}
		var prior struct{ Digest, Response string }
		q := tx.Raw(`SELECT digest,response::text FROM legui_private.care_receipts WHERE actor_kind=? AND actor_id=? AND operation_key=?`, p.Kind, p.ID, key).Scan(&prior)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected > 0 {
			if prior.Digest != fingerprint {
				return invalid("La clave ya se usó con otros datos")
			}
			result = json.RawMessage(prior.Response)
			return nil
		}
		output, e := fn(tx)
		if e != nil {
			return e
		}
		result, e = json.Marshal(output)
		if e != nil {
			return e
		}
		return tx.Exec(`INSERT INTO legui_private.care_receipts(actor_kind,actor_id,operation_key,digest,response) VALUES(?,?,?,?,?::jsonb)`, p.Kind, p.ID, key, fingerprint, string(result)).Error
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	return result, err
}
func brand(tx *gorm.DB, p auth.SessionPrincipal, id string, primary bool) error {
	if !p.Manager() {
		return denied()
	}
	if _, e := uuid.Parse(id); e != nil {
		return invalid("Seleccioná un restaurante")
	}
	var b struct{ ID, PrimaryOwnerID string }
	q := tx.Raw(`SELECT id::text,coalesce(primary_owner_id::text,'') primary_owner_id FROM public.brands WHERE id=? FOR UPDATE`, id).Scan(&b)
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected != 1 || (!p.Platform() && (p.BrandID != id || (primary && b.PrimaryOwnerID != p.ID))) {
		return denied()
	}
	return nil
}
func selected(r *http.Request, p auth.SessionPrincipal) string {
	if !p.Platform() {
		return p.BrandID
	}
	return r.URL.Query().Get("brand_id")
}
func (h *Handler) tickets(r *http.Request, p auth.SessionPrincipal) (any, error) {
	var raw string
	err := h.DB.WithContext(r.Context()).Raw(`SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.updated_at DESC),'[]')::text FROM (SELECT id,brand_id,subject,state,created_at,updated_at FROM legui_private.support_tickets WHERE ? OR (actor_kind=? AND actor_id=?) ORDER BY updated_at DESC LIMIT 100)t`, p.Platform(), p.Kind, p.ID).Scan(&raw).Error
	return json.RawMessage(raw), err
}
func ticket(tx *gorm.DB, p auth.SessionPrincipal, id string) error {
	if _, e := uuid.Parse(id); e != nil {
		return denied()
	}
	var ok bool
	e := tx.Raw(`SELECT EXISTS(SELECT 1 FROM legui_private.support_tickets WHERE id=? AND (? OR actor_kind=? AND actor_id=?))`, id, p.Platform(), p.Kind, p.ID).Scan(&ok).Error
	if e != nil {
		return e
	}
	if !ok {
		return denied()
	}
	return nil
}
func (h *Handler) createTicket(r *http.Request, p auth.SessionPrincipal) (any, error) {
	var in struct {
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if e := read(r, &in); e != nil {
		return nil, e
	}
	in.Subject = strings.TrimSpace(in.Subject)
	in.Body = strings.TrimSpace(in.Body)
	if len([]rune(in.Subject)) < 4 || len([]rune(in.Subject)) > 160 || len([]rune(in.Body)) < 4 || len([]rune(in.Body)) > 6000 {
		return nil, invalid("Completá un asunto de 4 a 160 caracteres y un mensaje de 4 a 6000")
	}
	return h.mutate(r, p, in, func(tx *gorm.DB) (any, error) {
		var count int64
		if e := tx.Raw(`SELECT count(*) FROM legui_private.support_tickets WHERE actor_kind=? AND actor_id=? AND created_at>now()-interval '1 hour'`, p.Kind, p.ID).Scan(&count).Error; e != nil {
			return nil, e
		}
		if count >= 10 {
			return nil, &problem{429, "Llegaste al límite de consultas por hora"}
		}
		id := uuid.NewString()
		if e := tx.Exec(`INSERT INTO legui_private.support_tickets(id,brand_id,actor_kind,actor_id,subject) VALUES(?,nullif(?,'')::uuid,?,?,?)`, id, p.BrandID, p.Kind, p.ID, in.Subject).Error; e != nil {
			return nil, e
		}
		if e := tx.Exec(`INSERT INTO legui_private.support_messages(ticket_id,actor_kind,actor_id,platform,body) VALUES(?,?,?,?,?)`, id, p.Kind, p.ID, p.Platform(), in.Body).Error; e != nil {
			return nil, e
		}
		if address := os.Getenv("SUPPORT_EMAIL"); address != "" {
			if _, e := mail.ParseAddress(address); e != nil {
				return nil, invalid("El correo de soporte requiere configuración")
			}
			if e := auth.QueueMail(tx, address, "Nueva consulta en LeguiBurger", "Hay una consulta pendiente. Revisá el centro de soporte dentro de la aplicación."); e != nil {
				return nil, e
			}
		}
		return map[string]string{"id": id}, auth.Audit(r.Context(), tx, p, "support.created", id, nil)
	})
}
func (h *Handler) messages(r *http.Request, p auth.SessionPrincipal) (any, error) {
	var result json.RawMessage
	err := h.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := ticket(tx, p, r.PathValue("id")); e != nil {
			return e
		}
		var raw string
		e := tx.Raw(`SELECT coalesce(jsonb_agg(to_jsonb(m) ORDER BY created_at,id),'[]')::text FROM (SELECT id,platform,body,created_at FROM legui_private.support_messages WHERE ticket_id=? ORDER BY created_at,id LIMIT 500)m`, r.PathValue("id")).Scan(&raw).Error
		result = json.RawMessage(raw)
		return e
	})
	return result, err
}
func (h *Handler) reply(r *http.Request, p auth.SessionPrincipal) (any, error) {
	var in struct {
		Body string `json:"body"`
	}
	if e := read(r, &in); e != nil {
		return nil, e
	}
	in.Body = strings.TrimSpace(in.Body)
	if len([]rune(in.Body)) < 4 || len([]rune(in.Body)) > 6000 {
		return nil, invalid("El mensaje debe tener de 4 a 6000 caracteres")
	}
	return h.mutate(r, p, in, func(tx *gorm.DB) (any, error) {
		id := r.PathValue("id")
		if e := ticket(tx, p, id); e != nil {
			return nil, e
		}
		if e := tx.Exec(`SELECT id FROM legui_private.support_tickets WHERE id=? FOR UPDATE`, id).Error; e != nil {
			return nil, e
		}
		var count int64
		if e := tx.Raw(`SELECT count(*) FROM legui_private.support_messages WHERE actor_kind=? AND actor_id=? AND created_at>now()-interval '1 hour'`, p.Kind, p.ID).Scan(&count).Error; e != nil {
			return nil, e
		}
		if count >= 60 {
			return nil, &problem{429, "Esperá antes de enviar más mensajes"}
		}
		if e := tx.Exec(`INSERT INTO legui_private.support_messages(ticket_id,actor_kind,actor_id,platform,body) VALUES(?,?,?,?,?)`, id, p.Kind, p.ID, p.Platform(), in.Body).Error; e != nil {
			return nil, e
		}
		if e := tx.Exec(`UPDATE legui_private.support_tickets SET state='open',updated_at=now() WHERE id=?`, id).Error; e != nil {
			return nil, e
		}
		if p.Platform() {
			var address string
			if e := tx.Raw(`SELECT u.email FROM legui_private.support_tickets t JOIN public.users u ON t.actor_kind='user' AND u.id=t.actor_id WHERE t.id=? UNION ALL SELECT e.email FROM legui_private.support_tickets t JOIN public.employees e ON t.actor_kind='employee' AND e.id=t.actor_id WHERE t.id=?`, id, id).Scan(&address).Error; e != nil {
				return nil, e
			}
			if address != "" {
				if e := auth.QueueMail(tx, address, "Respondimos tu consulta", "Tenés una respuesta en el centro de soporte de LeguiBurger. Ingresá a la aplicación para leerla."); e != nil {
					return nil, e
				}
			}
		}
		return map[string]bool{"saved": true}, auth.Audit(r.Context(), tx, p, "support.replied", id, nil)
	})
}
func (h *Handler) closeTicket(r *http.Request, p auth.SessionPrincipal) (any, error) {
	return h.mutate(r, p, struct{}{}, func(tx *gorm.DB) (any, error) {
		id := r.PathValue("id")
		if e := ticket(tx, p, id); e != nil {
			return nil, e
		}
		e := tx.Exec(`UPDATE legui_private.support_tickets SET state='closed',updated_at=now() WHERE id=?`, id).Error
		return map[string]bool{"closed": true}, e
	})
}
func (h *Handler) runtimeErrors(r *http.Request, p auth.SessionPrincipal) (any, error) {
	if !p.Platform() {
		return nil, denied()
	}
	var raw string
	e := h.DB.WithContext(r.Context()).Raw(`SELECT coalesce(jsonb_agg(to_jsonb(e) ORDER BY last_seen_at DESC),'[]')::text FROM legui_private.runtime_errors e`).Scan(&raw).Error
	return json.RawMessage(raw), e
}
func (h *Handler) reportError(r *http.Request, p auth.SessionPrincipal) (any, error) {
	var in struct {
		Code string `json:"code"`
	}
	if e := read(r, &in); e != nil {
		return nil, e
	}
	if in.Code != "render" && in.Code != "navigation" && in.Code != "unhandled" {
		return nil, invalid("Categoría inválida")
	}
	e := h.DB.WithContext(r.Context()).Exec(`INSERT INTO legui_private.auth_rate_limits(bucket,window_start,attempts) VALUES(?,date_trunc('minute',now()),1) ON CONFLICT(bucket) DO UPDATE SET attempts=CASE WHEN legui_private.auth_rate_limits.window_start<date_trunc('minute',now()) THEN 1 ELSE least(legui_private.auth_rate_limits.attempts+1,101) END,window_start=date_trunc('minute',now())`, "care-errors:"+p.ID).Error
	if e != nil {
		return nil, e
	}
	var count int
	e = h.DB.WithContext(r.Context()).Raw(`SELECT attempts FROM legui_private.auth_rate_limits WHERE bucket=? AND window_start=date_trunc('minute',now())`, "care-errors:"+p.ID).Scan(&count).Error
	if e != nil {
		return nil, e
	}
	if count > 10 {
		return nil, &problem{429, "Esperá antes de reintentar"}
	}
	return map[string]bool{"received": true}, Record(r.Context(), h.DB, "web", in.Code, "")
}
func Record(ctx context.Context, db *gorm.DB, component, code, id string) error {
	return db.WithContext(ctx).Exec(`INSERT INTO legui_private.runtime_errors(component,code,request_id) VALUES(?,?,nullif(?,'')::uuid) ON CONFLICT(component,code) DO UPDATE SET occurrences=legui_private.runtime_errors.occurrences+1,last_seen_at=now(),request_id=excluded.request_id`, component, code, id).Error
}
