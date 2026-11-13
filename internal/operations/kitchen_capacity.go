package operations

import (
	"encoding/json"
	"gorm.io/gorm"
	"net/http"
)

func reserveKitchenSlot(tx *gorm.DB, tenant, environment string) error {
	if environment != "production" {
		return nil
	}
	// Separate statements ensure the count sees commits made while waiting for the lock.
	if e := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "kitchen-capacity:"+tenant).Error; e != nil {
		return e
	}
	var full bool
	if e := tx.Raw(`SELECT legui_private.kitchen_full(?)`, tenant).Scan(&full).Error; e != nil {
		return e
	}
	if full {
		return conflict("La cocina está al máximo de su capacidad. Esperá unos minutos para hacer un nuevo pedido")
	}
	return nil
}
func kitchenLoadData(tx *gorm.DB, tenant string) (json.RawMessage, error) {
	return jsonQuery(tx, `SELECT jsonb_build_object('active_count',legui_private.kitchen_load(t.id),'max_active_orders',coalesce(c.max_active_orders,0),'full',legui_private.kitchen_full(t.id),'queue_minutes',legui_private.queue_delay(t.id),'prep_min',coalesce(c.prep_min,20)+legui_private.queue_delay(t.id),'prep_max',coalesce(c.prep_max,35)+legui_private.queue_delay(t.id))::text payload FROM public.tenants t LEFT JOIN legui_private.tenant_ordering c ON c.tenant_id=t.id WHERE t.id=?`, tenant)
}
func (h *Handler) kitchenLoad(r *http.Request, p Principal) (Result, error) {
	scope, e := h.tenantScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	return resultJSON(kitchenLoadData(h.repo.DB.WithContext(r.Context()), scope.TenantID))
}
