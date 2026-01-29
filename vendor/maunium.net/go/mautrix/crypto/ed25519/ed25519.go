// Copyright 2024 Sumner Evans.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ed25519 implements the Ed25519 signature algorithm. See
// https://ed25519.cr.yp.to/.
//
// This package stores the private key in the NaCl format, which is a different
// format than that used by the [crypto/ed25519] package in the standard
// library.
//
// This picture will help with the rest of the explanation:
// https://blog.mozilla.org/warner/files/2011/11/key-formats.png
//
// The private key in the [crypto/ed25519] package is a 64-byte value where the
// first 32-bytes are the seed and the last 32-bytes are the public key.
//
// The private key in this package is stored in the NaCl format. That is, the
// left 32-bytes are the private scalar A and the right 32-bytes are the right
// half of the SHA512 result.
//
// The contents of this package are mostly copied from the standard library,
// and as such the source code is licensed under the BSD license of the
// standard library implementation.
//
// Other notable changes from the standard library include:
//
//   - The Seed function of the standard library is not implemented in this
//     package because there is no way to recover the seed after hashing it.
package ed25519

import (
	"crypto"
	"crypto/ed25519"
	cryptorand "crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"errors"
	"io"
	"strconv"

	"filippo.io/edwards25519"
)

const (
	// PublicKeySize is the size, in bytes, of public keys as used in this package.
	PublicKeySize = 32
	// PrivateKeySize is the size, in bytes, of private keys as used in this package.
	PrivateKeySize = 64
	// SignatureSize is the size, in bytes, of signatures generated and verified by this package.
	SignatureSize = 64
	// SeedSize is the size, in bytes, of private key seeds. These are the private key representations used by RFC 8032.
	SeedSize = 32
)

// PublicKey is the type of Ed25519 public keys.
type PublicKey []byte

// Any methods implemented on PublicKey might need to also be implemented on
// PrivateKey, as the latter embeds the former and will expose its methods.

// Equal reports whether pub and x have the same value.
func (pub PublicKey) Equal(x crypto.PublicKey) bool {
	switch x := x.(type) {
	case PublicKey:
		return subtle.ConstantTimeCompare(pub, x) == 1
	case ed25519.PublicKey:
		return subtle.ConstantTimeCompare(pub, x) == 1
	default:
		return false
	}
}

// PrivateKey is the type of Ed25519 private keys. It implements [crypto.Signer].
type PrivateKey []byte

// Public returns the [PublicKey] corresponding to priv.
//
// This method differs from the standard library because it calculates the
// public key instead of returning the right half of the private key (which
// contains the public key in the standard library).
func (priv PrivateKey) Public() crypto.PublicKey {
	s, err := edwards25519.NewScalar().SetBytesWithClamping(priv[:32])
	if err != nil {
		panic("ed25519: internal error: setting scalar failed")
	}
	return (&edwards25519.Point{}).ScalarBaseMult(s).Bytes()
}

// Equal reports whether priv and x have the same value.
func (priv PrivateKey) Equal(x crypto.PrivateKey) bool {
	// TODO do we have any need to check equality with standard library ed25519
	// private keys?
	xx, ok := x.(PrivateKey)
	if !ok {
		return false
	}
	return subtle.ConstantTimeCompare(priv, xx) == 1
}

// Sign signs the given message with priv. rand is ignored and can be nil.
//
// If opts.HashFunc() is [crypto.SHA512], the pre-hashed variant Ed25519ph is used
// and message is expected to be a SHA-512 hash, otherwise opts.HashFunc() must
// be [crypto.Hash](0) and the message must not be hashed, as Ed25519 performs two
// passes over messages to be signed.
//
// A value of type [Options] can be used as opts, or crypto.Hash(0) or
// crypto.SHA512 directly to select plain Ed25519 or Ed25519ph, respectively.
func (priv PrivateKey) Sign(rand io.Reader, message []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	hash := opts.HashFunc()
	context := ""
	if opts, ok := opts.(*Options); ok {
		context = opts.Context
	}
	switch {
	case hash == crypto.SHA512: // Ed25519ph
		if l := len(message); l != sha512.Size {
			return nil, errors.New("ed25519: bad Ed25519ph message hash length: " + strconv.Itoa(l))
		}
		if l := len(context); l > 255 {
			return nil, errors.New("ed25519: bad Ed25519ph context length: " + strconv.Itoa(l))
		}
		signature := make([]byte, SignatureSize)
		sign(signature, priv, message, domPrefixPh, context)
		return signature, nil
	case hash == crypto.Hash(0) && context != "": // Ed25519ctx
		if l := len(context); l > 255 {
			return nil, errors.New("ed25519: bad Ed25519ctx context length: " + strconv.Itoa(l))
		}
		signature := make([]byte, SignatureSize)
		sign(signature, priv, message, domPrefixCtx, context)
		return signature, nil
	case hash == crypto.Hash(0): // Ed25519
		return Sign(priv, message), nil
	default:
		return nil, errors.New("ed25519: expected opts.HashFunc() zero (unhashed message, for standard Ed25519) or SHA-512 (for Ed25519ph)")
	}
}

// Options can be used with [PrivateKey.Sign] or [VerifyWithOptions]
// to select Ed25519 variants.
type Options struct {
	// Hash can be zero for regular Ed25519, or crypto.SHA512 for Ed25519ph.
	Hash crypto.Hash

	// Context, if not empty, selects Ed25519ctx or provides the context string
	// for Ed25519ph. It can be at most 255 bytes in length.
	Context string
}

// HashFunc returns o.Hash.
func (o *Options) HashFunc() crypto.Hash { return o.Hash }

