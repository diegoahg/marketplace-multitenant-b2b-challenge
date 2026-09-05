package messaging

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"marketplace/internal/application"
	d "marketplace/internal/domain"
	"marketplace/internal/ports"
	"time"
)

type Worker struct {
	Effects ports.Effects
	ERP     ports.ERPClient
	Push    ports.NotificationClient
}

func (w *Worker) Handle(ctx context.Context, event d.OrderConfirmedEvent) error {
	if !application.ValidUUID(event.EventID) || event.EventType != "OrderConfirmed" || event.EventVersion != 1 || event.Order.Status != "CONFIRMED" || event.TenantID != event.Order.TenantID || event.Country != event.Order.Country || !application.ValidUUID(event.Order.OrderID) {
		return fmt.Errorf("invalid OrderConfirmed event")
	}
	if event.TenantID == "" || event.Country == "" || event.Order.CustomerID == "" || event.Order.QuoteID == "" || event.Order.OrderNumber == "" || event.OccurredAt.IsZero() || event.Order.CreatedAt.IsZero() || !event.Order.Total.Valid() || event.Order.Total.Amount < 0 || event.Order.Currency != event.Order.Total.Currency {
		return fmt.Errorf("invalid OrderConfirmed event")
	}
	for _, effect := range []struct {
		name string
		send func(context.Context, d.OrderConfirmedEvent) error
	}{{"erp", w.ERP.SendOrder}, {"push", w.Push.SendOrderConfirmed}} {
		owner := application.NewID()
		claim, e := w.Effects.ClaimEffect(ctx, event.EventID, effect.name, owner, time.Now().UTC())
		if e != nil {
			return e
		}
		if !claim {
			continue
		}
		effectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		e = effect.send(effectCtx, event)
		cancel()
		if e != nil {
			cleanup, done := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
			_ = w.Effects.ReleaseEffect(cleanup, event.EventID, effect.name, owner)
			done()
			return fmt.Errorf("%s: %w", effect.name, e)
		}
		if e = w.Effects.CompleteEffect(ctx, event.EventID, effect.name, owner); e != nil {
			return e
		}
	}
	return nil
}
func DispatchOnce(ctx context.Context, outbox ports.Outbox, publisher ports.Publisher) error {
	events, e := outbox.Pending(ctx, 50)
	if e != nil {
		return e
	}
	for _, event := range events {
		if e = publisher.Publish(ctx, event); e != nil {
			return e
		}
		if e = outbox.MarkPublished(ctx, event.EventID); e != nil {
			return e
		}
	}
	return nil
}
func RunPublisher(ctx context.Context, outbox ports.Outbox, publisher ports.Publisher, log *slog.Logger) {
	for ctx.Err() == nil {
		if e := DispatchOnce(ctx, outbox, publisher); e != nil && ctx.Err() == nil {
			log.Error("outbox retry", "error", e)
		}
		if !pause(ctx, time.Second) {
			return
		}
	}
}
func RunConsumer(ctx context.Context, p *Emulator, w *Worker, log *slog.Logger) {
	for ctx.Err() == nil {
		deliveries, e := p.Pull(ctx)
		if e != nil {
			if ctx.Err() == nil {
				log.Error("pubsub pull retry", "error", e)
			}
			if !pause(ctx, time.Second) {
				return
			}
			continue
		}
		for _, msg := range deliveries {
			bytes, err := base64.StdEncoding.DecodeString(msg.Message.Data)
			var event d.OrderConfirmedEvent
			if err == nil {
				err = json.Unmarshal(bytes, &event)
			}
			if err == nil {
				err = w.Handle(ctx, event)
			}
			if err != nil {
				log.Error("event processing retry", "eventId", event.EventID, "error", err)
				if nackErr := p.Nack(ctx, msg.AckID); nackErr != nil && ctx.Err() == nil {
					log.Error("nack failed", "error", nackErr)
				}
				if !pause(ctx, time.Second) {
					return
				}
				continue
			}
			if e = p.Ack(ctx, msg.AckID); e != nil && ctx.Err() == nil {
				log.Error("ack failed; redelivery is safe", "error", e)
			}
		}
		if len(deliveries) == 0 && !pause(ctx, 200*time.Millisecond) {
			return
		}
	}
}
func pause(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
