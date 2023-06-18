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
	"crypto/sha256"
	"errors"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

var (
	// PeerAgePublicKey - The implant's age public key
	PeerAgePublicKey = "{{.Config.ECCPublicKey}}"
	// peerPrivateKey - The implant's age private key
	peerAgePrivateKey = "{{.Config.ECCPrivateKey}}"
	// PublicKeySignature - The implant's age public key minisigned'd
	PeerAgePublicKeySignature = `{{.Config.ECCPublicKeySignature}}`
	// serverPublicKey - Server's ECC public key
	serverAgePublicKey = "{{.Config.ECCServerPublicKey}}"
	// serverMinisignPublicKey - The server's minisign public key
	serverMinisignPublicKey = `{{.Config.MinisignServerPublicKey}}`

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

// GetAgeKeyPair - Get the implant's key pair
func GetAgeKeyPair() *AgeKeyPair {
	return &AgeKeyPair{
		Public:  PeerAgePublicKey,
		Private: peerAgePrivateKey,
	}
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

// GetServerAgePublicKey - Get the decoded server public key
func GetServerAgePublicKey() string {
	return serverAgePublicKey
}

// AgeEncryptToServer - Encrypt using the server's public key
func AgeEncryptToServer(plaintext []byte) ([]byte, error) {
	recipientPublicKey := GetServerAgePublicKey()
	if recipientPublicKey == "" {
		panic("no server public key")
	}
	keyPair := GetAgeKeyPair()
	ciphertext, err := AgeEncrypt(recipientPublicKey, plaintext)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256([]byte(keyPair.Public))
	msg := make([]byte, 32+len(ciphertext))
	copy(msg, digest[:])
	copy(msg[32:], ciphertext)
	return msg, nil
}

// AgeEncryptToPeer - Encrypt using the peer's public key
func AgeEncryptToPeer(recipientPublicKey []byte, recipientPublicKeySig string, plaintext []byte) ([]byte, error) {
	valid := MinisignVerify(recipientPublicKey, recipientPublicKeySig)
	if !valid {
		return nil, ErrInvalidPeerKey
	}
	var peerPublicKey [32]byte
	copy(peerPublicKey[:], recipientPublicKey)
	keyPair := GetAgeKeyPair()
	ciphertext, err := AgeEncrypt(keyPair.Public, plaintext)
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
	keyPair := GetAgeKeyPair()
	plaintext, err := AgeDecrypt(keyPair.Private, ciphertext)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
