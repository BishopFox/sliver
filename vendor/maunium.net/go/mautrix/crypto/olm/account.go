// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package olm

import (
	"maunium.net/go/mautrix/id"
)

type Account interface {
	// Pickle returns an Account as a base64 string. Encrypts the Account using the
	// supplied key.
	Pickle(key []byte) ([]byte, error)

	// Unpickle loads an Account from a pickled base64 string. Decrypts the
	// Account using the supplied key. Returns error on failure.
	Unpickle(pickled, key []byte) error

	// IdentityKeysJSON returns the public parts of the identity keys for the Account.
	IdentityKeysJSON() ([]byte, error)

	// IdentityKeys returns the public parts of the Ed25519 and Curve25519 identity
	// keys for the Account.
	IdentityKeys() (id.Ed25519, id.Curve25519, error)

	// Sign returns the signature of a message using the ed25519 key for this
	// Account.
	Sign(message []byte) ([]byte, error)

	// OneTimeKeys returns the public parts of the unpublished one time keys for
	// the Account.
	//
	// The returned data is a struct with the single value "Curve25519", which is
	// itself an object mapping key id to base64-encoded Curve25519 key.  For
	// example:
	//
	//	{
	//	    Curve25519: {
	//	        "AAAAAA": "wo76WcYtb0Vk/pBOdmduiGJ0wIEjW4IBMbbQn7aSnTo",
	//	        "AAAAAB": "LRvjo46L1X2vx69sS9QNFD29HWulxrmW11Up5AfAjgU"
	//	    }
	//	}
	OneTimeKeys() (map[string]id.Curve25519, error)

	// MarkKeysAsPublished marks the current set of one time keys as being
	// published.
	MarkKeysAsPublished()

	// MaxNumberOfOneTimeKeys returns the largest number of one time keys this
	// Account can store.
	MaxNumberOfOneTimeKeys() uint

	// GenOneTimeKeys generates a number of new one time keys.  If the total
	// number of keys stored by this Account exceeds MaxNumberOfOneTimeKeys
	// then the old keys are discarded.
	GenOneTimeKeys(num uint) error

	// NewOutboundSession creates a new out-bound session for sending messages to a
	// given curve25519 identityKey and oneTimeKey.  Returns error on failure.  If the
	// keys couldn't be decoded as base64 then the error will be "INVALID_BASE64"
	NewOutboundSession(theirIdentityKey, theirOneTimeKey id.Curve25519) (Session, error)

	// NewInboundSession creates a new in-bound session for sending/receiving
	// messages from an incoming PRE_KEY message.  Returns error on failure.  If
	// the base64 couldn't be decoded then the error will be "INVALID_BASE64".  If
	// the message was for an unsupported protocol version then the error will be
	// "BAD_MESSAGE_VERSION".  If the message couldn't be decoded then then the
	// error will be "BAD_MESSAGE_FORMAT".  If the message refers to an unknown one
	// time key then the error will be "BAD_MESSAGE_KEY_ID".
	NewInboundSession(oneTimeKeyMsg string) (Session, error)

	// NewInboundSessionFrom creates a new in-bound session for sending/receiving
	// messages from an incoming PRE_KEY message.  Returns error on failure.  If
	// the base64 couldn't be decoded then the error will be "INVALID_BASE64".  If
	// the message was for an unsupported protocol version then the error will be
	// "BAD_MESSAGE_VERSION".  If the message couldn't be decoded then then the
	// error will be "BAD_MESSAGE_FORMAT".  If the message refers to an unknown one
	// time key then the error will be "BAD_MESSAGE_KEY_ID".
	NewInboundSessionFrom(theirIdentityKey *id.Curve25519, oneTimeKeyMsg string) (Session, error)

	// RemoveOneTimeKeys removes the one time keys that the session used from the
	// Account.  Returns error on failure.  If the Account doesn't have any
	// matching one time keys then the error will be "BAD_MESSAGE_KEY_ID".
	RemoveOneTimeKeys(s Session) error
}

var Driver = "none"

var InitBlankAccount func() Account
var InitNewAccount func() (Account, error)
var InitNewAccountFromPickled func(pickled, key []byte) (Account, error)

// NewAccount creates a new Account.
func NewAccount() (Account, error) {
	return InitNewAccount()
}

func NewBlankAccount() Account {
	return InitBlankAccount()
}

// AccountFromPickled loads an Account from a pickled base64 string.  Decrypts
// the Account using the supplied key.  Returns error on failure.  If the key
// doesn't match the one used to encrypt the Account then the error will be
// "BAD_ACCOUNT_KEY".  If the base64 couldn't be decoded then the error will be
// "INVALID_BASE64".
func AccountFromPickled(pickled, key []byte) (Account, error) {
	return InitNewAccountFromPickled(pickled, key)
}