// GenerateKey generates a public/private key pair using entropy from rand.
// If rand is nil, [crypto/rand.Reader] will be used.
//
// The output of this function is deterministic, and equivalent to reading
// [SeedSize] bytes from rand, and passing them to [NewKeyFromSeed].
func GenerateKey(rand io.Reader) (PublicKey, PrivateKey, error) {
	if rand == nil {
		rand = cryptorand.Reader
	}

	seed := make([]byte, SeedSize)
	if _, err := io.ReadFull(rand, seed); err != nil {
		return nil, nil, err
	}

	privateKey := NewKeyFromSeed(seed)
	return PublicKey(privateKey.Public().([]byte)), privateKey, nil
}

// NewKeyFromSeed calculates a private key from a seed. It will panic if
// len(seed) is not [SeedSize]. This function is provided for interoperability
// with RFC 8032. RFC 8032's private keys correspond to seeds in this
// package.
func NewKeyFromSeed(seed []byte) PrivateKey {
	// Outline the function body so that the returned key can be stack-allocated.
	privateKey := make([]byte, PrivateKeySize)
	newKeyFromSeed(privateKey, seed)
	return privateKey
}

func newKeyFromSeed(privateKey, seed []byte) {
	if l := len(seed); l != SeedSize {
		panic("ed25519: bad seed length: " + strconv.Itoa(l))
	}

	h := sha512.Sum512(seed)

	// Apply clamping to get A in the left half, and leave the right half
	// as-is. This gets the private key into the NaCl format.
	h[0] &= 248
	h[31] &= 63
	h[31] |= 64
	copy(privateKey, h[:])
}

// Sign signs the message with privateKey and returns a signature. It will
// panic if len(privateKey) is not [PrivateKeySize].
func Sign(privateKey PrivateKey, message []byte) []byte {
	// Outline the function body so that the returned signature can be
	// stack-allocated.
	signature := make([]byte, SignatureSize)
	sign(signature, privateKey, message, domPrefixPure, "")
	return signature
}

// Domain separation prefixes used to disambiguate Ed25519/Ed25519ph/Ed25519ctx.
// See RFC 8032, Section 2 and Section 5.1.
const (
	// domPrefixPure is empty for pure Ed25519.
	domPrefixPure = ""
	// domPrefixPh is dom2(phflag=1) for Ed25519ph. It must be followed by the
	// uint8-length prefixed context.
	domPrefixPh = "SigEd25519 no Ed25519 collisions\x01"
	// domPrefixCtx is dom2(phflag=0) for Ed25519ctx. It must be followed by the
	// uint8-length prefixed context.
	domPrefixCtx = "SigEd25519 no Ed25519 collisions\x00"
)

func sign(signature []byte, privateKey PrivateKey, message []byte, domPrefix, context string) {
	if l := len(privateKey); l != PrivateKeySize {
		panic("ed25519: bad private key length: " + strconv.Itoa(l))
	}
	// We have to extract the public key from the private key.
	publicKey := privateKey.Public().([]byte)
	// The private key is already the hashed value of the seed.
	h := privateKey

	s, err := edwards25519.NewScalar().SetBytesWithClamping(h[:32])
	if err != nil {
		panic("ed25519: internal error: setting scalar failed")
	}
	prefix := h[32:]

	mh := sha512.New()
	if domPrefix != domPrefixPure {
		mh.Write([]byte(domPrefix))
		mh.Write([]byte{byte(len(context))})
		mh.Write([]byte(context))
	}
	mh.Write(prefix)
	mh.Write(message)
	messageDigest := make([]byte, 0, sha512.Size)
	messageDigest = mh.Sum(messageDigest)
	r, err := edwards25519.NewScalar().SetUniformBytes(messageDigest)
	if err != nil {
		panic("ed25519: internal error: setting scalar failed")
	}

	R := (&edwards25519.Point{}).ScalarBaseMult(r)

	kh := sha512.New()
	if domPrefix != domPrefixPure {
		kh.Write([]byte(domPrefix))
		kh.Write([]byte{byte(len(context))})
		kh.Write([]byte(context))
	}
	kh.Write(R.Bytes())
	kh.Write(publicKey)
	kh.Write(message)
	hramDigest := make([]byte, 0, sha512.Size)
	hramDigest = kh.Sum(hramDigest)
	k, err := edwards25519.NewScalar().SetUniformBytes(hramDigest)
	if err != nil {
		panic("ed25519: internal error: setting scalar failed")
	}

	S := edwards25519.NewScalar().MultiplyAdd(k, s, r)

	copy(signature[:32], R.Bytes())
	copy(signature[32:], S.Bytes())
}

// Verify reports whether sig is a valid signature of message by publicKey. It
// will panic if len(publicKey) is not [PublicKeySize].
//
// This is just a wrapper around [ed25519.Verify] from the standard library.
func Verify(publicKey PublicKey, message, sig []byte) bool {
	return ed25519.Verify(ed25519.PublicKey(publicKey), message, sig)
}

// VerifyWithOptions reports whether sig is a valid signature of message by
// publicKey. A valid signature is indicated by returning a nil error. It will
// panic if len(publicKey) is not [PublicKeySize].
//
// If opts.Hash is [crypto.SHA512], the pre-hashed variant Ed25519ph is used and
// message is expected to be a SHA-512 hash, otherwise opts.Hash must be
// [crypto.Hash](0) and the message must not be hashed, as Ed25519 performs two
// passes over messages to be signed.
//
// This is just a wrapper around [ed25519.VerifyWithOptions] from the standard
// library.
func VerifyWithOptions(publicKey PublicKey, message, sig []byte, opts *Options) error {
	return ed25519.VerifyWithOptions(ed25519.PublicKey(publicKey), message, sig, &ed25519.Options{
		Hash:    opts.Hash,
		Context: opts.Context,
	})
}
