// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package id

import (
	"errors"
	"fmt"
	"slices"
)

type RoomVersion string

const (
	RoomV0  RoomVersion = "" // No room version, used for rooms created before room versions were introduced, equivalent to v1
	RoomV1  RoomVersion = "1"
	RoomV2  RoomVersion = "2"
	RoomV3  RoomVersion = "3"
	RoomV4  RoomVersion = "4"
	RoomV5  RoomVersion = "5"
	RoomV6  RoomVersion = "6"
	RoomV7  RoomVersion = "7"
	RoomV8  RoomVersion = "8"
	RoomV9  RoomVersion = "9"
	RoomV10 RoomVersion = "10"
	RoomV11 RoomVersion = "11"
	RoomV12 RoomVersion = "12"
)

func (rv RoomVersion) Equals(versions ...RoomVersion) bool {
	return slices.Contains(versions, rv)
}

func (rv RoomVersion) NotEquals(versions ...RoomVersion) bool {
	return !rv.Equals(versions...)
}

var ErrUnknownRoomVersion = errors.New("unknown room version")

func (rv RoomVersion) unknownVersionError() error {
	return fmt.Errorf("%w %s", ErrUnknownRoomVersion, rv)
}

func (rv RoomVersion) IsKnown() bool {
	switch rv {
	case RoomV0, RoomV1, RoomV2, RoomV3, RoomV4, RoomV5, RoomV6, RoomV7, RoomV8, RoomV9, RoomV10, RoomV11, RoomV12:
		return true
	default:
		return false
	}
}

type StateResVersion int

const (
	// StateResV1 is the original state resolution algorithm.
	StateResV1 StateResVersion = 0
	// StateResV2 is state resolution v2 introduced by https://github.com/matrix-org/matrix-spec-proposals/pull/1759
	StateResV2 StateResVersion = 1
	// StateResV2_1 is state resolution v2.1 introduced by https://github.com/matrix-org/matrix-spec-proposals/pull/4297
	StateResV2_1 StateResVersion = 2
)

// StateResVersion returns the version of the state resolution algorithm used by this room version.
func (rv RoomVersion) StateResVersion() StateResVersion {
	switch rv {
	case RoomV0, RoomV1:
		return StateResV1
	case RoomV2, RoomV3, RoomV4, RoomV5, RoomV6, RoomV7, RoomV8, RoomV9, RoomV10, RoomV11:
		return StateResV2
	case RoomV12:
		return StateResV2_1
	default:
		panic(rv.unknownVersionError())
	}
}

type EventIDFormat int

const (
	// EventIDFormatCustom is the original format used by room v1 and v2.
	// Event IDs in this format are an arbitrary string followed by a colon and the server name.
	EventIDFormatCustom EventIDFormat = 0
	// EventIDFormatBase64 is the format used by room v3 introduced by https://github.com/matrix-org/matrix-spec-proposals/pull/1659.
	// Event IDs in this format are the standard unpadded base64-encoded SHA256 reference hash of the event.
	EventIDFormatBase64 EventIDFormat = 1
	// EventIDFormatURLSafeBase64 is the format used by room v4 and later introduced by https://github.com/matrix-org/matrix-spec-proposals/pull/2002.
	// Event IDs in this format are the url-safe unpadded base64-encoded SHA256 reference hash of the event.
	EventIDFormatURLSafeBase64 EventIDFormat = 2
)

// EventIDFormat returns the format of event IDs used by this room version.
func (rv RoomVersion) EventIDFormat() EventIDFormat {
	switch rv {
	case RoomV0, RoomV1, RoomV2:
		return EventIDFormatCustom
	case RoomV3:
		return EventIDFormatBase64
	default:
		return EventIDFormatURLSafeBase64
	}
}

/////////////////////
// Room v5 changes //
/////////////////////
// https://github.com/matrix-org/matrix-spec-proposals/pull/2077

