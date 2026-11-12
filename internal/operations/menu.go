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
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	if len([]rune(category)) > 60 {
		writeError(w, badInput("Categoría inválida"))
		return
	}
	data, e := h.repo.JSON(ctx, `WITH location AS(SELECT t.id,t.brand_id,coalesce(nullif(t.display_name,''),t.subdomain) name,t.address,t.contact_phone,t.service_hours,b.name brand_name
 FROM public.tenants t JOIN public.brands b ON b.id=t.brand_id WHERE t.id=? AND t.active AND t.menu_published AND b.suspended_at IS NULL AND b.closed_at IS NULL),
 menu AS(SELECT p.id,p.name,p.description,p.image_url,p.category,coalesce(tp.price_override,p.base_price)::text price,
 (tp.is_available AND (NOT tp.track_stock OR tp.current_stock>0)) available
 FROM location l JOIN public.tenant_products tp ON tp.tenant_id=l.id JOIN public.products p ON p.id=tp.product_id AND p.brand_id=l.brand_id WHERE tp.is_active AND p.is_active),
 filtered AS(SELECT * FROM menu WHERE (?='' OR strpos(lower(name||' '||coalesce(description,'')),lower(?))>0) AND (?='' OR category=?))
 SELECT jsonb_build_object('location',jsonb_build_object('id',l.id,'name',l.name,'brand_name',l.brand_name,'address',l.address,'phone',l.contact_phone,'hours',l.service_hours),
 'currency','ARS','categories',coalesce((SELECT jsonb_agg(category ORDER BY category) FROM(SELECT DISTINCT category FROM menu)c),'[]'::jsonb),
 'total',(SELECT count(*) FROM filtered),'limit',?::int,'offset',?::int,
 'items',coalesce((SELECT jsonb_agg(to_jsonb(v) ORDER BY v.category,v.name,v.id) FROM(SELECT * FROM filtered ORDER BY category,name,id LIMIT ? OFFSET ?)v),'[]'::jsonb))::text payload FROM location l`, id, search, search, category, category, limit, offset, limit, offset)
	if e != nil {
		writeError(w, e)
		return
	}
	w.Write(data)
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
		return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('published',menu_published,'path','/catalogo/'||id::text)::text payload FROM public.tenants WHERE id=?`, scope.TenantID))
	}
	input, e := decode[struct {
		Published *bool `json:"published"`
	}](r)
	if e != nil {
		return Result{}, e
	}
	if input.Published == nil {
		return Result{}, badInput("Indicá si querés publicar la carta")
	}
	return h.mutate(r, p, scope, "menu.publication", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		data, e := jsonQuery(tx, `UPDATE public.tenants SET menu_published=? WHERE id=? AND brand_id=? RETURNING jsonb_build_object('published',menu_published,'path','/catalogo/'||id::text)::text payload`, *input.Published, scope.TenantID, scope.BrandID)
		return data, 200, e
	})
}
