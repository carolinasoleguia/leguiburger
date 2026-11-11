package businesscare

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"leguiburger/internal/auth"
	"net/http"
	"strings"
)

func (h *Handler) closure(r *http.Request, p auth.SessionPrincipal) (any, error) {
	id := selected(r, p)
	var raw string
	e := h.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := brand(tx, p, id, false); e != nil {
			return e
		}
		return tx.Raw(`SELECT jsonb_build_object('closed_at',b.closed_at,'name',b.name,'primary',b.primary_owner_id=?::uuid,'renewals',coalesce((SELECT jsonb_agg(jsonb_build_object('id',s.id,'cancel_at_period_end',s.cancel_at_period_end,'provider',s.provider,'provider_status',m.status,'cancel_pending',coalesce(m.cancel_pending,false),'operation_state',o.state)) FROM legui_private.subscriptions s JOIN legui_private.billing_accounts a ON a.id=s.account_id LEFT JOIN legui_private.mp_subscription_mandates m ON m.subscription_id=s.id LEFT JOIN LATERAL(SELECT state FROM legui_private.mp_operations WHERE kind='cancel_subscription' AND target_id=s.id ORDER BY created_at DESC LIMIT 1)o ON true WHERE a.brand_id=b.id AND s.status<>'canceled'),'[]'))::text FROM public.brands b WHERE b.id=?`, p.ID, id).Scan(&raw).Error
	})
	return json.RawMessage(raw), e
}
func (h *Handler) closeBusiness(r *http.Request, p auth.SessionPrincipal) (any, error) {
	if e := auth.LimitSensitiveAction(h.DB, r, "business-closure", p.ID); e != nil {
		return nil, &problem{429, "Esperá antes de volver a intentar el cierre"}
	}
	var in struct {
		Password     string `json:"password"`
		Confirmation string `json:"confirmation"`
		Reason       string `json:"reason"`
	}
	if e := read(r, &in); e != nil {
		return nil, e
	}
	if in.Confirmation != "CERRAR RESTAURANTE" || len([]rune(strings.TrimSpace(in.Reason))) < 4 || len([]rune(in.Reason)) > 500 {
		return nil, invalid("Escribí CERRAR RESTAURANTE y el motivo de la baja")
	}
	id := selected(r, p)
	return h.mutate(r, p, in, func(tx *gorm.DB) (any, error) {
		if e := brand(tx, p, id, true); e != nil {
			return nil, e
		}
		if e := auth.Reauthenticate(r.Context(), tx, p, in.Password); e != nil {
			return nil, e
		}
		var prior int64
		if e := tx.Raw(`SELECT count(*) FROM legui_private.business_closures WHERE brand_id=?`, id).Scan(&prior).Error; e != nil {
			return nil, e
		}
		if prior > 0 {
			return map[string]bool{"archived": true}, nil
		}
		if e := tx.Exec(`INSERT INTO legui_private.business_closures(brand_id,actor_id,reason) VALUES(?,?,?)`, id, p.ID, strings.TrimSpace(in.Reason)).Error; e != nil {
			return nil, e
		}
		// Preserve paid periods and history. Provider cancellation has its own durable
		// journal: an archived business does not mean a remote debit was confirmed off.
		if e := tx.Exec(`UPDATE legui_private.subscriptions SET cancel_at_period_end=true WHERE account_id IN(SELECT id FROM legui_private.billing_accounts WHERE brand_id=?) AND status<>'canceled'`, id).Error; e != nil {
			return nil, e
		}
		var mandates []struct{ ID, ConnectionID, RemoteID string }
		if e := tx.Raw(`SELECT s.id::text,m.connection_id::text,m.remote_id FROM legui_private.subscriptions s JOIN legui_private.billing_accounts a ON a.id=s.account_id JOIN legui_private.mp_subscription_mandates m ON m.subscription_id=s.id WHERE a.brand_id=? AND s.status<>'canceled' AND m.status<>'cancelled' FOR UPDATE OF s,m`, id).Scan(&mandates).Error; e != nil {
			return nil, e
		}
		for _, m := range mandates {
			if e := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "mp-cancel:"+m.ID).Error; e != nil {
				return nil, e
			}
			var pending int64
			if e := tx.Raw(`SELECT count(*) FROM legui_private.mp_operations WHERE kind='cancel_subscription' AND target_id=? AND state IN ('prepared','submitting','review','completed')`, m.ID).Scan(&pending).Error; e != nil {
				return nil, e
			}
			if pending == 0 {
				digest := sha256.Sum256([]byte("cancel:" + m.ID))
				if e := tx.Exec(`INSERT INTO legui_private.mp_operations(id,connection_id,kind,target_id,resource_id,operation_key,request_digest,actor_kind,actor_id) VALUES(?,?,'cancel_subscription',?,?,?,?,?,?)`, uuid.NewString(), m.ConnectionID, m.ID, m.RemoteID, "closure:"+m.ID, hex.EncodeToString(digest[:]), p.Kind, p.ID).Error; e != nil {
					return nil, e
				}
			}
			if e := tx.Exec(`UPDATE legui_private.mp_subscription_mandates SET cancel_pending=true WHERE subscription_id=?`, m.ID).Error; e != nil {
				return nil, e
			}
		}
		if e := tx.Exec(`UPDATE public.brands SET closed_at=now(),suspended_at=coalesce(suspended_at,now()),suspension_reason='Restaurante cerrado por su responsable' WHERE id=?`, id).Error; e != nil {
			return nil, e
		}
		return map[string]any{"archived": true, "provider_cancellations_pending": len(mandates)}, auth.Audit(r.Context(), tx, p, "business.closed", id, nil)
	})
}
