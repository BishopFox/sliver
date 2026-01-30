// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package olm

import "maunium.net/go/mautrix/id"

type InboundGroupSession interface {
	// Pickle returns an InboundGroupSession as a base64 string.  Encrypts the
	// InboundGroupSession using the supplied key.
	Pickle(key []byte) ([]byte, error)

	// Unpickle loads an [InboundGroupSession] from a pickled base64 string.
	// Decrypts the [InboundGroupSession] using the supplied key.
	Unpickle(pickled, key []byte) error

	// Decrypt decrypts a message using the [InboundGroupSession]. Returns the
	// plain-text and message index on success.  Returns error on failure.  If
	// the base64 couldn't be decoded then the error will be "INVALID_BASE64".
	// If the message is for an unsupported version of the protocol then the
	// error will be "BAD_MESSAGE_VERSION".  If the message couldn't be decoded
	// then the error will be BAD_MESSAGE_FORMAT".  If the MAC on the message
	// was invalid then the error will be "BAD_MESSAGE_MAC".  If we do not have
	// a session key corresponding to the message's index (ie, it was sent
	// before the session key was shared with us) the error will be
	// "OLM_UNKNOWN_MESSAGE_INDEX".
	Decrypt(message []byte) ([]byte, uint, error)

	// ID returns a base64-encoded identifier for this session.
	ID() id.SessionID

	// FirstKnownIndex returns the first message index we know how to decrypt.
	FirstKnownIndex() uint32

	// IsVerified check if the session has been verified as a valid session.
	// (A session is verified either because the original session share was
	// signed, or because we have subsequently successfully decrypted a
	// message.)
	IsVerified() bool

	// Export returns the base64-encoded ratchet key for this session, at the
	// given index, in a format which can be used by
	// InboundGroupSession.InboundGroupSessionImport().  Encrypts the
	// InboundGroupSession using the supplied key.  Returns error on failure.
	// if we do not have a session key corresponding to the given index (ie, it
	// was sent before the session key was shared with us) the error will be
	// "OLM_UNKNOWN_MESSAGE_INDEX".
	Export(messageIndex uint32) ([]byte, error)
}

var InitInboundGroupSessionFromPickled func(pickled, key []byte) (InboundGroupSession, error)
var InitNewInboundGroupSession func(sessionKey []byte) (InboundGroupSession, error)
var InitInboundGroupSessionImport func(sessionKey []byte) (InboundGroupSession, error)
var InitBlankInboundGroupSession func() InboundGroupSession

// InboundGroupSessionFromPickled loads an InboundGroupSession from a pickled
// base64 string. Decrypts the InboundGroupSession using the supplied key.
// Returns error on failure.
func InboundGroupSessionFromPickled(pickled, key []byte) (InboundGroupSession, error) {
	return InitInboundGroupSessionFromPickled(pickled, key)
}

// NewInboundGroupSession creates a new inbound group session from a key
// exported from OutboundGroupSession.Key(). Returns error on failure.
func NewInboundGroupSession(sessionKey []byte) (InboundGroupSession, error) {
	return InitNewInboundGroupSession(sessionKey)
}

// InboundGroupSessionImport imports an inbound group session from a previous
// export. Returns error on failure.
func InboundGroupSessionImport(sessionKey []byte) (InboundGroupSession, error) {
	return InitInboundGroupSessionImport(sessionKey)
}

func NewBlankInboundGroupSession() InboundGroupSession {
	return InitBlankInboundGroupSession()
}
