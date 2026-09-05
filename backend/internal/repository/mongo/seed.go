package mongo

import (
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"io"
	d "marketplace/internal/domain"
)

type SeedData struct {
	Countries  []d.CountryConfig `json:"countries"`
	Products   []d.Product       `json:"products"`
	Customers  []d.Customer      `json:"customers"`
	Promotions []d.Promotion     `json:"promotions"`
	Credits    []d.CreditAccount `json:"credits"`
}

func (s *Store) Seed(ctx context.Context, reader io.Reader) error {
	var data SeedData
	if e := json.NewDecoder(reader).Decode(&data); e != nil {
		return e
	}
	insert := func(collection string, filter bson.M, entity any) error {
		_, e := s.DB.Collection(collection).UpdateOne(ctx, filter, bson.M{"$setOnInsert": entity}, options.UpdateOne().SetUpsert(true))
		if e != nil {
			return fmt.Errorf("seed %s: %w", collection, e)
		}
		return nil
	}
	for _, v := range data.Countries {
		if e := insert("countries", bson.M{"tenantId": v.TenantID, "country": v.Country}, v); e != nil {
			return e
		}
	}
	for _, v := range data.Products {
		if e := insert("products", bson.M{"tenantId": v.TenantID, "country": v.Country, "sku": v.SKU}, v); e != nil {
			return e
		}
	}
	for _, v := range data.Customers {
		if e := insert("customers", scopeFilter(v.Scope), v); e != nil {
			return e
		}
	}
	for _, v := range data.Promotions {
		if e := insert("promotions", bson.M{"tenantId": v.TenantID, "country": v.Country, "id": v.ID}, v); e != nil {
			return e
		}
	}
	for _, v := range data.Credits {
		if e := insert("credits", scopeFilter(v.Scope), v); e != nil {
			return e
		}
	}
	return nil
}
