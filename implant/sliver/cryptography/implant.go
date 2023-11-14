package cryptography

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"crypto/hmac"
	"crypto/sha256"
	"errors"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

var (
	// PeerAgePublicKey - The implant's age public key
	PeerAgePublicKey = "{{.Build.PeerPublicKey}}"
	// peerPrivateKey - The implant's age private key
	peerAgePrivateKey = "{{.Build.PeerPrivateKey}}"
	// PublicKeySignature - The implant's age public key minisigned'd
	PeerAgePublicKeySignature = `{{.Build.PeerPublicKeySignature}}`
	// serverPublicKey - Server's ECC public key
	serverAgePublicKey = "{{.Build.AgeServerPublicKey}}"
	// serverMinisignPublicKey - The server's minisign public key
	serverMinisignPublicKey = `{{.Build.MinisignServerPublicKey}}`

	// ErrInvalidPeerKey - Peer to peer key exchange failed
	ErrInvalidPeerKey = errors.New("invalid peer key")
)

// {{if .Config.Debug}} - Used for unit tests, remove from normal builds where these values are set at compile-time
func SetSecrets(peerPublicKey, peerPrivateKey, peerPublicKeySignature, serverPublicKey, minisignServerPublicKey string) {
	PeerAgePublicKey = peerPublicKey
	peerAgePrivateKey = peerPrivateKey
	PeerAgePublicKeySignature = peerPublicKeySignature
	serverAgePublicKey = serverPublicKey
	serverMinisignPublicKey = minisignServerPublicKey
}

// {{end}}

// GetPeerAgeKeyPair - Get the implant's key pair
func GetPeerAgeKeyPair() *AgeKeyPair {
	return &AgeKeyPair{
		Public:  PeerAgePublicKey,
		Private: peerAgePrivateKey,
	}
}

// GetServerAgePublicKey - Get the decoded server public key
func GetServerAgePublicKey() string {
	return serverAgePublicKey
}

// MinisignVerify - Verify a minisign signature
func MinisignVerify(message []byte, signature string) bool {
	serverPublicKey, err := DecodeMinisignPublicKey(serverMinisignPublicKey)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to decode minisign public key: %s", err)
		// {{end}}
		return false
	}
	sig, err := DecodeMinisignSignature(signature)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to decode minisign signature: %s", err)
		// {{end}}
		return false
	}
	valid, err := serverPublicKey.Verify(message, sig)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("minisign signature validation error: %s", err)
		// {{end}}
		return false
	}
	// {{if .Config.Debug}}
	log.Printf("minisign signature validation: %v", valid)
	// {{end}}
	return valid
}

// AgeKeyExToServer - Encrypt using the server's public key
func AgeKeyExToServer(plaintext []byte) ([]byte, error) {
	recipientPublicKey := GetServerAgePublicKey()
	if recipientPublicKey == "" {
		panic("no server public key")
	}

	peerKeyPair := GetPeerAgeKeyPair()

	// First HMAC the plaintext with the hash of the implant's private key
	// this ensures that the server is talking to a valid implant
	privateDigest := sha256.New()
	privateDigest.Write([]byte(peerKeyPair.Private))
	mac := hmac.New(sha256.New, privateDigest.Sum(nil))
	mac.Write(plaintext)

	// Next encrypt using server's Age public key
	ciphertext, err := AgeEncrypt(recipientPublicKey, append(mac.Sum(nil), plaintext...))
	if err != nil {
		return nil, err
	}

	// Sender includes hash of it's implant specific peer public key
	publicDigest := sha256.Sum256([]byte(peerKeyPair.Public))
	msg := make([]byte, 32+len(ciphertext))
	copy(msg, publicDigest[:])
	copy(msg[32:], ciphertext)
	return msg, nil
}

// AgeEncryptToPeer - Encrypt using the peer's public key
func AgeEncryptToPeer(recipientPublicKey []byte, recipientPublicKeySig string, plaintext []byte) ([]byte, error) {
	valid := MinisignVerify(recipientPublicKey, recipientPublicKeySig)
	if !valid {
		return nil, ErrInvalidPeerKey
	}
	ciphertext, err := AgeEncrypt(string(recipientPublicKey), plaintext)
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}

// AgeDecryptFromPeer - Decrypt a message from a peer
func AgeDecryptFromPeer(senderPublicKey []byte, senderPublicKeySig string, ciphertext []byte) ([]byte, error) {
	valid := MinisignVerify(senderPublicKey, senderPublicKeySig)
	if !valid {
		return nil, ErrInvalidPeerKey
	}
	var peerPublicKey [32]byte
	copy(peerPublicKey[:], senderPublicKey)
	keyPair := GetPeerAgeKeyPair()
	plaintext, err := AgeDecrypt(keyPair.Private, ciphertext)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
