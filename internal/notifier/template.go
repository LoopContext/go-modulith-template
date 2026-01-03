package notifier

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"sync"
)

// TemplateManager handles loading and rendering of notification templates.
// It supports both embedded and file-system based templates.
type TemplateManager struct {
	htmlTemplates *template.Template
	textTemplates *template.Template
	mu            sync.RWMutex
}

// TemplateData represents the data passed to notification templates.
type TemplateData struct {
	// Common fields
	AppName     string
	AppURL      string
	SupportURL  string
	CompanyName string
	Year        int

	// User-specific
	UserName  string
	UserEmail string
	UserPhone string

	// Content-specific (varies by template)
	Code        string // Magic code, verification code, etc.
	ActionURL   string // Click-through URL
	ActionLabel string // Button text
	ExpiresIn   string // Human-readable expiration

	// Custom data for extensibility
	Extra map[string]interface{}
}

// NewTemplateManager creates a new TemplateManager with default templates.
func NewTemplateManager() *TemplateManager {
	tm := &TemplateManager{}
	tm.loadDefaultTemplates()

	return tm
}

// NewTemplateManagerWithFS creates a TemplateManager loading templates from an embed.FS.
func NewTemplateManagerWithFS(fs embed.FS, htmlPattern, textPattern string) (*TemplateManager, error) {
	tm := &TemplateManager{}

	var err error

	if htmlPattern != "" {
		tm.htmlTemplates, err = template.ParseFS(fs, htmlPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to parse HTML templates: %w", err)
		}
	}

	if textPattern != "" {
		tm.textTemplates, err = template.ParseFS(fs, textPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to parse text templates: %w", err)
		}
	}

	return tm, nil
}

// RenderHTML renders an HTML template with the given data.
func (tm *TemplateManager) RenderHTML(templateName string, data TemplateData) (string, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.htmlTemplates == nil {
		return "", fmt.Errorf("no HTML templates loaded")
	}

	var buf bytes.Buffer
	if err := tm.htmlTemplates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", fmt.Errorf("failed to render HTML template %s: %w", templateName, err)
	}

	return buf.String(), nil
}

// RenderText renders a text template with the given data.
func (tm *TemplateManager) RenderText(templateName string, data TemplateData) (string, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.textTemplates == nil {
		return "", fmt.Errorf("no text templates loaded")
	}

	var buf bytes.Buffer
	if err := tm.textTemplates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", fmt.Errorf("failed to render text template %s: %w", templateName, err)
	}

	return buf.String(), nil
}

// AddHTMLTemplate adds or updates an HTML template.
func (tm *TemplateManager) AddHTMLTemplate(name, content string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.htmlTemplates == nil {
		var err error

		tm.htmlTemplates, err = template.New(name).Parse(content)
		if err != nil {
			return fmt.Errorf("failed to parse HTML template: %w", err)
		}

		return nil
	}

	_, err := tm.htmlTemplates.New(name).Parse(content)
	if err != nil {
		return fmt.Errorf("failed to add HTML template: %w", err)
	}

	return nil
}

// AddTextTemplate adds or updates a text template.
func (tm *TemplateManager) AddTextTemplate(name, content string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.textTemplates == nil {
		var err error

		tm.textTemplates, err = template.New(name).Parse(content)
		if err != nil {
			return fmt.Errorf("failed to parse text template: %w", err)
		}

		return nil
	}

	_, err := tm.textTemplates.New(name).Parse(content)
	if err != nil {
		return fmt.Errorf("failed to add text template: %w", err)
	}

	return nil
}

