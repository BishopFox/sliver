// Copyright (c) 2021 Andreas Auernhammer. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package minisign

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/scrypt"
)

// PrivateKeyFromFile reads and decrypts the private key
// file with the given password.
func PrivateKeyFromFile(password, path string) (PrivateKey, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return PrivateKey{}, err
	}
	return DecryptKey(password, bytes)
}

// PrivateKey is a minisign private key.
//
// A private key can sign messages to prove the
// their origin and authenticity.
//
// PrivateKey implements the crypto.Signer interface.
type PrivateKey struct {
	_ [0]func() // prevent direct comparison: p1 == p2.

	id    uint64
	bytes [ed25519.PrivateKeySize]byte
}

var _ crypto.Signer = (*PrivateKey)(nil) // compiler check

// ID returns the 64 bit key ID.
func (p PrivateKey) ID() uint64 { return p.id }

// Public returns the corresponding public key.
func (p PrivateKey) Public() crypto.PublicKey {
	var bytes [ed25519.PublicKeySize]byte
	copy(bytes[:], p.bytes[32:])

	return PublicKey{
		id:    p.ID(),
		bytes: bytes,
	}
}

// Sign signs the given message.
//
// The minisign signature scheme relies on Ed25519 and supports
// plain as well as pre-hashed messages. Therefore, opts can be
// either crypto.Hash(0) to signal that the message has not been
// hashed or crypto.BLAKE2b_512 to signal that the message is a
// BLAKE2b-512 digest. If opts is crypto.BLAKE2b_512 then message
// must be a 64 bytes long.
//
// Minisign signatures are deterministic such that no randomness
// is necessary.
func (p PrivateKey) Sign(_ io.Reader, message []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	var (
		trustedComment   = "timestamp:" + strconv.FormatInt(time.Now().Unix(), 10)
		untrustedComment = "signature from private key: " + strings.ToUpper(strconv.FormatUint(p.ID(), 16))
	)
	switch h := opts.HashFunc(); h {
	case crypto.Hash(0):
		const isHashed = false
		return sign(p, message, trustedComment, untrustedComment, isHashed), nil
	case crypto.BLAKE2b_512:
		const isHashed = true
		if n := len(message); n != blake2b.Size {
			return nil, errors.New("minisign: invalid message length " + strconv.Itoa(n))
		}
		return sign(p, message, trustedComment, untrustedComment, isHashed), nil
	default:
		return nil, errors.New("minisign: cannot sign messages hashed with " + strconv.Itoa(int(h)))
	}
}

// Equal returns true if and only if p and x have equivalent values.
func (p PrivateKey) Equal(x crypto.PrivateKey) bool {
	xx, ok := x.(PrivateKey)
	if !ok {
		return false
	}
	return p.id == xx.id && subtle.ConstantTimeCompare(p.bytes[:], xx.bytes[:]) == 1
}

const (
	scryptAlgorithm  = 0x6353 // hex value for "Sc"
	blake2bAlgorithm = 0x3242 // hex value for "B2"

	scryptOpsLimit = 0x2000000  // max. Scrypt ops limit based on libsodium
	scryptMemLimit = 0x40000000 // max. Scrypt mem limit based on libsodium

	privateKeySize = 158 // 2 + 2 + 2 + 32 + 8 + 8 + 104
)

// EncryptKey encrypts the private key with the given password
// using some entropy from the RNG of the OS.
func EncryptKey(password string, privateKey PrivateKey) ([]byte, error) {
	var privateKeyBytes [72]byte
	binary.LittleEndian.PutUint64(privateKeyBytes[:], privateKey.ID())
	copy(privateKeyBytes[8:], privateKey.bytes[:])

	var salt [32]byte
	if _, err := io.ReadFull(rand.Reader, salt[:]); err != nil {
		return nil, err
	}

	var bytes [privateKeySize]byte
	binary.LittleEndian.PutUint16(bytes[0:], EdDSA)
	binary.LittleEndian.PutUint16(bytes[2:], scryptAlgorithm)
	binary.LittleEndian.PutUint16(bytes[4:], blake2bAlgorithm)

	const ( // TODO(aead): Callers may want to customize the cost parameters
		defaultOps = 33554432   // libsodium OPS_LIMIT_SENSITIVE
		defaultMem = 1073741824 // libsodium MEM_LIMIT_SENSITIVE
	)
	copy(bytes[6:38], salt[:])
	binary.LittleEndian.PutUint64(bytes[38:], defaultOps)
	binary.LittleEndian.PutUint64(bytes[46:], defaultMem)
	copy(bytes[54:], encryptKey(password, salt[:], defaultOps, defaultMem, privateKeyBytes[:]))

	const comment = "untrusted comment: minisign encrypted secret key\n"
	encodedBytes := make([]byte, len(comment)+base64.StdEncoding.EncodedLen(len(bytes)))
	copy(encodedBytes, []byte(comment))
	base64.StdEncoding.Encode(encodedBytes[len(comment):], bytes[:])
	return encodedBytes, nil
}

var errDecrypt = errors.New("minisign: decryption failed")

