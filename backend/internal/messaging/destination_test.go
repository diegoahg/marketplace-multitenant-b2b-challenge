package messaging

import (
	"encoding/base64"
	"testing"
)

func TestMalformedDeliveryHasStableRetryKey(t *testing.T) {
	for _, raw := range []string{"{broken", `{"eventId":"11111111-1111-4111-8111-111111111111","eventType":"OrderConfirmed"} trailing`} {
		var msg Delivery
		msg.Message.Data = base64.StdEncoding.EncodeToString([]byte(raw))
		a, _ := decodeDelivery(msg)
		b, _ := decodeDelivery(msg)
		if a.EventID == "" || a.EventID != b.EventID || validateEvent(a) == nil {
			t.Fatal("malformed message escaped bounded retries", a)
		}
		if a.EventID == "11111111-1111-4111-8111-111111111111" {
			t.Fatal("invalid payload reuses the valid event's retry state")
		}
	}
}
