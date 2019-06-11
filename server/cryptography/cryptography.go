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

	---
	This package contains wrappers around Golang's crypto package that make it easier to use
	we manage things like the nonces/iv's
*/

import (
	"crypto/aes"
	"crypto/cipher"
	secureRand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"io"
)

const (
	// AESKeySize - Always use 256 bit keys
	AESKeySize = 16

	// GCMNonceSize - 96 bit nonces for GCM
	GCMNonceSize = 12
)

// AESKey - 128 bit key
type AESKey [AESKeySize]byte

// AESIV - 128 bit IV
type AESIV [aes.BlockSize]byte

// RandomAESKey - Generate random ID of randomIDSize bytes
func RandomAESKey() AESKey {
	randBuf := make([]byte, 64)
	secureRand.Read(randBuf)
	digest := sha256.Sum256(randBuf)
	var key AESKey
	copy(key[:], digest[:AESKeySize])
	return key
}

// AESKeyFromBytes - Convert byte slice to AESKey
func AESKeyFromBytes(data []byte) (AESKey, error) {
	if len(data) != AESKeySize {
		return AESKey{}, errors.New("Invalid length")
	}
	var key AESKey
	copy(key[:], data[:AESKeySize])
	return key, nil
}

// RandomAESIV - 128 bit Random IV
func RandomAESIV() AESIV {
	data := RandomAESKey()
	var iv AESIV
	copy(iv[:], data[:16])
	return iv
}

// RSAEncrypt - Encrypt a msg with a public rsa key
func RSAEncrypt(msg []byte, pub *rsa.PublicKey) ([]byte, error) {
	hash := sha256.New()
	ciphertext, err := rsa.EncryptOAEP(hash, secureRand.Reader, pub, msg, nil)
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}

// RSADecrypt - Decrypt ciphertext with rsa private key
func RSADecrypt(ciphertext []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	hash := sha256.New()
	plaintext, err := rsa.DecryptOAEP(hash, secureRand.Reader, privateKey, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// GCMEncrypt - Encrypt using AES GCM
func GCMEncrypt(key AESKey, plaintext []byte) ([]byte, error) {
	block, _ := aes.NewCipher(key[:])
	nonce := make([]byte, GCMNonceSize)
	if _, err := io.ReadFull(secureRand.Reader, nonce); err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)

	// Prepend nonce to ciphertext
	ciphertext = append(nonce, ciphertext...)
	return ciphertext, nil
}

// GCMDecrypt - Decrypt GCM ciphertext
func GCMDecrypt(key AESKey, ciphertext []byte) ([]byte, error) {
	block, _ := aes.NewCipher(key[:])
	aesgcm, _ := cipher.NewGCM(block)
	plaintext, err := aesgcm.Open(nil, ciphertext[:GCMNonceSize], ciphertext[GCMNonceSize:], nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
