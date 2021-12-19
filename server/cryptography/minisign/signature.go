// Copyright (c) 2021 Andreas Auernhammer. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package minisign

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

// SignatureFromFile reads a new Signature from the
// given file.
func SignatureFromFile(file string) (Signature, error) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return Signature{}, err
	}

	var signature Signature
	if err = signature.UnmarshalText(bytes); err != nil {
		return Signature{}, err
	}
	return signature, nil
}

// Signature is a structured representation of a minisign
// signature.
//
// A signature is generated when signing a message with
// a private key:
//   signature = Sign(privateKey, message)
//
// The signature of a message can then be verified with the
// corresponding public key:
//   if Verify(publicKey, message, signature) {
//      // => signature is valid
//      // => message has been signed with correspoding private key
//   }
//
type Signature struct {
	_ [0]func() // enforce named assignment and prevent direct comparison

	// Algorithm is the signature algorithm. It is either
	// EdDSA or HashEdDSA.
	Algorithm uint16

	// KeyID may be the 64 bit ID of the private key that was used
	// to produce this signature. It can be used to identify the
	// corresponding public key that can verify the signature.
	//
	// However, key IDs are random identifiers and not protected at all.
	// A key ID is just a hint to quickly identify a public key candidate.
	KeyID uint64

	// TrustedComment is a comment that has been signed and is
	// verified during signature verification.
	TrustedComment string

	// UntrustedComment is a comment that has not been signed
	// and is not verified during signature verification.
	//
	// It must not be considered authentic - in contrast to the
	// TrustedComment.
	UntrustedComment string

	// Signature is the Ed25519 signature of the message that
	// has been signed.
	Signature [ed25519.SignatureSize]byte

	// CommentSignature is the Ed25519 signature of Signature
	// concatenated with the TrustedComment:
	//
	//    CommentSignature = ed25519.Sign(PrivateKey, Signature || TrustedComment)
	//
	// It is used to verify that the TrustedComment is authentic.
	CommentSignature [ed25519.SignatureSize]byte
}

// String returns a string representation of the Signature s.
//
// In contrast to MarshalText, String does not fail if s is
// not a valid minisign signature.
func (s Signature) String() string {
	var buffer strings.Builder
	buffer.WriteString("untrusted comment: ")
	buffer.WriteString(s.UntrustedComment)
	buffer.WriteByte('\n')

	var signature [2 + 8 + ed25519.SignatureSize]byte
	binary.LittleEndian.PutUint16(signature[:2], s.Algorithm)
	binary.LittleEndian.PutUint64(signature[2:10], s.KeyID)
	copy(signature[10:], s.Signature[:])

	buffer.WriteString(base64.StdEncoding.EncodeToString(signature[:]))
	buffer.WriteByte('\n')

	buffer.WriteString("trusted comment: ")
	buffer.WriteString(s.TrustedComment)
	buffer.WriteByte('\n')

	buffer.WriteString(base64.StdEncoding.EncodeToString(s.CommentSignature[:]))
	return buffer.String()
}

// Equal reports whether s and x have equivalent values.
//
// The untrusted comments of two equivalent signatures may differ.
func (s Signature) Equal(x Signature) bool {
	return s.Algorithm == x.Algorithm &&
		s.KeyID == x.KeyID &&
		s.Signature == x.Signature &&
		s.CommentSignature == x.CommentSignature &&
		s.TrustedComment == x.TrustedComment
}

// MarshalText returns a textual representation of the Signature s.
//
// It returns an error if s cannot be a valid signature - e.g.
// because the signature algorithm is neither EdDSA nor HashEdDSA.
func (s Signature) MarshalText() ([]byte, error) {
	if s.Algorithm != EdDSA && s.Algorithm != HashEdDSA {
		return nil, errors.New("minisign: invalid signature algorithm " + strconv.Itoa(int(s.Algorithm)))
	}
	return []byte(s.String()), nil
}

// UnmarshalText parses text as textual-encoded signature.
// It returns an error if text is not a well-formed minisign
// signature.
func (s *Signature) UnmarshalText(text []byte) error {
	segments := strings.SplitN(string(text), "\n", 4)
	if len(segments) != 4 {
		return errors.New("minisign: invalid signature")
	}

	var (
		untrustedComment        = strings.TrimRight(segments[0], "\r")
		encodedSignature        = segments[1]
		trustedComment          = strings.TrimRight(segments[2], "\r")
		encodedCommentSignature = segments[3]
	)
	if !strings.HasPrefix(untrustedComment, "untrusted comment: ") {
		return errors.New("minisign: invalid signature: invalid untrusted comment")
	}
	if !strings.HasPrefix(trustedComment, "trusted comment: ") {
		return errors.New("minisign: invalid signature: invalid trusted comment")
	}

	rawSignature, err := base64.StdEncoding.DecodeString(encodedSignature)
	if err != nil {
		return fmt.Errorf("minisign: invalid signature: %v", err)
	}
	if n := len(rawSignature); n != 2+8+ed25519.SignatureSize {
		return errors.New("minisign: invalid signature length " + strconv.Itoa(n))
	}
	commentSignature, err := base64.StdEncoding.DecodeString(encodedCommentSignature)
	if err != nil {
		return fmt.Errorf("minisign: invalid signature: %v", err)
	}
	if n := len(commentSignature); n != ed25519.SignatureSize {
		return errors.New("minisign: invalid comment signature length " + strconv.Itoa(n))
	}

	var (
		algorithm = binary.LittleEndian.Uint16(rawSignature[:2])
		keyID     = binary.LittleEndian.Uint64(rawSignature[2:10])
	)
	if algorithm != EdDSA && algorithm != HashEdDSA {
		return errors.New("minisign: invalid signature: invalid algorithm " + strconv.Itoa(int(algorithm)))
	}

	s.Algorithm = algorithm
	s.KeyID = keyID
	s.TrustedComment = strings.TrimPrefix(trustedComment, "trusted comment: ")
	s.UntrustedComment = strings.TrimPrefix(untrustedComment, "untrusted comment: ")
	copy(s.Signature[:], rawSignature[10:])
	copy(s.CommentSignature[:], commentSignature)
	return nil
}
