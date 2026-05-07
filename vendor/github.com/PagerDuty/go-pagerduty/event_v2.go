package pagerduty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// V2Event includes the incident/alert details
type V2Event struct {
	RoutingKey string        `json:"routing_key"`
	Action     string        `json:"event_action"`
	DedupKey   string        `json:"dedup_key,omitempty"`
	Images     []interface{} `json:"images,omitempty"`
	Links      []interface{} `json:"links,omitempty"`
	Client     string        `json:"client,omitempty"`
	ClientURL  string        `json:"client_url,omitempty"`
	Payload    *V2Payload    `json:"payload,omitempty"`
}

// V2Payload represents the individual event details for an event
type V2Payload struct {
	Summary   string      `json:"summary"`
	Source    string      `json:"source"`
	Severity  string      `json:"severity"`
	Timestamp string      `json:"timestamp,omitempty"`
	Component string      `json:"component,omitempty"`
	Group     string      `json:"group,omitempty"`
	Class     string      `json:"class,omitempty"`
	Details   interface{} `json:"custom_details,omitempty"`
}

// V2EventResponse is the json response body for an event
type V2EventResponse struct {
	Status   string   `json:"status,omitempty"`
	DedupKey string   `json:"dedup_key,omitempty"`
	Message  string   `json:"message,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

const v2eventEndPoint = "https://events.pagerduty.com/v2/enqueue"

// ManageEvent handles the trigger, acknowledge, and resolve methods for an
// event.
//
// Deprecated: Use ManageEventWithContext instead.
func ManageEvent(e V2Event) (*V2EventResponse, error) {
	return ManageEventWithContext(context.Background(), e)
}

// EventsAPIV2Error represents the error response received when an Events API V2 call fails. The
// HTTP response code is set inside of the StatusCode field, with the EventsAPIV2Error
// field being the wrapper around the JSON error object returned from the Events API V2.
//
// This type also provides some helper methods like .BadRequest(), .RateLimited(),
// and .Temporary() to help callers reason about how to handle the error.
type EventsAPIV2Error struct {
	// StatusCode is the HTTP response status code.
	StatusCode int `json:"-"`

	// APIError represents the object returned by the API when an error occurs,
	// which includes messages that should hopefully provide useful context
	// to the end user.
	//
	// If the API response did not contain an error object, the .Valid field of
	// APIError will be false. If .Valid is true, the .ErrorObject field is
	// valid and should be consulted.
	APIError NullEventsAPIV2ErrorObject

	message string
}

var _ json.Unmarshaler = (*EventsAPIV2Error)(nil) // assert that it satisfies the json.Unmarshaler interface.

// Error satisfies the error interface, and should contain the StatusCode,
// APIError.Message, APIError.ErrorObject.Status, and APIError.Errors.
func (e EventsAPIV2Error) Error() string {
	if len(e.message) > 0 {
		return e.message
	}

	if !e.APIError.Valid {
		return fmt.Sprintf("HTTP response failed with status code %d and no JSON error object was present", e.StatusCode)
	}

	if len(e.APIError.ErrorObject.Errors) == 0 {
		return fmt.Sprintf(
			"HTTP response failed with status code %d, status: %s, message: %s",
			e.StatusCode, e.APIError.ErrorObject.Status, e.APIError.ErrorObject.Message,
		)
	}

	return fmt.Sprintf(
		"HTTP response failed with status code %d, status: %s, message: %s: %s",
		e.StatusCode,
		e.APIError.ErrorObject.Status,
		e.APIError.ErrorObject.Message,
		apiErrorsDetailString(e.APIError.ErrorObject.Errors),
	)
}

// UnmarshalJSON satisfies encoding/json.Unmarshaler.
func (e *EventsAPIV2Error) UnmarshalJSON(data []byte) error {
	var eo EventsAPIV2ErrorObject
	if err := json.Unmarshal(data, &eo); err != nil {
		return err
	}

	e.APIError.ErrorObject = eo
	e.APIError.Valid = true

	return nil
}

// BadRequest returns whether the event request was rejected by PagerDuty as an
// incorrect or invalid event structure.
func (e EventsAPIV2Error) BadRequest() bool {
	return e.StatusCode == http.StatusBadRequest
}

// RateLimited returns whether the response had a status of 429, and as such the
// client is rate limited. The PagerDuty rate limits should reset once per
// minute, and for the REST API they are an account-wide rate limit (not per
// API key or IP).
func (e EventsAPIV2Error) RateLimited() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

// APITimeout returns whether whether the response had a status of 408,
// indicating there was a request timeout on PagerDuty's side. This error is
// considered temporary, and so the request should be retried.
//
// Please note, this does not returnn true if the Go context.Context deadline
// was exceeded when making the request.
func (e EventsAPIV2Error) APITimeout() bool {
	return e.StatusCode == http.StatusRequestTimeout
}

// Temporary returns whether it was a temporary error, one of which is a
// RateLimited error.
func (e EventsAPIV2Error) Temporary() bool {
	return e.RateLimited() || e.APITimeout() || (e.StatusCode >= 500 && e.StatusCode < 600)
}

// NullEventsAPIV2ErrorObject is a wrapper around the EventsAPIV2ErrorObject type. If the Valid
// field is true, the API response included a structured error JSON object. This
// structured object is then set on the ErrorObject field.
//
// We assume it's possible in exceptional failure modes for error objects to be omitted by PagerDuty.
// As such, this wrapper type provides us a way to check if the object was
// provided while avoiding consumers accidentally missing a nil pointer check,
// thus crashing their whole program.
type NullEventsAPIV2ErrorObject struct {
	Valid       bool
	ErrorObject EventsAPIV2ErrorObject
}

// EventsAPIV2ErrorObject represents the object returned by the Events V2 API when an error
// occurs. This includes messages that should hopefully provide useful context
// to the end user.
type EventsAPIV2ErrorObject struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`

	// Errors is likely to be empty, with the relevant error presented via the
	// Status field instead.
	Errors []string `json:"errors,omitempty"`
}

