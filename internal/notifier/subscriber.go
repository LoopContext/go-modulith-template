package notifier

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/cmelgarejo/go-modulith-template/internal/events"
)

// Event names used for the notification bus
const (
	EventMagicCodeRequested = "notifier.magic_code_requested"
)

// Subscriber handles notification events
type Subscriber struct {
	notifier Notifier
}

// NewSubscriber creates a new notifier event subscriber
func NewSubscriber(n Notifier) *Subscriber {
	return &Subscriber{notifier: n}
}

// SubscribeToEvents registers the subscriber to the event bus
func (s *Subscriber) SubscribeToEvents(bus *events.Bus) {
	bus.Subscribe(EventMagicCodeRequested, s.handleMagicCodeRequested)
}

func (s *Subscriber) handleMagicCodeRequested(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(map[string]string)
	if !ok {
		return fmt.Errorf("invalid payload type for magic code event: %T", e.Payload)
	}

	email := payload["email"]
	phone := payload["phone"]
	code := payload["code"]

	var err error
	if email != "" {
		err = s.notifier.SendEmail(ctx, Message{
			To:      email,
			Subject: "Tu Código de Acceso",
			Body:    "Tu código mágico es: " + code,
		})
	} else if phone != "" {
		err = s.notifier.SendSMS(ctx, Message{
			To:   phone,
			Body: "Tu código mágico es: " + code,
		})
	}

	if err != nil {
		slog.ErrorContext(ctx, "failed to send async notification", "error", err)
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}
