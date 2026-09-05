package integration

import (
	"context"
	d "marketplace/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPIdempotencyKeyAndErrors(t *testing.T) {
	status := 503
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Idempotency-Key") != "event:erp" {
			t.Error("missing recipient idempotency key")
		}
		w.WriteHeader(status)
	}))
	defer server.Close()
	c := &Client{Kind: "erp", URL: server.URL, HTTP: server.Client()}
	event := d.OrderConfirmedEvent{EventID: "event"}
	if c.SendOrder(context.Background(), event) == nil {
		t.Fatal("503 ignored")
	}
	status = 204
	if e := c.SendOrder(context.Background(), event); e != nil {
		t.Fatal(e)
	}
}
