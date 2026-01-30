// Copyright 2020 Enrico Hoffmann
// Copyright 2021 Adam Chalkley
//
// https://github.com/atc0005/go-teams-notify
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package goteamsnotify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// logger is a package logger that can be enabled from client code to allow
// logging output from this package when desired/needed for troubleshooting
var logger *log.Logger

// Known webhook URL prefixes for submitting messages to Microsoft Teams
const (
	WebhookURLOfficecomPrefix  = "https://outlook.office.com"
	WebhookURLOffice365Prefix  = "https://outlook.office365.com"
	WebhookURLOrgWebhookPrefix = "https://example.webhook.office.com"
)

// Known Workflow URL patterns for submitting messages to Microsoft Teams.
const (
	WorkflowURLBaseDomain = `^https:\/\/(?:.*)(:?\.azure-api|logic\.azure|api\.powerplatform)\.(?:com|net)`
)

// DisableWebhookURLValidation is a special keyword used to indicate to
// validation function(s) that webhook URL validation should be disabled.
//
// Deprecated: prefer using API.SkipWebhookURLValidationOnSend(bool) method instead
const DisableWebhookURLValidation string = "DISABLE_WEBHOOK_URL_VALIDATION"

// Regular Expression related constants that we can use to validate incoming
// webhook URLs provided by the user.
const (

	// DefaultWebhookURLValidationPattern is a minimal regex for matching known valid
	// webhook URL prefix patterns.
	DefaultWebhookURLValidationPattern = `^https:\/\/(?:.*\.webhook|outlook)\.office(?:365)?\.com`

	// Note: The regex allows for capital letters in the GUID patterns. This is
	// allowed based on light testing which shows that mixed case works and the
	// assumption that since Teams and Office 365 are Microsoft products case
	// would be ignored (e.g., Windows, IIS do not consider 'A' and 'a' to be
	// different).
	// webhookURLRegex           = `^https:\/\/(?:.*\.webhook|outlook)\.office(?:365)?\.com\/webhook(?:b2)?\/[-a-zA-Z0-9]{36}@[-a-zA-Z0-9]{36}\/IncomingWebhook\/[-a-zA-Z0-9]{32}\/[-a-zA-Z0-9]{36}$`

	// webhookURLSubURIWebhookPrefix         = "webhook"
	// webhookURLSubURIWebhookb2Prefix       = "webhookb2"
	// webhookURLOfficialDocsSampleURI       = "a1269812-6d10-44b1-abc5-b84f93580ba0@9e7b80c7-d1eb-4b52-8582-76f921e416d9/IncomingWebhook/3fdd6767bae44ac58e5995547d66a4e4/f332c8d9-3397-4ac5-957b-b8e3fc465a8c"
)

// ExpectedWebhookURLResponseText represents the expected response text
// provided by the remote webhook endpoint when submitting messages.
const ExpectedWebhookURLResponseText string = "1"

// DefaultWebhookSendTimeout specifies how long the message operation may take
// before it times out and is cancelled.
const DefaultWebhookSendTimeout = 5 * time.Second

// DefaultUserAgent is the project-specific user agent used when submitting
// messages unless overridden by client code. This replaces the Go default
// user agent value of "Go-http-client/1.1".
//
// The major.minor numbers reflect when this project first diverged from the
// "upstream" or parent project.
const DefaultUserAgent string = "go-teams-notify/2.2"

// ErrWebhookURLUnexpected is returned when a provided webhook URL does
// not match a set of confirmed webhook URL patterns.
var ErrWebhookURLUnexpected = errors.New("webhook URL does not match one of expected patterns")

// ErrWebhookURLUnexpectedPrefix is returned when a provided webhook URL does
// not match a set of confirmed webhook URL prefixes.
//
// Deprecated: Use ErrWebhookURLUnexpected instead.
var ErrWebhookURLUnexpectedPrefix = ErrWebhookURLUnexpected

// ErrInvalidWebhookURLResponseText is returned when the remote webhook
// endpoint indicates via response text that a message submission was
// unsuccessful.
var ErrInvalidWebhookURLResponseText = errors.New("invalid webhook URL response text")

