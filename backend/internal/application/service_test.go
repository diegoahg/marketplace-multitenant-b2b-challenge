package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"marketplace/internal/application"
	d "marketplace/internal/domain"
	"marketplace/internal/testkit"
	"os"
	"sync"
	"testing"
	"time"
)

func TestGT06Snapshot(t *testing.T) {
	m := testkit.NewMemory()
	s := m.Service()
	ctx := context.Background()
	q, e := s.Quote(ctx, testkit.Cart(d.CartItem{SKU: "A", Quantity: 2}))
	if e != nil {
		t.Fatal(e)
	}
	m.Products[0].UnitPrice.Amount = 99999
	m.Promotions = []d.Promotion{testkit.Scale()}
	o, e := s.Confirm(ctx, q.Scope, application.NewID(), d.OrderRequest{QuoteID: q.QuoteID, CustomerID: q.CustomerID})
	if e != nil {
		t.Fatal(e)
	}
	data, e := os.ReadFile("../../testdata/golden/GT06.json")
	if e != nil {
		t.Fatal(e)
	}
	var golden struct{ QuoteTotal, OrderTotal int64 }
	if e = json.Unmarshal(data, &golden); e != nil {
		t.Fatal(e)
	}
	if q.Total != o.Total || q.Total.Amount != golden.QuoteTotal || o.Total.Amount != golden.OrderTotal {
		t.Fatal(q.Total, o.Total)
	}
	if _, e = s.Orders.GetOrder(ctx, q.Scope, o.OrderID); e != nil {
		t.Fatal("201 must be immediately readable", e)
	}
}
func TestGT07ConcurrentIdempotency(t *testing.T) {
	m := testkit.NewMemory()
	s := m.Service()
	ctx := context.Background()
	q, e := s.Quote(ctx, testkit.Cart(d.CartItem{SKU: "A", Quantity: 2}))
	if e != nil {
		t.Fatal(e)
	}
	key := application.NewID()
	var wg sync.WaitGroup
	ids := make(chan string, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o, e := s.Confirm(ctx, q.Scope, key, d.OrderRequest{QuoteID: q.QuoteID, CustomerID: q.CustomerID})
			if e != nil {
				t.Error(e)
				return
			}
			ids <- o.OrderID
		}()
	}
	wg.Wait()
	close(ids)
	one := ""
	for id := range ids {
		if one != "" && one != id {
			t.Fatal("different orders")
		}
		one = id
	}
	if len(m.Orders) != 1 || len(m.Events) != 1 || m.Credit.Amount != 100000-q.Total.Amount {
		t.Fatal("duplicate side effects")
	}
}
func TestGT08ConcurrentCredit(t *testing.T) {
	m := testkit.NewMemory()
	m.Credit.Amount = 1000
	s := m.Service()
	ctx := context.Background()
	quotes := []d.Quote{}
	for _, n := range []int64{7, 5} {
		q, e := s.Quote(ctx, testkit.Cart(d.CartItem{SKU: "A", Quantity: n}))
		if e != nil {
			t.Fatal(e)
		}
		quotes = append(quotes, q)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, q := range quotes {
		wg.Add(1)
		go func(q d.Quote) {
			defer wg.Done()
			_, e := s.Confirm(ctx, q.Scope, application.NewID(), d.OrderRequest{QuoteID: q.QuoteID, CustomerID: q.CustomerID})
			results <- e
		}(q)
	}
	wg.Wait()
	close(results)
	success := 0
	for e := range results {
		if e == nil {
			success++
		} else {
			var de *d.Error
			if !errors.As(e, &de) || de.Code != "CREDIT_INSUFFICIENT" {
				t.Fatal(e)
			}
		}
	}
	if success != 1 || len(m.Orders) != 1 || len(m.Events) != 1 || m.Credit.Amount < 0 {
		t.Fatal("overspending")
	}
}
func TestGT11NoEventOnRejection(t *testing.T) {
	m := testkit.NewMemory()
	m.Credit.Amount = 1
	s := m.Service()
	q, e := s.Quote(context.Background(), testkit.Cart(d.CartItem{SKU: "A", Quantity: 2}))
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.Confirm(context.Background(), q.Scope, application.NewID(), d.OrderRequest{QuoteID: q.QuoteID, CustomerID: q.CustomerID})
	if e == nil || len(m.Orders) != 0 || len(m.Events) != 0 || m.Credit.Amount != 1 {
		t.Fatal("rejected order caused effects")
	}
}
func TestConflictExpirationAndScope(t *testing.T) {
	m := testkit.NewMemory()
	s := m.Service()
	ctx := context.Background()
	q, e := s.Quote(ctx, testkit.Cart(d.CartItem{SKU: "A", Quantity: 1}))
	if e != nil {
		t.Fatal(e)
	}
	key := application.NewID()
	req := d.OrderRequest{QuoteID: q.QuoteID, CustomerID: q.CustomerID}
	first, e := s.Confirm(ctx, q.Scope, key, req)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.Confirm(ctx, q.Scope, application.NewID(), req)
	if e == nil {
		t.Fatal("same quote bought twice")
	}
	req.QuoteID = "other"
	_, e = s.Confirm(ctx, q.Scope, key, req)
	var de *d.Error
	if !errors.As(e, &de) || de.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatal(e)
	}
	s.Now = func() time.Time { return testkit.Now().Add(time.Hour) }
	req.QuoteID = q.QuoteID
	retry, e := s.Confirm(ctx, q.Scope, key, req)
	if e != nil || retry.OrderID != first.OrderID {
		t.Fatal("expired retry must replay", e)
	}
	q2, e := m.Service().Quote(ctx, testkit.Cart(d.CartItem{SKU: "A", Quantity: 1}))
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.Confirm(ctx, q2.Scope, application.NewID(), d.OrderRequest{QuoteID: q2.QuoteID, CustomerID: q2.CustomerID})
	if e == nil {
		t.Fatal("expired quote confirmed")
	}
	scope := q.Scope
	scope.TenantID = "other"
	if _, e = s.Quotes.GetQuote(ctx, scope, q.QuoteID); e == nil {
		t.Fatal("cross tenant read")
	}
	if _, e = s.Orders.GetOrder(ctx, scope, first.OrderID); e == nil {
		t.Fatal("cross tenant order")
	}
}
func TestConcurrentQuotes(t *testing.T) {
	s := testkit.NewMemory().Service()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := s.Quote(context.Background(), testkit.Cart(d.CartItem{SKU: "A", Quantity: 12})); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
}
