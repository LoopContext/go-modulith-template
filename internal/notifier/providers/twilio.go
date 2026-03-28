package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LoopContext/go-modulith-template/internal/notifier"
)

// TwilioConfig holds configuration for the Twilio provider.
type TwilioConfig struct {
	AccountSID   string
	AuthToken    string
	FromNumber   string // Your Twilio phone number
	MessagingSID string // Optional: Messaging Service SID
}

// TwilioProvider implements SMSProvider using Twilio's API.
type TwilioProvider struct {
	config     TwilioConfig
	httpClient *http.Client
}

// NewTwilioProvider creates a new Twilio SMS provider.
func NewTwilioProvider(cfg TwilioConfig) *TwilioProvider {
	return &TwilioProvider{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, //nolint:mnd
		},
	}
}

// SendSMS sends an SMS using Twilio's API.
func (p *TwilioProvider) SendSMS(ctx context.Context, msg notifier.Message) error {
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", p.config.AccountSID)

	data := url.Values{}
	data.Set("To", msg.To)
	data.Set("Body", msg.Body)

	// Use Messaging Service SID if available, otherwise use From number
	if p.config.MessagingSID != "" {
		data.Set("MessagingServiceSid", p.config.MessagingSID)
	} else {
		data.Set("From", p.config.FromNumber)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.config.AccountSID, p.config.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("twilio API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}
