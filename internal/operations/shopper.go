package operations

import (
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"leguiburger/internal/buyeraccess"
	"leguiburger/internal/shopperidentity"
	"net/http"
	"strings"
	"time"
)

func (h *Handler) shopperProfile(r *http.Request) (shopperidentity.Profile, error) {
	p, e := shopperidentity.Current(r.Context(), h.repo.DB, r)
	if e != nil {
		return p, unauthorized()
	}
	return p, nil
}
func (h *Handler) shopperOrders(r *http.Request) (Result, error) {
	p, e := h.shopperProfile(r)
	if e != nil {
		return Result{}, e
	}
	limit, offset, e := pagination(r)
	if e != nil {
		return Result{}, e
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb))::text payload FROM (SELECT o.id,o.tenant_id,b.name business,coalesce(nullif(t.display_name,''),t.subdomain) location,s.code status,o.created_at,o.total_due::text,o.total_paid::text,o.currency,o.fulfillment,(SELECT count(*) FROM public.order_details d WHERE d.order_id=o.id) lines FROM legui_private.shopper_orders a JOIN public.orders o ON o.id=a.order_id JOIN public.tenants t ON t.id=o.tenant_id JOIN public.brands b ON b.id=t.brand_id JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE a.shopper_id=? ORDER BY o.created_at DESC,o.id DESC LIMIT ? OFFSET ?)v`, p.ID, limit, offset))
}
func (h *Handler) claimOrder(r *http.Request) (Result, error) {
	p, e := h.shopperProfile(r)
	if e != nil {
		return Result{}, e
	}
	in, e := decode[struct {
		TenantID string `json:"tenant_id"`
	}](r)
	if e != nil || !validID(in.TenantID) {
		return Result{}, badInput("Elegí el local del pedido")
	}
	r.Header.Set("X-Buyer-Tenant", in.TenantID)
	r.Header.Del("X-Shopper-Order")
	g, e := buyeraccess.Resolve(r.Context(), h.repo.DB, buyeraccess.Token(r), false)
	if e != nil || g.TenantID != in.TenantID {
		return Result{}, notFound()
	}
	e = h.repo.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec(`INSERT INTO legui_private.shopper_orders(order_id,shopper_id) VALUES(?,?) ON CONFLICT(order_id) DO NOTHING`, g.OrderID, p.ID).Error; e != nil {
			return e
		}
		var own bool
		if e := tx.Raw(`SELECT shopper_id=? FROM legui_private.shopper_orders WHERE order_id=?`, p.ID, g.OrderID).Scan(&own).Error; e != nil {
			return e
		}
		if !own {
			return conflict("El pedido ya está vinculado a otra cuenta")
		}
		return nil
	})
	if e != nil {
		return Result{}, e
	}
	return Result{Data: json.RawMessage(`{"ok":true}`)}, nil
}
func (h *Handler) shopperPreferences(r *http.Request) (Result, error) {
	p, e := h.shopperProfile(r)
	if e != nil {
		return Result{}, e
	}
	if r.Method == "GET" {
		return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb))::text payload FROM(SELECT b.id,b.name,coalesce(p.newsletter,false) newsletter,coalesce(p.personalization,false) personalization FROM public.brands b LEFT JOIN legui_private.shopper_preferences p ON p.brand_id=b.id AND p.shopper_id=? WHERE p.shopper_id IS NOT NULL OR (b.suspended_at IS NULL AND b.closed_at IS NULL AND EXISTS(SELECT 1 FROM public.tenants t WHERE t.brand_id=b.id AND t.active AND t.menu_published)) ORDER BY b.name)v`, p.ID))
	}
	in, e := decode[struct {
		BrandID         string `json:"brand_id"`
		Newsletter      *bool  `json:"newsletter"`
		Personalization *bool  `json:"personalization"`
	}](r)
	if e != nil || !validID(in.BrandID) || (in.Newsletter == nil && in.Personalization == nil) {
		return Result{}, badInput("Preferencia inválida")
	}
	var exists, previous bool
	if e = h.repo.DB.WithContext(r.Context()).Raw(`SELECT EXISTS(SELECT 1 FROM public.tenants t JOIN public.brands b ON b.id=t.brand_id WHERE t.brand_id=? AND t.menu_published AND t.active AND b.suspended_at IS NULL AND b.closed_at IS NULL)`, in.BrandID).Scan(&exists).Error; e != nil {
		return Result{}, e
	}
	if !exists {
		if e = h.repo.DB.WithContext(r.Context()).Raw(`SELECT EXISTS(SELECT 1 FROM legui_private.shopper_preferences WHERE shopper_id=? AND brand_id=?)`, p.ID, in.BrandID).Scan(&previous).Error; e != nil {
			return Result{}, e
		}
		if !previous {
			return Result{}, notFound()
		}
		if in.Newsletter != nil && *in.Newsletter || in.Personalization != nil && *in.Personalization {
			return Result{}, conflict("La marca no está disponible. Podés desactivar sus preferencias")
		}
	}
	e = h.repo.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if _, e := shopperidentity.Lock(r.Context(), tx, r); e != nil {
			return e
		}
		if e := tx.Exec(`INSERT INTO legui_private.shopper_preferences(shopper_id,brand_id,newsletter,personalization) VALUES(?,?,coalesce(?,false),coalesce(?,false)) ON CONFLICT(shopper_id,brand_id) DO UPDATE SET newsletter=coalesce(?,shopper_preferences.newsletter),personalization=coalesce(?,shopper_preferences.personalization),updated_at=clock_timestamp()`, p.ID, in.BrandID, in.Newsletter, in.Personalization, in.Newsletter, in.Personalization).Error; e != nil {
			return e
		}
		return tx.Exec(`INSERT INTO legui_private.shopper_preference_events(shopper_id,brand_id,newsletter,personalization) SELECT shopper_id,brand_id,newsletter,personalization FROM legui_private.shopper_preferences WHERE shopper_id=? AND brand_id=?`, p.ID, in.BrandID).Error
	})
	if e != nil {
		return Result{}, e
	}
	return Result{Data: json.RawMessage(`{"ok":true}`)}, nil
}
func (h *Handler) shopperCoupons(r *http.Request) (Result, error) {
	p, e := h.shopperProfile(r)
	if e != nil {
		return Result{}, e
	}
	return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb))::text payload FROM(SELECT c.id,c.code,c.percent,c.minimum::text,c.expires_at,b.name business,c.tenant_id,(c.shopper_id IS NOT NULL) personal,(SELECT count(*) FROM legui_private.shopper_coupon_uses u WHERE u.coupon_id=c.id AND u.shopper_id=?)>0 used,(SELECT t.id FROM public.tenants t WHERE t.brand_id=c.brand_id AND t.active AND t.menu_published AND (c.tenant_id IS NULL OR t.id=c.tenant_id) ORDER BY t.id LIMIT 1) location_id FROM legui_private.shopper_coupons c JOIN public.brands b ON b.id=c.brand_id AND b.suspended_at IS NULL AND b.closed_at IS NULL WHERE c.active AND c.expires_at>clock_timestamp() AND (c.shopper_id IS NULL OR c.shopper_id=?) AND (SELECT count(*) FROM legui_private.shopper_coupon_uses u WHERE u.coupon_id=c.id)<c.max_uses AND EXISTS(SELECT 1 FROM public.tenants t WHERE t.brand_id=c.brand_id AND t.active AND t.menu_published AND(c.tenant_id IS NULL OR c.tenant_id=t.id)) ORDER BY c.expires_at LIMIT 100)v`, p.ID, p.ID))
}
func (h *Handler) marketing(r *http.Request, p Principal) (Result, error) {
	if !p.Manager() {
		return Result{}, forbidden()
	}
	scope, e := h.tenantScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	if strings.HasSuffix(r.URL.Path, "/customers") {
		return resultJSON(h.repo.JSON(r.Context(), `WITH own AS(SELECT o.*,s.code status FROM public.orders o JOIN public.tenants t ON t.id=o.tenant_id JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE t.brand_id=? AND o.payment_environment='production'),customers AS(SELECT a.shopper_id,count(*) orders,sum(CASE WHEN o.status NOT IN ('canceled','rejected') THEN o.total_paid ELSE 0 END) spent,count(*) FILTER(WHERE o.total_paid>=o.total_due AND o.total_due>0 AND o.status NOT IN ('pending','canceled','rejected')) paid_orders,max(o.created_at) FILTER(WHERE o.total_paid>0 AND o.status NOT IN ('canceled','rejected')) last_purchase FROM own o JOIN legui_private.shopper_orders a ON a.order_id=o.id GROUP BY a.shopper_id) SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb))::text payload FROM(SELECT p.id,p.name,c.orders,coalesce(pref.newsletter,false) AND p.email_verified newsletter_confirmed,coalesce(pref.personalization,false) personalization,CASE WHEN pref.personalization THEN c.spent::text END spent,CASE WHEN pref.personalization THEN c.last_purchase END last_purchase,CASE WHEN pref.personalization THEN c.paid_orders END paid_orders,CASE WHEN pref.personalization THEN CASE WHEN c.last_purchase<clock_timestamp()-interval '30 days' THEN 'Para recuperar' WHEN c.paid_orders>=3 THEN 'Frecuente' WHEN c.paid_orders=0 THEN 'Primera compra' ELSE 'En crecimiento' END END segment,CASE WHEN pref.personalization THEN (SELECT d.product_name FROM own o JOIN legui_private.shopper_orders a ON a.order_id=o.id JOIN public.order_details d ON d.order_id=o.id WHERE a.shopper_id=p.id AND o.total_paid>0 AND o.status NOT IN ('canceled','rejected') GROUP BY d.product_id,d.product_name ORDER BY sum(d.quantity) DESC,d.product_name LIMIT 1) END favorite_product FROM customers c JOIN legui_private.shoppers p ON p.id=c.shopper_id LEFT JOIN legui_private.shopper_preferences pref ON pref.shopper_id=p.id AND pref.brand_id=? ORDER BY p.name LIMIT 100)v`, scope.BrandID, scope.BrandID))
	}
	if r.Method == "GET" {
		return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(to_jsonb(v)),'[]'::jsonb))::text payload FROM(SELECT c.id,c.code,c.percent,c.minimum::text,c.max_uses,c.active,c.expires_at,c.tenant_id,p.name recipient,(SELECT count(*) FROM legui_private.shopper_coupon_uses u WHERE u.coupon_id=c.id) uses FROM legui_private.shopper_coupons c LEFT JOIN legui_private.shoppers p ON p.id=c.shopper_id WHERE c.brand_id=? ORDER BY c.created_at DESC LIMIT 100)v`, scope.BrandID))
	}
	if r.Method == "PATCH" {
		id := r.PathValue("id")
		if !validID(id) {
			return Result{}, notFound()
		}
		in, e := decode[struct {
			Active bool `json:"active"`
		}](r)
		if e != nil {
			return Result{}, e
		}
		return h.mutate(r, p, scope, "marketing.coupon_status", in, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
			data, e := jsonQuery(tx, `UPDATE legui_private.shopper_coupons SET active=? WHERE id=? AND brand_id=? RETURNING jsonb_build_object('id',id,'active',active)::text payload`, in.Active, id, scope.BrandID)
			return data, 200, e
		})
	}
	in, e := decode[struct {
		Code         string  `json:"code"`
		Percent      int     `json:"percent"`
		Minimum      string  `json:"minimum"`
		MaxUses      int     `json:"max_uses"`
		Days         int     `json:"days"`
		ShopperID    *string `json:"shopper_id"`
		AllLocations bool    `json:"all_locations"`
	}](r)
	if e != nil {
		return Result{}, e
	}
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	if len(in.Code) < 3 || len(in.Code) > 32 || strings.IndexFunc(in.Code, func(c rune) bool { return !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') }) >= 0 || in.Percent < 1 || in.Percent > 50 || in.Days < 1 || in.Days > 90 || in.MaxUses < 1 || in.MaxUses > 100000 || !optionalID(in.ShopperID) {
		return Result{}, badInput("Revisá código, porcentaje, vigencia y usos")
	}
	if in.Minimum == "" {
		in.Minimum = "0"
	}
	if _, e := priceRat(in.Minimum); e != nil || len(in.Minimum) > 12 {
		return Result{}, badInput("Mínimo inválido")
	}
	return h.mutate(r, p, scope, "marketing.coupon_create", in, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		if in.ShopperID != nil {
			var known bool
			if e := tx.Raw(`SELECT EXISTS(SELECT 1 FROM legui_private.shopper_orders a JOIN public.orders o ON o.id=a.order_id JOIN public.tenants t ON t.id=o.tenant_id WHERE a.shopper_id=? AND t.brand_id=?)`, *in.ShopperID, scope.BrandID).Scan(&known).Error; e != nil {
				return nil, 0, e
			}
			if !known {
				return nil, 0, notFound()
			}
		}
		var local any = scope.TenantID
		if in.AllLocations {
			local = nil
		}
		data, e := jsonQuery(tx, `INSERT INTO legui_private.shopper_coupons(id,brand_id,tenant_id,shopper_id,code,percent,minimum,max_uses,expires_at) VALUES(?,?,?,?,?,?,?::numeric,?,?) RETURNING jsonb_build_object('id',id,'code',code)::text payload`, uuid.NewString(), scope.BrandID, local, in.ShopperID, in.Code, in.Percent, in.Minimum, in.MaxUses, time.Now().Add(time.Duration(in.Days)*24*time.Hour))
		return data, 201, e
	})
}

