package operations

import (
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"strings"
)

type optionRule struct {
	ID, Name string
	Min, Max int
}
type optionValue struct{ ExtraID, GroupID, GroupName, Price string }

func loadProductOptions(tx *gorm.DB, tenant, product string) (bool, []optionRule, map[string]optionValue, error) {
	var configured bool
	if e := tx.Raw(`SELECT options_configured FROM public.tenant_products WHERE tenant_id=? AND product_id=?`, tenant, product).Scan(&configured).Error; e != nil {
		return false, nil, nil, e
	}
	values := map[string]optionValue{}
	rules := []optionRule{}
	if !configured {
		return false, rules, values, nil
	}
	if e := tx.Raw(`SELECT id::text,name,min_selections min,max_selections max FROM legui_private.product_option_groups WHERE tenant_id=? AND product_id=? ORDER BY position,id`, tenant, product).Scan(&rules).Error; e != nil {
		return true, nil, nil, e
	}
	var rows []optionValue
	if e := tx.Raw(`SELECT i.extra_id::text,i.group_id::text,g.name group_name,coalesce(i.price_override,e.current_price)::text price FROM legui_private.product_option_items i JOIN legui_private.product_option_groups g ON g.id=i.group_id JOIN public.extras e ON e.id=i.extra_id AND e.tenant_id=i.tenant_id WHERE i.tenant_id=? AND i.product_id=?`, tenant, product).Scan(&rows).Error; e != nil {
		return true, nil, nil, e
	}
	for _, row := range rows {
		values[row.ExtraID] = row
	}
	return true, rules, values, nil
}
func resolveProductOptions(tx *gorm.DB, tenant, product string, extras []OrderExtraRequest) (map[string]optionValue, error) {
	configured, rules, values, e := loadProductOptions(tx, tenant, product)
	if e != nil {
		return nil, e
	}
	if !configured {
		return values, nil
	}
	counts := map[string]int{}
	seen := map[string]bool{}
	for _, extra := range extras {
		choice, ok := values[extra.ExtraID]
		if !ok || seen[extra.ExtraID] || extra.Quantity < 1 || extra.Quantity > 5 {
			return nil, conflict("Una opción no corresponde a este producto o tiene una cantidad inválida")
		}
		seen[extra.ExtraID] = true
		counts[choice.GroupID] += extra.Quantity
	}
	for _, rule := range rules {
		if counts[rule.ID] < rule.Min || counts[rule.ID] > rule.Max {
			return nil, conflict("Revisá " + rule.Name + ": elegí entre " + strconv.Itoa(rule.Min) + " y " + strconv.Itoa(rule.Max) + " opciones por unidad")
		}
	}
	return values, nil
}
func optionsData(tx *gorm.DB, scope Scope, id string) (json.RawMessage, error) {
	return jsonQuery(tx, `SELECT jsonb_build_object('product_id',p.id,'name',p.name,'use_product_options',tp.options_configured,'groups',coalesce((SELECT jsonb_agg(jsonb_build_object('name',g.name,'min',g.min_selections,'max',g.max_selections,'options',coalesce((SELECT jsonb_agg(jsonb_build_object('extra_id',i.extra_id,'price_override',i.price_override::text) ORDER BY i.position,i.extra_id) FROM legui_private.product_option_items i WHERE i.group_id=g.id),'[]'::jsonb)) ORDER BY g.position,g.id) FROM legui_private.product_option_groups g WHERE g.tenant_id=tp.tenant_id AND g.product_id=p.id),'[]'::jsonb))::text payload FROM public.tenant_products tp JOIN public.products p ON p.id=tp.product_id WHERE tp.tenant_id=? AND p.id=? AND p.brand_id=?`, scope.TenantID, id, scope.BrandID)
}

type optionGroupInput struct {
	Name    string `json:"name"`
	Min     int    `json:"min"`
	Max     int    `json:"max"`
	Options []struct {
		ExtraID       string   `json:"extra_id"`
		PriceOverride *Decimal `json:"price_override"`
	} `json:"options"`
}

func (h *Handler) productOptions(r *http.Request, p Principal) (Result, error) {
	if !p.Manager() {
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
	if r.Method == "GET" {
		return resultJSON(optionsData(h.repo.DB.WithContext(r.Context()), scope, id))
	}
	in, e := decode[struct {
		Configured bool               `json:"use_product_options"`
		Groups     []optionGroupInput `json:"groups"`
	}](r)
	if e != nil {
		return Result{}, e
	}
	if len(in.Groups) > 6 {
		return Result{}, badInput("Cada producto admite hasta 6 grupos")
	}
	seen := map[string]bool{}
	for i := range in.Groups {
		g := &in.Groups[i]
		g.Name = strings.TrimSpace(g.Name)
		if len([]rune(g.Name)) < 2 || len([]rune(g.Name)) > 80 || g.Min < 0 || g.Max < 1 || g.Max > 5 || g.Min > g.Max || len(g.Options) < 1 || len(g.Options) > 30 || g.Min > len(g.Options)*5 {
			return Result{}, badInput("Revisá nombres, límites y opciones de cada grupo")
		}
		for _, o := range g.Options {
			if !validID(o.ExtraID) || seen[o.ExtraID] || (o.PriceOverride != nil && !o.PriceOverride.Money()) {
				return Result{}, badInput("Cada extra puede estar una sola vez en este producto; revisá sus precios")
			}
			seen[o.ExtraID] = true
		}
	}
	return h.mutate(r, p, scope, "product.options", in, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		var product string
		q := tx.Raw(`SELECT p.id::text FROM public.tenant_products tp JOIN public.products p ON p.id=tp.product_id AND p.brand_id=? WHERE tp.tenant_id=? AND p.id=? FOR UPDATE OF tp`, scope.BrandID, scope.TenantID, id).Scan(&product)
		if q.Error != nil {
			return nil, 0, q.Error
		}
		if q.RowsAffected != 1 {
			return nil, 0, notFound()
		}
		for _, g := range in.Groups {
			for _, o := range g.Options {
				var valid bool
				if e := tx.Raw(`SELECT EXISTS(SELECT 1 FROM public.extras WHERE tenant_id=? AND id=?)`, scope.TenantID, o.ExtraID).Scan(&valid).Error; e != nil {
					return nil, 0, e
				}
				if !valid {
					return nil, 0, notFound()
				}
			}
		}
		if e := tx.Exec(`DELETE FROM legui_private.product_option_groups WHERE tenant_id=? AND product_id=?`, scope.TenantID, id).Error; e != nil {
			return nil, 0, e
		}
		for i, g := range in.Groups {
			gid := uuid.NewString()
			if e := tx.Exec(`INSERT INTO legui_private.product_option_groups(id,tenant_id,product_id,name,min_selections,max_selections,position) VALUES(?,?,?,?,?,?,?)`, gid, scope.TenantID, id, g.Name, g.Min, g.Max, i).Error; e != nil {
				return nil, 0, e
			}
			for j, o := range g.Options {
				if e := tx.Exec(`INSERT INTO legui_private.product_option_items(tenant_id,product_id,group_id,extra_id,price_override,position) VALUES(?,?,?,?,?::numeric,?)`, scope.TenantID, id, gid, o.ExtraID, o.PriceOverride, j).Error; e != nil {
					return nil, 0, e
				}
			}
		}
		if e := tx.Exec(`UPDATE public.tenant_products SET options_configured=? WHERE tenant_id=? AND product_id=?`, in.Configured, scope.TenantID, id).Error; e != nil {
			return nil, 0, e
		}
		data, e := optionsData(tx, scope, id)
		return data, 200, e
	})
}
