// Copyright (c) 2023 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package pushrules

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/tidwall/gjson"
	"go.mau.fi/util/glob"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

// Room is an interface with the functions that are needed for processing room-specific push conditions
type Room interface {
	GetOwnDisplayname() string
	GetMemberCount() int
}

type PowerLevelfulRoom interface {
	Room
	GetPowerLevels() *event.PowerLevelsEventContent
}

// EventfulRoom is an extension of Room to support MSC3664.
type EventfulRoom interface {
	Room
	GetEvent(id.EventID) *event.Event
}

// PushCondKind is the type of a push condition.
type PushCondKind string

// The allowed push condition kinds as specified in https://spec.matrix.org/v1.2/client-server-api/#conditions-1
const (
	KindEventMatch                   PushCondKind = "event_match"
	KindContainsDisplayName          PushCondKind = "contains_display_name"
	KindRoomMemberCount              PushCondKind = "room_member_count"
	KindEventPropertyIs              PushCondKind = "event_property_is"
	KindEventPropertyContains        PushCondKind = "event_property_contains"
	KindSenderNotificationPermission PushCondKind = "sender_notification_permission"

	// MSC3664: https://github.com/matrix-org/matrix-spec-proposals/pull/3664

	KindRelatedEventMatch         PushCondKind = "related_event_match"
	KindUnstableRelatedEventMatch PushCondKind = "im.nheko.msc3664.related_event_match"
)

// PushCondition wraps a condition that is required for a specific PushRule to be used.
type PushCondition struct {
	// The type of the condition.
	Kind PushCondKind `json:"kind"`
	// The dot-separated field of the event to match. Only applicable if kind is EventMatch.
	Key string `json:"key,omitempty"`
	// The glob-style pattern to match the field against. Only applicable if kind is EventMatch.
	Pattern string `json:"pattern,omitempty"`
	// The exact value to match the field against. Only applicable if kind is EventPropertyIs or EventPropertyContains.
	Value any `json:"value,omitempty"`
	// The condition that needs to be fulfilled for RoomMemberCount-type conditions.
	// A decimal integer optionally prefixed by ==, <, >, >= or <=. Prefix "==" is assumed if no prefix found.
	MemberCountCondition string `json:"is,omitempty"`

	// The relation type for related_event_match from MSC3664
	RelType event.RelationType `json:"rel_type,omitempty"`
}

// MemberCountFilterRegex is the regular expression to parse the MemberCountCondition of PushConditions.
var MemberCountFilterRegex = regexp.MustCompile("^(==|[<>]=?)?([0-9]+)$")

// Match checks if this condition is fulfilled for the given event in the given room.
func (cond *PushCondition) Match(room Room, evt *event.Event) bool {
	switch cond.Kind {
	case KindEventMatch, KindEventPropertyIs, KindEventPropertyContains:
		return cond.matchValue(evt)
	case KindRelatedEventMatch, KindUnstableRelatedEventMatch:
		return cond.matchRelatedEvent(room, evt)
	case KindContainsDisplayName:
		return cond.matchDisplayName(room, evt)
	case KindRoomMemberCount:
		return cond.matchMemberCount(room)
	case KindSenderNotificationPermission:
		return cond.matchSenderNotificationPermission(room, evt.Sender, cond.Key)
	default:
		return false
	}
}

func splitWithEscaping(s string, separator, escape byte) []string {
	var token []byte
	var tokens []string
	for i := 0; i < len(s); i++ {
		if s[i] == separator {
			tokens = append(tokens, string(token))
			token = token[:0]
		} else if s[i] == escape && i+1 < len(s) {
			i++
			token = append(token, s[i])
		} else {
			token = append(token, s[i])
		}
	}
	tokens = append(tokens, string(token))
	return tokens
}

func hackyNestedGet(data map[string]any, path []string) (any, bool) {
	val, ok := data[path[0]]
	if len(path) == 1 {
		// We don't have any more path parts, return the value regardless of whether it exists or not.
		return val, ok
	} else if ok {
		if mapVal, ok := val.(map[string]any); ok {
			val, ok = hackyNestedGet(mapVal, path[1:])
			if ok {
				return val, true
			}
		}
	}
	// If we don't find the key, try to combine the first two parts.
	// e.g. if the key is content.m.relates_to.rel_type, we'll first try data["m"], which will fail,
	//      then combine m and relates_to to get data["m.relates_to"], which should succeed.
	path[1] = path[0] + "." + path[1]
	return hackyNestedGet(data, path[1:])
}

func stringifyForPushCondition(val interface{}) string {
	// Implement MSC3862 to allow matching any type of field
	// https://github.com/matrix-org/matrix-spec-proposals/pull/3862
	switch typedVal := val.(type) {
	case string:
		return typedVal
	case nil:
		return "null"
	case float64:
		// Floats aren't allowed in Matrix events, but the JSON parser always stores numbers as floats,
		// so just handle that and convert to int
		return strconv.FormatInt(int64(typedVal), 10)
	default:
		return fmt.Sprint(val)
	}
}

func (cond *PushCondition) getValue(evt *event.Event) (any, bool) {
	key, subkey, _ := strings.Cut(cond.Key, ".")

	switch key {
	case "type":
		return evt.Type.Type, true
	case "sender":
		return evt.Sender.String(), true
	case "room_id":
		return evt.RoomID.String(), true
	case "state_key":
		if evt.StateKey == nil {
			return nil, false
		}
		return *evt.StateKey, true
	case "content":
		// Split the match key with escaping to implement https://github.com/matrix-org/matrix-spec-proposals/pull/3873
		splitKey := splitWithEscaping(subkey, '.', '\\')
		// Then do a hacky nested get that supports combining parts for the backwards-compat part of MSC3873
		return hackyNestedGet(evt.Content.Raw, splitKey)
	default:
		return nil, false
	}
}

