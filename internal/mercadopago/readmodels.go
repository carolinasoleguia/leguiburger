package mercadopago

import (
	"gorm.io/gorm"
	"leguiburger/internal/auth"
	"net/http"
)

func (h *Handler) listCheckouts(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if !p.Manager() && !(p.Kind == "employee" && p.Role == "cashier") {
		return denied()
	}
	limit, offset, e := page(r)
	if e != nil {
		return e
	}
	var rows []map[string]any
	e = h.db.WithContext(r.Context()).Raw(`SELECT id,purpose,brand_id,tenant_id,order_id,invoice_id,subscription_id,amount,currency,environment,state,checkout_url,reason,created_at,expires_at FROM legui_private.mp_checkouts WHERE (? OR brand_id=nullif(?,'')::uuid) AND (? OR purpose='order' AND tenant_id=nullif(?,'')::uuid) ORDER BY created_at DESC LIMIT ? OFFSET ?`, p.Platform(), p.BrandID, p.Manager(), p.TenantID, limit, offset).Scan(&rows).Error
	if e != nil {
		return e
	}
	respond(w, 200, map[string]any{"items": rows, "limit": limit, "offset": offset})
	return nil
}
func (h *Handler) transactions(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if e := manager(p); e != nil {
		return e
	}
	limit, offset, e := page(r)
	if e != nil {
		return e
	}
	var rows []map[string]any
	e = h.db.WithContext(r.Context()).Raw(`SELECT pp.id,pp.payment_id,pp.environment,pp.provider_status,pp.provider_detail,pp.currency,pp.amount,pp.refunded_amount,pp.fee_amount,pp.net_received,pp.available_at,pp.paid_at,pp.updated_at,pp.connection_id,c.scope,co.purpose,co.order_id,co.invoice_id,co.subscription_id,co.brand_id,co.tenant_id,EXISTS(SELECT 1 FROM legui_private.mp_payment_allocations a WHERE a.provider_payment_id=pp.id AND a.component='charge') AS allocated,EXISTS(SELECT 1 FROM legui_private.mp_reviews rv WHERE rv.connection_id=pp.connection_id AND rv.environment=pp.environment AND rv.resource_id=pp.payment_id AND rv.resolved_at IS NULL) AS under_review FROM legui_private.mp_provider_payments pp JOIN legui_private.mp_connections c ON c.id=pp.connection_id LEFT JOIN legui_private.mp_checkouts co ON co.id=pp.checkout_id WHERE (? OR co.brand_id=nullif(?,'')::uuid) ORDER BY pp.updated_at DESC LIMIT ? OFFSET ?`, p.Platform(), p.BrandID, limit, offset).Scan(&rows).Error
	if e != nil {
		return e
	}
	respond(w, 200, map[string]any{"items": rows, "limit": limit, "offset": offset})
	return nil
}
func (h *Handler) reviews(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if e := manager(p); e != nil {
		return e
	}
	limit, offset, e := page(r)
	if e != nil {
		return e
	}
	var rows []map[string]any
	e = h.db.WithContext(r.Context()).Raw(`SELECT rv.id,rv.connection_id,rv.checkout_id,rv.environment,rv.resource_id,rv.reason,rv.occurrences,rv.first_seen_at,rv.last_seen_at,rv.resolved_at,rv.resolution,rv.resolution_note,c.scope,co.brand_id,co.tenant_id FROM legui_private.mp_reviews rv JOIN legui_private.mp_connections c ON c.id=rv.connection_id LEFT JOIN legui_private.mp_checkouts co ON co.id=rv.checkout_id WHERE (? OR co.brand_id=nullif(?,'')::uuid OR c.scope='business' AND c.brand_id=nullif(?,'')::uuid) ORDER BY rv.resolved_at NULLS FIRST,rv.last_seen_at DESC LIMIT ? OFFSET ?`, p.Platform(), p.BrandID, p.BrandID, limit, offset).Scan(&rows).Error
	if e != nil {
		return e
	}
	respond(w, 200, map[string]any{"items": rows, "limit": limit, "offset": offset})
	return nil
}
func (h *Handler) resolveReview(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if e := manager(p); e != nil {
		return e
	}
	id, e := routeID(r)
	if e != nil {
		return e
	}
	var input struct {
		Resolution string `json:"resolution"`
		Note       string `json:"note"`
	}
	if e = decode(r, &input); e != nil {
		return e
	}
	if !oneOf(input.Resolution, "reprocess", "acknowledged") || len(input.Note) < 10 || len(input.Note) > 500 {
		return bad("Indicá una resolución y un motivo de 10 a 500 caracteres")
	}
	var review struct {
		ConnectionID string
		ResourceID   string
		CheckoutID   string
		Scope        string
		BrandID      string
		Purpose      string
		Environment  string
		RemoteID     string
	}
	e = h.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := auth.LockSession(r.Context(), tx); e != nil {
			return e
		}
		result := tx.Raw(`SELECT rv.connection_id::text,rv.resource_id,coalesce(rv.checkout_id::text,'') checkout_id,c.scope,coalesce(co.brand_id,c.brand_id)::text brand_id,co.purpose,rv.environment,coalesce(co.remote_id,'') remote_id FROM legui_private.mp_reviews rv JOIN legui_private.mp_connections c ON c.id=rv.connection_id LEFT JOIN legui_private.mp_checkouts co ON co.id=rv.checkout_id WHERE rv.id=? AND rv.resolved_at IS NULL FOR UPDATE OF rv`, id).Scan(&review)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return missing()
		}
		if !p.Platform() && review.BrandID != p.BrandID {
			return denied()
		}
		if e := tx.Exec(`UPDATE legui_private.mp_reviews SET resolved_at=clock_timestamp(),resolved_by=?,resolution=?,resolution_note=? WHERE id=? AND resolved_at IS NULL`, p.ID, input.Resolution, input.Note, id).Error; e != nil {
			return e
		}
		if input.Resolution == "reprocess" {
			if review.CheckoutID != "" && review.ResourceID == review.CheckoutID {
				if e := tx.Exec(`UPDATE legui_private.mp_checkouts SET reconcile_at=clock_timestamp() WHERE id=?`, review.CheckoutID).Error; e != nil {
					return e
				}
				return auth.Audit(r.Context(), tx, p, "payments.review_resolved", id, map[string]string{"resolution": input.Resolution, "note": input.Note})
			}
			topic := "payment"
			if review.Purpose == "subscription" && review.ResourceID == review.RemoteID {
				topic = "subscription_preapproval"
			}
			if e := tx.Exec(`INSERT INTO legui_private.mp_webhook_inbox(connection_id,environment,event_key,topic,resource_id) VALUES(?,?,?,?,?)`, review.ConnectionID, review.Environment, "review:"+id, topic, review.ResourceID).Error; e != nil {
				return e
			}
		}
		return auth.Audit(r.Context(), tx, p, "payments.review_resolved", id, map[string]string{"resolution": input.Resolution, "note": input.Note})
	})
	if e != nil {
		return e
	}
	respond(w, 200, map[string]bool{"resolved": true})
	return nil
}
func (h *Handler) myBilling(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if e := manager(p); e != nil {
		return e
	}
	brand := p.BrandID
	if p.Platform() {
		brand = r.URL.Query().Get("brand_id")
	}
	if !validID(brand) {
		return bad("Elegí una marca para consultar su membresía")
	}
	var row struct{ Payload string }
	result := h.db.WithContext(r.Context()).Raw(`SELECT jsonb_build_object('accounts',coalesce((SELECT jsonb_agg(to_jsonb(a)) FROM legui_private.billing_accounts a WHERE a.brand_id=?),'[]'::jsonb),'subscriptions',coalesce((SELECT jsonb_agg(to_jsonb(s)) FROM legui_private.subscriptions s JOIN legui_private.billing_accounts a ON a.id=s.account_id WHERE a.brand_id=?),'[]'::jsonb),'invoices',coalesce((SELECT jsonb_agg(to_jsonb(v)) FROM(SELECT i.* FROM legui_private.billing_invoices i JOIN legui_private.billing_accounts a ON a.id=i.account_id WHERE a.brand_id=? ORDER BY i.created_at DESC LIMIT 100)v),'[]'::jsonb))::text payload`, brand, brand, brand).Scan(&row)
	if result.Error != nil {
		return result.Error
	}
	w.WriteHeader(200)
	_, _ = w.Write([]byte(row.Payload))
	return nil
}
