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
	e := h.db.WithContext(r.Context()).Raw(`SELECT price.id,plan.code,plan.name,price.currency,price.amount,price.interval_months,price.billing_unit,price.version,price.maximum_locations,price.maximum_collaborators,price.membership_tier,price.included_locations,price.extra_location_amount,CASE WHEN price.membership_tier='legacy' THEN 'business' ELSE 'location' END collaborator_scope FROM legui_private.billing_plan_prices price JOIN legui_private.billing_plans plan ON plan.id=price.plan_id WHERE price.is_active AND plan.is_active AND price.membership_tier<>'legacy' AND price.currency='ARS' AND NOT EXISTS(SELECT 1 FROM legui_private.billing_plan_prices newer WHERE newer.plan_id=price.plan_id AND newer.currency=price.currency AND newer.interval_months=price.interval_months AND newer.billing_unit=price.billing_unit AND newer.is_active AND newer.version>price.version) ORDER BY plan.name,price.interval_months LIMIT 100`).Scan(&rows).Error
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
		PriceID            string `json:"price_id"`
		TenantID           string `json:"tenant_id"`
		BrandID            string `json:"brand_id"`
		PayerEmail         string `json:"payer_email"`
		PurchasedLocations int    `json:"purchased_locations,omitempty"`
	}
	if e := decode(r, &input); e != nil {
		return e
	}
	if !validID(input.PriceID) || !validEmail(input.PayerEmail) {
		return bad("Elegí un plan e ingresá el correo del pagador")
	}
	if input.PurchasedLocations == 0 {
		input.PurchasedLocations = 1
	}
	if input.PurchasedLocations < 1 || input.PurchasedLocations > 10000 {
		return bad("Cantidad de locales inválida")
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
			MembershipTier string
		}
		result := tx.Raw(`SELECT pp.currency,pp.billing_unit,pp.amount::text,pp.interval_months,pp.membership_tier FROM legui_private.billing_plan_prices pp JOIN legui_private.billing_plans p ON p.id=pp.plan_id AND p.is_active WHERE pp.id=? AND pp.is_active FOR SHARE OF pp`, input.PriceID).Scan(&price)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 || price.Currency != "ARS" || !positiveMoney(price.Amount) {
			return bad("El plan no está disponible")
		}
		if price.MembershipTier == "legacy" && !p.Platform() {
			return bad("Elegí una membresía Clásica o Premium publicada")
		}
		if price.MembershipTier != "legacy" {
			if e := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "business-capacity:"+input.BrandID).Error; e != nil {
				return e
			}
			var count int
			if e := tx.Raw(`SELECT count(*) FROM public.tenants WHERE brand_id=? AND active`, input.BrandID).Scan(&count).Error; e != nil {
				return e
			}
			if count > input.PurchasedLocations {
				return conflict("Contratá cupo para todos los locales activos o desactivá los que no vayas a usar")
			}
		}
		if (price.BillingUnit == "tenant") != (input.TenantID != "") {
			return bad("El plan debe corresponder a una marca o a un local")
		}
		if price.MembershipTier != "premium" && input.PurchasedLocations != 1 {
			return bad("Solo Premium permite contratar locales adicionales")
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
			ID                 string
			PlanPriceID        string
			ProviderReference  string
			PurchasedLocations int
		}
		result = tx.Raw(`SELECT id::text,plan_price_id::text,purchased_locations,coalesce(provider_reference,'') provider_reference FROM legui_private.subscriptions WHERE account_id=? AND environment=? AND status<>'canceled' FOR UPDATE`, accountID, h.cfg.Environment).Scan(&prior)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			if prior.PlanPriceID != input.PriceID || prior.PurchasedLocations != input.PurchasedLocations {
				return conflict("Ya existe una suscripción con otro plan. Solicitá el cambio de plan")
			}
			subscriptionID = prior.ID
			return nil
		}
		subscriptionID = uuid.NewString()
		if e := tx.Exec(`INSERT INTO legui_private.subscriptions(id,account_id,plan_price_id,currency,interval_months,billing_unit,contracted_amount,quantity,purchased_locations,environment,status) VALUES(?,?,?,?,?,?,?::numeric,1,?,?,'incomplete')`, subscriptionID, accountID, input.PriceID, price.Currency, price.IntervalMonths, price.BillingUnit, price.Amount, input.PurchasedLocations, h.cfg.Environment).Error; e != nil {
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
