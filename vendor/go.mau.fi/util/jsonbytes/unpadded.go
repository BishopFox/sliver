// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package jsonbytes

import (
	"encoding/base64"
	"encoding/json"
)

// UnpaddedBytes is a byte slice that is encoded and decoded using
// [base64.RawStdEncoding] instead of the default padded base64.
type UnpaddedBytes []byte

func (b UnpaddedBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(base64.RawStdEncoding.EncodeToString(b))
}

func (b *UnpaddedBytes) UnmarshalJSON(data []byte) error {
	var b64str string
	err := json.Unmarshal(data, &b64str)
	if err != nil {
		return err
	}
	*b, err = base64.RawStdEncoding.DecodeString(b64str)
	return err
}

// UnpaddedURLBytes is a byte slice that is encoded and decoded using
// [base64.RawURLEncoding] instead of the default padded base64.
type UnpaddedURLBytes []byte

func (b UnpaddedURLBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(base64.RawURLEncoding.EncodeToString(b))
}

func (b *UnpaddedURLBytes) UnmarshalJSON(data []byte) error {
	var b64str string
	err := json.Unmarshal(data, &b64str)
	if err != nil {
		return err
	}
	*b, err = base64.RawURLEncoding.DecodeString(b64str)
	return err
}
