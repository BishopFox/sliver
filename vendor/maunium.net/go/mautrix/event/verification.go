// Copyright (c) 2020 Nikos Filippakis
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package event

import (
	"go.mau.fi/util/jsonbytes"
	"go.mau.fi/util/jsontime"

	"maunium.net/go/mautrix/id"
)

type VerificationMethod string

const (
	VerificationMethodSAS VerificationMethod = "m.sas.v1"

	VerificationMethodReciprocate VerificationMethod = "m.reciprocate.v1"
	VerificationMethodQRCodeShow  VerificationMethod = "m.qr_code.show.v1"
	VerificationMethodQRCodeScan  VerificationMethod = "m.qr_code.scan.v1"
)

type VerificationTransactionable interface {
	GetTransactionID() id.VerificationTransactionID
	SetTransactionID(id.VerificationTransactionID)
}

// ToDeviceVerificationEvent contains the fields common to all to-device
// verification events.
type ToDeviceVerificationEvent struct {
	// TransactionID is an opaque identifier for the verification request. Must
	// be unique with respect to the devices involved.
	TransactionID id.VerificationTransactionID `json:"transaction_id,omitempty"`
}

var _ VerificationTransactionable = (*ToDeviceVerificationEvent)(nil)

func (ve *ToDeviceVerificationEvent) GetTransactionID() id.VerificationTransactionID {
	return ve.TransactionID
}

func (ve *ToDeviceVerificationEvent) SetTransactionID(id id.VerificationTransactionID) {
	ve.TransactionID = id
}

// InRoomVerificationEvent contains the fields common to all in-room
// verification events.
type InRoomVerificationEvent struct {
	// RelatesTo indicates the m.key.verification.request that this message is
	// related to. Note that for encrypted messages, this property should be in
	// the unencrypted portion of the event.
	RelatesTo *RelatesTo `json:"m.relates_to,omitempty"`
}

var _ Relatable = (*InRoomVerificationEvent)(nil)

func (ve *InRoomVerificationEvent) GetRelatesTo() *RelatesTo {
	if ve.RelatesTo == nil {
		ve.RelatesTo = &RelatesTo{}
	}
	return ve.RelatesTo
}

func (ve *InRoomVerificationEvent) OptionalGetRelatesTo() *RelatesTo {
	return ve.RelatesTo
}

func (ve *InRoomVerificationEvent) SetRelatesTo(rel *RelatesTo) {
	ve.RelatesTo = rel
}

// VerificationRequestEventContent represents the content of an
// [m.key.verification.request] to-device event as described in [Section
// 11.12.2.1] of the Spec.
//
// For the in-room version, use a standard [MessageEventContent] struct.
//
// [m.key.verification.request]: https://spec.matrix.org/v1.9/client-server-api/#mkeyverificationrequest
// [Section 11.12.2.1]: https://spec.matrix.org/v1.9/client-server-api/#key-verification-framework
type VerificationRequestEventContent struct {
	ToDeviceVerificationEvent
	// FromDevice is the device ID which is initiating the request.
	FromDevice id.DeviceID `json:"from_device"`
	// Methods is a list of the verification methods supported by the sender.
	Methods []VerificationMethod `json:"methods"`
	// Timestamp is the time at which the request was made.
	Timestamp jsontime.UnixMilli `json:"timestamp,omitempty"`
}

// VerificationRequestEventContentFromMessage converts an in-room verification
// request message event to a [VerificationRequestEventContent].
func VerificationRequestEventContentFromMessage(evt *Event) *VerificationRequestEventContent {
	content := evt.Content.AsMessage()
	return &VerificationRequestEventContent{
		ToDeviceVerificationEvent: ToDeviceVerificationEvent{
			TransactionID: id.VerificationTransactionID(evt.ID),
		},
		Timestamp:  jsontime.UMInt(evt.Timestamp),
		FromDevice: content.FromDevice,
		Methods:    content.Methods,
	}
}

// VerificationReadyEventContent represents the content of an
// [m.key.verification.ready] event (both the to-device and the in-room
// version) as described in [Section 11.12.2.1] of the Spec.
//
// [m.key.verification.ready]: https://spec.matrix.org/v1.9/client-server-api/#mkeyverificationready
// [Section 11.12.2.1]: https://spec.matrix.org/v1.9/client-server-api/#key-verification-framework
type VerificationReadyEventContent struct {
	ToDeviceVerificationEvent
	InRoomVerificationEvent

	// FromDevice is the device ID which is initiating the request.
	FromDevice id.DeviceID `json:"from_device"`
	// Methods is a list of the verification methods supported by the sender.
	Methods []VerificationMethod `json:"methods"`
}

type KeyAgreementProtocol string

