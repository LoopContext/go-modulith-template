# Sistema de Notificaciones

Este documento describe el sistema de notificaciones extensible del modulith template.

## Arquitectura

```
internal/notifier/
├── notifier.go          # Interfaces base (EmailProvider, SMSProvider, Notifier)
├── template.go          # TemplateManager para renderizado de templates
├── composite.go         # CompositeNotifier que combina múltiples providers
├── log_notifier.go      # LogNotifier para desarrollo (solo logs)
├── subscriber.go        # Subscriber para escuchar eventos del bus
└── providers/
    ├── sendgrid.go      # Provider de email SendGrid
    ├── ses.go           # Provider de email AWS SES
    ├── twilio.go        # Provider de SMS Twilio
    └── sns.go           # Provider de SMS AWS SNS
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

## Providers Disponibles

### Email

| Provider | Descripción | Configuración |
|----------|-------------|---------------|
| SendGrid | API de SendGrid v3 | `SENDGRID_API_KEY`, `SENDGRID_FROM_EMAIL` |
| AWS SES | Simple Email Service | `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` |
| LogNotifier | Solo logging (dev) | Ninguna |

### SMS

| Provider | Descripción | Configuración |
|----------|-------------|---------------|
| Twilio | API de Twilio | `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_FROM_NUMBER` |
| AWS SNS | Simple Notification Service | `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` |
| LogNotifier | Solo logging (dev) | Ninguna |

## Uso Básico

### Desarrollo (LogNotifier)

```go
import "github.com/cmelgarejo/go-modulith-template/internal/notifier"

ntf := notifier.NewLogNotifier()
ntf.SendEmail(ctx, notifier.Message{
    To:      "user@example.com",
    Subject: "Test",
    Body:    "Hello!",
})
```

### Producción (CompositeNotifier)

```go
import (
    "github.com/cmelgarejo/go-modulith-template/internal/notifier"
    "github.com/cmelgarejo/go-modulith-template/internal/notifier/providers"
)

// Crear providers
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

// Crear composite con fallbacks
ntf := notifier.NewCompositeNotifier(notifier.CompositeConfig{
    EmailProviders: []notifier.EmailProvider{sendgrid},
    SMSProviders:   []notifier.SMSProvider{twilio},
})
```

## Sistema de Templates

### Templates Incluidos

| Template | Tipo | Uso |
|----------|------|-----|
| `magic_code_email` | HTML/Text | Login con código mágico |
| `magic_code_sms` | Text | SMS con código de login |
| `welcome_email` | HTML/Text | Email de bienvenida |
| `email_change_verification` | HTML/Text | Verificación de cambio de email |
| `phone_change_sms` | Text | SMS de verificación de teléfono |

### Usar Templates

```go
tm := notifier.NewTemplateManager()

data := notifier.TemplateData{
    AppName:     "Mi App",
    CompanyName: "Mi Empresa",
    Year:        time.Now().Year(),
    Code:        "123456",
    ExpiresIn:   "15 minutos",
    UserName:    "Juan",
}

// Renderizar HTML
htmlBody, err := tm.RenderHTML("magic_code_email", data)

// Renderizar texto plano
textBody, err := tm.RenderText("magic_code_sms", data)
```

### Templates Personalizados

```go
tm := notifier.NewTemplateManager()

// Añadir template HTML
err := tm.AddHTMLTemplate("custom_alert", `
<!DOCTYPE html>
<html>
<body>
    <h1>Alerta: {{.Subject}}</h1>
    <p>{{.Body}}</p>
</body>
</html>
`)

// Añadir template de texto
err := tm.AddTextTemplate("custom_sms", `{{.AppName}}: {{.Body}}`)
```

### Cargar desde Filesystem

```go
//go:embed templates/*
var templatesFS embed.FS

tm, err := notifier.NewTemplateManagerWithFS(
    templatesFS,
    "templates/*.html",  // Patrón HTML
    "templates/*.txt",   // Patrón texto
)
```

## Integración con Event Bus

El notifier puede suscribirse a eventos del bus:

```go
ebus := events.NewBus()

ntf := notifier.NewCompositeNotifier(cfg)
subscriber := notifier.NewSubscriber(ntf)
subscriber.SubscribeToEvents(ebus)

// En otro lugar del código
ebus.Publish(ctx, events.Event{
    Name: "magic_code_requested",
    Payload: map[string]string{
        "email": "user@example.com",
        "code":  "123456",
    },
})
```

## Mejores Prácticas

### 1. Usar CompositeNotifier con Fallbacks

```go
ntf := notifier.NewCompositeNotifier(notifier.CompositeConfig{
    EmailProviders: []notifier.EmailProvider{
        primarySendGrid,   // Provider principal
        fallbackSES,       // Fallback si SendGrid falla
    },
})
```

### 2. Separar Configuración por Entorno

```go
func createNotifier(env string) notifier.Notifier {
    if env == "dev" {
        return notifier.NewLogNotifier()
    }
    return createProductionNotifier()
}
```

### 3. Usar Templates para Consistencia

```go
// Preferir
ntf.SendTemplatedEmail(ctx, email, "magic_code_email", data)

// En lugar de
ntf.SendEmail(ctx, Message{Body: "Tu código es: " + code})
```

### 4. Manejar Errores Apropiadamente

```go
if err := ntf.SendEmail(ctx, msg); err != nil {
    // Log pero no fallar - la notificación no es crítica para la operación
    slog.WarnContext(ctx, "failed to send notification", "error", err)
}
```

## Agregar un Nuevo Provider

1. Implementar la interface `EmailProvider` o `SMSProvider`:

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

