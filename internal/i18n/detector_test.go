package i18n

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestDetectLocale_FromContext(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithLocale(ctx, "es")

	locale := DetectLocale(ctx, "en")
	if locale != "es" {
		t.Errorf("expected locale 'es' from context, got '%s'", locale)
	}
}

func TestDetectLocale_FromMetadata(t *testing.T) {
	md := metadata.New(map[string]string{
		"accept-language": "es-ES,es;q=0.9",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	locale := DetectLocale(ctx, "en")
	if locale != "es" {
		t.Errorf("expected locale 'es' from Accept-Language, got '%s'", locale)
	}
}

func TestDetectLocale_DefaultFallback(t *testing.T) {
	ctx := context.Background()

	locale := DetectLocale(ctx, "en")
	if locale != "en" {
		t.Errorf("expected default locale 'en', got '%s'", locale)
	}
}

func TestDetectLocale_EmptyDefault(t *testing.T) {
	ctx := context.Background()

	locale := DetectLocale(ctx, "")
	if locale != DefaultLocale {
		t.Errorf("expected default locale '%s', got '%s'", DefaultLocale, locale)
	}
}

func TestDetectLocale_ComplexAcceptLanguage(t *testing.T) {
	md := metadata.New(map[string]string{
		"accept-language": "en-US,en;q=0.9,es;q=0.8,fr;q=0.7",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	locale := DetectLocale(ctx, "en")
	if locale != "en" {
		t.Errorf("expected locale 'en' from Accept-Language, got '%s'", locale)
	}
}

func TestDetectLocale_UnsupportedLanguage(t *testing.T) {
	md := metadata.New(map[string]string{
		"accept-language": "fr-FR,fr;q=0.9",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	locale := DetectLocale(ctx, "en")
	if locale != "en" {
		t.Errorf("expected default locale 'en' for unsupported language, got '%s'", locale)
	}
}

func TestNormalizeLocale(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "en", "en"},
		{"with region", "en-US", "en"},
		{"lowercase", "ES", "es"},
		{"mixed case", "En-Us", "en"},
		{"unsupported", "fr", DefaultLocale},
		{"empty", "", DefaultLocale},
		{"whitespace", "  en  ", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeLocale(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeLocale(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
