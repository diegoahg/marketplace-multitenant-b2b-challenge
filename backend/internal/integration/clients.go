package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"io"
	d "marketplace/internal/domain"
	"net/http"
	"time"
)

// Client either calls a configured HTTP endpoint or records a durable fake effect.
// HTTP recipients must honor Idempotency-Key across retries, including lost responses.
type Client struct {
	Kind, URL     string
	HTTP          *http.Client
	Receipts      *mongo.Collection
	DummyFailures *mongo.Collection
}

func New(kind, url string, db *mongo.Database) *Client {
	return &Client{Kind: kind, URL: url, HTTP: &http.Client{Timeout: 8 * time.Second}, Receipts: db.Collection("mock_integration_receipts"), DummyFailures: db.Collection("dummy_failures")}
}
func (c *Client) SendOrder(ctx context.Context, event d.OrderConfirmedEvent) error {
	return c.send(ctx, event)
}
func (c *Client) SendOrderConfirmed(ctx context.Context, event d.OrderConfirmedEvent) error {
	return c.send(ctx, event)
}
func (c *Client) send(ctx context.Context, event d.OrderConfirmedEvent) error {
	key := event.EventID + ":" + c.Kind
	if c.URL == "" {
		// Demo-only failure injection, scoped to exactly one event and destination.
		if c.DummyFailures != nil {
			err := c.DummyFailures.FindOneAndUpdate(ctx, bson.M{"_id": key, "remaining": bson.M{"$gt": 0}}, bson.M{"$inc": bson.M{"remaining": -1}}).Err()
			if err == nil {
				return fmt.Errorf("dummy %s failure", c.Kind)
			}
			if !errors.Is(err, mongo.ErrNoDocuments) {
				return err
			}
		}
		_, e := c.Receipts.UpdateOne(ctx, bson.M{"_id": key}, bson.M{"$setOnInsert": bson.M{"event": event, "kind": c.Kind, "createdAt": time.Now().UTC()}}, options.UpdateOne().SetUpsert(true))
		if mongo.IsDuplicateKeyError(e) {
			return nil
		}
		return e
	}
	data, e := json.Marshal(event)
	if e != nil {
		return e
	}
	request, e := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(data))
	if e != nil {
		return e
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", key)
	response, e := c.HTTP.Do(request)
	if e != nil {
		return e
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%s HTTP %d", c.Kind, response.StatusCode)
	}
	return nil
}
