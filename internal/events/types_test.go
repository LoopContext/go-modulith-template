package events

import "testing"

const testUserID = "user_123"

//nolint:cyclop,funlen // Test function necessarily has many test cases
func TestEventPayloadHelpers(t *testing.T) {
	t.Run("NewUserCreatedPayload", func(t *testing.T) {
		payload := NewUserCreatedPayload(testUserID, "test@example.com")

		if payload["user_id"] != testUserID {
			t.Errorf("expected user_id to be 'user_123', got '%v'", payload["user_id"])
		}

		if payload["email"] != "test@example.com" {
			t.Errorf("expected email to be 'test@example.com', got '%v'", payload["email"])
		}
	})

	t.Run("NewMagicCodeRequestedPayload with email", func(t *testing.T) {
		payload := NewMagicCodeRequestedPayload("test@example.com", "", "123456")

		if payload["email"] != "test@example.com" {
			t.Errorf("expected email to be 'test@example.com', got '%v'", payload["email"])
		}

		if payload["code"] != "123456" {
			t.Errorf("expected code to be '123456', got '%v'", payload["code"])
		}

		if _, exists := payload["phone"]; exists {
			t.Error("expected phone to not be in payload")
		}
	})

	t.Run("NewMagicCodeRequestedPayload with phone", func(t *testing.T) {
		payload := NewMagicCodeRequestedPayload("", "+1234567890", "123456")

		if payload["phone"] != "+1234567890" {
			t.Errorf("expected phone to be '+1234567890', got '%v'", payload["phone"])
		}

		if _, exists := payload["email"]; exists {
			t.Error("expected email to not be in payload")
		}
	})

	t.Run("NewSessionCreatedPayload", func(t *testing.T) {
		payload := NewSessionCreatedPayload(testUserID, "session_456")

		if payload["user_id"] != testUserID {
			t.Errorf("expected user_id to be 'user_123', got '%v'", payload["user_id"])
		}

		if payload["session_id"] != "session_456" {
			t.Errorf("expected session_id to be 'session_456', got '%v'", payload["session_id"])
		}
	})

	t.Run("NewProfileUpdatedPayload", func(t *testing.T) {
		payload := NewProfileUpdatedPayload(testUserID, "John Doe", "https://example.com/avatar.jpg")

		if payload["user_id"] != testUserID {
			t.Errorf("expected user_id to be 'user_123', got '%v'", payload["user_id"])
		}

		if payload["display_name"] != "John Doe" {
			t.Errorf("expected display_name to be 'John Doe', got '%v'", payload["display_name"])
		}

		if payload["avatar_url"] != "https://example.com/avatar.jpg" {
			t.Errorf("expected avatar_url to be 'https://example.com/avatar.jpg', got '%v'", payload["avatar_url"])
		}
	})

	t.Run("NewOAuthAccountLinkedPayload", func(t *testing.T) {
		payload := NewOAuthAccountLinkedPayload(testUserID, "google", "google_456")

		if payload["user_id"] != testUserID {
			t.Errorf("expected user_id to be 'user_123', got '%v'", payload["user_id"])
		}

		if payload["provider"] != "google" {
			t.Errorf("expected provider to be 'google', got '%v'", payload["provider"])
		}

		if payload["provider_user_id"] != "google_456" {
			t.Errorf("expected provider_user_id to be 'google_456', got '%v'", payload["provider_user_id"])
		}
	})
}