// API is the legacy interface representing a client used to submit messages
// to a Microsoft Teams channel.
type API interface {
	Send(webhookURL string, webhookMessage MessageCard) error
	SendWithContext(ctx context.Context, webhookURL string, webhookMessage MessageCard) error
	SendWithRetry(ctx context.Context, webhookURL string, webhookMessage MessageCard, retries int, retriesDelay int) error
	SkipWebhookURLValidationOnSend(skip bool) API
	AddWebhookURLValidationPatterns(patterns ...string) API
	ValidateWebhook(webhookURL string) error
}

// MessageSender describes the behavior of a baseline Microsoft Teams client.
//
// An unexported method is used to prevent client code from implementing this
// interface in order to support future changes (and not violate backwards
// compatibility).
type MessageSender interface {
	HTTPClient() *http.Client
	UserAgent() string
	ValidateWebhook(webhookURL string) error

	// A private method to prevent client code from implementing the interface
	// so that any future changes to it will not violate backwards
	// compatibility.
	private()
}

// messagePreparer is a message type that supports marshaling its fields
// as preparation for delivery to an endpoint.
type messagePreparer interface {
	Prepare() error
}

// messageValidator is a message type that provides validation of its format.
type messageValidator interface {
	Validate() error
}

// TeamsMessage is the interface shared by all supported message formats for
// submission to a Microsoft Teams channel.
type TeamsMessage interface {
	messagePreparer
	messageValidator

	Payload() io.Reader
}

// teamsClient is the legacy client used for submitting messages to a
// Microsoft Teams channel.
type teamsClient struct {
	httpClient                   *http.Client
	userAgent                    string
	webhookURLValidationPatterns []string
	skipWebhookURLValidation     bool
}

// TeamsClient provides functionality for submitting messages to a Microsoft
// Teams channel.
type TeamsClient struct {
	httpClient                   *http.Client
	userAgent                    string
	webhookURLValidationPatterns []string
	skipWebhookURLValidation     bool
}

func init() {
	// Disable logging output by default unless client code explicitly
	// requests it
	logger = log.New(os.Stderr, "[goteamsnotify] ", 0)
	logger.SetOutput(ioutil.Discard)
}

// EnableLogging enables logging output from this package. Output is muted by
// default unless explicitly requested (by calling this function).
func EnableLogging() {
	logger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logger.SetOutput(os.Stderr)
}

// DisableLogging reapplies default package-level logging settings of muting
// all logging output.
func DisableLogging() {
	logger.SetFlags(0)
	logger.SetOutput(ioutil.Discard)
}

// NewClient - create a brand new client for MS Teams notify
//
// Deprecated: use NewTeamsClient() function instead.
func NewClient() API {
	client := teamsClient{
		httpClient: &http.Client{
			// We're using a context instead of setting this directly
			// Timeout: DefaultWebhookSendTimeout,
		},
		skipWebhookURLValidation: false,
	}
	return &client
}

// NewTeamsClient constructs a minimal client for submitting messages to a
// Microsoft Teams channel.
func NewTeamsClient() *TeamsClient {
	client := TeamsClient{
		httpClient: &http.Client{
			// We're using a context instead of setting this directly
			// Timeout: DefaultWebhookSendTimeout,
		},
		skipWebhookURLValidation: false,
	}
	return &client
}

// private prevents client code from implementing the MessageSender interface
// so that any future changes to it will not violate backwards compatibility.
func (c *teamsClient) private() {}

// private prevents client code from implementing the MessageSender interface
// so that any future changes to it will not violate backwards compatibility.
func (c *TeamsClient) private() {}

// SetHTTPClient accepts a custom http.Client value which replaces the
// existing default http.Client.
func (c *TeamsClient) SetHTTPClient(httpClient *http.Client) *TeamsClient {
	c.httpClient = httpClient

	return c
}

// SetUserAgent accepts a custom user agent string. This custom user agent is
// used when submitting messages to Microsoft Teams.
func (c *TeamsClient) SetUserAgent(userAgent string) *TeamsClient {
	c.userAgent = userAgent

	return c
}

// UserAgent returns the configured user agent string for the client. If a
// custom value is not set the default package user agent is returned.
//
// Deprecated: use TeamsClient.UserAgent() method instead.
func (c *teamsClient) UserAgent() string {
	switch {
	case c.userAgent != "":
		return c.userAgent
	default:
		return DefaultUserAgent
	}
}

// UserAgent returns the configured user agent string for the client. If a
// custom value is not set the default package user agent is returned.
func (c *TeamsClient) UserAgent() string {
	switch {
	case c.userAgent != "":
		return c.userAgent
	default:
		return DefaultUserAgent
	}
}

