// Package i18n provides internationalization support for the application.
package i18n

import (
	"context"
)

type ctxKey string

const (
	ctxLocale ctxKey = "i18n.locale"
	// DefaultLocale is the default locale used when no locale is specified.
	DefaultLocale = "en"
)

// ContextWithLocale injects a locale into the context.
func ContextWithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, ctxLocale, locale)
}

// LocaleFromContext extracts the locale from context, or returns empty string if not found.
func LocaleFromContext(ctx context.Context) string {
	if locale, ok := ctx.Value(ctxLocale).(string); ok && locale != "" {
		return locale
	}

	return ""
}
