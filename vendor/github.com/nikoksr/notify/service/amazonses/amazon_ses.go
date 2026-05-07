package amazonses

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

//go:generate mockery --name=sesClient --output=. --case=underscore --inpackage
type sesClient interface {
	SendEmail(
		ctx context.Context,
		params *ses.SendEmailInput,
		optFns ...func(options *ses.Options),
	) (*ses.SendEmailOutput, error)
}

// Compile-time check to ensure that ses.Client implements the sesClient interface.
var _ sesClient = new(ses.Client)

// AmazonSES struct holds necessary data to communicate with the Amazon Simple Email Service API.
type AmazonSES struct {
	client            sesClient
	senderAddress     *string
	receiverAddresses []string
}

// New returns a new instance of a AmazonSES notification service.
// You will need an Amazon Simple Email Service API access key and secret.
// See https://aws.github.io/aws-sdk-go-v2/docs/getting-started/
func New(accessKeyID, secretKey, region, senderAddress string) (*AmazonSES, error) {
	credProvider := credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, "")

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithCredentialsProvider(credProvider),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	return &AmazonSES{
		client:            ses.NewFromConfig(cfg),
		senderAddress:     aws.String(senderAddress),
		receiverAddresses: []string{},
	}, nil
}

// AddReceivers takes email addresses and adds them to the internal address list. The Send method will send
// a given message to all those addresses.
func (a *AmazonSES) AddReceivers(addresses ...string) {
	a.receiverAddresses = append(a.receiverAddresses, addresses...)
}

// Send takes a message subject and a message body and sends them to all previously set chats. Message body supports
// html as markup language.
func (a AmazonSES) Send(ctx context.Context, subject, message string) error {
	input := &ses.SendEmailInput{
		Source: a.senderAddress,
		Destination: &types.Destination{
			ToAddresses: a.receiverAddresses,
		},
		Message: &types.Message{
			Body: &types.Body{
				Html: &types.Content{
					Data: aws.String(message),
				},
				// Text: &types.Content{
				//     Data:    aws.String(message),
				// },
			},
			Subject: &types.Content{
				Data: aws.String(subject),
			},
		},
	}

	_, err := a.client.SendEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("send mail using Amazon SES service: %w", err)
	}

	return nil
}
