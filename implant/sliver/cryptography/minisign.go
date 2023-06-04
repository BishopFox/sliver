package cryptography

/*
	MIT License

	Copyright (c) 2018-2021 Frank Denis

	Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"), to deal
	in the Software without restriction, including without limitation the rights
	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
	copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all
	copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
	SOFTWARE.
*/

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strings"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/ed25519"
)

var (
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

// PublicKey - Represents a public key
type PublicKey struct {
	SignatureAlgorithm [2]byte
	KeyId              [8]byte
	PublicKey          [32]byte
}

// ID returns the 64 bit key ID.
func (p PublicKey) ID() uint64 {
	return binary.LittleEndian.Uint64(p.KeyId[:])
}

// Signature - Represents a minisign signature
type Signature struct {
	UntrustedComment   string
	SignatureAlgorithm [2]byte
	KeyId              [8]byte
	Signature          [64]byte
	TrustedComment     string
	GlobalSignature    [64]byte
}

// minisignPublicKey - Creates a new public key
func minisignPublicKey(publicKeyStr string) (PublicKey, error) {
	var publicKey PublicKey
	bin, err := base64.StdEncoding.DecodeString(publicKeyStr)
	if err != nil || len(bin) != 42 {
		return publicKey, errors.New("invalid encoded public key")
	}
	copy(publicKey.SignatureAlgorithm[:], bin[0:2])
	copy(publicKey.KeyId[:], bin[2:10])
	copy(publicKey.PublicKey[:], bin[10:42])
	return publicKey, nil
}

func DecodeMinisignPublicKey(in string) (PublicKey, error) {
	var publicKey PublicKey
	lines := strings.SplitN(in, "\n", 2)
	if len(lines) < 2 {
		return publicKey, errors.New("incomplete encoded public key")
	}
	return minisignPublicKey(lines[1])
}

func trimCarriageReturn(input string) string {
	return strings.TrimRight(input, "\r")
}

// DecodeMinisignSignature - Decodes a signature
func DecodeMinisignSignature(in string) (Signature, error) {
	var signature Signature
	lines := strings.SplitN(in, "\n", 4)
	if len(lines) < 4 {
		return signature, errors.New("incomplete encoded signature")
	}
	signature.UntrustedComment = trimCarriageReturn(lines[0])
	bin1, err := base64.StdEncoding.DecodeString(lines[1])
	if err != nil || len(bin1) != 74 {
		return signature, errors.New("invalid encoded signature")
	}
	signature.TrustedComment = trimCarriageReturn(lines[2])
	bin2, err := base64.StdEncoding.DecodeString(lines[3])
	if err != nil || len(bin2) != 64 {
		return signature, errors.New("invalid encoded signature")
	}
	copy(signature.SignatureAlgorithm[:], bin1[0:2])
	copy(signature.KeyId[:], bin1[2:10])
	copy(signature.Signature[:], bin1[10:74])
	copy(signature.GlobalSignature[:], bin2)
	return signature, nil
}

// Verify - Verifies a signature of a buffer
func (publicKey *PublicKey) Verify(bin []byte, signature Signature) (bool, error) {
	if publicKey.SignatureAlgorithm != [2]byte{'E', 'd'} {
		return false, errors.New("incompatible signature algorithm")
	}
	prehashed := false
	if signature.SignatureAlgorithm[0] == 0x45 && signature.SignatureAlgorithm[1] == 0x64 {
		prehashed = false
	} else if signature.SignatureAlgorithm[0] == 0x45 && signature.SignatureAlgorithm[1] == 0x44 {
		prehashed = true
	} else {
		return false, errors.New("unsupported signature algorithm")
	}
	if publicKey.KeyId != signature.KeyId {
		return false, errors.New("incompatible key identifiers")
	}
	if !strings.HasPrefix(signature.TrustedComment, "trusted comment: ") {
		return false, errors.New("unexpected format for the trusted comment")
	}

	if prehashed {
		h, _ := blake2b.New512(nil)
		h.Write(bin)
		bin = h.Sum(nil)
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey.PublicKey[:]), bin, signature.Signature[:]) {
		return false, errors.New("invalid signature")
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey.PublicKey[:]), append(signature.Signature[:], []byte(signature.TrustedComment)[17:]...), signature.GlobalSignature[:]) {
		return false, errors.New("invalid global signature")
	}
	return true, nil
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
	if !ed25519.Verify(ed25519.PublicKey(publicKey.PublicKey[:]), message, signature[:]) {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(publicKey.PublicKey[:]), message, signature[:])
}
