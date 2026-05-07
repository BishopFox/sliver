// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package backup

import (
	"bytes"
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"

	"go.mau.fi/util/jsonbytes"
	"golang.org/x/crypto/hkdf"

	"maunium.net/go/mautrix/crypto/aescbc"
)

var ErrInvalidMAC = errors.New("invalid MAC")

// EncryptedSessionData is the encrypted session_data field of a key backup as
// defined in [Section 11.12.3.2.2 of the Spec].
//
// The type parameter T represents the format of the session data contained in
// the encrypted payload.
//
// [Section 11.12.3.2.2 of the Spec]: https://spec.matrix.org/v1.9/client-server-api/#backup-algorithm-mmegolm_backupv1curve25519-aes-sha2
type EncryptedSessionData[T any] struct {
	Ciphertext jsonbytes.UnpaddedBytes `json:"ciphertext"`
	Ephemeral  EphemeralKey            `json:"ephemeral"`
	MAC        jsonbytes.UnpaddedBytes `json:"mac"`
}

func calculateEncryptionParameters(sharedSecret []byte) (key, macKey, iv []byte, err error) {
	hkdfReader := hkdf.New(sha256.New, sharedSecret, nil, nil)
	encryptionParams := make([]byte, 80)
	_, err = hkdfReader.Read(encryptionParams)
	if err != nil {
		return nil, nil, nil, err
	}

	return encryptionParams[:32], encryptionParams[32:64], encryptionParams[64:], nil
}

// calculateCompatMAC calculates the MAC as described in step 5 of according to
// [Section 11.12.3.2.2] of the Spec which was updated in spec version 1.10 to
// reflect what is actually implemented in libolm and Vodozemac.
//
// Libolm implemented the MAC functionality incorrectly. The MAC is computed
// over an empty string rather than the ciphertext. Vodozemac implemented this
// functionality the same way as libolm for compatibility. In version 1.10 of
// the spec, the description of step 5 was updated to reflect the de-facto
// standard of libolm and Vodozemac.
//
// [Section 11.12.3.2.2]: https://spec.matrix.org/v1.11/client-server-api/#backup-algorithm-mmegolm_backupv1curve25519-aes-sha2
func calculateCompatMAC(macKey []byte) []byte {
	hash := hmac.New(sha256.New, macKey)
	return hash.Sum(nil)[:8]
}

// EncryptSessionData encrypts the given session data with the given recovery
// key as defined in [Section 11.12.3.2.2 of the Spec].
//
// [Section 11.12.3.2.2 of the Spec]: https://spec.matrix.org/v1.9/client-server-api/#backup-algorithm-mmegolm_backupv1curve25519-aes-sha2
func EncryptSessionData[T any](backupKey *MegolmBackupKey, sessionData T) (*EncryptedSessionData[T], error) {
	return EncryptSessionDataWithPubkey(backupKey.PublicKey(), sessionData)
}

func EncryptSessionDataWithPubkey[T any](pubkey *ecdh.PublicKey, sessionData T) (*EncryptedSessionData[T], error) {
	sessionJSON, err := json.Marshal(sessionData)
	if err != nil {
		return nil, err
	}

	ephemeralKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	sharedSecret, err := ephemeralKey.ECDH(pubkey)
	if err != nil {
		return nil, err
	}

	key, macKey, iv, err := calculateEncryptionParameters(sharedSecret)
	if err != nil {
		return nil, err
	}

	ciphertext, err := aescbc.Encrypt(key, iv, sessionJSON)
	if err != nil {
		return nil, err
	}

	return &EncryptedSessionData[T]{
		Ciphertext: ciphertext,
		Ephemeral:  EphemeralKey{ephemeralKey.PublicKey()},
		MAC:        calculateCompatMAC(macKey),
	}, nil
}

// Decrypt decrypts the [EncryptedSessionData] into a *T using the recovery key
// by reversing the process described in [Section 11.12.3.2.2 of the Spec].
//
// [Section 11.12.3.2.2 of the Spec]: https://spec.matrix.org/v1.9/client-server-api/#backup-algorithm-mmegolm_backupv1curve25519-aes-sha2
func (esd *EncryptedSessionData[T]) Decrypt(backupKey *MegolmBackupKey) (*T, error) {
	sharedSecret, err := backupKey.ECDH(esd.Ephemeral.PublicKey)
	if err != nil {
		return nil, err
	}

	key, macKey, iv, err := calculateEncryptionParameters(sharedSecret)
	if err != nil {
		return nil, err
	}

	// Verify the MAC before decrypting.
	if !bytes.Equal(calculateCompatMAC(macKey), esd.MAC) {
		return nil, ErrInvalidMAC
	}

	plaintext, err := aescbc.Decrypt(key, iv, esd.Ciphertext)
	if err != nil {
		return nil, err
	}

	var sessionData T
	err = json.Unmarshal(plaintext, &sessionData)
	return &sessionData, err
}
