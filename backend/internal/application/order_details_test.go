package application_test

import (
	"context"
	"marketplace/internal/application"
	d "marketplace/internal/domain"
	"marketplace/internal/testkit"
	"testing"
)

type trackingStub struct{ calls int }

func (s *trackingStub) OrderDeliveries(_ context.Context, _ d.Scope, _ string) (map[string]d.DeliveryState, error) {
	s.calls++
	return map[string]d.DeliveryState{"erp": {Done: true}, "push": {Failures: 2}}, nil
}
func TestOrderDetailsKeepSnapshotAndIsolateCustomer(t *testing.T) {
	ctx := context.Background()
	memory := testkit.NewMemory()
	service := memory.Service()
	tracking := &trackingStub{}
	service.Tracking = tracking
	q, err := service.Quote(ctx, testkit.Cart(d.CartItem{SKU: "A", Quantity: 2}))
	if err != nil {
		t.Fatal(err)
	}
	order, err := service.Confirm(ctx, q.Scope, application.NewID(), d.OrderRequest{QuoteID: q.QuoteID, CustomerID: q.CustomerID})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := service.OrderDetails(ctx, q.Scope, order.OrderID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Quote.QuoteID != q.QuoteID || detail.Order.Total != q.Total || !detail.Deliveries["erp"].Done || detail.Deliveries["push"].Done || detail.Deliveries["push"].Failures != 2 {
		t.Fatalf("wrong detail: %+v", detail)
	}
	for _, scope := range []d.Scope{{TenantID: "other", Country: q.Country, CustomerID: q.CustomerID}, {TenantID: q.TenantID, Country: "other", CustomerID: q.CustomerID}, {TenantID: q.TenantID, Country: q.Country, CustomerID: "other"}} {
		if _, err = service.OrderDetails(ctx, scope, order.OrderID); err == nil {
			t.Fatal("cross-scope access allowed")
		}
	}
	if tracking.calls != 1 {
		t.Fatal("tracking accessed before ownership check")
	}
}
