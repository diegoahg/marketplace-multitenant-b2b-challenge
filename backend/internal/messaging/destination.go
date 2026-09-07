package messaging

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"marketplace/internal/application"
	d "marketplace/internal/domain"
	"marketplace/internal/ports"
	"time"
)

type Destination struct {
	Kind  string
	Store ports.Deliveries
	Send  func(context.Context, d.OrderConfirmedEvent) error
	Now   func() time.Time
}

func RetryDelay(failures int) time.Duration {
	if failures < 1 {
		failures = 1
	}
	if failures > 5 {
		failures = 5
	}
	return time.Second * time.Duration(1<<(failures-1))
}

// Process returns whether it is safe to ACK and when an unacknowledged delivery is due.
func (w *Destination) Process(ctx context.Context, event d.OrderConfirmedEvent, raw string) (bool, time.Duration, error) {
	now := w.Now().UTC()
	state, err := w.Store.DeliveryState(ctx, event.EventID, w.Kind)
	if err != nil {
		return false, time.Second, err
	}
	if state.Done || state.Dead {
		return true, 0, nil
	}
	if state.NextAttempt.After(now) {
		return false, state.NextAttempt.Sub(now), nil
	}
	owner := application.NewID()
	claimed, err := w.Store.ClaimEffect(ctx, event.EventID, w.Kind, owner, now)
	if err != nil || !claimed {
		return false, time.Second, err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		_ = w.Store.ReleaseEffect(cleanup, event.EventID, w.Kind, owner)
	}()
	// Read again under the lease: a competing delivery may have changed retry state.
	state, err = w.Store.DeliveryState(ctx, event.EventID, w.Kind)
	if err != nil {
		return false, time.Second, err
	}
	if state.Done || state.Dead {
		return true, 0, nil
	}
	if state.NextAttempt.After(now) {
		return false, state.NextAttempt.Sub(now), nil
	}
	err = validateEvent(event)
	if err == nil {
		effectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err = w.Send(effectCtx, event)
		cancel()
	}
	if ctx.Err() != nil {
		return false, time.Second, ctx.Err()
	}
	if err == nil {
		err = w.Store.CompleteEffect(ctx, event.EventID, w.Kind, owner)
		return err == nil, time.Second, err
	}
	failures := state.Failures + 1
	terminal := failures >= 6 // Initial execution + five retries.
	delay := RetryDelay(failures)
	letter := d.DeadLetter{ID: event.EventID + ":" + w.Kind, Event: event, Destination: w.Kind, Error: err.Error(), Raw: raw, Attempts: failures, CreatedAt: w.Now().UTC()}
	if saveErr := w.Store.FailDelivery(ctx, letter, owner, w.Now().UTC().Add(delay), terminal); saveErr != nil {
		return false, time.Second, saveErr
	}
	return terminal, delay, err
}

func decodeDelivery(msg Delivery) (d.OrderConfirmedEvent, string) {
	bytes, err := base64.StdEncoding.DecodeString(msg.Message.Data)
	var event d.OrderConfirmedEvent
	if err == nil {
		err = json.Unmarshal(bytes, &event)
	}
	if err != nil || validateEvent(event) != nil {
		// Invalid payloads have their own stable key, even if they copy a real
		// eventId. Their retries/DLQ must not poison a valid event's progress.
		sum := sha256.Sum256([]byte(msg.Message.Data))
		event.EventID = fmt.Sprintf("%x-%x-%x-%x-%x", sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
		event.EventType = "InvalidPayload"
	}
	return event, msg.Message.Data
}

func RunDestination(ctx context.Context, p *Emulator, w *Destination, log *slog.Logger) {
	for ctx.Err() == nil {
		messages, err := p.Pull(ctx)
		if err != nil {
			log.Error("subscription retry", "destination", w.Kind, "error", err)
			if !pause(ctx, time.Second) {
				return
			}
			continue
		}
		for _, msg := range messages {
			event, raw := decodeDelivery(msg)
			ack, delay, err := w.Process(ctx, event, raw)
			if err != nil {
				log.Warn("destination failed", "destination", w.Kind, "eventId", event.EventID, "error", err, "terminal", ack)
			}
			if ack {
				err = p.Ack(ctx, msg.AckID)
			} else {
				err = p.Defer(ctx, msg.AckID, delay)
			}
			if err != nil && ctx.Err() == nil {
				log.Error("delivery disposition failed", "error", err)
			}
		}
		if len(messages) == 0 && !pause(ctx, 200*time.Millisecond) {
			return
		}
	}
}

// Create both subscriptions before publishing, including after emulator resource loss.
type FanoutPublisher struct{ ERP, Push *Emulator }

func (p FanoutPublisher) Publish(ctx context.Context, event d.OrderConfirmedEvent) error {
	if err := p.ERP.Setup(ctx); err != nil {
		return err
	}
	if err := p.Push.Setup(ctx); err != nil {
		return err
	}
	return p.ERP.Publish(ctx, event)
}

func AlertOnce(ctx context.Context, queue ports.DeadLetters, sender ports.AlertSender) error {
	owner := application.NewID()
	letter, found, err := queue.ClaimAlert(ctx, owner, time.Now().UTC())
	if err != nil || !found {
		return err
	}
	err = sender.Send(ctx, letter)
	message := ""
	if err != nil {
		message = err.Error()
	}
	if finishErr := queue.FinishAlert(ctx, letter.ID, owner, err == nil, message); finishErr != nil {
		return finishErr
	}
	return err
}
func RunAlerts(ctx context.Context, queue ports.DeadLetters, sender ports.AlertSender, log *slog.Logger) {
	for ctx.Err() == nil {
		if err := AlertOnce(ctx, queue, sender); err != nil && ctx.Err() == nil {
			log.Error("DLQ email retry", "error", err)
		}
		if !pause(ctx, time.Second) {
			return
		}
	}
}
