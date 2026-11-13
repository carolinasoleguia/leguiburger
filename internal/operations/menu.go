package operations

import (
	"context"
	"encoding/json"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"
)

func searchQuery(r *http.Request) (string, error) {
	value := strings.TrimSpace(r.URL.Query().Get("search"))
	if len([]rune(value)) > 100 {
		return "", badInput("La búsqueda admite hasta 100 caracteres")
	}
	return value, nil
}
func (h *Handler) publicMenu(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	id := r.PathValue("id")
	if !validID(id) {
		writeError(w, notFound())
		return
	}
	limit, offset, e := pagination(r)
	if e != nil {
		writeError(w, e)
		return
	}
	search, e := searchQuery(r)
	if e != nil {
		writeError(w, e)
		return
	}
	productFilter := r.URL.Query().Get("product_id")
	if productFilter != "" && !validID(productFilter) {
		writeError(w, badInput("Producto inválido"))
		return
	}
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	if len([]rune(category)) > 60 {
		writeError(w, badInput("Categoría inválida"))
		return
	}
	data, e := h.repo.JSON(ctx, `WITH location AS(SELECT t.id,t.brand_id,coalesce(nullif(t.display_name,''),t.subdomain) name,t.address,t.contact_phone,t.service_hours,t.ordering_enabled,b.name brand_name
 FROM public.tenants t JOIN public.brands b ON b.id=t.brand_id WHERE t.id=? AND t.active AND t.menu_published AND b.suspended_at IS NULL AND b.closed_at IS NULL),
 menu AS(SELECT p.id,p.name,p.description,p.image_url,p.category,coalesce(tp.price_override,p.base_price)::text price,
 (tp.is_available AND coalesce(legui_private.product_capacity(tp.tenant_id,p.id)>0,true) AND legui_private.product_groups_ready(tp.tenant_id,p.id)) available,tp.options_configured,legui_private.product_options(tp.tenant_id,p.id) option_groups
 FROM location l JOIN public.tenant_products tp ON tp.tenant_id=l.id JOIN public.products p ON p.id=tp.product_id AND p.brand_id=l.brand_id WHERE tp.is_active AND p.is_active),
 filtered AS(SELECT * FROM menu WHERE (?='' OR strpos(lower(name||' '||coalesce(description,'')),lower(?))>0) AND (?='' OR category=?) AND (?='' OR id=nullif(?,'')::uuid))
 SELECT jsonb_build_object('location',jsonb_build_object('id',l.id,'name',l.name,'brand_name',l.brand_name,'address',l.address,'phone',l.contact_phone,'hours',l.service_hours),
 'currency','ARS','ordering_enabled',l.ordering_enabled,'ordering_ready',legui_private.tenant_receiving(l.id),'capacity_full',legui_private.kitchen_full(l.id),'prep_min',coalesce((SELECT prep_min FROM legui_private.tenant_ordering WHERE tenant_id=l.id),20)+legui_private.queue_delay(l.id),'prep_max',coalesce((SELECT prep_max FROM legui_private.tenant_ordering WHERE tenant_id=l.id),35)+legui_private.queue_delay(l.id),'extras',coalesce((SELECT jsonb_agg(jsonb_build_object('id',x.id,'name',x.name,'price',x.current_price::text,'available',NOT x.track_stock OR x.current_stock>0) ORDER BY x.name,x.id) FROM public.extras x WHERE x.tenant_id=l.id AND x.is_active),'[]'::jsonb),'delivery_zones_enabled',coalesce((SELECT zones_enabled FROM legui_private.delivery_settings WHERE tenant_id=l.id),false),'delivery_zones',coalesce((SELECT jsonb_agg(jsonb_build_object('id',z.id,'name',m.name,'description',z.description,'price',m.cost::text,'minimum',z.minimum::text,'postal_codes',z.postal_codes) ORDER BY m.name,z.id) FROM legui_private.delivery_zones z JOIN public.shipping_methods m ON m.id=z.shipping_method_id AND m.tenant_id=z.tenant_id AND m.is_active WHERE z.tenant_id=l.id),'[]'::jsonb),'shipping_methods',coalesce((SELECT jsonb_agg(jsonb_build_object('id',m.id,'name',m.name,'price',m.cost::text) ORDER BY m.name,m.id) FROM public.shipping_methods m WHERE m.tenant_id=l.id AND m.is_active),'[]'::jsonb),'categories',coalesce((SELECT jsonb_agg(category ORDER BY category) FROM(SELECT DISTINCT category FROM menu)c),'[]'::jsonb),
 'total',(SELECT count(*) FROM filtered),'limit',?::int,'offset',?::int,
 'items',coalesce((SELECT jsonb_agg(to_jsonb(v) ORDER BY v.category,v.name,v.id) FROM(SELECT * FROM filtered ORDER BY category,name,id LIMIT ? OFFSET ?)v),'[]'::jsonb))::text payload FROM location l`, id, search, search, category, category, productFilter, productFilter, limit, offset, limit, offset)
	if e != nil {
		writeError(w, e)
		return
	}
	var out map[string]any
	json.Unmarshal(data, &out)
	online := false
	if h.buyer.OnlinePaymentAvailable != nil {
		online, _ = h.buyer.OnlinePaymentAvailable(ctx, id)
	}
	out["payment_options"] = map[string]bool{"at_local": true, "mercadopago": online}
	json.NewEncoder(w).Encode(out)
}
func (h *Handler) menuSettings(r *http.Request, p Principal) (Result, error) {
	if !p.Manager() {
		return Result{}, forbidden()
	}
	scope, e := h.tenantScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	if r.Method == http.MethodGet {
		return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('published',menu_published,'ordering_enabled',ordering_enabled,'path','/'||(SELECT slug FROM public.brands WHERE id=public.tenants.brand_id)||'/'||slug||'/carta')::text payload FROM public.tenants WHERE id=?`, scope.TenantID))
	}
	input, e := decode[struct {
		Published       *bool `json:"published"`
		OrderingEnabled *bool `json:"ordering_enabled"`
	}](r)
	if e != nil {
		return Result{}, e
	}
	if input.Published == nil && input.OrderingEnabled == nil {
		return Result{}, badInput("Indicá si querés publicar la carta")
	}
	return h.mutate(r, p, scope, "menu.publication", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		data, e := jsonQuery(tx, `UPDATE public.tenants SET menu_published=coalesce(?,menu_published),ordering_enabled=coalesce(?,ordering_enabled) WHERE id=? AND brand_id=? RETURNING jsonb_build_object('published',menu_published,'ordering_enabled',ordering_enabled,'path','/'||(SELECT slug FROM public.brands WHERE id=public.tenants.brand_id)||'/'||slug||'/carta')::text payload`, input.Published, input.OrderingEnabled, scope.TenantID, scope.BrandID)
		return data, 200, e
	})
}

func (h *Handler) publicRestaurants(r *http.Request) (Result, error) {
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb))::text payload FROM(SELECT t.id,b.name business,coalesce(nullif(t.display_name,''),t.subdomain) location,t.address,'/'||b.slug||'/'||t.slug||'/carta' path FROM public.tenants t JOIN public.brands b ON b.id=t.brand_id WHERE t.active AND t.menu_published AND b.suspended_at IS NULL AND b.closed_at IS NULL ORDER BY b.name,t.id LIMIT 100)v`))
}

// Resolve only published, active locations; a forged brand/local pair cannot
// select a different restaurant or disclose unpublished business information.
func (h *Handler) resolvePublicMenu(r *http.Request) (Result, error) {
	brand, local := r.PathValue("brand"), r.PathValue("local")
	if !publicSlug(brand) || !publicSlug(local) {
		return Result{}, notFound()
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('tenant_id',t.id,'path','/'||b.slug||'/'||t.slug||'/carta')::text payload FROM public.tenants t JOIN public.brands b ON b.id=t.brand_id WHERE b.slug=? AND t.slug=? AND t.active AND t.menu_published AND b.suspended_at IS NULL AND b.closed_at IS NULL`, brand, local))
}
func publicSlug(value string) bool {
	if len(value) == 0 || len(value) > 80 || value[0] == '-' || value[len(value)-1] == '-' {
		return false
	}
	previous := byte(0)
	for i := 0; i < len(value); i++ {
		c := value[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' && previous != '-') {
			return false
		}
		previous = c
	}
	return true
}
