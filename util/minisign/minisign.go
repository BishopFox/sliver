// Copyright (c) 2021 Andreas Auernhammer. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.
// Modifications moloch--

// Package minisign implements the minisign signature scheme.
package minisign

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"hash"
	"io"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/blake2b"
)

const (
	// EdDSA refers to the Ed25519 signature scheme.
	//
	// Minisign uses this signature scheme to sign and
	// verify (non-hashed) messages.
	EdDSA uint16 = 0x6445

	// HashEdDSA refers to a Ed25519 signature scheme
	// with pre-hashed messages.
	//
	// Minisign uses this signature scheme to sign and
	// verify message that don't fit into memory.
	HashEdDSA uint16 = 0x4445

	RawSigSize = 2 + 8 + ed25519.SignatureSize
)

// GenerateKey generates a public/private key pair using entropy
// from random. If random is nil, crypto/rand.Reader will be used.
func GenerateKey(random io.Reader) (PublicKey, PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(random)
	if err != nil {
		return PublicKey{}, PrivateKey{}, err
	}

	id := blake2b.Sum256(pub[:])
	publicKey := PublicKey{
		id: binary.LittleEndian.Uint64(id[:8]),
	}
	copy(publicKey.bytes[:], pub)

	privateKey := PrivateKey{
		RawID: publicKey.ID(),
	}
	copy(privateKey.RawBytes[:], priv)

	return publicKey, privateKey, nil
}

// Reader is an io.Reader that reads a message
// while, at the same time, computes its digest.
//
// At any point, typically at the end of the message,
// Reader can sign the message digest with a private
// key or try to verify the message with a public key
// and signature.
type Reader struct {
	message io.Reader
	hash    hash.Hash
}

// NewReader returns a new Reader that reads from r
// and computes a digest of the read data.
func NewReader(r io.Reader) *Reader {
	h, err := blake2b.New512(nil)
	if err != nil {
		panic(err)
	}
	return &Reader{
		message: r,
		hash:    h,
	}
}

// Read reads from the underlying io.Reader as specified
// by the io.Reader interface.
func (r *Reader) Read(p []byte) (int, error) {
	n, err := r.message.Read(p)
	r.hash.Write(p[:n])
	return n, err
}

// Sign signs whatever has been read from the underlying
// io.Reader up to this point in time with the given private
// key.
//
// It behaves like SignWithComments but uses some generic comments.
func (r *Reader) Sign(privateKey PrivateKey) []byte {
	var (
		trustedComment = strconv.FormatInt(time.Now().Unix(), 10)
	)
	return r.SignWithComments(privateKey, trustedComment, "")
}

// SignWithComments signs whatever has been read from the underlying
// io.Reader up to this point in time with the given private key.
//
// The trustedComment as well as the untrustedComment are embedded into the
// returned signature. The trustedComment is signed and will be checked
// when the signature is verified. The untrustedComment is not signed and
// must not be trusted.
//
// SignWithComments computes the digest as a snapshot. So, it is possible
// to create multiple signatures of different message prefixes by reading
// up to a certain byte, signing this message prefix, and then continue
// reading.
func (r *Reader) SignWithComments(privateKey PrivateKey, trustedComment, untrustedComment string) []byte {
	const isHashed = true
	return sign(privateKey, r.hash.Sum(nil), trustedComment, untrustedComment, isHashed)
}

// Verify checks whether whatever has been read from the underlying
// io.Reader up to this point in time is authentic by verifying it
// with the given public key and signature.
//
// Verify computes the digest as a snapshot. Therefore, Verify can
// verify any signature produced by Sign or SignWithComments,
// including signatures of partial messages, given the correct
// public key and signature.
func (r *Reader) Verify(publicKey PublicKey, signature []byte) bool {
	const isHashed = true
	return verify(publicKey, r.hash.Sum(nil), signature, isHashed)
}

// Sign signs the given message with the private key.
//
// It behaves like SignWithComments with some generic comments.
func Sign(privateKey PrivateKey, message []byte) []byte {
	var (
		trustedComment = strconv.FormatInt(time.Now().Unix(), 10)
	)
	return SignWithComments(privateKey, message, trustedComment, "")
}

// SignRawBuf - Sign buffer with raw signature
func SignRawBuf(privateKey PrivateKey, message []byte) [RawSigSize]byte {
	return signRaw(privateKey, message, false)
}

