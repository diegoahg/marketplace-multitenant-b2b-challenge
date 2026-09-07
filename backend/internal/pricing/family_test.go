package pricing_test

import (
	d "marketplace/internal/domain"
	"marketplace/internal/pricing"
	"marketplace/internal/testkit"
	"reflect"
	"testing"
)

func TestFamilyScaleAggregatesAndIsOrderIndependent(t *testing.T) {
	for _, price := range []int64{100, 200} {
		p := testkit.Scale()
		p.Eligibility = d.Eligibility{Family: "drinks", Minimum: 6}
		products := testkit.Products()
		products[1].UnitPrice.Amount = price
		var previous d.Quote
		for i, items := range [][]d.CartItem{{{SKU: "A", Quantity: 3}, {SKU: "B", Quantity: 3}}, {{SKU: "B", Quantity: 3}, {SKU: "A", Quantity: 3}}} {
			q, err := pricing.Calculate(testkit.Cart(items...), testkit.Config(), products, []d.Promotion{p}, testkit.Config().Money(100000), testkit.Now())
			if err != nil {
				t.Fatal(err)
			}
			if q.DiscountTotal.Amount != price/20 || len(q.Adjustments) != 1 || q.Adjustments[0].SKU != "B" || q.Adjustments[0].Quantity != 1 {
				t.Fatalf("family sixth unit missing: %+v", q)
			}
			if i > 0 && (q.Total != previous.Total || !reflect.DeepEqual(q.Adjustments, previous.Adjustments)) {
				t.Fatal("cart order changed family pricing")
			}
			previous = q
		}
	}
}
func TestFamilyScaleUsesComboRemainderAndReservesWholeGroup(t *testing.T) {
	p := testkit.Scale()
	p.Eligibility = d.Eligibility{Family: "drinks", Minimum: 6}
	combo := testkit.Combo()
	combo.Items = []d.CartItem{{SKU: "A", Quantity: 2}, {SKU: "C", Quantity: 1}}
	combo.ComboPrice.Amount = 200
	second := testkit.Scale()
	second.ID = "later"
	second.Priority = 2
	second.Tiers = []d.Tier{{From: 1, To: 0, BPS: 1000}}
	q, err := pricing.Calculate(testkit.Cart(d.CartItem{SKU: "A", Quantity: 5}, d.CartItem{SKU: "B", Quantity: 3}, d.CartItem{SKU: "C", Quantity: 1}), testkit.Config(), testkit.Products(), []d.Promotion{combo, p, second}, testkit.Config().Money(100000), testkit.Now())
	if err != nil {
		t.Fatal(err)
	}
	// Combo: 50 off; remaining family A3+B3 gives 5% on one B unit (10).
	if q.DiscountTotal.Amount != 60 {
		t.Fatal(q.DiscountTotal)
	}
	for _, a := range q.Adjustments {
		if a.PromotionID == "later" {
			t.Fatal("second scale reused zero-rate units from an occupied family")
		}
	}
	p.IncompatibleWith = []string{combo.ID}
	q, err = pricing.Calculate(testkit.Cart(d.CartItem{SKU: "A", Quantity: 5}, d.CartItem{SKU: "B", Quantity: 3}, d.CartItem{SKU: "C", Quantity: 1}), testkit.Config(), testkit.Products(), []d.Promotion{combo, p}, testkit.Config().Money(100000), testkit.Now())
	if err != nil || q.DiscountTotal.Amount != 50 {
		t.Fatal("incompatible family scale applied", q, err)
	}
}
func TestFamilyScaleBoundary(t *testing.T) {
	p := testkit.Scale()
	p.Eligibility = d.Eligibility{Family: "drinks"}
	q, err := pricing.Calculate(testkit.Cart(d.CartItem{SKU: "A", Quantity: 2}, d.CartItem{SKU: "B", Quantity: 3}), testkit.Config(), testkit.Products(), []d.Promotion{p}, testkit.Config().Money(100000), testkit.Now())
	if err != nil || q.DiscountTotal.Amount != 0 {
		t.Fatal(q, err)
	}
}
