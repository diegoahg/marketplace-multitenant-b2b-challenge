package application

import (
	"context"
	d "marketplace/internal/domain"
)

type OrderDetails struct {
	Order      d.Order                    `json:"order"`
	Quote      d.Quote                    `json:"quote"`
	Deliveries map[string]d.DeliveryState `json:"deliveries"`
}

func (s *Service) OrderDetails(ctx context.Context, scope d.Scope, id string) (OrderDetails, error) {
	o, err := s.Orders.GetOrder(ctx, scope, id)
	if err != nil {
		return OrderDetails{}, err
	}
	q, err := s.Quotes.GetQuote(ctx, scope, o.QuoteID)
	if err != nil {
		return OrderDetails{}, err
	}
	result := OrderDetails{Order: o, Quote: q}
	if s.Tracking != nil {
		result.Deliveries, err = s.Tracking.OrderDeliveries(ctx, scope, id)
	}
	return result, err
}
