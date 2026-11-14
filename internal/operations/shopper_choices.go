package operations

import (
	"encoding/json"
	"gorm.io/gorm"
	"leguiburger/internal/buyeraccess"
	"leguiburger/internal/shopperidentity"
	"math/big"
	"net/http"
)

func (h *Handler) shopperFavorites(r *http.Request) (Result, error) {
	p, e := h.shopperProfile(r)
	if e != nil {
		return Result{}, e
	}
	if r.Method == "GET" {
		limit, offset, e := pagination(r)
		if e != nil {
			return Result{}, e
		}
		tenant := r.URL.Query().Get("tenant_id")
		if tenant != "" && !validID(tenant) {
			return Result{}, badInput("Local inválido")
		}
		return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v) ORDER BY v.created_at DESC,v.product_id),'[]'::jsonb))::text payload FROM(SELECT f.tenant_id,f.product_id,f.created_at,p.name,p.image_url,b.name business,coalesce(nullif(t.display_name,''),t.subdomain) location,coalesce(tp.price_override,p.base_price)::text price,(tp.is_available AND coalesce(legui_private.product_capacity(t.id,p.id)>0,true) AND legui_private.product_groups_ready(t.id,p.id)) available,tp.options_configured,legui_private.product_options(t.id,p.id) option_groups FROM legui_private.shopper_favorites f JOIN public.tenants t ON t.id=f.tenant_id AND t.active AND t.menu_published JOIN public.brands b ON b.id=t.brand_id AND b.suspended_at IS NULL AND b.closed_at IS NULL JOIN public.products p ON p.id=f.product_id AND p.brand_id=b.id AND p.is_active JOIN public.tenant_products tp ON tp.product_id=p.id AND tp.tenant_id=t.id AND tp.is_active WHERE f.shopper_id=? AND (?='' OR f.tenant_id=nullif(?,'')::uuid) ORDER BY f.created_at DESC,f.product_id LIMIT ? OFFSET ?)v`, p.ID, tenant, tenant, limit, offset))
	}
	if e := buyeraccess.Rate(r.Context(), h.repo.DB, r, "favorites:"+p.ID, 240); e != nil {
		return Result{}, buyerProblem(e)
	}
	tenant, product := r.PathValue("tenant"), r.PathValue("product")
	if !validID(tenant) || !validID(product) {
		return Result{}, notFound()
	}
	if _, e = decode[struct{}](r); e != nil {
		return Result{}, e
	}
	e = h.repo.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if _, e := shopperidentity.Lock(r.Context(), tx, r); e != nil {
			return unauthorized()
		}
		if e := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "favorites:"+p.ID).Error; e != nil {
			return e
		}
		if r.Method == "DELETE" {
			return tx.Exec(`DELETE FROM legui_private.shopper_favorites WHERE shopper_id=? AND tenant_id=? AND product_id=?`, p.ID, tenant, product).Error
		}
		var visible bool
		if e := tx.Raw(`SELECT EXISTS(SELECT 1 FROM public.tenant_products tp JOIN public.tenants t ON t.id=tp.tenant_id AND t.active AND t.menu_published JOIN public.brands b ON b.id=t.brand_id AND b.suspended_at IS NULL AND b.closed_at IS NULL JOIN public.products p ON p.id=tp.product_id AND p.brand_id=b.id AND p.is_active WHERE tp.tenant_id=? AND tp.product_id=? AND tp.is_active)`, tenant, product).Scan(&visible).Error; e != nil {
			return e
		}
		if !visible {
			return notFound()
		}
		var n int64
		if e := tx.Raw(`SELECT count(*) FROM legui_private.shopper_favorites WHERE shopper_id=? AND NOT(tenant_id=? AND product_id=?)`, p.ID, tenant, product).Scan(&n).Error; e != nil {
			return e
		}
		if n >= 200 {
			return conflict("Podés guardar hasta 200 favoritos")
		}
		return tx.Exec(`INSERT INTO legui_private.shopper_favorites(shopper_id,tenant_id,product_id) VALUES(?,?,?) ON CONFLICT DO NOTHING`, p.ID, tenant, product).Error
	})
	return resultJSON(json.RawMessage(`{"ok":true}`), e)
}

type repeatItem struct {
	OptionsConfigured bool            `json:"options_configured"`
	OptionGroups      json.RawMessage `json:"option_groups"`
	ProductID         string          `json:"product_id"`
	Name              string          `json:"name"`
	Price             string          `json:"price"`
	Quantity          int             `json:"quantity"`
	Note              string          `json:"note"`
	Extras            []BuyerExtra    `json:"extras"`
}

func (h *Handler) repeatOrder(r *http.Request) (Result, error) {
	p, e := h.shopperProfile(r)
	if e != nil {
		return Result{}, e
	}
	if e := buyeraccess.Rate(r.Context(), h.repo.DB, r, "repeat:"+p.ID, 120); e != nil {
		return Result{}, buyerProblem(e)
	}
	id, e := resourceID(r)
	if e != nil {
		return Result{}, e
	}
	if _, e = decode[struct{}](r); e != nil {
		return Result{}, e
	}
	var data json.RawMessage
	e = h.repo.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		var source struct {
			TenantID, Status string
			Settled          bool
		}
		q := tx.Raw(`SELECT o.tenant_id::text,s.code status,o.total_paid>=o.total_due settled FROM legui_private.shopper_orders so JOIN public.orders o ON o.id=so.order_id JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE so.shopper_id=? AND o.id=?`, p.ID, id).Scan(&source)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected != 1 {
			return notFound()
		}
		if !(isOneOf(source.Status, "canceled", "rejected") || source.Status == "delivered" && source.Settled) {
			return conflict("Esperá a recibir y completar el pago del pedido antes de repetirlo")
		}
		scope, e := buyerLocation(tx, source.TenantID)
		if e != nil {
			return e
		}
		var details []struct {
			ID, ProductID, Name, Note, OldUnit string
			Quantity                           int
		}
		if e = tx.Raw(`SELECT d.id::text,d.product_id::text,d.product_name name,d.preparation_note note,d.quantity,(d.final_price_unit+coalesce((SELECT sum(x.extra_price_charged*x.quantity) FROM public.order_detail_extras x WHERE x.order_detail_id=d.id),0))::text old_unit FROM public.order_details d WHERE d.order_id=? ORDER BY d.id LIMIT 30`, id).Scan(&details).Error; e != nil {
			return e
		}
		items := []repeatItem{}
		changes := []string{}
		basket := BuyerBasket{Fulfillment: "pickup"}
		total := 0
		for _, old := range details {
			var current struct {
				ID, Name, Price   string
				Groups            string
				OptionsConfigured bool
				Available         bool
			}
			q := tx.Raw(`SELECT p.id::text,p.name,coalesce(tp.price_override,p.base_price)::text price,tp.options_configured,legui_private.product_options(tp.tenant_id,p.id)::text groups,(tp.is_available AND coalesce(legui_private.product_capacity(tp.tenant_id,p.id)>=?,true) AND legui_private.product_groups_ready(tp.tenant_id,p.id)) available FROM public.products p JOIN public.tenant_products tp ON tp.product_id=p.id AND tp.tenant_id=? AND tp.is_active WHERE p.id=? AND p.brand_id=? AND p.is_active FOR SHARE OF p,tp`, old.Quantity, scope.TenantID, old.ProductID, scope.BrandID).Scan(&current)
			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected != 1 || !current.Available || old.Quantity > 20 || total+old.Quantity > 100 {
				changes = append(changes, old.Name+": no está disponible en esa cantidad y no se agregó.")
				continue
			}
			currentPrice, err := priceRat(current.Price)
			if err != nil {
				return err
			}
			configured, rules, optionValues, e := loadProductOptions(tx, scope.TenantID, current.ID)
			if e != nil {
				return e
			}
			groupCounts := map[string]int{}
			groupLimits := map[string]int{}
			for _, rule := range rules {
				groupLimits[rule.ID] = rule.Max
			}
			item := repeatItem{OptionsConfigured: current.OptionsConfigured, OptionGroups: json.RawMessage(current.Groups), ProductID: current.ID, Name: current.Name, Price: currentPrice.FloatString(2), Quantity: old.Quantity, Note: old.Note, Extras: []BuyerExtra{}}
			requested := OrderItemRequest{ProductID: item.ProductID, Quantity: item.Quantity, Note: item.Note}
			unit, _ := priceRat(item.Price)
			var extras []struct {
				ID, Name string
				Quantity int
			}
			if e = tx.Raw(`SELECT extra_id::text id,extra_name name,quantity FROM public.order_detail_extras WHERE order_detail_id=? ORDER BY id`, old.ID).Scan(&extras).Error; e != nil {
				return e
			}
			for _, x := range extras {
				choice, allowed := optionValues[x.ID]
				if configured && (!allowed || groupCounts[choice.GroupID]+x.Quantity > groupLimits[choice.GroupID]) {
					changes = append(changes, old.Name+": "+x.Name+" ya no corresponde a las opciones actuales. Volvé a elegirlo.")
					continue
				}

				var extra struct {
					Name, Price string
					Available   bool
				}
				q := tx.Raw(`SELECT name,current_price::text price,(NOT track_stock OR current_stock>=?) available FROM public.extras WHERE id=? AND tenant_id=? AND is_active FOR SHARE`, x.Quantity*old.Quantity, x.ID, scope.TenantID).Scan(&extra)
				if q.Error != nil {
					return q.Error
				}
				if q.RowsAffected != 1 || !extra.Available {
					changes = append(changes, old.Name+": el extra "+x.Name+" ya no está disponible y se quitó.")
					continue
				}
				groupName := ""
				if configured {
					extra.Price = choice.Price
					groupName = choice.GroupName
					groupCounts[choice.GroupID] += x.Quantity
				}
				item.Extras = append(item.Extras, BuyerExtra{ID: x.ID, Name: extra.Name, Price: extra.Price, Quantity: x.Quantity, GroupName: groupName})
				requested.Extras = append(requested.Extras, OrderExtraRequest{ExtraID: x.ID, Quantity: x.Quantity})
				price, _ := priceRat(extra.Price)
				unit.Add(unit, new(big.Rat).Mul(price, big.NewRat(int64(x.Quantity), 1)))
			}
			previous, _ := priceRat(old.OldUnit)
			if previous != nil && unit.Cmp(previous) != 0 {
				changes = append(changes, item.Name+": el precio actual es diferente al de tu compra anterior.")
			}
			items = append(items, item)
			basket.Items = append(basket.Items, requested)
			total += item.Quantity
		}
		if len(items) > 0 {
			if _, e := quoteBuyer(tx, scope, basket); e != nil {
				if problem, ok := e.(*apiError); ok {
					changes = append(changes, problem.Message)
				} else {
					return e
				}
			}
		}
		data, e = json.Marshal(map[string]any{"tenant_id": scope.TenantID, "source_order_id": id, "items": items, "changes": changes, "notice": "Revisá tu pedido con los precios actuales. Los cupones y el envío anteriores no se vuelven a aplicar."})
		return e
	})
	return resultJSON(data, e)
}
