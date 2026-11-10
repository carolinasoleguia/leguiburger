package operations

import (
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"sort"
	"strings"
	"time"
)

func orderJSON(tx *gorm.DB, tenant, id string) (json.RawMessage, error) {
	return jsonQuery(tx, `SELECT (to_jsonb(o)||jsonb_build_object('status',s.code,'items',coalesce((SELECT jsonb_agg(to_jsonb(d)||jsonb_build_object('extras',coalesce((SELECT jsonb_agg(to_jsonb(e) ORDER BY e.id) FROM public.order_detail_extras e WHERE e.order_detail_id=d.id),'[]'::jsonb)) ORDER BY d.id) FROM public.order_details d WHERE d.order_id=o.id),'[]'::jsonb)))::text AS payload FROM public.orders o JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE o.tenant_id=? AND o.id=?`, tenant, id)
}
func (h *Handler) listOrders(r *http.Request, p Principal) (Result, error) {
	if !p.Cashier() {
		return Result{}, forbidden()
	}
	scope, err := h.tenantScope(r.Context(), p, r)
	if err != nil {
		return Result{}, err
	}
	limit, offset, err := pagination(r)
	if err != nil {
		return Result{}, err
	}
	env, err := environmentQuery(r)
	if err != nil {
		return Result{}, err
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb),'limit',?::int,'offset',?::int)::text AS payload FROM(SELECT o.*,s.code status FROM public.orders o JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE o.tenant_id=? AND o.payment_environment=? ORDER BY o.created_at DESC,o.id DESC LIMIT ? OFFSET ?)v`, limit, offset, scope.TenantID, env, limit, offset))
}
func (h *Handler) getOrder(r *http.Request, p Principal) (Result, error) {
	if !p.Cashier() {
		return Result{}, forbidden()
	}
	scope, err := h.tenantScope(r.Context(), p, r)
	if err != nil {
		return Result{}, err
	}
	id, err := resourceID(r)
	if err != nil {
		return Result{}, err
	}
	return resultJSON(orderJSON(h.repo.DB.WithContext(r.Context()), scope.TenantID, id))
}
func environmentQuery(r *http.Request) (string, error) {
	env := r.URL.Query().Get("environment")
	if env == "" {
		env = "production"
	}
	if !isOneOf(env, "production", "sandbox") {
		return "", badInput("environment debe ser production o sandbox")
	}
	return env, nil
}

func (h *Handler) createOrder(r *http.Request, p Principal) (Result, error) {
	if !p.Cashier() {
		return Result{}, forbidden()
	}
	scope, err := h.tenantScope(r.Context(), p, r)
	if err != nil {
		return Result{}, err
	}
	input, err := decode[CreateOrderRequest](r)
	if err != nil {
		return Result{}, err
	}
	if input.Channel == "" {
		input.Channel = "manual"
	}
	if input.Environment == "" {
		input.Environment = "production"
	}
	if !isOneOf(input.Channel, "manual", "counter", "web", "delivery", "external") || !isOneOf(input.Environment, "production", "sandbox") || !optionalID(input.CustomerID) || !optionalID(input.ShippingMethodID) || len(input.ShippingAddress) > 1000 || len(input.Items) < 1 || len(input.Items) > 50 {
		return Result{}, badInput("Pedido inválido; debe contener de 1 a 50 productos")
	}
	for _, item := range input.Items {
		if !validID(item.ProductID) || item.Quantity < 1 || item.Quantity > 500 || len(item.Extras) > 20 {
			return Result{}, badInput("Producto o cantidad inválidos")
		}
		seen := map[string]bool{}
		for _, extra := range item.Extras {
			if !validID(extra.ExtraID) || extra.Quantity < 1 || extra.Quantity > 20 || seen[extra.ExtraID] {
				return Result{}, badInput("Extra inválido o repetido")
			}
			seen[extra.ExtraID] = true
		}
	}
	return h.mutate(r, p, scope, "orders.create", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		id := uuid.NewString()
		shipping := "0"
		if input.ShippingMethodID != nil {
			var row struct{ Cost string }
			q := tx.Raw(`SELECT cost::text FROM public.shipping_methods WHERE id=? AND tenant_id=? AND is_active FOR SHARE`, *input.ShippingMethodID, scope.TenantID).Scan(&row)
			if q.Error != nil {
				return nil, 0, q.Error
			}
			if q.RowsAffected == 0 {
				return nil, 0, notFound()
			}
			shipping = row.Cost
		}
		if err := tx.Exec(`INSERT INTO public.orders(id,tenant_id,customer_id,shipping_method_id,shipping_address,channel,payment_environment,shipping_cost,idempotency_key) VALUES(?,?,?,?,?,?,?,?::numeric,?)`, id, scope.TenantID, input.CustomerID, input.ShippingMethodID, input.ShippingAddress, input.Channel, input.Environment, shipping, key).Error; err != nil {
			return nil, 0, err
		}
		for _, item := range input.Items {
			var product struct {
				Name  string
				Price string
			}
			q := tx.Raw(`SELECT p.name,coalesce(tp.price_override,p.base_price)::text price FROM public.tenant_products tp JOIN public.products p ON p.id=tp.product_id WHERE tp.tenant_id=? AND tp.product_id=? AND tp.is_active AND tp.is_available AND p.is_active AND p.brand_id=? FOR SHARE OF p,tp`, scope.TenantID, item.ProductID, scope.BrandID).Scan(&product)
			if q.Error != nil {
				return nil, 0, q.Error
			}
			if q.RowsAffected == 0 {
				return nil, 0, conflict("El producto no está disponible en este local")
			}
			detailID := uuid.NewString()
			if err := tx.Exec(`INSERT INTO public.order_details(id,tenant_id,order_id,product_id,product_name,quantity,list_price_unit,discount_unit,final_price_unit) VALUES(?,?,?,?,?,?,?::numeric,0,?::numeric)`, detailID, scope.TenantID, id, item.ProductID, product.Name, item.Quantity, product.Price, product.Price).Error; err != nil {
				return nil, 0, err
			}
			for _, extra := range item.Extras {
				var row struct {
					Name  string
					Price string
				}
				q := tx.Raw(`SELECT name,current_price::text price FROM public.extras WHERE id=? AND tenant_id=? AND is_active FOR SHARE`, extra.ExtraID, scope.TenantID).Scan(&row)
				if q.Error != nil {
					return nil, 0, q.Error
				}
				if q.RowsAffected == 0 {
					return nil, 0, conflict("El extra no está disponible en este local")
				}
				if err := tx.Exec(`INSERT INTO public.order_detail_extras(order_detail_id,extra_id,extra_name,extra_price_charged,quantity) VALUES(?,?,?,?::numeric,?)`, detailID, extra.ExtraID, row.Name, row.Price, extra.Quantity).Error; err != nil {
					return nil, 0, err
				}
			}
		}
		if err := tx.Exec(`UPDATE public.orders SET subtotal=(SELECT coalesce(sum(d.quantity*(d.final_price_unit+coalesce((SELECT sum(e.extra_price_charged*e.quantity) FROM public.order_detail_extras e WHERE e.order_detail_id=d.id),0))),0) FROM public.order_details d WHERE d.order_id=?) WHERE tenant_id=? AND id=?`, id, scope.TenantID, id).Error; err != nil {
			return nil, 0, err
		}
		data, err := orderJSON(tx, scope.TenantID, id)
		return data, 201, err
	})
}

var orderTransitions = map[string][]string{"pending": {"accepted", "rejected", "canceled"}, "accepted": {"preparing", "canceled"}, "preparing": {"ready", "canceled"}, "ready": {"dispatched", "delivered", "canceled"}, "dispatched": {"delivered", "canceled"}}

func validTransition(from, to string) bool {
	if from == to {
		return true
	}
	for _, allowed := range orderTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}
func (h *Handler) changeOrderStatus(r *http.Request, p Principal) (Result, error) {
	if !p.Cashier() {
		return Result{}, forbidden()
	}
	scope, err := h.tenantScope(r.Context(), p, r)
	if err != nil {
		return Result{}, err
	}
	id, err := resourceID(r)
	if err != nil {
		return Result{}, err
	}
	input, err := decode[ChangeOrderStatusRequest](r)
	if err != nil {
		return Result{}, err
	}
	if input.ExpectedVersion < 1 || !isOneOf(input.Status, "accepted", "preparing", "ready", "dispatched", "delivered", "canceled", "rejected") || len(input.Reason) > 500 {
		return Result{}, badInput("Estado o motivo inválidos")
	}
	if isOneOf(input.Status, "canceled", "rejected") && len(strings.TrimSpace(input.Reason)) < 3 {
		return Result{}, badInput("Indicá un motivo de al menos tres caracteres")
	}
	return h.mutate(r, p, scope, "orders.status", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		var order struct {
			Status      string
			Paid        string
			Version     int64
			Environment string
		}
		q := tx.Raw(`SELECT s.code status,o.total_paid::text paid,o.version,o.payment_environment AS environment FROM public.orders o JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE o.tenant_id=? AND o.id=? FOR UPDATE OF o`, scope.TenantID, id).Scan(&order)
		if q.Error != nil {
			return nil, 0, q.Error
		}
		if q.RowsAffected == 0 {
			return nil, 0, notFound()
		}
		if !validTransition(order.Status, input.Status) {
			return nil, 0, conflict("El cambio de estado no está permitido")
		}
		if order.Version != input.ExpectedVersion {
			return nil, 0, conflict("El pedido cambió; actualizá la versión antes de continuar")
		}
		if order.Status == input.Status {
			data, err := orderJSON(tx, scope.TenantID, id)
			return data, 200, err
		}
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0)),set_config('leguiburger.source_order_id',?,true)`, "inventory:"+scope.TenantID, id).Error; err != nil {
			return nil, 0, err
		}
		if input.Status == "accepted" {
			if err := postOrderSale(tx, scope, id, order.Environment); err != nil {
				return nil, 0, err
			}
		}
		if input.Status == "canceled" || input.Status == "rejected" {
			var hasPaid bool
			if err := tx.Raw(`SELECT total_paid>0 FROM public.orders WHERE id=?`, id).Scan(&hasPaid).Error; err != nil {
				return nil, 0, err
			}
			if hasPaid {
				return nil, 0, conflict("Registrá el reintegro completo antes de cancelar un pedido cobrado")
			}
			if err := restoreOrderStock(tx, scope, id); err != nil {
				return nil, 0, err
			}
			if err := tx.Exec(`UPDATE public.sales SET voided_at=clock_timestamp(),void_reason=? WHERE tenant_id=? AND order_id=? AND voided_at IS NULL`, input.Reason, scope.TenantID, id).Error; err != nil {
				return nil, 0, err
			}
		}
		if err := tx.Exec(`UPDATE public.orders SET order_status_id=(SELECT id FROM public.order_statuses_lookup WHERE code=?) WHERE tenant_id=? AND id=?`, input.Status, scope.TenantID, id).Error; err != nil {
			return nil, 0, err
		}
		data, err := orderJSON(tx, scope.TenantID, id)
		return data, 200, err
	})
}

