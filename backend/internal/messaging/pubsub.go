package messaging

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	d "marketplace/internal/domain"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Emulator uses the Pub/Sub REST protocol. It deliberately has no cloud credentials.
type Emulator struct {
	BaseURL, Topic, Subscription string
	Client                       *http.Client
}
type responseError struct {
	status  int
	message string
}

func (e *responseError) Error() string { return e.message }

// The emulator loses its topic/subscription on restart. Recreate them on 404;
// outbox reconciliation supplies any previously acknowledged but lost messages.
func (p *Emulator) recoverCall(ctx context.Context, path string, input, output any) error {
	err := p.call(ctx, http.MethodPost, path, input, output, false)
	var failure *responseError
	if !errors.As(err, &failure) || failure.status != http.StatusNotFound {
		return err
	}
	if err = p.Setup(ctx); err != nil {
		return err
	}
	return p.call(ctx, http.MethodPost, path, input, output, false)
}

func NewEmulator(host, project, topic, subscription string) (*Emulator, error) {
	if host == "" || strings.Contains(host, "://") {
		return nil, fmt.Errorf("PUBSUB_EMULATOR_HOST must be host:port")
	}
	for _, name := range []string{project, topic, subscription} {
		if name == "" || strings.ContainsAny(name, "/?#") {
			return nil, fmt.Errorf("invalid Pub/Sub resource name")
		}
	}
	return &Emulator{BaseURL: "http://" + host, Topic: "projects/" + project + "/topics/" + topic, Subscription: "projects/" + project + "/subscriptions/" + subscription, Client: &http.Client{Timeout: 15 * time.Second}}, nil
}
func (p *Emulator) call(ctx context.Context, method, path string, input, output any, allowConflict bool) error {
	data, e := json.Marshal(input)
	if e != nil {
		return e
	}
	req, e := http.NewRequestWithContext(ctx, method, p.BaseURL+"/v1/"+path, bytes.NewReader(data))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/json")
	res, e := p.Client.Do(req)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	if allowConflict && res.StatusCode == http.StatusConflict {
		return nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return &responseError{status: res.StatusCode, message: fmt.Sprintf("pubsub %s: HTTP %d: %s", path, res.StatusCode, body)}
	}
	if output != nil {
		return json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(output)
	}
	_, e = io.Copy(io.Discard, res.Body)
	return e
}
func (p *Emulator) Setup(ctx context.Context) error {
	if _, e := url.ParseRequestURI(p.BaseURL); e != nil {
		return e
	}
	if e := p.call(ctx, http.MethodPut, p.Topic, map[string]any{}, nil, true); e != nil {
		return e
	}
	return p.call(ctx, http.MethodPut, p.Subscription, map[string]any{"topic": p.Topic, "ackDeadlineSeconds": 60}, nil, true)
}
func (p *Emulator) Ready(ctx context.Context) error {
	return p.call(ctx, http.MethodGet, p.Subscription, nil, nil, false)
}
func (p *Emulator) Publish(ctx context.Context, event d.OrderConfirmedEvent) error {
	b, e := json.Marshal(event)
	if e != nil {
		return e
	}
	return p.recoverCall(ctx, p.Topic+":publish", map[string]any{"messages": []any{map[string]any{"data": base64.StdEncoding.EncodeToString(b), "attributes": map[string]string{"eventId": event.EventID, "eventType": event.EventType}}}}, nil)
}

type Delivery struct {
	AckID   string `json:"ackId"`
	Message struct {
		Data string `json:"data"`
	} `json:"message"`
}

func (p *Emulator) Pull(ctx context.Context) ([]Delivery, error) {
	var out struct {
		Received []Delivery `json:"receivedMessages"`
	}
	e := p.recoverCall(ctx, p.Subscription+":pull", map[string]any{"maxMessages": 1, "returnImmediately": true}, &out)
	return out.Received, e
}
func (p *Emulator) Ack(ctx context.Context, id string) error {
	return p.call(ctx, http.MethodPost, p.Subscription+":acknowledge", map[string]any{"ackIds": []string{id}}, nil, false)
}
func (p *Emulator) Nack(ctx context.Context, id string) error {
	return p.call(ctx, http.MethodPost, p.Subscription+":modifyAckDeadline", map[string]any{"ackIds": []string{id}, "ackDeadlineSeconds": 0}, nil, false)
}

func (p *Emulator) Defer(ctx context.Context, id string, delay time.Duration) error {
	seconds := int((delay + time.Second - 1) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	if seconds > 600 {
		seconds = 600
	}
	return p.call(ctx, http.MethodPost, p.Subscription+":modifyAckDeadline", map[string]any{"ackIds": []string{id}, "ackDeadlineSeconds": seconds}, nil, false)
}
