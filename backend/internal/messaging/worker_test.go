package messaging_test

import (
	"context"
	"errors"
	"marketplace/internal/application"
	d "marketplace/internal/domain"
	"marketplace/internal/messaging"
	"marketplace/internal/testkit"
	"sync"
	"testing"
)

type target struct {
	mu    sync.Mutex
	calls int
	fail  bool
}

func (t *target) SendOrder(ctx context.Context, e d.OrderConfirmedEvent) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.fail {
		return errors.New("offline")
	}
	t.calls++
	return nil
}
func (t *target) SendOrderConfirmed(ctx context.Context, e d.OrderConfirmedEvent) error {
	return t.SendOrder(ctx, e)
}
func event(t *testing.T) (*testkit.Memory, d.OrderConfirmedEvent) {
	t.Helper()
	m := testkit.NewMemory()
	s := m.Service()
	q, e := s.Quote(context.Background(), testkit.Cart(d.CartItem{SKU: "A", Quantity: 1}))
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.Confirm(context.Background(), q.Scope, application.NewID(), d.OrderRequest{QuoteID: q.QuoteID, CustomerID: q.CustomerID})
	if e != nil {
		t.Fatal(e)
	}
	for _, v := range m.Events {
		return m, v
	}
	t.Fatal("missing event")
	return nil, d.OrderConfirmedEvent{}
}
func TestGT12RedeliveryAndPartialFailure(t *testing.T) {
	m, e := event(t)
	erp, push := &target{}, &target{fail: true}
	w := &messaging.Worker{Effects: m, ERP: erp, Push: push}
	if w.Handle(context.Background(), e) == nil {
		t.Fatal("failure ignored")
	}
	if erp.calls != 1 {
		t.Fatal("ERP missing")
	}
	push.fail = false
	for i := 0; i < 100; i++ {
		if err := w.Handle(context.Background(), e); err != nil {
			t.Fatal(err)
		}
	}
	if erp.calls != 1 || push.calls != 1 {
		t.Fatal("duplicate external effects")
	}
	if len(m.Orders) != 1 {
		t.Fatal("worker changed order")
	}
}
func TestConcurrentRedelivery(t *testing.T) {
	m, e := event(t)
	erp, push := &target{}, &target{}
	w := &messaging.Worker{Effects: m, ERP: erp, Push: push}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = w.Handle(context.Background(), e) }()
	}
	wg.Wait()
	if err := w.Handle(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if erp.calls != 1 || push.calls != 1 {
		t.Fatal("duplicate effects")
	}
}

type publisher struct {
	fail   bool
	events []d.OrderConfirmedEvent
}

func (p *publisher) Publish(ctx context.Context, e d.OrderConfirmedEvent) error {
	if p.fail {
		return errors.New("pubsub down")
	}
	p.events = append(p.events, e)
	return nil
}
func TestOutboxRecovery(t *testing.T) {
	m, _ := event(t)
	p := &publisher{fail: true}
	if messaging.DispatchOnce(context.Background(), m, p) == nil {
		t.Fatal("error ignored")
	}
	if len(m.Published) != 0 {
		t.Fatal("lost event")
	}
	p.fail = false
	if e := messaging.DispatchOnce(context.Background(), m, p); e != nil {
		t.Fatal(e)
	}
	if e := messaging.DispatchOnce(context.Background(), m, p); e != nil {
		t.Fatal(e)
	}
	if len(p.events) != 1 || len(m.Published) != 1 {
		t.Fatal("outbox incorrect")
	}
}
func TestRejectInvalidEvent(t *testing.T) {
	m, e := event(t)
	e.Order.Status = "PENDING"
	w := &messaging.Worker{Effects: m, ERP: &target{}, Push: &target{}}
	if w.Handle(context.Background(), e) == nil {
		t.Fatal("unconfirmed event accepted")
	}
}

func TestSustainedMessages(t *testing.T) {
	m, template := event(t)
	erp, push := &target{}, &target{}
	w := &messaging.Worker{Effects: m, ERP: erp, Push: push}
	for i := 0; i < 200; i++ {
		e := template
		e.EventID = application.NewID()
		e.Order.OrderID = application.NewID()
		if err := w.Handle(context.Background(), e); err != nil { t.Fatal(err) }
		if err := w.Handle(context.Background(), e); err != nil { t.Fatal(err) }
	}
	if erp.calls != 200 || push.calls != 200 { t.Fatal("lost or duplicated effects") }
}