const (
	KeyAgreementProtocolCurve25519           KeyAgreementProtocol = "curve25519"
	KeyAgreementProtocolCurve25519HKDFSHA256 KeyAgreementProtocol = "curve25519-hkdf-sha256"
)

type VerificationHashMethod string

const VerificationHashMethodSHA256 VerificationHashMethod = "sha256"

type MACMethod string

const (
	MACMethodHKDFHMACSHA256   MACMethod = "hkdf-hmac-sha256"
	MACMethodHKDFHMACSHA256V2 MACMethod = "hkdf-hmac-sha256.v2"
)

type SASMethod string

const (
	SASMethodDecimal SASMethod = "decimal"
	SASMethodEmoji   SASMethod = "emoji"
)

// VerificationStartEventContent represents the content of an
// [m.key.verification.start] event (both the to-device and the in-room
// version) as described in [Section 11.12.2.1] of the Spec.
//
// This struct also contains the fields for an [m.key.verification.start] event
// using the [VerificationMethodSAS] method as described in [Section
// 11.12.2.2.2] and an [m.key.verification.start] using
// [VerificationMethodReciprocate] as described in [Section 11.12.2.4.2].
//
// [m.key.verification.start]: https://spec.matrix.org/v1.9/client-server-api/#mkeyverificationstart
// [Section 11.12.2.1]: https://spec.matrix.org/v1.9/client-server-api/#key-verification-framework
// [Section 11.12.2.2.2]: https://spec.matrix.org/v1.9/client-server-api/#verification-messages-specific-to-sas
// [Section 11.12.2.4.2]: https://spec.matrix.org/v1.9/client-server-api/#verification-messages-specific-to-qr-codes
type VerificationStartEventContent struct {
	ToDeviceVerificationEvent
	InRoomVerificationEvent

	// FromDevice is the device ID which is initiating the request.
	FromDevice id.DeviceID `json:"from_device"`
	// Method is the verification method to use.
	Method VerificationMethod `json:"method"`
	// NextMethod is an optional method to use to verify the other user's key.
	// Applicable when the method chosen only verifies one user’s key. This
	// field will never be present if the method verifies keys both ways.
	NextMethod VerificationMethod `json:"next_method,omitempty"`

	// Hashes are the hash methods the sending device understands. This field
	// is only applicable when the method is m.sas.v1.
	Hashes []VerificationHashMethod `json:"hashes,omitempty"`
	// KeyAgreementProtocols is the list of key agreement protocols the sending
	// device understands. This field is only applicable when the method is
	// m.sas.v1.
	KeyAgreementProtocols []KeyAgreementProtocol `json:"key_agreement_protocols,omitempty"`
	// MessageAuthenticationCodes is a list of the MAC methods that the sending
	// device understands. This field is only applicable when the method is
	// m.sas.v1.
	MessageAuthenticationCodes []MACMethod `json:"message_authentication_codes"`
	// ShortAuthenticationString is a list of SAS methods the sending device
	// (and the sending device's user) understands. This field is only
	// applicable when the method is m.sas.v1.
	ShortAuthenticationString []SASMethod `json:"short_authentication_string"`

	// Secret is the shared secret from the QR code. This field is only
	// applicable when the method is m.reciprocate.v1.
	Secret jsonbytes.UnpaddedBytes `json:"secret,omitempty"`
}

// VerificationDoneEventContent represents the content of an
// [m.key.verification.done] event (both the to-device and the in-room version)
// as described in [Section 11.12.2.1] of the Spec.
//
// This type is an alias for [VerificationRelatable] since there are no
// additional fields defined by the spec.
//
// [m.key.verification.done]: https://spec.matrix.org/v1.9/client-server-api/#mkeyverificationdone
// [Section 11.12.2.1]: https://spec.matrix.org/v1.9/client-server-api/#mkeyverificationdone
type VerificationDoneEventContent struct {
	ToDeviceVerificationEvent
	InRoomVerificationEvent
}

type VerificationCancelCode string

