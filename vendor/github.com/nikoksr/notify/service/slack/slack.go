package slack

import (
	"context"
	"fmt"

	"github.com/slack-go/slack"
)

type slackClient interface {
	PostMessageContext(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error)
}

// Compile-time check to ensure that slack.Client implements the slackClient interface.
var _ slackClient = new(slack.Client)

// Slack struct holds necessary data to communicate with the Slack API.
type Slack struct {
	client     slackClient
	channelIDs []string
}

// New returns a new instance of a Slack notification service.
// For more information about slack api token:
//
//	-> https://pkg.go.dev/github.com/slack-go/slack#New
func New(apiToken string) *Slack {
	client := slack.New(apiToken)

	s := &Slack{
		client:     client,
		channelIDs: []string{},
	}

	return s
}

// AddReceivers takes Slack channel IDs and adds them to the internal channel ID list. The Send method will send
// a given message to all those channels.
func (s *Slack) AddReceivers(channelIDs ...string) {
	s.channelIDs = append(s.channelIDs, channelIDs...)
}

// Send takes a message subject and a message body and sends them to all previously set channels.
// you will need a slack app with the chat:write.public and chat:write permissions.
// see https://api.slack.com/
func (s Slack) Send(ctx context.Context, subject, message string) error {
	fullMessage := subject + "\n" + message // Treating subject as message title

	for _, channelID := range s.channelIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, _, err := s.client.PostMessageContext(
				ctx,
				channelID,
				slack.MsgOptionText(fullMessage, false),
			)
			if err != nil {
				return fmt.Errorf("send message to channel %q: %w", channelID, err)
			}
		}
	}

	return nil
}
