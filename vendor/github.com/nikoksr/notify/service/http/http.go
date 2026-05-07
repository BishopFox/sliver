package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nikoksr/notify"
)

type (
	// PreSendHookFn defines a function signature for a pre-send hook.
	PreSendHookFn func(req *http.Request) error

	// PostSendHookFn defines a function signature for a post-send hook.
	PostSendHookFn func(req *http.Request, resp *http.Response) error

	// BuildPayloadFn defines a function signature for a function that builds a payload.
	BuildPayloadFn func(subject, message string) (payload any)

	// Serializer is used to serialize the payload to a byte slice.
	Serializer interface {
		Marshal(contentType string, payload any) (payloadRaw []byte, err error)
	}

	// Webhook represents a single webhook receiver. It contains all the information needed to send a valid request to
	// the receiver. The BuildPayload function is used to build the payload that will be sent to the receiver from the
	// given subject and message.
	Webhook struct {
		ContentType  string
		Header       http.Header
		Method       string
		URL          string
		BuildPayload BuildPayloadFn
	}

	// Service is the main struct of this package. It contains all the information needed to send notifications to a
	// list of receivers. The receivers are represented by Webhooks and are expected to be valid HTTP endpoints. The
	// Service also allows.
	Service struct {
		client        *http.Client
		webhooks      []*Webhook
		preSendHooks  []PreSendHookFn
		postSendHooks []PostSendHookFn
		Serializer    Serializer
	}
)

const (
	defaultUserAgent     = "notify/" + notify.Version
	defaultContentType   = "application/json; charset=utf-8"
	defaultRequestMethod = http.MethodPost

	// Defining these as constants for testing purposes.
	defaultSubjectKey = "subject"
	defaultMessageKey = "message"
)

type defaultMarshaller struct{}

// Marshal takes a payload and serializes it to a byte slice. The content type is used to determine the serialization
// format. If the content type is not supported, an error is returned. The default marshaller supports the following
// content types: application/json, text/plain.
// NOTE: should we expand the default marshaller to support more content types?
func (defaultMarshaller) Marshal(contentType string, payload any) ([]byte, error) {
	var out []byte
	var err error

	switch {
	case strings.HasPrefix(contentType, "application/json"):
		out, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
	case strings.HasPrefix(contentType, "text/plain"):
		str, ok := payload.(string)
		if !ok {
			return nil, fmt.Errorf("payload was expected to be of type string, got %T", payload)
		}
		out = []byte(str)
	default:
		return nil, errors.New("unsupported content type")
	}

	return out, nil
}

// buildDefaultPayload is the default payload builder. It builds a payload that is a map with the keys "subject" and
// "message".
func buildDefaultPayload(subject, message string) any {
	return map[string]string{
		defaultSubjectKey: subject,
		defaultMessageKey: message,
	}
}

// New returns a new instance of a Service notification service. Parameter 'tag' is used as a log prefix and may be left
// empty, it has a fallback value.
func New() *Service {
	return &Service{
		client:        http.DefaultClient,
		webhooks:      []*Webhook{},
		preSendHooks:  []PreSendHookFn{},
		postSendHooks: []PostSendHookFn{},
		Serializer:    defaultMarshaller{},
	}
}

func newWebhook(url string) *Webhook {
	return &Webhook{
		ContentType:  defaultContentType,
		Header:       http.Header{},
		Method:       defaultRequestMethod,
		URL:          url,
		BuildPayload: buildDefaultPayload,
	}
}

// String returns a string representation of the webhook. It implements the fmt.Stringer interface.
func (w *Webhook) String() string {
	if w == nil {
		return ""
	}

	return strings.TrimSpace(fmt.Sprintf("%s %s %s", strings.ToUpper(w.Method), w.URL, w.ContentType))
}

// AddReceivers accepts a list of Webhooks and adds them as receivers. The Webhooks are expected to be valid HTTP
// endpoints.
func (s *Service) AddReceivers(webhooks ...*Webhook) {
	s.webhooks = append(s.webhooks, webhooks...)
}

