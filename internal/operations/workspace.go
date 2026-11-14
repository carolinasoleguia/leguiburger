package operations

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

// Kitchen sees preparation snapshots only. Customer identity, address and payments
// are intentionally absent even when the same order has a public tracking link.
func (h *Handler) kitchenOrders(r *http.Request, p Principal) (Result, error) {
	if !p.Operator() {
		return Result{}, forbidden()
	}
	scope, e := h.tenantScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v) ORDER BY v.created_at,v.id),'[]'::jsonb))::text payload FROM (
 SELECT o.id,o.created_at,o.version,s.code status,
 coalesce((SELECT jsonb_agg(jsonb_build_object('product_name',d.product_name,'quantity',d.quantity,'note',d.preparation_note,'extras',coalesce((SELECT jsonb_agg(jsonb_build_object('name',x.extra_name,'quantity',x.quantity,'group_name',x.option_group_name)) FROM public.order_detail_extras x WHERE x.order_detail_id=d.id),'[]'::jsonb))) FROM public.order_details d WHERE d.order_id=o.id),'[]'::jsonb) items
 FROM public.orders o JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE o.tenant_id=? AND o.payment_environment='production' AND s.code IN ('accepted','preparing','ready') ORDER BY o.created_at,o.id LIMIT 100)v`, scope.TenantID))
}
func (h *Handler) kitchenStatus(r *http.Request, p Principal) (Result, error) {
	if !p.Operator() {
		return Result{}, forbidden()
	}
	scope, e := h.tenantScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	id, e := resourceID(r)
	if e != nil {
		return Result{}, e
	}
	input, e := decode[ChangeOrderStatusRequest](r)
	if e != nil {
		return Result{}, e
	}
	if input.ExpectedVersion < 1 || !isOneOf(input.Status, "preparing", "ready") || input.Reason != "" {
		return Result{}, badInput("La cocina sólo puede iniciar o terminar la preparación")
	}
	return h.mutate(r, p, scope, "kitchen.status", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		from := "accepted"
		if input.Status == "ready" {
			from = "preparing"
		}
		data, e := jsonQuery(tx, `UPDATE public.orders SET order_status_id=(SELECT id FROM public.order_statuses_lookup WHERE code=?) WHERE id=? AND tenant_id=? AND payment_environment='production' AND version=? AND order_status_id=(SELECT id FROM public.order_statuses_lookup WHERE code=?) RETURNING jsonb_build_object('id',id,'version',version,'status',?::text)::text payload`, input.Status, id, scope.TenantID, input.ExpectedVersion, from, input.Status)
		if e != nil {
			return nil, 0, e
		}
		return data, 200, nil
	})
}
func (h *Handler) notifications(r *http.Request, p Principal) (Result, error) {
	limit, offset, e := pagination(r)
	if e != nil {
		return Result{}, e
	}
	destination := "destination"
	if p.Kind == "user" && p.Role == "admin" {
		destination = `CASE WHEN destination='/orders' THEN '/payments' WHEN destination IN('/cash','/kitchen') THEN '/admin' ELSE destination END AS destination`
	}
	return resultJSON(h.repo.JSON(r.Context(), `WITH scoped AS(SELECT id,title,message,`+destination+`,created_at,read_at FROM legui_private.notifications WHERE `+notificationScopeSQL+`) SELECT jsonb_build_object('unread',(SELECT count(*) FROM scoped WHERE read_at IS NULL),'items',coalesce((SELECT jsonb_agg(to_jsonb(v) ORDER BY v.created_at DESC,v.id DESC) FROM(SELECT * FROM scoped ORDER BY created_at DESC,id DESC LIMIT ? OFFSET ?)v),'[]'::jsonb))::text payload`, p.Kind, p.ID, p.Platform(), p.Kind, p.BrandID, p.Kind, p.TenantID, limit, offset))
}
func (h *Handler) readNotifications(r *http.Request, p Principal) (Result, error) {
	input, e := decode[struct {
		ID string `json:"id"`
	}](r)
	if e != nil {
		return Result{}, e
	}
	if input.ID != "" && !validID(input.ID) {
		return Result{}, badInput("Notificación inválida")
	}
	e = h.repo.DB.WithContext(r.Context()).Exec(`UPDATE legui_private.notifications SET read_at=clock_timestamp() WHERE `+notificationScopeSQL+` AND read_at IS NULL AND (?='' OR id=nullif(?,'')::uuid)`, p.Kind, p.ID, p.Platform(), p.Kind, p.BrandID, p.Kind, p.TenantID, input.ID, input.ID).Error
	return Result{Data: json.RawMessage(`{"ok":true}`)}, e
}

const notificationScopeSQL = `principal_kind=? AND principal_id=? AND (? OR (?='user' AND brand_id=nullif(?,'')::uuid) OR (?='employee' AND tenant_id=nullif(?,'')::uuid))`

func (h *Handler) onboarding(r *http.Request, p Principal) (Result, error) {
	scope, e := h.businessScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	return resultJSON(h.repo.JSON(r.Context(), onboardingSQL, scope.BrandID))
}

const onboardingSQL = `WITH b AS(SELECT id,name FROM public.brands WHERE id=?), checks AS(SELECT b.id,
 jsonb_build_array(jsonb_build_object('key','brand','title','La identidad de tu hamburguesería','done',length(trim(b.name))>0,'to','/admin/catalog'),
 jsonb_build_object('key','location','title','Un local activo','done',EXISTS(SELECT 1 FROM public.tenants WHERE brand_id=b.id AND active),'to','/locations'),
 jsonb_build_object('key','menu','title','La carta disponible en tu local','done',EXISTS(SELECT 1 FROM public.products p JOIN public.tenant_products tp ON tp.product_id=p.id JOIN public.tenants t ON t.id=tp.tenant_id WHERE p.brand_id=b.id AND p.is_active AND tp.is_active AND tp.is_available AND t.active),'to','/admin/catalog'),
 jsonb_build_object('key','team','title','Tu equipo de trabajo','done',EXISTS(SELECT 1 FROM public.employees e JOIN public.tenants t ON t.id=e.tenant_id WHERE t.brand_id=b.id AND e.is_active),'to','/team/invitations'),
 jsonb_build_object('key','payments','title','Tu cuenta para cobrar online (opcional)','optional',true,'done',EXISTS(SELECT 1 FROM legui_private.mp_connections WHERE brand_id=b.id AND scope='business' AND state='connected' AND environment='production'),'to','/payments')) steps FROM b)
 SELECT jsonb_build_object('steps',steps,'completed_at',(SELECT completed_at FROM legui_private.onboarding_completions WHERE brand_id=checks.id))::text payload FROM checks`

func (h *Handler) completeOnboarding(r *http.Request, p Principal) (Result, error) {
	scope, e := h.businessScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	return h.mutate(r, p, scope, "onboarding.complete", struct{}{}, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		raw, e := jsonQuery(tx, onboardingSQL, scope.BrandID)
		if e != nil {
			return nil, 0, e
		}
		var data struct {
			Steps []struct {
				Done     bool `json:"done"`
				Optional bool `json:"optional"`
			} `json:"steps"`
		}
		if e = json.Unmarshal(raw, &data); e != nil {
			return nil, 0, e
		}
		for _, step := range data.Steps {
			if !step.Done && !step.Optional {
				return nil, 0, conflict("Completá la identidad, el local, la carta y el equipo antes de finalizar")
			}
		}
		result, e := jsonQuery(tx, `INSERT INTO legui_private.onboarding_completions(brand_id,completed_by) VALUES(?,?) ON CONFLICT(brand_id) DO UPDATE SET brand_id=excluded.brand_id RETURNING jsonb_build_object('completed_at',completed_at)::text payload`, scope.BrandID, p.ID)
		return result, 200, e
	})
}
func capabilityDigest(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
func (h *Handler) orderLink(r *http.Request, p Principal) (Result, error) {
	if !p.Cashier() {
		return Result{}, forbidden()
	}
	scope, e := h.tenantScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	id, e := resourceID(r)
	if e != nil {
		return Result{}, e
	}
	var bytes [32]byte
	if _, e = rand.Read(bytes[:]); e != nil {
		return Result{}, e
	}
	raw := base64.RawURLEncoding.EncodeToString(bytes[:])
	return h.mutate(r, p, scope, "orders.link", struct{}{}, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		if r.Method == "DELETE" {
			e := tx.Exec(`UPDATE legui_private.order_links SET revoked_at=clock_timestamp() WHERE order_id=? AND tenant_id=? AND revoked_at IS NULL`, id, scope.TenantID).Error
			return json.RawMessage(`{"ok":true}`), 200, e
		}
		var found string
		q := tx.Raw(`SELECT id::text FROM public.orders WHERE id=? AND tenant_id=? FOR SHARE`, id, scope.TenantID).Scan(&found)
		if q.Error != nil {
			return nil, 0, q.Error
		}
		if found == "" {
			return nil, 0, notFound()
		}
		e := tx.Exec(`INSERT INTO legui_private.order_links(digest,order_id,tenant_id) VALUES(?,?,?)`, capabilityDigest(raw), id, scope.TenantID).Error
		data, _ := json.Marshal(map[string]string{"path": "/order#token=" + raw})
		return data, 201, e
	})
}
func (h *Handler) publicOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	// The token is sent in a header, never in a query/access log. It grants GET only.
	raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if len(raw) != 43 {
		writeError(w, notFound())
		return
	}
	data, e := h.repo.JSON(r.Context(), `SELECT jsonb_build_object('status',s.code,'created_at',o.created_at,'currency',o.currency,'total_due',o.total_due,'total_paid',o.total_paid,'environment',o.payment_environment,'business',b.name,'items',coalesce((SELECT jsonb_agg(jsonb_build_object('name',d.product_name,'quantity',d.quantity)) FROM public.order_details d WHERE d.order_id=o.id),'[]'::jsonb))::text payload FROM legui_private.order_links l JOIN public.orders o ON o.id=l.order_id AND o.tenant_id=l.tenant_id JOIN public.tenants t ON t.id=o.tenant_id AND t.active JOIN public.brands b ON b.id=t.brand_id AND b.suspended_at IS NULL JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE l.digest=? AND l.expires_at>clock_timestamp() AND l.revoked_at IS NULL`, capabilityDigest(raw))
	if e != nil {
		writeError(w, e)
		return
	}
	_, _ = w.Write(data)
}
func (h *Handler) workerHealth(r *http.Request, p Principal) (Result, error) {
	if !p.Platform() {
		return Result{}, forbidden()
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('workers',coalesce((SELECT jsonb_agg(jsonb_build_object('instance_id',instance_id,'last_seen_at',last_seen_at,'last_success_at',last_success_at,'failed',failed,'stale',last_seen_at<clock_timestamp()-interval '2 minutes')) FROM legui_private.worker_heartbeats WHERE last_seen_at>clock_timestamp()-interval '1 day'),'[]'::jsonb),'mail_pending',(SELECT count(*) FROM legui_private.mail_outbox WHERE delivered_at IS NULL AND expires_at>clock_timestamp()),'mail_failed',(SELECT count(*) FROM legui_private.mail_outbox WHERE delivered_at IS NULL AND attempts>=8),'webhooks_pending',(SELECT count(*) FROM legui_private.mp_webhook_inbox WHERE state IN ('received','processing')))::text payload`))
}
