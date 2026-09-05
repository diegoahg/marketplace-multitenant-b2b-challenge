package mongo

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"marketplace/internal/application"
	d "marketplace/internal/domain"
	"marketplace/internal/ports"
)

func (s *Store) Confirm(ctx context.Context, input ports.Confirmation) (d.Order, error) {
	session, e := s.Client.StartSession()
	if e != nil {
		return d.Order{}, e
	}
	defer session.EndSession(ctx)
	result, e := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		record, found, e := s.FindIdempotency(tx, input.Idempotency.TenantID, input.Idempotency.Key)
		if e != nil {
			return nil, e
		}
		if found {
			return application.Replay(record, input.Idempotency.RequestHash)
		}
		// Claim the snapshot in the same transaction; a second key cannot buy it twice.
		qfilter := scopeFilter(input.Quote.Scope)
		qfilter["quoteId"] = input.Quote.QuoteID
		qfilter["expiresAt"] = bson.M{"$gt": input.Now}
		qfilter["confirmedOrderId"] = bson.M{"$exists": false}
		claimed, e := s.DB.Collection("quotes").UpdateOne(tx, qfilter, bson.M{"$set": bson.M{"confirmedOrderId": input.Order.OrderID}})
		if e != nil {
			return nil, e
		}
		if claimed.MatchedCount != 1 {
			return nil, d.Fail("QUOTE_INVALID", "quote expired or already confirmed")
		}
		if input.Quote.PaymentMethod == "CREDIT" {
			f := scopeFilter(input.Quote.Scope)
			f["available.currency"] = input.Order.Total.Currency
			f["available.scale"] = input.Order.Total.Scale
			f["available.amount"] = bson.M{"$gte": input.Order.Total.Amount}
			r, e := s.DB.Collection("credits").UpdateOne(tx, f, bson.M{"$inc": bson.M{"available.amount": -input.Order.Total.Amount}})
			if e != nil {
				return nil, e
			}
			if r.MatchedCount != 1 {
				return nil, d.Fail("CREDIT_INSUFFICIENT", "available credit is insufficient")
			}
		}
		if _, e = s.DB.Collection("orders").InsertOne(tx, input.Order); e != nil {
			return nil, e
		}
		if _, e = s.DB.Collection("idempotency").InsertOne(tx, input.Idempotency); e != nil {
			return nil, e
		}
		if _, e = s.DB.Collection("outbox_events").InsertOne(tx, input.Event); e != nil {
			return nil, e
		}
		return input.Order, nil
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	if e != nil {
		// Duplicate-key errors can occur after a concurrent transaction commits.
		var business *d.Error
		if driver.IsDuplicateKeyError(e) || errors.As(e, &business) {
			record, found, lookupErr := s.FindIdempotency(ctx, input.Idempotency.TenantID, input.Idempotency.Key)
			if lookupErr == nil && found {
				return application.Replay(record, input.Idempotency.RequestHash)
			}
		}
		return d.Order{}, fmt.Errorf("confirm transaction: %w", e)
	}
	return result.(d.Order), nil
}
