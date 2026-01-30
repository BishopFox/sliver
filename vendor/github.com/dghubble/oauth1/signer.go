package oauth1

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"hash"
	"strings"
)

// A Signer signs messages to create signed OAuth1 Requests.
type Signer interface {
	// Name returns the name of the signing method.
	Name() string
	// Sign signs the message using the given secret key.
	Sign(key string, message string) (string, error)
}

// HMACSigner signs messages with an HMAC SHA1 digest, using the concatenated
// consumer secret and token secret as the key.
type HMACSigner struct {
	ConsumerSecret string
}

// Name returns the HMAC-SHA1 method.
func (s *HMACSigner) Name() string {
	return "HMAC-SHA1"
}

func hmacSign(consumerSecret, tokenSecret, message string, algo func() hash.Hash) (string, error) {
	signingKey := strings.Join([]string{PercentEncode(consumerSecret), PercentEncode(tokenSecret)}, "&")
	mac := hmac.New(algo, []byte(signingKey))
	mac.Write([]byte(message))
	signatureBytes := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(signatureBytes), nil
}

// Sign creates a concatenated consumer and token secret key and calculates
// the HMAC digest of the message. Returns the base64 encoded digest bytes.
func (s *HMACSigner) Sign(tokenSecret, message string) (string, error) {
	return hmacSign(s.ConsumerSecret, tokenSecret, message, sha1.New)
}

// HMAC256Signer signs messages with an HMAC SHA256 digest, using the concatenated
// consumer secret and token secret as the key.
type HMAC256Signer struct {
	ConsumerSecret string
}

// Name returns the HMAC-SHA256 method.
func (s *HMAC256Signer) Name() string {
	return "HMAC-SHA256"
}

// Sign creates a concatenated consumer and token secret key and calculates
// the HMAC digest of the message. Returns the base64 encoded digest bytes.
func (s *HMAC256Signer) Sign(tokenSecret, message string) (string, error) {
	return hmacSign(s.ConsumerSecret, tokenSecret, message, sha256.New)
}

// RSASigner RSA PKCS1-v1_5 signs SHA1 digests of messages using the given
// RSA private key.
type RSASigner struct {
	PrivateKey *rsa.PrivateKey
}

// Name returns the RSA-SHA1 method.
func (s *RSASigner) Name() string {
	return "RSA-SHA1"
}

// Sign uses RSA PKCS1-v1_5 to sign a SHA1 digest of the given message. The
// tokenSecret is not used with this signing scheme.
func (s *RSASigner) Sign(tokenSecret, message string) (string, error) {
	digest := sha1.Sum([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.PrivateKey, crypto.SHA1, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}
