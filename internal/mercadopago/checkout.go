package mercadopago

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"leguiburger/internal/auth"
	"leguiburger/internal/buyeraccess"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type checkout struct {
	ActorKind         string
	ID                string
	Purpose           string
	ConnectionID      string
	ConnectionVersion string
	BrandID           string
	TenantID          string
	OrderID           string
	InvoiceID         string
	SubscriptionID    string
	Environment       string
	Currency          string
	Amount            string
	TargetVersion     int64
	IntervalMonths    int
	PayerEmail        string
	Reference         string
	State             string
	Claim             string
	RemoteID          string
	CheckoutURL       string
	Reason            string
	CreatedAt         time.Time
	ExpiresAt         time.Time
	SubmittedAt       *time.Time
	RequestDigest     string
}

const checkoutColumns = `actor_kind,id::text,purpose,connection_id::text,connection_version::text,brand_id::text,coalesce(tenant_id::text,'') tenant_id,coalesce(order_id::text,'') order_id,coalesce(invoice_id::text,'') invoice_id,coalesce(subscription_id::text,'') subscription_id,environment,currency,amount::text,target_version,coalesce(interval_months,0) interval_months,coalesce(payer_email,'') payer_email,reference,state,coalesce(claim::text,'') claim,coalesce(remote_id,'') remote_id,coalesce(checkout_url,'') checkout_url,coalesce(reason,'') reason,created_at,expires_at,submitted_at,request_digest`

type checkoutInput struct {
	Amount     string `json:"amount,omitempty"`
	PayerEmail string `json:"payer_email,omitempty"`
}