// AddWebhookURLValidationPatterns collects given patterns for validation of
// the webhook URL.
//
// Deprecated: use TeamsClient.AddWebhookURLValidationPatterns() method instead.
func (c *teamsClient) AddWebhookURLValidationPatterns(patterns ...string) API {
	c.webhookURLValidationPatterns = append(c.webhookURLValidationPatterns, patterns...)
	return c
}

// AddWebhookURLValidationPatterns collects given patterns for validation of
// the webhook URL.
func (c *TeamsClient) AddWebhookURLValidationPatterns(patterns ...string) *TeamsClient {
	c.webhookURLValidationPatterns = append(c.webhookURLValidationPatterns, patterns...)
	return c
}

// HTTPClient returns the internal pointer to an http.Client. This can be used
// to further modify specific http.Client field values.
//
// Deprecated: use TeamsClient.HTTPClient() method instead.
func (c *teamsClient) HTTPClient() *http.Client {
	return c.httpClient
}

// HTTPClient returns the internal pointer to an http.Client. This can be used
// to further modify specific http.Client field values.
func (c *TeamsClient) HTTPClient() *http.Client {
	return c.httpClient
}

// Send is a wrapper function around the SendWithContext method in order to
// provide backwards compatibility.
//
// Deprecated: use TeamsClient.Send() method instead.
func (c *teamsClient) Send(webhookURL string, webhookMessage MessageCard) error {
	// Create context that can be used to emulate existing timeout behavior.
	ctx, cancel := context.WithTimeout(context.Background(), DefaultWebhookSendTimeout)
	defer cancel()

	return sendWithContext(ctx, c, webhookURL, &webhookMessage)
}

// Send is a wrapper function around the SendWithContext method in order to
// provide backwards compatibility.
func (c *TeamsClient) Send(webhookURL string, message TeamsMessage) error {
	// Create context that can be used to emulate existing timeout behavior.
	ctx, cancel := context.WithTimeout(context.Background(), DefaultWebhookSendTimeout)
	defer cancel()

	return sendWithContext(ctx, c, webhookURL, message)
}

// SendWithContext submits a given message to a Microsoft Teams channel using
// the provided webhook URL. The http client request honors the cancellation
// or timeout of the provided context.
//
// Deprecated: use TeamsClient.SendWithContext() method instead.
func (c *teamsClient) SendWithContext(ctx context.Context, webhookURL string, webhookMessage MessageCard) error {
	return sendWithContext(ctx, c, webhookURL, &webhookMessage)
}

// SendWithContext submits a given message to a Microsoft Teams channel using
// the provided webhook URL. The http client request honors the cancellation
// or timeout of the provided context.
func (c *TeamsClient) SendWithContext(ctx context.Context, webhookURL string, message TeamsMessage) error {
	return sendWithContext(ctx, c, webhookURL, message)
}

// SendWithRetry provides message retry support when submitting messages to a
// Microsoft Teams channel. The caller is responsible for providing the
// desired context timeout, the number of retries and retries delay.
//
// Deprecated: use TeamsClient.SendWithRetry() method instead.
func (c *teamsClient) SendWithRetry(ctx context.Context, webhookURL string, webhookMessage MessageCard, retries int, retriesDelay int) error {
	return sendWithRetry(ctx, c, webhookURL, &webhookMessage, retries, retriesDelay)
}

// SendWithRetry provides message retry support when submitting messages to a
// Microsoft Teams channel. The caller is responsible for providing the
// desired context timeout, the number of retries and retries delay.
func (c *TeamsClient) SendWithRetry(ctx context.Context, webhookURL string, message TeamsMessage, retries int, retriesDelay int) error {
	return sendWithRetry(ctx, c, webhookURL, message, retries, retriesDelay)
}

// SkipWebhookURLValidationOnSend allows the caller to optionally disable
// webhook URL validation.
//
// Deprecated: use TeamsClient.SkipWebhookURLValidationOnSend() method instead.
func (c *teamsClient) SkipWebhookURLValidationOnSend(skip bool) API {
	c.skipWebhookURLValidation = skip
	return c
}

// SkipWebhookURLValidationOnSend allows the caller to optionally disable
// webhook URL validation.
func (c *TeamsClient) SkipWebhookURLValidationOnSend(skip bool) *TeamsClient {
	c.skipWebhookURLValidation = skip
	return c
}

