// Package notifier provides notification services like email and SMS.
package notifier

import (
	"context"
	"log/slog"
)

// LogNotifier is a provider that logs notifications to slog.
// Ideal for development and debugging.
type LogNotifier struct{}

// NewLogNotifier creates a new Logger-based notifier
func NewLogNotifier() *LogNotifier {
	return &LogNotifier{}
}

// SendEmail logs a simulated email message.
func (l *LogNotifier) SendEmail(ctx context.Context, msg Message) error {
	slog.InfoContext(ctx, "📧 [EMAIL SENT]",
		"to", msg.To,
		"subject", msg.Subject,
		"body", msg.Body,
	)

	return nil
}

// SendSMS logs a simulated SMS message.
func (l *LogNotifier) SendSMS(ctx context.Context, msg Message) error {
	slog.InfoContext(ctx, "📱 [SMS SENT]",
		"to", msg.To,
		"body", msg.Body,
	)

	return nil
}
