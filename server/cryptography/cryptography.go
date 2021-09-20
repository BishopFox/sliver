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
	This package contains wrappers around Golang's crypto package that make
	it easier to use we manage things like the nonces/iv's
*/

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	secureRand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"time"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	// AESKeySize - Always use 256 bit keys
	AESKeySize = 16

	// GCMNonceSize - 96 bit nonces for GCM
	GCMNonceSize = 12

	// TOTPDigits - Number of digits in the TOTP
	TOTPDigits = 8
	TOTPPeriod = uint(30)
)

var (
	// ErrInvalidKeyLength - Invalid key length
	ErrInvalidKeyLength = errors.New("invalid length")

	// ErrReplayAttack - Replay attack
	ErrReplayAttack = errors.New("replay attack detected")
)

// AESKey - 256 bit key
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

// RandomKey - Generate random ID of randomIDSize bytes
func RandomKey() [chacha20poly1305.KeySize]byte {
	randBuf := make([]byte, 64)
	secureRand.Read(randBuf)
	digest := sha256.Sum256(randBuf)
	var key [chacha20poly1305.KeySize]byte
	copy(key[:], digest[:chacha20poly1305.KeySize])
	return key
}

// AESKeyFromBytes - Convert byte slice to AESKey
func AESKeyFromBytes(data []byte) (AESKey, error) {
	if len(data) != AESKeySize {
		return AESKey{}, ErrInvalidKeyLength
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

// Encrypt - Encrypt using chacha20poly1305
// https://pkg.go.dev/golang.org/x/crypto/chacha20poly1305
func Encrypt(key [chacha20poly1305.KeySize]byte, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize(), aead.NonceSize()+len(plaintext)+aead.Overhead())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt - Decrypt using chacha20poly1305
// https://pkg.go.dev/golang.org/x/crypto/chacha20poly1305
func Decrypt(key [chacha20poly1305.KeySize]byte, ciphertext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < aead.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	// Split nonce and ciphertext.
	nonce, ciphertext := ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():]

	// Decrypt the message and check it wasn't tampered with.
	return aead.Open(nil, nonce, ciphertext, nil)
}

// AESEncrypt - Encrypt using AES GCM
func AESEncrypt(key AESKey, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
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

// AESDecrypt - Decrypt GCM ciphertext
func AESDecrypt(key AESKey, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := aesgcm.Open(nil, ciphertext[:GCMNonceSize], ciphertext[GCMNonceSize:], nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// NewCipherContext - Wrapper around creating a cipher context from a key
func NewCipherContext(key AESKey) *CipherContext {
	return &CipherContext{
		Key:    key,
		replay: &sync.Map{},
	}
}

// CipherContext - Tracks a series of messages encrypted under the same key
// and detects/prevents replay attacks.
type CipherContext struct {
	Key    AESKey
	replay *sync.Map
}

// Decrypt - Decrypt a message with the contextual key and check for replay attacks
func (c *CipherContext) Decrypt(data []byte) ([]byte, error) {
	digest := sha256.New()
	digest.Write(data)
	if _, ok := c.replay.LoadOrStore(digest.Sum(nil), true); ok {
		return nil, ErrReplayAttack
	}
	plaintext, err := AESDecrypt(c.Key, data)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// Encrypt - Encrypt a message with the contextual key
func (c *CipherContext) Encrypt(plaintext []byte) ([]byte, error) {
	ciphertext, err := AESEncrypt(c.Key, plaintext)
	if err != nil {
		return nil, err
	}
	digest := sha256.New()
	digest.Write(ciphertext)
	c.replay.Store(digest.Sum(nil), true)
	return ciphertext, nil
}

// TOTPOptions - Customized totp validation options
func TOTPOptions() totp.ValidateOpts {
	return totp.ValidateOpts{
		Digits:    TOTPDigits,
		Algorithm: otp.AlgorithmSHA256,
		Period:    TOTPPeriod,
		Skew:      uint(1),
	}
}

// TOTPServerSecret - Get the server-wide totp secret value, the goal of the totp
// is for the implant to prove it was generated by this server. To that end we simply
// use a server-wide secret and ignore issuers/accounts. In order to bypass this check
// you'd have to extract the totp secret from a binary generated by the server.
func TOTPServerSecret() (string, error) {
	totpSecretPath := path.Join(assets.GetRootAppDir(), "totp.secret")
	if _, err := os.Stat(totpSecretPath); os.IsNotExist(err) {
		return totpGenerateSecret(totpSecretPath)
	} else {
		data, err := ioutil.ReadFile(totpSecretPath)
		return string(data), err
	}
}

// ValidateTOTP - Validate a TOTP code
func ValidateTOTP(code string) (bool, error) {
	secret, err := TOTPServerSecret()
	if err != nil {
		return false, err
	}
	valid, err := totp.ValidateCustom(code, secret, time.Now().UTC(), TOTPOptions())
	if err != nil {
		return false, err
	}
	return valid, nil
}

func totpGenerateSecret(saveTo string) (string, error) {
	otpSecret, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "foo",
		AccountName: "bar",
		Digits:      TOTPDigits,
		Algorithm:   otp.AlgorithmSHA256,
		Period:      TOTPPeriod,
	})
	if err != nil {
		return "", err
	}
	secret := otpSecret.Secret()
	err = ioutil.WriteFile(saveTo, []byte(secret), 0600)
	return secret, err
}
