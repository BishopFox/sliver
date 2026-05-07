// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package event

import (
	"fmt"
)

// Source represents the part of the sync response that an event came from.
type Source int

const (
	SourcePresence Source = 1 << iota
	SourceJoin
	SourceInvite
	SourceLeave
	SourceAccountData
	SourceTimeline
	SourceState
	SourceEphemeral
	SourceToDevice
	SourceDecrypted
)

const primaryTypes = SourcePresence | SourceAccountData | SourceToDevice | SourceTimeline | SourceState
const roomSections = SourceJoin | SourceInvite | SourceLeave
const roomableTypes = SourceAccountData | SourceTimeline | SourceState
const encryptableTypes = roomableTypes | SourceToDevice

func (es Source) String() string {
	var typeName string
	switch es & primaryTypes {
	case SourcePresence:
		typeName = "presence"
	case SourceAccountData:
		typeName = "account data"
	case SourceToDevice:
		typeName = "to-device"
	case SourceTimeline:
		typeName = "timeline"
	case SourceState:
		typeName = "state"
	default:
		return fmt.Sprintf("unknown (%d)", es)
	}
	if es&roomableTypes != 0 {
		switch es & roomSections {
		case SourceJoin:
			typeName = "joined room " + typeName
		case SourceInvite:
			typeName = "invited room " + typeName
		case SourceLeave:
			typeName = "left room " + typeName
		default:
			return fmt.Sprintf("unknown (%s+%d)", typeName, es)
		}
		es &^= roomSections
	}
	if es&encryptableTypes != 0 && es&SourceDecrypted != 0 {
		typeName += " (decrypted)"
		es &^= SourceDecrypted
	}
	es &^= primaryTypes
	if es != 0 {
		return fmt.Sprintf("unknown (%s+%d)", typeName, es)
	}
	return typeName
}
