package i18n

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
	"golang.org/x/text/language"
)

// DetectLocale detects the locale from the context.
// Priority:
// 1. Locale stored in context (from previous detection or user preference)
// 2. Accept-Language header from gRPC metadata
// 3. Default locale
func DetectLocale(ctx context.Context, defaultLocale string) string {
	// Check if locale is already in context
	if locale := LocaleFromContext(ctx); locale != "" {
		return normalizeLocale(locale)
	}

	// Extract Accept-Language from gRPC metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if acceptLang := md.Get("accept-language"); len(acceptLang) > 0 && acceptLang[0] != "" {
			locale := parseAcceptLanguage(acceptLang[0], defaultLocale)
			return normalizeLocale(locale)
		}
	}

	// Fallback to default
	if defaultLocale == "" {
		return DefaultLocale
	}

	return normalizeLocale(defaultLocale)
}

// parseAcceptLanguage parses the Accept-Language header and returns the best match.
// Accept-Language format: "en-US,en;q=0.9,es;q=0.8"
func parseAcceptLanguage(acceptLang string, defaultLocale string) string {
	// Parse the Accept-Language header
	tags, _, err := language.ParseAcceptLanguage(acceptLang)
	if err != nil || len(tags) == 0 {
		return defaultLocale
	}

	// Get the base language from the first (highest priority) tag
	base, _ := tags[0].Base()
	locale := base.String()

	// Normalize to supported locales
	return normalizeLocale(locale)
}

// normalizeLocale normalizes a locale string to a supported locale.
// Currently supports: en, es
// Falls back to "en" if not supported.
func normalizeLocale(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))

	// Extract base language (e.g., "en-US" -> "en")
	if idx := strings.Index(locale, "-"); idx > 0 {
		locale = locale[:idx]
	}

	// Check if supported
	switch locale {
	case "en", "es":
		return locale
	default:
		return DefaultLocale
	}
}

