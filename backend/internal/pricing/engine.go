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

type engine struct {
	calculation
	q                        d.Quote
	config                   d.CountryConfig
	catalog                  map[string]d.Product
	indices                  map[string]int
	remaining                map[string]int64
	applied, blocked, scaled map[string]bool
}

func invalid(message string) error { return d.Fail("INVALID_REQUEST", message) }
func validateContext(cart d.Cart, cfg d.CountryConfig, available d.Money) error {
	if cart.TenantID == "" || cart.Country == "" || cart.CustomerID == "" || len(cart.Items) == 0 || len(cart.Items) > 100 {
		return invalid("cart requires tenant, country, customer and 1..100 items")
	}
	if cart.PaymentMethod != "CREDIT" && cart.PaymentMethod != "CASH" {
		return invalid("paymentMethod must be CREDIT or CASH")
	}
	if cfg.TenantID != cart.TenantID || cfg.Country != cart.Country || !cfg.Money(0).Valid() || (cfg.Rounding != d.HalfUp && cfg.Rounding != d.HalfEven) {
		return invalid("invalid country configuration")
	}
	if !available.Same(cfg.Money(0)) || available.Amount < 0 {
		return invalid("invalid credit currency")
	}
	return nil
}
func Calculate(cart d.Cart, config d.CountryConfig, products []d.Product, promotions []d.Promotion, available d.Money, now time.Time) (d.Quote, error) {
	if err := validateContext(cart, config, available); err != nil {
		return d.Quote{}, err
	}
	zero := config.Money(0)
	e := &engine{config: config, catalog: map[string]d.Product{}, indices: map[string]int{}, remaining: map[string]int64{}, applied: map[string]bool{}, blocked: map[string]bool{}, scaled: map[string]bool{}}
	e.q = d.Quote{Scope: cart.Scope, Currency: config.Currency, PaymentMethod: cart.PaymentMethod, Lines: []d.Line{}, Adjustments: []d.Adjustment{}, Gifts: []d.Gift{}, GrossSubtotal: zero, DiscountTotal: zero, TaxableBase: zero, TaxTotal: zero, Total: zero, CreatedAt: now}
	for _, p := range products {
		if p.TenantID == cart.TenantID && p.Country == cart.Country {
			e.catalog[p.SKU] = p
		}
	}
	for _, item := range cart.Items {
		if err := e.addLine(item); err != nil {
			return d.Quote{}, err
		}
	}
	rules := append([]d.Promotion(nil), promotions...)
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Priority == rules[j].Priority {
			return rules[i].ID < rules[j].ID
		}
		return rules[i].Priority < rules[j].Priority
	})
	for _, stage := range []struct {
		kind  string
		apply func(d.Promotion) error
	}{{"COMBO", e.combo}, {"SCALE", e.scale}, {"GIFT", e.gift}} {
		for _, p := range rules {
			if p.Type == stage.kind && e.active(p) {
				if err := stage.apply(p); err != nil {
					return d.Quote{}, err
				}
			}
		}
	}
	if err := e.totals(available); err != nil {
		return d.Quote{}, err
	}
	return e.q, nil
}
func (e *engine) taxRate(p d.Product) (int64, error) {
	rate, ok := e.config.TaxBPS[p.TaxCategory]
	if !ok || rate < 0 || rate > 10000 {
		return 0, invalid("missing or invalid tax category")
	}
	return rate, nil
}
func (e *engine) addLine(item d.CartItem) error {
	if item.SKU == "" || item.Quantity < 1 || item.Quantity > 1000000 {
		return invalid("quantity must be between 1 and 1000000")
	}
	if _, ok := e.indices[item.SKU]; ok {
		return invalid("duplicate SKU; combine quantities")
	}
	p, ok := e.catalog[item.SKU]
	if !ok {
		return d.Fail("PRODUCT_NOT_FOUND", item.SKU)
	}
	zero := e.config.Money(0)
	if !p.UnitPrice.Same(zero) || p.UnitPrice.Amount < 0 {
		return invalid("invalid product price")
	}
	if _, err := e.taxRate(p); err != nil {
		return err
	}
	e.indices[item.SKU] = len(e.q.Lines)
	e.remaining[item.SKU] = item.Quantity
	e.q.Lines = append(e.q.Lines, d.Line{SKU: item.SKU, Quantity: item.Quantity, UnitPrice: p.UnitPrice, Gross: e.mul(p.UnitPrice, item.Quantity), Discount: zero, TaxableBase: zero, Tax: zero})
	return nil
}
func (e *engine) active(p d.Promotion) bool {
	if p.TenantID != e.q.TenantID || p.Country != e.q.Country || e.q.CreatedAt.Before(p.ValidFrom) || !e.q.CreatedAt.Before(p.ValidTo) || e.blocked[p.ID] {
		return false
	}
	for _, id := range p.IncompatibleWith {
		if e.applied[id] {
			return false
		}
	}
	return true
}
func (e *engine) mark(p d.Promotion) {
	e.applied[p.ID] = true
	for _, id := range p.IncompatibleWith {
		e.blocked[id] = true
	}
}
func (e *engine) adjust(p d.Promotion, sku string, n int64, amount d.Money) {
	i := e.indices[sku]
	e.q.Lines[i].Discount = e.add(e.q.Lines[i].Discount, amount)
	e.q.Adjustments = append(e.q.Adjustments, d.Adjustment{PromotionID: p.ID, PromotionType: p.Type, SKU: sku, Quantity: n, DiscountAmount: amount})
}
func (e *engine) comboSize(p d.Promotion) (int64, d.Money, error) {
	gross := e.config.Money(0)
	count := int64(1000000)
	seen := map[string]bool{}
	if len(p.Items) == 0 || !p.ComboPrice.Same(gross) || p.ComboPrice.Amount < 0 {
		return 0, gross, invalid("invalid combo")
	}
	for _, it := range p.Items {
		if it.Quantity <= 0 || seen[it.SKU] {
			return 0, gross, invalid("invalid combo items")
		}
		seen[it.SKU] = true
		count = min(count, e.remaining[it.SKU]/it.Quantity)
		if product, ok := e.catalog[it.SKU]; ok {
			gross = e.add(gross, e.mul(product.UnitPrice, it.Quantity))
		}
	}
	return count, gross, nil
}
func (e *engine) combo(p d.Promotion) error {
	count, gross, err := e.comboSize(p)
	if err != nil || count == 0 {
		return err
	}
	if gross.Amount < p.ComboPrice.Amount || gross.Amount <= 0 {
		return invalid("combo must not increase price")
	}
	left := e.mul(e.sub(gross, p.ComboPrice), count)
	weightLeft := e.mul(gross, count)
	for index, it := range p.Items {
		n := it.Quantity * count
		weight := e.mul(e.catalog[it.SKU].UnitPrice, n)
		amount := left
		if index < len(p.Items)-1 && weightLeft.Amount > 0 {
			amount = e.take(left.Ratio(weight.Amount, weightLeft.Amount, e.config.Rounding))
		}
		if amount.Amount > weight.Amount {
			return invalid("invalid combo allocation")
		}
		e.adjust(p, it.SKU, n, amount)
		left = e.sub(left, amount)
		weightLeft = e.sub(weightLeft, weight)
		e.remaining[it.SKU] -= n
		e.q.Lines[e.indices[it.SKU]].ComboQuantity += n
	}
	e.mark(p)
	return nil
}
func validateTiers(p d.Promotion) error {
	previous := int64(0)
	for i, t := range p.Tiers {
		if t.From != previous+1 || t.BPS < 0 || t.BPS > 10000 || (t.To != 0 && t.To < t.From) || (t.To == 0 && i != len(p.Tiers)-1) {
			return invalid("invalid cumulative tiers")
		}
		previous = t.To
	}
	return nil
}
func (e *engine) scale(p d.Promotion) error {
	if err := validateTiers(p); err != nil {
		return err
	}
	skus := []string{}
	for _, line := range e.q.Lines {
		if !e.scaled[line.SKU] && e.remaining[line.SKU] > 0 && eligible(p, e.catalog[line.SKU]) {
			skus = append(skus, line.SKU)
		}
	}
	sort.Strings(skus)
	if p.Eligibility.Family != "" {
		e.scaleGroup(p, skus)
	} else {
		for _, sku := range skus {
			e.scaleGroup(p, []string{sku})
		}
	}
	return nil
}

