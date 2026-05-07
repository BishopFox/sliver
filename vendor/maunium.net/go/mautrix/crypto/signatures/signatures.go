// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package signatures

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.mau.fi/util/exgjson"

	"maunium.net/go/mautrix/crypto/canonicaljson"
	"maunium.net/go/mautrix/crypto/goolm/crypto"
	"maunium.net/go/mautrix/id"
)

var (
	ErrEmptyInput        = errors.New("empty input")
	ErrSignatureNotFound = errors.New("input JSON doesn't contain signature from specified device")
)

// Signatures represents a set of signatures for some data from multiple users
// and keys.
type Signatures map[id.UserID]map[id.KeyID]string

// NewSingleSignature creates a new [Signatures] object with a single
// signature.
func NewSingleSignature(userID id.UserID, algorithm id.KeyAlgorithm, keyID string, signature string) Signatures {
	return Signatures{
		userID: {
			id.NewKeyID(algorithm, keyID): signature,
		},
	}
}

// VerifySignature verifies an Ed25519 signature.
func VerifySignature(message []byte, key id.Ed25519, signature []byte) (ok bool, err error) {
	if len(message) == 0 || len(key) == 0 || len(signature) == 0 {
		return false, ErrEmptyInput
	}
	keyDecoded, err := base64.RawStdEncoding.DecodeString(key.String())
	if err != nil {
		return false, err
	}
	publicKey := crypto.Ed25519PublicKey(keyDecoded)
	return publicKey.Verify(message, signature), nil
}

// VerifySignatureJSON verifies the signature in the given JSON object "obj"
// as described in [Appendix 3] of the Matrix Spec.
//
// This function is a wrapper over [Utility.VerifySignatureJSON] that creates
// and destroys the [Utility] object transparently.
//
// If the "obj" is not already a [json.RawMessage], it will re-encoded as JSON
// for the verification, so "json" tags will be honored.
//
// [Appendix 3]: https://spec.matrix.org/v1.9/appendices/#signing-json
func VerifySignatureJSON(obj any, userID id.UserID, keyName string, key id.Ed25519) (bool, error) {
	var err error
	objJSON, ok := obj.(json.RawMessage)
	if !ok {
		objJSON, err = json.Marshal(obj)
		if err != nil {
			return false, err
		}
	}

	sig := gjson.GetBytes(objJSON, exgjson.Path("signatures", string(userID), fmt.Sprintf("ed25519:%s", keyName)))
	if !sig.Exists() || sig.Type != gjson.String {
		return false, ErrSignatureNotFound
	}
	objJSON, err = sjson.DeleteBytes(objJSON, "unsigned")
	if err != nil {
		return false, err
	}
	objJSON, err = sjson.DeleteBytes(objJSON, "signatures")
	if err != nil {
		return false, err
	}
	objJSONString := canonicaljson.CanonicalJSONAssumeValid(objJSON)
	sigBytes, err := base64.RawStdEncoding.DecodeString(sig.Str)
	if err != nil {
		return false, err
	}
	return VerifySignature(objJSONString, key, sigBytes)
}
