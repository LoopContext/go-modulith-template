package i18n

import (
	"context"
	"testing"
)

func TestContextWithLocale(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithLocale(ctx, "es")

	locale := LocaleFromContext(ctx)
	if locale != "es" {
		t.Errorf("expected locale 'es', got '%s'", locale)
	}
}

func TestLocaleFromContext_Empty(t *testing.T) {
	ctx := context.Background()

	locale := LocaleFromContext(ctx)
	if locale != "" {
		t.Errorf("expected empty locale, got '%s'", locale)
	}
}

func TestLocaleFromContext_NotSet(t *testing.T) {
	type otherKey string

	ctx := context.WithValue(context.Background(), otherKey("other_key"), "value")

	locale := LocaleFromContext(ctx)
	if locale != "" {
		t.Errorf("expected empty locale, got '%s'", locale)
	}
}

