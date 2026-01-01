package i18n

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed resources/en.json resources/es.json
var translationFiles embed.FS

var (
	bundle     *i18n.Bundle
	bundleOnce sync.Once
)

// Init initializes the i18n bundle and loads translation files.
// This should be called once at application startup.
func Init(defaultLocale string) error {
	var initErr error

	bundleOnce.Do(func() {
		bundle = i18n.NewBundle(language.MustParse(defaultLocale))
		bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

		// Load embedded translation files
		entries, err := translationFiles.ReadDir("resources")
		if err != nil {
			initErr = fmt.Errorf("failed to read translation directory: %w", err)
			return
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			if !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}

			data, err := translationFiles.ReadFile("resources/" + entry.Name())
			if err != nil {
				slog.Warn("Failed to load translation file", "file", entry.Name(), "error", err)
				continue
			}

			if _, err := bundle.ParseMessageFileBytes(data, entry.Name()); err != nil {
				slog.Warn("Failed to parse translation file", "file", entry.Name(), "error", err)
				continue
			}

			slog.Info("Loaded translation file", "file", entry.Name())
		}
	})

	return initErr
}

// T translates a message key using the locale from context.
// If no locale is found in context, it uses the default locale.
func T(ctx context.Context, defaultLocale string, messageID string, templateData map[string]interface{}) string {
	locale := DetectLocale(ctx, defaultLocale)

	localizer := i18n.NewLocalizer(bundle, locale)

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})
	if err != nil {
		slog.Debug("Translation not found, using message ID", "messageID", messageID, "locale", locale, "error", err)

		return messageID
	}

	return msg
}

// MustT is like T but panics if the translation is not found.
// Use this only for critical translations that must exist.
func MustT(ctx context.Context, defaultLocale string, messageID string, templateData map[string]interface{}) string {
	locale := DetectLocale(ctx, defaultLocale)

	localizer := i18n.NewLocalizer(bundle, locale)

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})
	if err != nil {
		panic(fmt.Sprintf("translation not found: %s (locale: %s)", messageID, locale))
	}

	return msg
}

