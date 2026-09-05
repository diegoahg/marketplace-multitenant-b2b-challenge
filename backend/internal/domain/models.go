package domain

import "time"

type Scope struct {
	TenantID   string `json:"tenantId" bson:"tenantId"`
	Country    string `json:"country" bson:"country"`
	CustomerID string `json:"customerId" bson:"customerId"`
}
type CountryConfig struct {
	TenantID string           `json:"tenantId" bson:"tenantId"`
	Country  string           `json:"country" bson:"country"`
	Currency string           `json:"currency" bson:"currency"`
	Scale    int              `json:"scale" bson:"scale"`
	Rounding Rounding         `json:"rounding" bson:"rounding"`
	TaxBPS   map[string]int64 `json:"taxBps" bson:"taxBps"`
	TaxGifts bool             `json:"taxGifts" bson:"taxGifts"`
}

func (c CountryConfig) Money(n int64) Money { return Money{n, c.Currency, c.Scale} }

type Product struct {
	TenantID    string `json:"tenantId" bson:"tenantId"`
	Country     string `json:"country" bson:"country"`
	SKU         string `json:"sku" bson:"sku"`
	Description string `json:"description" bson:"description"`
	Family      string `json:"family" bson:"family"`
	UnitPrice   Money  `json:"unitPrice" bson:"unitPrice"`
	TaxCategory string `json:"taxCategory" bson:"taxCategory"`
}
type Customer struct {
	Scope `bson:",inline"`
}
type CartItem struct {
	SKU      string `json:"sku" bson:"sku"`
	Quantity int64  `json:"quantity" bson:"quantity"`
}
type Cart struct {
	Scope         `bson:",inline"`
	PaymentMethod string     `json:"paymentMethod" bson:"paymentMethod"`
	Items         []CartItem `json:"items" bson:"items"`
}
type Tier struct {
	From int64 `json:"from" bson:"from"`
	To   int64 `json:"to" bson:"to"`
	BPS  int64 `json:"bps" bson:"bps"`
}
type Eligibility struct {
	SKU     string `json:"sku" bson:"sku"`
	Family  string `json:"family" bson:"family"`
	Minimum int64  `json:"minimum" bson:"minimum"`
}
type Promotion struct {
	ID               string      `json:"id" bson:"id"`
	TenantID         string      `json:"tenantId" bson:"tenantId"`
	Country          string      `json:"country" bson:"country"`
	Type             string      `json:"type" bson:"type"`
	Priority         int         `json:"priority" bson:"priority"`
	Eligibility      Eligibility `json:"eligibility" bson:"eligibility"`
	IncompatibleWith []string    `json:"incompatibleWith" bson:"incompatibleWith"`
	ValidFrom        time.Time   `json:"validFrom" bson:"validFrom"`
	ValidTo          time.Time   `json:"validTo" bson:"validTo"`
	Items            []CartItem  `json:"items,omitempty" bson:"items"`
	ComboPrice       Money       `json:"comboPrice" bson:"comboPrice"`
	Tiers            []Tier      `json:"tiers,omitempty" bson:"tiers"`
	GiftSKU          string      `json:"giftSku,omitempty" bson:"giftSku"`
	GiftQuantity     int64       `json:"giftQuantity,omitempty" bson:"giftQuantity"`
}
type Line struct {
	SKU           string `json:"sku" bson:"sku"`
	Quantity      int64  `json:"quantity" bson:"quantity"`
	ComboQuantity int64  `json:"comboQuantity" bson:"comboQuantity"`
	UnitPrice     Money  `json:"unitPrice" bson:"unitPrice"`
	Gross         Money  `json:"gross" bson:"gross"`
	Discount      Money  `json:"discount" bson:"discount"`
	TaxableBase   Money  `json:"taxableBase" bson:"taxableBase"`
	Tax           Money  `json:"tax" bson:"tax"`
}
type Adjustment struct {
	PromotionID    string `json:"promotionId" bson:"promotionId"`
	PromotionType  string `json:"promotionType" bson:"promotionType"`
	SKU            string `json:"sku" bson:"sku"`
	Quantity       int64  `json:"quantity" bson:"quantity"`
	DiscountAmount Money  `json:"discountAmount" bson:"discountAmount"`
}
type Gift struct {
	SKU            string `json:"sku" bson:"sku"`
	Quantity       int64  `json:"quantity" bson:"quantity"`
	PromotionID    string `json:"promotionId" bson:"promotionId"`
	ReferenceValue Money  `json:"referenceValue" bson:"referenceValue"`
	Tax            Money  `json:"tax" bson:"tax"`
}
type CreditEvaluation struct {
	Eligible  bool  `json:"eligible" bson:"eligible"`
	Available Money `json:"available" bson:"available"`
}
type Quote struct {
	Scope            `bson:",inline"`
	QuoteID          string           `json:"quoteId" bson:"quoteId"`
	Currency         string           `json:"currency" bson:"currency"`
	PaymentMethod    string           `json:"paymentMethod" bson:"paymentMethod"`
	Lines            []Line           `json:"lines" bson:"lines"`
	Adjustments      []Adjustment     `json:"adjustments" bson:"adjustments"`
	Gifts            []Gift           `json:"gifts" bson:"gifts"`
	GrossSubtotal    Money            `json:"grossSubtotal" bson:"grossSubtotal"`
	DiscountTotal    Money            `json:"discountTotal" bson:"discountTotal"`
	TaxableBase      Money            `json:"taxableBase" bson:"taxableBase"`
	TaxTotal         Money            `json:"taxTotal" bson:"taxTotal"`
	Total            Money            `json:"total" bson:"total"`
	CreditEvaluation CreditEvaluation `json:"creditEvaluation" bson:"creditEvaluation"`
	CreatedAt        time.Time        `json:"createdAt" bson:"createdAt"`
	ExpiresAt        time.Time        `json:"expiresAt" bson:"expiresAt"`
}
type Order struct {
	Scope       `bson:",inline"`
	OrderID     string    `json:"orderId" bson:"orderId"`
	OrderNumber string    `json:"orderNumber" bson:"orderNumber"`
	QuoteID     string    `json:"quoteId" bson:"quoteId"`
	Currency    string    `json:"currency" bson:"currency"`
	Total       Money     `json:"total" bson:"total"`
	Status      string    `json:"status" bson:"status"`
	CreatedAt   time.Time `json:"createdAt" bson:"createdAt"`
}
type CreditAccount struct {
	Scope     `bson:",inline"`
	Available Money `json:"available" bson:"available"`
}
type OrderRequest struct {
	QuoteID    string `json:"quoteId"`
	CustomerID string `json:"customerId"`
}
type OrderConfirmedEvent struct {
	EventID      string    `json:"eventId" bson:"eventId"`
	EventType    string    `json:"eventType" bson:"eventType"`
	EventVersion int       `json:"eventVersion" bson:"eventVersion"`
	OccurredAt   time.Time `json:"occurredAt" bson:"occurredAt"`
	TenantID     string    `json:"tenantId" bson:"tenantId"`
	Country      string    `json:"country" bson:"country"`
	Order        Order     `json:"order" bson:"order"`
}
type Idempotency struct {
	TenantID    string    `bson:"tenantId"`
	Key         string    `bson:"key"`
	RequestHash string    `bson:"requestHash"`
	OrderID     string    `bson:"orderId"`
	Response    Order     `bson:"response"`
	Status      int       `bson:"status"`
	CreatedAt   time.Time `bson:"createdAt"`
}
