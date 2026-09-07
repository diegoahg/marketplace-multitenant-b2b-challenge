package pricing_test

import (
	"encoding/json"
	d "marketplace/internal/domain"
	"marketplace/internal/pricing"
	"os"
	"testing"
	"time"
)

func TestExpandedSeedPromotionsInEveryCountry(t *testing.T) {
	var seed struct {
		Countries  []d.CountryConfig
		Products   []d.Product
		Promotions []d.Promotion
	}
	raw, err := os.ReadFile("../../testdata/seed.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &seed); err != nil {
		t.Fatal(err)
	}
	scenarios := []struct {
		id    string
		items []d.CartItem
	}{
		{"COMBO-002", []d.CartItem{{SKU: "SKU-003", Quantity: 2}, {SKU: "SKU-005", Quantity: 1}}},
		{"COMBO-003", []d.CartItem{{SKU: "SKU-007", Quantity: 2}, {SKU: "SKU-011", Quantity: 1}}},
		{"GIFT-002", []d.CartItem{{SKU: "SKU-005", Quantity: 6}}},
		{"GIFT-003", []d.CartItem{{SKU: "SKU-009", Quantity: 12}}},
		{"SCALE-002", []d.CartItem{{SKU: "SKU-003", Quantity: 12}, {SKU: "SKU-004", Quantity: 12}}},
		{"SCALE-003", []d.CartItem{{SKU: "SKU-005", Quantity: 24}}},
		{"SCALE-004", []d.CartItem{{SKU: "SKU-007", Quantity: 24}}},
		{"SCALE-005", []d.CartItem{{SKU: "SKU-009", Quantity: 24}}},
		{"SCALE-006", []d.CartItem{{SKU: "SKU-011", Quantity: 24}}},
	}
	for _, cfg := range seed.Countries {
		var products []d.Product
		var promos []d.Promotion
		for _, p := range seed.Products {
			if p.Country == cfg.Country {
				products = append(products, p)
			}
		}
		for _, p := range seed.Promotions {
			if p.Country == cfg.Country {
				promos = append(promos, p)
			}
		}
		if len(products) != 15 || len(promos) != 12 {
			t.Fatalf("%s: unexpected catalog size %d/%d", cfg.Country, len(products), len(promos))
		}
		for _, scenario := range scenarios {
			t.Run(cfg.Country+"/"+scenario.id, func(t *testing.T) {
				cart := d.Cart{Scope: d.Scope{TenantID: cfg.TenantID, Country: cfg.Country, CustomerID: "CUSTOMER-001"}, PaymentMethod: "CASH", Items: scenario.items}
				q, err := pricing.Calculate(cart, cfg, products, promos, cfg.Money(100000000), time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC))
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, a := range q.Adjustments {
					if a.PromotionID == scenario.id {
						found = true
					}
				}
				for _, g := range q.Gifts {
					if g.PromotionID == scenario.id {
						found = true
					}
				}
				if !found {
					t.Fatalf("promotion %s not applied", scenario.id)
				}
			})
		}
	}
}
