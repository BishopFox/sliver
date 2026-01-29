// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/go-teams-notify
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package adaptivecard

import (
	"encoding/json"
	"strings"
)

// Credit:
//
// These resources were used while developing the json.Marshaler and
// json.Unmarshler interface implementations used in this file:
//
// https://stackoverflow.com/questions/31048557/assigning-null-to-json-fields-instead-of-empty-strings
// https://stackoverflow.com/questions/25087960/json-unmarshal-time-that-isnt-in-rfc-3339-format/

// Add an "implements assertion" to fail the build if the json.Unmarshaler
// implementation isn't correct.
//
// This resolves the unparam linter error:
// (*NullString).UnmarshalJSON - result 0 (error) is always nil (unparam)
//
// https://github.com/mvdan/unparam/issues/52
var _ json.Unmarshaler = (*NullString)(nil)

// Perform similar "implements assertion" for the json.Marshaler interface.
var _ json.Marshaler = (*NullString)(nil)

// NullString represents a string value used in component fields that may
// potentially be null in the input JSON feed.
type NullString string

// MarshalJSON implements the json.Marshaler interface. This compliments the
// custom Unmarshaler implementation to handle potentially null component
// description field value.
func (ns NullString) MarshalJSON() ([]byte, error) {
	if len(string(ns)) == 0 {
		return []byte("null"), nil
	}

	// NOTE: If we fail to convert the type, an infinite loop will occur.
	return json.Marshal(string(ns))
}

// UnmarshalJSON implements the json.Unmarshaler interface to handle
// potentially null component description field value.
func (ns *NullString) UnmarshalJSON(data []byte) error {
	str := string(data)
	if str == "null" {
		*ns = ""
		return nil
	}

	*ns = NullString(strings.Trim(str, "\""))

	return nil
}
