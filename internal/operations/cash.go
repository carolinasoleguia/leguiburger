package operations

import (
	"encoding/json"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

type OpenCashRequest struct {
	OpeningBalance Decimal `json:"opening_balance"`
	Currency       string  `json:"currency"`
}
type CashMovementRequest struct {
	Kind   string  `json:"kind"`
	Amount Decimal `json:"amount"`
	Note   string  `json:"note"`
}
type CloseCashRequest struct {
	CountedBalance Decimal `json:"counted_balance"`
	Note           string  `json:"note"`
}

const cashSnapshotSQL = `SELECT (to_jsonb(s)||jsonb_build_object('opening_balance',s.opening_balance::text,'expected_balance',coalesce(s.expected_balance,s.opening_balance+coalesce((SELECT sum(CASE WHEN e.kind IN ('charge','cash_in') THEN e.amount ELSE -e.amount END) FROM legui_private.cash_entries e WHERE e.shift_id=s.id),0))::text,'counted_balance',s.counted_balance::text,'difference',s.difference::text,
 'cash_charged',coalesce((SELECT sum(amount) FROM legui_private.cash_entries WHERE shift_id=s.id AND kind='charge'),0)::text,
 'cash_refunded',coalesce((SELECT sum(amount) FROM legui_private.cash_entries WHERE shift_id=s.id AND kind='refund'),0)::text,
 'cash_in',coalesce((SELECT sum(amount) FROM legui_private.cash_entries WHERE shift_id=s.id AND kind='cash_in'),0)::text,
 'cash_out',coalesce((SELECT sum(amount) FROM legui_private.cash_entries WHERE shift_id=s.id AND kind='cash_out'),0)::text))::text payload FROM legui_private.cash_shifts s WHERE s.tenant_id=? AND s.id=?`

func cashLock(tx *gorm.DB, tenant, currency string) error {
	return tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "cash:"+tenant+":"+currency).Error
}
func cashActors(p Principal) (any, any) {
	if p.Kind == "user" {
		return p.ID, nil
	}
	return nil, p.ID
}
func lockCashShift(tx *gorm.DB, tenant, id string) (string, error) {
	var currency string
	q := tx.Raw(`SELECT currency FROM legui_private.cash_shifts WHERE tenant_id=? AND id=?`, tenant, id).Scan(&currency)
	if q.Error != nil {
		return "", q.Error
	}
	if q.RowsAffected != 1 {
		return "", notFound()
	}
	if e := cashLock(tx, tenant, currency); e != nil {
		return "", e
	}
	var open bool
	q = tx.Raw(`SELECT closed_at IS NULL FROM legui_private.cash_shifts WHERE tenant_id=? AND id=? FOR UPDATE`, tenant, id).Scan(&open)
	if q.Error != nil {
		return "", q.Error
	}
	if !open {
		return "", conflict("Ese turno de caja ya está cerrado")
	}
	return currency, nil
}
func (h *Handler) cashShifts(r *http.Request, p Principal) (Result, error) {
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
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v) ORDER BY (v.closed_at IS NULL) DESC,v.opened_at DESC,v.id DESC),'[]'::jsonb))::text payload FROM(SELECT s.id,s.currency,s.opened_at,s.closed_at,s.opening_balance::text,s.expected_balance::text,s.counted_balance::text,s.difference::text FROM legui_private.cash_shifts s WHERE tenant_id=? ORDER BY (closed_at IS NULL) DESC,opened_at DESC,id DESC LIMIT ? OFFSET ?)v`, scope.TenantID, limit, offset))
}
func (h *Handler) cashShift(r *http.Request, p Principal) (Result, error) {
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
	data, e := h.repo.JSON(r.Context(), cashSnapshotSQL, scope.TenantID, id)
	return resultJSON(data, e)
}
func (h *Handler) cashEntries(r *http.Request, p Principal) (Result, error) {
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
	limit, offset, e := pagination(r)
	if e != nil {
		return Result{}, e
	}
	if _, e = h.repo.JSON(r.Context(), cashSnapshotSQL, scope.TenantID, id); e != nil {
		return Result{}, e
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v) ORDER BY v.created_at DESC,v.id DESC),'[]'::jsonb))::text payload FROM(SELECT id,kind,amount::text,note,created_at,payment_id FROM legui_private.cash_entries WHERE tenant_id=? AND shift_id=? ORDER BY created_at DESC,id DESC LIMIT ? OFFSET ?)v`, scope.TenantID, id, limit, offset))
}
func (h *Handler) openCash(r *http.Request, p Principal) (Result, error) {
	if !p.Cashier() {
		return Result{}, forbidden()
	}
	scope, e := h.tenantScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	input, e := decode[OpenCashRequest](r)
	if e != nil {
		return Result{}, e
	}
	if input.Currency == "" {
		input.Currency = "ARS"
	}
	if !validMoney(input.OpeningBalance, false) || !currencyPattern.MatchString(input.Currency) {
		return Result{}, badInput("Revisá el efectivo inicial y la moneda")
	}
	return h.mutate(r, p, scope, "cash.open", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		var enabled bool
		q := tx.Raw(`SELECT closed_at IS NULL AND suspended_at IS NULL FROM public.brands WHERE id=? FOR SHARE`, scope.BrandID).Scan(&enabled)
		if q.Error != nil {
			return nil, 0, q.Error
		}
		if q.RowsAffected != 1 || !enabled {
			return nil, 0, forbidden()
		}

		if e := cashLock(tx, scope.TenantID, input.Currency); e != nil {
			return nil, 0, e
		}
		u, emp := cashActors(p)
		data, e := jsonQuery(tx, `INSERT INTO legui_private.cash_shifts(tenant_id,currency,opening_balance,opened_by_user_id,opened_by_employee_id) VALUES(?,?,?::numeric,?,?) RETURNING to_jsonb(cash_shifts)::text payload`, scope.TenantID, input.Currency, string(input.OpeningBalance), u, emp)
		return data, 201, e
	})
}
func (h *Handler) moveCash(r *http.Request, p Principal) (Result, error) {
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
	input, e := decode[CashMovementRequest](r)
	if e != nil {
		return Result{}, e
	}
	input.Note = strings.TrimSpace(input.Note)
	if !isOneOf(input.Kind, "cash_in", "cash_out") || !validMoney(input.Amount, true) || len([]rune(input.Note)) < 4 || len([]rune(input.Note)) > 1000 {
		return Result{}, badInput("Indicá un ingreso o retiro, importe y motivo de al menos 4 caracteres")
	}
	return h.mutate(r, p, scope, "cash.movement", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		currency, e := lockCashShift(tx, scope.TenantID, id)
		if e != nil {
			return nil, 0, e
		}
		if input.Kind == "cash_out" {
			var enough bool
			e = tx.Raw(`SELECT opening_balance+coalesce((SELECT sum(CASE WHEN kind IN ('charge','cash_in') THEN amount ELSE -amount END) FROM legui_private.cash_entries WHERE shift_id=s.id),0)>=?::numeric FROM legui_private.cash_shifts s WHERE id=?`, string(input.Amount), id).Scan(&enough).Error
			if e != nil {
				return nil, 0, e
			}
			if !enough {
				return nil, 0, conflict("El retiro supera el efectivo esperado en la caja")
			}
		}
		u, emp := cashActors(p)
		data, e := jsonQuery(tx, `INSERT INTO legui_private.cash_entries(tenant_id,shift_id,currency,kind,amount,note,actor_user_id,actor_employee_id) VALUES(?,?,?,?,?::numeric,?,?,?) RETURNING to_jsonb(cash_entries)::text payload`, scope.TenantID, id, currency, input.Kind, string(input.Amount), input.Note, u, emp)
		return data, 201, e
	})
}
func (h *Handler) closeCash(r *http.Request, p Principal) (Result, error) {
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
	input, e := decode[CloseCashRequest](r)
	if e != nil {
		return Result{}, e
	}
	input.Note = strings.TrimSpace(input.Note)
	if !validMoney(input.CountedBalance, false) || len([]rune(input.Note)) > 1000 {
		return Result{}, badInput("Revisá el efectivo contado y la nota de cierre")
	}
	return h.mutate(r, p, scope, "cash.close", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		if _, e := lockCashShift(tx, scope.TenantID, id); e != nil {
			return nil, 0, e
		}
		if input.Note == "" {
			var differs bool
			if e := tx.Raw(`SELECT opening_balance+coalesce((SELECT sum(CASE WHEN kind IN ('charge','cash_in') THEN amount ELSE -amount END) FROM legui_private.cash_entries WHERE shift_id=s.id),0)<>?::numeric FROM legui_private.cash_shifts s WHERE id=?`, string(input.CountedBalance), id).Scan(&differs).Error; e != nil {
				return nil, 0, e
			}
			if differs {
				return nil, 0, badInput("Explicá el motivo de la diferencia antes de cerrar la caja")
			}
		}
		u, emp := cashActors(p)
		if e := tx.Exec(`UPDATE legui_private.cash_shifts s SET closed_at=clock_timestamp(),counted_balance=?::numeric,closing_note=?,closed_by_user_id=?,closed_by_employee_id=?,expected_balance=opening_balance+coalesce((SELECT sum(CASE WHEN kind IN ('charge','cash_in') THEN amount ELSE -amount END) FROM legui_private.cash_entries WHERE shift_id=s.id),0) WHERE tenant_id=? AND id=?`, string(input.CountedBalance), input.Note, u, emp, scope.TenantID, id).Error; e != nil {
			return nil, 0, e
		}
		data, e := jsonQuery(tx, cashSnapshotSQL, scope.TenantID, id)
		return data, 200, e
	})
}
