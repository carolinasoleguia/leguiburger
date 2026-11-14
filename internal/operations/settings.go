package operations

import (
	"encoding/json"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

func (h *Handler) updateInventorySettings(r *http.Request, p Principal) (Result, error) {
	if !p.Manager() {
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
	kind := r.PathValue("kind")
	table, column, err := inventoryTable(kind)
	if err != nil {
		return Result{}, err
	}
	input, err := decode[InventorySettingsRequest](r)
	if err != nil {
		return Result{}, err
	}
	if input.MinimumStock == nil && input.UnitCost == nil {
		return Result{}, badInput("Indicá mínimo de stock o costo unitario")
	}
	if input.MinimumStock != nil && (!decimalPattern.MatchString(string(*input.MinimumStock)) || strings.HasPrefix(string(*input.MinimumStock), "-") || (kind != "supply" && !integerDecimal(*input.MinimumStock))) {
		return Result{}, badInput("Mínimo de stock inválido")
	}
	if input.UnitCost != nil && (kind == "product" || !validMoney(*input.UnitCost, false)) {
		return Result{}, badInput("El costo debe ser válido; el costo del producto se calcula desde su receta")
	}
	return h.mutate(r, p, scope, "inventory.settings", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		var minimum, cost any
		if input.MinimumStock != nil {
			minimum = string(*input.MinimumStock)
		}
		if input.UnitCost != nil {
			cost = string(*input.UnitCost)
		}
		assignment := `minimum_stock=coalesce(?::numeric,minimum_stock)`
		args := []any{minimum}
		if kind == "supply" {
			assignment += `,current_wholesale_cost=coalesce(?::numeric,current_wholesale_cost)`
			args = append(args, cost)
		}
		if kind == "extra" {
			assignment += `,current_unit_cost=coalesce(?::numeric,current_unit_cost)`
			args = append(args, cost)
		}
		args = append(args, scope.TenantID, id)
		data, err := jsonQuery(tx, `UPDATE `+table+` SET `+assignment+` WHERE tenant_id=? AND `+column+`=? RETURNING to_jsonb(`+strings.TrimPrefix(table, "public.")+`)::text payload`, args...)
		return data, 200, err
	})
}
func (h *Handler) voidExpense(r *http.Request, p Principal) (Result, error) {
	if !p.Manager() {
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
	input, err := decode[VoidRequest](r)
	if err != nil {
		return Result{}, err
	}
	if len(strings.TrimSpace(input.Reason)) < 3 || len(input.Reason) > 500 {
		return Result{}, badInput("Indicá un motivo de 3 a 500 caracteres")
	}
	return h.mutate(r, p, scope, "expenses.void", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		data, err := jsonQuery(tx, `UPDATE public.operating_expenses SET voided_at=clock_timestamp(),void_reason=? WHERE tenant_id=? AND id=? AND voided_at IS NULL RETURNING to_jsonb(operating_expenses)::text payload`, input.Reason, scope.TenantID, id)
		return data, 200, err
	})
}
