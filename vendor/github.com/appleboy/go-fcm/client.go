package fcm

import (
	"context"
	"net/http"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

var scopes = []string{
	"https://www.googleapis.com/auth/firebase.messaging",
}

// Client abstracts the interaction between the application server and the
// FCM server via HTTP protocol. The developer must obtain an API key from the
// Google APIs Console page and pass it to the `Client` so that it can
// perform authorized requests on the application server's behalf.
// To send a message to one or more devices use the Client's Send.
//
// If the `HTTP` field is nil, a zeroed http.Client will be allocated and used
// to send messages.
//
// Authorization Scopes
// Requires one of the following OAuth scopes:
// - https://www.googleapis.com/auth/firebase.messaging
type Client struct {
	client          *messaging.Client
	serviceAcount   string
	projectID       string
	options         []option.ClientOption
	httpClient      *http.Client
	credentialsJSON []byte // credentialsJSON is the JSON representation of the service account credentials.
	debug           bool
}

// NewClient creates new Firebase Cloud Messaging Client based on API key and
// with default endpoint and http client.
func NewClient(ctx context.Context, opts ...Option) (*Client, error) {
	var err error
	c := &Client{}
	for _, o := range opts {
		if err := o(c); err != nil {
			return nil, err
		}
	}

	var conf *firebase.Config
	if c.serviceAcount != "" || c.projectID != "" {
		conf = &firebase.Config{
			ServiceAccountID: c.serviceAcount,
			ProjectID:        c.projectID,
		}
	}

	if c.debug {
		if c.httpClient == nil {
			c.httpClient = &http.Client{
				Transport: debugTransport{
					t: http.DefaultTransport,
				},
			}
		} else {
			c.httpClient.Transport = debugTransport{
				t: c.httpClient.Transport,
			}
		}
	}

	if c.httpClient != nil {
		ctxWithClient := context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
		creds, err := google.CredentialsFromJSON(ctxWithClient, c.credentialsJSON, scopes...)
		if err != nil {
			return nil, err
		}

		// And this is how we insert proxy for the Firebase calls. Initialize base transport with our proxy.
		tr := &oauth2.Transport{
			Source: creds.TokenSource,
			Base:   c.httpClient.Transport,
		}

		hCl := &http.Client{
			Transport: tr,
			Timeout:   10 * time.Second,
		}
		c.options = append(c.options, option.WithHTTPClient(hCl))
	}

	app, err := firebase.NewApp(ctx, conf, c.options...)
	if err != nil {
		return nil, err
	}

	c.client, err = app.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// SendWithContext sends a message to the FCM server without retrying in case of service
// unavailability. A non-nil error is returned if a non-recoverable error
// occurs (i.e. if the response status is not "200 OK").
// Behaves just like regular send, but uses external context.
func (c *Client) Send(ctx context.Context, message ...*messaging.Message) (*messaging.BatchResponse, error) {
	resp, err := c.client.SendEach(ctx, message)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// SendDryRun sends the messages in the given array via Firebase Cloud Messaging in the
// dry run (validation only) mode.
func (c *Client) SendDryRun(ctx context.Context, message ...*messaging.Message) (*messaging.BatchResponse, error) {
	resp, err := c.client.SendEachDryRun(ctx, message)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// SendEachForMulticast sends the given multicast message to all the FCM registration tokens specified.
func (c *Client) SendMulticast(ctx context.Context, message *messaging.MulticastMessage) (*messaging.BatchResponse, error) {
	resp, err := c.client.SendEachForMulticast(ctx, message)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// SendEachForMulticastDryRun sends the given multicast message to all the specified FCM registration
// tokens in the dry run (validation only) mode.
func (c *Client) SendMulticastDryRun(ctx context.Context, message *messaging.MulticastMessage) (*messaging.BatchResponse, error) {
	resp, err := c.client.SendEachForMulticastDryRun(ctx, message)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// SubscribeToTopic subscribes a list of registration tokens to a topic.
//
// The tokens list must not be empty, and have at most 1000 tokens.
func (c *Client) SubscribeTopic(ctx context.Context, tokens []string, topic string) (*messaging.TopicManagementResponse, error) {
	resp, err := c.client.SubscribeToTopic(ctx, tokens, topic)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// UnsubscribeFromTopic unsubscribes a list of registration tokens from a topic.
//
// The tokens list must not be empty, and have at most 1000 tokens.
func (c *Client) UnsubscribeTopic(ctx context.Context, tokens []string, topic string) (*messaging.TopicManagementResponse, error) {
	resp, err := c.client.UnsubscribeFromTopic(ctx, tokens, topic)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
