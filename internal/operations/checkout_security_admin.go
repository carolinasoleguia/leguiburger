package operations

import (
	"encoding/json"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

func (h *Handler) abuseIncidents(r *http.Request, p Principal) (Result, error) {
	if !p.Platform() {
		return Result{}, forbidden()
	}
	limit, offset, e := pagination(r)
	if e != nil {
		return Result{}, e
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "open"
	}
	if !isOneOf(status, "open", "blocked", "all") {
		return Result{}, badInput("Filtro de incidentes inválido")
	}
	return resultJSON(h.repo.JSON(r.Context(), `WITH filtered AS (
 SELECT i.id,i.reason,i.occurrences,i.evidence,i.first_seen_at,i.last_seen_at,i.resolved_at,i.resolution_note,s.label,left(s.subject_hash,12) reference,s.blocked_until,(s.blocked_until>clock_timestamp()) is_blocked,coalesce(b.name,'Plataforma') business,coalesce(nullif(t.display_name,''),t.subdomain,'Varios locales') location
 FROM legui_private.abuse_incidents i JOIN legui_private.abuse_subjects s USING(subject_hash) LEFT JOIN public.tenants t ON t.id=i.tenant_id LEFT JOIN public.brands b ON b.id=t.brand_id
 WHERE (?='all' OR ?='open' AND i.resolved_at IS NULL OR ?='blocked' AND s.blocked_until>clock_timestamp()))
 SELECT jsonb_build_object('items',coalesce((SELECT jsonb_agg(to_jsonb(v)) FROM(SELECT * FROM filtered ORDER BY last_seen_at DESC,id LIMIT ? OFFSET ?)v),'[]'::jsonb),'total',(SELECT count(*) FROM filtered),'pending',(SELECT count(*) FROM legui_private.abuse_incidents WHERE resolved_at IS NULL),'blocked',(SELECT count(*) FROM legui_private.abuse_subjects WHERE blocked_until>clock_timestamp()),'verification_configured',?::boolean)::text payload`, status, status, status, limit, offset, h.security.configured()))
}
func (h *Handler) abuseAction(r *http.Request, p Principal) (Result, error) {
	if !p.Platform() {
		return Result{}, forbidden()
	}
	id, e := resourceID(r)
	if e != nil {
		return Result{}, e
	}
	in, e := decode[struct {
		Action  string `json:"action"`
		Minutes int    `json:"minutes"`
		Note    string `json:"note"`
	}](r)
	if e != nil {
		return Result{}, e
	}
	in.Note = strings.TrimSpace(in.Note)
	if !isOneOf(in.Action, "block", "unblock", "resolve") || len([]rune(in.Note)) < 10 || len([]rune(in.Note)) > 500 || in.Action == "block" && !isOneOfInt(in.Minutes, 15, 60, 1440) {
		return Result{}, badInput("Elegí una acción y escribí un motivo de 10 a 500 caracteres")
	}
	return h.mutate(r, p, Scope{}, "platform.security."+in.Action, in, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		// Subject first, then incident: same lock order used by the signal writer.
		var subject string
		q := tx.Raw(`SELECT subject_hash FROM legui_private.abuse_incidents WHERE id=?`, id).Scan(&subject)
		if q.Error != nil {
			return nil, 0, q.Error
		}
		if q.RowsAffected != 1 {
			return nil, 0, notFound()
		}
		if e := tx.Exec(`SELECT subject_hash FROM legui_private.abuse_subjects WHERE subject_hash=? FOR UPDATE`, subject).Error; e != nil {
			return nil, 0, e
		}
		switch in.Action {
		case "block":
			if e := tx.Exec(`UPDATE legui_private.abuse_subjects SET blocked_until=clock_timestamp()+make_interval(mins=>?::int),released_until=NULL WHERE subject_hash=?`, in.Minutes, subject).Error; e != nil {
				return nil, 0, e
			}
			if e := tx.Exec(`UPDATE legui_private.abuse_incidents SET resolved_at=NULL,resolution_note=?,last_seen_at=clock_timestamp() WHERE id=?`, in.Note, id).Error; e != nil {
				return nil, 0, e
			}
		case "unblock":
			if e := tx.Exec(`UPDATE legui_private.abuse_subjects SET blocked_until=NULL,released_until=clock_timestamp()+interval '5 minutes',window_start=clock_timestamp(),new_sessions=0,invalid_requests=0,orders=0 WHERE subject_hash=?`, subject).Error; e != nil {
				return nil, 0, e
			}
			if e := tx.Exec(`UPDATE legui_private.abuse_incidents SET resolved_at=clock_timestamp(),resolution_note=? WHERE subject_hash=? AND resolved_at IS NULL`, in.Note, subject).Error; e != nil {
				return nil, 0, e
			}
		case "resolve":
			if e := tx.Exec(`UPDATE legui_private.abuse_incidents SET resolved_at=clock_timestamp(),resolution_note=? WHERE id=?`, in.Note, id).Error; e != nil {
				return nil, 0, e
			}
		}
		// Immutable actor-attributed audit, including the reason, in the same transaction.
		if e := tx.Exec(`INSERT INTO legui_private.security_events(actor_kind,actor_id,event_name,target_id,facts) VALUES('user',?,?,?,jsonb_build_object('note',?::text,'minutes',?::int))`, p.ID, "checkout_abuse."+in.Action, id, in.Note, in.Minutes).Error; e != nil {
			return nil, 0, e
		}
		return json.RawMessage(`{"ok":true}`), 200, nil
	})
}
func isOneOfInt(v int, values ...int) bool {
	for _, x := range values {
		if x == v {
			return true
		}
	}
	return false
}

// Runs in the existing worker. Never retain provider tokens, session digests or
// incident evidence indefinitely; preserve the separate administrator audit.
func CleanupCheckoutSecurity(db *gorm.DB) error {
	return db.Exec(`DELETE FROM legui_private.abuse_clearances WHERE expires_at<clock_timestamp()-interval '1 day'; DELETE FROM legui_private.abuse_verified_tokens WHERE expires_at<clock_timestamp(); DELETE FROM legui_private.abuse_incidents WHERE last_seen_at<clock_timestamp()-interval '30 days' AND NOT EXISTS(SELECT 1 FROM legui_private.abuse_subjects s WHERE s.subject_hash=abuse_incidents.subject_hash AND s.blocked_until>clock_timestamp()); DELETE FROM legui_private.abuse_subjects WHERE last_seen_at<clock_timestamp()-interval '30 days' AND (blocked_until IS NULL OR blocked_until<clock_timestamp()) AND NOT EXISTS(SELECT 1 FROM legui_private.abuse_incidents i WHERE i.subject_hash=abuse_subjects.subject_hash)`).Error
}
