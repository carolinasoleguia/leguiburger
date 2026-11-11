package mercadopago

import (
	"context"
	"leguiburger/internal/auth"
	"net/http"
)

// Only a durable command created by the reauthenticated closure transaction can
// enter here. It uses the same provider journal, lease and reconciliation path.
func (h *Handler) submitClosureCancellation(ctx context.Context) error {
	if e := h.enabled(h.cfg.Environment); e != nil {
		return e
	}
	var row struct{ ID, BrandID, ActorID string }
	q := h.db.WithContext(ctx).Raw(`SELECT o.id::text,a.brand_id::text,o.actor_id::text FROM legui_private.mp_operations o JOIN legui_private.subscriptions s ON s.id=o.target_id JOIN legui_private.billing_accounts a ON a.id=s.account_id JOIN legui_private.business_closures c ON c.brand_id=a.brand_id JOIN legui_private.mp_connections mc ON mc.id=o.connection_id WHERE o.kind='cancel_subscription' AND o.state='prepared' AND mc.environment=? ORDER BY o.created_at LIMIT 1`, h.cfg.Environment).Scan(&row)
	if q.Error != nil || q.RowsAffected == 0 {
		return q.Error
	}
	var op remoteOperation
	if e := h.db.WithContext(ctx).Raw(`SELECT id::text,connection_id::text,kind,target_id::text,resource_id,state FROM legui_private.mp_operations WHERE id=?`, row.ID).Scan(&op).Error; e != nil {
		return e
	}
	r, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://worker.invalid/", nil)
	return h.submitOperation(discardResponse{header: http.Header{}}, r, auth.SessionPrincipal{Kind: "user", ID: row.ActorID, Role: "admin", BrandID: row.BrandID}, op)
}

type discardResponse struct{ header http.Header }

func (w discardResponse) Header() http.Header       { return w.header }
func (discardResponse) Write(p []byte) (int, error) { return len(p), nil }
func (discardResponse) WriteHeader(int)             {}
