// Copyright (c) 2020 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package pushrules

import (
	"encoding/gob"
	"encoding/json"
	"reflect"

	"maunium.net/go/mautrix/event"
)

// EventContent represents the content of a m.push_rules account data event.
// https://spec.matrix.org/v1.2/client-server-api/#mpush_rules
type EventContent struct {
	Ruleset *PushRuleset `json:"global"`
}

func init() {
	event.TypeMap[event.AccountDataPushRules] = reflect.TypeOf(EventContent{})
	gob.Register(&EventContent{})
}

// EventToPushRules converts a m.push_rules event to a PushRuleset by passing the data through JSON.
func EventToPushRules(evt *event.Event) (*PushRuleset, error) {
	content := &EventContent{}
	err := json.Unmarshal(evt.Content.VeryRaw, content)
	if err != nil {
		return nil, err
	}

	return content.Ruleset, nil
}
