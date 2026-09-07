package integration

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	d "marketplace/internal/domain"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

// Local SMTP sink for the demo. Production SMTP requires a TLS/auth adapter.
type Mail struct{ Address, From, To string }

func (m Mail) Send(ctx context.Context, letter d.DeadLetter) error {
	for _, address := range []string{m.From, m.To} {
		if strings.ContainsAny(address, "\r\n") {
			return fmt.Errorf("invalid mail address")
		}
		if _, err := mail.ParseAddress(address); err != nil {
			return err
		}
	}
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", m.Address)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err = conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	host, _, err := net.SplitHostPort(m.Address)
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if err = client.Mail(m.From); err != nil {
		return err
	}
	if err = client.Rcpt(m.To); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	body, err := json.MarshalIndent(letter, "", "  ")
	if err != nil {
		return err
	}
	messageID := sha256.Sum256([]byte(letter.ID))
	_, err = fmt.Fprintf(writer, "From: %s\r\nTo: %s\r\nSubject: MariposaMarket DLQ - delivery failed\r\nMessage-ID: <%x@mariposamarket.test>\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nDestination failed after initial attempt and five retries.\r\n%s\r\n", m.From, m.To, messageID, body)
	if err != nil {
		return err
	}
	return writer.Close()
}