// EnforceSigningKeyValidity returns true if the `valid_until_ts` field of federation signing keys
// must be enforced on received events.
//
// See https://github.com/matrix-org/matrix-spec-proposals/pull/2076
func (rv RoomVersion) EnforceSigningKeyValidity() bool {
	return rv.NotEquals(RoomV0, RoomV1, RoomV2, RoomV3, RoomV4)
}

/////////////////////
// Room v6 changes //
/////////////////////
// https://github.com/matrix-org/matrix-spec-proposals/pull/2240

// SpecialCasedAliasesAuth returns true if the `m.room.aliases` event authorization is special cased
// to only always allow servers to modify the state event with their own server name as state key.
// This also implies that the `aliases` field is protected from redactions.
//
// See https://github.com/matrix-org/matrix-spec-proposals/pull/2432
func (rv RoomVersion) SpecialCasedAliasesAuth() bool {
	return rv.Equals(RoomV0, RoomV1, RoomV2, RoomV3, RoomV4, RoomV5)
}

// ForbidFloatsAndBigInts returns true if floats and integers greater than 2^53-1 or lower than -2^53+1 are forbidden everywhere.
//
// See https://github.com/matrix-org/matrix-spec-proposals/pull/2540
func (rv RoomVersion) ForbidFloatsAndBigInts() bool {
	return rv.NotEquals(RoomV0, RoomV1, RoomV2, RoomV3, RoomV4, RoomV5)
}

// NotificationsPowerLevels returns true if the `notifications` field in `m.room.power_levels` is validated in event auth.
// However, the field is not protected from redactions.
//
// See https://github.com/matrix-org/matrix-spec-proposals/pull/2209
func (rv RoomVersion) NotificationsPowerLevels() bool {
	return rv.NotEquals(RoomV0, RoomV1, RoomV2, RoomV3, RoomV4, RoomV5)
}

/////////////////////
// Room v7 changes //
/////////////////////
// https://github.com/matrix-org/matrix-spec-proposals/pull/2998

// Knocks returns true if the `knock` join rule is supported.
//
// See https://github.com/matrix-org/matrix-spec-proposals/pull/2403
func (rv RoomVersion) Knocks() bool {
	return rv.NotEquals(RoomV0, RoomV1, RoomV2, RoomV3, RoomV4, RoomV5, RoomV6)
}

/////////////////////
// Room v8 changes //
/////////////////////
// https://github.com/matrix-org/matrix-spec-proposals/pull/3289

// RestrictedJoins returns true if the `restricted` join rule is supported.
// This also implies that the `allow` field in the `m.room.join_rules` event is supported and protected from redactions.
//
// See https://github.com/matrix-org/matrix-spec-proposals/pull/3083
func (rv RoomVersion) RestrictedJoins() bool {
	return rv.NotEquals(RoomV0, RoomV1, RoomV2, RoomV3, RoomV4, RoomV5, RoomV6, RoomV7)
}

/////////////////////
// Room v9 changes //
/////////////////////
// https://github.com/matrix-org/matrix-spec-proposals/pull/3375

// RestrictedJoinsFix returns true if the `join_authorised_via_users_server` field in `m.room.member` events is protected from redactions.
//
// See https://github.com/matrix-org/matrix-spec-proposals/pull/3375
func (rv RoomVersion) RestrictedJoinsFix() bool {
	return rv.RestrictedJoins() && rv != RoomV8
}

//////////////////////
// Room v10 changes //
//////////////////////
// https://github.com/matrix-org/matrix-spec-proposals/pull/3604

// ValidatePowerLevelInts returns true if the known values in `m.room.power_levels` must be integers (and not strings).
//
// See https://github.com/matrix-org/matrix-spec-proposals/pull/3667
func (rv RoomVersion) ValidatePowerLevelInts() bool {
	return rv.NotEquals(RoomV0, RoomV1, RoomV2, RoomV3, RoomV4, RoomV5, RoomV6, RoomV7, RoomV8, RoomV9)
}

