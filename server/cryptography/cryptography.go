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
	------------------------------------------------------------------------

	This package contains wrappers around Golang's crypto package that make
	it easier to use we manage things like the nonces/iv's

*/

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/bishopfox/sliver/server/cryptography/minisign"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/nacl/box"
)

const (
	// TOTPDigits - Number of digits in the TOTP
	TOTPDigits               = 8
	TOTPPeriod               = uint(30)
	TOTPSecretKey            = "server.totp"
	ServerECCKeyPairKey      = "server.ecc"
	serverMinisignPrivateKey = "server.minisign"
)

var (
	// ErrInvalidKeyLength - Invalid key length
	ErrInvalidKeyLength = errors.New("invalid length")

	// ErrReplayAttack - Replay attack
	ErrReplayAttack = errors.New("replay attack detected")

	// ErrDecryptFailed
	ErrDecryptFailed = errors.New("decryption failed")
)

// deriveKeyFrom - Derives a key from input data using SHA256
func deriveKeyFrom(data []byte) [chacha20poly1305.KeySize]byte {
	digest := sha256.Sum256(data)
	var key [chacha20poly1305.KeySize]byte
	copy(key[:], digest[:chacha20poly1305.KeySize])
	return key
}

// RandomKey - Generate random ID of randomIDSize bytes
func RandomKey() [chacha20poly1305.KeySize]byte {
	randBuf := make([]byte, 64)
	rand.Read(randBuf)
	return deriveKeyFrom(randBuf)
}

// KeyFromBytes - Convert to fixed length buffer
func KeyFromBytes(data []byte) ([chacha20poly1305.KeySize]byte, error) {
	var key [chacha20poly1305.KeySize]byte
	if len(data) != chacha20poly1305.KeySize {
		// We cannot return nil due to the fixed length buffer type ...
		// and it seems like a really bad idea to return a zero key in case
		// the error is not checked by the caller, so instead we return a
		// random key, which should break everything if the error is not checked.
		return RandomKey(), ErrInvalidKeyLength
	}
	copy(key[:], data)
	return key, nil
}

// ECCKeyPair - Holds the public/private key pair
type ECCKeyPair struct {
	Public  *[32]byte `json:"public"`
	Private *[32]byte `json:"private"`
}

// PublicBase64 - Base64 encoded public key
func (e *ECCKeyPair) PublicBase64() string {
	return base64.RawStdEncoding.EncodeToString(e.Public[:])
}

// PrivateBase64 - Base64 encoded private key
func (e *ECCKeyPair) PrivateBase64() string {
	return base64.RawStdEncoding.EncodeToString(e.Private[:])
}

// RandomECCKeyPair - Generate a random Curve 25519 key pair
func RandomECCKeyPair() (*ECCKeyPair, error) {
	public, private, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &ECCKeyPair{Public: public, Private: private}, nil
}

// ECCEncrypt - Encrypt using Nacl Box
func ECCEncrypt(recipientPublicKey *[32]byte, senderPrivateKey *[32]byte, plaintext []byte) ([]byte, error) {
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, err
	}
	encrypted := box.Seal(nonce[:], plaintext, &nonce, recipientPublicKey, senderPrivateKey)
	return encrypted, nil
}

// ECCDecrypt - Decrypt using Curve 25519 + ChaCha20Poly1305
func ECCDecrypt(senderPublicKey *[32]byte, recipientPrivateKey *[32]byte, ciphertext []byte) ([]byte, error) {
	var decryptNonce [24]byte
	copy(decryptNonce[:], ciphertext[:24])
	plaintext, ok := box.Open(nil, ciphertext[24:], &decryptNonce, senderPublicKey, recipientPrivateKey)
	if !ok {
		return nil, ErrDecryptFailed
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

// TOTPOptions - Customized totp validation options
func TOTPOptions() totp.ValidateOpts {
	return totp.ValidateOpts{
		Digits:    TOTPDigits,
		Algorithm: otp.AlgorithmSHA256,
		Period:    TOTPPeriod,
		Skew:      uint(1),
	}
}

// ECCServerKeyPair - Get teh server's ECC key pair
func ECCServerKeyPair() *ECCKeyPair {
	data, err := db.GetKeyValue(ServerECCKeyPairKey)
	if err == db.ErrRecordNotFound {
		keyPair, err := generateServerECCKeyPair()
		if err != nil {
			panic(err)
		}
		return keyPair
	}
	keyPair := &ECCKeyPair{}
	err = json.Unmarshal([]byte(data), keyPair)
	if err != nil {
		panic(err)
	}
	return keyPair

}

func generateServerECCKeyPair() (*ECCKeyPair, error) {
	keyPair, err := RandomECCKeyPair()
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(keyPair)
	if err != nil {
		return nil, err
	}
	err = db.SetKeyValue(ServerECCKeyPairKey, string(data))
	return keyPair, err
}

// TOTPServerSecret - Get the server-wide totp secret value, the goal of the totp
// is for the implant to prove it was generated by this server. To that end we simply
// use a server-wide secret and ignore issuers/accounts. In order to bypass this check
// you'd have to extract the totp secret from a binary generated by the server.
func TOTPServerSecret() (string, error) {
	secret, err := db.GetKeyValue(TOTPSecretKey)
	if err == db.ErrRecordNotFound {
		secret, err = totpGenerateSecret()
	}
	return secret, err
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

func totpGenerateSecret() (string, error) {
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
	err = db.SetKeyValue(TOTPSecretKey, secret)
	return secret, err
}

// minisignPrivateKey - This is here so we can marshal to/from JSON
type minisignPrivateKey struct {
	ID         uint64 `json:"id"`
	PrivateKey []byte `json:"private_key"`
}

// MinisignServerPublicKey - Get the server's minisign public key string
func MinisignServerPublicKey() string {
	publicKey := MinisignServerPrivateKey().Public().(minisign.PublicKey)
	publicKeyText, err := publicKey.MarshalText()
	if err != nil {
		panic(err)
	}
	return string(publicKeyText)
}

// MinisignServerSign - Sign a message with the server's minisign private key
func MinisignServerSign(message []byte) string {
	privateKey := MinisignServerPrivateKey()
	return string(minisign.Sign(*privateKey, message))
}

// MinisignServerPrivateKey - Get the server's minisign key pair
func MinisignServerPrivateKey() *minisign.PrivateKey {
	data, err := db.GetKeyValue(serverMinisignPrivateKey)
	if err == db.ErrRecordNotFound {
		privateKey, err := generateServerMinisignPrivateKey()
		if err != nil {
			panic(err)
		}
		return privateKey
	}
	privateKey := &minisignPrivateKey{}
	err = json.Unmarshal([]byte(data), privateKey)
	if err != nil {
		panic(err)
	}
	rawBytes := [ed25519.PrivateKeySize]byte{}
	copy(rawBytes[:], privateKey.PrivateKey)
	return &minisign.PrivateKey{
		RawID:    privateKey.ID,
		RawBytes: rawBytes,
	}
}

func generateServerMinisignPrivateKey() (*minisign.PrivateKey, error) {
	_, privateKey, err := minisign.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(&minisignPrivateKey{
		ID:         privateKey.ID(),
		PrivateKey: privateKey.Bytes(),
	})
	err = db.SetKeyValue(serverMinisignPrivateKey, string(data))
	if err != nil {
		return nil, err
	}
	return &privateKey, err
}