// loadDefaultTemplates loads built-in notification templates.
//
//nolint:funlen // Template definitions require many lines
func (tm *TemplateManager) loadDefaultTemplates() {
	// Magic code email (HTML)
	_ = tm.AddHTMLTemplate("magic_code_email", `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Your Login Code</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 30px; border-radius: 8px 8px 0 0; text-align: center;">
        <h1 style="color: white; margin: 0;">{{.AppName}}</h1>
    </div>
    <div style="background: #ffffff; padding: 30px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 8px 8px;">
        <h2 style="color: #333; margin-top: 0;">Your Login Code</h2>
        <p style="color: #666; font-size: 16px;">Use the following code to complete your login:</p>
        <div style="background: #f5f5f5; padding: 20px; text-align: center; border-radius: 8px; margin: 20px 0;">
            <span style="font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #333;">{{.Code}}</span>
        </div>
        <p style="color: #999; font-size: 14px;">This code expires in {{.ExpiresIn}}.</p>
        <p style="color: #999; font-size: 14px;">If you didn't request this code, you can safely ignore this email.</p>
    </div>
    <div style="text-align: center; padding: 20px; color: #999; font-size: 12px;">
        <p>&copy; {{.Year}} {{.CompanyName}}. All rights reserved.</p>
    </div>
</body>
</html>`)

	// Magic code email (text)
	_ = tm.AddTextTemplate("magic_code_email", `{{.AppName}} - Your Login Code

Use the following code to complete your login:

{{.Code}}

This code expires in {{.ExpiresIn}}.

If you didn't request this code, you can safely ignore this email.

---
{{.Year}} {{.CompanyName}}`)

	// Magic code SMS
	_ = tm.AddTextTemplate("magic_code_sms", `{{.AppName}}: Your login code is {{.Code}}. Expires in {{.ExpiresIn}}.`)

	// Welcome email (HTML)
	_ = tm.AddHTMLTemplate("welcome_email", `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to {{.AppName}}</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%); padding: 30px; border-radius: 8px 8px 0 0; text-align: center;">
        <h1 style="color: white; margin: 0;">Welcome to {{.AppName}}!</h1>
    </div>
    <div style="background: #ffffff; padding: 30px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 8px 8px;">
        <h2 style="color: #333; margin-top: 0;">Hello{{if .UserName}}, {{.UserName}}{{end}}!</h2>
        <p style="color: #666; font-size: 16px;">Thank you for joining us. We're excited to have you on board.</p>
        {{if .ActionURL}}
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.ActionURL}}" style="background: #11998e; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; font-weight: bold;">{{if .ActionLabel}}{{.ActionLabel}}{{else}}Get Started{{end}}</a>
        </div>
        {{end}}
        <p style="color: #999; font-size: 14px;">If you have any questions, feel free to reach out to our support team.</p>
    </div>
    <div style="text-align: center; padding: 20px; color: #999; font-size: 12px;">
        <p>&copy; {{.Year}} {{.CompanyName}}. All rights reserved.</p>
    </div>
</body>
</html>`)

	// Welcome email (text)
	_ = tm.AddTextTemplate("welcome_email", `Welcome to {{.AppName}}!

Hello{{if .UserName}}, {{.UserName}}{{end}}!

Thank you for joining us. We're excited to have you on board.

{{if .ActionURL}}Get started: {{.ActionURL}}{{end}}

If you have any questions, feel free to reach out to our support team.

---
{{.Year}} {{.CompanyName}}`)

	// Email change verification
	_ = tm.AddHTMLTemplate("email_change_verification", `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Verify Your New Email</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); padding: 30px; border-radius: 8px 8px 0 0; text-align: center;">
        <h1 style="color: white; margin: 0;">{{.AppName}}</h1>
    </div>
    <div style="background: #ffffff; padding: 30px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 8px 8px;">
        <h2 style="color: #333; margin-top: 0;">Verify Your New Email</h2>
        <p style="color: #666; font-size: 16px;">Use the following code to verify this email address:</p>
        <div style="background: #f5f5f5; padding: 20px; text-align: center; border-radius: 8px; margin: 20px 0;">
            <span style="font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #333;">{{.Code}}</span>
        </div>
        <p style="color: #999; font-size: 14px;">This code expires in {{.ExpiresIn}}.</p>
        <p style="color: #999; font-size: 14px;">If you didn't request this change, please secure your account immediately.</p>
    </div>
    <div style="text-align: center; padding: 20px; color: #999; font-size: 12px;">
        <p>&copy; {{.Year}} {{.CompanyName}}. All rights reserved.</p>
    </div>
</body>
</html>`)

	// Email change verification (text)
	_ = tm.AddTextTemplate("email_change_verification", `{{.AppName}} - Verify Your New Email

Use the following code to verify this email address:

{{.Code}}

This code expires in {{.ExpiresIn}}.

If you didn't request this change, please secure your account immediately.

---
{{.Year}} {{.CompanyName}}`)

	// Phone change verification SMS
	_ = tm.AddTextTemplate("phone_change_sms", `{{.AppName}}: Your phone verification code is {{.Code}}. Expires in {{.ExpiresIn}}.`)
}
