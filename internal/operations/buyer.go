package operations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"leguiburger/internal/buyeraccess"
	"leguiburger/internal/orderstream"
	"leguiburger/internal/shopperidentity"
	"log"
	"math/big"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

type BuyerOptions struct {
	Stream                 *orderstream.Hub
	OnlinePaymentAvailable func(context.Context, string) (bool, error)
}
type BuyerBasket struct {
	DeliveryZoneID   *string            `json:"delivery_zone_id,omitempty"`
	PostalCode       string             `json:"postal_code,omitempty"`
	CouponCode       string             `json:"coupon_code,omitempty"`
	ShopperID        string             `json:"-"`
	CouponLock       bool               `json:"-"`
	Items            []OrderItemRequest `json:"items"`
	Fulfillment      string             `json:"fulfillment"`
	ShippingMethodID *string            `json:"shipping_method_id,omitempty"`
}
type BuyerOrderRequest struct {
	RememberContact bool `json:"remember_contact,omitempty"`
	BuyerBasket
	QuoteDigest   string `json:"quote_digest"`
	Name          string `json:"name"`
	Phone         string `json:"phone"`
	Email         string `json:"email,omitempty"`
	Address       string `json:"address,omitempty"`
	PaymentMethod string `json:"payment_method"`
}
type BuyerExtra struct {
	GroupName string `json:"group_name,omitempty"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Quantity  int    `json:"quantity"`
	Price     string `json:"price"`
}
type BuyerLine struct {
	ProductID string       `json:"product_id"`
	Name      string       `json:"name"`
	Quantity  int          `json:"quantity"`
	UnitPrice string       `json:"unit_price"`
	Total     string       `json:"total"`
	Extras    []BuyerExtra `json:"extras"`
	Note      string       `json:"note"`
}
type BuyerQuote struct {
	ShippingMethodID *string     `json:"-"`
	DeliveryZoneID   *string     `json:"delivery_zone_id,omitempty"`
	DeliveryZoneName string      `json:"delivery_zone_name,omitempty"`
	PostalCode       string      `json:"postal_code,omitempty"`
	Discount         string      `json:"discount"`
	CouponID         string      `json:"-"`
	CouponPercent    int         `json:"-"`
	Items            []BuyerLine `json:"items"`
	Subtotal         string      `json:"subtotal"`
	Shipping         string      `json:"shipping"`
	ShippingName     string      `json:"shipping_name"`
	Total            string      `json:"total"`
	Currency         string      `json:"currency"`
	Digest           string      `json:"digest"`
}

func buyerProblem(e error) error {
	if errors.Is(e, buyeraccess.ErrInvalid) {
		return notFound()
	}
	if errors.Is(e, buyeraccess.ErrRateLimited) {
		return &apiError{429, "RATE_LIMITED", "Detectamos demasiados intentos. Esperá 15 minutos antes de volver a intentar; tu pedido no se duplica."}
	}
	return e
}
func (h *Handler) buyerWrap(fn func(*http.Request) (Result, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			writeError(w, forbidden())
			return
		}
		if !h.buyerGate(w, r) {
			return
		}
		if h.security != nil && r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/public/menu/") && !strings.HasSuffix(r.URL.Path, "/orders") {
			if e := h.security.blockCheck(r.Context(), r); e != nil {
				writeError(w, e)
				return
			}
		}
		result, e := fn(r)
		if e != nil {
			h.observeBuyerFailure(r, e)
			if errors.Is(e, buyeraccess.ErrRateLimited) {
				w.Header().Set("Retry-After", "900")
				if h.security != nil {
					tenant := r.PathValue("id")
					if !validID(tenant) {
						tenant = ""
					}
					if err := h.security.observe(r.Context(), r, tenant, "burst", "rate_limit"); err != nil {
						log.Printf("abuse rate incident unavailable (%T)", err)
					}
				}
			}
			writeError(w, buyerProblem(e))
			return
		}
		if result.Status == 0 {
			result.Status = 200
		}
		if result.Replayed {
			w.Header().Set("Idempotent-Replayed", "true")
		}
		w.WriteHeader(result.Status)
		w.Write(result.Data)
	}
}
func validateBuyerBasket(input *BuyerBasket) error {
	input.CouponCode = strings.ToUpper(strings.TrimSpace(input.CouponCode))
	if len(input.CouponCode) > 32 {
		return badInput("Cupón inválido")
	}

	if input.Fulfillment == "" {
		input.Fulfillment = "pickup"
	}
	if !isOneOf(input.Fulfillment, "pickup", "delivery") || (input.Fulfillment == "delivery") != (input.ShippingMethodID != nil || input.DeliveryZoneID != nil) || (input.ShippingMethodID != nil && input.DeliveryZoneID != nil) || !optionalID(input.ShippingMethodID) || !optionalID(input.DeliveryZoneID) || len(input.PostalCode) > 16 || len(input.Items) < 1 || len(input.Items) > 30 {
		return badInput("Revisá los productos y la forma de entrega")
	}
	total := 0
	for _, item := range input.Items {
		total += item.Quantity
		if !validID(item.ProductID) || item.Quantity < 1 || item.Quantity > 20 || len(item.Extras) > 10 || len([]rune(item.Note)) > 300 {
			return badInput("Revisá las cantidades y preferencias del producto")
		}
		seen := map[string]bool{}
		for _, x := range item.Extras {
			if !validID(x.ExtraID) || x.Quantity < 1 || x.Quantity > 5 || seen[x.ExtraID] {
				return badInput("Extra inválido o repetido")
			}
			seen[x.ExtraID] = true
		}
	}
	if total > 100 {
		return badInput("El pedido admite hasta 100 productos")
	}
	return nil
}
func priceRat(raw string) (*big.Rat, error) {
	n, ok := new(big.Rat).SetString(raw)
	if !ok || n.Sign() < 0 {
		return nil, badInput("Precio inválido en la carta")
	}
	return n, nil
}
func buyerLocation(tx *gorm.DB, id string) (Scope, error) {
	var scope Scope
	q := tx.Raw(`SELECT t.id::text tenant_id,t.brand_id::text FROM public.tenants t JOIN public.brands b ON b.id=t.brand_id WHERE t.id=? AND t.active AND t.menu_published AND t.ordering_enabled AND b.suspended_at IS NULL AND b.closed_at IS NULL FOR SHARE OF t,b`, id).Scan(&scope)
	if q.Error != nil {
		return scope, q.Error
	}
	if q.RowsAffected != 1 {
		return scope, conflict("El local no está recibiendo pedidos online en este momento")
	}
	// Read availability only after obtaining the tenant lock, so a concurrent
	// pause committed while waiting is visible in this statement's new snapshot.
	var receiving bool
	if e := tx.Raw(`SELECT legui_private.tenant_receiving(?)`, id).Scan(&receiving).Error; e != nil {
		return scope, e
	}
	if !receiving {
		var full bool
		if e := tx.Raw(`SELECT legui_private.kitchen_full(?)`, id).Scan(&full).Error; e != nil {
			return scope, e
		}
		if full {
			return scope, conflict("La cocina está al máximo de su capacidad. Esperá unos minutos para hacer un nuevo pedido")
		}
		return scope, conflict("El local no está recibiendo pedidos online en este momento")
	}
	return scope, nil
}
func quoteBuyer(tx *gorm.DB, scope Scope, input BuyerBasket) (BuyerQuote, error) {
	quote := BuyerQuote{Currency: "ARS", Items: []BuyerLine{}, Shipping: "0.00"}
	var coupon struct {
		ID      string
		Percent int
		Minimum string
	}
	if input.CouponCode != "" {
		if input.ShopperID == "" {
			return quote, unauthorized()
		}
		lock := " FOR SHARE OF c"
		if input.CouponLock {
			lock = " FOR UPDATE OF c"
		}
		q := tx.Raw(`SELECT c.id::text,c.percent,c.minimum::text FROM legui_private.shopper_coupons c WHERE c.code=? AND c.brand_id=? AND (c.tenant_id IS NULL OR c.tenant_id=?) AND (c.shopper_id IS NULL OR c.shopper_id=?) AND c.active AND c.expires_at>clock_timestamp()`+lock, input.CouponCode, scope.BrandID, scope.TenantID, input.ShopperID).Scan(&coupon)
		if q.Error != nil {
			return quote, q.Error
		}
		if q.RowsAffected != 1 {
			return quote, conflict("El cupón no está disponible para tu cuenta y este local")
		}
		var used, own int64
		if e := tx.Raw(`SELECT count(*) FROM legui_private.shopper_coupon_uses WHERE coupon_id=?`, coupon.ID).Scan(&used).Error; e != nil {
			return quote, e
		}
		if e := tx.Raw(`SELECT count(*) FROM legui_private.shopper_coupon_uses WHERE coupon_id=? AND shopper_id=?`, coupon.ID, input.ShopperID).Scan(&own).Error; e != nil {
			return quote, e
		}
		var max int64
		if e := tx.Raw(`SELECT max_uses FROM legui_private.shopper_coupons WHERE id=?`, coupon.ID).Scan(&max).Error; e != nil {
			return quote, e
		}
		if own > 0 || used >= max {
			return quote, conflict("El cupón ya fue utilizado o agotó sus usos")
		}
		quote.CouponID = coupon.ID
		quote.CouponPercent = coupon.Percent
	}
	discounts := new(big.Rat)
	subtotal := new(big.Rat)
	usedProducts, usedExtras := map[string]int64{}, map[string]int64{}
	for _, item := range input.Items {
		var p struct {
			Name, Price  string
			CurrentStock int64
			TrackStock   bool
		}
		q := tx.Raw(`SELECT p.name,coalesce(tp.price_override,p.base_price)::text price,tp.current_stock,tp.track_stock FROM public.products p JOIN public.tenant_products tp ON tp.product_id=p.id WHERE tp.tenant_id=? AND p.brand_id=? AND p.id=? AND p.is_active AND tp.is_active AND tp.is_available FOR SHARE OF p,tp`, scope.TenantID, scope.BrandID, item.ProductID).Scan(&p)
		if q.Error != nil {
			return quote, q.Error
		}
		usedProducts[item.ProductID] += int64(item.Quantity)
		if q.RowsAffected != 1 || p.TrackStock && usedProducts[item.ProductID] > p.CurrentStock {
			return quote, conflict("Un producto ya no está disponible en esa cantidad. Revisá la carta")
		}
		optionPrices, e := resolveProductOptions(tx, scope.TenantID, item.ProductID, item.Extras)
		if e != nil {
			return quote, e
		}
		unit, e := priceRat(p.Price)
		if e != nil {
			return quote, e
		}
		if coupon.Percent > 0 {
			discount, _ := priceRat(new(big.Rat).Mul(unit, big.NewRat(int64(coupon.Percent), 100)).FloatString(2))
			unit.Sub(unit, discount)
			discounts.Add(discounts, new(big.Rat).Mul(discount, big.NewRat(int64(item.Quantity), 1)))
		}
		line := BuyerLine{ProductID: item.ProductID, Name: p.Name, Quantity: item.Quantity, Extras: []BuyerExtra{}, Note: strings.TrimSpace(item.Note)}
		for _, extra := range item.Extras {
			var x struct {
				Name, Price  string
				CurrentStock int64
				TrackStock   bool
			}
			q = tx.Raw(`SELECT name,current_price::text price,current_stock,track_stock FROM public.extras WHERE id=? AND tenant_id=? AND is_active FOR SHARE`, extra.ExtraID, scope.TenantID).Scan(&x)
			if q.Error != nil {
				return quote, q.Error
			}
			usedExtras[extra.ExtraID] += int64(extra.Quantity * item.Quantity)
			if q.RowsAffected != 1 || x.TrackStock && usedExtras[extra.ExtraID] > x.CurrentStock {
				return quote, conflict("Un extra ya no está disponible en esa cantidad")
			}
			groupName := ""
			if option, ok := optionPrices[extra.ExtraID]; ok {
				x.Price = option.Price
				groupName = option.GroupName
			}
			price, e := priceRat(x.Price)
			if e != nil {
				return quote, e
			}
			unit.Add(unit, new(big.Rat).Mul(price, big.NewRat(int64(extra.Quantity), 1)))
			line.Extras = append(line.Extras, BuyerExtra{ID: extra.ExtraID, Name: x.Name, Price: price.FloatString(2), Quantity: extra.Quantity, GroupName: groupName})
		}
		line.UnitPrice = unit.FloatString(2)
		amount := new(big.Rat).Mul(unit, big.NewRat(int64(item.Quantity), 1))
		line.Total = amount.FloatString(2)
		subtotal.Add(subtotal, amount)
		quote.Items = append(quote.Items, line)
	}
	if e := validateRecipeBasket(tx, scope.TenantID, input.Items); e != nil {
		return quote, e
	}
	if e := quoteDelivery(tx, scope, input, subtotal, &quote); e != nil {
		return quote, e
	}
	shipping, e := priceRat(quote.Shipping)
	if e != nil {
		return quote, e
	}
	if coupon.ID != "" && discounts.Sign() <= 0 {
		return quote, badInput("El cupón no genera descuento en estos productos")
	}
	quote.Discount = discounts.FloatString(2)
	quote.Subtotal = new(big.Rat).Add(subtotal, discounts).FloatString(2)
	if coupon.ID != "" {
		minimum, e := priceRat(coupon.Minimum)
		if e != nil {
			return quote, e
		}
		gross, _ := priceRat(quote.Subtotal)
		if gross.Cmp(minimum) < 0 {
			return quote, conflict("El carrito no alcanza el mínimo del cupón")
		}
	}
	quote.Total = new(big.Rat).Add(subtotal, shipping).FloatString(2)
	raw, e := canonicalJSON(struct {
		Tenant string
		Basket BuyerBasket
		Quote  BuyerQuote
	}{scope.TenantID, input, quote})
	if e != nil {
		return quote, e
	}
	sum := sha256.Sum256(raw)
	quote.Digest = hex.EncodeToString(sum[:])
	return quote, nil
}
func (h *Handler) buyerQuote(r *http.Request) (Result, error) {
	id, e := resourceID(r)
	if e != nil {
		return Result{}, notFound()
	}
	input, e := decode[BuyerBasket](r)
	if e != nil {
		return Result{}, e
	}
	if e = validateBuyerBasket(&input); e != nil {
		return Result{}, e
	}
	if e = buyeraccess.Rate(r.Context(), h.repo.DB, r, "quote:"+id, 240); e != nil {
		return Result{}, e
	}
	if profile, err := shopperidentity.Current(r.Context(), h.repo.DB, r); err == nil {
		input.ShopperID = profile.ID
	}
	var quote BuyerQuote
	e = h.repo.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		scope, e := buyerLocation(tx, id)
		if e != nil {
			return e
		}
		quote, e = quoteBuyer(tx, scope, input)
		return e
	})
	if e != nil {
		return Result{}, e
	}
	data, _ := json.Marshal(quote)
	return Result{Data: data}, nil
}
func (h *Handler) buyerReceiptData(r *http.Request, db *gorm.DB, grant buyeraccess.Grant) (json.RawMessage, error) {
	data, e := jsonQuery(db, `SELECT jsonb_build_object('id',o.id,'status',s.code,'created_at',o.created_at,'pending_deadline',o.pending_deadline,'ended_reason',o.ended_reason,'prep_min',o.prep_min,'prep_max',o.prep_max,'estimated_ready_at',o.estimated_ready_at,'currency',o.currency,'subtotal',o.subtotal::text,'discount',o.total_discounts::text,'total_due',o.total_due::text,'total_paid',o.total_paid::text,'shipping',o.shipping_cost::text,'fulfillment',o.fulfillment,'delivery_zone_name',o.delivery_zone_name,'delivery_postal_code',o.delivery_postal_code,'payment_method',o.buyer_payment_method,'payment',(SELECT jsonb_build_object('state',x.state,'amount',x.amount::text,'checkout_url',CASE WHEN x.state='created' AND x.expires_at>clock_timestamp() AND o.total_due>o.total_paid AND s.code NOT IN ('pending','canceled','rejected') THEN x.checkout_url ELSE NULL END) FROM legui_private.mp_checkouts x WHERE x.order_id=o.id AND x.tenant_id=o.tenant_id AND x.purpose='order' AND x.state IN ('prepared','submitting','created','review') AND (x.expires_at>clock_timestamp() OR x.state IN ('submitting','review')) ORDER BY x.created_at DESC LIMIT 1),'business',b.name,'location',coalesce(nullif(t.display_name,''),t.subdomain),'tenant_id',t.id,'items',coalesce((SELECT jsonb_agg(jsonb_build_object('name',d.product_name,'quantity',d.quantity,'unit_price',d.final_price_unit::text,'note',d.preparation_note,'extras',coalesce((SELECT jsonb_agg(jsonb_build_object('name',x.extra_name,'quantity',x.quantity,'price',x.extra_price_charged::text,'group_name',x.option_group_name)) FROM public.order_detail_extras x WHERE x.order_detail_id=d.id),'[]'::jsonb))) FROM public.order_details d WHERE d.order_id=o.id),'[]'::jsonb))::text payload FROM public.orders o JOIN public.order_statuses_lookup s ON s.id=o.order_status_id JOIN public.tenants t ON t.id=o.tenant_id JOIN public.brands b ON b.id=t.brand_id WHERE o.id=? AND o.tenant_id=?`, grant.OrderID, grant.TenantID)
	if e != nil {
		return nil, e
	}
	var out map[string]any
	json.Unmarshal(data, &out)
	online := false
	if h.buyer.OnlinePaymentAvailable != nil {
		online, _ = h.buyer.OnlinePaymentAvailable(r.Context(), grant.TenantID)
	}
	out["online_available"] = online
	if r.Header.Get("X-Shopper-Order") == "" {
		out["tracking_path"] = "/order#token=" + buyeraccess.TrackingToken(buyeraccess.Token(r))
	} else {
		out["account_payment_key"] = uuid.NewSHA1(uuid.NameSpaceOID, []byte("account-payment:"+grant.OrderID)).String()
	}
	return json.Marshal(out)
}
func (h *Handler) buyerReceipt(r *http.Request) (Result, error) {
	grant, e := buyeraccess.ResolveRequest(r.Context(), h.repo.DB, r, false)
	if e == nil && r.Header.Get("Authorization") == "" && r.Header.Get("X-Shopper-Order") == "" && grant.TenantID != r.Header.Get("X-Buyer-Tenant") {
		return Result{}, notFound()
	}
	if e != nil {
		return Result{}, e
	}
	return resultJSON(h.buyerReceiptData(r, h.repo.DB, grant))
}
func (h *Handler) buyerCreate(r *http.Request) (Result, error) {
	id, e := resourceID(r)
	if e != nil {
		return Result{}, notFound()
	}
	secret := buyeraccess.Token(r)
	if r.Header.Get("Authorization") == "" {
		s, ok := buyeraccess.CookieSession(r, id)
		if !ok || s.Secret != secret || s.Key != r.Header.Get("Idempotency-Key") {
			return Result{}, badInput("La confirmación no coincide con la sesión de compra")
		}
	}
	key := r.Header.Get("Idempotency-Key")
	if !buyeraccess.ValidToken(secret) || !validID(key) {
		return Result{}, badInput("Falta una confirmación de compra válida")
	}
	input, e := decode[BuyerOrderRequest](r)
	if e != nil {
		return Result{}, e
	}
	if e = validateBuyerBasket(&input.BuyerBasket); e != nil {
		return Result{}, e
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Address = strings.TrimSpace(input.Address)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if input.PaymentMethod == "" {
		input.PaymentMethod = "at_local"
	}
	digits := 0
	for _, c := range input.Phone {
		if c >= '0' && c <= '9' {
			digits++
		} else if !strings.ContainsRune(" +()-", c) {
			return Result{}, badInput("Ingresá un teléfono válido")
		}
	}
	if len([]rune(input.Name)) < 2 || len([]rune(input.Name)) > 100 || len(input.Phone) > 30 || digits < 6 || digits > 20 || len([]rune(input.Address)) > 500 || input.Fulfillment == "delivery" && len([]rune(input.Address)) < 8 || input.Fulfillment == "pickup" && input.Address != "" || !isOneOf(input.PaymentMethod, "at_local", "mercadopago") || len(input.QuoteDigest) != 64 {
		return Result{}, badInput("Revisá el nombre, teléfono, entrega y resumen del pedido")
	}
	if input.Email != "" {
		email, e := mail.ParseAddress(input.Email)
		if e != nil || email.Address != input.Email || len(input.Email) > 254 {
			return Result{}, badInput("Ingresá un correo válido")
		}
	}
	if profile, err := shopperidentity.Current(r.Context(), h.repo.DB, r); err == nil {
		input.ShopperID = profile.ID
	}
	if input.RememberContact && input.ShopperID == "" {
		return Result{}, unauthorized()
	}
	input.CouponLock = true
	encoded, _ := canonicalJSON(input)
	digest := buyeraccess.Digest(id + "\n" + string(encoded))
	requireClearance := false
	var priorCheck struct{ AccessDigest, RequestDigest string }
	priorQuery := h.repo.DB.WithContext(r.Context()).Raw(`SELECT access_digest,request_digest FROM legui_private.buyer_orders WHERE tenant_id=? AND creation_key=?`, id, key).Scan(&priorCheck)
	if priorQuery.Error != nil {
		return Result{}, priorQuery.Error
	}
	replay := priorQuery.RowsAffected == 1 && priorCheck.AccessDigest == buyeraccess.Digest(secret) && priorCheck.RequestDigest == digest
	if !replay {
		if h.security != nil {
			requireClearance, e = h.security.requireOrderCheck(r, id, secret, key)
			if e != nil {
				return Result{}, e
			}
		}
		if e = buyeraccess.Rate(r.Context(), h.repo.DB, r, "order:"+id, 120); e != nil {
			return Result{}, e
		}
		if e = buyeraccess.RateIdentity(r.Context(), h.repo.DB, "order-session:"+id, secret, 10); e != nil {
			return Result{}, e
		}

	}

	var result Result
	e = h.repo.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec(`SET LOCAL lock_timeout='5s'`).Error; e != nil {
			return e
		}
		if e := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "buyer:"+id+":"+key).Error; e != nil {
			return e
		}
		if input.ShopperID != "" {
			p, e := shopperidentity.Lock(r.Context(), tx, r)
			if e != nil || p.ID != input.ShopperID {
				return unauthorized()
			}
		}
		var prior struct{ OrderID, AccessDigest, RequestDigest string }
		q := tx.Raw(`SELECT order_id::text,access_digest,request_digest FROM legui_private.buyer_orders WHERE tenant_id=? AND creation_key=?`, id, key).Scan(&prior)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected == 1 {
			if prior.AccessDigest != buyeraccess.Digest(secret) {
				return notFound()
			}
			if prior.RequestDigest != digest {
				return conflict("La confirmación ya se usó con otros datos. Consultá tu pedido")
			}
			grant, e := buyeraccess.Resolve(r.Context(), tx, secret, true)
			if e != nil {
				return e
			}
			data, e := h.buyerReceiptData(r, tx, grant)
			result = Result{Data: data, Status: 201, Replayed: true}
			return e
		}
		scope, e := buyerLocation(tx, id)
		if e != nil {
			return e
		}
		quote, e := quoteBuyer(tx, scope, input.BuyerBasket)
		if e != nil {
			return e
		}
		if quote.Digest != input.QuoteDigest {
			return conflict("La carta o el precio cambiaron. Revisá el resumen actualizado antes de confirmar")
		}
		total, _ := priceRat(quote.Total)
		if total.Sign() <= 0 {
			return badInput("El pedido necesita un importe mayor a cero")
		}
		if input.PaymentMethod == "mercadopago" {
			if h.buyer.OnlinePaymentAvailable == nil {
				return conflict("El local todavía no habilitó el pago online")
			}
			enabled, e := h.buyer.OnlinePaymentAvailable(r.Context(), id)
			if e != nil {
				return e
			}
			if !enabled {
				return conflict("El pago online no está disponible. Elegí pagar al recibir")
			}
		}
		if requireClearance {
			if e := h.security.consume(tx, r, id, secret, key); e != nil {
				return e
			}
		}
		order, e := writeOrder(tx, scope, CreateOrderRequest{ShippingMethodID: quote.ShippingMethodID, ShippingAddress: input.Address, Channel: "web", Environment: "production", Items: input.Items}, "buyer:"+key)
		if e != nil {
			return e
		}
		orderID, e := idFromJSON(order)
		if e != nil {
			return e
		}
		if input.ShopperID != "" {
			if e = tx.Exec(`INSERT INTO legui_private.shopper_orders(order_id,shopper_id) VALUES(?,?)`, orderID, input.ShopperID).Error; e != nil {
				return e
			}
			if input.RememberContact {
				if e = tx.Exec(`UPDATE legui_private.shoppers SET phone=?,default_address=CASE WHEN ?='delivery' THEN ? ELSE default_address END WHERE id=?`, input.Phone, input.Fulfillment, input.Address, input.ShopperID).Error; e != nil {
					return e
				}
			}
		}
		if quote.CouponID != "" {
			if e = tx.Exec(`UPDATE public.order_details SET discount_unit=round(list_price_unit*?::numeric/100,2),final_price_unit=list_price_unit-round(list_price_unit*?::numeric/100,2) WHERE order_id=?`, quote.CouponPercent, quote.CouponPercent, orderID).Error; e != nil {
				return e
			}
			if e = tx.Exec(`UPDATE public.orders SET total_discounts=(SELECT sum(quantity*discount_unit) FROM public.order_details WHERE order_id=?) WHERE id=?`, orderID, orderID).Error; e != nil {
				return e
			}
			if e = tx.Exec(`INSERT INTO legui_private.shopper_coupon_uses(coupon_id,shopper_id,order_id,discount) VALUES(?,?,?,?::numeric)`, quote.CouponID, input.ShopperID, orderID, quote.Discount).Error; e != nil {
				return e
			}
		}
		if e = tx.Exec(`UPDATE public.orders SET buyer_name=?,buyer_phone=?,buyer_email=?,fulfillment=?,buyer_payment_method=?,delivery_zone_id=?,delivery_zone_name=?,delivery_postal_code=? WHERE id=? AND tenant_id=?`, input.Name, input.Phone, input.Email, input.Fulfillment, input.PaymentMethod, quote.DeliveryZoneID, quote.DeliveryZoneName, quote.PostalCode, orderID, id).Error; e != nil {
			return e
		}
		if e = tx.Exec(`INSERT INTO legui_private.buyer_orders(order_id,tenant_id,creation_key,access_digest,request_digest) VALUES(?,?,?,?,?)`, orderID, id, key, buyeraccess.Digest(secret), digest).Error; e != nil {
			return e
		}
		if e = tx.Exec(`INSERT INTO legui_private.order_links(digest,order_id,tenant_id) VALUES(?,?,?)`, capabilityDigest(buyeraccess.TrackingToken(secret)), orderID, id).Error; e != nil {
			return e
		}
		if e = tx.Exec(`INSERT INTO public.activity_events(brand_id,tenant_id,event_name,source,environment,facts) VALUES(?,?,'orders.buyer_create','server','production','{}')`, scope.BrandID, id).Error; e != nil {
			return e
		}
		if h.security != nil {
			localSecurity := *h.security
			localSecurity.db = tx
			if e := localSecurity.observe(r.Context(), r, id, "order", ""); e != nil {
				return e
			}
		}
		data, e := h.buyerReceiptData(r, tx, buyeraccess.Grant{OrderID: orderID, TenantID: id, BrandID: scope.BrandID})
		result = Result{Data: data, Status: 201}
		return e
	})
	return result, e
}
