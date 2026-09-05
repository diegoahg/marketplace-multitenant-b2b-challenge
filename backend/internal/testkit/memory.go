package testkit

import (
	"context"
	"marketplace/internal/application"
	d "marketplace/internal/domain"
	"marketplace/internal/ports"
	"sync"
	"time"
)

// Memory is a test double. Production always uses MongoDB transactions.
type Memory struct {
	Mu          sync.Mutex
	Config      d.CountryConfig
	Products    []d.Product
	Promotions  []d.Promotion
	Credit      d.Money
	Quotes      map[string]d.Quote
	Orders      map[string]d.Order
	Idempotency map[string]d.Idempotency
	Events      map[string]d.OrderConfirmedEvent
	Published   map[string]bool
	Effects     map[string]string
	Done        map[string]bool
}

func NewMemory() *Memory {
	return &Memory{Config: Config(), Products: Products(), Credit: Config().Money(100000), Quotes: map[string]d.Quote{}, Orders: map[string]d.Order{}, Idempotency: map[string]d.Idempotency{}, Events: map[string]d.OrderConfirmedEvent{}, Published: map[string]bool{}, Effects: map[string]string{}, Done: map[string]bool{}}
}
func (m *Memory) Service() *application.Service {
	return &application.Service{Catalog: m, Quotes: m, Orders: m, Now: Now, QuoteTTL: 15 * time.Minute}
}
func (m *Memory) Catalog(ctx context.Context, scope d.Scope) (d.CountryConfig, []d.Product, []d.Promotion, d.Money, error) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	return m.Config, append([]d.Product(nil), m.Products...), append([]d.Promotion(nil), m.Promotions...), m.Credit, nil
}
func (m *Memory) SaveQuote(ctx context.Context, q d.Quote) error {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.Quotes[q.QuoteID] = q
	return nil
}
func (m *Memory) GetQuote(ctx context.Context, scope d.Scope, id string) (d.Quote, error) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	q, ok := m.Quotes[id]
	if !ok || q.Scope != scope {
		return d.Quote{}, d.Fail("QUOTE_NOT_FOUND", "not found")
	}
	return q, nil
}
func (m *Memory) GetOrder(ctx context.Context, scope d.Scope, id string) (d.Order, error) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	o, ok := m.Orders[id]
	if !ok || o.Scope != scope {
		return d.Order{}, d.Fail("ORDER_NOT_FOUND", "not found")
	}
	return o, nil
}
func (m *Memory) FindIdempotency(ctx context.Context, tenant, key string) (d.Idempotency, bool, error) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	r, ok := m.Idempotency[tenant+":"+key]
	return r, ok, nil
}
func (m *Memory) Confirm(ctx context.Context, in ports.Confirmation) (d.Order, error) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	key := in.Idempotency.TenantID + ":" + in.Idempotency.Key
	if r, ok := m.Idempotency[key]; ok {
		return application.Replay(r, in.Idempotency.RequestHash)
	}
	for _, o := range m.Orders {
		if o.QuoteID == in.Quote.QuoteID {
			return d.Order{}, d.Fail("QUOTE_INVALID", "already confirmed")
		}
	}
	if in.Quote.PaymentMethod == "CREDIT" {
		if m.Credit.Amount < in.Order.Total.Amount {
			return d.Order{}, d.Fail("CREDIT_INSUFFICIENT", "insufficient")
		}
		var e error
		m.Credit, e = m.Credit.Sub(in.Order.Total)
		if e != nil {
			return d.Order{}, e
		}
	}
	m.Orders[in.Order.OrderID] = in.Order
	m.Idempotency[key] = in.Idempotency
	m.Events[in.Event.EventID] = in.Event
	return in.Order, nil
}
func (m *Memory) Pending(ctx context.Context, n int) ([]d.OrderConfirmedEvent, error) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	out := []d.OrderConfirmedEvent{}
	for id, e := range m.Events {
		if !m.Published[id] && len(out) < n {
			out = append(out, e)
		}
	}
	return out, nil
}
func (m *Memory) MarkPublished(ctx context.Context, id string) error {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.Published[id] = true
	return nil
}
func (m *Memory) ClaimEffect(ctx context.Context, event, kind, owner string, now time.Time) (bool, error) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	k := event + kind
	if m.Done[k] {
		return false, nil
	}
	if m.Effects[k] != "" {
		return false, d.Fail("EFFECT_BUSY", "busy")
	}
	m.Effects[k] = owner
	return true, nil
}
func (m *Memory) CompleteEffect(ctx context.Context, event, kind, owner string) error {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	k := event + kind
	if m.Effects[k] != owner {
		return d.Fail("EFFECT_BUSY", "lease lost")
	}
	m.Done[k] = true
	delete(m.Effects, k)
	return nil
}
func (m *Memory) ReleaseEffect(ctx context.Context, event, kind, owner string) error {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	k := event + kind
	if m.Effects[k] == owner {
		delete(m.Effects, k)
	}
	return nil
}
