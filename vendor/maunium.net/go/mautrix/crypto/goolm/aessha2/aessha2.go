// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package aessha2 implements the m.megolm.v1.aes-sha2 encryption algorithm
// described in [Section 10.12.4.3] in the Spec
//
// [Section 10.12.4.3]: https://spec.matrix.org/v1.12/client-server-api/#mmegolmv1aes-sha2
package aessha2

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"io"

	"golang.org/x/crypto/hkdf"

	"maunium.net/go/mautrix/crypto/aescbc"
)

type AESSHA2 struct {
	aesKey, hmacKey, iv []byte
}

func NewAESSHA2(secret, info []byte) (AESSHA2, error) {
	kdf := hkdf.New(sha256.New, secret, nil, info)
	keymatter := make([]byte, 80)
	_, err := io.ReadFull(kdf, keymatter)
	return AESSHA2{
		keymatter[:32],   // AES Key
		keymatter[32:64], // HMAC Key
		keymatter[64:],   // IV
	}, err
}

func (a *AESSHA2) Encrypt(plaintext []byte) ([]byte, error) {
	return aescbc.Encrypt(a.aesKey, a.iv, plaintext)
}

func (a *AESSHA2) Decrypt(ciphertext []byte) ([]byte, error) {
	return aescbc.Decrypt(a.aesKey, a.iv, ciphertext)
}

func (a *AESSHA2) MAC(ciphertext []byte) ([]byte, error) {
	hash := hmac.New(sha256.New, a.hmacKey)
	_, err := hash.Write(ciphertext)
	return hash.Sum(nil), err
}

func (a *AESSHA2) VerifyMAC(ciphertext, theirMAC []byte) (bool, error) {
	if mac, err := a.MAC(ciphertext); err != nil {
		return false, err
	} else {
		return subtle.ConstantTimeCompare(mac[:len(theirMAC)], theirMAC) == 1, nil
	}
}
