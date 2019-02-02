package main

/*
This package contains wrappers around Golang's crypto package that make it easier to use
we manage things like the nonces/iv's. The preferred choice is to always use GCM but for
saving space we also have CTR mode available but it does not provide integrity checks.
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

func (AESKey) FromBytes(data []byte) AESKey {
	var key AESKey
	copy(key[:], data[:AESKeySize])
	return key
}

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

// CTREncrypt - AES CTR Encrypt
func CTREncrypt(key AESKey, plaintext []byte) []byte {
	block, _ := aes.NewCipher(key[:])
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := RandomAESIV()
	copy(ciphertext[:aes.BlockSize], iv[:])
	stream := cipher.NewCTR(block, iv[:])
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	return ciphertext
}

// CTRDecrypt - AES CTR Decrypt
func CTRDecrypt(key AESKey, ciphertext []byte) []byte {
	plaintext := make([]byte, len(ciphertext)-aes.BlockSize)
	block, _ := aes.NewCipher(key[:])
	iv := ciphertext[:aes.BlockSize]
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(plaintext, ciphertext[aes.BlockSize:])
	return plaintext
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
	if len(ciphertext) < GCMNonceSize+1 {
		return nil, errors.New("Invalid ciphertext length")
	}
	block, _ := aes.NewCipher(key[:])
	aesgcm, _ := cipher.NewGCM(block)
	plaintext, err := aesgcm.Open(nil, ciphertext[:GCMNonceSize], ciphertext[GCMNonceSize:], nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
