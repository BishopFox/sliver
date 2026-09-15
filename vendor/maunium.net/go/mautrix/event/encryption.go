// Copyright (c) 2020 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package event

import (
	"encoding/json"
	"fmt"

	"go.mau.fi/util/jsonbytes"

	"maunium.net/go/mautrix/id"
)

// EncryptionEventContent represents the content of a m.room.encryption state event.
// https://spec.matrix.org/v1.2/client-server-api/#mroomencryption
type EncryptionEventContent struct {
	// The encryption algorithm to be used to encrypt messages sent in this room. Must be 'm.megolm.v1.aes-sha2'.
	Algorithm id.Algorithm `json:"algorithm"`
	// How long the session should be used before changing it. 604800000 (a week) is the recommended default.
	RotationPeriodMillis int64 `json:"rotation_period_ms,omitempty"`
	// How many messages should be sent before changing the session. 100 is the recommended default.
	RotationPeriodMessages int `json:"rotation_period_msgs,omitempty"`
}

// EncryptedEventContent represents the content of a m.room.encrypted message event.
// https://spec.matrix.org/v1.2/client-server-api/#mroomencrypted
//
// Note that sender_key and device_id are deprecated in Megolm events as of https://github.com/matrix-org/matrix-spec-proposals/pull/3700
type EncryptedEventContent struct {
	Algorithm id.Algorithm `json:"algorithm"`
	SenderKey id.SenderKey `json:"sender_key,omitempty"`
	// Deprecated: Matrix v1.3
	DeviceID id.DeviceID `json:"device_id,omitempty"`
	// Only present for Megolm events
	SessionID id.SessionID `json:"session_id,omitempty"`
	// Only present for beeper stream events
	StreamID string `json:"stream_id,omitempty"`
	// Only present for beeper stream events
	IV jsonbytes.UnpaddedBytes `json:"iv,omitempty"`

	Ciphertext json.RawMessage `json:"ciphertext"`

	MegolmCiphertext []byte `json:"-"`
	// Only present for beeper stream events
	BeeperStreamCiphertext jsonbytes.UnpaddedBytes `json:"-"`
	OlmCiphertext          OlmCiphertexts          `json:"-"`

	RelatesTo *RelatesTo `json:"m.relates_to,omitempty"`
	Mentions  *Mentions  `json:"m.mentions,omitempty"`
}

type OlmCiphertexts map[id.Curve25519]struct {
	Body string        `json:"body"`
	Type id.OlmMsgType `json:"type"`
}

type serializableEncryptedEventContent EncryptedEventContent

func (content *EncryptedEventContent) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, (*serializableEncryptedEventContent)(content))
	if err != nil {
		return err
	}
	switch content.Algorithm {
	case id.AlgorithmOlmV1:
		content.OlmCiphertext = make(OlmCiphertexts)
		return json.Unmarshal(content.Ciphertext, &content.OlmCiphertext)
	case id.AlgorithmMegolmV1:
		if len(content.Ciphertext) == 0 || content.Ciphertext[0] != '"' || content.Ciphertext[len(content.Ciphertext)-1] != '"' {
			return fmt.Errorf("ciphertext %w", id.ErrInputNotJSONString)
		}
		content.MegolmCiphertext = content.Ciphertext[1 : len(content.Ciphertext)-1]
	case id.AlgorithmBeeperStreamV1:
		return json.Unmarshal(content.Ciphertext, &content.BeeperStreamCiphertext)
	}
	return nil
}

func (content *EncryptedEventContent) MarshalJSON() ([]byte, error) {
	var err error
	switch content.Algorithm {
	case id.AlgorithmOlmV1:
		content.Ciphertext, err = json.Marshal(content.OlmCiphertext)
	case id.AlgorithmMegolmV1:
		content.Ciphertext = make([]byte, len(content.MegolmCiphertext)+2)
		content.Ciphertext[0] = '"'
		content.Ciphertext[len(content.Ciphertext)-1] = '"'
		copy(content.Ciphertext[1:len(content.Ciphertext)-1], content.MegolmCiphertext)
	case id.AlgorithmBeeperStreamV1:
		content.Ciphertext, err = json.Marshal(content.BeeperStreamCiphertext)
	}
	if err != nil {
		return nil, err
	}
	return json.Marshal((*serializableEncryptedEventContent)(content))
}

