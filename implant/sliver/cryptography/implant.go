package cryptography

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"crypto/sha256"
	"encoding/base64"
)

var (
	// ECCPublicKey - The implant's ECC public key
	eccPublicKey = "{{.Config.ECCPublicKey}}"
	// eccPrivateKey - The implant's ECC private key
	eccPrivateKey = "{{.Config.ECCPrivateKey}}"
	// eccPublicKeySignature - The implant's public key minisigned'd
	ECCPublicKeySignature = "{{.Config.ECCPublicKeySignature}}"
	// eccServerPublicKey - Server's ECC public key
	eccServerPublicKey = "{{.Config.ECCServerPublicKey}}"
	// minisignServerPublicKey - The server's minisign public key
	minisignServerPublicKey = "{{.Config.MinisignServerPublicKey}}"

	// TOTP secret value
	totpSecret = "{{.OTPSecret}}"
)

// {{if .Config.Debug}} - Used for unit tests, remove from normal builds where these values are set at compile-time
func SetSecrets(newEccPublicKey, newEccPrivateKey, newEccPublicKeySignature, newEccServerPublicKey, newTotpSecret string) {
	eccPublicKey = newEccPublicKey
	eccPrivateKey = newEccPrivateKey
	ECCPublicKeySignature = newEccPublicKeySignature
	eccServerPublicKey = newEccServerPublicKey
	totpSecret = newTotpSecret
}

// {{end}}

// GetECCKeyPair - Get the implant's key pair
func GetECCKeyPair() *ECCKeyPair {
	publicRaw, err := base64.RawStdEncoding.DecodeString(eccPublicKey)
	if err != nil {
		panic("no public key")
	}
	var public [32]byte
	copy(public[:], publicRaw)
	privateRaw, err := base64.RawStdEncoding.DecodeString(eccPrivateKey)
	if err != nil {
		panic("no private key")
	}
	var private [32]byte
	copy(private[:], privateRaw)
	return &ECCKeyPair{
		Public:  &public,
		Private: &private,
	}
}

// GetServerPublicKey - Get the decoded server public key
func GetServerPublicKey() *[32]byte {
	publicRaw, err := base64.RawStdEncoding.DecodeString(eccServerPublicKey)
	if err != nil {
		return nil
	}
	var public [32]byte
	copy(public[:], publicRaw)
	return &public
}

// ECCEncryptToServer - Encrypt using the server's public key
func ECCEncryptToServer(plaintext []byte) ([]byte, error) {
	recipientPublicKey := GetServerPublicKey()
	if recipientPublicKey == nil {
		panic("no server public key")
	}
	keyPair := GetECCKeyPair()
	ciphertext, err := ECCEncrypt(recipientPublicKey, keyPair.Private, plaintext)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256((*keyPair.Public)[:])
	msg := make([]byte, 32+len(ciphertext))
	copy(msg, digest[:])
	copy(msg[32:], ciphertext)
	return msg, nil
}