// VerifyRawBuf - Verify buffer with raw signature
func VerifyRawBuf(publicKey PublicKey, rawMessage []byte) bool {
	return verifyRaw(publicKey, rawMessage, false)
}

// SignWithComments signs the given message with the private key.
//
// The trustedComment as well as the untrustedComment are embedded
// into the returned signature. The trustedComment is signed and
// will be checked when the signature is verified.
// The untrustedComment is not signed and must not be trusted.
func SignWithComments(privateKey PrivateKey, message []byte, trustedComment, untrustedComment string) []byte {
	const isHashed = false
	return sign(privateKey, message, trustedComment, untrustedComment, isHashed)
}

// Verify checks whether message is authentic by verifying
// it with the given public key and signature. It returns
// true if and only if the signature verification is successful.
func Verify(publicKey PublicKey, message, signature []byte) bool {
	const isHashed = false
	return verify(publicKey, message, signature, isHashed)
}

func signRaw(privateKey PrivateKey, message []byte, isHashed bool) [RawSigSize]byte {
	var algorithm = EdDSA
	if isHashed {
		algorithm = HashEdDSA
	}
	var (
		msgSignature = ed25519.Sign(ed25519.PrivateKey(privateKey.RawBytes[:]), message)
	)
	var rawSig [RawSigSize]byte
	binary.LittleEndian.PutUint16(rawSig[:2], algorithm)
	binary.LittleEndian.PutUint64(rawSig[2:10], privateKey.ID())
	copy(rawSig[10:], msgSignature[:])
	return rawSig
}

func verifyRaw(publicKey PublicKey, rawMessage []byte, isHashed bool) bool {
	if len(rawMessage) < RawSigSize+1 {
		return false
	}
	rawSigBuf := rawMessage[:RawSigSize]
	algorithm := binary.LittleEndian.Uint16(rawSigBuf[:2])
	keyID := binary.LittleEndian.Uint64(rawSigBuf[2:10])
	signature := rawSigBuf[10:]
	message := rawMessage[RawSigSize:]

	if keyID != publicKey.ID() {
		return false
	}
	if algorithm == HashEdDSA && !isHashed {
		h := blake2b.Sum512(message)
		message = h[:]
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey.bytes[:]), message, signature[:]) {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(publicKey.bytes[:]), message, signature[:])
}

func sign(privateKey PrivateKey, message []byte, trustedComment, untrustedComment string, isHashed bool) []byte {
	var algorithm = EdDSA
	if isHashed {
		algorithm = HashEdDSA
	}

	var (
		msgSignature     = ed25519.Sign(ed25519.PrivateKey(privateKey.RawBytes[:]), message)
		commentSignature = ed25519.Sign(ed25519.PrivateKey(privateKey.RawBytes[:]), append(msgSignature, []byte(trustedComment)...))
	)
	signature := Signature{
		Algorithm: algorithm,
		KeyID:     privateKey.ID(),

		TrustedComment:   trustedComment,
		UntrustedComment: untrustedComment,
	}
	copy(signature.Signature[:], msgSignature)
	copy(signature.CommentSignature[:], commentSignature)

	text, err := signature.MarshalText()
	if err != nil {
		panic(err)
	}
	return text
}

func verify(publicKey PublicKey, message, signature []byte, isHashed bool) bool {
	var s Signature
	if err := s.UnmarshalText(signature); err != nil {
		return false
	}
	if s.KeyID != publicKey.ID() {
		return false
	}
	if s.Algorithm == HashEdDSA && !isHashed {
		h := blake2b.Sum512(message)
		message = h[:]
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey.bytes[:]), message, s.Signature[:]) {
		return false
	}
	globalMessage := append(s.Signature[:], []byte(s.TrustedComment)...)
	return ed25519.Verify(ed25519.PublicKey(publicKey.bytes[:]), globalMessage, s.CommentSignature[:])
}

// trimUntrustedComment returns text with a potential
// untrusted comment line.
func trimUntrustedComment(text []byte) []byte {
	s := bytes.SplitN(text, []byte{'\n'}, 2)
	if len(s) == 2 && strings.HasPrefix(string(s[0]), "untrusted comment: ") {
		return s[1]
	}
	return s[0]
}
