package notifier

import (
	"context"
	"testing"
)

func TestNewLogNotifier(t *testing.T) {
	notifier := NewLogNotifier()
	if notifier == nil {
		t.Fatal("expected notifier to not be nil")
	}
}

func TestLogNotifier_SendEmail(t *testing.T) {
	notifier := NewLogNotifier()
	ctx := context.Background()

	msg := Message{
		To:      "test@example.com",
		Subject: "Test Subject",
		Body:    "Test Body",
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	err := notifier.SendEmail(ctx, msg)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestLogNotifier_SendSMS(t *testing.T) {
	notifier := NewLogNotifier()
	ctx := context.Background()

	msg := Message{
		To:   "+1234567890",
		Body: "Test SMS Body",
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	err := notifier.SendSMS(ctx, msg)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestLogNotifier_SendEmail_EmptyFields(t *testing.T) {
	notifier := NewLogNotifier()
	ctx := context.Background()

	msg := Message{
		To:      "",
		Subject: "",
		Body:    "",
	}

	err := notifier.SendEmail(ctx, msg)
	if err != nil {
		t.Errorf("expected no error for empty fields, got %v", err)
	}
}

func TestLogNotifier_SendSMS_EmptyFields(t *testing.T) {
	notifier := NewLogNotifier()
	ctx := context.Background()

	msg := Message{
		To:   "",
		Body: "",
	}

	err := notifier.SendSMS(ctx, msg)
	if err != nil {
		t.Errorf("expected no error for empty fields, got %v", err)
	}
}

func TestLogNotifier_ImplementsInterfaces(_ *testing.T) {
	var _ EmailProvider = (*LogNotifier)(nil)

	var _ SMSProvider = (*LogNotifier)(nil)

	var _ Notifier = (*LogNotifier)(nil)
}

