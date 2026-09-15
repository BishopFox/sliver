// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package olm

import "errors"

// Those are the most common used errors
var (
	ErrBadSignature             = errors.New("bad signature")
	ErrBadMAC                   = errors.New("the message couldn't be decrypted (bad mac)")
	ErrBadMessageFormat         = errors.New("the message couldn't be decoded")
	ErrBadVerification          = errors.New("bad verification")
	ErrWrongProtocolVersion     = errors.New("wrong protocol version")
	ErrEmptyInput               = errors.New("empty input")
	ErrNoKeyProvided            = errors.New("no key provided")
	ErrBadMessageKeyID          = errors.New("the message references an unknown key ID")
	ErrUnknownMessageIndex      = errors.New("attempt to decode a message whose index is earlier than our earliest known session key")
	ErrMsgIndexTooHigh          = errors.New("message index too high")
	ErrProtocolViolation        = errors.New("not protocol message order")
	ErrMessageKeyNotFound       = errors.New("message key not found")
	ErrChainTooHigh             = errors.New("chain index too high")
	ErrBadInput                 = errors.New("bad input")
	ErrUnknownOlmPickleVersion  = errors.New("unknown olm pickle version")
	ErrUnknownJSONPickleVersion = errors.New("unknown JSON pickle version")
	ErrInputToSmall             = errors.New("input too small (truncated?)")
)

// Error codes from go-olm
var (
	ErrNotEnoughGoRandom  = errors.New("couldn't get enough randomness from crypto/rand")
	ErrInputNotJSONString = errors.New("input doesn't look like a JSON string")
)

// Error codes from olm code
var (
	ErrLibolmInvalidBase64 = errors.New("the input base64 was invalid")

	ErrLibolmNotEnoughRandom        = errors.New("not enough entropy was supplied")
	ErrLibolmOutputBufferTooSmall   = errors.New("supplied output buffer is too small")
	ErrLibolmBadAccountKey          = errors.New("the supplied account key is invalid")
	ErrLibolmCorruptedPickle        = errors.New("the pickled object couldn't be decoded")
	ErrLibolmBadSessionKey          = errors.New("attempt to initialise an inbound group session from an invalid session key")
	ErrLibolmBadLegacyAccountPickle = errors.New("attempt to unpickle an account which uses pickle version 1")
)

// Deprecated: use variables prefixed with Err
var (
	EmptyInput             = ErrEmptyInput
	BadSignature           = ErrBadSignature
	InvalidBase64          = ErrLibolmInvalidBase64
	BadMessageKeyID        = ErrBadMessageKeyID
	BadMessageFormat       = ErrBadMessageFormat
	BadMessageVersion      = ErrWrongProtocolVersion
	BadMessageMAC          = ErrBadMAC
	UnknownPickleVersion   = ErrUnknownOlmPickleVersion
	NotEnoughRandom        = ErrLibolmNotEnoughRandom
	OutputBufferTooSmall   = ErrLibolmOutputBufferTooSmall
	BadAccountKey          = ErrLibolmBadAccountKey
	CorruptedPickle        = ErrLibolmCorruptedPickle
	BadSessionKey          = ErrLibolmBadSessionKey
	UnknownMessageIndex    = ErrUnknownMessageIndex
	BadLegacyAccountPickle = ErrLibolmBadLegacyAccountPickle
	InputBufferTooSmall    = ErrInputToSmall
	NoKeyProvided          = ErrNoKeyProvided

	NotEnoughGoRandom  = ErrNotEnoughGoRandom
	InputNotJSONString = ErrInputNotJSONString

	ErrBadVersion          = ErrUnknownJSONPickleVersion
	ErrWrongPickleVersion  = ErrUnknownJSONPickleVersion
	ErrRatchetNotAvailable = ErrUnknownMessageIndex
)
