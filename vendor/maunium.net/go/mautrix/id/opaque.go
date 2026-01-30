// Copyright (c) 2020 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package id

import (
	"fmt"
)

// A RoomID is a string starting with ! that references a specific room.
// https://matrix.org/docs/spec/appendices#room-ids-and-event-ids
type RoomID string

// A RoomAlias is a string starting with # that can be resolved into.
// https://matrix.org/docs/spec/appendices#room-aliases
type RoomAlias string

func NewRoomAlias(localpart, server string) RoomAlias {
	return RoomAlias(fmt.Sprintf("#%s:%s", localpart, server))
}

// An EventID is a string starting with $ that references a specific event.
//
// https://matrix.org/docs/spec/appendices#room-ids-and-event-ids
// https://matrix.org/docs/spec/rooms/v4#event-ids
type EventID string

// A BatchID is a string identifying a batch of events being backfilled to a room.
// https://github.com/matrix-org/matrix-doc/pull/2716
type BatchID string

// A DelayID is a string identifying a delayed event.
type DelayID string

func (roomID RoomID) String() string {
	return string(roomID)
}

func (roomID RoomID) URI(via ...string) *MatrixURI {
	if roomID == "" {
		return nil
	}
	return &MatrixURI{
		Sigil1: '!',
		MXID1:  string(roomID)[1:],
		Via:    via,
	}
}

func (roomID RoomID) EventURI(eventID EventID, via ...string) *MatrixURI {
	if roomID == "" {
		return nil
	} else if eventID == "" {
		return roomID.URI(via...)
	}
	return &MatrixURI{
		Sigil1: '!',
		MXID1:  string(roomID)[1:],
		Sigil2: '$',
		MXID2:  string(eventID)[1:],
		Via:    via,
	}
}

func (roomAlias RoomAlias) String() string {
	return string(roomAlias)
}

func (roomAlias RoomAlias) URI() *MatrixURI {
	if roomAlias == "" {
		return nil
	}
	return &MatrixURI{
		Sigil1: '#',
		MXID1:  string(roomAlias)[1:],
	}
}

// Deprecated: room alias event links should not be used. Use room IDs instead.
func (roomAlias RoomAlias) EventURI(eventID EventID) *MatrixURI {
	if roomAlias == "" {
		return nil
	}
	return &MatrixURI{
		Sigil1: '#',
		MXID1:  string(roomAlias)[1:],
		Sigil2: '$',
		MXID2:  string(eventID)[1:],
	}
}

func (eventID EventID) String() string {
	return string(eventID)
}

func (batchID BatchID) String() string {
	return string(batchID)
}
