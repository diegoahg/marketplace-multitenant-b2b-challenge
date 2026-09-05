package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"marketplace/internal/application"
	d "marketplace/internal/domain"
	api "marketplace/internal/http"
	"marketplace/internal/testkit"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPJourney(t *testing.T) {
	m := testkit.NewMemory()
	h := api.Router(m.Service(), func(context.Context) error { return nil }, slog.New(slog.NewTextHandler(io.Discard, nil)))
	call := func(method, path string, v any, key string) *httptest.ResponseRecorder {
		data, _ := json.Marshal(v)
		r := httptest.NewRequest(method, path, bytes.NewReader(data))
		r.Header.Set("X-Tenant-ID", "tenant-demo")
		r.Header.Set("X-Country", "PE")
		r.Header.Set("X-Customer-ID", "CUSTOMER-001")
		r.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	w := call("POST", "/quotes", testkit.Cart(d.CartItem{SKU: "A", Quantity: 2}), "")
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var q d.Quote
	if e := json.Unmarshal(w.Body.Bytes(), &q); e != nil {
		t.Fatal(e)
	}
	key := application.NewID()
	req := d.OrderRequest{QuoteID: q.QuoteID, CustomerID: q.CustomerID}
	w = call("POST", "/orders", req, key)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	first := w.Body.String()
	var o d.Order
	if e := json.Unmarshal(w.Body.Bytes(), &o); e != nil {
		t.Fatal(e)
	}
	if call("GET", "/orders/"+o.OrderID, nil, "").Code != 200 {
		t.Fatal("201 not readable")
	}
	w = call("POST", "/orders", req, key)
	if w.Code != 201 || w.Body.String() != first {
		t.Fatal("retry response changed")
	}
	req.QuoteID = "different"
	if call("POST", "/orders", req, key).Code != 409 {
		t.Fatal("expected conflict")
	}
	if call("POST", "/orders", req, "").Code != 400 {
		t.Fatal("missing key accepted")
	}
	if call("GET", "/quotes/"+q.QuoteID, nil, "").Code != 200 {
		t.Fatal("quote not readable")
	}
	if call("GET", "/health", nil, "").Code != http.StatusOK {
		t.Fatal("health")
	}
}
func TestInvalidJSONAndTenantHeaders(t *testing.T) {
	h := api.Router(testkit.NewMemory().Service(), func(context.Context) error { return nil }, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, body := range []string{"{} {}", "{invalid}", `{"unknown":true}`} {
		r := httptest.NewRequest("POST", "/quotes", bytes.NewBufferString(body))
		r.Header.Set("X-Tenant-ID", "tenant-demo")
		r.Header.Set("X-Country", "PE")
		r.Header.Set("X-Customer-ID", "CUSTOMER-001")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatal(w.Code)
		}
	}
	r := httptest.NewRequest("GET", "/orders/other", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatal("missing scope accepted")
	}
}
