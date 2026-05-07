package sendgrid

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendGrid struct holds necessary data to communicate with the SendGrid API.
type SendGrid struct {
	usePlainText      bool
	client            *sendgrid.Client
	senderAddress     string
	senderName        string
	receiverAddresses []string
}

// BodyType is used to specify the format of the body.
type BodyType int

const (
	// PlainText is used to specify that the body is plain text.
	PlainText BodyType = iota
	// HTML is used to specify that the body is HTML.
	HTML
)

// New returns a new instance of a SendGrid notification service.
// You will need a SendGrid API key.
// See https://sendgrid.com/docs/for-developers/sending-email/api-getting-started/
func New(apiKey, senderAddress, senderName string) *SendGrid {
	return &SendGrid{
		client:            sendgrid.NewSendClient(apiKey),
		senderAddress:     senderAddress,
		senderName:        senderName,
		receiverAddresses: []string{},
		usePlainText:      false,
	}
}

// AddReceivers takes email addresses and adds them to the internal address list. The Send method will send
// a given message to all those addresses.
func (s *SendGrid) AddReceivers(addresses ...string) {
	s.receiverAddresses = append(s.receiverAddresses, addresses...)
}

// BodyFormat can be used to specify the format of the body.
// Default BodyType is HTML.
func (s *SendGrid) BodyFormat(format BodyType) {
	switch format {
	case PlainText:
		s.usePlainText = true
	case HTML:
		s.usePlainText = false
	default:
		s.usePlainText = false
	}
}

// Send takes a message subject and a message body and sends them to all previously set chats. Message body supports
// html as markup language.
func (s SendGrid) Send(ctx context.Context, subject, message string) error {
	from := mail.NewEmail(s.senderName, s.senderAddress)
	var contentType string
	if s.usePlainText {
		contentType = "text/plain"
	} else {
		contentType = "text/html"
	}
	content := mail.NewContent(contentType, message)

	// Create a new personalization instance to be able to add multiple receiver addresses.
	personalization := mail.NewPersonalization()
	personalization.Subject = subject

	for _, receiverAddress := range s.receiverAddresses {
		personalization.AddTos(mail.NewEmail(receiverAddress, receiverAddress))
	}

	mailMessage := mail.NewV3Mail()
	mailMessage.AddPersonalizations(personalization)
	mailMessage.AddContent(content)
	mailMessage.SetFrom(from)

	resp, err := s.client.SendWithContext(ctx, mailMessage)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	if resp.StatusCode != http.StatusAccepted {
		return errors.New("the SendGrid endpoint did not accept the message")
	}

	return nil
}
