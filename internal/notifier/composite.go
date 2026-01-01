package notifier

import (
	"context"
	"fmt"
	"log/slog"
)

// CompositeNotifier combines multiple providers into a single Notifier.
// It allows using different providers for email and SMS, with fallbacks.
type CompositeNotifier struct {
	emailProviders []EmailProvider
	smsProviders   []SMSProvider
	templates      *TemplateManager
}

// CompositeConfig holds configuration for creating a CompositeNotifier.
type CompositeConfig struct {
	EmailProviders []EmailProvider
	SMSProviders   []SMSProvider
	Templates      *TemplateManager
}

// NewCompositeNotifier creates a new CompositeNotifier with the given providers.
// Providers are tried in order; if one fails, the next is attempted.
func NewCompositeNotifier(cfg CompositeConfig) *CompositeNotifier {
	templates := cfg.Templates
	if templates == nil {
		templates = NewTemplateManager()
	}

	return &CompositeNotifier{
		emailProviders: cfg.EmailProviders,
		smsProviders:   cfg.SMSProviders,
		templates:      templates,
	}
}

// SendEmail attempts to send an email using configured providers.
// Providers are tried in order until one succeeds.
func (c *CompositeNotifier) SendEmail(ctx context.Context, msg Message) error {
	if len(c.emailProviders) == 0 {
		slog.WarnContext(ctx, "no email providers configured, email not sent", "to", msg.To)

		return nil
	}

	var lastErr error

	for i, provider := range c.emailProviders {
		if err := provider.SendEmail(ctx, msg); err != nil {
			slog.WarnContext(ctx, "email provider failed",
				"provider", i,
				"to", msg.To,
				"error", err,
			)
			lastErr = err

			continue
		}

		slog.InfoContext(ctx, "email sent successfully", "to", msg.To, "provider", i)

		return nil
	}

	return fmt.Errorf("all email providers failed, last error: %w", lastErr)
}

// SendSMS attempts to send an SMS using configured providers.
// Providers are tried in order until one succeeds.
func (c *CompositeNotifier) SendSMS(ctx context.Context, msg Message) error {
	if len(c.smsProviders) == 0 {
		slog.WarnContext(ctx, "no SMS providers configured, SMS not sent", "to", msg.To)

		return nil
	}

	var lastErr error

	for i, provider := range c.smsProviders {
		if err := provider.SendSMS(ctx, msg); err != nil {
			slog.WarnContext(ctx, "SMS provider failed",
				"provider", i,
				"to", msg.To,
				"error", err,
			)
			lastErr = err

			continue
		}

		slog.InfoContext(ctx, "SMS sent successfully", "to", msg.To, "provider", i)

		return nil
	}

	return fmt.Errorf("all SMS providers failed, last error: %w", lastErr)
}

// Templates returns the template manager for rendering notification content.
func (c *CompositeNotifier) Templates() *TemplateManager {
	return c.templates
}

// SendTemplatedEmail renders a template and sends the email.
func (c *CompositeNotifier) SendTemplatedEmail(ctx context.Context, to, templateName string, data TemplateData) error {
	body, err := c.templates.RenderHTML(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	msg := Message{
		To:      to,
		Subject: getSubjectFromTemplate(templateName, data),
		Body:    body,
	}

	return c.SendEmail(ctx, msg)
}

// SendTemplatedSMS renders a template and sends the SMS.
func (c *CompositeNotifier) SendTemplatedSMS(ctx context.Context, to, templateName string, data TemplateData) error {
	body, err := c.templates.RenderText(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to render SMS template: %w", err)
	}

	msg := Message{
		To:   to,
		Body: body,
	}

	return c.SendSMS(ctx, msg)
}

// getSubjectFromTemplate returns an appropriate subject based on template name.
func getSubjectFromTemplate(templateName string, data TemplateData) string {
	subjects := map[string]string{
		"magic_code_email":           "Your Login Code",
		"welcome_email":              "Welcome to " + data.AppName,
		"email_change_verification":  "Verify Your New Email",
		"password_reset":             "Reset Your Password",
		"account_security_alert":     "Security Alert",
	}

	if subject, ok := subjects[templateName]; ok {
		return subject
	}

	return "Notification from " + data.AppName
}

