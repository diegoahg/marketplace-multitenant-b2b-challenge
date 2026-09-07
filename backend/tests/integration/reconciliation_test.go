//go:build integration

package integration_test

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"marketplace/internal/application"
	d "marketplace/internal/domain"
	"marketplace/internal/integration"
	"marketplace/internal/messaging"
	"testing"
	"time"
)

type unavailablePush struct{}

func (unavailablePush) SendOrderConfirmed(context.Context, d.OrderConfirmedEvent) error {
	return errors.New("push offline")
}
func TestReconcilePartialEffects(t *testing.T) {
	store, s, ctx := setup(t)
	q := quote(t, s, ctx, 2)
	if _, err := s.Confirm(ctx, q.Scope, application.NewID(), d.OrderRequest{QuoteID: q.QuoteID, CustomerID: q.CustomerID}); err != nil {
		t.Fatal(err)
	}
	events, err := store.Pending(ctx, 10)
	if err != nil || len(events) != 1 {
		t.Fatal(events, err)
	}
	event := events[0]
	if err = store.MarkPublished(ctx, event.EventID); err != nil {
		t.Fatal(err)
	}
	recent, err := store.Pending(ctx, 10)
	if err != nil || len(recent) != 0 {
		t.Fatal("reconciliation cooldown", recent, err)
	}
	if _, err = store.DB.Collection("outbox_events").UpdateOne(ctx, bson.M{"eventId": event.EventID}, bson.M{"$set": bson.M{"publishedAt": time.Now().Add(-2 * time.Minute)}}); err != nil {
		t.Fatal(err)
	}
	w := &messaging.Worker{Effects: store, ERP: integration.New("erp", "", store.DB), Push: unavailablePush{}}
	if err = w.Handle(ctx, event); err == nil {
		t.Fatal("expected unavailable push")
	}
	pending, err := store.Pending(ctx, 10)
	if err != nil || len(pending) != 1 || pending[0].EventID != event.EventID {
		t.Fatal("partial progress lost", pending, err)
	}
	w.Push = integration.New("push", "", store.DB)
	if err = w.Handle(ctx, pending[0]); err != nil {
		t.Fatal(err)
	}
	if pending, err = store.Pending(ctx, 10); err != nil || len(pending) != 0 {
		t.Fatal("completed event replayed", pending, err)
	}
	count, err := store.DB.Collection("mock_integration_receipts").CountDocuments(ctx, bson.M{})
	if err != nil || count != 2 {
		t.Fatal("duplicate or missing effects", count, err)
	}
}
