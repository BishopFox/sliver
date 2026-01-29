package events

import (
	"fmt"
	"reflect"
	"time"

	jsoniter "github.com/json-iterator/go"
)

// Event - all events returned by the EventIterator conform to this interface
type Event interface {
	GetName() string
	SetName(name string)
	GetTimestamp() time.Time
	SetTimestamp(time.Time)
	GetID() string
	SetID(id string)
}

// EventNames - a list of all JSON event types returned by the /events API
var EventNames = map[string]func() Event{
	"accepted":                 new_(Accepted{}),
	"clicked":                  new_(Clicked{}),
	"complained":               new_(Complained{}),
	"delivered":                new_(Delivered{}),
	"failed":                   new_(Failed{}),
	"opened":                   new_(Opened{}),
	"rejected":                 new_(Rejected{}),
	"stored":                   new_(Stored{}),
	"unsubscribed":             new_(Unsubscribed{}),
	"list_member_uploaded":     new_(ListMemberUploaded{}),
	"list_member_upload_error": new_(ListMemberUploadError{}),
	"list_uploaded":            new_(ListUploaded{}),
}

// new_ is a universal event "constructor".
func new_(e any) func() Event {
	typ := reflect.TypeOf(e)
	return func() Event {
		//nolint:revive // unchecked-type-assertion: TODO: return error?
		return reflect.New(typ).Interface().(Event)
	}
}

func parseResponse(raw []byte) ([]Event, error) {
	var resp Response
	if err := jsoniter.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to un-marshall event.Response: %s", err)
	}

	var result []Event
	for _, value := range resp.Items {
		event, err := ParseEvent(value)
		if err != nil {
			return nil, fmt.Errorf("while parsing event: %s", err)
		}
		result = append(result, event)
	}
	return result, nil
}

// ParseEvents given a slice of events.RawJSON events return a slice of Event for each parsed event
func ParseEvents(raw []RawJSON) ([]Event, error) {
	var result []Event
	for _, value := range raw {
		event, err := ParseEvent(value)
		if err != nil {
			return nil, fmt.Errorf("while parsing event: %s", err)
		}
		result = append(result, event)
	}
	return result, nil
}

// ParseEvent converts raw bytes data into an event struct. Can accept events.RawJSON as input
func ParseEvent(raw []byte) (Event, error) {
	// Try to recognize the event first.
	var e EventName
	if err := jsoniter.Unmarshal(raw, &e); err != nil {
		return nil, fmt.Errorf("failed to recognize event: %v", err)
	}

	// Get the event "constructor" from the map.
	newEvent, ok := EventNames[e.GetName()]
	if !ok {
		return nil, fmt.Errorf("unsupported event: '%s'", e.GetName())
	}
	event := newEvent()

	// Parse the known event.
	if err := jsoniter.Unmarshal(raw, event); err != nil {
		return nil, fmt.Errorf("failed to parse event '%s': %v", e.GetName(), err)
	}

	return event, nil
}
