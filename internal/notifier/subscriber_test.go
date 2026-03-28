package notifier

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/LoopContext/go-modulith-template/internal/events"
	"github.com/LoopContext/go-modulith-template/internal/i18n"
)

type mockNotifier struct {
	mu         sync.Mutex
	emailCalls []Message
	smsCalls   []Message
	emailErr   error
	smsErr     error
}

func (m *mockNotifier) SendEmail(_ context.Context, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.emailCalls = append(m.emailCalls, msg)

	return m.emailErr
}

func (m *mockNotifier) SendSMS(_ context.Context, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.smsCalls = append(m.smsCalls, msg)

	return m.smsErr
}

func (m *mockNotifier) getEmailCallsCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.emailCalls)
}

func init() {
	// Initialize i18n for tests
	_ = i18n.Init("en")
}

func TestNewSubscriber(t *testing.T) {
	notifier := &mockNotifier{}
	subscriber := NewSubscriber(notifier, "en")

	if subscriber == nil {
		t.Fatal("expected subscriber to not be nil")
	}

	if subscriber.notifier != notifier {
		t.Error("expected subscriber to use provided notifier")
	}
}

func TestSubscriber_SubscribeToEvents(t *testing.T) {
	notifier := &mockNotifier{}
	subscriber := NewSubscriber(notifier, "en")
	bus := events.NewBus()

	subscriber.SubscribeToEvents(bus)

	// Verify that the event handler is registered by publishing an event
	ctx := context.Background()

	payload := map[string]interface{}{
		"email": "test@example.com",
		"code":  "123456",
	}

	bus.Publish(ctx, events.Event{
		Name:    EventMagicCodeRequested,
		Payload: payload,
	})

	// Wait for async event processing
	time.Sleep(100 * time.Millisecond)

	if notifier.getEmailCallsCount() != 1 {
		t.Errorf("expected 1 email call, got %d", notifier.getEmailCallsCount())
	}
}

func TestSubscriber_HandleMagicCodeRequested_Email(t *testing.T) {
	notifier := &mockNotifier{}
	subscriber := NewSubscriber(notifier, "en")

	payload := map[string]interface{}{
		"email": "user@example.com",
		"code":  "654321",
	}

	event := events.Event{
		Name:    EventMagicCodeRequested,
		Payload: payload,
	}

	err := subscriber.handleMagicCodeRequested(context.Background(), event)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	notifier.mu.Lock()
	defer notifier.mu.Unlock()

	if len(notifier.emailCalls) != 1 {
		t.Fatalf("expected 1 email call, got %d", len(notifier.emailCalls))
	}

	emailMsg := notifier.emailCalls[0]
	if emailMsg.To != "user@example.com" {
		t.Errorf("expected email to 'user@example.com', got %s", emailMsg.To)
	}

	if emailMsg.Subject != "Your Access Code" {
		t.Errorf("expected subject 'Your Access Code', got %s", emailMsg.Subject)
	}

	expectedBody := "Your magic code is: 654321"
	if emailMsg.Body != expectedBody {
		t.Errorf("expected body '%s', got %s", expectedBody, emailMsg.Body)
	}

	if len(notifier.smsCalls) != 0 {
		t.Errorf("expected 0 SMS calls, got %d", len(notifier.smsCalls))
	}
}

func TestSubscriber_HandleMagicCodeRequested_Phone(t *testing.T) {
	notifier := &mockNotifier{}
	subscriber := NewSubscriber(notifier, "en")

	payload := map[string]interface{}{
		"phone": "+1234567890",
		"code":  "999888",
	}

	event := events.Event{
		Name:    EventMagicCodeRequested,
		Payload: payload,
	}

	err := subscriber.handleMagicCodeRequested(context.Background(), event)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	notifier.mu.Lock()
	defer notifier.mu.Unlock()

	if len(notifier.smsCalls) != 1 {
		t.Fatalf("expected 1 SMS call, got %d", len(notifier.smsCalls))
	}

	smsMsg := notifier.smsCalls[0]
	if smsMsg.To != "+1234567890" {
		t.Errorf("expected SMS to '+1234567890', got %s", smsMsg.To)
	}

	expectedBody := "Your magic code is: 999888"
	if smsMsg.Body != expectedBody {
		t.Errorf("expected body '%s', got %s", expectedBody, smsMsg.Body)
	}

	if len(notifier.emailCalls) != 0 {
		t.Errorf("expected 0 email calls, got %d", len(notifier.emailCalls))
	}
}

