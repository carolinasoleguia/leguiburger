package operations

import (
	"encoding/json"
	"gorm.io/gorm"
	"net/http"
)

func validateRecipeBasket(tx *gorm.DB, tenant string, items []OrderItemRequest) error {
	raw, _ := json.Marshal(items)
	var locked []struct{ ID string }
	if e := tx.Raw(`SELECT s.id::text FROM public.supplies s WHERE s.tenant_id=? AND s.id IN(SELECT r.supply_id FROM public.recipes r JOIN public.tenant_products tp ON tp.product_id=r.product_id AND tp.tenant_id=s.tenant_id AND NOT tp.track_stock WHERE r.product_id IN(SELECT product_id FROM jsonb_to_recordset(?::jsonb) AS x(product_id uuid))) ORDER BY s.id FOR SHARE`, tenant, string(raw)).Scan(&locked).Error; e != nil {
		return e
	}
	var missing bool
	e := tx.Raw(`WITH requested AS(SELECT product_id,sum(quantity) n FROM jsonb_to_recordset(?::jsonb) AS x(product_id uuid,quantity integer) GROUP BY product_id),needed AS(SELECT s.id,s.current_stock,s.is_active,sum(r.quantity_used*x.n) demand FROM requested x JOIN public.tenant_products tp ON tp.product_id=x.product_id AND tp.tenant_id=? AND NOT tp.track_stock JOIN public.recipes r ON r.product_id=x.product_id JOIN public.supplies s ON s.id=r.supply_id AND s.tenant_id=tp.tenant_id GROUP BY s.id) SELECT EXISTS(SELECT 1 FROM needed WHERE NOT is_active OR demand>current_stock)`, string(raw), tenant).Scan(&missing).Error
	if e != nil {
		return e
	}
	if missing {
		return conflict("No quedan ingredientes suficientes para esta combinación. Ajustá las cantidades o elegí otro producto")
	}
	return nil
}
func (h *Handler) menuAvailability(r *http.Request, p Principal) (Result, error) {
	if !p.Manager() && !p.Cashier() {
		return Result{}, forbidden()
	}
	scope, e := h.tenantScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	if r.Method == "GET" {
		search, e := searchQuery(r)
		if e != nil {
			return Result{}, e
		}
		limit, offset, e := pagination(r)
		if e != nil {
			return Result{}, e
		}
		return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v) ORDER BY v.name,v.id),'[]'::jsonb))::text payload FROM(SELECT p.id,p.name,tp.is_available enabled,(coalesce(legui_private.product_capacity(tp.tenant_id,p.id)>0,true) AND legui_private.product_groups_ready(tp.tenant_id,p.id)) in_stock,(tp.is_available AND (coalesce(legui_private.product_capacity(tp.tenant_id,p.id)>0,true) AND legui_private.product_groups_ready(tp.tenant_id,p.id))) available FROM public.tenant_products tp JOIN public.products p ON p.id=tp.product_id AND p.brand_id=? AND p.is_active WHERE tp.tenant_id=? AND tp.is_active AND (?='' OR strpos(lower(p.name),lower(?))>0) ORDER BY p.name,p.id LIMIT ? OFFSET ?)v`, scope.BrandID, scope.TenantID, search, search, limit, offset))
	}
	id, e := resourceID(r)
	if e != nil {
		return Result{}, e
	}
	in, e := decode[struct {
		Enabled *bool `json:"enabled"`
	}](r)
	if e != nil {
		return Result{}, e
	}
	if in.Enabled == nil {
		return Result{}, badInput("Indicá la disponibilidad")
	}
	return h.mutate(r, p, scope, "menu.availability", in, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		data, e := jsonQuery(tx, `UPDATE public.tenant_products tp SET is_available=? FROM public.products p WHERE p.id=tp.product_id AND p.brand_id=? AND tp.tenant_id=? AND p.id=? RETURNING jsonb_build_object('id',tp.product_id,'enabled',tp.is_available)::text payload`, *in.Enabled, scope.BrandID, scope.TenantID, id)
		return data, 200, e
	})
}