func (h *Handler) shopperContact(r *http.Request) (Result, error) {
	p, e := h.shopperProfile(r)
	if e != nil {
		return Result{}, e
	}
	in, e := decode[struct {
		Name    string `json:"name"`
		Phone   string `json:"phone"`
		Address string `json:"delivery_address"`
	}](r)
	if e != nil {
		return Result{}, e
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Address = strings.TrimSpace(in.Address)
	if len([]rune(in.Name)) < 2 || len([]rune(in.Name)) > 100 || !validShopperPhone(in.Phone) || len([]rune(in.Address)) > 500 {
		return Result{}, badInput("Revisá tu nombre, teléfono y dirección")
	}
	e = h.repo.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if _, e := shopperidentity.Lock(r.Context(), tx, r); e != nil {
			return e
		}
		return tx.Exec(`UPDATE legui_private.shoppers SET name=?,phone=?,default_address=? WHERE id=?`, in.Name, in.Phone, in.Address, p.ID).Error
	})
	if e != nil {
		return Result{}, e
	}
	return Result{Data: json.RawMessage(`{"ok":true}`)}, nil
}
func validShopperPhone(raw string) bool {
	if raw == "" {
		return true
	}
	if len(raw) > 30 {
		return false
	}
	digits := 0
	for _, c := range raw {
		if c >= '0' && c <= '9' {
			digits++
		} else if !strings.ContainsRune(" +()-", c) {
			return false
		}
	}
	return digits >= 6 && digits <= 20
}
