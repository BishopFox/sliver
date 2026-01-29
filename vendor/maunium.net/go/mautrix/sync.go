// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package mautrix

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

// EventHandler handles a single event from a sync response.
type EventHandler func(ctx context.Context, evt *event.Event)

// SyncHandler handles a whole sync response. If the return value is false, handling will be stopped completely.
type SyncHandler func(ctx context.Context, resp *RespSync, since string) bool

// Syncer is an interface that must be satisfied in order to do /sync requests on a client.
type Syncer interface {
	// ProcessResponse processes the /sync response. The since parameter is the since= value that was used to produce the response.
	// This is useful for detecting the very first sync (since=""). If an error is return, Syncing will be stopped permanently.
	ProcessResponse(ctx context.Context, resp *RespSync, since string) error
	// OnFailedSync returns either the time to wait before retrying or an error to stop syncing permanently.
	OnFailedSync(res *RespSync, err error) (time.Duration, error)
	// GetFilterJSON for the given user ID. NOT the filter ID.
	GetFilterJSON(userID id.UserID) *Filter
}

type ExtensibleSyncer interface {
	OnSync(callback SyncHandler)
	OnEvent(callback EventHandler)
	OnEventType(eventType event.Type, callback EventHandler)
}

type DispatchableSyncer interface {
	Dispatch(ctx context.Context, evt *event.Event)
}

// DefaultSyncer is the default syncing implementation. You can either write your own syncer, or selectively
// replace parts of this default syncer (e.g. the ProcessResponse method). The default syncer uses the observer
// pattern to notify callers about incoming events. See DefaultSyncer.OnEventType for more information.
type DefaultSyncer struct {
	// syncListeners want the whole sync response, e.g. the crypto machine
	syncListeners []SyncHandler
	// globalListeners want all events
	globalListeners []EventHandler
	// listeners want a specific event type
	listeners map[event.Type][]EventHandler
	// ParseEventContent determines whether or not event content should be parsed before passing to handlers.
	ParseEventContent bool
	// ParseErrorHandler is called when event.Content.ParseRaw returns an error.
	// If it returns false, the event will not be forwarded to listeners.
	ParseErrorHandler func(evt *event.Event, err error) bool
	// FilterJSON is used when the client starts syncing and doesn't get an existing filter ID from SyncStore's LoadFilterID.
	FilterJSON *Filter
}

var _ Syncer = (*DefaultSyncer)(nil)
var _ ExtensibleSyncer = (*DefaultSyncer)(nil)

// NewDefaultSyncer returns an instantiated DefaultSyncer
func NewDefaultSyncer() *DefaultSyncer {
	return &DefaultSyncer{
		listeners:         make(map[event.Type][]EventHandler),
		syncListeners:     []SyncHandler{},
		globalListeners:   []EventHandler{},
		ParseEventContent: true,
		ParseErrorHandler: func(evt *event.Event, err error) bool {
			// By default, drop known events that can't be parsed, but let unknown events through
			return errors.Is(err, event.ErrUnsupportedContentType) ||
				// Also allow events that had their content already parsed by some other function
				errors.Is(err, event.ErrContentAlreadyParsed)
		},
	}
}

// ProcessResponse processes the /sync response in a way suitable for bots. "Suitable for bots" means a stream of
// unrepeating events. Returns a fatal error if a listener panics.
func (s *DefaultSyncer) ProcessResponse(ctx context.Context, res *RespSync, since string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("ProcessResponse panicked! since=%s panic=%s\n%s", since, r, debug.Stack())
		}
	}()
	ctx = context.WithValue(ctx, SyncTokenContextKey, since)

	for _, listener := range s.syncListeners {
		if !listener(ctx, res, since) {
			return
		}
	}

	s.processSyncEvents(ctx, "", res.ToDevice.Events, event.SourceToDevice, false)
	s.processSyncEvents(ctx, "", res.Presence.Events, event.SourcePresence, false)
	s.processSyncEvents(ctx, "", res.AccountData.Events, event.SourceAccountData, false)

	for roomID, roomData := range res.Rooms.Join {
		if roomData.StateAfter == nil {
			s.processSyncEvents(ctx, roomID, roomData.State.Events, event.SourceJoin|event.SourceState, false)
			s.processSyncEvents(ctx, roomID, roomData.Timeline.Events, event.SourceJoin|event.SourceTimeline, false)
		} else {
			s.processSyncEvents(ctx, roomID, roomData.Timeline.Events, event.SourceJoin|event.SourceTimeline, true)
			s.processSyncEvents(ctx, roomID, roomData.StateAfter.Events, event.SourceJoin|event.SourceState, false)
		}
		s.processSyncEvents(ctx, roomID, roomData.Ephemeral.Events, event.SourceJoin|event.SourceEphemeral, false)
		s.processSyncEvents(ctx, roomID, roomData.AccountData.Events, event.SourceJoin|event.SourceAccountData, false)
	}
	for roomID, roomData := range res.Rooms.Invite {
		s.processSyncEvents(ctx, roomID, roomData.State.Events, event.SourceInvite|event.SourceState, false)
	}
	for roomID, roomData := range res.Rooms.Leave {
		s.processSyncEvents(ctx, roomID, roomData.State.Events, event.SourceLeave|event.SourceState, false)
		s.processSyncEvents(ctx, roomID, roomData.Timeline.Events, event.SourceLeave|event.SourceTimeline, false)
	}
	return
}

func (s *DefaultSyncer) processSyncEvents(ctx context.Context, roomID id.RoomID, events []*event.Event, source event.Source, ignoreState bool) {
	for _, evt := range events {
		s.processSyncEvent(ctx, roomID, evt, source, ignoreState)
	}
}

