package i18n

import (
	"context"
	"testing"
)

func TestInit(t *testing.T) {
	err := Init("en")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Test that translations are loaded
	ctx := context.Background()

	msg := T(ctx, "en", "errors.user_not_found", nil)
	if msg == "errors.user_not_found" {
		t.Error("translation not found, expected translated message")
	}
}

func TestT_WithContext(t *testing.T) {
	if err := Init("en"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	ctx := ContextWithLocale(context.Background(), "es")
	msg := T(ctx, "en", "errors.user_not_found", nil)

	// Should be in Spanish
	if msg == "errors.user_not_found" {
		t.Error("translation not found")
	}

	// Should not be English
	if msg == "User not found" {
		t.Error("expected Spanish translation, got English")
	}
}

func TestT_WithTemplateData(t *testing.T) {
	if err := Init("en"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	ctx := context.Background()
	msg := T(ctx, "en", "notifications.magic_code_body", map[string]interface{}{
		"Code": "123456",
	})

	if msg == "notifications.magic_code_body" {
		t.Error("translation not found")
	}

	if msg == "" {
		t.Error("translation returned empty string")
	}
}

func TestT_MissingTranslation(t *testing.T) {
	if err := Init("en"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	ctx := context.Background()
	msg := T(ctx, "en", "nonexistent.key", nil)

	// Should return the key if translation not found
	if msg != "nonexistent.key" {
		t.Errorf("expected key 'nonexistent.key' for missing translation, got '%s'", msg)
	}
}

