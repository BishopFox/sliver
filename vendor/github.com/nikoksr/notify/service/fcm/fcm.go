package fcm

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"firebase.google.com/go/v4/messaging"
	"github.com/appleboy/go-fcm"

	"github.com/nikoksr/notify"
)

// Compile-time check that Service satisfies the Notifier interface.
var _ notify.Notifier = (*Service)(nil)

// Used to generate mocks for the FCM client.
type fcmClient interface {
	Send(ctx context.Context, message ...*messaging.Message) (*messaging.BatchResponse, error)
	SendMulticast(ctx context.Context, message *messaging.MulticastMessage) (*messaging.BatchResponse, error)
}

// Service encapsulates the FCM client along with internal state for storing device tokens.
type Service struct {
	client       fcmClient
	deviceTokens []string
}

// Option is a function that configures a Service.
type Option func(*Service) error

// WithCredentialsFile returns an Option to configure the FCM client with a credentials file.
func WithCredentialsFile(filename string) Option {
	return func(s *Service) error {
		client, ok := s.client.(*fcm.Client)
		if !ok {
			return errors.New("client is not of type *fcm.Client")
		}

		return fcm.WithCredentialsFile(filename)(client)
	}
}

// WithProjectID returns an Option to configure the FCM client with a project ID.
func WithProjectID(projectID string) Option {
	return func(s *Service) error {
		client, ok := s.client.(*fcm.Client)
		if !ok {
			return errors.New("client is not of type *fcm.Client")
		}

		return fcm.WithProjectID(projectID)(client)
	}
}

// WithHTTPClient returns an Option to configure the FCM client with a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(s *Service) error {
		client, ok := s.client.(*fcm.Client)
		if !ok {
			return errors.New("client is not of type *fcm.Client")
		}

		return fcm.WithHTTPClient(httpClient)(client)
	}
}

// New returns a new instance of a FCM notification service.
func New(ctx context.Context, opts ...Option) (*Service, error) {
	client, err := fcm.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create FCM client: %w", err)
	}

	s := &Service{
		client:       client,
		deviceTokens: []string{},
	}

	for _, opt := range opts {
		if err = opt(s); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	return s, nil
}

// AddReceivers takes FCM device tokens and appends them to the internal device tokens slice.
func (s *Service) AddReceivers(deviceTokens ...string) {
	s.deviceTokens = append(s.deviceTokens, deviceTokens...)
}

// Send takes a message subject and a message body and sends them to all previously set devices.
func (s *Service) Send(ctx context.Context, subject, message string) error {
	if len(s.deviceTokens) == 0 {
		return errors.New("no device tokens set")
	}

	if len(s.deviceTokens) == 1 {
		msg := &messaging.Message{
			Token: s.deviceTokens[0],
			Notification: &messaging.Notification{
				Title: subject,
				Body:  message,
			},
		}

		_, err := s.client.Send(ctx, msg)
		if err != nil {
			return fmt.Errorf("send message to FCM device with token %q: %w", s.deviceTokens[0], err)
		}
	} else {
		msg := &messaging.MulticastMessage{
			Tokens: s.deviceTokens,
			Notification: &messaging.Notification{
				Title: subject,
				Body:  message,
			},
		}

		_, err := s.client.SendMulticast(ctx, msg)
		if err != nil {
			return fmt.Errorf("send multicast message to FCM devices: %w", err)
		}
	}

	return nil
}