func (s *DefaultSyncer) processSyncEvent(ctx context.Context, roomID id.RoomID, evt *event.Event, source event.Source, ignoreState bool) {
	evt.RoomID = roomID

	// Ensure the type class is correct. It's safe to mutate the class since the event type is not a pointer.
	// Listeners are keyed by type structs, which means only the correct class will pass.
	switch {
	case evt.StateKey != nil:
		evt.Type.Class = event.StateEventType
	case source == event.SourcePresence, source&event.SourceEphemeral != 0:
		evt.Type.Class = event.EphemeralEventType
	case source&event.SourceAccountData != 0:
		evt.Type.Class = event.AccountDataEventType
	case source == event.SourceToDevice:
		evt.Type.Class = event.ToDeviceEventType
	default:
		evt.Type.Class = event.MessageEventType
	}

	if s.ParseEventContent {
		err := evt.Content.ParseRaw(evt.Type)
		if err != nil && !s.ParseErrorHandler(evt, err) {
			return
		}
	}

	evt.Mautrix.EventSource = source
	evt.Mautrix.IgnoreState = ignoreState
	s.Dispatch(ctx, evt)
}

func (s *DefaultSyncer) Dispatch(ctx context.Context, evt *event.Event) {
	for _, fn := range s.globalListeners {
		fn(ctx, evt)
	}
	listeners, exists := s.listeners[evt.Type]
	if exists {
		for _, fn := range listeners {
			fn(ctx, evt)
		}
	}
}

// OnEventType allows callers to be notified when there are new events for the given event type.
// There are no duplicate checks.
func (s *DefaultSyncer) OnEventType(eventType event.Type, callback EventHandler) {
	_, exists := s.listeners[eventType]
	if !exists {
		s.listeners[eventType] = []EventHandler{}
	}
	s.listeners[eventType] = append(s.listeners[eventType], callback)
}

func (s *DefaultSyncer) OnSync(callback SyncHandler) {
	s.syncListeners = append(s.syncListeners, callback)
}

func (s *DefaultSyncer) OnEvent(callback EventHandler) {
	s.globalListeners = append(s.globalListeners, callback)
}

// OnFailedSync always returns a 10 second wait period between failed /syncs, never a fatal error.
func (s *DefaultSyncer) OnFailedSync(res *RespSync, err error) (time.Duration, error) {
	if errors.Is(err, MUnknownToken) {
		return 0, err
	}
	return 10 * time.Second, nil
}

var defaultFilter = Filter{
	Room: &RoomFilter{
		Timeline: &FilterPart{
			Limit: 50,
		},
	},
}

// GetFilterJSON returns a filter with a timeline limit of 50.
func (s *DefaultSyncer) GetFilterJSON(userID id.UserID) *Filter {
	if s.FilterJSON == nil {
		defaultFilterCopy := defaultFilter
		s.FilterJSON = &defaultFilterCopy
	}
	return s.FilterJSON
}

// DontProcessOldEvents is a sync handler that removes rooms that the user just joined.
// It's meant for bots to ignore events from before the bot joined the room.
//
// To use it, register it with your Syncer, e.g.:
//
//	cli.Syncer.(mautrix.ExtensibleSyncer).OnSync(cli.DontProcessOldEvents)
func (cli *Client) DontProcessOldEvents(_ context.Context, resp *RespSync, since string) bool {
	return dontProcessOldEvents(cli.UserID, resp, since)
}

var _ SyncHandler = (*Client)(nil).DontProcessOldEvents

func dontProcessOldEvents(userID id.UserID, resp *RespSync, since string) bool {
	if since == "" {
		return false
	}
	// This is a horrible hack because /sync will return the most recent messages for a room
	// as soon as you /join it. We do NOT want to process those events in that particular room
	// because they may have already been processed (if you toggle the bot in/out of the room).
	//
	// Work around this by inspecting each room's timeline and seeing if an m.room.member event for us
	// exists and is "join" and then discard processing that room entirely if so.
	// TODO: We probably want to process messages from after the last join event in the timeline.
	for roomID, roomData := range resp.Rooms.Join {
		for i := len(roomData.Timeline.Events) - 1; i >= 0; i-- {
			evt := roomData.Timeline.Events[i]
			if evt.Type == event.StateMember && evt.GetStateKey() == string(userID) {
				membership, _ := evt.Content.Raw["membership"].(string)
				if membership == "join" {
					_, ok := resp.Rooms.Join[roomID]
					if !ok {
						continue
					}
					delete(resp.Rooms.Join, roomID)   // don't re-process messages
					delete(resp.Rooms.Invite, roomID) // don't re-process invites
					break
				}
			}
		}
	}
	return true
}

// MoveInviteState is a sync handler that moves events from the state event list to the InviteRoomState in the invite event.
//
// To use it, register it with your Syncer, e.g.:
//
//	cli.Syncer.(mautrix.ExtensibleSyncer).OnSync(cli.MoveInviteState)
func (cli *Client) MoveInviteState(ctx context.Context, resp *RespSync, _ string) bool {
	for _, meta := range resp.Rooms.Invite {
		var inviteState []*event.Event
		var inviteEvt *event.Event
		for _, evt := range meta.State.Events {
			if evt.Type == event.StateMember && evt.GetStateKey() == cli.UserID.String() {
				inviteEvt = evt
			} else {
				evt.Type.Class = event.StateEventType
				_ = evt.Content.ParseRaw(evt.Type)
				inviteState = append(inviteState, evt)
			}
		}
		if inviteEvt != nil {
			inviteEvt.Unsigned.InviteRoomState = inviteState
			meta.State.Events = []*event.Event{inviteEvt}
		}
	}
	return true
}

var _ SyncHandler = (*Client)(nil).MoveInviteState
