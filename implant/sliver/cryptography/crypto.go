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
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"filippo.io/age"
	"github.com/bishopfox/sliver/implant/sliver/encoders"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/chacha20poly1305"
)

var (
	totpOptions = totp.ValidateOpts{
		Digits:    8,
		Algorithm: otp.AlgorithmSHA256,
		Period:    uint(30),
		Skew:      uint(1),
	}

	// ErrReplayAttack - Replay attack
	ErrReplayAttack = errors.New("replay attack detected")
	// ErrDecryptFailed
	ErrDecryptFailed = errors.New("decryption failed")

	ageMsgPrefix        = []byte("age-encryption.org/v1\n-> X25519 ")
	agePublicKeyPrefix  = "age1"
	agePrivateKeyPrefix = "AGE-SECRET-KEY-1"
)

// AgeKeyPair - Holds the public/private key pair
type AgeKeyPair struct {
	Public  string
	Private string
}

// AgeEncrypt - Encrypt using Nacl Box
func AgeEncrypt(recipientPublicKey string, plaintext []byte) ([]byte, error) {
	if !strings.HasPrefix(recipientPublicKey, agePublicKeyPrefix) {
		recipientPublicKey = agePublicKeyPrefix + recipientPublicKey
	}
	recipient, err := age.ParseX25519Recipient(recipientPublicKey)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer([]byte{})
	stream, err := age.Encrypt(buf, recipient)
	if err != nil {
		return nil, err
	}
	if _, err := stream.Write(plaintext); err != nil {
		return nil, err
	}
	if err := stream.Close(); err != nil {
		return nil, err
	}
	return bytes.TrimPrefix(buf.Bytes(), ageMsgPrefix), nil
}

// AgeDecrypt - Decrypt using Curve 25519 + ChaCha20Poly1305
func AgeDecrypt(recipientPrivateKey string, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 24 {
		return nil, errors.New("ciphertext too short")
	}
	if !strings.HasPrefix(recipientPrivateKey, agePrivateKeyPrefix) {
		recipientPrivateKey = agePrivateKeyPrefix + recipientPrivateKey
	}
	identity, err := age.ParseX25519Identity(recipientPrivateKey)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(append(ageMsgPrefix, ciphertext...))
	stream, err := age.Decrypt(buf, identity)
	if err != nil {
		return nil, err
	}
	plaintext, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// RandomKey - Generate random ID of randomIDSize bytes
func RandomKey() [chacha20poly1305.KeySize]byte {
	randBuf := make([]byte, 64)
	rand.Read(randBuf)
	return deriveKeyFrom(randBuf)
}

// deriveKeyFrom - Derives a key from input data using SHA256
func deriveKeyFrom(data []byte) [chacha20poly1305.KeySize]byte {
	digest := sha256.Sum256(data)
	var key [chacha20poly1305.KeySize]byte
	copy(key[:], digest[:chacha20poly1305.KeySize])
	return key
}

// Encrypt - Encrypt using chacha20poly1305
// https://pkg.go.dev/golang.org/x/crypto/chacha20poly1305
func Encrypt(key [chacha20poly1305.KeySize]byte, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, err
	}
	plaintext = bytes.NewBuffer(encoders.GzipBuf(plaintext)).Bytes()
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
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return encoders.GunzipBuf(plaintext), nil
}

// NewCipherContext - Wrapper around creating a cipher context from a key
func NewCipherContext(key [chacha20poly1305.KeySize]byte) *CipherContext {
	return &CipherContext{
		Key:    key,
		replay: &sync.Map{},
	}
}

// CipherContext - Tracks a series of messages encrypted under the same key
// and detects/prevents replay attacks.
type CipherContext struct {
	Key    [chacha20poly1305.KeySize]byte
	replay *sync.Map
}

// Decrypt - Decrypt a message with the contextual key and check for replay attacks
func (c *CipherContext) Decrypt(ciphertext []byte) ([]byte, error) {
	plaintext, err := Decrypt(c.Key, ciphertext)
	if err != nil {
		return nil, err
	}
	if 0 < len(ciphertext) {
		digest := sha256.Sum256(ciphertext)
		b64Digest := base64.RawStdEncoding.EncodeToString(digest[:])
		if _, ok := c.replay.LoadOrStore(b64Digest, true); ok {
			return nil, ErrReplayAttack
		}
	}
	return plaintext, nil
}

// Encrypt - Encrypt a message with the contextual key
func (c *CipherContext) Encrypt(plaintext []byte) ([]byte, error) {
	ciphertext, err := Encrypt(c.Key, plaintext)
	if err != nil {
		return nil, err
	}
	if 0 < len(ciphertext) {
		digest := sha256.Sum256(ciphertext)
		b64Digest := base64.RawStdEncoding.EncodeToString(digest[:])
		c.replay.Store(b64Digest, true)
	}
	return ciphertext, nil
}

// GetExactOTPCode - Get the OTP code for a specific timestamp
func GetExactOTPCode(timestamp time.Time) string {
	code, _ := totp.GenerateCodeCustom(totpSecret, timestamp, totpOptions)
	// {{if .Config.Debug}}
	log.Printf("TOTP Code (%s): %s", timestamp, code)
	// {{end}}
	return code
}

// GetOTPCode - Get the current OTP code
func GetOTPCode() string {
	now := time.Now().UTC()
	code, _ := totp.GenerateCodeCustom(totpSecret, now, totpOptions)
	// {{if .Config.Debug}}
	log.Printf("TOTP Code: %s", code)
	// {{end}}
	return code
}

// ValidateTOTP - Validate a TOTP code
func ValidateTOTP(code string) (bool, error) {
	now := time.Now().UTC()
	valid, err := totp.ValidateCustom(code, totpSecret, now, totpOptions)
	if err != nil {
		return false, err
	}
	return valid, nil
}

// rootOnlyVerifyCertificate - Go doesn't provide a method for only skipping hostname validation so
// we have to disable all of the certificate validation and re-implement everything.
// https://github.com/golang/go/issues/21971
func RootOnlyVerifyCertificate(caCertPEM string, rawCerts [][]byte, _ [][]*x509.Certificate) error {
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(caCertPEM))
	if !ok {
		// {{if .Config.Debug}}
		log.Printf("Failed to parse root certificate")
		// {{end}}
		os.Exit(3)
	}

	cert, err := x509.ParseCertificate(rawCerts[0]) // We should only get one cert
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to parse certificate: " + err.Error())
		// {{end}}
		return err
	}

	// Basically we only care if the certificate was signed by our authority
	// Go selects sensible defaults for time and EKU, basically we're only
	// skipping the hostname check, I think?
	options := x509.VerifyOptions{
		Roots: roots,
	}
	if _, err := cert.Verify(options); err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to verify certificate: " + err.Error())
		// {{end}}
		return err
	}

	return nil
}

//// GzipBuf - Gzip a buffer
//func GzipBuf(data []byte) []byte {
//	var buf bytes.Buffer
//	zip := gzip.NewWriter(&buf)
//	zip.Write(data)
//	zip.Close()
//	return buf.Bytes()
//}
//
//// GunzipBuf - Gunzip a buffer
//func GunzipBuf(data []byte) []byte {
//	zip, _ := gzip.NewReader(bytes.NewBuffer(data))
//	var buf bytes.Buffer
//	buf.ReadFrom(zip)
//	return buf.Bytes()
//}