func (h *Handler) orderCheckout(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if !p.Manager() && !(p.Kind == "employee" && p.Role == "cashier") {
		return denied()
	}
	return h.beginCheckout(w, r, p, "order")
}
func (h *Handler) invoiceCheckout(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if e := manager(p); e != nil {
		return e
	}
	return h.beginCheckout(w, r, p, "invoice")
}
func (h *Handler) subscriptionCheckout(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if e := manager(p); e != nil {
		return e
	}
	return h.beginCheckout(w, r, p, "subscription")
}
func (h *Handler) beginCheckout(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal, purpose string) error {
	if p.Kind == "buyer" {
		if purpose != "order" {
			return denied()
		}
		if e := h.lockBuyer(r, p); e != nil {
			return e
		}
	}
	if e := h.limitAction(r, "checkout", p.ID, 60); e != nil {
		return e
	}
	id, e := routeID(r)
	if e != nil {
		return e
	}
	operation, e := key(r)
	if e != nil {
		return e
	}
	var input checkoutInput
	if e = decode(r, &input); e != nil {
		return e
	}
	if p.Kind == "buyer" && input.Amount != "" {
		return bad("El importe lo calcula el servidor")
	}
	if input.Amount != "" && !positiveMoney(input.Amount) {
		return bad("Importe inválido")
	}
	if input.PayerEmail != "" && !validEmail(input.PayerEmail) {
		return bad("Correo inválido")
	}
	var c connection
	if purpose == "order" {
		tenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
		if tenant == "" {
			tenant = p.TenantID
		}
		brand, e := h.scope(r, p, tenant)
		if e != nil {
			return e
		}
		var owns int64
		if e := h.db.WithContext(r.Context()).Raw(`SELECT count(*) FROM public.orders WHERE id=? AND tenant_id=?`, id, tenant).Scan(&owns).Error; e != nil {
			return e
		}
		if owns != 1 {
			return missing()
		}
		result := h.db.WithContext(r.Context()).Raw(`SELECT `+connectionColumns+` FROM legui_private.mp_connections WHERE scope='business' AND brand_id=? AND (tenant_id=?::uuid OR tenant_id IS NULL) AND environment=? AND state='connected' ORDER BY tenant_id NULLS LAST LIMIT 1`, brand, tenant, h.cfg.Environment).Scan(&c)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return conflict("Conectá una cuenta de Mercado Pago para este local")
		}
	} else {
		c, e = h.platform(r.Context())
		if e != nil {
			return e
		}
	}
	current, token, e := h.account(r.Context(), c.ID)
	if e != nil {
		return e
	}
	c = current
	var result checkout
	requestDigest := hash(jsonString(map[string]string{"target": id, "purpose": purpose, "amount": input.Amount, "payer_email": input.PayerEmail}))
	e = h.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {

		if p.Kind == "buyer" {
			grant, e := buyeraccess.ResolveRequest(r.Context(), tx, r, true)
			if e != nil || grant.OrderID != p.ID || grant.TenantID != p.TenantID {
				return denied()
			}
		} else {
			if e := auth.LockSession(r.Context(), tx); e != nil {
				return e
			}
		}

		if e := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "mp-checkout:"+purpose+":"+id).Error; e != nil {
			return e
		}
		if p.Kind == "buyer" {
			var accepted bool
			if e := tx.Raw(`SELECT s.code NOT IN ('pending','canceled','rejected') AND o.total_due>o.total_paid FROM public.orders o JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE o.id=? AND o.tenant_id=? FOR UPDATE OF o`, id, p.TenantID).Scan(&accepted).Error; e != nil {
				return e
			}
			if !accepted {
				return conflict("El restaurante tiene que aceptar el pedido antes de habilitar el pago online")
			}
		}
		existing := tx.Raw(`SELECT `+checkoutColumns+` FROM legui_private.mp_checkouts WHERE actor_kind=? AND actor_id=? AND purpose=? AND operation_key=?`, p.Kind, p.ID, purpose, operation).Scan(&result)
		if existing.Error != nil {
			return existing.Error
		}
		if existing.RowsAffected > 0 {
			if result.RequestDigest != requestDigest {
				return conflict("La clave ya fue usada con otros datos")
			}
			if !p.Platform() && result.BrandID != p.BrandID {
				return denied()
			}
			return nil
		}
		var target struct {
			BrandID        string
			TenantID       string
			Environment    string
			Currency       string
			Amount         string
			Version        int64
			IntervalMonths int
			PayerEmail     string
		}
		var selected *gorm.DB

		switch purpose {
		case "order":
			tenant := r.Header.Get("X-Tenant-ID")
			if tenant == "" {
				tenant = p.TenantID
			}
			selected = tx.Raw(`SELECT t.brand_id::text,o.tenant_id::text,o.payment_environment environment,o.currency,(o.total_due-o.total_paid)::text amount,o.version FROM public.orders o JOIN public.tenants t ON t.id=o.tenant_id AND t.active JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE o.id=? AND o.tenant_id=? AND s.code NOT IN ('canceled','rejected') FOR UPDATE OF o`, id, tenant).Scan(&target)
		case "invoice":
			selected = tx.Raw(`SELECT a.brand_id::text,coalesce(a.tenant_id::text,'') tenant_id,i.environment,i.currency,i.balance_due::text amount,0 AS version,a.contact_email payer_email FROM legui_private.billing_invoices i JOIN legui_private.billing_accounts a ON a.id=i.account_id WHERE i.id=? AND i.status='open' AND a.closed_at IS NULL FOR UPDATE OF i`, id).Scan(&target)
		case "subscription":
			selected = tx.Raw(`SELECT a.brand_id::text,coalesce(a.tenant_id::text,'') tenant_id,s.environment,s.currency,(s.contracted_amount*s.quantity)::text amount,s.version,s.interval_months,a.contact_email payer_email FROM legui_private.subscriptions s JOIN legui_private.billing_accounts a ON a.id=s.account_id WHERE s.id=? AND s.status IN ('incomplete','trialing','active','past_due') AND s.provider_reference IS NULL AND a.closed_at IS NULL FOR UPDATE OF s`, id).Scan(&target)
		}
		if selected.Error != nil {
			return selected.Error
		}
		if selected.RowsAffected != 1 {
			return missing()
		}
		if !p.Platform() && target.BrandID != p.BrandID {
			return denied()
		}
		if e := h.enabled(target.Environment); e != nil {
			return e
		}
		if target.Environment != c.Environment || target.Currency != "ARS" || !positiveMoney(target.Amount) {
			return conflict("El recurso no tiene un saldo cobrable en este entorno")
		}
		if purpose == "order" {
			if target.BrandID != c.BrandID || c.TenantID != "" && c.TenantID != target.TenantID {
				return denied()
			}
		}
		if input.Amount != "" {
			if purpose != "order" {
				return bad("El importe de la membresía lo calcula el servidor")
			}
			a, _ := money(input.Amount)
			b, _ := money(target.Amount)
			if a.Cmp(b) > 0 {
				return bad("El importe supera el saldo")
			}
			target.Amount = a.FloatString(2)
		}
		if input.PayerEmail != "" {
			target.PayerEmail = input.PayerEmail
		}
		if purpose == "subscription" && !validEmail(target.PayerEmail) {
			return bad("La suscripción necesita el correo del pagador")
		}
		// An unresolved creation must be reconciled, never replaced by another blind POST.
		var active int64
		if e := tx.Raw(`SELECT count(*) FROM legui_private.mp_checkouts x WHERE purpose=? AND coalesce(order_id,invoice_id,subscription_id)=?::uuid AND state IN ('prepared','submitting','created','review') AND (expires_at>clock_timestamp() OR state IN ('submitting','review')) AND NOT EXISTS(SELECT 1 FROM legui_private.mp_provider_payments pp JOIN legui_private.mp_payment_allocations a ON a.provider_payment_id=pp.id WHERE pp.checkout_id=x.id AND a.component='charge')`, purpose, id).Scan(&active).Error; e != nil {
			return e
		}
		if active > 0 {
			return conflict("Ya existe un checkout pendiente para este recurso. Revisalo antes de crear otro")
		}
		var order, invoice, subscription any
		if purpose == "order" {
			order = id
		} else if purpose == "invoice" {
			invoice = id
		} else {
			subscription = id
		}
		resultID := uuid.NewString()
		reference := "LEG:UI:" + resultID
		selected = tx.Raw(`INSERT INTO legui_private.mp_checkouts(id,purpose,connection_id,connection_version,brand_id,tenant_id,order_id,invoice_id,subscription_id,environment,currency,amount,target_version,interval_months,payer_email,reference,actor_kind,actor_id,operation_key,request_digest) VALUES(?,?,?,?,?,nullif(?,'')::uuid,?,?,?,?,?,?::numeric,?,nullif(?,0),nullif(?,''),?,?,?,?,?) RETURNING `+checkoutColumns, resultID, purpose, c.ID, c.Version, target.BrandID, target.TenantID, order, invoice, subscription, target.Environment, target.Currency, target.Amount, target.Version, target.IntervalMonths, target.PayerEmail, reference, p.Kind, p.ID, operation, requestDigest).Scan(&result)
		if selected.Error != nil {
			return selected.Error
		}
		if p.Kind == "buyer" {
			return tx.Exec(`INSERT INTO public.activity_events(brand_id,tenant_id,event_name,source,environment,facts) VALUES(?,?,'payments.buyer_checkout','server','production','{}')`, p.BrandID, p.TenantID).Error
		}
		return auth.Audit(r.Context(), tx, p, "payments.checkout_prepared", resultID, map[string]string{"purpose": purpose})
	})
	if e != nil {
		return e
	}
	if result.State == "prepared" {
		result, e = h.submitCheckout(r.Context(), result, c, token)
		if e != nil {
			return e
		}
	}
	respond(w, 200, checkoutPublic(result))
	return nil
}
func checkoutPublic(c checkout) map[string]any {
	return map[string]any{"id": c.ID, "purpose": c.Purpose, "state": c.State, "amount": c.Amount, "currency": c.Currency, "checkout_url": c.CheckoutURL, "reason": c.Reason, "created_at": c.CreatedAt, "expires_at": c.ExpiresAt}
}
func (h *Handler) submitCheckout(ctx context.Context, c checkout, account connection, token string) (checkout, error) {
	claim := uuid.NewString()
	result := h.db.WithContext(ctx).Exec(`UPDATE legui_private.mp_checkouts SET state='submitting',claim=?,lease_until=clock_timestamp()+interval '1 minute',submitted_at=clock_timestamp() WHERE id=? AND state='prepared' AND expires_at>clock_timestamp() AND EXISTS(SELECT 1 FROM legui_private.mp_connections WHERE id=mp_checkouts.connection_id AND version=mp_checkouts.connection_version AND state='connected')`, claim, c.ID)
	if result.Error != nil {
		return c, result.Error
	}
	if result.RowsAffected != 1 {
		return c, conflict("El checkout cambió; consultá su estado")
	}
	path := "/checkout/preferences"
	back := h.cfg.PublicOrigin + "/payments/return?checkout=" + url.QueryEscape(c.ID)
	if c.ActorKind == "buyer" && c.Purpose == "order" {
		back = h.cfg.PublicOrigin + "/catalogo/" + c.TenantID + "?pedido=" + c.OrderID
	}
	var payload map[string]any
	if c.Purpose == "subscription" {
		path = "/preapproval"
		payload = map[string]any{"reason": "Membresía LeguiBurger", "external_reference": c.Reference, "payer_email": c.PayerEmail, "auto_recurring": map[string]any{"frequency": c.IntervalMonths, "frequency_type": "months", "transaction_amount": json.Number(c.Amount), "currency_id": c.Currency}, "back_url": back, "status": "pending"}
	} else {
		payload = map[string]any{"items": []any{map[string]any{"id": c.Reference, "title": "LeguiBurger · " + map[string]string{"order": "Pedido", "invoice": "Membresía"}[c.Purpose], "quantity": 1, "currency_id": c.Currency, "unit_price": json.Number(c.Amount)}}, "external_reference": c.Reference, "notification_url": h.cfg.APIOrigin + "/api/webhooks/mercadopago/" + c.ConnectionID, "back_urls": map[string]string{"success": back, "pending": back, "failure": back}, "auto_return": "approved", "expires": true, "expiration_date_to": c.ExpiresAt.Format(time.RFC3339)}
	}
	remote, e := h.provider.call(ctx, "POST", path, token, "", payload)
	if e != nil {
		state, reason := "review", "PROVIDER_OUTCOME_UNKNOWN"
		if failure, ok := e.(*providerFailure); ok && !failure.Ambiguous {
			state, reason = "failed", "PROVIDER_REJECTED"
		}
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = h.db.WithContext(cleanup).Exec(`UPDATE legui_private.mp_checkouts SET state=?,reason=?,lease_until=NULL WHERE id=? AND claim=? AND state='submitting'`, state, reason, c.ID, claim).Error
		c.State = state
		c.Reason = reason
		return c, nil
	}
	return h.finishCheckout(ctx, c, account, claim, remote)
}
func (h *Handler) finishCheckout(ctx context.Context, c checkout, account connection, claim string, remote map[string]any) (checkout, error) {
	remoteID, link := text(remote["id"]), text(remote["init_point"])
	if c.Environment == "sandbox" && text(remote["sandbox_init_point"]) != "" {
		link = text(remote["sandbox_init_point"])
	}
	valid := resourcePattern.MatchString(remoteID) && checkoutURL(link, c.Environment) && text(remote["external_reference"]) == c.Reference
	if c.Purpose == "subscription" {
		recurring := object(remote["auto_recurring"])
		valid = valid && text(remote["collector_id"]) == account.CollectorID && sameMoney(text(recurring["transaction_amount"]), c.Amount) && text(recurring["currency_id"]) == c.Currency && text(recurring["frequency"]) == json.Number(stringInt(c.IntervalMonths)).String() && text(recurring["frequency_type"]) == "months"
	} else {
		valid = valid && text(remote["collector_id"]) == account.CollectorID
		items, _ := remote["items"].([]any)
		valid = valid && len(items) == 1
		if len(items) == 1 {
			item := object(items[0])
			valid = valid && sameMoney(text(item["unit_price"]), c.Amount) && text(item["quantity"]) == "1" && text(item["currency_id"]) == c.Currency
		}
	}
	if !valid {
		_ = h.db.WithContext(ctx).Exec(`UPDATE legui_private.mp_checkouts SET state='review',reason='PROVIDER_TERMS_MISMATCH',lease_until=NULL WHERE id=? AND claim=? AND state='submitting'`, c.ID, claim).Error
		c.State = "review"
		c.Reason = "PROVIDER_TERMS_MISMATCH"
		return c, nil
	}
	e := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		r := tx.Exec(`UPDATE legui_private.mp_checkouts SET state='created',remote_id=?,checkout_url=?,reason=NULL,lease_until=NULL,reconcile_at=clock_timestamp() WHERE id=? AND claim=? AND state IN ('submitting','review') AND EXISTS(SELECT 1 FROM legui_private.mp_connections WHERE id=mp_checkouts.connection_id AND version=mp_checkouts.connection_version AND state='connected')`, remoteID, link, c.ID, claim)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected != 1 {
			return conflict("La cuenta o el checkout cambió durante la creación")
		}
		if c.Purpose == "subscription" {
			updated, e := providerTime(remote["last_modified"])
			if e != nil {
				return e
			}
			if e = tx.Exec(`INSERT INTO legui_private.mp_subscription_mandates(subscription_id,checkout_id,connection_id,environment,remote_id,status,provider_updated_at,proof_digest) VALUES(?,?,?,?,?,?,?,?)`, c.SubscriptionID, c.ID, c.ConnectionID, c.Environment, remoteID, text(remote["status"]), updated, mandateDigest(remote)).Error; e != nil {
				return e
			}
			r = tx.Exec(`UPDATE legui_private.subscriptions SET provider='mercadopago',provider_reference=? WHERE id=? AND provider_reference IS NULL AND status<>'canceled' AND version=?`, remoteID, c.SubscriptionID, c.TargetVersion)
			if r.Error != nil {
				return r.Error
			}
			if r.RowsAffected != 1 {
				return conflict("La suscripción cambió durante la creación")
			}
		}
		return nil
	})
	if e != nil {
		return c, e
	}
	c.State = "created"
	c.RemoteID = remoteID
	c.CheckoutURL = link
	return c, nil
}
func stringInt(value int) string { return strconv.Itoa(value) }