// RoomKeyEventContent represents the content of a m.room_key to_device event.
// https://spec.matrix.org/v1.2/client-server-api/#mroom_key
type RoomKeyEventContent struct {
	Algorithm     id.Algorithm `json:"algorithm"`
	RoomID        id.RoomID    `json:"room_id"`
	SessionID     id.SessionID `json:"session_id"`
	SessionKey    string       `json:"session_key"`
	SharedHistory *bool        `json:"shared_history,omitempty"`

	MaxAge      int64 `json:"com.beeper.max_age_ms,omitempty"`
	MaxMessages int   `json:"com.beeper.max_messages,omitempty"`
	IsScheduled bool  `json:"com.beeper.is_scheduled,omitempty"`
}

// ForwardedRoomKeyEventContent represents the content of a m.forwarded_room_key to_device event.
// https://spec.matrix.org/v1.2/client-server-api/#mforwarded_room_key
type ForwardedRoomKeyEventContent struct {
	RoomKeyEventContent
	SenderKey          id.SenderKey `json:"sender_key"`
	SenderClaimedKey   id.Ed25519   `json:"sender_claimed_ed25519_key"`
	ForwardingKeyChain []string     `json:"forwarding_curve25519_key_chain"`
}

type RoomKeyBundleEventContent struct {
	File   EncryptedFileInfo `json:"file"`
	RoomID id.RoomID         `json:"room_id"`
}

type KeyRequestAction string

const (
	KeyRequestActionRequest = "request"
	KeyRequestActionCancel  = "request_cancellation"
)

// RoomKeyRequestEventContent represents the content of a m.room_key_request to_device event.
// https://spec.matrix.org/v1.2/client-server-api/#mroom_key_request
type RoomKeyRequestEventContent struct {
	Body               RequestedKeyInfo `json:"body"`
	Action             KeyRequestAction `json:"action"`
	RequestingDeviceID id.DeviceID      `json:"requesting_device_id"`
	RequestID          string           `json:"request_id"`
}

type RequestedKeyInfo struct {
	Algorithm id.Algorithm `json:"algorithm"`
	RoomID    id.RoomID    `json:"room_id"`
	SessionID id.SessionID `json:"session_id"`
	// Deprecated: Matrix v1.3
	SenderKey id.SenderKey `json:"sender_key"`
}

type RoomKeyWithheldCode string

const (
	RoomKeyWithheldBlacklisted  RoomKeyWithheldCode = "m.blacklisted"
	RoomKeyWithheldUnverified   RoomKeyWithheldCode = "m.unverified"
	RoomKeyWithheldUnauthorized RoomKeyWithheldCode = "m.unauthorised"
	RoomKeyWithheldUnavailable  RoomKeyWithheldCode = "m.unavailable"
	RoomKeyWithheldNoOlmSession RoomKeyWithheldCode = "m.no_olm"

	RoomKeyWithheldBeeperRedacted RoomKeyWithheldCode = "com.beeper.redacted"
)

type RoomKeyWithheldEventContent struct {
	RoomID    id.RoomID           `json:"room_id,omitempty"`
	Algorithm id.Algorithm        `json:"algorithm"`
	SessionID id.SessionID        `json:"session_id,omitempty"`
	SenderKey id.SenderKey        `json:"sender_key"`
	Code      RoomKeyWithheldCode `json:"code"`
	Reason    string              `json:"reason,omitempty"`
}

const groupSessionWithheldMsg = "group session has been withheld: %s"

func (withheld *RoomKeyWithheldEventContent) Error() string {
	switch withheld.Code {
	case RoomKeyWithheldBlacklisted, RoomKeyWithheldUnverified, RoomKeyWithheldUnauthorized, RoomKeyWithheldUnavailable, RoomKeyWithheldNoOlmSession:
		return fmt.Sprintf(groupSessionWithheldMsg, withheld.Code)
	default:
		return fmt.Sprintf(groupSessionWithheldMsg+" (%s)", withheld.Code, withheld.Reason)
	}
}

func (withheld *RoomKeyWithheldEventContent) Is(other error) bool {
	otherWithheld, ok := other.(*RoomKeyWithheldEventContent)
	if !ok {
		return false
	}
	return withheld.Code == "" || otherWithheld.Code == "" || withheld.Code == otherWithheld.Code
}

type SecretRequestAction string

func (a SecretRequestAction) String() string {
	return string(a)
}

const (
	SecretRequestRequest      = "request"
	SecretRequestCancellation = "request_cancellation"
)

type SecretRequestEventContent struct {
	Name               id.Secret           `json:"name,omitempty"`
	Action             SecretRequestAction `json:"action"`
	RequestingDeviceID id.DeviceID         `json:"requesting_device_id"`
	RequestID          string              `json:"request_id"`
}

type SecretSendEventContent struct {
	RequestID string `json:"request_id"`
	Secret    string `json:"secret"`
}

type DummyEventContent struct{}
