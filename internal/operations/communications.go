package operations

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"leguiburger/internal/auth"
	"leguiburger/internal/buyeraccess"
)

func ProcessStorefrontWork(ctx context.Context, db *gorm.DB) error {
	tx := db.WithContext(ctx)
	if e := tx.Exec(`SELECT legui_private.expire_pending_orders()`).Error; e != nil {
		return e
	}
	for i := 0; i < 20 && commerceSite() != ""; i++ {
		changed, e := queueOrderMail(tx)
		if e != nil {
			return e
		}
		if !changed {
			break
		}
	}
	if commerceSite() != "" {
		for i := 0; i < 20; i++ {
			changed, e := queueCampaignMail(tx)
			if e != nil {
				return e
			}
			if !changed {
				break
			}
		}
	}
	return tx.Exec(`UPDATE legui_private.newsletter_campaigns c SET state='completed' WHERE c.state='queued' AND NOT EXISTS(SELECT 1 FROM legui_private.newsletter_recipients r LEFT JOIN legui_private.mail_outbox m ON m.id=r.outbox_id WHERE r.campaign_id=c.id AND(r.status='pending' OR(m.body_cipher IS NOT NULL AND m.delivered_at IS NULL AND m.attempts<8 AND m.expires_at>clock_timestamp())))`).Error
}
func queueOrderMail(db *gorm.DB) (bool, error) {
	changed := false
	e := db.Transaction(func(tx *gorm.DB) error {
		var row struct{ ID, OrderID, Status, Paid string }
		q := tx.Raw(`SELECT id::text,order_id::text,status,paid::text FROM legui_private.order_mail_events WHERE processed_at IS NULL ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&row)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected == 0 {
			return nil
		}
		changed = true
		var target struct{ ShopperID, Email, Business, Status, Paid, Total, Reason string }
		q = tx.Raw(`SELECT p.id::text shopper_id,p.email,b.name business,s.code status,o.total_paid::text paid,o.total_due::text total,o.ended_reason reason FROM public.orders o JOIN public.order_statuses_lookup s ON s.id=o.order_status_id JOIN public.tenants t ON t.id=o.tenant_id AND t.active JOIN public.brands b ON b.id=t.brand_id AND b.suspended_at IS NULL AND b.closed_at IS NULL JOIN legui_private.shopper_orders so ON so.order_id=o.id JOIN legui_private.shoppers p ON p.id=so.shopper_id AND p.active AND p.email_verified WHERE o.id=? AND s.code=? AND o.total_paid=?::numeric`, row.OrderID, row.Status, row.Paid).Scan(&target)
		if q.Error != nil {
			return q.Error
		}
		mailID := ""
		if q.RowsAffected == 1 && commerceSite() != "" {
			labels := map[string]string{"pending": "Recibimos tu pedido", "accepted": "Tu pedido fue confirmado", "preparing": "Estamos preparando tu pedido", "ready": "Tu pedido está listo", "dispatched": "Tu pedido está en camino", "delivered": "Tu pedido fue entregado", "canceled": "Tu pedido fue cancelado", "rejected": "El local no pudo aceptar tu pedido"}
			title := labels[target.Status]
			if title == "" {
				title = "Tu pedido se actualizó"
			}
			body := fmt.Sprintf(`%s
%s · Pedido #%s
Total: ARS %s
Recibido: ARS %s
%s
Seguí tu pedido desde tu cuenta: %s/mi-cuenta`, title, target.Business, row.OrderID[:8], target.Total, target.Paid, target.Reason, commerceSite())
			var e error
			mailID, e = auth.QueueCommerceMail(tx, target.Email, target.Business+" · "+title, body, "", target.ShopperID, "", row.ID)
			if e != nil {
				return e
			}
		}
		return tx.Exec(`UPDATE legui_private.order_mail_events SET processed_at=clock_timestamp(),outbox_id=nullif(?,'')::uuid WHERE id=?`, mailID, row.ID).Error
	})
	return changed, e
}
func queueCampaignMail(db *gorm.DB) (bool, error) {
	changed := false
	e := db.Transaction(func(tx *gorm.DB) error {
		var row struct {
			CampaignID, ShopperID, RecipientEmail, BrandID, Subject, BodyCipher, Business string
			Eligible                                                                      bool
		}
		q := tx.Raw(`SELECT r.campaign_id::text,r.shopper_id::text,r.recipient_email,c.brand_id::text,c.subject,c.body_cipher,b.name business,(c.state='queued' AND b.suspended_at IS NULL AND b.closed_at IS NULL AND p.active AND p.email_verified AND p.email=r.recipient_email AND pref.newsletter AND(c.segment='all' OR pref.personalization)) eligible FROM legui_private.newsletter_recipients r JOIN legui_private.newsletter_campaigns c ON c.id=r.campaign_id JOIN public.brands b ON b.id=c.brand_id JOIN legui_private.shoppers p ON p.id=r.shopper_id JOIN legui_private.shopper_preferences pref ON pref.shopper_id=p.id AND pref.brand_id=c.brand_id WHERE r.status='pending' ORDER BY c.created_at,r.shopper_id LIMIT 1 FOR UPDATE OF r SKIP LOCKED`).Scan(&row)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected == 0 {
			return nil
		}
		changed = true
		state, mailID := "skipped", ""
		if row.Eligible {
			body, e := auth.OpenCampaign(row.CampaignID, row.BodyCipher)
			if e != nil {
				return e
			}
			raw, e := unsubscribeToken()
			if e != nil {
				return e
			}
			if e = tx.Exec(`INSERT INTO legui_private.newsletter_unsubscribes(digest,brand_id,shopper_id) VALUES(?,?,?)`, buyeraccess.Digest(raw), row.BrandID, row.ShopperID).Error; e != nil {
				return e
			}
			body += "\n\nRecibís este correo porque te suscribiste a " + row.Business + ".\nDejar de recibir novedades de esta marca: " + commerceSite() + "/baja#token=" + raw
			mailID, e = auth.QueueCommerceMail(tx, row.RecipientEmail, row.Subject, body, row.CampaignID, row.ShopperID, row.BrandID, "")
			if e != nil {
				return e
			}
			state = "queued"
		}
		return tx.Exec(`UPDATE legui_private.newsletter_recipients SET status=?,outbox_id=nullif(?,'')::uuid WHERE campaign_id=? AND shopper_id=?`, state, mailID, row.CampaignID, row.ShopperID).Error
	})
	return changed, e
}
