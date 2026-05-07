// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package backup

import (
	"crypto/ecdh"
	"crypto/rand"
)

// MegolmBackupKey is a wrapper around an ECDH X25519 private key that is used
// to decrypt a megolm key backup.
type MegolmBackupKey struct {
	*ecdh.PrivateKey
}

func NewMegolmBackupKey() (*MegolmBackupKey, error) {
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &MegolmBackupKey{key}, nil
}

func MegolmBackupKeyFromBytes(bytes []byte) (*MegolmBackupKey, error) {
	key, err := ecdh.X25519().NewPrivateKey(bytes)
	if err != nil {
		return nil, err
	}
	return &MegolmBackupKey{key}, nil
}
