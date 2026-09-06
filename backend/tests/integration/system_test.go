//go:build integration

package integration_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	"marketplace/internal/application"
	d "marketplace/internal/domain"
	"marketplace/internal/integration"
	"marketplace/internal/messaging"
	mongo "marketplace/internal/repository/mongo"
	"marketplace/internal/testkit"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMongoConcurrentDifferentRequestsSameKey(t *testing.T) {
	store, s, ctx := setup(t)
	quotes := []d.Quote{quote(t, s, ctx, 1), quote(t, s, ctx, 2)}
	key := application.NewID()
	results := make(chan error, 2)
	for _, q := range quotes {
		go func(q d.Quote) {
			_, err := s.Confirm(ctx, q.Scope, key, d.OrderRequest{QuoteID: q.QuoteID, CustomerID: q.CustomerID})
			results <- err
		}(q)
	}
	success, conflicts := 0, 0
	for range quotes {
		err := <-results
		if err == nil {
			success++
			continue
		}
		var business *d.Error
		if errors.As(err, &business) && business.Code == "IDEMPOTENCY_CONFLICT" {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatal(success, conflicts)
	}
	n, err := store.DB.Collection("outbox_events").CountDocuments(ctx, bson.M{})
	if err != nil || n != 1 {
		t.Fatal("duplicate event", n, err)
	}
}

func setup(t *testing.T) (*mongo.Store, *application.Service, context.Context) {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Fatal("MONGO_TEST_URI is required for integration tests (Mongo replica set)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	database := "marketplace_test_" + strings.ReplaceAll(application.NewID(), "-", "")
	store, e := mongo.Connect(ctx, uri, database)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		if e := store.DB.Drop(cleanup); e != nil {
			t.Error(e)
		}
		_ = store.Close(cleanup)
	})
	if e = store.Init(ctx); e != nil {
		t.Fatal(e)
	}
	data := mongo.SeedData{Countries: []d.CountryConfig{testkit.Config()}, Products: testkit.Products(), Customers: []d.Customer{{Scope: testkit.Scope()}}, Credits: []d.CreditAccount{{Scope: testkit.Scope(), Available: testkit.Config().Money(100000)}}}
	bytes, e := json.Marshal(data)
	if e != nil {
		t.Fatal(e)
	}
	if e = store.Seed(ctx, strings.NewReader(string(bytes))); e != nil {
		t.Fatal(e)
	}
	return store, &application.Service{Catalog: store, Quotes: store, Orders: store, Now: testkit.Now, QuoteTTL: 15 * time.Minute}, ctx
}
func quote(t *testing.T, s *application.Service, ctx context.Context, n int64) d.Quote {
	t.Helper()
	q, e := s.Quote(ctx, testkit.Cart(d.CartItem{SKU: "A", Quantity: n}))
	if e != nil {
		t.Fatal(e)
	}
	return q
}
func TestMongoGT07HundredConcurrentRetries(t *testing.T) {
	store, s, ctx := setup(t)
	q := quote(t, s, ctx, 2)
	key := application.NewID()
	ids := make(chan string, 100)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o, e := s.Confirm(ctx, q.Scope, key, d.OrderRequest{QuoteID: q.QuoteID, CustomerID: q.CustomerID})
			if e != nil {
				t.Error(e)
				return
			}
			if _, e = store.GetOrder(ctx, q.Scope, o.OrderID); e != nil {
				t.Error("201 not readable", e)
			}
			ids <- o.OrderID
		}()
	}
	wg.Wait()
	close(ids)
	unique := map[string]bool{}
	count := 0
	for id := range ids {
		unique[id] = true
		count++
	}
	if count != 100 || len(unique) != 1 {
		t.Fatalf("responses=%d orders=%d", count, len(unique))
	}
	for _, col := range []string{"orders", "idempotency", "outbox_events"} {
		n, e := store.DB.Collection(col).CountDocuments(ctx, bson.M{})
		if e != nil || n != 1 {
			t.Fatal(col, n, e)
		}
	}
	var credit d.CreditAccount
	if e := store.DB.Collection("credits").FindOne(ctx, bson.M{}).Decode(&credit); e != nil {
		t.Fatal(e)
	}
	if credit.Available.Amount != 100000-q.Total.Amount {
		t.Fatal("credit debited twice")
	}
}
func TestMongoGT08CreditAndRollback(t *testing.T) {
	store, s, ctx := setup(t)
	if _, e := store.DB.Collection("credits").UpdateOne(ctx, bson.M{}, bson.M{"$set": bson.M{"available.amount": int64(1000)}}); e != nil {
		t.Fatal(e)
	}
	qs := []d.Quote{quote(t, s, ctx, 7), quote(t, s, ctx, 5)}
	results := make(chan error, 2)
	for _, q := range qs {
		go func(q d.Quote) {
			_, e := s.Confirm(ctx, q.Scope, application.NewID(), d.OrderRequest{QuoteID: q.QuoteID, CustomerID: q.CustomerID})
			results <- e
		}(q)
	}
	success := 0
	for range qs {
		if <-results == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatal("overspending", success)
	}
	for _, col := range []string{"orders", "outbox_events", "idempotency"} {
		n, e := store.DB.Collection(col).CountDocuments(ctx, bson.M{})
		if e != nil || n != 1 {
			t.Fatal("rollback failed", col, n, e)
		}
	}
	n, e := store.DB.Collection("quotes").CountDocuments(ctx, bson.M{"confirmedOrderId": bson.M{"$exists": true}})
	if e != nil || n != 1 {
		t.Fatal("quote reservation was not rolled back", n, e)
	}
}
func TestMongoSnapshotAndTenantIsolation(t *testing.T) {
	store, s, ctx := setup(t)
	q := quote(t, s, ctx, 2)
	if _, e := store.DB.Collection("products").UpdateMany(ctx, bson.M{}, bson.M{"$set": bson.M{"unitPrice.amount": int64(99999)}}); e != nil {
		t.Fatal(e)
	}
	o, e := s.Confirm(ctx, q.Scope, application.NewID(), d.OrderRequest{QuoteID: q.QuoteID, CustomerID: q.CustomerID})
	if e != nil || o.Total != q.Total {
		t.Fatal(o, e)
	}
	other := q.Scope
	other.Country = "CL"
	if _, e = store.GetOrder(ctx, other, o.OrderID); e == nil {
		t.Fatal("cross country read")
	}
	if _, e = store.GetQuote(ctx, other, q.QuoteID); e == nil {
		t.Fatal("cross country quote")
	}
}
func TestMongoPubSubOutboxAndRedelivery(t *testing.T) {
	store, s, ctx := setup(t)
	host := os.Getenv("PUBSUB_EMULATOR_HOST")
	if host == "" {
		t.Fatal("PUBSUB_EMULATOR_HOST required")
	}
	suffix := strings.ReplaceAll(application.NewID(), "-", "")
	p, e := messaging.NewEmulator(host, "test-project", "orders-"+suffix, "worker-"+suffix)
	if e != nil {
		t.Fatal(e)
	}
	for {
		if e = p.Setup(ctx); e == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal(e)
		case <-time.After(time.Second):
		}
	}
	q := quote(t, s, ctx, 2)
	if _, e = s.Confirm(ctx, q.Scope, application.NewID(), d.OrderRequest{QuoteID: q.QuoteID, CustomerID: q.CustomerID}); e != nil {
		t.Fatal(e)
	}
	if e = messaging.DispatchOnce(ctx, store, p); e != nil {
		t.Fatal(e)
	}
	pending, e := store.Pending(ctx, 10)
	if e != nil || len(pending) != 0 {
		t.Fatal(pending, e)
	}
	w := &messaging.Worker{Effects: store, ERP: integration.New("erp", "", store.DB), Push: integration.New("push", "", store.DB)}
	var delivered d.OrderConfirmedEvent
	pull := func() {
		t.Helper()
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			messages, e := p.Pull(ctx)
			if e != nil {
				t.Fatal(e)
			}
			if len(messages) > 0 {
				raw, e := base64.StdEncoding.DecodeString(messages[0].Message.Data)
				if e != nil {
					t.Fatal(e)
				}
				if e = json.Unmarshal(raw, &delivered); e != nil {
					t.Fatal(e)
				}
				if e = w.Handle(ctx, delivered); e != nil {
					t.Fatal(e)
				}
				if e = p.Ack(ctx, messages[0].AckID); e != nil {
					t.Fatal(e)
				}
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatal("message not delivered")
	}
	pull()
	if e = p.Publish(ctx, delivered); e != nil {
		t.Fatal(e)
	}
	pull()
	n, e := store.DB.Collection("mock_integration_receipts").CountDocuments(ctx, bson.M{})
	if e != nil || n != 2 {
		t.Fatal("duplicate effects", n, e)
	}
	// Simulate a worker dying after the recipient committed but before the local marker.
	if _, e = store.DB.Collection("processed_events").DeleteOne(ctx, bson.M{"eventId": delivered.EventID}); e != nil {
		t.Fatal(e)
	}
	if e = w.Handle(ctx, delivered); e != nil {
		t.Fatal(e)
	}
	n, e = store.DB.Collection("mock_integration_receipts").CountDocuments(ctx, bson.M{})
	if e != nil || n != 2 {
		t.Fatal(fmt.Sprint("crash window duplicated effects ", n, e))
	}
}
