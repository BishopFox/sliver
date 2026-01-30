package sendgrid

import (
	"encoding/base64"

	"github.com/sendgrid/rest"
)

// TwilioEmailOptions for GetTwilioEmailRequest
type TwilioEmailOptions struct {
	Username string
	Password string
	Endpoint string
	Host     string
}

// NewTwilioEmailSendClient constructs a new Twilio Email client given a username and password
func NewTwilioEmailSendClient(username, password string) *Client {
	request := GetTwilioEmailRequest(TwilioEmailOptions{Username: username, Password: password, Endpoint: "/v3/mail/send"})
	request.Method = "POST"
	return &Client{request}
}

// GetTwilioEmailRequest create Request
// @return [Request] a default request object
func GetTwilioEmailRequest(twilioEmailOptions TwilioEmailOptions) rest.Request {
	credentials := twilioEmailOptions.Username + ":" + twilioEmailOptions.Password
	encodedCreds := base64.StdEncoding.EncodeToString([]byte(credentials))

	options := options{
		Auth:     "Basic " + encodedCreds,
		Endpoint: twilioEmailOptions.Endpoint,
		Host:     twilioEmailOptions.Host,
	}

	if options.Host == "" {
		options.Host = "https://email.twilio.com"
	}

	return requestNew(options)
}