const (
	VerificationCancelCodeUser               VerificationCancelCode = "m.user"
	VerificationCancelCodeTimeout            VerificationCancelCode = "m.timeout"
	VerificationCancelCodeUnknownTransaction VerificationCancelCode = "m.unknown_transaction"
	VerificationCancelCodeUnknownMethod      VerificationCancelCode = "m.unknown_method"
	VerificationCancelCodeUnexpectedMessage  VerificationCancelCode = "m.unexpected_message"
	VerificationCancelCodeKeyMismatch        VerificationCancelCode = "m.key_mismatch"
	VerificationCancelCodeUserMismatch       VerificationCancelCode = "m.user_mismatch"
	VerificationCancelCodeInvalidMessage     VerificationCancelCode = "m.invalid_message"
	VerificationCancelCodeAccepted           VerificationCancelCode = "m.accepted"
	VerificationCancelCodeSASMismatch        VerificationCancelCode = "m.mismatched_sas"
	VerificationCancelCodeCommitmentMismatch VerificationCancelCode = "m.mismatched_commitment"

	// Non-spec codes
	VerificationCancelCodeInternalError       VerificationCancelCode = "com.beeper.internal_error"
	VerificationCancelCodeMasterKeyNotTrusted VerificationCancelCode = "com.beeper.master_key_not_trusted" // the master key is not trusted by this device, but the QR code that was scanned was from a device that doesn't trust the master key
)

// VerificationCancelEventContent represents the content of an
// [m.key.verification.cancel] event (both the to-device and the in-room
// version) as described in [Section 11.12.2.1] of the Spec.
//
// [m.key.verification.cancel]: https://spec.matrix.org/v1.9/client-server-api/#mkeyverificationcancel
// [Section 11.12.2.1]: https://spec.matrix.org/v1.9/client-server-api/#mkeyverificationdone
type VerificationCancelEventContent struct {
	ToDeviceVerificationEvent
	InRoomVerificationEvent

	// Code is the error code for why the process/request was cancelled by the
	// user.
	Code VerificationCancelCode `json:"code"`
	// Reason is a human readable description of the code. The client should
	// only rely on this string if it does not understand the code.
	Reason string `json:"reason"`
}

// VerificationAcceptEventContent represents the content of an
// [m.key.verification.accept] event (both the to-device and the in-room
// version) as described in [Section 11.12.2.2.2] of the Spec.
//
// [m.key.verification.accept]: https://spec.matrix.org/v1.9/client-server-api/#mkeyverificationaccept
// [Section 11.12.2.2.2]: https://spec.matrix.org/v1.9/client-server-api/#verification-messages-specific-to-sas
type VerificationAcceptEventContent struct {
	ToDeviceVerificationEvent
	InRoomVerificationEvent

	// Commitment is the hash of the concatenation of the device's ephemeral
	// public key (encoded as unpadded base64) and the canonical JSON
	// representation of the m.key.verification.start message.
	Commitment jsonbytes.UnpaddedBytes `json:"commitment"`
	// Hash is the hash method the device is choosing to use, out of the
	// options in the m.key.verification.start message.
	Hash VerificationHashMethod `json:"hash"`
	// KeyAgreementProtocol is the key agreement protocol the device is
	// choosing to use, out of the options in the m.key.verification.start
	// message.
	KeyAgreementProtocol KeyAgreementProtocol `json:"key_agreement_protocol"`
	// MessageAuthenticationCode is the message authentication code the device
	// is choosing to use, out of the options in the m.key.verification.start
	// message.
	MessageAuthenticationCode MACMethod `json:"message_authentication_code"`
	// ShortAuthenticationString is a list of SAS methods both devices involved
	// in the verification process understand. Must be a subset of the options
	// in the m.key.verification.start message.
	ShortAuthenticationString []SASMethod `json:"short_authentication_string"`
}

// VerificationKeyEventContent represents the content of an
// [m.key.verification.key] event (both the to-device and the in-room version)
// as described in [Section 11.12.2.2.2] of the Spec.
//
// [m.key.verification.key]: https://spec.matrix.org/v1.9/client-server-api/#mkeyverificationkey
// [Section 11.12.2.2.2]: https://spec.matrix.org/v1.9/client-server-api/#verification-messages-specific-to-sas
type VerificationKeyEventContent struct {
	ToDeviceVerificationEvent
	InRoomVerificationEvent

	// Key is the device’s ephemeral public key.
	Key jsonbytes.UnpaddedBytes `json:"key"`
}

// VerificationMACEventContent represents the content of an
// [m.key.verification.mac] event (both the to-device and the in-room version)
// as described in [Section 11.12.2.2.2] of the Spec.
//
// [m.key.verification.mac]: https://spec.matrix.org/v1.9/client-server-api/#mkeyverificationmac
// [Section 11.12.2.2.2]: https://spec.matrix.org/v1.9/client-server-api/#verification-messages-specific-to-sas
type VerificationMACEventContent struct {
	ToDeviceVerificationEvent
	InRoomVerificationEvent

	// Keys is the MAC of the comma-separated, sorted, list of key IDs given in
	// the MAC property.
	Keys jsonbytes.UnpaddedBytes `json:"keys"`
	// MAC is a map of the key ID to the MAC of the key, using the algorithm in
	// the verification process.
	MAC map[id.KeyID]jsonbytes.UnpaddedBytes `json:"mac"`
}