type stockUse struct {
	Kind     string
	ID       string
	Quantity string
}

func postOrderSale(tx *gorm.DB, scope Scope, order, environment string) error {
	// Lock recipe inputs and costs in a consistent order before computing usage and snapshots.
	for _, query := range []string{
		`SELECT p.id::text FROM public.products p WHERE p.id IN(SELECT product_id FROM public.order_details WHERE order_id=?) ORDER BY p.id FOR UPDATE`,
		`SELECT tp.product_id::text id FROM public.tenant_products tp WHERE (tp.tenant_id,tp.product_id) IN(SELECT tenant_id,product_id FROM public.order_details WHERE order_id=?) ORDER BY tp.product_id FOR UPDATE`,
		`SELECT s.id::text FROM public.supplies s JOIN public.recipes r ON r.supply_id=s.id WHERE r.product_id IN(SELECT product_id FROM public.order_details WHERE order_id=?) ORDER BY s.id,r.product_id FOR SHARE OF s,r`,
		`SELECT x.id::text FROM public.extras x WHERE x.id IN(SELECT e.extra_id FROM public.order_detail_extras e JOIN public.order_details d ON d.id=e.order_detail_id WHERE d.order_id=?) ORDER BY x.id FOR UPDATE`,
	} {
		var locked []struct{ ID string }
		if err := tx.Raw(query, order).Scan(&locked).Error; err != nil {
			return err
		}
	}
	uses := []stockUse{}
	query := `SELECT 'product' kind,d.product_id::text id,sum(d.quantity)::text quantity FROM public.order_details d JOIN public.tenant_products tp ON tp.tenant_id=d.tenant_id AND tp.product_id=d.product_id WHERE d.order_id=? AND tp.track_stock GROUP BY d.product_id
 UNION ALL SELECT 'extra',e.extra_id::text,sum(e.quantity*d.quantity)::text FROM public.order_detail_extras e JOIN public.order_details d ON d.id=e.order_detail_id JOIN public.extras x ON x.id=e.extra_id AND x.tenant_id=d.tenant_id WHERE d.order_id=? AND x.track_stock GROUP BY e.extra_id
 UNION ALL SELECT 'supply',s.id::text,sum(rec.quantity_used*d.quantity)::text FROM public.order_details d JOIN public.tenant_products tp ON tp.tenant_id=d.tenant_id AND tp.product_id=d.product_id JOIN public.recipes rec ON rec.product_id=d.product_id JOIN public.supplies s ON s.id=rec.supply_id AND s.tenant_id=d.tenant_id WHERE d.order_id=? AND NOT tp.track_stock GROUP BY s.id`
	if environment == "production" {
		if err := tx.Raw(query, order, order, order).Scan(&uses).Error; err != nil {
			return err
		}
	}
	sort.Slice(uses, func(i, j int) bool { return uses[i].Kind+uses[i].ID < uses[j].Kind+uses[j].ID })
	if err := tx.Exec(`SELECT set_config('leguiburger.inventory_reason','sale',true)`).Error; err != nil {
		return err
	}
	for _, use := range uses {
		if err := changeStock(tx, scope.TenantID, use.Kind, use.ID, "-"+use.Quantity, nil); err != nil {
			return err
		}
	}
	// Costs are frozen at acceptance; NULL means unknown, never a made-up zero margin.
	if err := tx.Exec(`UPDATE public.order_details d SET frozen_cost_unit=(SELECT sum(r.quantity_used*s.current_wholesale_cost) FROM public.recipes r JOIN public.supplies s ON s.id=r.supply_id AND s.tenant_id=d.tenant_id WHERE r.product_id=d.product_id) WHERE d.order_id=?`, order).Error; err != nil {
		return err
	}
	if err := tx.Exec(`UPDATE public.order_detail_extras e SET frozen_cost_unit=x.current_unit_cost FROM public.extras x WHERE x.id=e.extra_id AND e.order_detail_id IN(SELECT id FROM public.order_details WHERE order_id=?)`, order).Error; err != nil {
		return err
	}
	return tx.Exec(`INSERT INTO public.sales(tenant_id,order_id,order_detail_id,customer_id,shipping_method_id,payment_status_id,product_id,product_name,quantity,frozen_cost_unit,total_list_price,total_discount_applied,total_extras_amount,total_shipping_amount,total_net_revenue,currency,payment_environment)
 SELECT d.tenant_id,d.order_id,d.id,o.customer_id,o.shipping_method_id,o.payment_status_id,d.product_id,d.product_name,d.quantity,
 CASE WHEN d.frozen_cost_unit IS NULL OR EXISTS(SELECT 1 FROM public.order_detail_extras e WHERE e.order_detail_id=d.id AND e.frozen_cost_unit IS NULL) THEN NULL ELSE d.frozen_cost_unit+coalesce((SELECT sum(e.frozen_cost_unit*e.quantity) FROM public.order_detail_extras e WHERE e.order_detail_id=d.id),0) END,
 d.list_price_unit*d.quantity,d.discount_unit*d.quantity,coalesce(extras.amount,0)*d.quantity,
 CASE WHEN d.id=(SELECT min(first_line.id::text)::uuid FROM public.order_details first_line WHERE first_line.order_id=o.id) THEN o.shipping_cost ELSE 0 END,
 d.final_price_unit*d.quantity+coalesce(extras.amount,0)*d.quantity+CASE WHEN d.id=(SELECT min(first_line.id::text)::uuid FROM public.order_details first_line WHERE first_line.order_id=o.id) THEN o.shipping_cost ELSE 0 END,
 o.currency,o.payment_environment FROM public.order_details d JOIN public.orders o ON o.id=d.order_id LEFT JOIN LATERAL(SELECT sum(e.extra_price_charged*e.quantity) amount FROM public.order_detail_extras e WHERE e.order_detail_id=d.id)extras ON true WHERE o.id=?`, order).Error
}
func restoreOrderStock(tx *gorm.DB, scope Scope, order string) error {
	var uses []stockUse
	if err := tx.Raw(`SELECT item_kind kind,item_id::text id,(-sum(quantity_delta))::text quantity FROM public.inventory_movements WHERE tenant_id=? AND order_id=? AND reason='sale' GROUP BY item_kind,item_id ORDER BY item_kind,item_id`, scope.TenantID, order).Scan(&uses).Error; err != nil {
		return err
	}
	if err := tx.Exec(`SELECT set_config('leguiburger.inventory_reason','return',true)`).Error; err != nil {
		return err
	}
	for _, use := range uses {
		if err := changeStock(tx, scope.TenantID, use.Kind, use.ID, use.Quantity, nil); err != nil {
			return err
		}
	}
	return nil
}
func (h *Handler) createOrderPayment(r *http.Request, p Principal) (Result, error) {
	if !p.Cashier() {
		return Result{}, forbidden()
	}
	scope, err := h.tenantScope(r.Context(), p, r)
	if err != nil {
		return Result{}, err
	}
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
	return h.mutate(r, p, scope, "orders.payment", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		var paidAt any
		if input.SettledAt != nil {
			paidAt = *input.SettledAt
		}
		data, err := jsonQuery(tx, `INSERT INTO public.order_payments(tenant_id,order_id,currency,environment,kind,original_payment_id,amount,status,method,settled_at,idempotency_key,request_digest)
   SELECT tenant_id,id,currency,payment_environment,?,?,?::numeric,'succeeded',?,coalesce(?::timestamptz,clock_timestamp()),?,? FROM public.orders WHERE tenant_id=? AND id=? AND order_status_id NOT IN(SELECT id FROM public.order_statuses_lookup WHERE code IN ('canceled','rejected')) RETURNING to_jsonb(order_payments)::text payload`, input.Kind, input.OriginalPaymentID, string(input.Amount), input.Method, paidAt, key, digest, scope.TenantID, id)
		return data, 201, err
	})
}
func validateManualPayment(input PaymentRequest) error {
	if !validMoney(input.Amount, true) || !isOneOf(input.Kind, "charge", "refund") || !isOneOf(input.Method, "cash", "transfer", "card", "other") || !optionalID(input.OriginalPaymentID) || (input.Kind == "refund") != (input.OriginalPaymentID != nil) {
		return badInput("Pago o reintegro manual inválidos")
	}
	if input.SettledAt != nil && (input.SettledAt.After(time.Now().Add(5*time.Minute)) || input.SettledAt.Before(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))) {
		return badInput("Fecha de cobro inválida")
	}
	return nil
}
func (h *Handler) listOrderPayments(r *http.Request, p Principal) (Result, error) {
	if !p.Cashier() {
		return Result{}, forbidden()
	}
	scope, err := h.tenantScope(r.Context(), p, r)
	if err != nil {
		return Result{}, err
	}
	id, err := resourceID(r)
	if err != nil {
		return Result{}, err
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.created_at,x.id),'[]'::jsonb))::text payload FROM public.order_payments x WHERE tenant_id=? AND order_id=?`, scope.TenantID, id))
}
