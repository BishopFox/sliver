//nolint:gochecknoglobals // I agree with the linter, won't bother fixing this now, will be fixed in v2.
package webpush

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/SherClockHolmes/webpush-go"
)

type (
	// Urgency indicates the importance of the message. It's a type alias for webpush.Urgency.
	Urgency = webpush.Urgency

	// Options are optional settings for the sending of a message. It's a type alias for webpush.Options.
	Options = webpush.Options

	// Subscription is a JSON representation of a webpush subscription. It's a type alias for webpush.Subscription.
	Subscription = webpush.Subscription

	// messagePayload is the JSON payload that is sent to the webpush endpoint.
	messagePayload struct {
		Subject string         `json:"subject"`
		Message string         `json:"message"`
		Data    map[string]any `json:"data,omitempty"`
	}

	msgDataKey    struct{}
	msgOptionsKey struct{}
)

// optionsKey is used as a context.Context key to optionally add options to the messagePayload payload.
var optionsKey = msgOptionsKey{}

// dataKey is used as a context.Context key to optionally add data to the messagePayload payload.
var dataKey = msgDataKey{}

// These are exposed Urgency constants from the webpush package.
var (
	// UrgencyVeryLow requires device state: on power and Wi-Fi.
	UrgencyVeryLow Urgency = webpush.UrgencyVeryLow

	// UrgencyLow requires device state: on either power or Wi-Fi.
	UrgencyLow Urgency = webpush.UrgencyLow

	// UrgencyNormal excludes device state: low battery.
	UrgencyNormal Urgency = webpush.UrgencyNormal

	// UrgencyHigh admits device state: low battery.
	UrgencyHigh Urgency = webpush.UrgencyHigh
)

// Service encapsulates the webpush notification system along with the internal state.
type Service struct {
	subscriptions []webpush.Subscription
	options       webpush.Options
}

// New returns a new instance of the Service.
func New(vapidPublicKey string, vapidPrivateKey string) *Service {
	return &Service{
		subscriptions: []webpush.Subscription{},
		options: webpush.Options{
			VAPIDPublicKey:  vapidPublicKey,
			VAPIDPrivateKey: vapidPrivateKey,
		},
	}
}

// AddReceivers adds one or more subscriptions to the Service.
func (s *Service) AddReceivers(subscriptions ...Subscription) {
	s.subscriptions = append(s.subscriptions, subscriptions...)
}

// withOptions returns a new Options struct with the incoming options merged with the Service's options. The incoming
// options take precedence, except for the VAPID keys. Existing VAPID keys are only replaced if the incoming VAPID keys
// are not empty.
func (s *Service) withOptions(options Options) Options {
	if options.VAPIDPublicKey == "" {
		options.VAPIDPublicKey = s.options.VAPIDPublicKey
	}
	if options.VAPIDPrivateKey == "" {
		options.VAPIDPrivateKey = s.options.VAPIDPrivateKey
	}

	return options
}

// WithOptions binds the options to the context so that they will be used by the Service.Send method automatically.
// Options
// are settings that allow you to customize the sending behavior of a message.
func WithOptions(ctx context.Context, options Options) context.Context {
	return context.WithValue(ctx, optionsKey, options)
}

func optionsFromContext(ctx context.Context) Options {
	if options, ok := ctx.Value(optionsKey).(Options); ok {
		return options
	}

	return Options{}
}

// WithData binds the data to the context so that it will be used by the Service.Send method automatically. Data is a
// map[string]any and acts as a metadata field that is sent along with the message payload.
func WithData(ctx context.Context, data map[string]any) context.Context {
	return context.WithValue(ctx, dataKey, data)
}

func dataFromContext(ctx context.Context) map[string]any {
	if data, ok := ctx.Value(dataKey).(map[string]any); ok {
		return data
	}

	return map[string]any{}
}

// payloadFromContext returns a json encoded byte array of the messagePayload payload that is ready to be sent to the
// webpush endpoint. Internally, it uses the messagePayload and data from the context, and it combines it with the
// subject and message arguments into a single messagePayload.
func payloadFromContext(ctx context.Context, subject, message string) ([]byte, error) {
	payload := messagePayload{
		Subject: subject,
		Message: message,
	}

	payload.Data = dataFromContext(ctx) // Load optional data

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal messagePayload: %w", err)
	}

	return payloadBytes, nil
}

// send is a wrapper that makes it primarily easier to defer the closing of the response body.
func (s *Service) send(ctx context.Context, message []byte, subscription *Subscription, options *Options) error {
	res, err := webpush.SendNotificationWithContext(ctx, message, subscription, options)
	if err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated {
		return nil // Everything is fine
	}

	// Make sure to produce a helpful error message

	baseErr := fmt.Errorf(
		"send message to webpush subscription %s: unexpected status code %d",
		subscription.Endpoint, res.StatusCode,
	)

	if _, err = io.ReadAll(res.Body); err != nil {
		err = fmt.Errorf("read response body: %w", err)
	}

	err = errors.Join(baseErr, err)

	return err
}

// Send sends a message to all the webpush subscriptions that have been added to the Service. The subject and message
// arguments are the subject and message of the messagePayload payload. The context can be used to optionally add
// options and data to the messagePayload payload. See the WithOptions and WithData functions.
func (s *Service) Send(ctx context.Context, subject, message string) error {
	// Get the options from the context and merge them with the service's initial options
	options := optionsFromContext(ctx)
	options = s.withOptions(options)

	payload, err := payloadFromContext(ctx, subject, message)
	if err != nil {
		return err
	}

	for _, subscription := range s.subscriptions {
		if err = s.send(ctx, payload, &subscription, &options); err != nil {
			return err
		}
	}

	return nil
}