// KnockRestricted returns true if the `knock_restricted` join rule is supported.
//
// See https://github.com/matrix-org/matrix-spec-proposals/pull/3787
func (rv RoomVersion) KnockRestricted() bool {
	return rv.NotEquals(RoomV0, RoomV1, RoomV2, RoomV3, RoomV4, RoomV5, RoomV6, RoomV7, RoomV8, RoomV9)
}

//////////////////////
// Room v11 changes //
//////////////////////
// https://github.com/matrix-org/matrix-spec-proposals/pull/3820

// CreatorInContent returns true if the `m.room.create` event has a `creator` field in content.
//
// See https://github.com/matrix-org/matrix-spec-proposals/pull/2175
func (rv RoomVersion) CreatorInContent() bool {
	return rv.Equals(RoomV0, RoomV1, RoomV2, RoomV3, RoomV4, RoomV5, RoomV6, RoomV7, RoomV8, RoomV9, RoomV10)
}

// RedactsInContent returns true if the `m.room.redaction` event has the `redacts` field in content instead of at the top level.
// The redaction protection is also moved from the top level to the content field.
//
// See https://github.com/matrix-org/matrix-spec-proposals/pull/2174
// (and https://github.com/matrix-org/matrix-spec-proposals/pull/2176 for the redaction protection).
func (rv RoomVersion) RedactsInContent() bool {
	return rv.NotEquals(RoomV0, RoomV1, RoomV2, RoomV3, RoomV4, RoomV5, RoomV6, RoomV7, RoomV8, RoomV9, RoomV10)
}

// UpdatedRedactionRules returns true if various updates to the redaction algorithm are applied.
//
// Specifically:
//
// * the `membership`, `origin`, and `prev_state` fields at the top level of all events are no longer protected.
// * the entire content of `m.room.create` is protected.
// * the `redacts` field in `m.room.redaction` content is protected instead of the top-level field.
// * the `m.room.power_levels` event protects the `invite` field in content.
// * the `signed` field inside the `third_party_invite` field in content of `m.room.member` events is protected.
//
// See https://github.com/matrix-org/matrix-spec-proposals/pull/2176,
// https://github.com/matrix-org/matrix-spec-proposals/pull/3821, and
// https://github.com/matrix-org/matrix-spec-proposals/pull/3989
func (rv RoomVersion) UpdatedRedactionRules() bool {
	return rv.NotEquals(RoomV0, RoomV1, RoomV2, RoomV3, RoomV4, RoomV5, RoomV6, RoomV7, RoomV8, RoomV9, RoomV10)
}

//////////////////////
// Room v12 changes //
//////////////////////
// https://github.com/matrix-org/matrix-spec-proposals/pull/4304

// Return value of StateResVersion was changed to StateResV2_1

// PrivilegedRoomCreators returns true if the creator(s) of a room always have infinite power level.
// This also implies that the `m.room.create` event has an `additional_creators` field,
// and that the creators can't be present in the `m.room.power_levels` event.
//
// See https://github.com/matrix-org/matrix-spec-proposals/pull/4289
func (rv RoomVersion) PrivilegedRoomCreators() bool {
	return rv.NotEquals(RoomV0, RoomV1, RoomV2, RoomV3, RoomV4, RoomV5, RoomV6, RoomV7, RoomV8, RoomV9, RoomV10, RoomV11)
}

// RoomIDIsCreateEventID returns true if the ID of rooms is the same as the ID of the `m.room.create` event.
// This also implies that `m.room.create` events do not have a `room_id` field.
//
// See https://github.com/matrix-org/matrix-spec-proposals/pull/4291
func (rv RoomVersion) RoomIDIsCreateEventID() bool {
	return rv.NotEquals(RoomV0, RoomV1, RoomV2, RoomV3, RoomV4, RoomV5, RoomV6, RoomV7, RoomV8, RoomV9, RoomV10, RoomV11)
}
