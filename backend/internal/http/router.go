package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"marketplace/internal/application"
	d "marketplace/internal/domain"
	"net/http"
	"time"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"traceId"`
}

func Router(service *application.Service, ready func(context.Context) error, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { write(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if e := ready(ctx); e != nil {
			write(w, 503, ErrorResponse{"NOT_READY", "dependencies unavailable", application.NewID()})
			return
		}
		write(w, 200, map[string]string{"status": "ready"})
	})
	handle := func(fn func(http.ResponseWriter, *http.Request, d.Scope) error) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			trace := application.NewID()
			w.Header().Set("X-Trace-ID", trace)
			scope := d.Scope{TenantID: r.Header.Get("X-Tenant-ID"), Country: r.Header.Get("X-Country"), CustomerID: r.Header.Get("X-Customer-ID")}
			if scope.TenantID == "" || scope.Country == "" || scope.CustomerID == "" {
				failure(w, d.Fail("INVALID_REQUEST", "X-Tenant-ID, X-Country and X-Customer-ID are required"), trace, log)
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()
			if e := fn(w, r.WithContext(ctx), scope); e != nil {
				failure(w, e, trace, log)
			}
		}
	}
	mux.HandleFunc("POST /quotes", handle(func(w http.ResponseWriter, r *http.Request, scope d.Scope) error {
		var cart d.Cart
		if e := decode(w, r, &cart); e != nil {
			return e
		}
		if cart.Scope != scope {
			return d.Fail("INVALID_REQUEST", "body scope must match headers")
		}
		q, e := service.Quote(r.Context(), cart)
		if e != nil {
			return e
		}
		w.Header().Set("Location", "/quotes/"+q.QuoteID)
		write(w, 201, q)
		return nil
	}))
	mux.HandleFunc("GET /quotes/{id}", handle(func(w http.ResponseWriter, r *http.Request, scope d.Scope) error {
		q, e := service.Quotes.GetQuote(r.Context(), scope, r.PathValue("id"))
		if e != nil {
			return e
		}
		write(w, 200, q)
		return nil
	}))
	mux.HandleFunc("POST /orders", handle(func(w http.ResponseWriter, r *http.Request, scope d.Scope) error {
		var req d.OrderRequest
		if e := decode(w, r, &req); e != nil {
			return e
		}
		order, e := service.Confirm(r.Context(), scope, r.Header.Get("Idempotency-Key"), req)
		if e != nil {
			return e
		}
		w.Header().Set("Location", "/orders/"+order.OrderID)
		write(w, 201, order)
		return nil
	}))
	mux.HandleFunc("GET /orders/{id}", handle(func(w http.ResponseWriter, r *http.Request, scope d.Scope) error {
		order, e := service.Orders.GetOrder(r.Context(), scope, r.PathValue("id"))
		if e != nil {
			return e
		}
		write(w, 200, order)
		return nil
	}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				log.Error("http panic", "error", v)
				failure(w, errors.New("panic"), application.NewID(), log)
			}
		}()
		mux.ServeHTTP(w, r)
	})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if e := decoder.Decode(v); e != nil {
		return d.Fail("INVALID_REQUEST", "invalid JSON body")
	}
	var extra any
	if e := decoder.Decode(&extra); e != io.EOF {
		return d.Fail("INVALID_REQUEST", "body must contain one JSON value")
	}
	return nil
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func failure(w http.ResponseWriter, e error, trace string, log *slog.Logger) {
	status := 500
	code, message := "INTERNAL_ERROR", "unexpected server error"
	var business *d.Error
	if errors.As(e, &business) {
		code, message = business.Code, business.Message
		switch code {
		case "INVALID_REQUEST":
			status = 400
		case "PRODUCT_NOT_FOUND", "QUOTE_NOT_FOUND", "ORDER_NOT_FOUND":
			status = 404
		case "IDEMPOTENCY_CONFLICT", "QUOTE_INVALID", "CREDIT_INSUFFICIENT":
			status = 409
		}
	}
	if status == 500 {
		log.Error("request failed", "traceId", trace, "error", e)
	}
	write(w, status, ErrorResponse{code, message, trace})
}
