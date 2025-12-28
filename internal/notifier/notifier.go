package notifier

import "context"

// Message represents a generic notification content
type Message struct {
	To      string
	Subject string
	Body    string
	Data    map[string]interface{}
}

// EmailProvider defines the interface for sending emails
type EmailProvider interface {
	SendEmail(ctx context.Context, msg Message) error
}

// SMSProvider defines the interface for sending SMS or WhatsApp messages
type SMSProvider interface {
	SendSMS(ctx context.Context, msg Message) error
}

// Notifier combines multiple notification capabilities
type Notifier interface {
	EmailProvider
	SMSProvider
}
