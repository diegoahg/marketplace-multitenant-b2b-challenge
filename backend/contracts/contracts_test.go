package contracts_test

import (
	"encoding/json"
	d "marketplace/internal/domain"
	"marketplace/internal/pricing"
	mongo "marketplace/internal/repository/mongo"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestQuoteExampleMatchesEngineAndSeed(t *testing.T) {
	read := func(path string, v any) {
		t.Helper()
		b, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		if e = json.Unmarshal(b, v); e != nil {
			t.Fatal(e)
		}
	}
	var seed mongo.SeedData
	read("../testdata/seed.json", &seed)
	var cart d.Cart
	read("http/quote-request.json", &cart)
	var want d.Quote
	read("http/quote-response.json", &want)
	got, e := pricing.Calculate(cart, seed.Countries[0], seed.Products, seed.Promotions, seed.Credits[0].Available, want.CreatedAt)
	if e != nil {
		t.Fatal(e)
	}
	got.QuoteID = want.QuoteID
	got.ExpiresAt = got.CreatedAt.Add(15 * time.Minute)
	if !reflect.DeepEqual(got, want) {
		a, _ := json.Marshal(got)
		b, _ := json.Marshal(want)
		t.Fatalf("example differs from implementation\ngot %s\nwant %s", a, b)
	}
}
func TestJSONFilesAndOrderEventAgree(t *testing.T) {
	e := filepath.WalkDir(".", func(path string, entry os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		bytes, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		var v any
		return json.Unmarshal(bytes, &v)
	})
	if e != nil {
		t.Fatal(e)
	}
	bytes, e := os.ReadFile("events/order-confirmed.example.json")
	if e != nil {
		t.Fatal(e)
	}
	var event d.OrderConfirmedEvent
	if e = json.Unmarshal(bytes, &event); e != nil {
		t.Fatal(e)
	}
	bytes, e = os.ReadFile("http/order-response.json")
	if e != nil {
		t.Fatal(e)
	}
	var order d.Order
	if e = json.Unmarshal(bytes, &order); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(event.Order, order) {
		t.Fatal("event and order response disagree")
	}
}