func TestSubscriber_HandleMagicCodeRequested_EmailPriority(t *testing.T) {
	notifier := &mockNotifier{}
	subscriber := NewSubscriber(notifier, "en")

	// When both email and phone are present, email takes priority
	payload := map[string]interface{}{
		"email": "user@example.com",
		"phone": "+1234567890",
		"code":  "111222",
	}

	event := events.Event{
		Name:    EventMagicCodeRequested,
		Payload: payload,
	}

	err := subscriber.handleMagicCodeRequested(context.Background(), event)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	notifier.mu.Lock()
	defer notifier.mu.Unlock()

	if len(notifier.emailCalls) != 1 {
		t.Fatalf("expected 1 email call, got %d", len(notifier.emailCalls))
	}

	if len(notifier.smsCalls) != 0 {
		t.Errorf("expected 0 SMS calls when email is present, got %d", len(notifier.smsCalls))
	}
}

func TestSubscriber_HandleMagicCodeRequested_InvalidPayload(t *testing.T) {
	notifier := &mockNotifier{}
	subscriber := NewSubscriber(notifier, "en")

	// Invalid payload type
	event := events.Event{
		Name:    EventMagicCodeRequested,
		Payload: "not a map",
	}

	err := subscriber.handleMagicCodeRequested(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for invalid payload type")
	}

	notifier.mu.Lock()
	defer notifier.mu.Unlock()

	if len(notifier.emailCalls) != 0 {
		t.Errorf("expected 0 email calls, got %d", len(notifier.emailCalls))
	}

	if len(notifier.smsCalls) != 0 {
		t.Errorf("expected 0 SMS calls, got %d", len(notifier.smsCalls))
	}
}

func TestSubscriber_HandleMagicCodeRequested_NoEmailOrPhone(t *testing.T) {
	notifier := &mockNotifier{}
	subscriber := NewSubscriber(notifier, "en")

	payload := map[string]interface{}{
		"code": "123456",
	}

	event := events.Event{
		Name:    EventMagicCodeRequested,
		Payload: payload,
	}

	// Should not error, but also not send anything
	err := subscriber.handleMagicCodeRequested(context.Background(), event)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	notifier.mu.Lock()
	defer notifier.mu.Unlock()

	if len(notifier.emailCalls) != 0 {
		t.Errorf("expected 0 email calls, got %d", len(notifier.emailCalls))
	}

	if len(notifier.smsCalls) != 0 {
		t.Errorf("expected 0 SMS calls, got %d", len(notifier.smsCalls))
	}
}

func TestSubscriber_HandleMagicCodeRequested_EmailError(t *testing.T) {
	notifier := &mockNotifier{
		emailErr: errors.New("email send failed"),
	}
	subscriber := NewSubscriber(notifier, "en")

	payload := map[string]interface{}{
		"email": "user@example.com",
		"code":  "123456",
	}

	event := events.Event{
		Name:    EventMagicCodeRequested,
		Payload: payload,
	}

	err := subscriber.handleMagicCodeRequested(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when email send fails")
	}
}

func TestSubscriber_HandleMagicCodeRequested_SMSError(t *testing.T) {
	notifier := &mockNotifier{
		smsErr: errors.New("SMS send failed"),
	}
	subscriber := NewSubscriber(notifier, "en")

	payload := map[string]interface{}{
		"phone": "+1234567890",
		"code":  "123456",
	}

	event := events.Event{
		Name:    EventMagicCodeRequested,
		Payload: payload,
	}

	err := subscriber.handleMagicCodeRequested(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when SMS send fails")
	}
}
