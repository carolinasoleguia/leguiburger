package mercadopago

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"io"
	"leguiburger/internal/auth"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) webhook(w http.ResponseWriter, r *http.Request) error {
	if !h.cfg.ready() {
		return notConfigured()
	}
	id, e := routeID(r)
	if e != nil {
		return e
	}
	query := r.URL.Query()
	for _, key := range []string{"data.id", "type", "topic"} {
		if len(query[key]) > 1 {
			return bad("Notificación ambigua")
		}
	}
	resource := query.Get("data.id")
	if len(r.Header.Values("X-Signature")) != 1 || len(r.Header.Values("X-Request-ID")) != 1 || !signatureValid(r.Header.Get("X-Signature"), r.Header.Get("X-Request-ID"), resource, h.cfg.WebhookSecret, time.Now()) {
		return &apiError{401, "INVALID_SIGNATURE", "Firma de notificación inválida"}
	}
	var body struct {
		Type string `json:"type"`
		Data struct {
			ID any `json:"id"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	decoder.UseNumber()
	if decoder.Decode(&body) != nil {
		return bad("Notificación inválida")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return bad("Notificación ambigua")
	}
	if body.Data.ID != nil && text(body.Data.ID) != resource {
		return bad("El recurso no coincide con la firma")
	}
	topic := query.Get("type")
	if topic == "" {
		topic = query.Get("topic")
	}
	if topic == "" {
		topic = body.Type
	}
	if body.Type != "" && topic != body.Type {
		return bad("Tipo de notificación ambiguo")
	}
	if !oneOf(topic, "payment", "subscription_preapproval", "subscription_authorized_payment", "merchant_order", "topic_merchant_order_wh") {
		return bad("Tipo de notificación no soportado")
	}
	var c connection
	result := h.db.WithContext(r.Context()).Raw(`SELECT `+connectionColumns+` FROM legui_private.mp_connections WHERE id=? AND environment=?`, id, h.cfg.Environment).Scan(&c)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return missing()
	}
	if c.Scope != "platform" && strings.HasPrefix(topic, "subscription_") {
		return denied()
	}
	eventKey := hash(topic + ":" + resource + ":" + r.Header.Get("X-Request-ID"))
	state := "received"
	var processed any
	if oneOf(topic, "merchant_order", "topic_merchant_order_wh") {
		state = "ignored"
		processed = time.Now()
	}
	if e = h.db.WithContext(r.Context()).Exec(`INSERT INTO legui_private.mp_webhook_inbox(connection_id,environment,event_key,topic,resource_id,state,processed_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT DO NOTHING`, id, c.Environment, eventKey, topic, resource, state, processed).Error; e != nil {
		return e
	}
	respond(w, 202, map[string]bool{"received": true})
	return nil
}
func (h *Handler) platformWebhook(w http.ResponseWriter, r *http.Request) error {
	if !h.cfg.ready() {
		return notConfigured()
	}
	var id string
	result := h.db.WithContext(r.Context()).Raw(`SELECT id::text FROM legui_private.mp_connections WHERE scope='platform' AND environment=?`, h.cfg.Environment).Scan(&id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return notConfigured()
	}
	r.SetPathValue("id", id)
	return h.webhook(w, r)
}
func (h *Handler) StartWorker(ctx context.Context) {
	if !h.cfg.ready() {
		log.Print("Mercado Pago pendiente de configuración; no se enviarán operaciones al proveedor")
		return
	}
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				work, cancel := context.WithTimeout(ctx, 50*time.Second)
				if e := h.Tick(work); e != nil {
					log.Printf("mercadopago worker retry pending (%T)", e)
				}
				cancel()
			}
		}
	}()
}
func (h *Handler) Tick(ctx context.Context) error {
	if !h.cfg.ready() {
		return notConfigured()
	}
	if e := h.expireMemberships(ctx); e != nil {
		return e
	}
	if e := h.db.WithContext(ctx).Exec(`UPDATE legui_private.mp_checkouts SET state='review',reason='SUBMISSION_LEASE_EXPIRED' WHERE state='submitting' AND lease_until<clock_timestamp()`).Error; e != nil {
		return e
	}
	if e := h.db.WithContext(ctx).Exec(`UPDATE legui_private.mp_oauth_attempts SET status='review',verifier_cipher=NULL WHERE status IN ('pending','submitting') AND expires_at<clock_timestamp()`).Error; e != nil {
		return e
	}
	if e := h.db.WithContext(ctx).Exec(`UPDATE legui_private.mp_connections c SET state='needs_reconnect',updated_at=clock_timestamp() WHERE state='connecting' AND NOT EXISTS(SELECT 1 FROM legui_private.mp_oauth_attempts a WHERE a.connection_id=c.id AND a.connection_version=c.version AND a.status IN ('pending','submitting') AND a.expires_at>clock_timestamp())`).Error; e != nil {
		return e
	}
	for index := 0; index < 5; index++ {
		found, e := h.processInbox(ctx)
		if e != nil {
			return e
		}
		if !found {
			break
		}
	}
	a := h.reconcileOneCheckout(ctx, "")
	b := h.scanOne(ctx, "")
	c := h.reconcileOperation(ctx)
	return errors.Join(a, b, c)
}
func (h *Handler) processInbox(ctx context.Context) (bool, error) {
	var job struct {
		ID           string
		ConnectionID string
		Topic        string
		ResourceID   string
		Attempts     int
		Claim        string
	}
	result := h.db.WithContext(ctx).Raw(`WITH chosen AS(SELECT id FROM legui_private.mp_webhook_inbox WHERE attempts<8 AND next_attempt_at<=clock_timestamp() AND (state IN ('received','failed') OR state='processing' AND lease_until<clock_timestamp()) ORDER BY next_attempt_at LIMIT 1 FOR UPDATE SKIP LOCKED) UPDATE legui_private.mp_webhook_inbox i SET state='processing',attempts=attempts+1,claim=gen_random_uuid(),lease_until=clock_timestamp()+interval '2 minutes' FROM chosen WHERE i.id=chosen.id RETURNING i.id::text,i.connection_id::text,i.topic,i.resource_id,i.attempts,i.claim::text`).Scan(&job)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		return false, nil
	}
	state, e := h.processResource(ctx, job.ConnectionID, job.Topic, job.ResourceID)
	if e != nil {
		status := "failed"
		if job.Attempts >= 8 {
			status = "review"
		}
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		err := h.db.WithContext(cleanup).Exec(`UPDATE legui_private.mp_webhook_inbox SET state=?,error_code='PROVIDER_UNCONFIRMED',next_attempt_at=clock_timestamp()+make_interval(secs=>?::int),lease_until=NULL WHERE id=? AND claim=?`, status, 15*(1<<min(job.Attempts, 6)), job.ID, job.Claim).Error
		return true, err
	}
	status := "processed"
	if state == "review" {
		status = "review"
	}
	return true, h.db.WithContext(ctx).Exec(`UPDATE legui_private.mp_webhook_inbox SET state=?,processed_at=clock_timestamp(),error_code=NULL,lease_until=NULL WHERE id=? AND claim=?`, status, job.ID, job.Claim).Error
}
func (h *Handler) processResource(ctx context.Context, connectionID, topic, id string) (string, error) {
	if topic == "payment" {
		return h.observePayment(ctx, connectionID, id)
	}
	if topic == "subscription_preapproval" {
		return h.observeMandate(ctx, connectionID, id)
	}
	if topic == "subscription_authorized_payment" {
		c, token, e := h.account(ctx, connectionID)
		if e != nil {
			return "", e
		}
		if c.Scope != "platform" {
			return "", denied()
		}
		v, e := h.provider.call(ctx, "GET", "/authorized_payments/"+id, token, "", nil)
		if e != nil {
			return "", e
		}
		payment := text(object(v["payment"])["id"])
		if payment == "" {
			return "", errors.New("authorized payment has no payment yet")
		}
		return h.observePayment(ctx, connectionID, payment)
	}
	return "ignored", nil
}
func (h *Handler) observeMandate(ctx context.Context, connectionID, id string) (string, error) {
	c, token, e := h.account(ctx, connectionID)
	if e != nil {
		return "", e
	}
	if c.Scope != "platform" || !resourcePattern.MatchString(id) {
		return "", denied()
	}
	remote, e := h.provider.call(ctx, "GET", "/preapproval/"+id, token, "", nil)
	if e != nil {
		return "", e
	}
	updated, e := providerTime(remote["last_modified"])
	if e != nil {
		return "", e
	}
	state := "observed"
	digest := mandateDigest(remote)
	e = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var co checkout
		result := tx.Raw(`SELECT `+checkoutColumns+` FROM legui_private.mp_checkouts WHERE connection_id=? AND environment=? AND purpose='subscription' AND remote_id=? FOR UPDATE`, c.ID, c.Environment, id).Scan(&co)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			state = "review"
			return h.review(tx, c, "", id, "UNOWNED_MANDATE", digest)
		}
		recurring := object(remote["auto_recurring"])
		status := text(remote["status"])
		if text(remote["id"]) != id || text(remote["external_reference"]) != co.Reference || text(remote["collector_id"]) != c.CollectorID || !sameMoney(text(recurring["transaction_amount"]), co.Amount) || text(recurring["currency_id"]) != co.Currency || text(recurring["frequency"]) != stringInt(co.IntervalMonths) || text(recurring["frequency_type"]) != "months" || !oneOf(status, "pending", "authorized", "paused", "cancelled") || updated.After(time.Now().Add(5*time.Minute)) {
			state = "review"
			return h.review(tx, c, co.ID, id, "MANDATE_TERMS_MISMATCH", digest)
		}
		var prior struct {
			Updated     time.Time
			ProofDigest string
		}
		if e := tx.Raw(`SELECT provider_updated_at updated,proof_digest FROM legui_private.mp_subscription_mandates WHERE subscription_id=? FOR UPDATE`, co.SubscriptionID).Scan(&prior).Error; e != nil {
			return e
		}
		if updated.Before(prior.Updated) {
			state = "stale"
			return nil
		}
		if updated.Equal(prior.Updated) && prior.ProofDigest != digest {
			state = "review"
			return h.review(tx, c, co.ID, id, "MANDATE_VERSION_CONFLICT", digest)
		}
		if e := tx.Exec(`UPDATE legui_private.mp_subscription_mandates SET status=?,provider_updated_at=?,proof_digest=?,cancel_pending=CASE WHEN ?='cancelled' THEN false ELSE cancel_pending END,cancel_at_period_end=CASE WHEN ?='cancelled' THEN true ELSE cancel_at_period_end END WHERE subscription_id=?`, status, updated, digest, status, status, co.SubscriptionID).Error; e != nil {
			return e
		}
		if status == "cancelled" {
			return tx.Exec(`UPDATE legui_private.subscriptions SET cancel_at_period_end=true WHERE id=? AND status<>'canceled'`, co.SubscriptionID).Error
		}
		// Authorized merely permits future charges; it never grants a paid period.
		return nil
	})
	return state, e
}
func mandateDigest(remote map[string]any) string {
	return hash(jsonString(map[string]any{"id": remote["id"], "reference": remote["external_reference"], "status": remote["status"], "collector": remote["collector_id"], "recurring": remote["auto_recurring"], "updated": remote["last_modified"]}))
}
func (h *Handler) reconcileOneCheckout(ctx context.Context, connectionID string) error {
	var co checkout
	var picked string
	result := h.db.WithContext(ctx).Raw(`WITH chosen AS(SELECT id FROM legui_private.mp_checkouts WHERE (?='' OR connection_id=nullif(?,'')::uuid) AND state IN ('created','review') AND reconcile_at<=clock_timestamp() AND created_at>clock_timestamp()-interval '7 days' AND reason IS DISTINCT FROM 'PROVIDER_TERMS_MISMATCH' ORDER BY reconcile_at LIMIT 1 FOR UPDATE SKIP LOCKED) UPDATE legui_private.mp_checkouts x SET reconcile_checks=reconcile_checks+1,reconcile_at=clock_timestamp()+make_interval(mins=>least(120,5*(reconcile_checks+1))) FROM chosen WHERE x.id=chosen.id RETURNING x.id::text`, connectionID, connectionID).Scan(&picked)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}
	if e := h.db.WithContext(ctx).Raw(`SELECT `+checkoutColumns+` FROM legui_private.mp_checkouts WHERE id=?`, picked).Scan(&co).Error; e != nil {
		return e
	}
	c, token, e := h.account(ctx, co.ConnectionID)
	if e != nil {
		return e
	}
	if co.RemoteID == "" {
		path := "/checkout/preferences/search?"
		if co.Purpose == "subscription" {
			path = "/preapproval/search?"
		}
		response, e := h.provider.call(ctx, "GET", path+url.Values{"external_reference": {co.Reference}, "limit": {"2"}}.Encode(), token, "", nil)
		if e != nil {
			return e
		}
		raw, ok := response["elements"].([]any)
		if !ok {
			raw, _ = response["results"].([]any)
		}
		if len(raw) > 1 {
			return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				return h.review(tx, c, co.ID, co.ID, "AMBIGUOUS_REMOTE_CREATION", hash(co.Reference))
			})
		}
		if len(raw) == 1 {
			remoteID := text(object(raw[0])["id"])
			if !resourcePattern.MatchString(remoteID) {
				return errors.New("invalid checkout discovery")
			}
			path := "/checkout/preferences/"
			if co.Purpose == "subscription" {
				path = "/preapproval/"
			}
			remote, e := h.provider.call(ctx, "GET", path+remoteID, token, "", nil)
			if e != nil {
				return e
			}
			co, e = h.finishCheckout(ctx, co, c, co.Claim, remote)
			if e != nil {
				return e
			}
		}
	}
	if co.Purpose == "subscription" && co.RemoteID != "" {
		if _, e = h.observeMandate(ctx, c.ID, co.RemoteID); e != nil {
			return e
		}
	}
	response, e := h.provider.call(ctx, "GET", "/v1/payments/search?"+url.Values{"external_reference": {co.Reference}, "collector.id": {c.CollectorID}, "limit": {"100"}, "sort": {"date_created"}, "criteria": {"asc"}}.Encode(), token, "", nil)
	if e != nil {
		return e
	}
	items, _ := response["results"].([]any)
	for _, item := range items {
		payment := object(item)
		if text(payment["external_reference"]) != co.Reference {
			continue
		}
		id := text(payment["id"])
		if !resourcePattern.MatchString(id) {
			return errors.New("invalid payment discovery")
		}
		if _, e = h.observePayment(ctx, c.ID, id); e != nil {
			return e
		}
	}
	return nil
}
func (h *Handler) scanOne(ctx context.Context, connectionID string) error {
	var cursor struct {
		ConnectionID string
		WindowStart  time.Time
		WindowEnd    time.Time
		PageOffset   int
		Claim        string
	}
	e := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec(`INSERT INTO legui_private.mp_scan_cursors(connection_id) SELECT id FROM legui_private.mp_connections WHERE state='connected' AND environment=? AND (?='' OR id=nullif(?,'')::uuid) ON CONFLICT DO NOTHING`, h.cfg.Environment, connectionID, connectionID).Error; e != nil {
			return e
		}
		return tx.Raw(`WITH chosen AS(SELECT x.connection_id FROM legui_private.mp_scan_cursors x JOIN legui_private.mp_connections c ON c.id=x.connection_id AND c.state='connected' WHERE c.environment=? AND (?='' OR c.id=nullif(?,'')::uuid) AND x.next_scan_at<=clock_timestamp() AND (x.lease_until IS NULL OR x.lease_until<clock_timestamp()) ORDER BY x.next_scan_at LIMIT 1 FOR UPDATE OF x SKIP LOCKED) UPDATE legui_private.mp_scan_cursors x SET claim=gen_random_uuid(),lease_until=clock_timestamp()+interval '2 minutes',window_start=coalesce(window_start,scanned_until-interval '5 minutes'),window_end=coalesce(window_end,least(scanned_until+interval '24 hours',clock_timestamp())) FROM chosen WHERE x.connection_id=chosen.connection_id RETURNING x.connection_id::text,x.window_start,x.window_end,x.page_offset,x.claim::text`, h.cfg.Environment, connectionID, connectionID).Scan(&cursor).Error
	})
	if e != nil {
		return e
	}
	if cursor.ConnectionID == "" {
		return nil
	}
	c, token, e := h.account(ctx, cursor.ConnectionID)
	if e != nil {
		return e
	}
	response, e := h.provider.call(ctx, "GET", "/v1/payments/search?"+url.Values{"collector.id": {c.CollectorID}, "sort": {"date_last_updated"}, "criteria": {"asc"}, "range": {"date_last_updated"}, "begin_date": {cursor.WindowStart.UTC().Format("2006-01-02T15:04:05.000Z07:00")}, "end_date": {cursor.WindowEnd.UTC().Format("2006-01-02T15:04:05.000Z07:00")}, "offset": {strconv.Itoa(cursor.PageOffset)}, "limit": {"50"}}.Encode(), token, "", nil)
	if e != nil {
		return e
	}
	items, ok := response["results"].([]any)
	if !ok || len(items) > 50 {
		return errors.New("invalid discovery page")
	}
	for _, item := range items {
		v := object(item)
		reference := text(v["external_reference"])
		if !strings.HasPrefix(reference, "LEG:UI:") {
			continue
		}
		id := text(v["id"])
		if !resourcePattern.MatchString(id) {
			return errors.New("invalid discovered payment")
		}
		if _, e = h.observePayment(ctx, c.ID, id); e != nil {
			return e
		}
	}
	if len(items) == 50 {
		return h.db.WithContext(ctx).Exec(`UPDATE legui_private.mp_scan_cursors SET page_offset=page_offset+50,claim=NULL,lease_until=NULL WHERE connection_id=? AND claim=?`, cursor.ConnectionID, cursor.Claim).Error
	}
	return h.db.WithContext(ctx).Exec(`UPDATE legui_private.mp_scan_cursors SET scanned_until=window_end,window_start=NULL,window_end=NULL,page_offset=0,claim=NULL,lease_until=NULL,next_scan_at=clock_timestamp()+interval '5 minutes' WHERE connection_id=? AND claim=?`, cursor.ConnectionID, cursor.Claim).Error
}
func (h *Handler) expireMemberships(ctx context.Context) error {
	return h.db.WithContext(ctx).Exec(`UPDATE legui_private.subscriptions SET status=CASE WHEN cancel_at_period_end THEN 'canceled' ELSE 'past_due' END WHERE provider='mercadopago' AND status IN ('active','trialing') AND coalesce(current_period_end,trial_ends_at)<=clock_timestamp()`).Error
}
func (h *Handler) reconcileHTTP(w http.ResponseWriter, r *http.Request, p auth.SessionPrincipal) error {
	if e := manager(p); e != nil {
		return e
	}
	if e := h.limitAction(r, "reconcile", p.ID, 10); e != nil {
		return e
	}
	var input struct {
		ConnectionID string `json:"connection_id"`
	}
	if e := decode(r, &input); e != nil {
		return e
	}
	if !validID(input.ConnectionID) {
		return bad("Conexión inválida")
	}
	c, e := h.connection(r, p, input.ConnectionID)
	if e != nil {
		return e
	}
	if e = h.enabled(c.Environment); e != nil {
		return e
	}
	if e = h.reconcileOneCheckout(r.Context(), c.ID); e != nil {
		return e
	}
	if e = h.scanOne(r.Context(), c.ID); e != nil {
		return e
	}
	respond(w, 200, map[string]bool{"reconciled": true})
	return nil
}
