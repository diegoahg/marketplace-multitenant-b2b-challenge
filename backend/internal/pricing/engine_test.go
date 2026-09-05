package pricing_test

import (
	"encoding/json"
	"fmt"
	d "marketplace/internal/domain"
	"marketplace/internal/pricing"
	"marketplace/internal/testkit"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type summary struct {
	Gross      int64 `json:"gross"`
	Discount   int64 `json:"discount"`
	Base       int64 `json:"base"`
	Tax        int64 `json:"tax"`
	Total      int64 `json:"total"`
	ComboUnits int64 `json:"comboUnits"`
	ScaleUnits int64 `json:"scaleUnits"`
	GiftUnits  int64 `json:"giftUnits"`
	Eligible   bool  `json:"eligible"`
}

func TestGoldenPricing(t *testing.T) {
	cases := []struct {
		name   string
		items  []d.CartItem
		rules  []d.Promotion
		credit int64
	}{
		{"GT01", []d.CartItem{{SKU: "A", Quantity: 12}}, []d.Promotion{testkit.Scale()}, 100000},
		{"GT02", []d.CartItem{{SKU: "A", Quantity: 8}, {SKU: "B", Quantity: 1}}, []d.Promotion{testkit.Combo(), testkit.Scale()}, 100000},
		{"GT03", []d.CartItem{{SKU: "A", Quantity: 6}}, []d.Promotion{testkit.Gift()}, 100000},
		{"GT04", []d.CartItem{{SKU: "A", Quantity: 2}, {SKU: "B", Quantity: 1}}, []d.Promotion{testkit.Combo()}, 100000},
		{"GT05", []d.CartItem{{SKU: "A", Quantity: 2}}, nil, 235},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q, e := pricing.Calculate(testkit.Cart(tc.items...), testkit.Config(), testkit.Products(), tc.rules, testkit.Config().Money(tc.credit), testkit.Now())
			if e != nil {
				t.Fatal(e)
			}
			got := summary{Gross: q.GrossSubtotal.Amount, Discount: q.DiscountTotal.Amount, Base: q.TaxableBase.Amount, Tax: q.TaxTotal.Amount, Total: q.Total.Amount, Eligible: q.CreditEvaluation.Eligible}
			for _, line := range q.Lines {
				got.ComboUnits += line.ComboQuantity
			}
			for _, a := range q.Adjustments {
				if a.PromotionType == "SCALE" {
					got.ScaleUnits += a.Quantity
				}
				if a.PromotionID == "" {
					t.Fatal("missing provenance")
				}
			}
			for _, g := range q.Gifts {
				got.GiftUnits += g.Quantity
				if g.SKU != "C" {
					t.Fatal("gift SKU missing")
				}
			}
			bytes, e := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", tc.name+".json"))
			if e != nil {
				t.Fatal(e)
			}
			var want summary
			if e = json.Unmarshal(bytes, &want); e != nil {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %+v\nwant %+v", got, want)
			}
		})
	}
}
func TestEdges(t *testing.T) {
	for _, tc := range []struct {
		name      string
		items     []d.CartItem
		wantError bool
		rules     []d.Promotion
		discount  int64
	}{
		{name: "empty", wantError: true}, {name: "zero", items: []d.CartItem{{SKU: "A", Quantity: 0}}, wantError: true}, {name: "negative", items: []d.CartItem{{SKU: "A", Quantity: -1}}, wantError: true}, {name: "unknown", items: []d.CartItem{{SKU: "unknown", Quantity: 1}}, wantError: true},
		{name: "duplicate", items: []d.CartItem{{SKU: "A", Quantity: 1}, {SKU: "A", Quantity: 1}}, wantError: true},
		{name: "incomplete combo", items: []d.CartItem{{SKU: "A", Quantity: 2}}, rules: []d.Promotion{testkit.Combo()}},
		{name: "scale boundary", items: []d.CartItem{{SKU: "A", Quantity: 5}}, rules: []d.Promotion{testkit.Scale()}},
		{name: "scale next unit", items: []d.CartItem{{SKU: "A", Quantity: 6}}, rules: []d.Promotion{testkit.Scale()}, discount: 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, e := pricing.Calculate(testkit.Cart(tc.items...), testkit.Config(), testkit.Products(), tc.rules, testkit.Config().Money(100000), testkit.Now())
			if (e != nil) != tc.wantError {
				t.Fatalf("error=%v", e)
			}
			if e == nil && q.DiscountTotal.Amount != tc.discount {
				t.Fatal(q.DiscountTotal)
			}
		})
	}
}
func TestExpiredAndCandidates(t *testing.T) {
	p := testkit.Scale()
	p.ValidTo = testkit.Now()
	q, e := pricing.Calculate(testkit.Cart(d.CartItem{SKU: "A", Quantity: 12}), testkit.Config(), testkit.Products(), []d.Promotion{p}, testkit.Config().Money(100000), testkit.Now())
	if e != nil || q.DiscountTotal.Amount != 0 {
		t.Fatal(q, e)
	}
	p = testkit.Scale()
	other := p
	other.ID = "second"
	other.Priority = 2
	q, e = pricing.Calculate(testkit.Cart(d.CartItem{SKU: "A", Quantity: 12}), testkit.Config(), testkit.Products(), []d.Promotion{other, p}, testkit.Config().Money(100000), testkit.Now())
	if e != nil || q.DiscountTotal.Amount != 45 {
		t.Fatal(q, e)
	}
}
func TestCreditBoundaries(t *testing.T) {
	for _, amount := range []int64{235, 236} {
		q, e := pricing.Calculate(testkit.Cart(d.CartItem{SKU: "A", Quantity: 2}), testkit.Config(), testkit.Products(), nil, testkit.Config().Money(amount), testkit.Now())
		if e != nil || q.CreditEvaluation.Eligible != (amount == 236) {
			t.Fatal(q, e)
		}
	}
}
func TestGT10MultiCountry(t *testing.T) {
	for _, tc := range []struct {
		country, currency string
		scale             int
		rate, total       int64
	}{{"PE", "PEN", 2, 1800, 118}, {"CL", "CLP", 0, 1900, 119}} {
		cfg := testkit.Config()
		cfg.Country = tc.country
		cfg.Currency = tc.currency
		cfg.Scale = tc.scale
		cfg.TaxBPS = map[string]int64{"standard": tc.rate}
		products := testkit.Products()
		for i := range products {
			products[i].Country = tc.country
			products[i].UnitPrice = cfg.Money(products[i].UnitPrice.Amount)
		}
		cart := testkit.Cart(d.CartItem{SKU: "A", Quantity: 1})
		cart.Country = tc.country
		q, e := pricing.Calculate(cart, cfg, products, nil, cfg.Money(100000), testkit.Now())
		if e != nil || q.Total.Amount != tc.total || q.Currency != tc.currency {
			t.Fatal(q, e)
		}
	}
}
func TestGiftTaxAndCompatibility(t *testing.T) {
	cfg := testkit.Config()
	cfg.TaxGifts = true
	p := testkit.Gift()
	q, e := pricing.Calculate(testkit.Cart(d.CartItem{SKU: "A", Quantity: 6}), cfg, testkit.Products(), []d.Promotion{p}, cfg.Money(10000), testkit.Now())
	if e != nil || q.TaxTotal.Amount != 117 {
		t.Fatal(q, e)
	}
	scale := testkit.Scale()
	scale.IncompatibleWith = []string{"gift"}
	q, e = pricing.Calculate(testkit.Cart(d.CartItem{SKU: "A", Quantity: 6}), cfg, testkit.Products(), []d.Promotion{scale, p}, cfg.Money(10000), testkit.Now())
	if e != nil || len(q.Gifts) != 0 {
		t.Fatal(q, e)
	}
}
func BenchmarkPricing100Lines20Promotions(b *testing.B) {
	cfg := testkit.Config()
	products := []d.Product{}
	items := []d.CartItem{}
	promos := []d.Promotion{}
	for i := 0; i < 100; i++ {
		p := testkit.Products()[0]
		p.SKU = fmt.Sprint("SKU-", i)
		products = append(products, p)
		items = append(items, d.CartItem{SKU: p.SKU, Quantity: 12})
	}
	for i := 0; i < 20; i++ {
		p := testkit.Scale()
		p.ID = fmt.Sprint(i)
		p.Eligibility.SKU = products[i].SKU
		promos = append(promos, p)
	}
	cart := testkit.Cart(items...)
	now := testkit.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, e := pricing.Calculate(cart, cfg, products, promos, cfg.Money(10000000), now); e != nil {
			b.Fatal(e)
		}
	}
}
func TestQuoteDeterminism(t *testing.T) {
	cart := testkit.Cart(d.CartItem{SKU: "A", Quantity: 12})
	now := testkit.Now().Truncate(time.Second)
	a, e := pricing.Calculate(cart, testkit.Config(), testkit.Products(), []d.Promotion{testkit.Scale()}, testkit.Config().Money(100000), now)
	if e != nil {
		t.Fatal(e)
	}
	b, e := pricing.Calculate(cart, testkit.Config(), testkit.Products(), []d.Promotion{testkit.Scale()}, testkit.Config().Money(100000), now)
	if e != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("non deterministic quote")
	}
}