// Family tranches use remaining quantities in ascending SKU order, independent of cart order.
// All units in a benefiting group are occupied by that scale, including its zero-rate tranche.
func (e *engine) scaleGroup(p d.Promotion, skus []string) {
	total := int64(0)
	for _, sku := range skus {
		total += e.remaining[sku]
	}
	if total < p.Eligibility.Minimum {
		return
	}
	offset := int64(0)
	used := false
	for _, sku := range skus {
		n := e.remaining[sku]
		for _, t := range p.Tiers {
			end := t.To
			if end == 0 {
				end = total
			}
			units := min(offset+n, end) - max(offset+1, t.From) + 1
			if units <= 0 || t.BPS == 0 {
				continue
			}
			e.adjust(p, sku, units, e.rate(e.mul(e.catalog[sku].UnitPrice, units), t.BPS, e.config.Rounding))
			used = true
		}
		offset += n
	}
	if used {
		for _, sku := range skus {
			e.scaled[sku] = true
		}
		e.mark(p)
	}
}
func (e *engine) gift(p d.Promotion) error {
	if p.Eligibility.Minimum <= 0 || p.GiftQuantity <= 0 || p.GiftQuantity > 1000000 {
		return invalid("invalid gift rule")
	}
	units := int64(0)
	for _, line := range e.q.Lines {
		if eligible(p, e.catalog[line.SKU]) {
			units += line.Quantity
		}
	}
	n := units / p.Eligibility.Minimum * p.GiftQuantity
	if n == 0 {
		return nil
	}
	product, ok := e.catalog[p.GiftSKU]
	if !ok {
		return d.Fail("PRODUCT_NOT_FOUND", p.GiftSKU)
	}
	if !product.UnitPrice.Same(e.config.Money(0)) || product.UnitPrice.Amount < 0 {
		return invalid("invalid gift reference price")
	}
	ref := e.mul(product.UnitPrice, n)
	tax := e.config.Money(0)
	if e.config.TaxGifts {
		rate, err := e.taxRate(product)
		if err != nil {
			return err
		}
		tax = e.rate(ref, rate, e.config.Rounding)
	}
	e.q.Gifts = append(e.q.Gifts, d.Gift{SKU: p.GiftSKU, Quantity: n, PromotionID: p.ID, ReferenceValue: ref, Tax: tax})
	e.q.TaxTotal = e.add(e.q.TaxTotal, tax)
	e.mark(p)
	return nil
}
func (e *engine) totals(available d.Money) error {
	for i := range e.q.Lines {
		line := &e.q.Lines[i]
		line.TaxableBase = e.sub(line.Gross, line.Discount)
		if line.TaxableBase.Amount < 0 {
			return invalid("discount exceeds gross")
		}
		line.Tax = e.rate(line.TaxableBase, e.config.TaxBPS[e.catalog[line.SKU].TaxCategory], e.config.Rounding)
		e.q.GrossSubtotal = e.add(e.q.GrossSubtotal, line.Gross)
		e.q.DiscountTotal = e.add(e.q.DiscountTotal, line.Discount)
		e.q.TaxableBase = e.add(e.q.TaxableBase, line.TaxableBase)
		e.q.TaxTotal = e.add(e.q.TaxTotal, line.Tax)
	}
	e.q.Total = e.add(e.q.TaxableBase, e.q.TaxTotal)
	e.q.CreditEvaluation = d.CreditEvaluation{Eligible: e.q.PaymentMethod != "CREDIT" || available.Amount >= e.q.Total.Amount, Available: available}
	if e.err != nil {
		return fmt.Errorf("pricing arithmetic: %w", e.err)
	}
	return nil
}
func eligible(p d.Promotion, product d.Product) bool {
	return (p.Eligibility.SKU == "" || p.Eligibility.SKU == product.SKU) && (p.Eligibility.Family == "" || p.Eligibility.Family == product.Family)
}
