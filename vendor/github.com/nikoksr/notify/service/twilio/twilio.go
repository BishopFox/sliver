package twilio

import (
	"context"
	"fmt"
	"net/url"

	"github.com/kevinburke/twilio-go"
)

// Compile-time check that twilio.MessageService satisfies twilioClient interface.
var _ twilioClient = &twilio.MessageService{}

// twilioClient abstracts twilio-go MessageService for writing unit tests.
type twilioClient interface {
	SendMessage(from, to, body string, mediaURLs []*url.URL) (*twilio.Message, error)
}

// Service encapsulates the Twilio Message Service client along with internal state for storing recipient phone numbers.
type Service struct {
	client twilioClient

	fromPhoneNumber string
	toPhoneNumbers  []string
}

// New returns a new instance of Twilio notification service.
func New(accountSID, authToken, fromPhoneNumber string) (*Service, error) {
	client := twilio.NewClient(accountSID, authToken, nil)

	s := &Service{
		client:          client.Messages,
		fromPhoneNumber: fromPhoneNumber,
		toPhoneNumbers:  []string{},
	}
	return s, nil
}

// AddReceivers takes strings of recipient phone numbers and appends them to the internal phone numbers slice.
// The Send method will send a given message to all those phone numbers.
func (s *Service) AddReceivers(phoneNumbers ...string) {
	s.toPhoneNumbers = append(s.toPhoneNumbers, phoneNumbers...)
}

// Send takes a message subject and a message body and sends them to all previously set phone numbers.
func (s *Service) Send(ctx context.Context, subject, message string) error {
	body := subject + "\n" + message

	for _, toPhoneNumber := range s.toPhoneNumbers {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:

			_, err := s.client.SendMessage(s.fromPhoneNumber, toPhoneNumber, body, []*url.URL{})
			if err != nil {
				return fmt.Errorf("send message to recipient %q: %w", toPhoneNumber, err)
			}
		}
	}

	return nil
}
