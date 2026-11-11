package mercadopago

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"leguiburger/internal/auth"
	"net/http"
	"time"
)

type remoteOperation struct {
	ID            string
	ConnectionID  string
	Kind          string
	TargetID      string
	ResourceID    string
	Amount        string
	State         string
	RequestDigest string
	RemoteID      string
}

func (h *Handler) refund(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if e := manager(p); e != nil {
		return e
	}
	id, e := routeID(r)
	if e != nil {
		return e
	}
	operationKey, e := key(r)
	if e != nil {
		return e
	}
	var input struct {
		Amount string `json:"amount"`
	}
	if e = decode(r, &input); e != nil {
		return e
	}
	if !positiveMoney(input.Amount) {
		return bad("Importe de devolución inválido")
	}
	var op remoteOperation
	digest := hash(id + ":" + input.Amount)
	e = h.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := auth.LockSession(r.Context(), tx); e != nil {
			return e
		}
		if e := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "mp-refund:"+id).Error; e != nil {
			return e
		}
		prior := tx.Raw(`SELECT id::text,connection_id::text,kind,target_id::text,resource_id,amount::text,state,request_digest,coalesce(remote_id,'') remote_id FROM legui_private.mp_operations WHERE actor_kind=? AND actor_id=? AND kind='refund' AND operation_key=?`, p.Kind, p.ID, operationKey).Scan(&op)
		if prior.Error != nil {
			return prior.Error
		}
		if prior.RowsAffected > 0 {
			if op.RequestDigest != digest {
				return conflict("La clave ya fue usada con otros datos")
			}
			return nil
		}
		var payment struct {
			ConnectionID string
			PaymentID    string
			BrandID      string
			Remaining    string
			Scope        string
			Environment  string
			Allocated    bool
		}
		result := tx.Raw(`SELECT pp.connection_id::text,pp.payment_id,coalesce(co.brand_id,c.brand_id)::text brand_id,(pp.amount-pp.refunded_amount)::text remaining,c.scope,pp.environment,EXISTS(SELECT 1 FROM legui_private.mp_payment_allocations a WHERE a.provider_payment_id=pp.id AND a.component='charge') allocated FROM legui_private.mp_provider_payments pp JOIN legui_private.mp_connections c ON c.id=pp.connection_id LEFT JOIN legui_private.mp_checkouts co ON co.id=pp.checkout_id WHERE pp.id=? FOR UPDATE OF pp`, id).Scan(&payment)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return missing()
		}
		if !p.Platform() && (payment.Scope == "platform" || payment.BrandID != p.BrandID) {
			return denied()
		}
		if !payment.Allocated {
			return conflict("El pago requiere conciliación antes de devolverlo")
		}
		if e := h.enabled(payment.Environment); e != nil {
			return e
		}
		var reserved string
		if e := tx.Raw(`SELECT coalesce(sum(amount),0)::text FROM legui_private.mp_operations WHERE kind='refund' AND target_id=? AND state IN ('prepared','submitting','review')`, id).Scan(&reserved).Error; e != nil {
			return e
		}
		available, e := subtract(payment.Remaining, reserved)
		if e != nil {
			return conflict("Hay una devolución pendiente de confirmar")
		}
		requested, _ := money(input.Amount)
		remaining, _ := money(available)
		if requested.Cmp(remaining) > 0 {
			return bad("La devolución supera el saldo disponible")
		}
		op = remoteOperation{ID: uuid.NewString(), ConnectionID: payment.ConnectionID, Kind: "refund", TargetID: id, ResourceID: payment.PaymentID, Amount: input.Amount, State: "prepared", RequestDigest: digest}
		if e := tx.Exec(`INSERT INTO legui_private.mp_operations(id,connection_id,kind,target_id,resource_id,amount,operation_key,request_digest,actor_kind,actor_id) VALUES(?,?,?,?,?,?::numeric,?,?,?,?)`, op.ID, op.ConnectionID, op.Kind, op.TargetID, op.ResourceID, op.Amount, operationKey, digest, p.Kind, p.ID).Error; e != nil {
			return e
		}
		return auth.Audit(r.Context(), tx, p, "payments.refund_requested", op.ID, nil)
	})
	if e != nil {
		return e
	}
	return h.submitOperation(w, r, p, op)
}
func (h *Handler) cancelRenewal(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if e := manager(p); e != nil {
		return e
	}
	id, e := routeID(r)
	if e != nil {
		return e
	}
	operationKey, e := key(r)
	if e != nil {
		return e
	}
	var input struct{}
	if e = decode(r, &input); e != nil {
		return e
	}
	var op remoteOperation
	digest := hash("cancel:" + id)
	e = h.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := auth.LockSession(r.Context(), tx); e != nil {
			return e
		}
		if e := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "mp-cancel:"+id).Error; e != nil {
			return e
		}
		prior := tx.Raw(`SELECT id::text,connection_id::text,kind,target_id::text,resource_id,coalesce(amount::text,'0') amount,state,request_digest,coalesce(remote_id,'') remote_id FROM legui_private.mp_operations WHERE actor_kind=? AND actor_id=? AND kind='cancel_subscription' AND operation_key=?`, p.Kind, p.ID, operationKey).Scan(&op)
		if prior.Error != nil {
			return prior.Error
		}
		if prior.RowsAffected > 0 {
			if op.RequestDigest != digest {
				return conflict("La clave ya fue usada con otros datos")
			}
			return nil
		}
		var mandate struct {
			ConnectionID string
			RemoteID     string
			BrandID      string
			Environment  string
		}
		result := tx.Raw(`SELECT m.connection_id::text,m.remote_id,a.brand_id::text,m.environment FROM legui_private.mp_subscription_mandates m JOIN legui_private.subscriptions s ON s.id=m.subscription_id JOIN legui_private.billing_accounts a ON a.id=s.account_id WHERE m.subscription_id=? AND s.status<>'canceled' FOR UPDATE OF s,m`, id).Scan(&mandate)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return missing()
		}
		if !p.Platform() && mandate.BrandID != p.BrandID {
			return denied()
		}
		if e := h.enabled(mandate.Environment); e != nil {
			return e
		}
		op = remoteOperation{ID: uuid.NewString(), ConnectionID: mandate.ConnectionID, Kind: "cancel_subscription", TargetID: id, ResourceID: mandate.RemoteID, State: "prepared", RequestDigest: digest}
		if e := tx.Exec(`INSERT INTO legui_private.mp_operations(id,connection_id,kind,target_id,resource_id,operation_key,request_digest,actor_kind,actor_id) VALUES(?,?,?,?,?,?,?,?,?)`, op.ID, op.ConnectionID, op.Kind, id, op.ResourceID, operationKey, digest, p.Kind, p.ID).Error; e != nil {
			return e
		}
		if e := tx.Exec(`UPDATE legui_private.mp_subscription_mandates SET cancel_pending=true WHERE subscription_id=?`, id).Error; e != nil {
			return e
		}
		return auth.Audit(r.Context(), tx, p, "payments.cancel_renewal_requested", op.ID, nil)
	})
	if e != nil {
		return e
	}
	return h.submitOperation(w, r, p, op)
}
func (h *Handler) submitOperation(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal, op remoteOperation) error {
	if op.Kind == "refund" {
		if _, e := h.connection(r, p, op.ConnectionID); e != nil {
			return e
		}
	} else {
		var brand string
		if e := h.db.WithContext(r.Context()).Raw(`SELECT a.brand_id::text FROM legui_private.subscriptions s JOIN legui_private.billing_accounts a ON a.id=s.account_id WHERE s.id=?`, op.TargetID).Scan(&brand).Error; e != nil {
			return e
		}
		if brand == "" || !p.Platform() && brand != p.BrandID {
			return denied()
		}
	}
	if op.State != "prepared" {
		respond(w, 200, map[string]string{"id": op.ID, "state": op.State})
		return nil
	}
	c, token, e := h.account(r.Context(), op.ConnectionID)
	if e != nil {
		return e
	}
	claim := uuid.NewString()
	result := h.db.WithContext(r.Context()).Exec(`UPDATE legui_private.mp_operations SET state='submitting',claim=?,lease_until=clock_timestamp()+interval '1 minute' WHERE id=? AND state='prepared'`, claim, op.ID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return conflict("La operación ya se está procesando")
	}
	method, path := "POST", "/v1/payments/"+op.ResourceID+"/refunds"
	payload := map[string]any{"amount": json.Number(op.Amount)}
	if op.Kind == "cancel_subscription" {
		method = "PUT"
		path = "/preapproval/" + op.ResourceID
		payload = map[string]any{"status": "cancelled"}
	}
	remote, e := h.provider.call(r.Context(), method, path, token, op.ID, payload)
	if e != nil {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
		defer cancel()
		_ = h.db.WithContext(cleanup).Transaction(func(tx *gorm.DB) error {
			if e := tx.Exec(`UPDATE legui_private.mp_operations SET state='review',reason='PROVIDER_OUTCOME_UNKNOWN' WHERE id=? AND claim=?`, op.ID, claim).Error; e != nil {
				return e
			}
			return h.review(tx, c, "", op.ResourceID, "OPERATION_OUTCOME_UNKNOWN", hash(op.ID))
		})
		respond(w, 202, map[string]string{"id": op.ID, "state": "review"})
		return nil
	}
	remoteID := text(remote["id"])
	if op.Kind == "refund" {
		if !resourcePattern.MatchString(remoteID) || !sameMoney(text(remote["amount"]), op.Amount) || text(remote["status"]) != "approved" {
			_ = h.db.WithContext(r.Context()).Exec(`UPDATE legui_private.mp_operations SET state='review',reason='REFUND_UNCONFIRMED' WHERE id=? AND claim=?`, op.ID, claim).Error
			respond(w, 202, map[string]string{"id": op.ID, "state": "review"})
			return nil
		}
		var state string
		state, e = h.observePayment(r.Context(), c.ID, op.ResourceID)
		if e == nil && state == "review" {
			e = conflict("La devolución requiere conciliación")
		}
		if e == nil {
			var count int64
			e = h.db.WithContext(r.Context()).Raw(`SELECT count(*) FROM legui_private.mp_payment_allocations WHERE provider_payment_id=? AND component=?`, op.TargetID, "refund:"+remoteID).Scan(&count).Error
			if e == nil && count != 1 {
				e = conflict("La devolución todavía no se reflejó en el pago")
			}
		}
	} else {
		if text(remote["id"]) != op.ResourceID || text(remote["status"]) != "cancelled" {
			e = conflict("La baja no pudo confirmarse")
		} else {
			var state string
			state, e = h.observeMandate(r.Context(), c.ID, op.ResourceID)
			if e == nil && state == "review" {
				e = conflict("La baja necesita revisión")
			}
		}
	}
	if e != nil {
		_ = h.db.WithContext(r.Context()).Exec(`UPDATE legui_private.mp_operations SET state='review',remote_id=?,reason='LOCAL_RECONCILIATION_PENDING' WHERE id=? AND claim=?`, remoteID, op.ID, claim).Error
		respond(w, 202, map[string]string{"id": op.ID, "state": "review"})
		return nil
	}
	if e = h.db.WithContext(r.Context()).Exec(`UPDATE legui_private.mp_operations SET state='completed',remote_id=?,lease_until=NULL WHERE id=? AND claim=?`, remoteID, op.ID, claim).Error; e != nil {
		return e
	}
	respond(w, 200, map[string]string{"id": op.ID, "state": "completed"})
	return nil
}