// AddReceiversURLs accepts a list of URLs and adds them as receivers. Internally it converts the URLs to Webhooks by
// using the default content-type ("application/json") and request method ("POST").
func (s *Service) AddReceiversURLs(urls ...string) {
	for _, url := range urls {
		s.AddReceivers(newWebhook(url))
	}
}

// WithClient sets the http client to be used for sending requests. Calling this method is optional, the default client
// will be used if this method is not called.
func (s *Service) WithClient(client *http.Client) {
	if client != nil {
		s.client = client
	}
}

// doPreSendHooks executes all the pre-send hooks. If any of the hooks returns an error, the execution is stopped and
// the error is returned.
func (s *Service) doPreSendHooks(req *http.Request) error {
	for _, hook := range s.preSendHooks {
		if err := hook(req); err != nil {
			return err
		}
	}

	return nil
}

// doPostSendHooks executes all the post-send hooks. If any of the hooks returns an error, the execution is stopped and
// the error is returned.
func (s *Service) doPostSendHooks(req *http.Request, resp *http.Response) error {
	for _, hook := range s.postSendHooks {
		if err := hook(req, resp); err != nil {
			return err
		}
	}

	return nil
}

// PreSend adds a pre-send hook to the service. The hook will be executed before sending a request to a receiver.
func (s *Service) PreSend(hook PreSendHookFn) {
	s.preSendHooks = append(s.preSendHooks, hook)
}

// PostSend adds a post-send hook to the service. The hook will be executed after sending a request to a receiver.
func (s *Service) PostSend(hook PostSendHookFn) {
	s.postSendHooks = append(s.postSendHooks, hook)
}

// newRequest creates a new http request with the given method, content-type, url and payload. Request created by this
// function will usually be passed to the Service.do method.
func newRequest(ctx context.Context, hook *Webhook, payload io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, hook.Method, hook.URL, payload)
	if err != nil {
		return nil, err
	}

	req.Header = hook.Header

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", defaultUserAgent)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", hook.ContentType)
	}

	return req, nil
}

// do sends the given request and returns an error if the request failed. A failed request gets identified by either
// an unsuccessful status code or a non-nil error. The given request is expected to be valid and was usually created
// by the newRequest function.
func (s *Service) do(req *http.Request) error {
	// Execute all pre-send hooks in order.
	if err := s.doPreSendHooks(req); err != nil {
		return fmt.Errorf("pre-send hooks: %w", err)
	}

	// Actually send the HTTP request.
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Execute all post-send hooks in order.
	if err = s.doPostSendHooks(req, resp); err != nil {
		return fmt.Errorf("post-send hooks: %w", err)
	}

	// Check if response code is 2xx. Should this be configurable?
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("responded with status code: %d", resp.StatusCode)
	}

	return nil
}

// send is a helper method that sends a message to a single webhook. It wraps the core logic of the Send method, which
// is creating a new request for the given webhook and sending it.
func (s *Service) send(ctx context.Context, webhook *Webhook, payload []byte) error {
	// Create a new HTTP request for the given webhook.
	req, err := newRequest(ctx, webhook, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	defer func() { _ = req.Body.Close() }()

	return s.do(req)
}

// Send takes a message and sends it to all webhooks.
func (s *Service) Send(ctx context.Context, subject, message string) error {
	// Send message to all webhooks.
	for _, webhook := range s.webhooks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Skip webhook if it is nil.
			if webhook == nil {
				continue
			}

			// Build the payload for the current webhook.
			payload := webhook.BuildPayload(subject, message)

			// Marshal the message into a payload.
			payloadRaw, err := s.Serializer.Marshal(webhook.ContentType, payload)
			if err != nil {
				return fmt.Errorf("marshal payload: %w", err)
			}

			// Send the payload to the webhook.
			if err = s.send(ctx, webhook, payloadRaw); err != nil {
				return fmt.Errorf("send to %s: %w", webhook.URL, err)
			}
		}
	}

	return nil
}
