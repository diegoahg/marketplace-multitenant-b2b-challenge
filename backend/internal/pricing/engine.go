package pricing

import (
	"fmt"
	d "marketplace/internal/domain"
	"sort"
	"time"
)

type calculation struct{ err error }

func (c *calculation) take(m d.Money, e error) d.Money {
	if e != nil && c.err == nil {
		c.err = e
	}
	return m
}
func (c *calculation) add(a, b d.Money) d.Money                      { return c.take(a.Add(b)) }
func (c *calculation) sub(a, b d.Money) d.Money                      { return c.take(a.Sub(b)) }
func (c *calculation) mul(a d.Money, n int64) d.Money                { return c.take(a.Mul(n)) }
func (c *calculation) rate(a d.Money, n int64, r d.Rounding) d.Money { return c.take(a.Percent(n, r)) }

func Calculate(cart d.Cart, config d.CountryConfig, products []d.Product, promotions []d.Promotion, available d.Money, now time.Time) (d.Quote, error) {
	bad := func(s string) (d.Quote, error) { return d.Quote{}, d.Fail("INVALID_REQUEST", s) }
	if cart.TenantID == "" || cart.Country == "" || cart.CustomerID == "" || len(cart.Items) == 0 || len(cart.Items) > 100 {
		return bad("cart requires tenant, country, customer and 1..100 items")
	}
	if cart.PaymentMethod != "CREDIT" && cart.PaymentMethod != "CASH" {
		return bad("paymentMethod must be CREDIT or CASH")
	}
	if config.TenantID != cart.TenantID || config.Country != cart.Country || !config.Money(0).Valid() || (config.Rounding != d.HalfUp && config.Rounding != d.HalfEven) {
		return bad("invalid country configuration")
	}
	if !available.Same(config.Money(0)) || available.Amount < 0 {
		return bad("invalid credit currency")
	}
	c := &calculation{}
	zero := config.Money(0)
	q := d.Quote{Scope: cart.Scope, Currency: config.Currency, PaymentMethod: cart.PaymentMethod, Lines: []d.Line{}, Adjustments: []d.Adjustment{}, Gifts: []d.Gift{}, GrossSubtotal: zero, DiscountTotal: zero, TaxableBase: zero, TaxTotal: zero, Total: zero, CreatedAt: now}
	catalog := map[string]d.Product{}
	for _, p := range products {
		if p.TenantID == cart.TenantID && p.Country == cart.Country {
			catalog[p.SKU] = p
		}
	}
	indices := map[string]int{}
	remaining := map[string]int64{}
	for _, item := range cart.Items {
		if item.SKU == "" || item.Quantity < 1 || item.Quantity > 1000000 {
			return bad("quantity must be between 1 and 1000000")
		}
		if _, ok := indices[item.SKU]; ok {
			return bad("duplicate SKU; combine quantities")
		}
		p, ok := catalog[item.SKU]
		if !ok {
			return d.Quote{}, d.Fail("PRODUCT_NOT_FOUND", item.SKU)
		}
		if !p.UnitPrice.Same(zero) || p.UnitPrice.Amount < 0 {
			return bad("invalid product price")
		}
		rate, ok := config.TaxBPS[p.TaxCategory]
		if !ok || rate < 0 || rate > 10000 {
			return bad("missing or invalid tax category")
		}
		indices[item.SKU] = len(q.Lines)
		remaining[item.SKU] = item.Quantity
		q.Lines = append(q.Lines, d.Line{SKU: item.SKU, Quantity: item.Quantity, UnitPrice: p.UnitPrice, Gross: c.mul(p.UnitPrice, item.Quantity), Discount: zero, TaxableBase: zero, Tax: zero})
	}
	rules := append([]d.Promotion(nil), promotions...)
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Priority == rules[j].Priority {
			return rules[i].ID < rules[j].ID
		}
		return rules[i].Priority < rules[j].Priority
	})
	applied := map[string]bool{}
	blocked := map[string]bool{}
	active := func(p d.Promotion) bool {
		if p.TenantID != cart.TenantID || p.Country != cart.Country || now.Before(p.ValidFrom) || !now.Before(p.ValidTo) || blocked[p.ID] {
			return false
		}
		for _, id := range p.IncompatibleWith {
			if applied[id] {
				return false
			}
		}
		return true
	}
	mark := func(p d.Promotion) {
		applied[p.ID] = true
		for _, id := range p.IncompatibleWith {
			blocked[id] = true
		}
	}
	adjust := func(p d.Promotion, sku string, n int64, amount d.Money) {
		i := indices[sku]
		q.Lines[i].Discount = c.add(q.Lines[i].Discount, amount)
		q.Adjustments = append(q.Adjustments, d.Adjustment{PromotionID: p.ID, PromotionType: p.Type, SKU: sku, Quantity: n, DiscountAmount: amount})
	}
	// Combo units are reserved before scale, irrespective of cross-type priority.
	for _, p := range rules {
		if p.Type != "COMBO" || !active(p) {
			continue
		}
		if len(p.Items) == 0 || !p.ComboPrice.Same(zero) || p.ComboPrice.Amount < 0 {
			return bad("invalid combo")
		}
		count := int64(1000000)
		seen := map[string]bool{}
		gross := zero
		for _, it := range p.Items {
			if it.Quantity <= 0 || seen[it.SKU] {
				return bad("invalid combo items")
			}
			seen[it.SKU] = true
			n := remaining[it.SKU] / it.Quantity
			if n < count {
				count = n
			}
			if product, ok := catalog[it.SKU]; ok {
				gross = c.add(gross, c.mul(product.UnitPrice, it.Quantity))
			}
		}
		if count == 0 {
			continue
		}
		if gross.Amount < p.ComboPrice.Amount || gross.Amount <= 0 {
			return bad("combo must not increase price")
		}
		totalDiscount := c.mul(c.sub(gross, p.ComboPrice), count)
		left := totalDiscount
		// Sequential proportional allocation preserves every minor unit, including rounding residue.
		weightLeft := c.mul(gross, count)
		for index, it := range p.Items {
			n := it.Quantity * count
			weight := c.mul(catalog[it.SKU].UnitPrice, n)
			amount := left
			if index < len(p.Items)-1 && weightLeft.Amount > 0 {
				amount = c.take(left.Ratio(weight.Amount, weightLeft.Amount, config.Rounding))
			}
			if amount.Amount > weight.Amount {
				return bad("invalid combo allocation")
			}
			adjust(p, it.SKU, n, amount)
			left = c.sub(left, amount)
			weightLeft = c.sub(weightLeft, weight)
			remaining[it.SKU] -= n
			q.Lines[indices[it.SKU]].ComboQuantity += n
		}
		mark(p)
	}
	scaled := map[string]bool{}
	for _, p := range rules {
		if p.Type != "SCALE" || !active(p) {
			continue
		}
		previous := int64(0)
		for i, t := range p.Tiers {
			if t.From != previous+1 || t.BPS < 0 || t.BPS > 10000 || (t.To != 0 && t.To < t.From) || (t.To == 0 && i != len(p.Tiers)-1) {
				return bad("invalid cumulative tiers")
			}
			previous = t.To
		}
		for _, line := range q.Lines {
			product := catalog[line.SKU]
			n := remaining[line.SKU]
			if scaled[line.SKU] || n <= 0 || !eligible(p, product) || n < p.Eligibility.Minimum {
				continue
			}
			used := false
			for _, t := range p.Tiers {
				end := t.To
				if end == 0 || end > n {
					end = n
				}
				units := end - t.From + 1
				if units <= 0 || t.BPS == 0 {
					continue
				}
				amount := c.rate(c.mul(product.UnitPrice, units), t.BPS, config.Rounding)
				adjust(p, line.SKU, units, amount)
				used = true
			}
			if used {
				scaled[line.SKU] = true
				mark(p)
			}
		}
	}
	for _, p := range rules {
		if p.Type != "GIFT" || !active(p) {
			continue
		}
		if p.Eligibility.Minimum <= 0 || p.GiftQuantity <= 0 || p.GiftQuantity > 1000000 {
			return bad("invalid gift rule")
		}
		var units int64
		for _, line := range q.Lines {
			if eligible(p, catalog[line.SKU]) {
				units += line.Quantity
			}
		}
		n := units / p.Eligibility.Minimum * p.GiftQuantity
		if n == 0 {
			continue
		}
		product, ok := catalog[p.GiftSKU]
		if !ok {
			return d.Quote{}, d.Fail("PRODUCT_NOT_FOUND", p.GiftSKU)
		}
		if !product.UnitPrice.Same(zero) || product.UnitPrice.Amount < 0 {
			return bad("invalid gift reference price")
		}
		ref := c.mul(product.UnitPrice, n)
		tax := zero
		if config.TaxGifts {
			rate, ok := config.TaxBPS[product.TaxCategory]
			if !ok || rate < 0 || rate > 10000 {
				return bad("invalid gift tax category")
			}
			tax = c.rate(ref, rate, config.Rounding)
		}
		q.Gifts = append(q.Gifts, d.Gift{SKU: p.GiftSKU, Quantity: n, PromotionID: p.ID, ReferenceValue: ref, Tax: tax})
		q.TaxTotal = c.add(q.TaxTotal, tax)
		mark(p)
	}
	for i := range q.Lines {
		line := &q.Lines[i]
		line.TaxableBase = c.sub(line.Gross, line.Discount)
		if line.TaxableBase.Amount < 0 {
			return bad("discount exceeds gross")
		}
		line.Tax = c.rate(line.TaxableBase, config.TaxBPS[catalog[line.SKU].TaxCategory], config.Rounding)
		q.GrossSubtotal = c.add(q.GrossSubtotal, line.Gross)
		q.DiscountTotal = c.add(q.DiscountTotal, line.Discount)
		q.TaxableBase = c.add(q.TaxableBase, line.TaxableBase)
		q.TaxTotal = c.add(q.TaxTotal, line.Tax)
	}
	q.Total = c.add(q.TaxableBase, q.TaxTotal)
	q.CreditEvaluation = d.CreditEvaluation{Eligible: cart.PaymentMethod != "CREDIT" || available.Amount >= q.Total.Amount, Available: available}
	if c.err != nil {
		return d.Quote{}, fmt.Errorf("pricing arithmetic: %w", c.err)
	}
	return q, nil
}
func eligible(p d.Promotion, product d.Product) bool {
	return (p.Eligibility.SKU == "" || p.Eligibility.SKU == product.SKU) && (p.Eligibility.Family == "" || p.Eligibility.Family == product.Family)
}
