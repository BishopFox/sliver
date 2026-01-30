// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package backup

import (
	"crypto/ecdh"
	"encoding/base64"
	"encoding/json"
)

// EphemeralKey is a wrapper around an ECDH X25519 public key that implements
// JSON marshalling and unmarshalling.
type EphemeralKey struct {
	*ecdh.PublicKey
}

func (k *EphemeralKey) MarshalJSON() ([]byte, error) {
	if k == nil || k.PublicKey == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(base64.RawStdEncoding.EncodeToString(k.Bytes()))
}

func (k *EphemeralKey) UnmarshalJSON(data []byte) error {
	var keyStr string
	err := json.Unmarshal(data, &keyStr)
	if err != nil {
		return err
	}

	keyBytes, err := base64.RawStdEncoding.DecodeString(keyStr)
	if err != nil {
		return err
	}
	k.PublicKey, err = ecdh.X25519().NewPublicKey(keyBytes)
	return err
}
