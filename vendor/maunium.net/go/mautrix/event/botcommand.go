// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package event

import (
	"encoding/json"
)

type BotCommandsEventContent struct {
	Sigil    string        `json:"sigil,omitempty"`
	Commands []*BotCommand `json:"commands,omitempty"`
}

type BotCommand struct {
	Syntax      string                   `json:"syntax"`
	Aliases     []string                 `json:"fi.mau.aliases,omitempty"` // Not in MSC (yet)
	Arguments   []*BotCommandArgument    `json:"arguments,omitempty"`
	Description *ExtensibleTextContainer `json:"description,omitempty"`
}

type BotArgumentType string

const (
	BotArgumentTypeString    BotArgumentType = "string"
	BotArgumentTypeEnum      BotArgumentType = "enum"
	BotArgumentTypeInteger   BotArgumentType = "integer"
	BotArgumentTypeBoolean   BotArgumentType = "boolean"
	BotArgumentTypeUserID    BotArgumentType = "user_id"
	BotArgumentTypeRoomID    BotArgumentType = "room_id"
	BotArgumentTypeRoomAlias BotArgumentType = "room_alias"
	BotArgumentTypeEventID   BotArgumentType = "event_id"
)

type BotCommandArgument struct {
	Type         BotArgumentType          `json:"type"`
	DefaultValue any                      `json:"fi.mau.default_value,omitempty"` // Not in MSC (yet)
	Description  *ExtensibleTextContainer `json:"description,omitempty"`
	Enum         []string                 `json:"enum,omitempty"`
	Variadic     bool                     `json:"variadic,omitempty"`
}

type BotCommandInput struct {
	Syntax    string          `json:"syntax"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}
