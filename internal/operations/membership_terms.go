package operations

import (
	"encoding/json"
	"net/http"

	"gorm.io/gorm"
)

// Prices describe immutable commercial terms; draft prices cannot be purchased.
func normalizePriceTerms(input *CreatePriceRequest) error {
	if input.MembershipTier == "" {
		input.MembershipTier = "legacy"
	}
	if input.ExtraLocationAmount == "" {
		input.ExtraLocationAmount = "0"
	}
	if !isOneOf(input.MembershipTier, "legacy", "standard", "premium") || !validMoney(input.ExtraLocationAmount, false) {
		return badInput("Tipo de membresía o adicional por local inválidos")
	}
	if input.Published == nil {
		published := input.MembershipTier == "legacy"
		input.Published = &published
	}
	if input.MembershipTier == "legacy" {
		if input.ExtraLocationAmount.Positive() {
			return badInput("Los adicionales por local requieren un plan Premium mensual por marca")
		}
		return nil
	}
	if input.BillingUnit != "brand" || input.IntervalMonths != 1 || input.TrialDays != 0 {
		return badInput("Las membresías comerciales se cobran por marca y por mes; la prueba se configura por separado")
	}
	seats := 2
	if input.MembershipTier == "premium" {
		seats = 5
		input.MaximumLocations = nil
	} else {
		one := 1
		input.MaximumLocations = &one
		if input.ExtraLocationAmount.Positive() {
			return badInput("El plan Clásico incluye un único local")
		}
	}
	input.MaximumCollaborators = &seats
	if *input.Published && (!validMoney(input.Amount, true) || (input.MembershipTier == "premium" && !input.ExtraLocationAmount.Positive())) {
		return badInput("Definí el precio mensual y el adicional Premium antes de publicar")
	}
	return nil
}

func (h *Handler) publishPlanPrice(r *http.Request, p Principal) (Result, error) {
	id, err := resourceID(r)
	if err != nil {
		return Result{}, err
	}
	input, err := decode[struct {
		Published *bool `json:"published"`
	}](r)
	if err != nil {
		return Result{}, err
	}
	if input.Published == nil {
		return Result{}, badInput("Indicá si el precio debe estar publicado")
	}
	return h.mutate(r, p, Scope{}, "platform.plan_price_publication", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		var plan string
		q := tx.Raw(`SELECT p.id::text FROM legui_private.billing_plans p WHERE p.id=(SELECT plan_id FROM legui_private.billing_plan_prices WHERE id=?) FOR UPDATE`, id).Scan(&plan)
		if q.Error != nil {
			return nil, 0, q.Error
		}
		if q.RowsAffected != 1 {
			return nil, 0, notFound()
		}
		if *input.Published {
			if err := tx.Exec(`UPDATE legui_private.billing_plan_prices other SET is_active=false FROM legui_private.billing_plan_prices selected WHERE selected.id=? AND selected.membership_tier<>'legacy' AND other.plan_id=selected.plan_id AND other.currency=selected.currency AND other.interval_months=selected.interval_months AND other.billing_unit=selected.billing_unit AND other.id<>selected.id AND other.is_active`, id).Error; err != nil {
				return nil, 0, err
			}
		}
		data, err := jsonQuery(tx, `UPDATE legui_private.billing_plan_prices SET is_active=? WHERE id=? RETURNING to_jsonb(billing_plan_prices)::text payload`, *input.Published, id)
		return data, http.StatusOK, err
	})
}