// ManageEventWithContext handles the trigger, acknowledge, and resolve methods for an event.
func ManageEventWithContext(ctx context.Context, e V2Event) (*V2EventResponse, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v2eventEndPoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("User-Agent", "go-pagerduty/"+Version)
	req.Header.Set("Content-Type", "application/json")

	// TODO(theckman): switch to a package-local default client
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }() // explicitly discard error
	if resp.StatusCode != http.StatusAccepted {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, EventsAPIV2Error{
				StatusCode: resp.StatusCode,
				message:    fmt.Sprintf("HTTP response with status code: %d: error: %s", resp.StatusCode, err),
			}
		}
		// now try to decode the response body into the error object.
		var eae EventsAPIV2Error
		err = json.Unmarshal(b, &eae)
		if err != nil {
			eae = EventsAPIV2Error{
				StatusCode: resp.StatusCode,
				message:    fmt.Sprintf("HTTP response with status code: %d, JSON unmarshal object body failed: %s, body: %s", resp.StatusCode, err, string(b)),
			}
			return nil, eae
		}

		eae.StatusCode = resp.StatusCode
		return nil, eae
	}

	var eventResponse V2EventResponse
	if err := json.NewDecoder(resp.Body).Decode(&eventResponse); err != nil {
		return nil, err
	}
	return &eventResponse, nil
}

// ManageEvent handles the trigger, acknowledge, and resolve methods for an
// event.
//
// Deprecated: Use ManageEventWithContext instead.
func (c *Client) ManageEvent(e *V2Event) (*V2EventResponse, error) {
	return c.ManageEventWithContext(context.Background(), e)
}

// ManageEventWithContext handles the trigger, acknowledge, and resolve methods for an event.
func (c *Client) ManageEventWithContext(ctx context.Context, e *V2Event) (*V2EventResponse, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	resp, err := c.doWithEndpoint(ctx, c.v2EventsAPIEndpoint, http.MethodPost, "/v2/enqueue", false, bytes.NewBuffer(data), nil)
	if err != nil {
		return nil, err
	}

	var result V2EventResponse
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, err
}