// prepareRequest is a helper function that prepares a http.Request (including
// all desired headers) in order to submit a given prepared message to an
// endpoint.
func prepareRequest(ctx context.Context, userAgent string, webhookURL string, preparedMessage io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, preparedMessage)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("User-Agent", userAgent)

	return req, nil
}

// processResponse is a helper function responsible for validating a response
// from an endpoint after submitting a message.
func processResponse(response *http.Response) (string, error) {
	// Get the response body, then convert to string for use with extended
	// error messages
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Println(err)

		return "", err
	}
	responseString := string(responseData)

	// TODO: Refactor for v3 series once O365 connector support is dropped.
	switch {
	// 400 Bad Response is likely an indicator that we failed to provide a
	// required field in our JSON payload. For example, when leaving out the
	// top level MessageCard Summary or Text field, the remote API returns
	// "Summary or Text is required." as a text string. We include that
	// response text in the error message that we return to the caller.
	case response.StatusCode >= 299:
		err = fmt.Errorf("error on notification: %v, %q", response.Status, responseString)

		logger.Println(err)

		return "", err

	case response.StatusCode == 202:
		// 202 Accepted response is expected for Workflow connector URL
		// submissions.

		logger.Println("202 Accepted response received as expected for workflow connector")

		return responseString, nil

	// DEPRECATED
	//
	// See https://github.com/atc0005/go-teams-notify/issues/262
	//
	// Microsoft Teams developers have indicated that receiving a 200 status
	// code when submitting payloads to O365 connectors is insufficient to
	// confirm that a message was successfully submitted.
	//
	// Instead, clients should ensure that a specific response string was also
	// returned along with a 200 status code to confirm that a message was
	// sent successfully. Because there is a chance that unintentional
	// whitespace could be included, we explicitly strip it out.
	//
	// See atc0005/go-teams-notify#59 for more information.
	case responseString != strings.TrimSpace(ExpectedWebhookURLResponseText):
		logger.Printf(
			"StatusCode: %v, Status: %v\n", response.StatusCode, response.Status,
		)
		logger.Printf("ResponseString: %v\n", responseString)

		err = fmt.Errorf(
			"got %q, expected %q: %w",
			responseString,
			ExpectedWebhookURLResponseText,
			ErrInvalidWebhookURLResponseText,
		)

		logger.Println(err)

		return "", err

	default:
		return responseString, nil
	}
}

// validateWebhook applies webhook URL validation unless explicitly disabled.
func validateWebhook(webhookURL string, skipWebhookValidation bool, patterns []string) error {
	if skipWebhookValidation || webhookURL == DisableWebhookURLValidation {
		logger.Printf("validateWebhook: Webhook URL will not be validated: %#v\n", webhookURL)

		return nil
	}

	u, err := url.Parse(webhookURL)
	if err != nil {
		return fmt.Errorf("unable to parse webhook URL %q: %w", webhookURL, err)
	}

	if len(patterns) == 0 {
		patterns = []string{
			DefaultWebhookURLValidationPattern,
			WorkflowURLBaseDomain,
		}
	}

	// Indicate passing validation if at least one pattern matches.
	for _, pat := range patterns {
		matched, err := regexp.MatchString(pat, webhookURL)
		if err != nil {
			return err
		}
		if matched {
			logger.Printf("Pattern %v matched", pat)

			return nil
		}
	}

	return fmt.Errorf(
		"%w; got: %q, patterns: %s",
		ErrWebhookURLUnexpected,
		u.String(),
		strings.Join(patterns, ","),
	)
}

// ValidateWebhook applies webhook URL validation unless explicitly disabled.
//
// Deprecated: use TeamsClient.ValidateWebhook() method instead.
func (c *teamsClient) ValidateWebhook(webhookURL string) error {
	return validateWebhook(webhookURL, c.skipWebhookURLValidation, c.webhookURLValidationPatterns)
}

// ValidateWebhook applies webhook URL validation unless explicitly disabled.
func (c *TeamsClient) ValidateWebhook(webhookURL string) error {
	return validateWebhook(webhookURL, c.skipWebhookURLValidation, c.webhookURLValidationPatterns)
}

