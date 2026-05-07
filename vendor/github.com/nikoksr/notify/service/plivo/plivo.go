package plivo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	plivo "github.com/plivo/plivo-go/v7"
)

// ClientOptions allow you to configure a Plivo SDK client.
type ClientOptions struct {
	AuthID    string // If empty, env variable PLIVO_AUTH_ID will be used
	AuthToken string // If empty, env variable PLIVO_AUTH_TOKEN will be used

	// Optional
	HTTPClient *http.Client // Bring Your Own Client
}

// MessageOptions allow you to configure options for sending a message.
type MessageOptions struct {
	Source string // a Plivo source phone number or a Plivo Powerpack UUID

	// Optional
	CallbackURL    string // URL to which status update callbacks for the message should be sent
	CallbackMethod string // The HTTP method to be used when calling CallbackURL - GET or POST(default)
}

// plivoMsgClient abstracts Plivo SDK for writing unit tests.
type plivoMsgClient interface {
	Create(plivo.MessageCreateParams) (*plivo.MessageCreateResponseBody, error)
}

// Service is a Plivo client.
type Service struct {
	client       plivoMsgClient
	mopts        MessageOptions
	destinations []string
}

// New creates a new instance of plivo service.
func New(cOpts *ClientOptions, mOpts *MessageOptions) (*Service, error) {
	if cOpts == nil {
		return nil, errors.New("client-options cannot be nil")
	}

	if mOpts == nil {
		return nil, errors.New("message-options cannot be nil")
	}

	if mOpts.Source == "" {
		return nil, errors.New("source cannot be empty")
	}

	client, err := plivo.NewClient(
		cOpts.AuthID,
		cOpts.AuthToken,
		&plivo.ClientOptions{
			HttpClient: cOpts.HTTPClient,
		},
	)
	if err != nil {
		return nil, err
	}

	return &Service{
		client: client.Messages,
		mopts:  *mOpts,
	}, nil
}

// AddReceivers adds the given destination phone numbers to the notifier.
func (s *Service) AddReceivers(phoneNumbers ...string) {
	s.destinations = append(s.destinations, phoneNumbers...)
}

// Send sends a SMS via Plivo to all previously added receivers.
func (s *Service) Send(ctx context.Context, subject, message string) error {
	text := subject + "\n" + message

	var dst string
	switch len(s.destinations) {
	case 0:
		return errors.New("no receivers added")
	case 1:
		dst = s.destinations[0]
	default:
		// multiple destinations, use bulk message syntax
		// see: https://www.plivo.com/docs/sms/api/message#bulk-messaging
		dst = strings.Join(s.destinations, "<")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, err := s.client.Create(plivo.MessageCreateParams{
			Dst:    dst,
			Text:   text,
			Src:    s.mopts.Source,
			URL:    s.mopts.CallbackURL,
			Method: s.mopts.CallbackMethod,
		})
		if err != nil {
			return fmt.Errorf("send SMS to %q: %w", dst, err)
		}
	}

	return nil
}
