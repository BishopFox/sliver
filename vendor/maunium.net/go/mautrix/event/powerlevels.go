// Copyright (c) 2020 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package event

import (
	"math"
	"slices"
	"sync"

	"go.mau.fi/util/ptr"
	"golang.org/x/exp/maps"

	"maunium.net/go/mautrix/id"
)

// PowerLevelsEventContent represents the content of a m.room.power_levels state event content.
// https://spec.matrix.org/v1.5/client-server-api/#mroompower_levels
type PowerLevelsEventContent struct {
	usersLock    sync.RWMutex
	Users        map[id.UserID]int `json:"users,omitempty"`
	UsersDefault int               `json:"users_default,omitempty"`

	eventsLock    sync.RWMutex
	Events        map[string]int `json:"events,omitempty"`
	EventsDefault int            `json:"events_default,omitempty"`

	Notifications *NotificationPowerLevels `json:"notifications,omitempty"`

	StateDefaultPtr *int `json:"state_default,omitempty"`

	InvitePtr *int `json:"invite,omitempty"`
	KickPtr   *int `json:"kick,omitempty"`
	BanPtr    *int `json:"ban,omitempty"`
	RedactPtr *int `json:"redact,omitempty"`

	// This is not a part of power levels, it's added by mautrix-go internally in certain places
	// in order to detect creator power accurately.
	CreateEvent *Event `json:"-"`
}

func (pl *PowerLevelsEventContent) Clone() *PowerLevelsEventContent {
	if pl == nil {
		return nil
	}
	return &PowerLevelsEventContent{
		Users:           maps.Clone(pl.Users),
		UsersDefault:    pl.UsersDefault,
		Events:          maps.Clone(pl.Events),
		EventsDefault:   pl.EventsDefault,
		StateDefaultPtr: ptr.Clone(pl.StateDefaultPtr),

		Notifications: pl.Notifications.Clone(),

		InvitePtr: ptr.Clone(pl.InvitePtr),
		KickPtr:   ptr.Clone(pl.KickPtr),
		BanPtr:    ptr.Clone(pl.BanPtr),
		RedactPtr: ptr.Clone(pl.RedactPtr),

		CreateEvent: pl.CreateEvent,
	}
}

type NotificationPowerLevels struct {
	RoomPtr *int `json:"room,omitempty"`
}

func (npl *NotificationPowerLevels) Clone() *NotificationPowerLevels {
	if npl == nil {
		return nil
	}
	return &NotificationPowerLevels{
		RoomPtr: ptr.Clone(npl.RoomPtr),
	}
}

func (npl *NotificationPowerLevels) Room() int {
	if npl != nil && npl.RoomPtr != nil {
		return *npl.RoomPtr
	}
	return 50
}

func (pl *PowerLevelsEventContent) Invite() int {
	if pl.InvitePtr != nil {
		return *pl.InvitePtr
	}
	return 0
}

func (pl *PowerLevelsEventContent) Kick() int {
	if pl.KickPtr != nil {
		return *pl.KickPtr
	}
	return 50
}

func (pl *PowerLevelsEventContent) Ban() int {
	if pl.BanPtr != nil {
		return *pl.BanPtr
	}
	return 50
}

func (pl *PowerLevelsEventContent) Redact() int {
	if pl.RedactPtr != nil {
		return *pl.RedactPtr
	}
	return 50
}

func (pl *PowerLevelsEventContent) StateDefault() int {
	if pl.StateDefaultPtr != nil {
		return *pl.StateDefaultPtr
	}
	return 50
}

func (pl *PowerLevelsEventContent) GetUserLevel(userID id.UserID) int {
	if pl.isCreator(userID) {
		return math.MaxInt
	}
	pl.usersLock.RLock()
	defer pl.usersLock.RUnlock()
	level, ok := pl.Users[userID]
	if !ok {
		return pl.UsersDefault
	}
	return level
}

const maxPL = 1<<53 - 1

func (pl *PowerLevelsEventContent) SetUserLevel(userID id.UserID, level int) {
	pl.usersLock.Lock()
	defer pl.usersLock.Unlock()
	if pl.isCreator(userID) {
		return
	}
	if level == math.MaxInt && maxPL < math.MaxInt {
		// Hack to avoid breaking on 32-bit systems (they're only slightly supported)
		x := int64(maxPL)
		level = int(x)
	}
	if level == pl.UsersDefault {
		delete(pl.Users, userID)
	} else {
		if pl.Users == nil {
			pl.Users = make(map[id.UserID]int)
		}
		pl.Users[userID] = level
	}
}

func (pl *PowerLevelsEventContent) EnsureUserLevel(target id.UserID, level int) bool {
	return pl.EnsureUserLevelAs("", target, level)
}

func (pl *PowerLevelsEventContent) createContent() *CreateEventContent {
	if pl.CreateEvent == nil {
		return &CreateEventContent{}
	}
	return pl.CreateEvent.Content.AsCreate()
}

func (pl *PowerLevelsEventContent) isCreator(userID id.UserID) bool {
	cc := pl.createContent()
	return cc.SupportsCreatorPower() && (userID == pl.CreateEvent.Sender || slices.Contains(cc.AdditionalCreators, userID))
}

func (pl *PowerLevelsEventContent) EnsureUserLevelAs(actor, target id.UserID, level int) bool {
	if pl.isCreator(target) {
		return false
	}
	existingLevel := pl.GetUserLevel(target)
	if actor != "" && !pl.isCreator(actor) {
		actorLevel := pl.GetUserLevel(actor)
		if actorLevel <= existingLevel || actorLevel < level {
			return false
		}
	}
	if existingLevel != level {
		pl.SetUserLevel(target, level)
		return true
	}
	return false
}

func (pl *PowerLevelsEventContent) GetEventLevel(eventType Type) int {
	pl.eventsLock.RLock()
	defer pl.eventsLock.RUnlock()
	level, ok := pl.Events[eventType.String()]
	if !ok {
		if eventType.IsState() {
			return pl.StateDefault()
		}
		return pl.EventsDefault
	}
	return level
}

func (pl *PowerLevelsEventContent) SetEventLevel(eventType Type, level int) {
	pl.eventsLock.Lock()
	defer pl.eventsLock.Unlock()
	if (eventType.IsState() && level == pl.StateDefault()) || (!eventType.IsState() && level == pl.EventsDefault) {
		delete(pl.Events, eventType.String())
	} else {
		if pl.Events == nil {
			pl.Events = make(map[string]int)
		}
		pl.Events[eventType.String()] = level
	}
}

func (pl *PowerLevelsEventContent) EnsureEventLevel(eventType Type, level int) bool {
	return pl.EnsureEventLevelAs("", eventType, level)
}

func (pl *PowerLevelsEventContent) EnsureEventLevelAs(actor id.UserID, eventType Type, level int) bool {
	existingLevel := pl.GetEventLevel(eventType)
	if actor != "" && !pl.isCreator(actor) {
		actorLevel := pl.GetUserLevel(actor)
		if existingLevel > actorLevel || level > actorLevel {
			return false
		}
	}
	if existingLevel != level {
		pl.SetEventLevel(eventType, level)
		return true
	}
	return false
}
