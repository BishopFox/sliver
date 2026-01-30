// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package olm

import "maunium.net/go/mautrix/id"

type OutboundGroupSession interface {
	// Pickle returns a Session as a base64 string. Encrypts the Session using
	// the supplied key.
	Pickle(key []byte) ([]byte, error)

	// Unpickle loads an [OutboundGroupSession] from a pickled base64 string.
	// Decrypts the [OutboundGroupSession] using the supplied key.
	Unpickle(pickled, key []byte) error

	// Encrypt encrypts a message using the [OutboundGroupSession]. Returns the
	// encrypted message as base64.
	Encrypt(plaintext []byte) ([]byte, error)

	// ID returns a base64-encoded identifier for this session.
	ID() id.SessionID

	// MessageIndex returns the message index for this session.  Each message
	// is sent with an increasing index; this returns the index for the next
	// message.
	MessageIndex() uint

	// Key returns the base64-encoded current ratchet key for this session.
	Key() string
}

var InitNewOutboundGroupSessionFromPickled func(pickled, key []byte) (OutboundGroupSession, error)
var InitNewOutboundGroupSession func() (OutboundGroupSession, error)
var InitNewBlankOutboundGroupSession func() OutboundGroupSession

// OutboundGroupSessionFromPickled loads an OutboundGroupSession from a pickled
// base64 string.  Decrypts the OutboundGroupSession using the supplied key.
// Returns error on failure.  If the key doesn't match the one used to encrypt
// the OutboundGroupSession then the error will be "BAD_SESSION_KEY".  If the
// base64 couldn't be decoded then the error will be "INVALID_BASE64".
func OutboundGroupSessionFromPickled(pickled, key []byte) (OutboundGroupSession, error) {
	return InitNewOutboundGroupSessionFromPickled(pickled, key)
}

// NewOutboundGroupSession creates a new outbound group session.
func NewOutboundGroupSession() (OutboundGroupSession, error) {
	return InitNewOutboundGroupSession()
}

// NewBlankOutboundGroupSession initialises an empty [OutboundGroupSession].
func NewBlankOutboundGroupSession() OutboundGroupSession {
	return InitNewBlankOutboundGroupSession()
}
