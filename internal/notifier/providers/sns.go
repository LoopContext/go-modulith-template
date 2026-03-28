package providers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/LoopContext/go-modulith-template/internal/notifier"
)

// SNSConfig holds configuration for the AWS SNS provider.
type SNSConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	SenderID        string // Optional: Sender ID for SMS (not supported in all regions)
}

// SNSProvider implements SMSProvider using AWS SNS.
type SNSProvider struct {
	cfg    SNSConfig
	client *sns.Client
}

// NewSNSProvider creates a new AWS SNS SMS provider.
func NewSNSProvider(ctx context.Context, cfg SNSConfig) (*SNSProvider, error) {
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

	return &SNSProvider{
		cfg:    cfg,
		client: sns.NewFromConfig(awsCfg),
	}, nil
}

// SendSMS sends an SMS using AWS SNS.
func (p *SNSProvider) SendSMS(ctx context.Context, msg notifier.Message) error {
	input := &sns.PublishInput{
		PhoneNumber: aws.String(msg.To),
		Message:     aws.String(msg.Body),
	}

	if p.cfg.SenderID != "" {
		input.MessageAttributes = map[string]types.MessageAttributeValue{
			"AWS.SNS.SMS.SenderID": {
				DataType:    aws.String("String"),
				StringValue: aws.String(p.cfg.SenderID),
			},
		}
	}

	_, err := p.client.Publish(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send SMS via SNS: %w", err)
	}

	return nil
}
