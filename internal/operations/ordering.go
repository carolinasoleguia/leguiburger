package operations

import (
	"encoding/json"
	"gorm.io/gorm"
	"leguiburger/internal/buyeraccess"
	"net/http"
	"strconv"
	"time"
	_ "time/tzdata"
)

type OrderingPeriod struct {
	Day    int    `json:"day"`
	Opens  string `json:"opens"`
	Closes string `json:"closes"`
}
type OrderingSettings struct {
	Timezone        string           `json:"timezone"`
	ScheduleEnabled bool             `json:"schedule_enabled"`
	Paused          bool             `json:"paused"`
	PauseMinutes    int              `json:"pause_minutes"`
	PendingMinutes  int              `json:"pending_minutes"`
	PrepMin         int              `json:"prep_min"`
	PrepMax         int              `json:"prep_max"`
	Periods         []OrderingPeriod `json:"periods"`
}

func orderingData(tx *gorm.DB, tenant string) (json.RawMessage, error) {
	return jsonQuery(tx, `SELECT jsonb_build_object('timezone',coalesce(c.timezone,'America/Argentina/Buenos_Aires'),'schedule_enabled',coalesce(c.schedule_enabled,false),'paused',coalesce(c.paused AND(c.pause_until IS NULL OR c.pause_until>clock_timestamp()),false),'pause_until',c.pause_until,'pause_minutes',0,'pending_minutes',coalesce(c.pending_minutes,15),'prep_min',coalesce(c.prep_min,20),'prep_max',coalesce(c.prep_max,35),'receiving',legui_private.tenant_receiving(t.id),'periods',coalesce((SELECT jsonb_agg(jsonb_build_object('day',p.day,'opens',lpad((p.opens/60)::text,2,'0')||':'||lpad((p.opens%60)::text,2,'0'),'closes',lpad((p.closes/60)::text,2,'0')||':'||lpad((p.closes%60)::text,2,'0')) ORDER BY p.day,p.opens) FROM legui_private.ordering_periods p WHERE p.tenant_id=t.id),'[]'::jsonb))::text payload FROM public.tenants t LEFT JOIN legui_private.tenant_ordering c ON c.tenant_id=t.id WHERE t.id=?`, tenant)
}
func clockMinutes(raw string, close bool) (int, error) {
	if raw == "24:00" && close {
		return 1440, nil
	}
	if len(raw) != 5 || raw[2] != ':' {
		return 0, badInput("Usá horarios HH:MM")
	}
	h, e := strconv.Atoi(raw[:2])
	m, f := strconv.Atoi(raw[3:])
	if e != nil || f != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, badInput("Horario inválido")
	}
	return h*60 + m, nil
}

type minutePeriod struct{ day, opens, closes int }

