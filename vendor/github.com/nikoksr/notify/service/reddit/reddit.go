package reddit

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/caarlos0/go-reddit/v3/reddit"
)

type redditMessageClient interface {
	Send(context.Context, *reddit.SendMessageRequest) (*reddit.Response, error)
}

// Compile-time check to ensure that reddit.MessageService implements the redditMessageClient interface.
var _ redditMessageClient = new(reddit.MessageService)

// Reddit struct holds necessary data to communicate with the Reddit API.
type Reddit struct {
	client     redditMessageClient
	recipients []string
}

// New returns a new instance of a Reddit notification service.
// For more information on obtaining client credentials:
//
//	-> https://github.com/reddit-archive/reddit/wiki/OAuth2
func New(clientID, clientSecret, username, password string) (*Reddit, error) {
	// Disable HTTP2 in http client
	// Details:
	// https://www.reddit.com/r/redditdev/comments/t8e8hc/getting_nothing_but_429_responses_when_using_go/i18yga2/
	h := http.Client{
		Transport: &http.Transport{
			TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
		},
	}
	rClient, err := reddit.NewClient(
		reddit.Credentials{
			ID:       clientID,
			Secret:   clientSecret,
			Username: username,
			Password: password,
		},
		reddit.WithHTTPClient(&h),
		reddit.WithUserAgent("github.com/nikoksr/notify"),
	)
	if err != nil {
		return nil, fmt.Errorf("create Reddit client: %w", err)
	}

	r := &Reddit{
		client:     rClient.Message,
		recipients: []string{},
	}

	return r, nil
}

// AddReceivers takes Reddit usernames and adds them to the internal recipient list. The Send method will send
// a given message to all of those users.
func (r *Reddit) AddReceivers(recipients ...string) {
	r.recipients = append(r.recipients, recipients...)
}

// Send takes a message subject and a message body and sends them to all previously set recipients.
func (r *Reddit) Send(ctx context.Context, subject, message string) error {
	for i := range r.recipients {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			m := reddit.SendMessageRequest{
				To:      r.recipients[i],
				Subject: subject,
				Text:    message,
			}

			if _, err := r.client.Send(ctx, &m); err != nil {
				return fmt.Errorf("send message to user %q: %w", r.recipients[i], err)
			}
		}
	}

	return nil
}
