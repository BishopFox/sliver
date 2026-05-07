// Copyright (c) 2020 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package id

import (
	"encoding/base64"
	"fmt"
	"strings"

	"go.mau.fi/util/random"
)

// OlmMsgType is an Olm message type
type OlmMsgType int

const (
	OlmMsgTypePreKey OlmMsgType = 0
	OlmMsgTypeMsg    OlmMsgType = 1
)

// Algorithm is a Matrix message encryption algorithm.
// https://spec.matrix.org/v1.2/client-server-api/#messaging-algorithm-names
type Algorithm string

const (
	AlgorithmOlmV1    Algorithm = "m.olm.v1.curve25519-aes-sha2"
	AlgorithmMegolmV1 Algorithm = "m.megolm.v1.aes-sha2"
)

type KeyAlgorithm string

const (
	KeyAlgorithmCurve25519       KeyAlgorithm = "curve25519"
	KeyAlgorithmEd25519          KeyAlgorithm = "ed25519"
	KeyAlgorithmSignedCurve25519 KeyAlgorithm = "signed_curve25519"
)

type CrossSigningUsage string

const (
	XSUsageMaster      CrossSigningUsage = "master"
	XSUsageSelfSigning CrossSigningUsage = "self_signing"
	XSUsageUserSigning CrossSigningUsage = "user_signing"
)

type KeyBackupAlgorithm string

const (
	KeyBackupAlgorithmMegolmBackupV1 KeyBackupAlgorithm = "m.megolm_backup.v1.curve25519-aes-sha2"
)

// BackupVersion is an arbitrary string that identifies a server side key backup.
type KeyBackupVersion string

func (version KeyBackupVersion) String() string {
	return string(version)
}

// A SessionID is an arbitrary string that identifies an Olm or Megolm session.
type SessionID string

func (sessionID SessionID) String() string {
	return string(sessionID)
}

// Ed25519 is the base64 representation of an Ed25519 public key
type Ed25519 string
type SigningKey = Ed25519

func (ed25519 Ed25519) String() string {
	return string(ed25519)
}

func (ed25519 Ed25519) Bytes() []byte {
	val, _ := base64.RawStdEncoding.DecodeString(string(ed25519))
	// TODO handle errors
	return val
}

func (ed25519 Ed25519) Fingerprint() string {
	spacedSigningKey := make([]byte, len(ed25519)+(len(ed25519)-1)/4)
	var ptr = 0
	for i, chr := range ed25519 {
		spacedSigningKey[ptr] = byte(chr)
		ptr++
		if i%4 == 3 {
			spacedSigningKey[ptr] = ' '
			ptr++
		}
	}
	return string(spacedSigningKey)
}

// Curve25519 is the base64 representation of an Curve25519 public key
type Curve25519 string
type SenderKey = Curve25519
type IdentityKey = Curve25519

func (curve25519 Curve25519) String() string {
	return string(curve25519)
}

func (curve25519 Curve25519) Bytes() []byte {
	val, _ := base64.RawStdEncoding.DecodeString(string(curve25519))
	// TODO handle errors
	return val
}

// A DeviceID is an arbitrary string that references a specific device.
type DeviceID string

func (deviceID DeviceID) String() string {
	return string(deviceID)
}

// A DeviceKeyID is a string formatted as <algorithm>:<device_id> that is used as the key in deviceid-key mappings.
type DeviceKeyID string

func NewDeviceKeyID(algorithm KeyAlgorithm, deviceID DeviceID) DeviceKeyID {
	return DeviceKeyID(fmt.Sprintf("%s:%s", algorithm, deviceID))
}

func (deviceKeyID DeviceKeyID) String() string {
	return string(deviceKeyID)
}

func (deviceKeyID DeviceKeyID) Parse() (Algorithm, DeviceID) {
	index := strings.IndexRune(string(deviceKeyID), ':')
	if index < 0 || len(deviceKeyID) <= index+1 {
		return "", ""
	}
	return Algorithm(deviceKeyID[:index]), DeviceID(deviceKeyID[index+1:])
}

// A KeyID a string formatted as <keyalgorithm>:<key_id> that is used as the key in one-time-key mappings.
type KeyID string

func NewKeyID(algorithm KeyAlgorithm, keyID string) KeyID {
	return KeyID(fmt.Sprintf("%s:%s", algorithm, keyID))
}

func (keyID KeyID) String() string {
	return string(keyID)
}

func (keyID KeyID) Parse() (KeyAlgorithm, string) {
	index := strings.IndexRune(string(keyID), ':')
	if index < 0 || len(keyID) <= index+1 {
		return "", ""
	}
	return KeyAlgorithm(keyID[:index]), string(keyID[index+1:])
}

// Device contains the identity details of a device and some additional info.
type Device struct {
	UserID      UserID
	DeviceID    DeviceID
	IdentityKey Curve25519
	SigningKey  Ed25519

	Trust   TrustState
	Deleted bool
	Name    string
}

func (device *Device) Fingerprint() string {
	return device.SigningKey.Fingerprint()
}

type CrossSigningKey struct {
	Key   Ed25519
	First Ed25519
}

// Secret storage keys
type Secret string

func (s Secret) String() string {
	return string(s)
}

const (
	SecretXSMaster       Secret = "m.cross_signing.master"
	SecretXSSelfSigning  Secret = "m.cross_signing.self_signing"
	SecretXSUserSigning  Secret = "m.cross_signing.user_signing"
	SecretMegolmBackupV1 Secret = "m.megolm_backup.v1"
)

// VerificationTransactionID is a unique identifier for a verification
// transaction.
type VerificationTransactionID string

func NewVerificationTransactionID() VerificationTransactionID {
	return VerificationTransactionID(random.String(32))
}

func (t VerificationTransactionID) String() string {
	return string(t)
}