func validateOrdering(input OrderingSettings) ([]minutePeriod, error) {
	if _, e := time.LoadLocation(input.Timezone); e != nil || input.Timezone == "" || input.Timezone == "Local" || len(input.Timezone) > 80 {
		return nil, badInput("Zona horaria inválida")
	}
	if input.PendingMinutes < 5 || input.PendingMinutes > 60 || input.PrepMin < 1 || input.PrepMin > 180 || input.PrepMax < input.PrepMin || input.PrepMax > 240 || input.PauseMinutes < 0 || input.PauseMinutes > 720 || len(input.Periods) > 28 {
		return nil, badInput("Revisá los plazos y horarios del local")
	}
	periods := []minutePeriod{}
	add := func(day, a, b int) error {
		if a == b {
			return nil
		}
		for _, p := range periods {
			if p.day == day && a < p.closes && b > p.opens {
				return badInput("Los horarios no pueden superponerse")
			}
		}
		periods = append(periods, minutePeriod{day, a, b})
		return nil
	}
	for _, p := range input.Periods {
		if p.Day < 0 || p.Day > 6 {
			return nil, badInput("Día inválido")
		}
		a, e := clockMinutes(p.Opens, false)
		if e != nil {
			return nil, e
		}
		b, e := clockMinutes(p.Closes, true)
		if e != nil {
			return nil, e
		}
		if a == b {
			return nil, badInput("La apertura y el cierre deben ser diferentes")
		}
		if b > a {
			if e = add(p.Day, a, b); e != nil {
				return nil, e
			}
		} else {
			if e = add(p.Day, a, 1440); e != nil {
				return nil, e
			}
			if e = add((p.Day+1)%7, 0, b); e != nil {
				return nil, e
			}
		}
	}
	if input.ScheduleEnabled && len(periods) == 0 {
		return nil, badInput("Agregá al menos un horario o pausá el local")
	}
	return periods, nil
}
func (h *Handler) orderingSettings(r *http.Request, p Principal) (Result, error) {
	if !p.Manager() {
		return Result{}, forbidden()
	}
	scope, e := h.tenantScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	if r.Method == "GET" {
		return resultJSON(orderingData(h.repo.DB.WithContext(r.Context()), scope.TenantID))
	}
	input, e := decode[OrderingSettings](r)
	if e != nil {
		return Result{}, e
	}
	periods, e := validateOrdering(input)
	if e != nil {
		return Result{}, e
	}
	return h.mutate(r, p, scope, "ordering.settings", input, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		// Tenant lock serializes settings changes against guest quote/create FOR SHARE.
		if e := tx.Exec(`SELECT id FROM public.tenants WHERE id=? FOR UPDATE`, scope.TenantID).Error; e != nil {
			return nil, 0, e
		}
		e := tx.Exec(`INSERT INTO legui_private.tenant_ordering(tenant_id,timezone,schedule_enabled,paused,pause_until,pending_minutes,prep_min,prep_max) VALUES(?,?,?,?,CASE WHEN ?>0 AND ? THEN clock_timestamp()+make_interval(mins=>?) ELSE NULL END,?,?,?) ON CONFLICT(tenant_id) DO UPDATE SET timezone=excluded.timezone,schedule_enabled=excluded.schedule_enabled,paused=excluded.paused,pause_until=excluded.pause_until,pending_minutes=excluded.pending_minutes,prep_min=excluded.prep_min,prep_max=excluded.prep_max,updated_at=clock_timestamp()`, scope.TenantID, input.Timezone, input.ScheduleEnabled, input.Paused, input.PauseMinutes, input.Paused, input.PauseMinutes, input.PendingMinutes, input.PrepMin, input.PrepMax).Error
		if e != nil {
			return nil, 0, e
		}
		if e = tx.Exec(`DELETE FROM legui_private.ordering_periods WHERE tenant_id=?`, scope.TenantID).Error; e != nil {
			return nil, 0, e
		}
		for _, v := range periods {
			if e = tx.Exec(`INSERT INTO legui_private.ordering_periods(tenant_id,day,opens,closes) VALUES(?,?,?,?)`, scope.TenantID, v.day, v.opens, v.closes).Error; e != nil {
				return nil, 0, e
			}
		}
		data, e := orderingData(tx, scope.TenantID)
		return data, 200, e
	})
}
func (h *Handler) orderInbox(r *http.Request, p Principal) (Result, error) {
	if !p.Cashier() {
		return Result{}, forbidden()
	}
	scope, e := h.tenantScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('pending_count',(SELECT count(*) FROM public.orders o JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE o.tenant_id=? AND o.payment_environment='production' AND s.code='pending'),'items',coalesce((SELECT jsonb_agg(to_jsonb(v) ORDER BY v.created_at,v.id) FROM(SELECT o.id,o.version,o.created_at,o.buyer_name,o.channel,o.total_due::text,o.pending_deadline,s.code status FROM public.orders o JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE o.tenant_id=? AND o.payment_environment='production' AND s.code='pending' ORDER BY o.created_at,o.id LIMIT 100)v),'[]'::jsonb))::text payload`, scope.TenantID, scope.TenantID))
}
func (h *Handler) buyerCancel(r *http.Request) (Result, error) {
	if e := buyeraccess.Rate(r.Context(), h.repo.DB, r, "cancel", 120); e != nil {
		return Result{}, buyerProblem(e)
	}
	if _, e := decode[struct{}](r); e != nil {
		return Result{}, e
	}
	var data json.RawMessage
	e := h.repo.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		grant, e := buyeraccess.ResolveRequest(r.Context(), tx, r, true)
		if e != nil {
			return e
		}
		var row struct {
			Status    string
			Paid      bool
			Uncertain bool
		}
		q := tx.Raw(`SELECT s.code status,o.total_paid>0 paid,EXISTS(SELECT 1 FROM legui_private.mp_checkouts x WHERE x.order_id=o.id AND x.state IN ('prepared','submitting','created','review') AND(x.expires_at>clock_timestamp() OR x.state IN ('submitting','review'))) uncertain FROM public.orders o JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE o.id=? AND o.tenant_id=? AND o.channel='web' AND o.payment_environment='production' FOR UPDATE OF o`, grant.OrderID, grant.TenantID).Scan(&row)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected != 1 {
			return notFound()
		}
		if row.Status != "canceled" {
			if row.Status != "pending" || row.Paid || row.Uncertain {
				return conflict("El pedido ya fue confirmado o tiene un cobro en curso. Contactá al local")
			}
			if e = tx.Exec(`UPDATE public.orders SET order_status_id=(SELECT id FROM public.order_statuses_lookup WHERE code='canceled'),ended_reason='Cancelado por el comprador antes de la confirmación' WHERE id=?`, grant.OrderID).Error; e != nil {
				return e
			}
		}
		data, e = h.buyerReceiptData(r, tx, grant)
		return e
	})
	return resultJSON(data, buyerProblem(e))
}
