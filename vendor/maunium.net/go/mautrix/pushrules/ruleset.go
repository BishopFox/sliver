// Copyright (c) 2020 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package pushrules

import (
	"encoding/json"

	"maunium.net/go/mautrix/event"
)

type PushRuleset struct {
	Override  PushRuleArray
	Content   PushRuleArray
	Room      PushRuleMap
	Sender    PushRuleMap
	Underride PushRuleArray
}

type rawPushRuleset struct {
	Override  PushRuleArray `json:"override"`
	Content   PushRuleArray `json:"content"`
	Room      PushRuleArray `json:"room"`
	Sender    PushRuleArray `json:"sender"`
	Underride PushRuleArray `json:"underride"`
}

// UnmarshalJSON parses JSON into this PushRuleset.
//
// For override, sender and underride push rule arrays, the type is added
// to each PushRule and the array is used as-is.
//
// For room and sender push rule arrays, the type is added to each PushRule
// and the array is converted to a map with the rule ID as the key and the
// PushRule as the value.
func (rs *PushRuleset) UnmarshalJSON(raw []byte) (err error) {
	data := rawPushRuleset{}
	err = json.Unmarshal(raw, &data)
	if err != nil {
		return
	}

	rs.Override = data.Override.SetType(OverrideRule)
	rs.Content = data.Content.SetType(ContentRule)
	rs.Room = data.Room.SetTypeAndMap(RoomRule)
	rs.Sender = data.Sender.SetTypeAndMap(SenderRule)
	rs.Underride = data.Underride.SetType(UnderrideRule)
	return
}

// MarshalJSON is the reverse of UnmarshalJSON()
func (rs *PushRuleset) MarshalJSON() ([]byte, error) {
	data := rawPushRuleset{
		Override:  rs.Override,
		Content:   rs.Content,
		Room:      rs.Room.Unmap(),
		Sender:    rs.Sender.Unmap(),
		Underride: rs.Underride,
	}
	return json.Marshal(&data)
}

// DefaultPushActions is the value returned if none of the rule
// collections in a Ruleset match the event given to GetActions()
var DefaultPushActions = PushActionArray{&PushAction{Action: ActionDontNotify}}

func (rs *PushRuleset) GetMatchingRule(room Room, evt *event.Event) (rule *PushRule) {
	if rs == nil {
		return nil
	}
	// Add push rule collections to array in priority order
	arrays := []PushRuleCollection{rs.Override, rs.Content, rs.Room, rs.Sender, rs.Underride}
	// Loop until one of the push rule collections matches the room/event combo.
	for _, pra := range arrays {
		if pra == nil {
			continue
		}
		if rule = pra.GetMatchingRule(room, evt); rule != nil {
			// Match found, return it.
			return
		}
	}
	// No match found
	return nil
}

// GetActions matches the given event against all of the push rule
// collections in this push ruleset in the order of priority as
// specified in spec section 11.12.1.4.
func (rs *PushRuleset) GetActions(room Room, evt *event.Event) (match PushActionArray) {
	actions := rs.GetMatchingRule(room, evt).GetActions()
	if actions == nil {
		// No match found, return default actions.
		return DefaultPushActions
	}
	return actions
}
