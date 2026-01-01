# Notification System

This document describes the extensible notification system of the modulith template.

## Architecture

```
internal/notifier/
├── notifier.go          # Base interfaces (EmailProvider, SMSProvider, Notifier)
├── template.go          # TemplateManager for template rendering
├── composite.go         # CompositeNotifier that combines multiple providers
├── log_notifier.go      # LogNotifier for development (logs only)
├── subscriber.go        # Subscriber to listen to bus events
└── providers/
    ├── sendgrid.go      # SendGrid email provider
    ├── ses.go           # AWS SES email provider
    ├── twilio.go        # Twilio SMS provider
    └── sns.go           # AWS SNS SMS provider
```

## Interfaces

### EmailProvider

```go
type EmailProvider interface {
    SendEmail(ctx context.Context, msg Message) error
}
```

### SMSProvider

```go
type SMSProvider interface {
    SendSMS(ctx context.Context, msg Message) error
}
```

### Notifier

```go
type Notifier interface {
    EmailProvider
    SMSProvider
}
```

## Available Providers

### Email

| Provider | Description | Configuration |
|----------|------------|---------------|
| SendGrid | SendGrid v3 API | `SENDGRID_API_KEY`, `SENDGRID_FROM_EMAIL` |
| AWS SES | Simple Email Service | `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` |
| LogNotifier | Logging only (dev) | None |

### SMS

| Provider | Description | Configuration |
|----------|------------|---------------|
| Twilio | Twilio API | `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_FROM_NUMBER` |
| AWS SNS | Simple Notification Service | `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` |
| LogNotifier | Logging only (dev) | None |

## Basic Usage

### Development (LogNotifier)

```go
import "github.com/cmelgarejo/go-modulith-template/internal/notifier"

ntf := notifier.NewLogNotifier()
ntf.SendEmail(ctx, notifier.Message{
    To:      "user@example.com",
    Subject: "Test",
    Body:    "Hello!",
})
```

### Production (CompositeNotifier)

```go
import (
    "github.com/cmelgarejo/go-modulith-template/internal/notifier"
    "github.com/cmelgarejo/go-modulith-template/internal/notifier/providers"
)

// Create providers
sendgrid := providers.NewSendGridProvider(providers.SendGridConfig{
    APIKey:    os.Getenv("SENDGRID_API_KEY"),
    FromEmail: "noreply@example.com",
    FromName:  "My App",
})

twilio := providers.NewTwilioProvider(providers.TwilioConfig{
    AccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
    AuthToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
    FromNumber: os.Getenv("TWILIO_FROM_NUMBER"),
})

// Create composite with fallbacks
ntf := notifier.NewCompositeNotifier(notifier.CompositeConfig{
    EmailProviders: []notifier.EmailProvider{sendgrid},
    SMSProviders:   []notifier.SMSProvider{twilio},
})
```

## Template System

### Included Templates

| Template | Type | Usage |
|----------|------|-------|
| `magic_code_email` | HTML/Text | Login with magic code |
| `magic_code_sms` | Text | SMS with login code |
| `welcome_email` | HTML/Text | Welcome email |
| `email_change_verification` | HTML/Text | Email change verification |
| `phone_change_sms` | Text | Phone verification SMS |

### Using Templates

```go
tm := notifier.NewTemplateManager()

data := notifier.TemplateData{
    AppName:     "My App",
    CompanyName: "My Company",
    Year:        time.Now().Year(),
    Code:        "123456",
    ExpiresIn:   "15 minutes",
    UserName:    "John",
}

// Render HTML
htmlBody, err := tm.RenderHTML("magic_code_email", data)

// Render plain text
textBody, err := tm.RenderText("magic_code_sms", data)
```

### Custom Templates

```go
tm := notifier.NewTemplateManager()

// Add HTML template
err := tm.AddHTMLTemplate("custom_alert", `
<!DOCTYPE html>
<html>
<body>
    <h1>Alert: {{.Subject}}</h1>
    <p>{{.Body}}</p>
</body>
</html>
`)

// Add text template
err := tm.AddTextTemplate("custom_sms", `{{.AppName}}: {{.Body}}`)
```

### Load from Filesystem

```go
//go:embed templates/*
var templatesFS embed.FS

tm, err := notifier.NewTemplateManagerWithFS(
    templatesFS,
    "templates/*.html",  // HTML pattern
    "templates/*.txt",   // Text pattern
)
```

## Event Bus Integration

The notifier can subscribe to bus events:

```go
ebus := events.NewBus()

ntf := notifier.NewCompositeNotifier(cfg)
subscriber := notifier.NewSubscriber(ntf)
subscriber.SubscribeToEvents(ebus)

// Elsewhere in code
ebus.Publish(ctx, events.Event{
    Name: "magic_code_requested",
    Payload: map[string]string{
        "email": "user@example.com",
        "code":  "123456",
    },
})
```

## Best Practices

### 1. Use CompositeNotifier with Fallbacks

```go
ntf := notifier.NewCompositeNotifier(notifier.CompositeConfig{
    EmailProviders: []notifier.EmailProvider{
        primarySendGrid,   // Primary provider
        fallbackSES,       // Fallback if SendGrid fails
    },
})
```

### 2. Separate Configuration by Environment

```go
func createNotifier(env string) notifier.Notifier {
    if env == "dev" {
        return notifier.NewLogNotifier()
    }
    return createProductionNotifier()
}
```

### 3. Use Templates for Consistency

```go
// Prefer
ntf.SendTemplatedEmail(ctx, email, "magic_code_email", data)

// Instead of
ntf.SendEmail(ctx, Message{Body: "Your code is: " + code})
```

### 4. Handle Errors Appropriately

```go
if err := ntf.SendEmail(ctx, msg); err != nil {
    // Log but don't fail - notification is not critical for operation
    slog.WarnContext(ctx, "failed to send notification", "error", err)
}
```

## Adding a New Provider

1. Implement the `EmailProvider` or `SMSProvider` interface:

```go
// providers/mailgun.go
package providers

type MailgunProvider struct {
    config MailgunConfig
}

func (p *MailgunProvider) SendEmail(ctx context.Context, msg notifier.Message) error {
    // Implementación
}
```

2. Registrar en el composite:

```go
mailgun := providers.NewMailgunProvider(cfg)
ntf := notifier.NewCompositeNotifier(notifier.CompositeConfig{
    EmailProviders: []notifier.EmailProvider{mailgun},
})
```

## Variables de Entorno

### SendGrid
```env
SENDGRID_API_KEY=SG.xxx
SENDGRID_FROM_EMAIL=noreply@example.com
SENDGRID_FROM_NAME=My App
```

### Twilio
```env
TWILIO_ACCOUNT_SID=ACxxx
TWILIO_AUTH_TOKEN=xxx
TWILIO_FROM_NUMBER=+15551234567
```

### AWS (SES/SNS)
```env
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=AKIA...
AWS_SECRET_ACCESS_KEY=xxx
```

