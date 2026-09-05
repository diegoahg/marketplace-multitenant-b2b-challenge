package ports

import (
	"context"
	d "marketplace/internal/domain"
	"time"
)

type Catalog interface {
	Catalog(context.Context, d.Scope) (d.CountryConfig, []d.Product, []d.Promotion, d.Money, error)
}
type Quotes interface {
	SaveQuote(context.Context, d.Quote) error
	GetQuote(context.Context, d.Scope, string) (d.Quote, error)
}
type Confirmation struct {
	Quote       d.Quote
	Order       d.Order
	Event       d.OrderConfirmedEvent
	Idempotency d.Idempotency
	Now         time.Time
}
type Orders interface {
	FindIdempotency(context.Context, string, string) (d.Idempotency, bool, error)
	Confirm(context.Context, Confirmation) (d.Order, error)
	GetOrder(context.Context, d.Scope, string) (d.Order, error)
}
type Outbox interface {
	Pending(context.Context, int) ([]d.OrderConfirmedEvent, error)
	MarkPublished(context.Context, string) error
}
type Publisher interface {
	Publish(context.Context, d.OrderConfirmedEvent) error
}
type ERPClient interface {
	SendOrder(context.Context, d.OrderConfirmedEvent) error
}
type NotificationClient interface {
	SendOrderConfirmed(context.Context, d.OrderConfirmedEvent) error
}
type Effects interface {
	ClaimEffect(context.Context, string, string, string, time.Time) (bool, error)
	CompleteEffect(context.Context, string, string, string) error
	ReleaseEffect(context.Context, string, string, string) error
}
