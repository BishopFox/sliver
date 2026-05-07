// Copyright 2017 Jan Christian Grünhage

package mautrix

import (
	"errors"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type EventFormat string

const (
	EventFormatClient     EventFormat = "client"
	EventFormatFederation EventFormat = "federation"
)

// Filter is used by clients to specify how the server should filter responses to e.g. sync requests
// Specified by: https://spec.matrix.org/v1.2/client-server-api/#filtering
type Filter struct {
	AccountData *FilterPart `json:"account_data,omitempty"`
	EventFields []string    `json:"event_fields,omitempty"`
	EventFormat EventFormat `json:"event_format,omitempty"`
	Presence    *FilterPart `json:"presence,omitempty"`
	Room        *RoomFilter `json:"room,omitempty"`

	BeeperToDevice *FilterPart `json:"com.beeper.to_device,omitempty"`
}

// RoomFilter is used to define filtering rules for room events
type RoomFilter struct {
	AccountData  *FilterPart `json:"account_data,omitempty"`
	Ephemeral    *FilterPart `json:"ephemeral,omitempty"`
	IncludeLeave bool        `json:"include_leave,omitempty"`
	NotRooms     []id.RoomID `json:"not_rooms,omitempty"`
	Rooms        []id.RoomID `json:"rooms,omitempty"`
	State        *FilterPart `json:"state,omitempty"`
	Timeline     *FilterPart `json:"timeline,omitempty"`
}

// FilterPart is used to define filtering rules for specific categories of events
type FilterPart struct {
	NotRooms                  []id.RoomID  `json:"not_rooms,omitempty"`
	Rooms                     []id.RoomID  `json:"rooms,omitempty"`
	Limit                     int          `json:"limit,omitempty"`
	NotSenders                []id.UserID  `json:"not_senders,omitempty"`
	NotTypes                  []event.Type `json:"not_types,omitempty"`
	Senders                   []id.UserID  `json:"senders,omitempty"`
	Types                     []event.Type `json:"types,omitempty"`
	ContainsURL               *bool        `json:"contains_url,omitempty"`
	LazyLoadMembers           bool         `json:"lazy_load_members,omitempty"`
	IncludeRedundantMembers   bool         `json:"include_redundant_members,omitempty"`
	UnreadThreadNotifications bool         `json:"unread_thread_notifications,omitempty"`
}

// Validate checks if the filter contains valid property values
func (filter *Filter) Validate() error {
	if filter.EventFormat != EventFormatClient && filter.EventFormat != EventFormatFederation {
		return errors.New("Bad event_format value. Must be one of [\"client\", \"federation\"]")
	}
	return nil
}

// DefaultFilter returns the default filter used by the Matrix server if no filter is provided in the request
func DefaultFilter() Filter {
	return Filter{
		AccountData: DefaultFilterPart(),
		EventFields: nil,
		EventFormat: "client",
		Presence:    DefaultFilterPart(),
		Room: &RoomFilter{
			AccountData:  DefaultFilterPart(),
			Ephemeral:    DefaultFilterPart(),
			IncludeLeave: false,
			NotRooms:     nil,
			Rooms:        nil,
			State:        DefaultFilterPart(),
			Timeline:     DefaultFilterPart(),
		},
	}
}

// DefaultFilterPart returns the default filter part used by the Matrix server if no filter is provided in the request
func DefaultFilterPart() *FilterPart {
	return &FilterPart{
		NotRooms:   nil,
		Rooms:      nil,
		Limit:      20,
		NotSenders: nil,
		NotTypes:   nil,
		Senders:    nil,
		Types:      nil,
	}
}
