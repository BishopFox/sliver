package amazonsns

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// snsSendMessageAPI Basic interface to send messages through SNS.
//
//go:generate mockery --name=snsSendMessageAPI --output=. --case=underscore --inpackage
type snsSendMessageAPI interface {
	SendMessage(ctx context.Context,
		params *sns.PublishInput,
		optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

// snsSendMessageClient Client specific for SNS using aws sdk v2.
type snsSendMessageClient struct {
	client *sns.Client
}

// SendMessage Client specific for SNS using aws sdk v2.
func (s snsSendMessageClient) SendMessage(ctx context.Context,
	params *sns.PublishInput,
	optFns ...func(*sns.Options),
) (*sns.PublishOutput, error) {
	return s.client.Publish(ctx, params, optFns...)
}

// AmazonSNS Basic structure with SNS information.
type AmazonSNS struct {
	sendMessageClient snsSendMessageAPI
	queueTopics       []string
}

// New creates a new AmazonSNS.
func New(accessKeyID, secretKey, region string) (*AmazonSNS, error) {
	credProvider := credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, "")
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithCredentialsProvider(credProvider),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}
	client := sns.NewFromConfig(cfg)
	return &AmazonSNS{
		sendMessageClient: snsSendMessageClient{client: client},
	}, nil
}

// AddReceivers takes queue urls and adds them to the internal topics
// list. The Send method will send a given message to all those
// Topics.
func (s *AmazonSNS) AddReceivers(queues ...string) {
	s.queueTopics = append(s.queueTopics, queues...)
}

// Send message to everyone on all topics.
func (s AmazonSNS) Send(ctx context.Context, subject, message string) error {
	// For each topic
	for _, topic := range s.queueTopics {
		// Create new input with subject, message and the specific topic
		input := &sns.PublishInput{
			Subject:  aws.String(subject),
			Message:  aws.String(message),
			TopicArn: aws.String(topic),
		}
		// Send the message
		_, err := s.sendMessageClient.SendMessage(ctx, input)
		if err != nil {
			return fmt.Errorf("send message using Amazon SNS to ARN TOPIC %q: %w", topic, err)
		}
	}
	return nil
}
