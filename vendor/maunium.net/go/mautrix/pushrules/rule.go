// Copyright (c) 2020 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package pushrules

import (
	"encoding/gob"
	"regexp"
	"strings"

	"go.mau.fi/util/exerrors"
	"go.mau.fi/util/glob"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func init() {
	gob.Register(PushRuleArray{})
	gob.Register(PushRuleMap{})
}

type PushRuleCollection interface {
	GetMatchingRule(room Room, evt *event.Event) *PushRule
	GetActions(room Room, evt *event.Event) PushActionArray
}

type PushRuleArray []*PushRule

func (rules PushRuleArray) SetType(typ PushRuleType) PushRuleArray {
	for _, rule := range rules {
		rule.Type = typ
	}
	return rules
}

func (rules PushRuleArray) GetMatchingRule(room Room, evt *event.Event) *PushRule {
	for _, rule := range rules {
		if !rule.Match(room, evt) {
			continue
		}
		return rule
	}
	return nil
}

func (rules PushRuleArray) GetActions(room Room, evt *event.Event) PushActionArray {
	return rules.GetMatchingRule(room, evt).GetActions()
}

type PushRuleMap struct {
	Map  map[string]*PushRule
	Type PushRuleType
}

func (rules PushRuleArray) SetTypeAndMap(typ PushRuleType) PushRuleMap {
	data := PushRuleMap{
		Map:  make(map[string]*PushRule),
		Type: typ,
	}
	for _, rule := range rules {
		rule.Type = typ
		data.Map[rule.RuleID] = rule
	}
	return data
}

func (ruleMap PushRuleMap) GetMatchingRule(room Room, evt *event.Event) *PushRule {
	var rule *PushRule
	var found bool
	switch ruleMap.Type {
	case RoomRule:
		rule, found = ruleMap.Map[string(evt.RoomID)]
	case SenderRule:
		rule, found = ruleMap.Map[string(evt.Sender)]
	}
	if found && rule.Match(room, evt) {
		return rule
	}
	return nil
}

func (ruleMap PushRuleMap) GetActions(room Room, evt *event.Event) PushActionArray {
	return ruleMap.GetMatchingRule(room, evt).GetActions()
}

func (ruleMap PushRuleMap) Unmap() PushRuleArray {
	array := make(PushRuleArray, len(ruleMap.Map))
	index := 0
	for _, rule := range ruleMap.Map {
		array[index] = rule
		index++
	}
	return array
}

type PushRuleType string

const (
	OverrideRule  PushRuleType = "override"
	ContentRule   PushRuleType = "content"
	RoomRule      PushRuleType = "room"
	SenderRule    PushRuleType = "sender"
	UnderrideRule PushRuleType = "underride"
)

type PushRule struct {
	// The type of this rule.
	Type PushRuleType `json:"-"`
	// The ID of this rule.
	// For room-specific rules and user-specific rules, this is the room or user ID (respectively)
	// For other types of rules, this doesn't affect anything.
	RuleID string `json:"rule_id"`
	// The actions this rule should trigger when matched.
	Actions PushActionArray `json:"actions"`
	// Whether this is a default rule, or has been set explicitly.
	Default bool `json:"default"`
	// Whether or not this push rule is enabled.
	Enabled bool `json:"enabled"`
	// The conditions to match in order to trigger this rule.
	// Only applicable to generic underride/override rules.
	Conditions []*PushCondition `json:"conditions,omitempty"`
	// Pattern for content-specific push rules
	Pattern string `json:"pattern,omitempty"`
}

func (rule *PushRule) GetActions() PushActionArray {
	if rule == nil {
		return nil
	}
	return rule.Actions
}

func (rule *PushRule) Match(room Room, evt *event.Event) bool {
	if rule == nil || !rule.Enabled {
		return false
	}
	if rule.RuleID == ".m.rule.contains_display_name" || rule.RuleID == ".m.rule.contains_user_name" || rule.RuleID == ".m.rule.roomnotif" {
		if _, containsMentions := evt.Content.Raw["m.mentions"]; containsMentions {
			// Disable legacy mention push rules when the event contains the new mentions key
			return false
		}
	}
	switch rule.Type {
	case OverrideRule, UnderrideRule:
		return rule.matchConditions(room, evt)
	case ContentRule:
		return rule.matchPattern(room, evt)
	case RoomRule:
		return id.RoomID(rule.RuleID) == evt.RoomID
	case SenderRule:
		return id.UserID(rule.RuleID) == evt.Sender
	default:
		return false
	}
}

func (rule *PushRule) matchConditions(room Room, evt *event.Event) bool {
	for _, cond := range rule.Conditions {
		if !cond.Match(room, evt) {
			return false
		}
	}
	return true
}

func (rule *PushRule) matchPattern(room Room, evt *event.Event) bool {
	msg, ok := evt.Content.Raw["body"].(string)
	if !ok {
		return false
	}
	var buf strings.Builder
	// As per https://spec.matrix.org/unstable/client-server-api/#push-rules, content rules are case-insensitive
	// and must match whole words, so wrap the converted glob in (?i) and \b.
	buf.WriteString(`(?i)\b`)
	// strings.Builder will never return errors
	exerrors.PanicIfNotNil(glob.ToRegexPattern(rule.Pattern, &buf))
	buf.WriteString(`\b`)
	pattern, err := regexp.Compile(buf.String())
	if err != nil {
		return false
	}
	return pattern.MatchString(msg)
}
