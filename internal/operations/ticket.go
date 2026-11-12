package operations

import (
	"net/http"
)

func (h *Handler) orderTicket(r *http.Request, p Principal) (Result, error) {
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "receipt"
	}
	if !isOneOf(kind, "receipt", "kitchen") {
		return Result{}, badInput("Tipo de comprobante inválido")
	}
	if kind == "receipt" && !p.Cashier() {
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
	// Kitchen projection deliberately excludes buyer contact, address and amounts.
	base := `jsonb_build_object('id',o.id,'kind',?::text,'business',b.name,'location',coalesce(nullif(t.display_name,''),t.subdomain),'status',s.code,'created_at',o.created_at,'fulfillment',o.fulfillment,'items',coalesce((SELECT jsonb_agg(jsonb_build_object('name',d.product_name,'quantity',d.quantity,'note',d.preparation_note,'extras',coalesce((SELECT jsonb_agg(jsonb_build_object('name',x.extra_name,'quantity',x.quantity) ORDER BY x.id) FROM public.order_detail_extras x WHERE x.order_detail_id=d.id),'[]'::jsonb)) ORDER BY d.id) FROM public.order_details d WHERE d.order_id=o.id),'[]'::jsonb))`
	if kind == "receipt" {
		base += `||jsonb_build_object('currency',o.currency,'total_due',o.total_due::text,'total_paid',o.total_paid::text,'shipping',o.shipping_cost::text,'discount',o.total_discounts::text,'name',o.buyer_name,'phone',o.buyer_phone,'address',o.shipping_address,'zone',o.delivery_zone_name,'postal_code',o.delivery_postal_code,'payment_method',o.buyer_payment_method)`
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT (`+base+`)::text payload FROM public.orders o JOIN public.order_statuses_lookup s ON s.id=o.order_status_id JOIN public.tenants t ON t.id=o.tenant_id JOIN public.brands b ON b.id=t.brand_id WHERE o.id=? AND o.tenant_id=?`, kind, id, scope.TenantID))
}
