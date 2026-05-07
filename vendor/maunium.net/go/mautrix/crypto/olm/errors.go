// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package olm

import "errors"

// Those are the most common used errors
var (
	ErrBadSignature         = errors.New("bad signature")
	ErrBadMAC               = errors.New("bad mac")
	ErrBadMessageFormat     = errors.New("bad message format")
	ErrBadVerification      = errors.New("bad verification")
	ErrWrongProtocolVersion = errors.New("wrong protocol version")
	ErrEmptyInput           = errors.New("empty input")
	ErrNoKeyProvided        = errors.New("no key")
	ErrBadMessageKeyID      = errors.New("bad message key id")
	ErrRatchetNotAvailable  = errors.New("ratchet not available: attempt to decode a message whose index is earlier than our earliest known session key")
	ErrMsgIndexTooHigh      = errors.New("message index too high")
	ErrProtocolViolation    = errors.New("not protocol message order")
	ErrMessageKeyNotFound   = errors.New("message key not found")
	ErrChainTooHigh         = errors.New("chain index too high")
	ErrBadInput             = errors.New("bad input")
	ErrBadVersion           = errors.New("wrong version")
	ErrWrongPickleVersion   = errors.New("wrong pickle version")
	ErrInputToSmall         = errors.New("input too small (truncated?)")
	ErrOverflow             = errors.New("overflow")
)

// Error codes from go-olm
var (
	EmptyInput         = errors.New("empty input")
	NoKeyProvided      = errors.New("no pickle key provided")
	NotEnoughGoRandom  = errors.New("couldn't get enough randomness from crypto/rand")
	SignatureNotFound  = errors.New("input JSON doesn't contain signature from specified device")
	InputNotJSONString = errors.New("input doesn't look like a JSON string")
)

// Error codes from olm code
var (
	NotEnoughRandom        = errors.New("not enough entropy was supplied")
	OutputBufferTooSmall   = errors.New("supplied output buffer is too small")
	BadMessageVersion      = errors.New("the message version is unsupported")
	BadMessageFormat       = errors.New("the message couldn't be decoded")
	BadMessageMAC          = errors.New("the message couldn't be decrypted")
	BadMessageKeyID        = errors.New("the message references an unknown key ID")
	InvalidBase64          = errors.New("the input base64 was invalid")
	BadAccountKey          = errors.New("the supplied account key is invalid")
	UnknownPickleVersion   = errors.New("the pickled object is too new")
	CorruptedPickle        = errors.New("the pickled object couldn't be decoded")
	BadSessionKey          = errors.New("attempt to initialise an inbound group session from an invalid session key")
	UnknownMessageIndex    = errors.New("attempt to decode a message whose index is earlier than our earliest known session key")
	BadLegacyAccountPickle = errors.New("attempt to unpickle an account which uses pickle version 1")
	BadSignature           = errors.New("received message had a bad signature")
	InputBufferTooSmall    = errors.New("the input data was too small to be valid")
)
