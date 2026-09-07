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

type alertFake struct {
	Fail  bool
	Calls int
}

func (a *alertFake) Send(context.Context, d.DeadLetter) error {
	a.Calls++
	if a.Fail {
		return errors.New("SMTP offline")
	}
	return nil
}

func TestDestinationRetriesAndDurableDLQ(t *testing.T) {
	for _, failCount := range []int{5, 6} {
		t.Run(string(rune('0'+failCount)), func(t *testing.T) {
			store, service, ctx := setup(t)
			q := quote(t, service, ctx, 2)
			if _, err := service.Confirm(ctx, q.Scope, application.NewID(), d.OrderRequest{QuoteID: q.QuoteID, CustomerID: q.CustomerID}); err != nil {
				t.Fatal(err)
			}
			events, err := store.Pending(ctx, 10)
			if err != nil || len(events) != 1 {
				t.Fatal(events, err)
			}
			event := events[0]
			now := time.Now().UTC().Truncate(time.Millisecond)
			calls := 0
			makeDestination := func() *messaging.Destination {
				return &messaging.Destination{Kind: "erp", Store: store, Now: func() time.Time { return now }, Send: func(context.Context, d.OrderConfirmedEvent) error {
					calls++
					if calls <= failCount {
						return errors.New("ERP offline")
					}
					return nil
				}}
			}
			push := &messaging.Destination{Kind: "push", Store: store, Now: func() time.Time { return now }, Send: integration.New("push", "", store.DB).SendOrderConfirmed}
			for attempt := 1; attempt <= 6; attempt++ {
				// Reconstruct the consumer every time: retry state must survive restart.
				w := makeDestination()
				ack, delay, err := w.Process(ctx, event, "raw")
				if attempt == 6 {
					if !ack {
						t.Fatalf("terminal disposition: %v", err)
					}
					break
				}
				if ack || err == nil || delay != time.Second*time.Duration(1<<(attempt-1)) {
					t.Fatal("retry policy", ack, delay, err)
				}
				state, err := store.DeliveryState(ctx, event.EventID, "erp")
				if err != nil || state.Failures != attempt {
					t.Fatal(state, err)
				}
				if ack, _, _ := w.Process(ctx, event, "raw"); ack || calls != attempt {
					t.Fatal("early retry executed")
				}
				if attempt == 1 {
					if ack, _, err := push.Process(ctx, event, "raw"); !ack || err != nil {
						t.Fatal("PUSH blocked by ERP", err)
					}
				}
				now = now.Add(delay)
			}
			if calls != 6 {
				t.Fatal("wrong attempt count", calls)
			}
			if ack, _, err := makeDestination().Process(ctx, event, "raw"); !ack || err != nil || calls != 6 {
				t.Fatal("terminal redelivery repeated action", err)
			}
			state, err := store.DeliveryState(ctx, event.EventID, "erp")
			if err != nil {
				t.Fatal(err)
			}
			count, err := store.DB.Collection("dead_letters").CountDocuments(ctx, bson.M{})
			if err != nil {
				t.Fatal(err)
			}
			if failCount == 5 {
				if !state.Done || state.Dead || count != 0 {
					t.Fatal(state, count)
				}
			} else {
				if state.Done || !state.Dead || count != 1 {
					t.Fatal(state, count)
				}
				var letter d.DeadLetter
				if err := store.DB.Collection("dead_letters").FindOne(ctx, bson.M{}).Decode(&letter); err != nil {
					t.Fatal(err)
				}
				if letter.Attempts != 6 || letter.Destination != "erp" || letter.Event.EventID != event.EventID || letter.Error != "ERP offline" {
					t.Fatal(letter)
				}
				mail := &alertFake{Fail: true}
				if err := messaging.AlertOnce(ctx, store, mail); err == nil {
					t.Fatal("expected SMTP failure")
				}
				if n, _ := store.DB.Collection("dead_letters").CountDocuments(ctx, bson.M{"emailedAt": bson.M{"$exists": true}}); n != 0 {
					t.Fatal("failed email marked sent")
				}
				if _, err := store.DB.Collection("dead_letters").UpdateOne(ctx, bson.M{}, bson.M{"$set": bson.M{"nextMail": time.Now().Add(-time.Minute)}}); err != nil {
					t.Fatal(err)
				}
				mail.Fail = false
				if err := messaging.AlertOnce(ctx, store, mail); err != nil {
					t.Fatal(err)
				}
				if err := messaging.AlertOnce(ctx, store, mail); err != nil || mail.Calls != 2 {
					t.Fatal("duplicate mail", err, mail.Calls)
				}
			}
			if _, err := store.DB.Collection("outbox_events").UpdateOne(ctx, bson.M{"eventId": event.EventID}, bson.M{"$set": bson.M{"publishedAt": time.Now().Add(-2 * time.Minute)}}); err != nil {
				t.Fatal(err)
			}
			if pending, err := store.Pending(ctx, 10); err != nil || len(pending) != 0 {
				t.Fatal("terminal destination republished", pending, err)
			}
		})
	}
}
