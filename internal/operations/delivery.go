package operations

import (
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
	"net/http"
	"regexp"
	"strings"
)

var postalPattern = regexp.MustCompile(`^(?:[0-9]{4}|[A-Z][0-9]{4}[A-Z]{3})$`)

func normalizePostal(raw string) (string, error) {
	value := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(raw), " ", ""))
	if !postalPattern.MatchString(value) {
		return "", badInput("Ingresá un código postal de 4 dígitos o un CPA completo")
	}
	if len(value) == 8 {
		return value[1:5], nil
	}
	return value, nil
}
func deliverySettingsData(tx *gorm.DB, tenant string) (json.RawMessage, error) {
	return jsonQuery(tx, `SELECT jsonb_build_object('zones_enabled',coalesce((SELECT zones_enabled FROM legui_private.delivery_settings WHERE tenant_id=t.id),false))::text payload FROM public.tenants t WHERE t.id=?`, tenant)
}
func (h *Handler) deliverySettings(r *http.Request, p Principal) (Result, error) {
	if !p.Manager() {
		return Result{}, forbidden()
	}
	scope, e := h.tenantScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	if r.Method == "GET" {
		return resultJSON(deliverySettingsData(h.repo.DB.WithContext(r.Context()), scope.TenantID))
	}
	in, e := decode[struct {
		Enabled bool `json:"zones_enabled"`
	}](r)
	if e != nil {
		return Result{}, e
	}
	return h.mutate(r, p, scope, "delivery.settings", in, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		if e := tx.Exec(`SELECT id FROM public.tenants WHERE id=? FOR UPDATE`, scope.TenantID).Error; e != nil {
			return nil, 0, e
		}
		if in.Enabled {
			var n int64
			if e := tx.Raw(`SELECT count(*) FROM legui_private.delivery_zones z JOIN public.shipping_methods m ON m.id=z.shipping_method_id AND m.tenant_id=z.tenant_id AND m.is_active WHERE z.tenant_id=?`, scope.TenantID).Scan(&n).Error; e != nil {
				return nil, 0, e
			}
			if n == 0 {
				return nil, 0, conflict("Agregá una zona activa antes de habilitar la cobertura")
			}
		}
		if e := tx.Exec(`INSERT INTO legui_private.delivery_settings(tenant_id,zones_enabled) VALUES(?,?) ON CONFLICT(tenant_id) DO UPDATE SET zones_enabled=excluded.zones_enabled`, scope.TenantID, in.Enabled).Error; e != nil {
			return nil, 0, e
		}
		data, e := deliverySettingsData(tx, scope.TenantID)
		return data, 200, e
	})
}

type deliveryZoneInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Cost        Decimal  `json:"cost"`
	Minimum     Decimal  `json:"minimum"`
	PostalCodes []string `json:"postal_codes"`
	Active      bool     `json:"active"`
}

