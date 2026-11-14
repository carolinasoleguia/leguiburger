package mercadopago

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"leguiburger/internal/auth"
)

type membershipChangeInput struct {
	PriceID            string `json:"price_id"`
	PurchasedLocations int    `json:"purchased_locations"`
}

type membershipChangeQuote struct {
	ID                   string    `json:"id"`
	SourceSubscriptionID string    `json:"source_subscription_id"`
	TargetPriceID        string    `json:"target_price_id"`
	Environment          string    `json:"environment"`
	Currency             string    `json:"currency"`
	SourceLocations      int       `json:"source_locations"`
	PurchasedLocations   int       `json:"purchased_locations"`
	SourceAmount         string    `json:"source_amount"`
	Amount               string    `json:"amount"`
	SourcePeriodEnd      time.Time `json:"source_period_end"`
	ExpiresAt            time.Time `json:"expires_at"`
}

const membershipChangeColumns = `id::text,source_subscription_id::text,target_price_id::text,environment,currency,source_locations,purchased_locations,source_amount::text,amount::text,source_period_end,expires_at`

// A preview persists the exact terms and operation identity. It does not change
// entitlements, cancel a mandate, create an invoice or contact the provider.
func (h *Handler) quoteMembershipChange(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if err := manager(p); err != nil {
		return err
	}
	id, err := routeID(r)
	if err != nil {
		return err
	}
	op, err := key(r)
	if err != nil {
		return err
	}
	var input membershipChangeInput
	if err = decode(r, &input); err != nil {
		return err
	}
	if !validID(input.PriceID) || input.PurchasedLocations < 1 || input.PurchasedLocations > 10000 {
		return bad("Elegí una tarifa y una cantidad válida de tiendas")
	}
	if err = h.limitAction(r, "membership-change-quote", p.ID, 30); err != nil {
		return err
	}
	digest := hash(id + ":" + input.PriceID + ":" + stringInt(input.PurchasedLocations))
	var quote membershipChangeQuote
	err = h.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := auth.LockSession(r.Context(), tx); err != nil {
			return err
		}
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "membership-change-key:"+p.ID+":"+op).Error; err != nil {
			return err
		}
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "membership-change:"+id).Error; err != nil {
			return err
		}
		var source struct {
			AccountID, BrandID, PriceID, Amount, Currency, Environment, MembershipTier string
			PurchasedLocations                                                         int
			Version                                                                    int64
			PeriodStart, PeriodEnd                                                     time.Time
		}
		q := tx.Raw(`SELECT s.account_id::text,a.brand_id::text,s.plan_price_id::text price_id,(s.contracted_amount*s.quantity)::text amount,s.currency,s.environment,pp.membership_tier,s.purchased_locations,s.version,s.current_period_start period_start,s.current_period_end period_end FROM legui_private.subscriptions s JOIN legui_private.billing_accounts a ON a.id=s.account_id AND a.closed_at IS NULL AND a.tenant_id IS NULL JOIN legui_private.billing_plan_prices pp ON pp.id=s.plan_price_id WHERE s.id=? AND s.environment=? AND (? OR a.brand_id=nullif(?,'')::uuid) FOR UPDATE OF s`, id, h.cfg.Environment, p.Platform(), p.BrandID).Scan(&source)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected != 1 {
			return missing()
		}
		var previous struct{ ID, Digest string }
		q = tx.Raw(`SELECT id::text,request_digest digest FROM legui_private.membership_changes WHERE actor_id=? AND operation_key=?`, p.ID, op).Scan(&previous)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected > 0 {
			if previous.Digest != digest {
				return conflict("La clave de operación ya corresponde a otra cotización")
			}
			return tx.Raw(`SELECT `+membershipChangeColumns+` FROM legui_private.membership_changes WHERE id=?`, previous.ID).Scan(&quote).Error
		}
		var eligible bool
		if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM legui_private.subscriptions s JOIN legui_private.mp_subscription_mandates m ON m.subscription_id=s.id WHERE s.id=? AND s.status IN ('active','past_due') AND s.activated_at IS NOT NULL AND s.current_period_start<=clock_timestamp() AND s.current_period_end>clock_timestamp()+interval '5 minutes' AND NOT s.cancel_at_period_end AND s.provider='mercadopago' AND s.provider_reference=m.remote_id AND m.status='authorized' AND NOT m.cancel_pending)`, id).Scan(&eligible).Error; err != nil {
			return err
		}
		if !eligible || source.MembershipTier == "legacy" {
			return conflict("La membresía necesita un período pagado y una renovación de Mercado Pago activa para ampliarse")
		}
		if input.PurchasedLocations < source.PurchasedLocations {
			return bad("La ampliación debe conservar o aumentar la cantidad de tiendas contratadas")
		}
		var target struct{ Name, Amount string }
		q = tx.Raw(`SELECT p.name,(pp.amount+(?::int-pp.included_locations)*pp.extra_location_amount)::text amount FROM legui_private.billing_plan_prices pp JOIN legui_private.billing_plans p ON p.id=pp.plan_id WHERE pp.id=? AND pp.membership_tier='premium' AND pp.billing_unit='brand' AND pp.interval_months=1 AND pp.currency=? AND ((pp.is_active AND p.is_active) OR pp.id=?) FOR SHARE OF pp`, input.PurchasedLocations, input.PriceID, source.Currency, source.PriceID).Scan(&target)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected != 1 || !positiveMoney(target.Amount) {
			return bad("La ampliación requiere una tarifa Premium disponible")
		}
		if input.PriceID == source.PriceID && input.PurchasedLocations == source.PurchasedLocations {
			return bad("La membresía ya tiene ese cupo y esa tarifa")
		}
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "business-capacity:"+source.BrandID).Error; err != nil {
			return err
		}
		var locations int
		if err := tx.Raw(`SELECT count(*) FROM public.tenants WHERE brand_id=? AND active`, source.BrandID).Scan(&locations).Error; err != nil {
			return err
		}
		if input.PurchasedLocations < locations {
			return conflict("El cupo debe cubrir todas las tiendas activas del negocio")
		}
		quote.ID = uuid.NewString()
		q = tx.Raw(`INSERT INTO legui_private.membership_changes(id,account_id,source_subscription_id,brand_id,target_price_id,environment,currency,source_version,source_locations,purchased_locations,source_amount,amount,source_period_start,source_period_end,expires_at,actor_id,operation_key,request_digest) VALUES(?,?,?,?,?,?,?,?,?,?,?::numeric,?::numeric,?,?,least(clock_timestamp()+interval '15 minutes',?::timestamptz-interval '1 minute'),?,?,?) RETURNING `+membershipChangeColumns, quote.ID, source.AccountID, id, source.BrandID, input.PriceID, source.Environment, source.Currency, source.Version, source.PurchasedLocations, input.PurchasedLocations, source.Amount, target.Amount, source.PeriodStart, source.PeriodEnd, source.PeriodEnd, p.ID, op, digest).Scan(&quote)
		if q.Error != nil {
			return q.Error
		}
		facts, _ := json.Marshal(map[string]any{"source_subscription_id": id, "purchased_locations": input.PurchasedLocations, "amount": quote.Amount, "currency": quote.Currency})
		return auth.Audit(r.Context(), tx, p, "membership.change_quoted", quote.ID, json.RawMessage(facts))
	})
	if err != nil {
		return err
	}
	respond(w, http.StatusOK, quote)
	return nil
}
