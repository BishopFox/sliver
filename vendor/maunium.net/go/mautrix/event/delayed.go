package event

import (
	"encoding/json"

	"go.mau.fi/util/jsontime"

	"maunium.net/go/mautrix/id"
)

type ScheduledDelayedEvent struct {
	DelayID      id.DelayID         `json:"delay_id"`
	RoomID       id.RoomID          `json:"room_id"`
	Type         Type               `json:"type"`
	StateKey     *string            `json:"state_key,omitempty"`
	Delay        int64              `json:"delay"`
	RunningSince jsontime.UnixMilli `json:"running_since"`
	Content      Content            `json:"content"`
}

func (e ScheduledDelayedEvent) AsEvent(eventID id.EventID, ts jsontime.UnixMilli) (*Event, error) {
	evt := &Event{
		ID:        eventID,
		RoomID:    e.RoomID,
		Type:      e.Type,
		StateKey:  e.StateKey,
		Content:   e.Content,
		Timestamp: ts.UnixMilli(),
	}
	return evt, evt.Content.ParseRaw(evt.Type)
}

type FinalisedDelayedEvent struct {
	DelayedEvent *ScheduledDelayedEvent `json:"scheduled_event"`
	Outcome      DelayOutcome           `json:"outcome"`
	Reason       DelayReason            `json:"reason"`
	Error        json.RawMessage        `json:"error,omitempty"`
	EventID      id.EventID             `json:"event_id,omitempty"`
	Timestamp    jsontime.UnixMilli     `json:"origin_server_ts"`
}

type DelayStatus string

var (
	DelayStatusScheduled DelayStatus = "scheduled"
	DelayStatusFinalised DelayStatus = "finalised"
)

type DelayAction string

var (
	DelayActionSend    DelayAction = "send"
	DelayActionCancel  DelayAction = "cancel"
	DelayActionRestart DelayAction = "restart"
)

type DelayOutcome string

var (
	DelayOutcomeSend   DelayOutcome = "send"
	DelayOutcomeCancel DelayOutcome = "cancel"
)

type DelayReason string

var (
	DelayReasonAction DelayReason = "action"
	DelayReasonError  DelayReason = "error"
	DelayReasonDelay  DelayReason = "delay"
)
