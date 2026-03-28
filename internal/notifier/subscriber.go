package notifier

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/LoopContext/go-modulith-template/internal/events"
	"github.com/LoopContext/go-modulith-template/internal/i18n"
)

// Event names used for the notification bus
const (
	EventMagicCodeRequested = "notifier.magic_code_requested"
)

// Subscriber handles notification events
type Subscriber struct {
	notifier      Notifier
	defaultLocale string
}

// NewSubscriber creates a new notifier event subscriber
func NewSubscriber(n Notifier, defaultLocale string) *Subscriber {
	return &Subscriber{
		notifier:      n,
		defaultLocale: defaultLocale,
	}
}

// SubscribeToEvents registers the subscriber to the event bus
func (s *Subscriber) SubscribeToEvents(bus *events.Bus) {
	bus.Subscribe(EventMagicCodeRequested, s.handleMagicCodeRequested)
}

func (s *Subscriber) handleMagicCodeRequested(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload type for magic code event: %T", e.Payload)
	}

	// Extract string values from payload
	email, _ := payload["email"].(string)
	phone, _ := payload["phone"].(string)
	code, _ := payload["code"].(string)

	// Extract locale from payload if available, otherwise detect from context
	var locale string
	if localeStr, ok := payload["locale"].(string); ok && localeStr != "" {
		locale = localeStr
		// Inject locale into context for translation
		ctx = i18n.ContextWithLocale(ctx, locale)
	} else {
		// Try to detect locale from context
		locale = i18n.DetectLocale(ctx, s.defaultLocale)
		ctx = i18n.ContextWithLocale(ctx, locale)
	}

	// Translate notification messages
	subject := i18n.T(ctx, s.defaultLocale, "notifications.magic_code_subject", nil)
	body := i18n.T(ctx, s.defaultLocale, "notifications.magic_code_body", map[string]interface{}{
		"Code": code,
	})

	var err error
	if email != "" {
		err = s.notifier.SendEmail(ctx, Message{
			To:      email,
			Subject: subject,
			Body:    body,
		})
	} else if phone != "" {
		err = s.notifier.SendSMS(ctx, Message{
			To:   phone,
			Body: body,
		})
	}

	if err != nil {
		slog.ErrorContext(ctx, "failed to send async notification", "error", err)
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}
