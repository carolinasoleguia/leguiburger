package operations

import (
	"encoding/json"
	"gorm.io/gorm"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
var planCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,49}$`)

func (h *Handler) listBusinesses(r *http.Request, p Principal) (Result, error) {
	limit, offset, err := pagination(r)
	if err != nil {
		return Result{}, err
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb),'limit',?::int,'offset',?::int)::text payload FROM(SELECT b.*,count(t.id) locations FROM public.brands b LEFT JOIN public.tenants t ON t.brand_id=b.id GROUP BY b.id ORDER BY b.created_at DESC,b.id DESC LIMIT ? OFFSET ?)v`, limit, offset, limit, offset))
}
func (h *Handler) updateBusiness(r *http.Request, p Principal) (Result, error) {
	id, err := resourceID(r)
	if err != nil {
		return Result{}, err
	}
	input, err := decode[BusinessSettingsRequest](r)
	if err != nil {
		return Result{}, err
	}
	if !isOneOf(input.Classification, "unclassified", "production", "test", "internal") || len(input.AcquisitionSource) > 100 || (input.ActivatedAt != nil && input.ActivatedAt.After(time.Now().Add(5*time.Minute))) {
		return Result{}, badInput("Clasificación o activación inválidas")
	}
	return h.mutate(r, p, Scope{BrandID: id}, "platform.business", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		data, err := jsonQuery(tx, `UPDATE public.brands SET data_classification=?,acquisition_source=coalesce(nullif(?,''),acquisition_source),activated_at=coalesce(?::timestamptz,activated_at),updated_at=clock_timestamp() WHERE id=? RETURNING to_jsonb(brands)::text payload`, input.Classification, input.AcquisitionSource, input.ActivatedAt, id)
		return data, 200, err
	})
}
func (h *Handler) listPlans(r *http.Request, p Principal) (Result, error) {
	limit, offset, err := pagination(r)
	if err != nil {
		return Result{}, err
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(v.payload),'[]'::jsonb),'limit',?::int,'offset',?::int)::text payload FROM(SELECT to_jsonb(plan)||jsonb_build_object('prices',coalesce((SELECT jsonb_agg(to_jsonb(price) ORDER BY price.version) FROM legui_private.billing_plan_prices price WHERE price.plan_id=plan.id),'[]'::jsonb)) payload FROM legui_private.billing_plans plan ORDER BY created_at DESC,id DESC LIMIT ? OFFSET ?)v`, limit, offset, limit, offset))
}
func (h *Handler) createPlan(r *http.Request, p Principal) (Result, error) {
	input, err := decode[CreatePlanRequest](r)
	if err != nil {
		return Result{}, err
	}
	if !planCodePattern.MatchString(input.Code) || len(strings.TrimSpace(input.Name)) < 2 || len(input.Name) > 120 {
		return Result{}, badInput("Código o nombre de plan inválidos")
	}
	return h.mutate(r, p, Scope{}, "platform.plan", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		data, err := jsonQuery(tx, `INSERT INTO legui_private.billing_plans(code,name) VALUES(?,?) RETURNING to_jsonb(billing_plans)::text payload`, input.Code, input.Name)
		return data, 201, err
	})
}
func (h *Handler) createPlanPrice(r *http.Request, p Principal) (Result, error) {
	id, err := resourceID(r)
	if err != nil {
		return Result{}, err
	}
	input, err := decode[CreatePriceRequest](r)
	if err != nil {
		return Result{}, err
	}
	if err = normalizePriceTerms(&input); err != nil {
		return Result{}, err
	}
	if !currencyPattern.MatchString(input.Currency) || !validMoney(input.Amount, false) || !isOneOf(input.BillingUnit, "brand", "tenant") || (input.IntervalMonths != 1 && input.IntervalMonths != 12) || input.TrialDays < 0 || input.TrialDays > 365 || (input.MaximumLocations != nil && *input.MaximumLocations < 1) || (input.MaximumCollaborators != nil && *input.MaximumCollaborators < 1) {
		return Result{}, badInput("Precio o condiciones del plan inválidos")
	}
	return h.mutate(r, p, Scope{}, "platform.plan_price", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		var exists string
		q := tx.Raw(`SELECT id::text FROM legui_private.billing_plans WHERE id=? AND is_active FOR UPDATE`, id).Scan(&exists)
		if q.Error != nil {
			return nil, 0, q.Error
		}
		if q.RowsAffected == 0 {
			return nil, 0, notFound()
		}
		if *input.Published && input.MembershipTier != "legacy" {
			if err := tx.Exec(`UPDATE legui_private.billing_plan_prices SET is_active=false WHERE plan_id=? AND currency=? AND interval_months=? AND billing_unit=? AND is_active`, id, input.Currency, input.IntervalMonths, input.BillingUnit).Error; err != nil {
				return nil, 0, err
			}
		}
		data, err := jsonQuery(tx, `INSERT INTO legui_private.billing_plan_prices(plan_id,currency,amount,interval_months,billing_unit,trial_days,maximum_locations,maximum_collaborators,membership_tier,extra_location_amount,is_active,version)
  SELECT ?,?,?::numeric,?,?,?,?,?,?,?::numeric,?,coalesce(max(version),0)+1 FROM legui_private.billing_plan_prices WHERE plan_id=? AND currency=? AND interval_months=? AND billing_unit=? RETURNING to_jsonb(billing_plan_prices)::text payload`, id, input.Currency, string(input.Amount), input.IntervalMonths, input.BillingUnit, input.TrialDays, input.MaximumLocations, input.MaximumCollaborators, input.MembershipTier, string(input.ExtraLocationAmount), *input.Published, id, input.Currency, input.IntervalMonths, input.BillingUnit)
		return data, 201, err
	})
}
func (h *Handler) listBillingAccounts(r *http.Request, p Principal) (Result, error) {
	limit, offset, err := pagination(r)
	if err != nil {
		return Result{}, err
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb),'limit',?::int,'offset',?::int)::text payload FROM(SELECT a.*,b.name business_name,b.data_classification FROM legui_private.billing_accounts a JOIN public.brands b ON b.id=a.brand_id ORDER BY a.created_at DESC,a.id DESC LIMIT ? OFFSET ?)v`, limit, offset, limit, offset))
}
func (h *Handler) createBillingAccount(r *http.Request, p Principal) (Result, error) {
	input, err := decode[CreateBillingAccountRequest](r)
	if err != nil {
		return Result{}, err
	}
	if input.Currency == "" {
		input.Currency = "ARS"
	}
	if !validID(input.BrandID) || !optionalID(input.TenantID) || !currencyPattern.MatchString(input.Currency) || len(input.ContactEmail) > 254 {
		return Result{}, badInput("Cuenta de facturación inválida")
	}
	return h.mutate(r, p, Scope{BrandID: input.BrandID}, "platform.billing_account", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		unit := "brand"
		if input.TenantID != nil {
			unit = "tenant"
		}
		data, err := jsonQuery(tx, `INSERT INTO legui_private.billing_accounts(brand_id,tenant_id,billing_unit,currency,contact_email) VALUES(?,?,?,?,nullif(?,'')) RETURNING to_jsonb(billing_accounts)::text payload`, input.BrandID, input.TenantID, unit, input.Currency, input.ContactEmail)
		return data, 201, err
	})
}
func (h *Handler) listSubscriptions(r *http.Request, p Principal) (Result, error) {
	limit, offset, err := pagination(r)
	if err != nil {
		return Result{}, err
	}
	env, err := environmentQuery(r)
	if err != nil {
		return Result{}, err
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb),'limit',?::int,'offset',?::int)::text payload FROM(SELECT s.*,a.brand_id,a.tenant_id FROM legui_private.subscriptions s JOIN legui_private.billing_accounts a ON a.id=s.account_id WHERE s.environment=? ORDER BY s.started_at DESC,s.id DESC LIMIT ? OFFSET ?)v`, limit, offset, env, limit, offset))
}
func (h *Handler) createSubscription(r *http.Request, p Principal) (Result, error) {
	input, err := decode[CreateSubscriptionRequest](r)
	if err != nil {
		return Result{}, err
	}
	if input.Status == "" {
		input.Status = "incomplete"
	}
	if input.Quantity == 0 {
		input.Quantity = 1
	}
	if input.PurchasedLocations == 0 {
		input.PurchasedLocations = 1
	}
	if input.PurchasedLocations < 1 || input.PurchasedLocations > 10000 {
		return Result{}, badInput("Cantidad de locales inválida")
	}
	if !validID(input.AccountID) || !validID(input.PlanPriceID) || !isOneOf(input.Environment, "production", "sandbox") || !isOneOf(input.Status, "incomplete", "trialing", "active") || input.Quantity < 1 || input.Quantity > 10000 || (input.PeriodStart.IsZero() != input.PeriodEnd.IsZero()) || (!input.PeriodEnd.IsZero() && !input.PeriodEnd.After(input.PeriodStart)) || (input.Status == "active" && input.PeriodEnd.IsZero()) {
		return Result{}, badInput("Suscripción, entorno o período inválidos")
	}
	return h.mutate(r, p, Scope{}, "platform.subscription", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		var starts, ends any
		if !input.PeriodStart.IsZero() {
			starts, ends = input.PeriodStart, input.PeriodEnd
		}
		data, err := jsonQuery(tx, `INSERT INTO legui_private.subscriptions(account_id,plan_price_id,currency,interval_months,billing_unit,contracted_amount,quantity,purchased_locations,environment,status,current_period_start,current_period_end,trial_ends_at)
  SELECT a.id,price.id,price.currency,price.interval_months,price.billing_unit,price.amount,?,?,?,?,?::timestamptz,?::timestamptz,CASE WHEN ?='trialing' AND price.trial_days>0 THEN now()+make_interval(days=>price.trial_days) ELSE NULL END
  FROM legui_private.billing_accounts a JOIN legui_private.billing_plan_prices price ON price.id=? AND price.currency=a.currency AND price.billing_unit=a.billing_unit JOIN legui_private.billing_plans plan ON plan.id=price.plan_id
  WHERE a.id=? AND a.closed_at IS NULL AND price.is_active AND plan.is_active RETURNING to_jsonb(subscriptions)::text payload`, input.Quantity, input.PurchasedLocations, input.Environment, input.Status, starts, ends, input.Status, input.PlanPriceID, input.AccountID)
		return data, 201, err
	})
}
func (h *Handler) updateSubscription(r *http.Request, p Principal) (Result, error) {
	id, err := resourceID(r)
	if err != nil {
		return Result{}, err
	}
	input, err := decode[UpdateSubscriptionRequest](r)
	if err != nil {
		return Result{}, err
	}
	if input.ExpectedVersion < 1 || (input.Status != "" && !isOneOf(input.Status, "active", "past_due", "paused", "canceled")) || (input.PeriodStart == nil) != (input.PeriodEnd == nil) || (input.PeriodEnd != nil && !input.PeriodEnd.After(*input.PeriodStart)) {
		return Result{}, badInput("Estado, versión o período inválidos")
	}
	return h.mutate(r, p, Scope{}, "platform.subscription_update", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		var version int64
		q := tx.Raw(`SELECT version FROM legui_private.subscriptions WHERE id=? FOR UPDATE`, id).Scan(&version)
		if q.Error != nil {
			return nil, 0, q.Error
		}
		if q.RowsAffected == 0 {
			return nil, 0, notFound()
		}
		if version != input.ExpectedVersion {
			return nil, 0, conflict("La suscripción cambió; actualizá la versión")
		}
		data, err := jsonQuery(tx, `UPDATE legui_private.subscriptions SET status=coalesce(nullif(?,''),status),cancel_at_period_end=coalesce(?,cancel_at_period_end),current_period_start=coalesce(?::timestamptz,current_period_start),current_period_end=coalesce(?::timestamptz,current_period_end) WHERE id=? RETURNING to_jsonb(subscriptions)::text payload`, input.Status, input.CancelAtPeriodEnd, input.PeriodStart, input.PeriodEnd, id)
		return data, 200, err
	})
}
func (h *Handler) listInvoices(r *http.Request, p Principal) (Result, error) {
	limit, offset, err := pagination(r)
	if err != nil {
		return Result{}, err
	}
	env, err := environmentQuery(r)
	if err != nil {
		return Result{}, err
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb),'limit',?::int,'offset',?::int)::text payload FROM(SELECT * FROM legui_private.billing_invoices WHERE environment=? ORDER BY created_at DESC,id DESC LIMIT ? OFFSET ?)v`, limit, offset, env, limit, offset))
}
func (h *Handler) createInvoice(r *http.Request, p Principal) (Result, error) {
	input, err := decode[CreateInvoiceRequest](r)
	if err != nil {
		return Result{}, err
	}
	if !validID(input.SubscriptionID) || input.PeriodStart.IsZero() || !input.PeriodEnd.After(input.PeriodStart) || input.DueAt.IsZero() || len(input.Description) > 1000 {
		return Result{}, badInput("Cargo de membresía o período inválidos")
	}
	return h.mutate(r, p, Scope{}, "platform.invoice", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		if input.DueAt.Before(time.Now()) {
			return nil, 0, badInput("El vencimiento debe ser posterior a la emisión")
		}
		var id string
		q := tx.Raw(`SELECT id::text FROM legui_private.subscriptions WHERE id=? AND status<>'canceled' FOR SHARE`, input.SubscriptionID).Scan(&id)
		if q.Error != nil {
			return nil, 0, q.Error
		}
		if q.RowsAffected == 0 {
			return nil, 0, notFound()
		}
		data, err := jsonQuery(tx, `INSERT INTO legui_private.billing_invoices(account_id,subscription_id,environment,currency,amount_total,status,period_start,period_end,issued_at,due_at,description,idempotency_key,request_digest)
   SELECT account_id,id,environment,currency,contracted_amount*quantity,'open',?,?,clock_timestamp(),?,?,?,? FROM legui_private.subscriptions WHERE id=? RETURNING to_jsonb(billing_invoices)::text payload`, input.PeriodStart, input.PeriodEnd, input.DueAt, input.Description, key, digest, input.SubscriptionID)
		return data, 201, err
	})
}
func (h *Handler) listBillingPayments(r *http.Request, p Principal) (Result, error) {
	id, err := resourceID(r)
	if err != nil {
		return Result{}, err
	}
	limit, offset, err := pagination(r)
	if err != nil {
		return Result{}, err
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb),'limit',?::int,'offset',?::int)::text payload FROM(SELECT * FROM legui_private.billing_payments WHERE invoice_id=? ORDER BY created_at DESC,id DESC LIMIT ? OFFSET ?)v`, limit, offset, id, limit, offset))
}
func (h *Handler) createBillingPayment(r *http.Request, p Principal) (Result, error) {
	id, err := resourceID(r)
	if err != nil {
		return Result{}, err
	}
	input, err := decode[PaymentRequest](r)
	if err != nil {
		return Result{}, err
	}
	if err = validateManualPayment(input); err != nil {
		return Result{}, err
	}
	return h.mutate(r, p, Scope{}, "platform.billing_payment", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		var settled any
		if input.SettledAt != nil {
			settled = *input.SettledAt
		}
		data, err := jsonQuery(tx, `INSERT INTO legui_private.billing_payments(account_id,invoice_id,environment,currency,kind,original_payment_id,amount,status,method,settled_at,idempotency_key,request_digest)
   SELECT account_id,id,environment,currency,?,?,?::numeric,'succeeded',?,coalesce(?::timestamptz,clock_timestamp()),?,? FROM legui_private.billing_invoices WHERE id=? RETURNING to_jsonb(billing_payments)::text payload`, input.Kind, input.OriginalPaymentID, string(input.Amount), input.Method, settled, key, digest, id)
		return data, 201, err
	})
}
func (h *Handler) createCreditNote(r *http.Request, p Principal) (Result, error) {
	id, err := resourceID(r)
	if err != nil {
		return Result{}, err
	}
	input, err := decode[CreditNoteRequest](r)
	if err != nil {
		return Result{}, err
	}
	if !validMoney(input.Amount, true) || len(strings.TrimSpace(input.Reason)) < 3 || len(input.Reason) > 1000 {
		return Result{}, badInput("Crédito de membresía inválido")
	}
	return h.mutate(r, p, Scope{}, "platform.credit_note", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		data, err := jsonQuery(tx, `INSERT INTO legui_private.billing_credit_notes(account_id,invoice_id,environment,currency,amount,reason,actor_user_id,idempotency_key,request_digest) SELECT account_id,id,environment,currency,?::numeric,?,?,?,? FROM legui_private.billing_invoices WHERE id=? RETURNING to_jsonb(billing_credit_notes)::text payload`, string(input.Amount), input.Reason, p.ID, key, digest, id)
		return data, 201, err
	})
}
func (h *Handler) listWebhookEvents(r *http.Request, p Principal) (Result, error) {
	limit, offset, err := pagination(r)
	if err != nil {
		return Result{}, err
	}
	env, err := environmentQuery(r)
	if err != nil {
		return Result{}, err
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb),'limit',?::int,'offset',?::int)::text payload FROM(SELECT * FROM legui_private.billing_webhook_events WHERE environment=? ORDER BY received_at DESC,id DESC LIMIT ? OFFSET ?)v`, limit, offset, env, limit, offset))
}
