// Copyright (c) 2020 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package event

import (
	"encoding/json"
	"strings"
	"time"

	"maunium.net/go/mautrix/id"
)

// TagEventContent represents the content of a m.tag room account data event.
// https://spec.matrix.org/v1.2/client-server-api/#mtag
type TagEventContent struct {
	Tags Tags `json:"tags"`
}

type Tags map[RoomTag]TagMetadata

type RoomTag string

const (
	RoomTagFavourite    RoomTag = "m.favourite"
	RoomTagLowPriority  RoomTag = "m.lowpriority"
	RoomTagServerNotice RoomTag = "m.server_notice"
)

func (rt RoomTag) IsUserDefined() bool {
	return strings.HasPrefix(string(rt), "u.")
}

func (rt RoomTag) String() string {
	return string(rt)
}

func (rt RoomTag) Name() string {
	if rt.IsUserDefined() {
		return string(rt[2:])
	}
	switch rt {
	case RoomTagFavourite:
		return "Favourite"
	case RoomTagLowPriority:
		return "Low priority"
	case RoomTagServerNotice:
		return "Server notice"
	default:
		return ""
	}
}

// Deprecated: type alias
type Tag = TagMetadata

type TagMetadata struct {
	Order json.Number `json:"order,omitempty"`

	MauDoublePuppetSource string `json:"fi.mau.double_puppet_source,omitempty"`
}

// DirectChatsEventContent represents the content of a m.direct account data event.
// https://spec.matrix.org/v1.2/client-server-api/#mdirect
type DirectChatsEventContent map[id.UserID][]id.RoomID

// FullyReadEventContent represents the content of a m.fully_read account data event.
// https://spec.matrix.org/v1.2/client-server-api/#mfully_read
type FullyReadEventContent struct {
	EventID id.EventID `json:"event_id"`
}

// IgnoredUserListEventContent represents the content of a m.ignored_user_list account data event.
// https://spec.matrix.org/v1.2/client-server-api/#mignored_user_list
type IgnoredUserListEventContent struct {
	IgnoredUsers map[id.UserID]IgnoredUser `json:"ignored_users"`
}

type IgnoredUser struct {
	// This is an empty object
}

type MarkedUnreadEventContent struct {
	Unread bool `json:"unread"`
}

type BeeperMuteEventContent struct {
	MutedUntil int64 `json:"muted_until,omitempty"`
}

func (bmec *BeeperMuteEventContent) IsMuted() bool {
	return bmec.MutedUntil < 0 || (bmec.MutedUntil > 0 && bmec.GetMutedUntilTime().After(time.Now()))
}

var MutedForever = time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)

func (bmec *BeeperMuteEventContent) GetMutedUntilTime() time.Time {
	if bmec.MutedUntil < 0 {
		return MutedForever
	} else if bmec.MutedUntil > 0 {
		return time.UnixMilli(bmec.MutedUntil)
	}
	return time.Time{}
}

func (bmec *BeeperMuteEventContent) GetMuteDuration() time.Duration {
	ts := bmec.GetMutedUntilTime()
	now := time.Now()
	if ts.Before(now) {
		return 0
	} else if ts == MutedForever {
		return -1
	} else {
		return ts.Sub(now)
	}
}
