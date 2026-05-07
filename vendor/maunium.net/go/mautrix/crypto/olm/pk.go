// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package olm

import (
	"maunium.net/go/mautrix/id"
)

// PKSigning is an interface for signing messages.
type PKSigning interface {
	// Seed returns the seed of the key.
	Seed() []byte

	// PublicKey returns the public key.
	PublicKey() id.Ed25519

	// Sign creates a signature for the given message using this key.
	Sign(message []byte) ([]byte, error)

	// SignJSON creates a signature for the given object after encoding it to
	// canonical JSON.
	SignJSON(obj any) (string, error)
}

// PKDecryption is an interface for decrypting messages.
type PKDecryption interface {
	// PublicKey returns the public key.
	PublicKey() id.Curve25519

	// Decrypt verifies and decrypts the given message.
	Decrypt(ephemeralKey, mac, ciphertext []byte) ([]byte, error)
}

var InitNewPKSigning func() (PKSigning, error)
var InitNewPKSigningFromSeed func(seed []byte) (PKSigning, error)
var InitNewPKDecryptionFromPrivateKey func(privateKey []byte) (PKDecryption, error)

// NewPKSigning creates a new [PKSigning] object, containing a key pair for
// signing messages.
func NewPKSigning() (PKSigning, error) {
	return InitNewPKSigning()
}

// NewPKSigningFromSeed creates a new PKSigning object using the given seed.
func NewPKSigningFromSeed(seed []byte) (PKSigning, error) {
	return InitNewPKSigningFromSeed(seed)
}

// NewPKDecryptionFromPrivateKey creates a new [PKDecryption] from a
// base64-encoded private key.
func NewPKDecryptionFromPrivateKey(privateKey []byte) (PKDecryption, error) {
	return InitNewPKDecryptionFromPrivateKey(privateKey)
}
