package mercadopago

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"strings"
	"time"
)

type paymentProof struct {
	ID           string     `json:"id"`
	Collector    string     `json:"collector"`
	Environment  string     `json:"environment"`
	Reference    string     `json:"reference"`
	Subscription string     `json:"subscription"`
	Status       string     `json:"status"`
	Detail       string     `json:"detail"`
	Currency     string     `json:"currency"`
	Amount       string     `json:"amount"`
	Refunded     string     `json:"refunded"`
	Fee          *string    `json:"fee"`
	Net          *string    `json:"net"`
	Available    *time.Time `json:"available_at"`
	Paid         *time.Time `json:"paid_at"`
	Updated      time.Time  `json:"updated_at"`
}

func parsePayment(v map[string]any, id string, c connection) (paymentProof, error) {
	p := paymentProof{ID: text(v["id"]), Collector: text(v["collector_id"]), Environment: c.Environment, Reference: text(v["external_reference"]), Subscription: text(object(object(v["point_of_interaction"])["transaction_data"])["subscription_id"]), Status: text(v["status"]), Detail: text(v["status_detail"]), Currency: text(v["currency_id"]), Amount: text(v["transaction_amount"]), Refunded: text(v["transaction_amount_refunded"])}
	live, ok := v["live_mode"].(bool)
	if !ok || live != (c.Environment == "production") || p.ID != id || p.Collector != c.CollectorID || p.Currency != "ARS" || !positiveMoney(p.Amount) {
		return p, errors.New("payment ownership or environment unconfirmed")
	}
	if p.Refunded == "" {
		p.Refunded = "0"
	}
	a, e := money(p.Amount)
	ref, f := money(p.Refunded)
	if e != nil || f != nil || ref.Cmp(a) > 0 {
		return p, errors.New("invalid refund amount")
	}
	if !oneOf(p.Status, "approved", "pending", "authorized", "in_process", "in_mediation", "rejected", "cancelled", "refunded", "charged_back") {
		return p, errors.New("unknown provider status")
	}
	p.Updated, e = providerTime(v["date_last_updated"])
	if e != nil || p.Updated.After(time.Now().Add(5*time.Minute)) {
		return p, errors.New("invalid provider timestamp")
	}
	if v["date_approved"] != nil {
		at, e := providerTime(v["date_approved"])
		if e != nil || at.After(time.Now().Add(5*time.Minute)) {
			return p, errors.New("invalid approval timestamp")
		}
		p.Paid = &at
	}
	if (p.Status == "approved" || p.Status == "refunded" || p.Status == "charged_back") && p.Paid == nil {
		return p, errors.New("missing approval timestamp")
	}
	if raw, exists := v["fee_details"]; exists && raw != nil {
		fees, ok := raw.([]any)
		if !ok {
			return p, errors.New("invalid provider fees")
		}
		total := "0.00"
		for _, item := range fees {
			amount := text(object(item)["amount"])
			var e error
			total, e = plus(total, amount)
			if e != nil {
				return p, e
			}
		}
		p.Fee = &total
	}
	if raw := object(v["transaction_details"])["net_received_amount"]; raw != nil {
		amount := text(raw)
		if _, e := money(amount); e != nil {
			return p, e
		}
		p.Net = &amount
	}
	if v["money_release_date"] != nil {
		at, e := providerTime(v["money_release_date"])
		if e != nil {
			return p, e
		}
		p.Available = &at
	}
	return p, nil
}
func oneOf(value string, options ...string) bool {
	for _, o := range options {
		if o == value {
			return true
		}
	}
	return false
}
func (h *Handler) review(tx *gorm.DB, c connection, checkoutID, resource, reason, proof string) error {
	return tx.Exec(`INSERT INTO legui_private.mp_reviews(connection_id,checkout_id,environment,resource_id,reason,proof_digest) VALUES(?,nullif(?,'')::uuid,?,?,?,?) ON CONFLICT(connection_id,environment,resource_id,reason) WHERE resolved_at IS NULL DO UPDATE SET last_seen_at=clock_timestamp(),occurrences=mp_reviews.occurrences+1,proof_digest=excluded.proof_digest`, c.ID, checkoutID, c.Environment, resource, reason, proof).Error
}
func (h *Handler) observePayment(ctx context.Context, connectionID, id string) (string, error) {
	if !resourcePattern.MatchString(id) {
		return "", bad("Identificador de pago inválido")
	}
	c, token, e := h.account(ctx, connectionID)
	if e != nil {
		return "", e
	}
	raw, e := h.provider.call(ctx, "GET", "/v1/payments/"+id, token, "", nil)
	if e != nil {
		return "", e
	}
	proof, e := parsePayment(raw, id, c)
	if e != nil {
		e = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return h.review(tx, c, "", id, "INVALID_PROVIDER_PROOF", hash(id+":"+c.Environment))
		})
		return "review", e
	}
	proofJSON := jsonString(proof)
	proofDigest := hash(proofJSON)
	state := "observed"
	e = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "mp-payment:"+c.Environment+":"+id).Error; e != nil {
			return e
		}
		var latest struct {
			ID           string
			ConnectionID string
			Updated      time.Time
			ProofDigest  string
		}
		result := tx.Raw(`SELECT id::text,connection_id::text,provider_updated_at updated,proof_digest FROM legui_private.mp_provider_payments WHERE environment=? AND payment_id=? FOR UPDATE`, c.Environment, id).Scan(&latest)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 && latest.ConnectionID != c.ID {
			state = "review"
			return h.review(tx, c, "", id, "PAYMENT_ACCOUNT_CONFLICT", proofDigest)
		}
		var current connection
		result = tx.Raw(`SELECT `+connectionColumns+` FROM legui_private.mp_connections WHERE id=? AND version=? AND credential_version=? AND state='connected' FOR SHARE`, c.ID, c.Version, c.CredentialVersion).Scan(&current)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return conflict("La conexión cambió durante la verificación")
		}
		if e := tx.Exec(`INSERT INTO legui_private.mp_payment_observations(connection_id,environment,payment_id,provider_updated_at,provider_status,proof_digest,proof) VALUES(?,?,?,?,?,?,?::jsonb) ON CONFLICT DO NOTHING`, c.ID, c.Environment, id, proof.Updated, proof.Status, proofDigest, proofJSON).Error; e != nil {
			return e
		}
		if latest.ID != "" && proof.Updated.Before(latest.Updated) {
			state = "stale"
			return nil
		}
		if latest.ID != "" && proof.Updated.Equal(latest.Updated) && proofDigest != latest.ProofDigest {
			state = "review"
			return h.review(tx, c, "", id, "PROVIDER_VERSION_CONFLICT", proofDigest)
		}
		var checkoutID string
		if strings.HasPrefix(proof.Reference, "LEG:UI:") {
			checkoutID = strings.TrimPrefix(proof.Reference, "LEG:UI:")
			if !validID(checkoutID) {
				checkoutID = ""
			}
		}
		var co checkout
		if checkoutID != "" {
			result = tx.Raw(`SELECT `+checkoutColumns+` FROM legui_private.mp_checkouts WHERE id=? AND connection_id=? AND environment=? FOR UPDATE`, checkoutID, c.ID, c.Environment).Scan(&co)
			if result.Error != nil {
				return result.Error
			}
		}
		if co.ID == "" {
			checkoutID = ""
		}
		var paymentID string
		if e := tx.Raw(`INSERT INTO legui_private.mp_provider_payments(connection_id,checkout_id,environment,payment_id,collector_id,provider_status,provider_detail,currency,amount,refunded_amount,fee_amount,net_received,available_at,provider_updated_at,paid_at,proof_digest) VALUES(?,nullif(?,'')::uuid,?,?,?,?,?,?,?::numeric,?::numeric,?::numeric,?::numeric,?,?,?,?) ON CONFLICT(environment,payment_id) DO UPDATE SET provider_status=excluded.provider_status,provider_detail=excluded.provider_detail,refunded_amount=excluded.refunded_amount,fee_amount=excluded.fee_amount,net_received=excluded.net_received,available_at=excluded.available_at,provider_updated_at=excluded.provider_updated_at,paid_at=excluded.paid_at,proof_digest=excluded.proof_digest,updated_at=clock_timestamp() RETURNING id::text`, c.ID, checkoutID, c.Environment, id, proof.Collector, proof.Status, proof.Detail, proof.Currency, proof.Amount, proof.Refunded, proof.Fee, proof.Net, proof.Available, proof.Updated, proof.Paid, proofDigest).Scan(&paymentID).Error; e != nil {
			return e
		}
		hold := func(reason string) error {
			state = "review"
			return h.review(tx, c, checkoutID, id, reason, proofDigest)
		}
		if latest.ID != "" {
			var original struct {
				CheckoutID string
				Amount     string
				Currency   string
			}
			if e := tx.Raw(`SELECT coalesce(checkout_id::text,'') checkout_id,amount::text,currency FROM legui_private.mp_provider_payments WHERE id=?`, paymentID).Scan(&original).Error; e != nil {
				return e
			}
			if original.CheckoutID != checkoutID || !sameMoney(original.Amount, proof.Amount) || original.Currency != proof.Currency {
				return hold("PAYMENT_TERMS_CHANGED")
			}
		}
		if co.ID == "" {
			return hold("UNOWNED_REFERENCE")
		}
		if co.ConnectionVersion != c.Version {
			return hold("CONNECTION_CHANGED")
		}
		if co.SubmittedAt == nil {
			return hold("NO_DURABLE_SUBMISSION")
		}
		if proof.Paid != nil && proof.Paid.Before(co.SubmittedAt.Add(-5*time.Minute)) {
			return hold("PAYMENT_PREDATES_CHECKOUT")
		}
		if !sameMoney(proof.Amount, co.Amount) || proof.Currency != co.Currency {
			return hold("AMOUNT_MISMATCH")
		}
		if proof.Status == "charged_back" {
			return hold("CHARGEBACK")
		}
		if proof.Status == "in_mediation" {
			return hold("PAYMENT_IN_MEDIATION")
		}
		var allocated int64
		if e := tx.Raw(`SELECT count(*) FROM legui_private.mp_payment_allocations WHERE provider_payment_id=? AND component='charge'`, paymentID).Scan(&allocated).Error; e != nil {
			return e
		}
		if allocated > 0 {
			state = "replayed"
			if oneOf(proof.Status, "pending", "authorized", "in_process", "rejected", "cancelled") {
				return hold("PAYMENT_STATUS_REGRESSED")
			}
			return nil
		}
		if proof.Status != "approved" || !sameMoney(proof.Refunded, "0") {
			if proof.Status == "refunded" || !sameMoney(proof.Refunded, "0") {
				return hold("REVERSED_BEFORE_ALLOCATION")
			}
			return nil
		}
		var open int64
		if e := tx.Raw(`SELECT count(*) FROM legui_private.mp_reviews WHERE connection_id=? AND environment=? AND resource_id=? AND resolved_at IS NULL`, c.ID, c.Environment, id).Scan(&open).Error; e != nil {
			return e
		}
		if open > 0 {
			return hold("OPEN_REVIEW")
		}
		allocationError := tx.Transaction(func(inner *gorm.DB) error {
			if e := h.allocate(inner, c, co, paymentID, proof, proofDigest); e != nil {
				return e
			}
			return inner.Exec(`INSERT INTO legui_private.security_events(brand_id,tenant_id,event_name,target_id,facts) VALUES(?,nullif(?,'')::uuid,'payments.verified',?,?::jsonb)`, co.BrandID, co.TenantID, paymentID, jsonString(map[string]string{"provider": "mercadopago", "payment_id": proof.ID, "purpose": co.Purpose, "environment": co.Environment})).Error
		})
		if allocationError != nil {
			return hold("ALLOCATION_REQUIRES_REVIEW")
		}
		state = "applied"
		return nil
	})
	if e != nil {
		return "", e
	}
	if !sameMoney(proof.Refunded, "0") && state != "review" {
		refundState, e := h.observeRefunds(ctx, c, token, id)
		if e != nil {
			return "", e
		}
		if refundState == "review" {
			state = "review"
		}
	}
	return state, nil
}
func (h *Handler) allocate(tx *gorm.DB, c connection, co checkout, paymentID string, proof paymentProof, proofDigest string) error {
	var rowID string
	if co.Purpose == "order" {
		var existing int64
		if e := tx.Raw(`SELECT count(*) FROM legui_private.mp_provider_payments p JOIN legui_private.mp_payment_allocations a ON a.provider_payment_id=p.id AND a.component='charge' WHERE p.checkout_id=?`, co.ID).Scan(&existing).Error; e != nil {
			return e
		}
		if existing > 0 {
			return conflict("El checkout ya fue pagado")
		}
		var order struct {
			Balance     string
			Environment string
			Currency    string
			Status      string
			BrandID     string
		}
		result := tx.Raw(`SELECT (o.total_due-o.total_paid)::text balance,o.payment_environment environment,o.currency,s.code status,t.brand_id::text FROM public.orders o JOIN public.tenants t ON t.id=o.tenant_id JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE o.id=? AND o.tenant_id=? FOR UPDATE OF o`, co.OrderID, co.TenantID).Scan(&order)
		if result.Error != nil {
			return result.Error
		}
		balance, e := money(order.Balance)
		amount, _ := money(proof.Amount)
		if result.RowsAffected != 1 || e != nil || balance.Cmp(amount) < 0 || order.Environment != c.Environment || order.Currency != proof.Currency || order.BrandID != co.BrandID || oneOf(order.Status, "canceled", "rejected") {
			return conflict("El pedido cambió o ya está pagado")
		}
		if e = tx.Raw(`INSERT INTO public.order_payments(tenant_id,order_id,currency,environment,kind,amount,status,method,provider,provider_reference,idempotency_key,request_digest,proof_digest,settled_at) VALUES(?,?,?,?,'charge',?::numeric,'succeeded','online','mercadopago',?,?,?,?,?) RETURNING id::text`, co.TenantID, co.OrderID, co.Currency, c.Environment, proof.Amount, proof.ID, "mp:"+proof.ID, proofDigest, proofDigest, proof.Paid).Scan(&rowID).Error; e != nil {
			return e
		}
		return tx.Exec(`INSERT INTO legui_private.mp_payment_allocations(provider_payment_id,component,order_payment_id) VALUES(?,'charge',?)`, paymentID, rowID).Error
	}
	invoiceID := co.InvoiceID
	if co.Purpose == "subscription" {
		if proof.Subscription == "" || proof.Subscription != co.RemoteID {
			return conflict("El pago no pertenece al mandato contratado")
		}
		var subscription struct {
			AccountID      string
			Status         string
			Environment    string
			Currency       string
			Amount         string
			IntervalMonths int
			PeriodEnd      *time.Time
			CancelAtEnd    bool
		}
		result := tx.Raw(`SELECT s.account_id::text,s.status,s.environment,s.currency,(s.contracted_amount*s.quantity)::text amount,s.interval_months,s.current_period_end period_end,s.cancel_at_period_end cancel_at_end FROM legui_private.subscriptions s JOIN legui_private.mp_subscription_mandates m ON m.subscription_id=s.id AND m.checkout_id=? WHERE s.id=? AND s.provider='mercadopago' AND s.provider_reference=? FOR UPDATE OF s`, co.ID, co.SubscriptionID, proof.Subscription).Scan(&subscription)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 || subscription.Status == "canceled" || subscription.CancelAtEnd || subscription.Environment != co.Environment || !sameMoney(subscription.Amount, co.Amount) || subscription.IntervalMonths != co.IntervalMonths {
			return conflict("Los términos de la suscripción cambiaron")
		}
		// A delayed old charge must not grant a fresh future period.
		start := proof.Paid.UTC()
		end := start.AddDate(0, subscription.IntervalMonths, 0)
		if subscription.PeriodEnd != nil {
			if !proof.Paid.After(subscription.PeriodEnd.Add(-48 * time.Hour)) {
				return conflict("El cobro se superpone con un período pagado")
			}
			start = *subscription.PeriodEnd
			if proof.Paid.After(start) {
				start = *proof.Paid
			}
			end = start.AddDate(0, subscription.IntervalMonths, 0)
		}
		invoiceID = uuid.NewString()
		if e := tx.Exec(`INSERT INTO legui_private.billing_invoices(id,account_id,subscription_id,environment,currency,amount_total,status,period_start,period_end,issued_at,idempotency_key,request_digest,description) VALUES(?,?,?,?,?,?::numeric,'open',?,?,? ,?,?,'Cuota Mercado Pago verificada')`, invoiceID, subscription.AccountID, co.SubscriptionID, co.Environment, co.Currency, co.Amount, start, end, proof.Paid, "mp-invoice:"+proof.ID, proofDigest).Error; e != nil {
			return e
		}
		if e := tx.Exec(`UPDATE legui_private.subscriptions SET status='active',current_period_start=?,current_period_end=?,activated_at=coalesce(activated_at,?) WHERE id=?`, start, end, proof.Paid, co.SubscriptionID).Error; e != nil {
			return e
		}
	}
	var invoice struct {
		AccountID   string
		Balance     string
		Status      string
		BrandID     string
		Environment string
	}
	result := tx.Raw(`SELECT i.account_id::text,i.balance_due::text balance,i.status,a.brand_id::text,i.environment FROM legui_private.billing_invoices i JOIN legui_private.billing_accounts a ON a.id=i.account_id WHERE i.id=? FOR UPDATE OF i`, invoiceID).Scan(&invoice)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 || invoice.BrandID != co.BrandID || invoice.Environment != co.Environment || invoice.Status != "open" || !sameMoney(invoice.Balance, proof.Amount) {
		return conflict("La factura cambió o ya está pagada")
	}
	if e := tx.Raw(`INSERT INTO legui_private.billing_payments(account_id,invoice_id,environment,currency,kind,amount,status,method,provider,provider_reference,idempotency_key,request_digest,proof_digest,settled_at) VALUES(?,?,?,?,'charge',?::numeric,'succeeded','online','mercadopago',?,?,?,?,?) RETURNING id::text`, invoice.AccountID, invoiceID, co.Environment, co.Currency, proof.Amount, proof.ID, "mp:"+proof.ID, proofDigest, proofDigest, proof.Paid).Scan(&rowID).Error; e != nil {
		return e
	}
	return tx.Exec(`INSERT INTO legui_private.mp_payment_allocations(provider_payment_id,component,billing_payment_id) VALUES(?,'charge',?)`, paymentID, rowID).Error
}
func (h *Handler) observeRefunds(ctx context.Context, c connection, token, payment string) (string, error) {
	raw, e := h.provider.callValue(ctx, "GET", "/v1/payments/"+payment+"/refunds", token, "", nil)
	if e != nil {
		return "", e
	}
	items, ok := raw.([]any)
	if !ok || len(items) > 100 {
		return "", errors.New("refund evidence unavailable")
	}
	state := "applied"
	e = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "mp-payment:"+c.Environment+":"+payment).Error; e != nil {
			return e
		}
		var original struct {
			ID               string
			CheckoutID       string
			OrderPaymentID   string
			BillingPaymentID string
			Refunded         string
		}
		result := tx.Raw(`SELECT p.id::text,coalesce(p.checkout_id::text,'') checkout_id,coalesce(a.order_payment_id::text,'') order_payment_id,coalesce(a.billing_payment_id::text,'') billing_payment_id,p.refunded_amount::text refunded FROM legui_private.mp_provider_payments p JOIN legui_private.mp_payment_allocations a ON a.provider_payment_id=p.id AND a.component='charge' WHERE p.connection_id=? AND p.environment=? AND p.payment_id=? FOR UPDATE OF p`, c.ID, c.Environment, payment).Scan(&original)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return conflict("La devolución requiere el pago original aplicado")
		}
		total := "0.00"
		for _, item := range items {
			refund := object(item)
			id := text(refund["id"])
			amount := text(refund["amount"])
			if text(refund["status"]) != "approved" {
				continue
			}
			if !resourcePattern.MatchString(id) || !positiveMoney(amount) {
				return errors.New("invalid refund evidence")
			}
			if resource := text(refund["payment_id"]); resource != "" && resource != payment {
				return errors.New("refund belongs to another payment")
			}
			var e error
			total, e = plus(total, amount)
			if e != nil {
				return e
			}
		}
		if !sameMoney(total, original.Refunded) {
			state = "review"
			return h.review(tx, c, original.CheckoutID, payment, "REFUND_TOTAL_MISMATCH", hash(jsonString(items)))
		}
		e := tx.Transaction(func(inner *gorm.DB) error {
			for _, item := range items {
				refund := object(item)
				if text(refund["status"]) != "approved" {
					continue
				}
				id, amount := text(refund["id"]), text(refund["amount"])
				component := "refund:" + id
				var count int64
				if e := inner.Raw(`SELECT count(*) FROM legui_private.mp_payment_allocations WHERE provider_payment_id=? AND component=?`, original.ID, component).Scan(&count).Error; e != nil {
					return e
				}
				if count > 0 {
					continue
				}
				at, e := providerTime(refund["date_created"])
				if e != nil {
					return e
				}
				var ledger string
				proof := hash(jsonString(refund))
				reference := payment + ":refund:" + id
				if original.OrderPaymentID != "" {
					e = inner.Raw(`INSERT INTO public.order_payments(tenant_id,order_id,currency,environment,kind,original_payment_id,amount,status,method,provider,provider_reference,idempotency_key,request_digest,proof_digest,settled_at) SELECT tenant_id,order_id,currency,environment,'refund',id,?::numeric,'succeeded','online','mercadopago',?,?,?,?,? FROM public.order_payments WHERE id=? RETURNING id::text`, amount, reference, "mp:"+reference, proof, proof, at, original.OrderPaymentID).Scan(&ledger).Error
				} else {
					e = inner.Raw(`INSERT INTO legui_private.billing_payments(account_id,invoice_id,environment,currency,kind,original_payment_id,amount,status,method,provider,provider_reference,idempotency_key,request_digest,proof_digest,settled_at) SELECT account_id,invoice_id,environment,currency,'refund',id,?::numeric,'succeeded','online','mercadopago',?,?,?,?,? FROM legui_private.billing_payments WHERE id=? RETURNING id::text`, amount, reference, "mp:"+reference, proof, proof, at, original.BillingPaymentID).Scan(&ledger).Error
				}
				if e != nil {
					return e
				}
				if ledger == "" {
					return errors.New("refund ledger missing")
				}
				var orderID, billingID any
				if original.OrderPaymentID != "" {
					orderID = ledger
				} else {
					billingID = ledger
				}
				if e = inner.Exec(`INSERT INTO legui_private.mp_payment_allocations(provider_payment_id,component,order_payment_id,billing_payment_id) VALUES(?,?,?,?)`, original.ID, component, orderID, billingID).Error; e != nil {
					return e
				}
			}
			return nil
		})
		if e != nil {
			state = "review"
			return h.review(tx, c, original.CheckoutID, payment, "REFUND_ALLOCATION_REVIEW", hash(jsonString(items)))
		}
		return nil
	})
	return state, e
}
