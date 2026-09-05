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
	d "marketplace/internal/domain"
	"time"
)

type Store struct {
	Client *driver.Client
	DB     *driver.Database
}

func Connect(ctx context.Context, uri, database string) (*Store, error) {
	client, e := driver.Connect(options.Client().ApplyURI(uri).SetServerSelectionTimeout(5 * time.Second))
	if e != nil {
		return nil, e
	}
	if e = client.Ping(ctx, nil); e != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo ping: %w", e)
	}
	return &Store{client, client.Database(database, options.Database().SetReadConcern(readconcern.Majority()).SetWriteConcern(writeconcern.Majority()))}, nil
}
func (s *Store) Close(ctx context.Context) error { return s.Client.Disconnect(ctx) }
func (s *Store) Ping(ctx context.Context) error  { return s.Client.Ping(ctx, nil) }
func scopeFilter(scope d.Scope) bson.M {
	return bson.M{"tenantId": scope.TenantID, "country": scope.Country, "customerId": scope.CustomerID}
}
func unique(keys ...string) driver.IndexModel {
	k := bson.D{}
	for _, key := range keys {
		k = append(k, bson.E{Key: key, Value: 1})
	}
	return driver.IndexModel{Keys: k, Options: options.Index().SetUnique(true)}
}
func (s *Store) Init(ctx context.Context) error {
	indices := map[string][]driver.IndexModel{
		"countries": {unique("tenantId", "country")}, "products": {unique("tenantId", "country", "sku")},
		"customers": {unique("tenantId", "country", "customerId")}, "promotions": {unique("tenantId", "country", "id")},
		"credits": {unique("tenantId", "country", "customerId")}, "quotes": {unique("tenantId", "country", "quoteId")},
		"orders":      {unique("tenantId", "orderId"), unique("tenantId", "country", "quoteId")},
		"idempotency": {unique("tenantId", "key")}, "processed_events": {unique("eventId")},
		"outbox_events": {unique("eventId"), {Keys: bson.D{{Key: "publishedAt", Value: 1}, {Key: "occurredAt", Value: 1}}}},
	}
	for collection, models := range indices {
		if _, e := s.DB.Collection(collection).Indexes().CreateMany(ctx, models); e != nil {
			return fmt.Errorf("index %s: %w", collection, e)
		}
	}
	return nil
}
func (s *Store) Catalog(ctx context.Context, scope d.Scope) (cfg d.CountryConfig, products []d.Product, promos []d.Promotion, credit d.Money, err error) {
	err = s.DB.Collection("customers").FindOne(ctx, scopeFilter(scope)).Err()
	if errors.Is(err, driver.ErrNoDocuments) {
		err = d.Fail("INVALID_REQUEST", "customer not found in tenant/country")
	}
	if err != nil {
		return
	}
	filter := bson.M{"tenantId": scope.TenantID, "country": scope.Country}
	err = s.DB.Collection("countries").FindOne(ctx, filter).Decode(&cfg)
	if errors.Is(err, driver.ErrNoDocuments) {
		err = d.Fail("INVALID_REQUEST", "country not configured")
	}
	if err != nil {
		return
	}
	products, err = findAll[d.Product](ctx, s.DB.Collection("products"), filter)
	if err != nil {
		return
	}
	promos, err = findAll[d.Promotion](ctx, s.DB.Collection("promotions"), filter)
	if err != nil {
		return
	}
	var account d.CreditAccount
	err = s.DB.Collection("credits").FindOne(ctx, scopeFilter(scope)).Decode(&account)
	if errors.Is(err, driver.ErrNoDocuments) {
		credit = cfg.Money(0)
		err = nil
		return
	}
	credit = account.Available
	return
}
func findAll[T any](ctx context.Context, col *driver.Collection, filter any) ([]T, error) {
	cur, e := col.Find(ctx, filter)
	if e != nil {
		return nil, e
	}
	defer cur.Close(ctx)
	out := []T{}
	e = cur.All(ctx, &out)
	return out, e
}
func (s *Store) SaveQuote(ctx context.Context, q d.Quote) error {
	_, e := s.DB.Collection("quotes").InsertOne(ctx, q)
	return e
}
func (s *Store) GetQuote(ctx context.Context, scope d.Scope, id string) (q d.Quote, e error) {
	f := scopeFilter(scope)
	f["quoteId"] = id
	e = s.DB.Collection("quotes").FindOne(ctx, f).Decode(&q)
	if errors.Is(e, driver.ErrNoDocuments) {
		e = d.Fail("QUOTE_NOT_FOUND", "quote not found")
	}
	return
}
func (s *Store) GetOrder(ctx context.Context, scope d.Scope, id string) (o d.Order, e error) {
	f := scopeFilter(scope)
	f["orderId"] = id
	e = s.DB.Collection("orders").FindOne(ctx, f).Decode(&o)
	if errors.Is(e, driver.ErrNoDocuments) {
		e = d.Fail("ORDER_NOT_FOUND", "order not found")
	}
	return
}
func (s *Store) FindIdempotency(ctx context.Context, tenant, key string) (r d.Idempotency, found bool, e error) {
	e = s.DB.Collection("idempotency").FindOne(ctx, bson.M{"tenantId": tenant, "key": key}).Decode(&r)
	if errors.Is(e, driver.ErrNoDocuments) {
		return r, false, nil
	}
	return r, e == nil, e
}
