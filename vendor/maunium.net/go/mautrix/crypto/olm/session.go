// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package olm

import "maunium.net/go/mautrix/id"

type Session interface {
	// Pickle returns a Session as a base64 string. Encrypts the Session using
	// the supplied key.
	Pickle(key []byte) ([]byte, error)

	// Unpickle loads a Session from a pickled base64 string. Decrypts the
	// Session using the supplied key.
	Unpickle(pickled, key []byte) error

	// ID returns an identifier for this Session. Will be the same for both
	// ends of the conversation.
	ID() id.SessionID

	// HasReceivedMessage returns true if this session has received any
	// message.
	HasReceivedMessage() bool

	// MatchesInboundSession checks if the PRE_KEY message is for this in-bound
	// Session. This can happen if multiple messages are sent to this Account
	// before this Account sends a message in reply. Returns true if the
	// session matches. Returns false if the session does not match. Returns
	// error on failure. If the base64 couldn't be decoded then the error will
	// be "INVALID_BASE64". If the message was for an unsupported protocol
	// version then the error will be "BAD_MESSAGE_VERSION". If the message
	// couldn't be decoded then then the error will be "BAD_MESSAGE_FORMAT".
	MatchesInboundSession(oneTimeKeyMsg string) (bool, error)

	// MatchesInboundSessionFrom checks if the PRE_KEY message is for this
	// in-bound Session. This can happen if multiple messages are sent to this
	// Account before this Account sends a message in reply. Returns true if
	// the session matches. Returns false if the session does not match.
	// Returns error on failure. If the base64 couldn't be decoded then the
	// error will be "INVALID_BASE64". If the message was for an unsupported
	// protocol version then the error will be "BAD_MESSAGE_VERSION". If the
	// message couldn't be decoded then then the error will be
	// "BAD_MESSAGE_FORMAT".
	MatchesInboundSessionFrom(theirIdentityKey, oneTimeKeyMsg string) (bool, error)

	// EncryptMsgType returns the type of the next message that Encrypt will
	// return. Returns MsgTypePreKey if the message will be a PRE_KEY message.
	// Returns MsgTypeMsg if the message will be a normal message.
	EncryptMsgType() id.OlmMsgType

	// Encrypt encrypts a message using the Session. Returns the encrypted
	// message as base64.
	Encrypt(plaintext []byte) (id.OlmMsgType, []byte, error)

	// Decrypt decrypts a message using the Session. Returns the plain-text on
	// success. Returns error on failure. If the base64 couldn't be decoded
	// then the error will be "INVALID_BASE64". If the message is for an
	// unsupported version of the protocol then the error will be
	// "BAD_MESSAGE_VERSION". If the message couldn't be decoded then the error
	// will be BAD_MESSAGE_FORMAT". If the MAC on the message was invalid then
	// the error will be "BAD_MESSAGE_MAC".
	Decrypt(message string, msgType id.OlmMsgType) ([]byte, error)

	// Describe generates a string describing the internal state of an olm
	// session for debugging and logging purposes.
	Describe() string
}

var InitSessionFromPickled func(pickled, key []byte) (Session, error)
var InitNewBlankSession func() Session

// SessionFromPickled loads a Session from a pickled base64 string.  Decrypts
// the Session using the supplied key.  Returns error on failure.
func SessionFromPickled(pickled, key []byte) (Session, error) {
	return InitSessionFromPickled(pickled, key)
}

func NewBlankSession() Session {
	return InitNewBlankSession()
}
