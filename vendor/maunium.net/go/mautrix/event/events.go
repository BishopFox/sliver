// Copyright (c) 2020 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package event

import (
	"encoding/json"
	"time"

	"maunium.net/go/mautrix/id"
)

// Event represents a single Matrix event.
type Event struct {
	StateKey  *string    `json:"state_key,omitempty"`        // The state key for the event. Only present on State Events.
	Sender    id.UserID  `json:"sender,omitempty"`           // The user ID of the sender of the event
	Type      Type       `json:"type"`                       // The event type
	Timestamp int64      `json:"origin_server_ts,omitempty"` // The unix timestamp when this message was sent by the origin server
	ID        id.EventID `json:"event_id,omitempty"`         // The unique ID of this event
	RoomID    id.RoomID  `json:"room_id,omitempty"`          // The room the event was sent to. May be nil (e.g. for presence)
	Content   Content    `json:"content"`                    // The JSON content of the event.
	Redacts   id.EventID `json:"redacts,omitempty"`          // The event ID that was redacted if a m.room.redaction event
	Unsigned  Unsigned   `json:"unsigned,omitempty"`         // Unsigned content set by own homeserver.

	Mautrix MautrixInfo `json:"-"`

	ToUserID   id.UserID   `json:"to_user_id,omitempty"`   // The user ID that the to-device event was sent to. Only present in MSC2409 appservice transactions.
	ToDeviceID id.DeviceID `json:"to_device_id,omitempty"` // The device ID that the to-device event was sent to. Only present in MSC2409 appservice transactions.
}

type eventForMarshaling struct {
	StateKey  *string    `json:"state_key,omitempty"`
	Sender    id.UserID  `json:"sender,omitempty"`
	Type      Type       `json:"type"`
	Timestamp int64      `json:"origin_server_ts,omitempty"`
	ID        id.EventID `json:"event_id,omitempty"`
	RoomID    id.RoomID  `json:"room_id,omitempty"`
	Content   Content    `json:"content"`
	Redacts   id.EventID `json:"redacts,omitempty"`
	Unsigned  *Unsigned  `json:"unsigned,omitempty"`

	PrevContent   *Content    `json:"prev_content,omitempty"`
	ReplacesState *id.EventID `json:"replaces_state,omitempty"`

	ToUserID   id.UserID   `json:"to_user_id,omitempty"`
	ToDeviceID id.DeviceID `json:"to_device_id,omitempty"`
}

// UnmarshalJSON unmarshals the event, including moving prev_content from the top level to inside unsigned.
func (evt *Event) UnmarshalJSON(data []byte) error {
	var efm eventForMarshaling
	err := json.Unmarshal(data, &efm)
	if err != nil {
		return err
	}
	evt.StateKey = efm.StateKey
	evt.Sender = efm.Sender
	evt.Type = efm.Type
	evt.Timestamp = efm.Timestamp
	evt.ID = efm.ID
	evt.RoomID = efm.RoomID
	evt.Content = efm.Content
	evt.Redacts = efm.Redacts
	if efm.Unsigned != nil {
		evt.Unsigned = *efm.Unsigned
	}
	if efm.PrevContent != nil && evt.Unsigned.PrevContent == nil {
		evt.Unsigned.PrevContent = efm.PrevContent
	}
	if efm.ReplacesState != nil && *efm.ReplacesState != "" && evt.Unsigned.ReplacesState == "" {
		evt.Unsigned.ReplacesState = *efm.ReplacesState
	}
	evt.ToUserID = efm.ToUserID
	evt.ToDeviceID = efm.ToDeviceID
	return nil
}

// MarshalJSON marshals the event, including omitting the unsigned field if it's empty.
//
// This is necessary because Unsigned is not a pointer (for convenience reasons),
// and encoding/json doesn't know how to check if a non-pointer struct is empty.
//
// TODO(tulir): maybe it makes more sense to make Unsigned a pointer and make an easy and safe way to access it?
func (evt *Event) MarshalJSON() ([]byte, error) {
	unsigned := &evt.Unsigned
	if unsigned.IsEmpty() {
		unsigned = nil
	}
	return json.Marshal(&eventForMarshaling{
		StateKey:   evt.StateKey,
		Sender:     evt.Sender,
		Type:       evt.Type,
		Timestamp:  evt.Timestamp,
		ID:         evt.ID,
		RoomID:     evt.RoomID,
		Content:    evt.Content,
		Redacts:    evt.Redacts,
		Unsigned:   unsigned,
		ToUserID:   evt.ToUserID,
		ToDeviceID: evt.ToDeviceID,
	})
}

type MautrixInfo struct {
	EventSource Source

	TrustState    id.TrustState
	ForwardedKeys bool
	WasEncrypted  bool
	TrustSource   *id.Device

	ReceivedAt         time.Time
	EditedAt           time.Time
	LastEditID         id.EventID
	DecryptionDuration time.Duration

	CheckpointSent bool
	// When using MSC4222 and the state_after field, this field is set
	// for timeline events to indicate they shouldn't update room state.
	IgnoreState bool
}

func (evt *Event) GetStateKey() string {
	if evt.StateKey != nil {
		return *evt.StateKey
	}
	return ""
}

type Unsigned struct {
	PrevContent     *Content   `json:"prev_content,omitempty"`
	PrevSender      id.UserID  `json:"prev_sender,omitempty"`
	Membership      Membership `json:"membership,omitempty"`
	ReplacesState   id.EventID `json:"replaces_state,omitempty"`
	Age             int64      `json:"age,omitempty"`
	TransactionID   string     `json:"transaction_id,omitempty"`
	Relations       *Relations `json:"m.relations,omitempty"`
	RedactedBecause *Event     `json:"redacted_because,omitempty"`
	InviteRoomState []*Event   `json:"invite_room_state,omitempty"`

	BeeperHSOrder       int64               `json:"com.beeper.hs.order,omitempty"`
	BeeperHSSuborder    int16               `json:"com.beeper.hs.suborder,omitempty"`
	BeeperHSOrderString *BeeperEncodedOrder `json:"com.beeper.hs.order_string,omitempty"`
	BeeperFromBackup    bool                `json:"com.beeper.from_backup,omitempty"`

	ElementSoftFailed         bool `json:"io.element.synapse.soft_failed,omitempty"`
	ElementPolicyServerSpammy bool `json:"io.element.synapse.policy_server_spammy,omitempty"`
}

func (us *Unsigned) IsEmpty() bool {
	return us.PrevContent == nil && us.PrevSender == "" && us.ReplacesState == "" && us.Age == 0 && us.Membership == "" &&
		us.TransactionID == "" && us.RedactedBecause == nil && us.InviteRoomState == nil && us.Relations == nil &&
		us.BeeperHSOrder == 0 && us.BeeperHSSuborder == 0 && us.BeeperHSOrderString.IsZero() &&
		!us.ElementSoftFailed
}
