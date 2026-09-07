package mongo

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	d "marketplace/internal/domain"
)

func (s *Store) OrderDeliveries(ctx context.Context, scope d.Scope, id string) (map[string]d.DeliveryState, error) {
	var event d.OrderConfirmedEvent
	filter := bson.M{"tenantId": scope.TenantID, "country": scope.Country, "order.customerId": scope.CustomerID, "order.orderId": id}
	err := s.DB.Collection("outbox_events").FindOne(ctx, filter).Decode(&event)
	if errors.Is(err, driver.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	states := make(map[string]d.DeliveryState, 2)
	for _, kind := range []string{"erp", "push"} {
		state, err := s.DeliveryState(ctx, event.EventID, kind)
		if err != nil {
			return nil, err
		}
		states[kind] = state
	}
	return states, nil
}
