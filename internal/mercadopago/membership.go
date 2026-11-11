package mercadopago

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"leguiburger/internal/auth"
	"net/http"
)

func (h *Handler) publicPlans(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if e := manager(p); e != nil {
		return e
	}
	var rows []map[string]any
	e := h.db.WithContext(r.Context()).Raw(`SELECT price.id,plan.code,plan.name,price.currency,price.amount,price.interval_months,price.billing_unit,price.version,price.maximum_locations,price.maximum_collaborators FROM legui_private.billing_plan_prices price JOIN legui_private.billing_plans plan ON plan.id=price.plan_id WHERE price.is_active AND plan.is_active AND price.currency='ARS' ORDER BY plan.name,price.interval_months,price.version DESC LIMIT 100`).Scan(&rows).Error
	if e != nil {
		return e
	}
	respond(w, 200, map[string]any{"items": rows})
	return nil
}
func (h *Handler) subscribe(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if e := manager(p); e != nil {
		return e
	}
	if e := h.enabled(h.cfg.Environment); e != nil {
		return e
	}
	var input struct {
		PriceID    string `json:"price_id"`
		TenantID   string `json:"tenant_id"`
		BrandID    string `json:"brand_id"`
		PayerEmail string `json:"payer_email"`
	}
	if e := decode(r, &input); e != nil {
		return e
	}
	if !validID(input.PriceID) || !validEmail(input.PayerEmail) {
		return bad("Elegí un plan e ingresá el correo del pagador")
	}
	if input.BrandID == "" {
		input.BrandID = p.BrandID
	}
	if !validID(input.BrandID) || !p.Platform() && input.BrandID != p.BrandID {
		return denied()
	}
	if input.TenantID != "" {
		brand, e := h.scope(r, p, input.TenantID)
		if e != nil {
			return e
		}
		if brand != input.BrandID {
			return denied()
		}
	}
	var subscriptionID string
	e := h.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := auth.LockSession(r.Context(), tx); e != nil {
			return e
		}
		if e := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "mp-subscribe:"+input.BrandID+":"+input.TenantID+":"+h.cfg.Environment).Error; e != nil {
			return e
		}
		var price struct {
			Currency       string
			BillingUnit    string
			Amount         string
			IntervalMonths int
		}
		result := tx.Raw(`SELECT pp.currency,pp.billing_unit,pp.amount::text,pp.interval_months FROM legui_private.billing_plan_prices pp JOIN legui_private.billing_plans p ON p.id=pp.plan_id AND p.is_active WHERE pp.id=? AND pp.is_active FOR SHARE OF pp`, input.PriceID).Scan(&price)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 || price.Currency != "ARS" || !positiveMoney(price.Amount) {
			return bad("El plan no está disponible")
		}
		if (price.BillingUnit == "tenant") != (input.TenantID != "") {
			return bad("El plan debe corresponder a una marca o a un local")
		}
		var accountID string
		result = tx.Raw(`SELECT id::text FROM legui_private.billing_accounts WHERE brand_id=? AND coalesce(tenant_id::text,'')=? AND closed_at IS NULL FOR UPDATE`, input.BrandID, input.TenantID).Scan(&accountID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			accountID = uuid.NewString()
			if e := tx.Exec(`INSERT INTO legui_private.billing_accounts(id,brand_id,tenant_id,billing_unit,currency,contact_email) VALUES(?,?,nullif(?,'')::uuid,?,?,?)`, accountID, input.BrandID, input.TenantID, price.BillingUnit, price.Currency, input.PayerEmail).Error; e != nil {
				return e
			}
		}
		var prior struct {
			ID                string
			PlanPriceID       string
			ProviderReference string
		}
		result = tx.Raw(`SELECT id::text,plan_price_id::text,coalesce(provider_reference,'') provider_reference FROM legui_private.subscriptions WHERE account_id=? AND environment=? AND status<>'canceled' FOR UPDATE`, accountID, h.cfg.Environment).Scan(&prior)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			if prior.PlanPriceID != input.PriceID {
				return conflict("Ya existe una suscripción con otro plan. Solicitá el cambio de plan")
			}
			subscriptionID = prior.ID
			return nil
		}
		subscriptionID = uuid.NewString()
		if e := tx.Exec(`INSERT INTO legui_private.subscriptions(id,account_id,plan_price_id,currency,interval_months,billing_unit,contracted_amount,quantity,environment,status) VALUES(?,?,?,?,?,?,?::numeric,1,?,'incomplete')`, subscriptionID, accountID, input.PriceID, price.Currency, price.IntervalMonths, price.BillingUnit, price.Amount, h.cfg.Environment).Error; e != nil {
			return e
		}
		return auth.Audit(r.Context(), tx, p, "payments.membership_selected", subscriptionID, nil)
	})
	if e != nil {
		return e
	}
	respond(w, 201, map[string]string{"subscription_id": subscriptionID})
	return nil
}
