package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	d "marketplace/internal/domain"
	"marketplace/internal/ports"
	"marketplace/internal/pricing"
	"time"
)

type Service struct {
	Catalog  ports.Catalog
	Quotes   ports.Quotes
	Orders   ports.Orders
	Tracking ports.OrderTracking
	Now      func() time.Time
	QuoteTTL time.Duration
}

func NewID() string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
func (s *Service) Quote(ctx context.Context, cart d.Cart) (d.Quote, error) {
	cfg, products, promotions, credit, e := s.Catalog.Catalog(ctx, cart.Scope)
	if e != nil {
		return d.Quote{}, fmt.Errorf("load quote context: %w", e)
	}
	q, e := pricing.Calculate(cart, cfg, products, promotions, credit, s.Now().UTC())
	if e != nil {
		return d.Quote{}, e
	}
	q.QuoteID = NewID()
	q.ExpiresAt = q.CreatedAt.Add(s.QuoteTTL)
	if e = s.Quotes.SaveQuote(ctx, q); e != nil {
		return d.Quote{}, fmt.Errorf("save quote: %w", e)
	}
	return q, nil
}
func (s *Service) Confirm(ctx context.Context, scope d.Scope, key string, request d.OrderRequest) (d.Order, error) {
	if !ValidUUID(key) || request.QuoteID == "" || request.CustomerID != scope.CustomerID {
		return d.Order{}, d.Fail("INVALID_REQUEST", "valid Idempotency-Key UUID, quoteId and matching customerId required")
	}
	payload, _ := json.Marshal(struct {
		Scope   d.Scope
		Request d.OrderRequest
	}{scope, request})
	sum := sha256.Sum256(payload)
	hash := hex.EncodeToString(sum[:])
	existing, found, e := s.Orders.FindIdempotency(ctx, scope.TenantID, key)
	if e != nil {
		return d.Order{}, e
	}
	if found {
		return Replay(existing, hash)
	}
	q, e := s.Quotes.GetQuote(ctx, scope, request.QuoteID)
	if e != nil {
		return d.Order{}, e
	}
	now := s.Now().UTC()
	if !now.Before(q.ExpiresAt) {
		return d.Order{}, d.Fail("QUOTE_INVALID", "quote expired; request a new quote")
	}
	id := NewID()
	order := d.Order{Scope: q.Scope, OrderID: id, OrderNumber: "ORD-" + id, QuoteID: q.QuoteID, Currency: q.Currency, Total: q.Total, Status: "CONFIRMED", CreatedAt: now}
	event := d.OrderConfirmedEvent{EventID: NewID(), EventType: "OrderConfirmed", EventVersion: 1, OccurredAt: now, TenantID: q.TenantID, Country: q.Country, Order: order}
	idem := d.Idempotency{TenantID: scope.TenantID, Key: key, RequestHash: hash, OrderID: id, Response: order, Status: 201, CreatedAt: now}
	return s.Orders.Confirm(ctx, ports.Confirmation{Quote: q, Order: order, Event: event, Idempotency: idem, Now: now})
}
func Replay(record d.Idempotency, hash string) (d.Order, error) {
	if record.RequestHash != hash {
		return d.Order{}, d.Fail("IDEMPOTENCY_CONFLICT", "key already used with a different request")
	}
	return record.Response, nil
}
func ValidUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}
