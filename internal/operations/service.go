package operations

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"leguiburger/internal/auth"
	"net/http"
	"regexp"
	"strings"
)

var operationKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,160}$`)

func (h *Handler) mutate(r *http.Request, p Principal, scope Scope, name string, input any, fn func(*gorm.DB, string, string) (json.RawMessage, int, error)) (Result, error) {
	key := r.Header.Get("Idempotency-Key")
	if !operationKeyPattern.MatchString(key) {
		return Result{}, badInput("Se requiere Idempotency-Key de 1 a 160 caracteres")
	}
	encoded, err := canonicalJSON(input)
	if err != nil {
		return Result{}, err
	}
	sum := sha256.Sum256(append([]byte(r.URL.Path+"\n"+scope.BrandID+"\n"+scope.TenantID+"\n"), encoded...))
	digest := hex.EncodeToString(sum[:])
	var result Result
	err = h.repo.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SET LOCAL lock_timeout='5s'`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`SET LOCAL statement_timeout='15s'`).Error; err != nil {
			return err
		}
		if err := auth.LockSession(r.Context(), tx); err != nil {
			return unauthorized()
		}
		if err := lockPrincipal(tx, p, scope); err != nil {
			return err
		}
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, p.Kind+":"+p.ID+":"+name+":"+key).Error; err != nil {
			return err
		}
		actorColumn := "actor_user_id"
		actorUser, actorEmployee := any(p.ID), any(nil)
		if p.Kind == "employee" {
			actorColumn = "actor_employee_id"
			actorUser, actorEmployee = nil, p.ID
		}
		var existing struct {
			Digest  string
			Payload string
			Status  int
		}
		found := tx.Raw(`SELECT request_digest AS digest,response::text AS payload,http_status AS status FROM legui_private.operation_receipts WHERE scope=? AND `+actorColumn+`=? AND operation_key=?`, name, p.ID, key).Scan(&existing)
		if found.Error != nil {
			return found.Error
		}
		if found.RowsAffected > 0 {
			if existing.Digest != digest {
				return conflict("La clave de operación ya fue usada con otros datos")
			}
			result = Result{Data: json.RawMessage(existing.Payload), Status: existing.Status, Replayed: true}
			return nil
		}
		userSetting, employeeSetting := "", ""
		if p.Kind == "user" {
			userSetting = p.ID
		} else {
			employeeSetting = p.ID
		}
		if err := tx.Exec(`SELECT set_config('leguiburger.actor_user_id',?,true),set_config('leguiburger.actor_employee_id',?,true),set_config('leguiburger.operation_key',?,true),set_config('leguiburger.request_digest',?,true)`, userSetting, employeeSetting, key, digest).Error; err != nil {
			return err
		}
		if err := tx.Exec(`SELECT set_config('leguiburger.expected_stock_version','',true),set_config('leguiburger.inventory_reason','',true),set_config('leguiburger.source_order_id','',true),set_config('leguiburger.source_production_report_id','',true)`).Error; err != nil {
			return err
		}
		data, status, err := fn(tx, key, digest)
		if err != nil {
			return err
		}
		if !json.Valid(data) {
			return fmt.Errorf("invalid operation response")
		}
		var brand, tenant any
		if scope.BrandID != "" {
			brand = scope.BrandID
		}
		if scope.TenantID != "" {
			tenant = scope.TenantID
		}
		if err = tx.Exec(`INSERT INTO legui_private.operation_receipts(scope,operation_key,request_digest,actor_user_id,actor_employee_id,brand_id,tenant_id,http_status,response)
   VALUES(?,?,?,?,?,?,?,?,?::jsonb)`, name, key, digest, actorUser, actorEmployee, brand, tenant, status, string(data)).Error; err != nil {
			return err
		}
		if p.Platform() {
			if err = tx.Exec(`INSERT INTO legui_private.platform_admin_events(actor_user_id,event_name,target_brand_id,operation_key,request_digest,facts) VALUES(?,?,?,?,?,'{}')`, p.ID, name, brand, name+":"+key, digest).Error; err != nil {
				return err
			}
		}
		if scope.BrandID != "" && !strings.HasPrefix(name, "platform.") {
			var contextFields struct {
				Environment        string `json:"environment"`
				PaymentEnvironment string `json:"payment_environment"`
			}
			_ = json.Unmarshal(data, &contextFields)
			environment := "production"
			if contextFields.Environment == "sandbox" || contextFields.PaymentEnvironment == "sandbox" {
				environment = "sandbox"
			}
			if err = tx.Exec(`INSERT INTO public.activity_events(brand_id,tenant_id,event_name,source,environment,actor_user_id,actor_employee_id,facts) VALUES(?,?,?,'server',?,?,?,'{}')`, scope.BrandID, tenant, name, environment, actorUser, actorEmployee).Error; err != nil {
				return err
			}
		}
		result = Result{Data: data, Status: status}
		return nil
	})
	return result, err
}
func idFromJSON(data json.RawMessage) (string, error) {
	var value struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return "", err
	}
	if value.ID == "" {
		return "", fmt.Errorf("missing resource ID")
	}
	return value.ID, nil
}
func isOneOf(value string, values ...string) bool {
	for _, allowed := range values {
		if value == allowed {
			return true
		}
	}
	return false
}
func validMoney(value Decimal, positive bool) bool {
	return value.Money() && (!positive || value.Positive())
}
