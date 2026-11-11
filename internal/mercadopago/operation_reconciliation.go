package mercadopago

import (
	"context"
	"gorm.io/gorm"
)

func (h *Handler) reconcileOperation(ctx context.Context) error {
	// Never resend a command whose provider outcome is unknown. Read the remote
	// payment/mandate and match a known refund ID before completing its journal.
	if e := h.db.WithContext(ctx).Exec(`UPDATE legui_private.mp_operations SET state='review',reason='SUBMISSION_LEASE_EXPIRED' WHERE state='submitting' AND lease_until<clock_timestamp()`).Error; e != nil {
		return e
	}
	var op remoteOperation
	result := h.db.WithContext(ctx).Raw(`WITH picked AS(SELECT o.id FROM legui_private.mp_operations o JOIN legui_private.mp_connections c ON c.id=o.connection_id AND c.state='connected' AND c.environment=? WHERE o.state='review' AND (o.lease_until IS NULL OR o.lease_until<clock_timestamp()) ORDER BY o.created_at LIMIT 1 FOR UPDATE OF o SKIP LOCKED) UPDATE legui_private.mp_operations o SET lease_until=clock_timestamp()+interval '5 minutes' FROM picked WHERE o.id=picked.id RETURNING o.id::text,o.connection_id::text,o.kind,o.target_id::text,o.resource_id,coalesce(o.remote_id,'') remote_id`, h.cfg.Environment).Scan(&op)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}
	if op.Kind == "cancel_subscription" {
		state, e := h.observeMandate(ctx, op.ConnectionID, op.ResourceID)
		if e != nil {
			return e
		}
		if state == "review" {
			return nil
		}
		return h.db.WithContext(ctx).Exec(`UPDATE legui_private.mp_operations SET state='completed',remote_id=resource_id,reason=NULL,lease_until=NULL WHERE id=? AND state='review' AND EXISTS(SELECT 1 FROM legui_private.mp_subscription_mandates m WHERE m.subscription_id=mp_operations.target_id AND m.status='cancelled')`, op.ID).Error
	}
	state, e := h.observePayment(ctx, op.ConnectionID, op.ResourceID)
	if e != nil {
		return e
	}
	if state == "review" || op.RemoteID == "" {
		var c connection
		if e := h.db.WithContext(ctx).Raw(`SELECT `+connectionColumns+` FROM legui_private.mp_connections WHERE id=?`, op.ConnectionID).Scan(&c).Error; e != nil {
			return e
		}
		return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return h.review(tx, c, "", op.ResourceID, "REFUND_COMMAND_NEEDS_REVIEW", hash(op.ID))
		})
	}
	return h.db.WithContext(ctx).Exec(`UPDATE legui_private.mp_operations SET state='completed',reason=NULL,lease_until=NULL WHERE id=? AND state='review' AND EXISTS(SELECT 1 FROM legui_private.mp_payment_allocations a WHERE a.provider_payment_id=mp_operations.target_id AND a.component='refund:'||mp_operations.remote_id)`, op.ID).Error
}