func numberToInt64(a any) int64 {
	switch typed := a.(type) {
	case float64:
		return int64(typed)
	case float32:
		return int64(typed)
	case int:
		return int64(typed)
	case int8:
		return int64(typed)
	case int16:
		return int64(typed)
	case int32:
		return int64(typed)
	case int64:
		return typed
	case uint:
		return int64(typed)
	case uint8:
		return int64(typed)
	case uint16:
		return int64(typed)
	case uint32:
		return int64(typed)
	case uint64:
		return int64(typed)
	default:
		return 0
	}
}

func valueEquals(a, b any) bool {
	// Convert floats to ints when comparing numbers (the JSON parser generates floats, but Matrix only allows integers)
	// Also allow other numeric types in case something generates events manually without json
	switch a.(type) {
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		switch b.(type) {
		case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return numberToInt64(a) == numberToInt64(b)
		}
	}
	return a == b
}

func (cond *PushCondition) matchValue(evt *event.Event) bool {
	val, ok := cond.getValue(evt)
	if !ok {
		return false
	}

	switch cond.Kind {
	case KindEventMatch, KindRelatedEventMatch, KindUnstableRelatedEventMatch:
		pattern := glob.CompileWithImplicitContains(cond.Pattern)
		if pattern == nil {
			return false
		}
		return pattern.Match(stringifyForPushCondition(val))
	case KindEventPropertyIs:
		return valueEquals(val, cond.Value)
	case KindEventPropertyContains:
		valArr, ok := val.([]any)
		if !ok {
			return false
		}
		for _, item := range valArr {
			if valueEquals(item, cond.Value) {
				return true
			}
		}
		return false
	default:
		panic(fmt.Errorf("matchValue called for unknown condition kind %s", cond.Kind))
	}
}

func (cond *PushCondition) getRelationEventID(relatesTo *event.RelatesTo) id.EventID {
	if relatesTo == nil {
		return ""
	}
	switch cond.RelType {
	case "":
		return relatesTo.EventID
	case "m.in_reply_to":
		if relatesTo.IsFallingBack || relatesTo.InReplyTo == nil {
			return ""
		}
		return relatesTo.InReplyTo.EventID
	default:
		if relatesTo.Type != cond.RelType {
			return ""
		}
		return relatesTo.EventID
	}
}

func (cond *PushCondition) matchRelatedEvent(room Room, evt *event.Event) bool {
	var relatesTo *event.RelatesTo
	if relatable, ok := evt.Content.Parsed.(event.Relatable); ok {
		relatesTo = relatable.OptionalGetRelatesTo()
	} else {
		res := gjson.GetBytes(evt.Content.VeryRaw, `m\.relates_to`)
		if res.Exists() && res.IsObject() {
			_ = json.Unmarshal([]byte(res.Raw), &relatesTo)
		}
	}
	if evtID := cond.getRelationEventID(relatesTo); evtID == "" {
		return false
	} else if eventfulRoom, ok := room.(EventfulRoom); !ok {
		return false
	} else if evt = eventfulRoom.GetEvent(relatesTo.EventID); evt == nil {
		return false
	} else {
		return cond.matchValue(evt)
	}
}

func (cond *PushCondition) matchDisplayName(room Room, evt *event.Event) bool {
	displayname := room.GetOwnDisplayname()
	if len(displayname) == 0 {
		return false
	}

	msg, ok := evt.Content.Raw["body"].(string)
	if !ok {
		return false
	}

	isAcceptable := func(r uint8) bool {
		return unicode.IsSpace(rune(r)) || unicode.IsPunct(rune(r))
	}
	length := len(displayname)
	for index := strings.Index(msg, displayname); index != -1; index = strings.Index(msg, displayname) {
		if (index <= 0 || isAcceptable(msg[index-1])) && (index+length >= len(msg) || isAcceptable(msg[index+length])) {
			return true
		}
		msg = msg[index+len(displayname):]
	}
	return false
}

func (cond *PushCondition) matchMemberCount(room Room) bool {
	group := MemberCountFilterRegex.FindStringSubmatch(cond.MemberCountCondition)
	if len(group) != 3 {
		return false
	}

	operator := group[1]
	wantedMemberCount, _ := strconv.Atoi(group[2])

	memberCount := room.GetMemberCount()

	switch operator {
	case "==", "":
		return memberCount == wantedMemberCount
	case ">":
		return memberCount > wantedMemberCount
	case ">=":
		return memberCount >= wantedMemberCount
	case "<":
		return memberCount < wantedMemberCount
	case "<=":
		return memberCount <= wantedMemberCount
	default:
		// Should be impossible due to regex.
		return false
	}
}

func (cond *PushCondition) matchSenderNotificationPermission(room Room, sender id.UserID, key string) bool {
	if key != "room" {
		return false
	}
	plRoom, ok := room.(PowerLevelfulRoom)
	if !ok {
		return false
	}
	pls := plRoom.GetPowerLevels()
	if pls == nil {
		return false
	}
	return pls.GetUserLevel(sender) >= pls.Notifications.Room()
}