// sendWithContext submits a given message to a Microsoft Teams channel using
// the provided webhook URL and client. The http client request honors the
// cancellation or timeout of the provided context.
func sendWithContext(ctx context.Context, client MessageSender, webhookURL string, message TeamsMessage) error {
	logger.Printf("sendWithContext: Webhook message received: %#v\n", message)

	if err := client.ValidateWebhook(webhookURL); err != nil {
		return fmt.Errorf(
			"failed to validate webhook URL: %w",
			err,
		)
	}

	if err := message.Validate(); err != nil {
		return fmt.Errorf(
			"failed to validate message: %w",
			err,
		)
	}

	if err := message.Prepare(); err != nil {
		return fmt.Errorf(
			"failed to prepare message: %w",
			err,
		)
	}

	req, err := prepareRequest(ctx, client.UserAgent(), webhookURL, message.Payload())
	if err != nil {
		return fmt.Errorf(
			"failed to prepare request: %w",
			err,
		)
	}

	// Submit message to endpoint.
	res, err := client.HTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf(
			"failed to submit message: %w",
			err,
		)
	}

	// Make sure that we close the response body once we're done with it
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Printf("error closing response body: %v", err)
		}
	}()

	responseText, err := processResponse(res)
	if err != nil {
		return fmt.Errorf(
			"failed to process response: %w",
			err,
		)
	}

	logger.Printf("sendWithContext: Response string from Microsoft Teams API: %v\n", responseText)

	return nil
}

// sendWithRetry provides message retry support when submitting messages to a
// Microsoft Teams channel. The caller is responsible for providing the
// desired context timeout, the number of retries and retries delay.
func sendWithRetry(ctx context.Context, client MessageSender, webhookURL string, message TeamsMessage, retries int, retriesDelay int) error {
	var result error

	// initial attempt + number of specified retries
	attemptsAllowed := 1 + retries

	// attempt to send message to Microsoft Teams, retry specified number of
	// times before giving up
	for attempt := 1; attempt <= attemptsAllowed; attempt++ {
		// the result from the last attempt is returned to the caller
		result = sendWithContext(ctx, client, webhookURL, message)

		switch {
		case result != nil:

			logger.Printf(
				"sendWithRetry: Attempt %d of %d to send message failed: %v",
				attempt,
				attemptsAllowed,
				result,
			)

			if ctx.Err() != nil {
				errMsg := fmt.Errorf(
					"sendWithRetry: context cancelled or expired: %v; "+
						"aborting message submission after %d of %d attempts: %w",
					ctx.Err().Error(),
					attempt,
					attemptsAllowed,
					result,
				)

				logger.Println(errMsg)

				return errMsg
			}

			ourRetryDelay := time.Duration(retriesDelay) * time.Second

			logger.Printf(
				"sendWithRetry: Context not cancelled yet, applying retry delay of %v",
				ourRetryDelay,
			)
			time.Sleep(ourRetryDelay)

		default:
			logger.Printf(
				"sendWithRetry: successfully sent message after %d of %d attempts\n",
				attempt,
				attemptsAllowed,
			)

			// No further retries needed
			return nil
		}
	}

	return result
}

// old deprecated helper functions --------------------------------------------------------------------------------------------------------------

// IsValidInput is a validation "wrapper" function. This function is intended
// to run current validation checks and offer easy extensibility for future
// validation requirements.
//
// Deprecated: use API.ValidateWebhook() and MessageCard.Validate()
// methods instead.
func IsValidInput(webhookMessage MessageCard, webhookURL string) (bool, error) {
	// validate url
	if valid, err := IsValidWebhookURL(webhookURL); !valid {
		return false, err
	}

	// validate message
	if valid, err := IsValidMessageCard(webhookMessage); !valid {
		return false, err
	}

	return true, nil
}

// IsValidWebhookURL performs validation checks on the webhook URL used to
// submit messages to Microsoft Teams.
//
// Deprecated: use API.ValidateWebhook() method instead.
func IsValidWebhookURL(webhookURL string) (bool, error) {
	c := teamsClient{}
	err := c.ValidateWebhook(webhookURL)
	return err == nil, err
}

// IsValidMessageCard performs validation/checks for known issues with
// MessardCard values.
//
// Deprecated: use MessageCard.Validate() instead.
func IsValidMessageCard(webhookMessage MessageCard) (bool, error) {
	err := webhookMessage.Validate()
	return err == nil, err
}
