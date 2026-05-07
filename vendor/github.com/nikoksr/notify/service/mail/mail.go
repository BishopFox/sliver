package mail

import (
	"context"
	"fmt"
	"net/smtp"
	"net/textproto"

	"github.com/jordan-wright/email"
)

// Mail struct holds necessary data to send emails.
type Mail struct {
	usePlainText      bool
	senderAddress     string
	smtpHostAddr      string
	smtpAuth          smtp.Auth
	receiverAddresses []string
}

// New returns a new instance of a Mail notification service.
func New(senderAddress, smtpHostAddress string) *Mail {
	return &Mail{
		usePlainText:      false,
		senderAddress:     senderAddress,
		smtpHostAddr:      smtpHostAddress,
		receiverAddresses: []string{},
	}
}

// BodyType is used to specify the format of the body.
type BodyType int

const (
	// PlainText is used to specify that the body is plain text.
	PlainText BodyType = iota
	// HTML is used to specify that the body is HTML.
	HTML
)

// AuthenticateSMTP authenticates you to send emails via smtp.
// Example values: "", "test@gmail.com", "password123", "smtp.gmail.com"
// For more information about smtp authentication, see here:
//
//	-> https://pkg.go.dev/net/smtp#PlainAuth
func (m *Mail) AuthenticateSMTP(identity, userName, password, host string) {
	m.smtpAuth = smtp.PlainAuth(identity, userName, password, host)
}

// AddReceivers takes email addresses and adds them to the internal address list. The Send method will send
// a given message to all those addresses.
func (m *Mail) AddReceivers(addresses ...string) {
	m.receiverAddresses = append(m.receiverAddresses, addresses...)
}

// BodyFormat can be used to specify the format of the body.
// Default BodyType is HTML.
func (m *Mail) BodyFormat(format BodyType) {
	switch format {
	case PlainText:
		m.usePlainText = true
	case HTML:
		m.usePlainText = false
	default:
		m.usePlainText = false
	}
}

func (m *Mail) newEmail(subject, message string) *email.Email {
	msg := &email.Email{
		To:      m.receiverAddresses,
		From:    m.senderAddress,
		Subject: subject,
		Headers: textproto.MIMEHeader{},
	}

	if m.usePlainText {
		msg.Text = []byte(message)
	} else {
		msg.HTML = []byte(message)
	}
	return msg
}

// Send takes a message subject and a message body and sends them to all previously set chats. Message body supports
// html as markup language.
func (m Mail) Send(ctx context.Context, subject, message string) error {
	msg := m.newEmail(subject, message)

	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		if err = msg.Send(m.smtpHostAddr, m.smtpAuth); err != nil {
			err = fmt.Errorf("send email: %w", err)
		}
	}

	return err
}
