package textmagic

import (
	"context"
	"fmt"
	"strings"

	textMagic "github.com/textmagic/textmagic-rest-go-v2/v3"
)

// Service allow you to configure a TextMagic SDK client.
type Service struct {
	userName     string
	apiKey       string
	phoneNumbers []string
	client       *textMagic.APIClient
}

// New creates a new text magic client. Use your user-name and API key from
// https://my.textmagic.com/online/api/rest-api/keys.
func New(userName, apiKey string) *Service {
	config := textMagic.NewConfiguration()
	client := textMagic.NewAPIClient(config)

	return &Service{
		client:   client,
		userName: userName,
		apiKey:   apiKey,
	}
}

// AddReceivers adds the given phone numbers to the notifier.
func (s *Service) AddReceivers(phoneNumbers ...string) {
	s.phoneNumbers = append(s.phoneNumbers, phoneNumbers...)
}

// Send sends a SMS via TextMagic to all previously added receivers.
func (s *Service) Send(ctx context.Context, subject, message string) error {
	auth := context.WithValue(ctx, textMagic.ContextBasicAuth, textMagic.BasicAuth{
		UserName: s.userName,
		Password: s.apiKey,
	})

	text := subject + "\n" + message
	phones := strings.Join(s.phoneNumbers, ",")

	_, _, err := s.client.TextMagicAPI.SendMessage(auth).
		SendMessageInputObject(textMagic.SendMessageRequest{
			Text:   &text,
			Phones: &phones,
		}).
		Execute()
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}
