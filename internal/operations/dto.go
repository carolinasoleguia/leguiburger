package operations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// Decimal preserves decimal input exactly; money is calculated by PostgreSQL numeric, never float64.
type Decimal string

var decimalPattern = regexp.MustCompile(`^-?[0-9]{1,14}(\.[0-9]{1,4})?$`)

func (d *Decimal) UnmarshalJSON(value []byte) error {
	var text string
	if len(value) > 0 && value[0] == '"' {
		if err := json.Unmarshal(value, &text); err != nil {
			return err
		}
	} else {
		text = string(value)
	}
	if !decimalPattern.MatchString(text) {
		return fmt.Errorf("importe o cantidad decimal inválidos")
	}
	*d = Decimal(text)
	return nil
}
func (d Decimal) Money() bool { return moneyPattern.MatchString(string(d)) }

var moneyPattern = regexp.MustCompile(`^[0-9]{1,14}(\.[0-9]{1,2})?$`)

func (d Decimal) Positive() bool {
	for _, c := range d {
		if c >= '1' && c <= '9' {
			return d[0] != '-'
		}
	}
	return false
}
func (d Decimal) MarshalJSON() ([]byte, error) {
	if !decimalPattern.MatchString(string(d)) {
		return nil, fmt.Errorf("invalid decimal")
	}
	return []byte(d), nil
}

type CreateOrderRequest struct {
	CustomerID       *string            `json:"customer_id,omitempty"`
	ShippingMethodID *string            `json:"shipping_method_id,omitempty"`
	ShippingAddress  string             `json:"shipping_address,omitempty"`
	Channel          string             `json:"channel"`
	Environment      string             `json:"environment"`
	Items            []OrderItemRequest `json:"items"`
}
type OrderItemRequest struct {
	Note      string              `json:"note,omitempty"`
	ProductID string              `json:"product_id"`
	Quantity  int                 `json:"quantity"`
	Extras    []OrderExtraRequest `json:"extras,omitempty"`
}
type OrderExtraRequest struct {
	ExtraID  string `json:"extra_id"`
	Quantity int    `json:"quantity"`
}
type ChangeOrderStatusRequest struct {
	Status          string `json:"status"`
	Reason          string `json:"reason,omitempty"`
	ExpectedVersion int64  `json:"expected_version"`
}
type PaymentRequest struct {
	Kind              string     `json:"kind"`
	Amount            Decimal    `json:"amount"`
	Method            string     `json:"method"`
	OriginalPaymentID *string    `json:"original_payment_id,omitempty"`
	SettledAt         *time.Time `json:"settled_at,omitempty"`
}
type StockMovementRequest struct {
	ItemKind        string  `json:"item_kind"`
	ItemID          string  `json:"item_id"`
	QuantityDelta   Decimal `json:"quantity_delta"`
	ExpectedVersion *int64  `json:"expected_version"`
	Reason          string  `json:"reason"`
}
type ExpenseRequest struct {
	Category    string    `json:"category"`
	Amount      Decimal   `json:"amount"`
	Currency    string    `json:"currency"`
	IncurredAt  time.Time `json:"incurred_at"`
	Description string    `json:"description,omitempty"`
}
type InventorySettingsRequest struct {
	MinimumStock *Decimal `json:"minimum_stock,omitempty"`
	UnitCost     *Decimal `json:"unit_cost,omitempty"`
}
type VoidRequest struct {
	Reason string `json:"reason"`
}
type CreatePlanRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
type CreatePriceRequest struct {
	Currency             string  `json:"currency"`
	Amount               Decimal `json:"amount"`
	IntervalMonths       int     `json:"interval_months"`
	BillingUnit          string  `json:"billing_unit"`
	TrialDays            int     `json:"trial_days"`
	MaximumLocations     *int    `json:"maximum_locations,omitempty"`
	MaximumCollaborators *int    `json:"maximum_collaborators,omitempty"`
	MembershipTier       string  `json:"membership_tier,omitempty"`
	ExtraLocationAmount  Decimal `json:"extra_location_amount,omitempty"`
	Published            *bool   `json:"published,omitempty"`
}
type CreateBillingAccountRequest struct {
	BrandID      string  `json:"brand_id"`
	TenantID     *string `json:"tenant_id,omitempty"`
	Currency     string  `json:"currency"`
	ContactEmail string  `json:"contact_email,omitempty"`
}
type CreateSubscriptionRequest struct {
	AccountID          string    `json:"account_id"`
	PlanPriceID        string    `json:"plan_price_id"`
	Environment        string    `json:"environment"`
	Quantity           int       `json:"quantity"`
	PurchasedLocations int       `json:"purchased_locations,omitempty"`
	Status             string    `json:"status"`
	PeriodStart        time.Time `json:"period_start"`
	PeriodEnd          time.Time `json:"period_end"`
}
type UpdateSubscriptionRequest struct {
	ExpectedVersion   int64      `json:"expected_version"`
	Status            string     `json:"status,omitempty"`
	CancelAtPeriodEnd *bool      `json:"cancel_at_period_end,omitempty"`
	PeriodStart       *time.Time `json:"period_start,omitempty"`
	PeriodEnd         *time.Time `json:"period_end,omitempty"`
}
type CreateInvoiceRequest struct {
	SubscriptionID string    `json:"subscription_id"`
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
	DueAt          time.Time `json:"due_at"`
	Description    string    `json:"description,omitempty"`
}
type CreditNoteRequest struct {
	Amount Decimal `json:"amount"`
	Reason string  `json:"reason"`
}
type BusinessSettingsRequest struct {
	Classification    string     `json:"data_classification"`
	AcquisitionSource string     `json:"acquisition_source,omitempty"`
	ActivatedAt       *time.Time `json:"activated_at,omitempty"`
}
type Result struct {
	Data     json.RawMessage
	Status   int
	Replayed bool
}
type Period struct {
	From     time.Time
	Until    time.Time
	Timezone string
}

func canonicalJSON(value any) ([]byte, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return bytes.TrimSpace(b), nil
}
