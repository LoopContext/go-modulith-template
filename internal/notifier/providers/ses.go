package providers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/cmelgarejo/go-modulith-template/internal/notifier"
)

// SESConfig holds configuration for the AWS SES provider.
type SESConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	FromEmail       string
	ConfigSetName   string // Optional: SES Configuration Set for tracking
}

// SESProvider implements EmailProvider using AWS SES.
type SESProvider struct {
	cfg    SESConfig
	client *ses.Client
}

// NewSESProvider creates a new AWS SES email provider.
func NewSESProvider(ctx context.Context, cfg SESConfig) (*SESProvider, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &SESProvider{
		cfg:    cfg,
		client: ses.NewFromConfig(awsCfg),
	}, nil
}

// SendEmail sends an email using AWS SES.
func (p *SESProvider) SendEmail(ctx context.Context, msg notifier.Message) error {
	input := &ses.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{msg.To},
		},
		Message: &types.Message{
			Body: &types.Body{
				Html: &types.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(msg.Body),
				},
			},
			Subject: &types.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(msg.Subject),
			},
		},
		Source: aws.String(p.cfg.FromEmail),
	}

	if p.cfg.ConfigSetName != "" {
		input.ConfigurationSetName = aws.String(p.cfg.ConfigSetName)
	}

	_, err := p.client.SendEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send email via SES: %w", err)
	}

	return nil
}