// DecryptKey tries to decrypt the encrypted private key with
// the given password.
func DecryptKey(password string, privateKey []byte) (PrivateKey, error) {
	privateKey = trimUntrustedComment(privateKey)
	bytes := make([]byte, base64.StdEncoding.DecodedLen(len(privateKey)))
	n, err := base64.StdEncoding.Decode(bytes, privateKey)
	if err != nil {
		return PrivateKey{}, err
	}
	bytes = bytes[:n]

	if len(bytes) != privateKeySize {
		return PrivateKey{}, errDecrypt
	}
	if a := binary.LittleEndian.Uint16(bytes[:2]); a != EdDSA {
		return PrivateKey{}, errDecrypt
	}
	if a := binary.LittleEndian.Uint16(bytes[2:4]); a != scryptAlgorithm {
		return PrivateKey{}, errDecrypt
	}
	if a := binary.LittleEndian.Uint16(bytes[4:6]); a != blake2bAlgorithm {
		return PrivateKey{}, errDecrypt
	}

	var (
		scryptOps = binary.LittleEndian.Uint64(bytes[38:46])
		scryptMem = binary.LittleEndian.Uint64(bytes[46:54])
	)
	if scryptOps > scryptOpsLimit {
		return PrivateKey{}, errDecrypt
	}
	if scryptMem > scryptMemLimit {
		return PrivateKey{}, errDecrypt
	}
	var salt [32]byte
	copy(salt[:], bytes[6:38])
	privateKeyBytes, err := decryptKey(password, salt[:], scryptOps, scryptMem, bytes[54:])
	if err != nil {
		return PrivateKey{}, err
	}

	key := PrivateKey{
		id: binary.LittleEndian.Uint64(privateKeyBytes[:8]),
	}
	copy(key.bytes[:], privateKeyBytes[8:])
	return key, nil
}

// encryptKey encrypts the plaintext and returns a ciphertext by:
//   1. tag        = BLAKE2b-256(EdDSA-const || plaintext)
//   2. keystream  = Scrypt(password, salt, convert(ops, mem))
//   3. ciphertext = (plaintext || tag) ⊕ keystream
//
// Therefore, decryptKey converts the ops and mem cost parameters
// to the (N, r, p)-tuple expected by Scrypt.
//
// The plaintext must be a private key ID concatenated with a raw
// Ed25519 private key, and therefore, 72 bytes long.
func encryptKey(password string, salt []byte, ops, mem uint64, plaintext []byte) []byte {
	const (
		plaintextLen  = 72
		messageLen    = 74
		ciphertextLen = 104
	)

	N, r, p := convertScryptParameters(ops, mem)
	keystream, err := scrypt.Key([]byte(password), salt, N, r, p, ciphertextLen)
	if err != nil {
		panic(err)
	}

	var message [messageLen]byte
	binary.LittleEndian.PutUint16(message[:2], EdDSA)
	copy(message[2:], plaintext)
	checksum := blake2b.Sum256(message[:])

	var ciphertext [ciphertextLen]byte
	copy(ciphertext[:plaintextLen], plaintext)
	copy(ciphertext[plaintextLen:], checksum[:])

	for i, k := range keystream {
		ciphertext[i] ^= k
	}
	return ciphertext[:]
}

// decryptKey decrypts the ciphertext and returns a plaintext by:
//   1. keystream        = Scrypt(password, salt, convert(ops, mem))
//   2. plaintext || tag = ciphertext ⊕ keystream
//   3. Check that: tag == BLAKE2b-256(EdDSA-const || plaintext)
//
// Therefore, decryptKey converts the ops and mem cost parameters to
// the (N, r, p)-tuple expected by Scrypt.
//
// It returns an error if the ciphertext is not valid - i.e. if the
// tag does not match the BLAKE2b-256 hash value.
func decryptKey(password string, salt []byte, ops, mem uint64, ciphertext []byte) ([]byte, error) {
	const (
		plaintextLen  = 72
		messageLen    = 74
		ciphertextLen = 104
	)
	if len(ciphertext) != ciphertextLen {
		return nil, errDecrypt
	}

	N, r, p := convertScryptParameters(ops, mem)
	keystream, err := scrypt.Key([]byte(password), salt, N, r, p, ciphertextLen)
	if err != nil {
		return nil, err
	}

	var plaintext [ciphertextLen]byte
	for i, k := range keystream {
		plaintext[i] = ciphertext[i] ^ k
	}
	var (
		privateKeyBytes = plaintext[:plaintextLen]
		checksum        = plaintext[plaintextLen:]
	)

	var message [messageLen]byte
	binary.LittleEndian.PutUint16(message[:2], EdDSA)
	copy(message[2:], privateKeyBytes)

	if sum := blake2b.Sum256(message[:]); subtle.ConstantTimeCompare(sum[:], checksum[:]) != 1 {
		return nil, errDecrypt
	}
	return privateKeyBytes, nil
}

// convertScryptParameters converts the operational and memory cost
// to the Scrypt parameters N, r and p.
//
// N is the overall memory / CPU cost and r * p has to be lower then
// 2³⁰. Refer to the scrypt.Key docs for more information.
func convertScryptParameters(ops, mem uint64) (N, r, p int) {
	const (
		minOps = 1 << 15
		maxRP  = 0x3fffffff
	)
	if ops < minOps {
		ops = minOps
	}

	if ops < mem/32 {
		r, p = 8, 1
		for n := 1; n < 63; n++ {
			if N = 1 << n; uint64(N) > (ops / (8 * uint64(r))) {
				break
			}
		}
	} else {
		r = 8
		for n := 1; n < 63; n++ {
			if N = 1 << n; uint64(N) > (mem / (256 * uint64(r))) {
				break
			}
		}
		if rp := (ops / 4) / uint64(N); rp < maxRP {
			p = int(rp) / r
		} else {
			p = maxRP / r
		}
	}
	return N, r, p
}
