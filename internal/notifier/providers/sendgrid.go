// Package providers implements real notification providers for email and SMS.
package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/LoopContext/go-modulith-template/internal/notifier"
)

// SendGridConfig holds configuration for the SendGrid provider.
type SendGridConfig struct {
	APIKey      string
	FromEmail   string
	FromName    string
	SandboxMode bool // If true, emails are validated but not sent
}

// SendGridProvider implements EmailProvider using SendGrid's API.
type SendGridProvider struct {
	config     SendGridConfig
	httpClient *http.Client
}

// NewSendGridProvider creates a new SendGrid email provider.
func NewSendGridProvider(cfg SendGridConfig) *SendGridProvider {
	return &SendGridProvider{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, //nolint:mnd
		},
	}
}

// sendGridMail represents a SendGrid mail request.
type sendGridMail struct {
	Personalizations []sendGridPersonalization `json:"personalizations"`
	From             sendGridAddress           `json:"from"`
	Subject          string                    `json:"subject"`
	Content          []sendGridContent         `json:"content"`
	MailSettings     *sendGridMailSettings     `json:"mail_settings,omitempty"`
}

type sendGridPersonalization struct {
	To []sendGridAddress `json:"to"`
}

type sendGridAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type sendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type sendGridMailSettings struct {
	SandboxMode *sendGridSandboxMode `json:"sandbox_mode,omitempty"`
}

type sendGridSandboxMode struct {
	Enable bool `json:"enable"`
}

// SendEmail sends an email using SendGrid's API.
func (p *SendGridProvider) SendEmail(ctx context.Context, msg notifier.Message) error {
	mail := sendGridMail{
		Personalizations: []sendGridPersonalization{
			{
				To: []sendGridAddress{{Email: msg.To}},
			},
		},
		From:    sendGridAddress{Email: p.config.FromEmail, Name: p.config.FromName},
		Subject: msg.Subject,
		Content: []sendGridContent{
			{Type: "text/html", Value: msg.Body},
		},
	}

	if p.config.SandboxMode {
		mail.MailSettings = &sendGridMailSettings{
			SandboxMode: &sendGridSandboxMode{Enable: true},
		}
	}

	body, err := json.Marshal(mail)
	if err != nil {
		return fmt.Errorf("failed to marshal email: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("sendgrid API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}
