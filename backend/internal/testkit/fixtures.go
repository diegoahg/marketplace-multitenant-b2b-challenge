package testkit

import (
	d "marketplace/internal/domain"
	"time"
)

func Now() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) }
func Scope() d.Scope {
	return d.Scope{TenantID: "tenant-demo", Country: "PE", CustomerID: "CUSTOMER-001"}
}
func Config() d.CountryConfig {
	return d.CountryConfig{TenantID: "tenant-demo", Country: "PE", Currency: "PEN", Scale: 2, Rounding: d.HalfUp, TaxBPS: map[string]int64{"standard": 1800}}
}
func Products() []d.Product {
	cfg := Config()
	return []d.Product{{TenantID: cfg.TenantID, Country: cfg.Country, SKU: "A", Description: "Bebida A", Family: "drinks", UnitPrice: cfg.Money(100), TaxCategory: "standard"}, {TenantID: cfg.TenantID, Country: cfg.Country, SKU: "B", Description: "Bebida B", Family: "drinks", UnitPrice: cfg.Money(200), TaxCategory: "standard"}, {TenantID: cfg.TenantID, Country: cfg.Country, SKU: "C", Description: "Obsequio", Family: "gifts", UnitPrice: cfg.Money(50), TaxCategory: "standard"}}
}
func Rule(id, kind string) d.Promotion {
	return d.Promotion{ID: id, Type: kind, TenantID: "tenant-demo", Country: "PE", ValidFrom: Now().Add(-time.Hour), ValidTo: Now().Add(time.Hour)}
}
func Scale() d.Promotion {
	p := Rule("scale", "SCALE")
	p.Eligibility.SKU = "A"
	p.Tiers = []d.Tier{{From: 1, To: 5, BPS: 0}, {From: 6, To: 10, BPS: 500}, {From: 11, To: 0, BPS: 1000}}
	return p
}
func Combo() d.Promotion {
	p := Rule("combo", "COMBO")
	p.Items = []d.CartItem{{SKU: "A", Quantity: 2}, {SKU: "B", Quantity: 1}}
	p.ComboPrice = Config().Money(300)
	return p
}
func Gift() d.Promotion {
	p := Rule("gift", "GIFT")
	p.Eligibility = d.Eligibility{SKU: "A", Minimum: 6}
	p.GiftSKU = "C"
	p.GiftQuantity = 1
	return p
}
func Cart(items ...d.CartItem) d.Cart {
	return d.Cart{Scope: Scope(), PaymentMethod: "CREDIT", Items: items}
}
