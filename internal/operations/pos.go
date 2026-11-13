package operations

import "net/http"

func (h *Handler) posCatalog(r *http.Request, p Principal) (Result, error) {
	if !p.Cashier() {
		return Result{}, forbidden()
	}
	scope, e := h.tenantScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	limit, offset, e := pagination(r)
	if e != nil {
		return Result{}, e
	}
	search, e := searchQuery(r)
	if e != nil {
		return Result{}, e
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb))::text payload FROM(SELECT p.id,p.name,coalesce(tp.price_override,p.base_price)::text price,tp.current_stock,tp.track_stock,tp.options_configured,legui_private.product_options(tp.tenant_id,p.id) option_groups,coalesce((SELECT jsonb_agg(jsonb_build_object('id',x.id,'name',x.name,'price',x.current_price::text,'available',NOT x.track_stock OR x.current_stock>0) ORDER BY x.name,x.id) FROM public.extras x WHERE x.tenant_id=tp.tenant_id AND x.is_active),'[]'::jsonb) extras FROM public.products p JOIN public.tenant_products tp ON tp.product_id=p.id WHERE tp.tenant_id=? AND p.is_active AND tp.is_active AND tp.is_available AND coalesce(legui_private.product_capacity(tp.tenant_id,p.id)>0,true) AND legui_private.product_groups_ready(tp.tenant_id,p.id) AND (?='' OR strpos(lower(p.name||' '||coalesce(p.description,'')),lower(?))>0) ORDER BY p.name,p.id LIMIT ? OFFSET ?)v`, scope.TenantID, search, search, limit, offset))
}
func (h *Handler) customerDirectory(r *http.Request, p Principal) (Result, error) {
	if !p.Cashier() {
		return Result{}, forbidden()
	}
	scope, e := h.tenantScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	limit, offset, e := pagination(r)
	if e != nil {
		return Result{}, e
	}
	search := r.URL.Query().Get("search")
	if len(search) > 100 {
		return Result{}, badInput("Búsqueda demasiado larga")
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb))::text payload FROM(
 SELECT c.id,c.first_name,c.last_name,c.email,c.phone,c.created_at,(SELECT count(*) FROM public.orders o WHERE o.customer_id=c.id AND o.tenant_id=c.tenant_id AND o.payment_environment='production') orders,(SELECT max(o.created_at) FROM public.orders o WHERE o.customer_id=c.id AND o.tenant_id=c.tenant_id AND o.payment_environment='production') last_order_at
 FROM public.customers c WHERE c.tenant_id=? AND (?='' OR strpos(lower(coalesce(c.first_name,'')||' '||coalesce(c.last_name,'')||' '||coalesce(c.email,'')||' '||coalesce(c.phone,'')),lower(?))>0) ORDER BY c.created_at DESC,c.id DESC LIMIT ? OFFSET ?)v`, scope.TenantID, search, search, limit, offset))
}
