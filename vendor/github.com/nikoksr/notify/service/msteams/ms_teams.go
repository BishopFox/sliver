package msteams

import (
	"context"
	"fmt"
	"net/http"

	teams "github.com/atc0005/go-teams-notify/v2"
	"github.com/atc0005/go-teams-notify/v2/adaptivecard"
)

type teamsClient interface {
	// https://pkg.go.dev/github.com/atc0005/go-teams-notify/v2#TeamsClient.SendWithContext
	SendWithContext(ctx context.Context, webhookURL string, message teams.TeamsMessage) error
	// https://pkg.go.dev/github.com/atc0005/go-teams-notify/v2#TeamsClient.SkipWebhookURLValidationOnSend
	SkipWebhookURLValidationOnSend(skip bool) *teams.TeamsClient
	// https://pkg.go.dev/github.com/atc0005/go-teams-notify/v2#TeamsClient.SetHTTPClient
	SetHTTPClient(httpClient *http.Client) *teams.TeamsClient
	// https://pkg.go.dev/github.com/atc0005/go-teams-notify/v2#TeamsClient.SetUserAgent
	SetUserAgent(userAgent string) *teams.TeamsClient
}

// Compile-time check to ensure that teams.Client implements the teamsClient interface.
var _ teamsClient = teams.NewTeamsClient()

// MSTeams struct holds necessary data to communicate with the MSTeams API.
type MSTeams struct {
	client   teamsClient
	webHooks []string

	wrapText bool
}

// New returns a new instance of a MSTeams notification service.
// For more information about telegram api token:
//
//	-> https://github.com/atc0005/go-teams-notify#example-basic
func New() *MSTeams {
	client := teams.NewTeamsClient()

	m := &MSTeams{
		client:   client,
		webHooks: []string{},
	}

	return m
}

// DisableWebhookValidation disables the validation of webhook URLs, including the validation of known prefixes so that
// custom/private webhook URL endpoints can be used (e.g., testing purposes).
// For more information about telegram api token:
//
//	-> https://github.com/atc0005/go-teams-notify#example-disable-webhook-url-prefix-validation
func (m *MSTeams) DisableWebhookValidation() {
	m.client.SkipWebhookURLValidationOnSend(true)
}

// WithWrapText sets the wrapText field to the provided value. This is disabled by default.
func (m *MSTeams) WithWrapText(wrapText bool) {
	m.wrapText = wrapText
}

// AddReceivers takes MSTeams channel web-hooks and adds them to the internal web-hook list. The Send method will send
// a given message to all those chats.
func (m *MSTeams) AddReceivers(webHooks ...string) {
	m.webHooks = append(m.webHooks, webHooks...)
}

// SetUseragent allows the user to set a custom user agent.
func (m *MSTeams) SetUseragent(userAgent string) {
	m.client = m.client.SetUserAgent(userAgent)
}

// SetHTTPClient allows the user to set a custom http client.
func (m *MSTeams) SetHTTPClient(httpClient *http.Client) {
	m.client = m.client.SetHTTPClient(httpClient)
}

// Send accepts a subject and a message body and sends them to all previously specified channels. Message body supports
// html as markup language.
// For more information about telegram api token:
//
//	-> https://github.com/atc0005/go-teams-notify#example-basic
func (m MSTeams) Send(ctx context.Context, subject, message string) error {
	msg, err := adaptivecard.NewSimpleMessage(message, subject, m.wrapText)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	for _, webHook := range m.webHooks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err = m.client.SendWithContext(ctx, webHook, msg); err != nil {
				return fmt.Errorf("send message to channel %q: %w", webHook, err)
			}
		}
	}

	return nil
}
