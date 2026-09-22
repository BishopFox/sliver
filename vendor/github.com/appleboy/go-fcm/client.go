package fcm

import (
	"context"
	"net/http"

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
// FCM server via the Firebase Cloud Messaging HTTP v1 API. Authenticate it with
// service-account credentials (WithCredentialsFile / WithCredentialsJSON), an
// OAuth2 token source (WithTokenSource), or Google Application Default
// Credentials so that it can perform authorized requests on the application
// server's behalf. To send a message to one or more devices use the Client's
// Send method.
//
// By default requests use a standard http.Client; supply your own with
// WithHTTPClient or route through a proxy with WithHTTPProxy.
//
// Authorization Scopes
// Requires one of the following OAuth scopes:
// - https://www.googleapis.com/auth/firebase.messaging
type Client struct {
	client          *messaging.Client
	serviceAccount  string
	projectID       string
	options         []option.ClientOption
	httpClient      *http.Client
	tokenSource     oauth2.TokenSource
	credentialsJSON []byte // credentialsJSON is the JSON representation of the service account credentials.
	debug           bool
}

// NewClient creates a new Firebase Cloud Messaging Client, applying the given
// options and using the default endpoint and http client unless overridden.
func NewClient(ctx context.Context, opts ...Option) (*Client, error) {
	c := &Client{}
	for _, o := range opts {
		if err := o(c); err != nil {
			return nil, err
		}
	}

	var conf *firebase.Config
	if c.serviceAccount != "" || c.projectID != "" {
		conf = &firebase.Config{
			ServiceAccountID: c.serviceAccount,
			ProjectID:        c.projectID,
		}
	}

	// Route Firebase API calls through a custom transport when the caller
	// supplied an http.Client, a proxy, or enabled debug logging. Because
	// option.WithHTTPClient bypasses the SDK's own auth wiring, re-apply the
	// selected credentials (service-account JSON or an explicit token source)
	// on top of that transport so debug/proxy stays compatible with every auth
	// method, not just inline JSON.
	if c.httpClient != nil || c.debug {
		base := http.DefaultTransport
		if c.httpClient != nil && c.httpClient.Transport != nil {
			base = c.httpClient.Transport
		}
		if c.debug {
			base = debugTransport{t: base}
		}

		// newHTTPClient wraps the given transport while preserving the caller's
		// other client settings (timeout, cookie jar, redirect policy) instead
		// of discarding them. It is reused for both the credential token fetch
		// and the final FCM client so neither silently drops those settings.
		newHTTPClient := func(rt http.RoundTripper) *http.Client {
			hc := &http.Client{Transport: rt}
			if c.httpClient != nil {
				hc.Timeout = c.httpClient.Timeout
				hc.CheckRedirect = c.httpClient.CheckRedirect
				hc.Jar = c.httpClient.Jar
			}
			return hc
		}

		var src oauth2.TokenSource
		switch {
		case len(c.credentialsJSON) > 0:
			ctxWithClient := context.WithValue(ctx, oauth2.HTTPClient, newHTTPClient(base))
			creds, err := google.CredentialsFromJSONWithType(
				ctxWithClient, c.credentialsJSON, google.ServiceAccount, scopes...,
			)
			if err != nil {
				return nil, err
			}
			src = creds.TokenSource
		case c.tokenSource != nil:
			src = c.tokenSource
		}

		transport := base
		if src != nil {
			transport = &oauth2.Transport{Source: src, Base: base}
		}

		c.options = append(c.options, option.WithHTTPClient(newHTTPClient(transport)))
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

// Send delivers one or more messages to the FCM server, sending each message in
// its own request via SendEach. The returned BatchResponse reports the outcome
// of every message in resp.Responses together with SuccessCount/FailureCount; a
// non-nil error is returned only when the batch as a whole cannot be sent, not
// when individual messages fail, so callers must inspect the response to detect
// per-message errors.
func (c *Client) Send(
	ctx context.Context,
	message ...*messaging.Message,
) (*messaging.BatchResponse, error) {
	return c.client.SendEach(ctx, message)
}

// SendDryRun sends the messages in the given array via Firebase Cloud Messaging in the
// dry run (validation only) mode.
func (c *Client) SendDryRun(
	ctx context.Context,
	message ...*messaging.Message,
) (*messaging.BatchResponse, error) {
	return c.client.SendEachDryRun(ctx, message)
}

// SendMulticast sends the given multicast message to all the FCM registration tokens specified.
func (c *Client) SendMulticast(
	ctx context.Context,
	message *messaging.MulticastMessage,
) (*messaging.BatchResponse, error) {
	return c.client.SendEachForMulticast(ctx, message)
}

// SendMulticastDryRun sends the given multicast message to all the specified FCM registration
// tokens in the dry run (validation only) mode.
func (c *Client) SendMulticastDryRun(
	ctx context.Context,
	message *messaging.MulticastMessage,
) (*messaging.BatchResponse, error) {
	return c.client.SendEachForMulticastDryRun(ctx, message)
}

// SubscribeTopic subscribes a list of registration tokens to a topic.
//
// The tokens list must not be empty, and have at most 1000 tokens.
func (c *Client) SubscribeTopic(
	ctx context.Context,
	tokens []string,
	topic string,
) (*messaging.TopicManagementResponse, error) {
	return c.client.SubscribeToTopic(ctx, tokens, topic)
}

// UnsubscribeTopic unsubscribes a list of registration tokens from a topic.
//
// The tokens list must not be empty, and have at most 1000 tokens.
func (c *Client) UnsubscribeTopic(
	ctx context.Context,
	tokens []string,
	topic string,
) (*messaging.TopicManagementResponse, error) {
	return c.client.UnsubscribeFromTopic(ctx, tokens, topic)
}
