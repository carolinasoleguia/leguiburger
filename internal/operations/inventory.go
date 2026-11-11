package operations

import (
	"encoding/json"
	"gorm.io/gorm"
	"math/big"
	"net/http"
	"strconv"
	"strings"
)

func inventoryTable(kind string) (string, string, error) {
	switch kind {
	case "product":
		return "public.tenant_products", "product_id", nil
	case "extra":
		return "public.extras", "id", nil
	case "supply":
		return "public.supplies", "id", nil
	}
	return "", "", badInput("Tipo de inventario inválido")
}
func integerDecimal(value Decimal) bool {
	n, ok := new(big.Rat).SetString(string(value))
	return ok && n.IsInt()
}
func changeStock(tx *gorm.DB, tenant, kind, id, delta string, expected *int64) error {
	table, column, err := inventoryTable(kind)
	if err != nil {
		return err
	}
	var item struct {
		Version int64
		Enough  bool
	}
	q := tx.Raw(`SELECT stock_version AS version,current_stock+?::numeric>=0 AS enough FROM `+table+` WHERE tenant_id=? AND `+column+`=? FOR UPDATE`, delta, tenant, id).Scan(&item)
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected == 0 {
		return notFound()
	}
	if !item.Enough {
		return conflict("No hay stock suficiente para completar la operación")
	}
	if expected != nil && item.Version != *expected {
		return conflict("El stock cambió; actualizá los datos antes de reintentar")
	}
	result := tx.Exec(`UPDATE `+table+` SET current_stock=current_stock+?::numeric WHERE tenant_id=? AND `+column+`=?`, delta, tenant, id)
	return result.Error
}
func (h *Handler) listInventory(r *http.Request, p Principal) (Result, error) {
	scope, err := h.tenantScope(r.Context(), p, r)
	if err != nil {
		return Result{}, err
	}
	limit, offset, err := pagination(r)
	if err != nil {
		return Result{}, err
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(CASE WHEN ? THEN to_jsonb(v) ELSE to_jsonb(v)-'unit_cost' END),'[]'::jsonb),'limit',?::int,'offset',?::int)::text payload FROM(
 SELECT * FROM(SELECT 'product' kind,tp.product_id id,p.name,tp.current_stock::numeric current_stock,tp.minimum_stock::numeric minimum_stock,tp.stock_version,'unit' unit,tp.track_stock,tp.is_available,(SELECT CASE WHEN count(*) FILTER(WHERE r.quantity_used*s.current_wholesale_cost IS NULL)>0 THEN NULL ELSE sum(r.quantity_used*s.current_wholesale_cost) END FROM public.recipes r LEFT JOIN public.supplies s ON s.id=r.supply_id AND s.tenant_id=tp.tenant_id WHERE r.product_id=tp.product_id) unit_cost FROM public.tenant_products tp JOIN public.products p ON p.id=tp.product_id WHERE tp.tenant_id=? AND tp.is_active AND p.is_active
 UNION ALL SELECT 'extra',id,name,current_stock,minimum_stock,stock_version,'unit',track_stock,is_active,current_unit_cost FROM public.extras WHERE tenant_id=? AND is_active
 UNION ALL SELECT 'supply',id,name,current_stock,minimum_stock,stock_version,measurement_unit,true,is_active,current_wholesale_cost FROM public.supplies WHERE tenant_id=? AND is_active)items ORDER BY kind,name,id LIMIT ? OFFSET ?)v`, p.Manager(), limit, offset, scope.TenantID, scope.TenantID, scope.TenantID, limit, offset))
}
func (h *Handler) listInventoryMovements(r *http.Request, p Principal) (Result, error) {
	scope, err := h.tenantScope(r.Context(), p, r)
	if err != nil {
		return Result{}, err
	}
	limit, offset, err := pagination(r)
	if err != nil {
		return Result{}, err
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb),'limit',?::int,'offset',?::int)::text payload FROM(SELECT * FROM public.inventory_movements WHERE tenant_id=? ORDER BY occurred_at DESC,id DESC LIMIT ? OFFSET ?)v`, limit, offset, scope.TenantID, limit, offset))
}
func (h *Handler) recordInventoryMovement(r *http.Request, p Principal) (Result, error) {
	if !p.Manager() {
		return Result{}, forbidden()
	}
	scope, err := h.tenantScope(r.Context(), p, r)
	if err != nil {
		return Result{}, err
	}
	input, err := decode[StockMovementRequest](r)
	if err != nil {
		return Result{}, err
	}
	if _, _, err = inventoryTable(input.ItemKind); err != nil {
		return Result{}, err
	}
	if !validID(input.ItemID) || input.ExpectedVersion == nil || *input.ExpectedVersion < 0 || !decimalPattern.MatchString(string(input.QuantityDelta)) || strings.Trim(string(input.QuantityDelta), "-0.") == "" || !isOneOf(input.Reason, "purchase", "production", "sale", "return", "waste", "adjustment") {
		return Result{}, badInput("Movimiento de stock inválido")
	}
	if input.ItemKind != "supply" && !integerDecimal(input.QuantityDelta) {
		return Result{}, badInput("Productos y extras requieren cantidades enteras")
	}
	return h.mutate(r, p, scope, "inventory.movement", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		if err := tx.Exec(`SELECT set_config('leguiburger.inventory_reason',?,true),set_config('leguiburger.expected_stock_version',?,true)`, input.Reason, strconv.FormatInt(*input.ExpectedVersion, 10)).Error; err != nil {
			return nil, 0, err
		}
		if err := changeStock(tx, scope.TenantID, input.ItemKind, input.ItemID, string(input.QuantityDelta), input.ExpectedVersion); err != nil {
			return nil, 0, err
		}
		data, err := jsonQuery(tx, `SELECT to_jsonb(m)::text payload FROM public.inventory_movements m WHERE tenant_id=? AND item_kind=? AND item_id=? AND operation_key=?`, scope.TenantID, input.ItemKind, input.ItemID, key)
		return data, 201, err
	})
}
func (h *Handler) listExpenses(r *http.Request, p Principal) (Result, error) {
	if !p.Manager() {
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
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb),'limit',?::int,'offset',?::int)::text payload FROM(SELECT * FROM public.operating_expenses WHERE tenant_id=? ORDER BY incurred_at DESC,id DESC LIMIT ? OFFSET ?)v`, limit, offset, scope.TenantID, limit, offset))
}
func (h *Handler) createExpense(r *http.Request, p Principal) (Result, error) {
	if !p.Manager() {
		return Result{}, forbidden()
	}
	scope, err := h.tenantScope(r.Context(), p, r)
	if err != nil {
		return Result{}, err
	}
	input, err := decode[ExpenseRequest](r)
	if err != nil {
		return Result{}, err
	}
	if input.Currency == "" {
		input.Currency = "ARS"
	}
	if !validMoney(input.Amount, true) || !currencyPattern.MatchString(input.Currency) || input.IncurredAt.IsZero() || len(input.Description) > 2000 || !isOneOf(input.Category, "payroll", "rent", "utilities", "processing_fees", "delivery", "marketing", "maintenance", "other") {
		return Result{}, badInput("Gasto operativo inválido")
	}
	return h.mutate(r, p, scope, "expenses.create", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		data, err := jsonQuery(tx, `INSERT INTO public.operating_expenses(tenant_id,category,amount,currency,incurred_at,description,created_by_user_id,idempotency_key) VALUES(?,?,?::numeric,?,?,?,?,?) RETURNING to_jsonb(operating_expenses)::text payload`, scope.TenantID, input.Category, string(input.Amount), input.Currency, input.IncurredAt, input.Description, p.ID, key)
		return data, 201, err
	})
}
