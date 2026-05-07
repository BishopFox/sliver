package twitter

import (
	"context"
	"fmt"

	"github.com/dghubble/oauth1"
	"github.com/drswork/go-twitter/twitter"
)

// Twitter struct holds necessary data to communicate with the Twitter API.
type Twitter struct {
	client     *twitter.Client
	twitterIDs []string
}

// Credentials contains the authentication credentials needed for twitter
// api access
//
// ConsumerKey and ConsumerSecret can be thought of as the user name
// and password that represents your Twitter developer app when making
// API requests.
//
// An access token and access token secret are user-specific credentials
// used to authenticate OAuth 1.0a API requests.
// They specify the Twitter account the request is made on behalf of.
//
// See https://developer.twitter.com/en/docs/authentication/oauth-1-0a for more details.
type Credentials struct {
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
}

// New returns a new instance of a Twitter service.
// For more information about Twitter access token:
//
//	-> https://developer.twitter.com/en/docs/authentication/oauth-1-0a/obtaining-user-access-tokens
func New(credentials Credentials) (*Twitter, error) {
	config := oauth1.NewConfig(credentials.ConsumerKey, credentials.ConsumerSecret)
	token := oauth1.NewToken(credentials.AccessToken, credentials.AccessTokenSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)

	// Verify Credentials
	verifyParams := &twitter.AccountVerifyParams{
		SkipStatus:   twitter.Bool(true),
		IncludeEmail: twitter.Bool(true),
	}

	// we can retrieve the user and verify if the credentials
	// we have used successfully allow us to log in!
	_, _, err := client.Accounts.VerifyCredentials(verifyParams)
	if err != nil {
		return nil, err
	}

	t := &Twitter{
		client:     client,
		twitterIDs: []string{},
	}

	return t, nil
}

// AddReceivers takes TwitterIds and adds them to the internal twitterIDs list.
func (t *Twitter) AddReceivers(twitterIDs ...string) {
	t.twitterIDs = append(t.twitterIDs, twitterIDs...)
}

// Send takes a message subject and a message body and sends them to all previously set twitterIDs as a DM.
// See
// https://developer.twitter.com/en/docs/twitter-api/v1/direct-messages/sending-and-receiving/api-reference/new-event
func (t Twitter) Send(ctx context.Context, subject, message string) error {
	directMessageData := &twitter.DirectMessageData{
		Text: subject + "\n" + message,
	}

	for _, twitterID := range t.twitterIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			directMessageTarget := &twitter.DirectMessageTarget{
				RecipientID: twitterID,
			}
			directMessageEvent := &twitter.DirectMessageEvent{
				Type: "message_create",
				Message: &twitter.DirectMessageEventMessage{
					Target: directMessageTarget,
					Data:   directMessageData,
				},
			}

			directMessageParams := &twitter.DirectMessageEventsNewParams{
				Event: directMessageEvent,
			}

			if _, _, err := t.client.DirectMessages.EventsNew(directMessageParams); err != nil {
				return fmt.Errorf("send message to %q: %w", twitterID, err)
			}
		}
	}

	return nil
}