func (h *Handler) deliveryZones(r *http.Request, p Principal) (Result, error) {
	if !p.Manager() {
		return Result{}, forbidden()
	}
	scope, e := h.tenantScope(r.Context(), p, r)
	if e != nil {
		return Result{}, e
	}
	if r.Method == "GET" {
		return resultJSON(h.repo.JSON(r.Context(), `SELECT jsonb_build_object('items',coalesce(jsonb_agg(jsonb_build_object('id',z.id,'name',m.name,'description',z.description,'cost',m.cost::text,'minimum',z.minimum::text,'postal_codes',z.postal_codes,'active',m.is_active) ORDER BY m.name,z.id),'[]'::jsonb))::text payload FROM legui_private.delivery_zones z JOIN public.shipping_methods m ON m.id=z.shipping_method_id AND m.tenant_id=z.tenant_id WHERE z.tenant_id=?`, scope.TenantID))
	}
	in, e := decode[deliveryZoneInput](r)
	if e != nil {
		return Result{}, e
	}
	in.Name = strings.TrimSpace(in.Name)
	if len([]rune(in.Name)) < 2 || len([]rune(in.Name)) > 80 || len([]rune(in.Description)) > 300 || !in.Cost.Money() || !in.Minimum.Money() || len(in.PostalCodes) < 1 || len(in.PostalCodes) > 100 {
		return Result{}, badInput("Revisá el nombre, importes y códigos postales de la zona")
	}
	cost, _ := new(big.Rat).SetString(string(in.Cost))
	minimum, _ := new(big.Rat).SetString(string(in.Minimum))
	if cost.Cmp(big.NewRat(99999999, 1)) > 0 || minimum.Cmp(big.NewRat(999999999, 1)) > 0 {
		return Result{}, badInput("Importe fuera de rango")
	}
	seen := map[string]bool{}
	codes := []string{}
	for _, raw := range in.PostalCodes {
		code, e := normalizePostal(raw)
		if e != nil {
			return Result{}, e
		}
		if !seen[code] {
			codes = append(codes, code)
			seen[code] = true
		}
	}
	in.PostalCodes = codes
	id := r.PathValue("id")
	if r.Method == "PATCH" && !validID(id) {
		return Result{}, notFound()
	}
	return h.mutate(r, p, scope, "delivery.zone", in, func(tx *gorm.DB, key, digest string) (json.RawMessage, int, error) {
		if e := tx.Exec(`SELECT id FROM public.tenants WHERE id=? FOR UPDATE`, scope.TenantID).Error; e != nil {
			return nil, 0, e
		}
		method := uuid.NewString()
		if id != "" {
			q := tx.Raw(`SELECT shipping_method_id::text FROM legui_private.delivery_zones WHERE tenant_id=? AND id=? FOR UPDATE`, scope.TenantID, id).Scan(&method)
			if q.Error != nil {
				return nil, 0, q.Error
			}
			if q.RowsAffected != 1 {
				return nil, 0, notFound()
			}
		}
		encoded, _ := json.Marshal(codes)
		var overlap bool
		if e := tx.Raw(`SELECT EXISTS(SELECT 1 FROM legui_private.delivery_zones z JOIN public.shipping_methods m ON m.id=z.shipping_method_id AND m.tenant_id=z.tenant_id WHERE z.tenant_id=? AND z.id<>coalesce(nullif(?,'')::uuid,'00000000-0000-0000-0000-000000000000') AND m.is_active AND z.postal_codes && ARRAY(SELECT jsonb_array_elements_text(?::jsonb)))`, scope.TenantID, id, string(encoded)).Scan(&overlap).Error; e != nil {
			return nil, 0, e
		}
		if in.Active && overlap {
			return nil, 0, conflict("Un código postal ya pertenece a otra zona activa")
		}
		status := 200
		if id == "" {
			var n int64
			if e := tx.Raw(`SELECT count(*) FROM legui_private.delivery_zones WHERE tenant_id=?`, scope.TenantID).Scan(&n).Error; e != nil {
				return nil, 0, e
			}
			if n >= 50 {
				return nil, 0, conflict("El local admite hasta 50 zonas")
			}
			id = uuid.NewString()
			status = 201
			if e := tx.Exec(`INSERT INTO public.shipping_methods(id,tenant_id,name,description,typification,cost,is_active) VALUES(?,?,?,'Zona configurada','DELIVERY',?::numeric,?)`, method, scope.TenantID, in.Name, in.Cost, in.Active).Error; e != nil {
				return nil, 0, e
			}
			if e := tx.Exec(`INSERT INTO legui_private.delivery_zones(id,tenant_id,shipping_method_id,description,minimum,postal_codes) VALUES(?,?,?,?,?::numeric,ARRAY(SELECT jsonb_array_elements_text(?::jsonb)))`, id, scope.TenantID, method, in.Description, in.Minimum, string(encoded)).Error; e != nil {
				return nil, 0, e
			}
		} else {
			if e := tx.Exec(`UPDATE public.shipping_methods SET name=?,cost=?::numeric,is_active=? WHERE id=? AND tenant_id=?`, in.Name, in.Cost, in.Active, method, scope.TenantID).Error; e != nil {
				return nil, 0, e
			}
			if e := tx.Exec(`UPDATE legui_private.delivery_zones SET description=?,minimum=?::numeric,postal_codes=ARRAY(SELECT jsonb_array_elements_text(?::jsonb)) WHERE id=? AND tenant_id=?`, in.Description, in.Minimum, string(encoded), id, scope.TenantID).Error; e != nil {
				return nil, 0, e
			}
		}
		data, _ := json.Marshal(map[string]string{"id": id})
		return data, status, nil
	})
}

// Called under the tenant share lock obtained by buyerLocation.
func quoteDelivery(tx *gorm.DB, scope Scope, input BuyerBasket, subtotal *big.Rat, quote *BuyerQuote) error {
	if input.Fulfillment != "delivery" {
		return nil
	}
	var zoned bool
	if e := tx.Raw(`SELECT coalesce((SELECT zones_enabled FROM legui_private.delivery_settings WHERE tenant_id=?),false)`, scope.TenantID).Scan(&zoned).Error; e != nil {
		return e
	}
	if zoned && input.DeliveryZoneID == nil {
		return conflict("Elegí una zona e ingresá tu código postal para verificar la cobertura")
	}
	method := input.ShippingMethodID
	if input.DeliveryZoneID != nil {
		code, e := normalizePostal(input.PostalCode)
		if e != nil {
			return e
		}
		var row struct{ Method, Name, Minimum string }
		q := tx.Raw(`SELECT z.shipping_method_id::text method,m.name,z.minimum::text FROM legui_private.delivery_zones z JOIN public.shipping_methods m ON m.id=z.shipping_method_id AND m.tenant_id=z.tenant_id AND m.is_active WHERE z.id=? AND z.tenant_id=? AND ?=ANY(z.postal_codes) FOR SHARE OF z,m`, *input.DeliveryZoneID, scope.TenantID, code).Scan(&row)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected != 1 {
			return conflict("No tenemos cobertura para ese código postal en la zona elegida")
		}
		min, _ := priceRat(row.Minimum)
		if subtotal.Cmp(min) < 0 {
			return conflict("La compra mínima de esta zona es ARS " + row.Minimum + ", sin envío y después de descuentos")
		}
		method = &row.Method
		quote.DeliveryZoneID = input.DeliveryZoneID
		quote.DeliveryZoneName = row.Name
		quote.PostalCode = code
	}
	if method == nil {
		return badInput("Elegí una opción de envío")
	}
	var row struct{ Name, Cost string }
	q := tx.Raw(`SELECT name,cost::text FROM public.shipping_methods WHERE id=? AND tenant_id=? AND is_active FOR SHARE`, *method, scope.TenantID).Scan(&row)
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected != 1 {
		return conflict("El envío elegido no está disponible para este local")
	}
	price, e := priceRat(row.Cost)
	if e != nil {
		return e
	}
	quote.Shipping = price.FloatString(2)
	quote.ShippingName = row.Name
	quote.ShippingMethodID = method
	return nil
}
