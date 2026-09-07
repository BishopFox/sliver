// Copyright (c) 2022 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package attachment

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"io"

	"maunium.net/go/mautrix/crypto/utils"
)

var (
	ErrHashMismatch         = errors.New("mismatching SHA-256 digest")
	ErrUnsupportedVersion   = errors.New("unsupported Matrix file encryption version")
	ErrUnsupportedAlgorithm = errors.New("unsupported JWK encryption algorithm")
	ErrInvalidKey           = errors.New("failed to decode key")
	ErrInvalidInitVector    = errors.New("failed to decode initialization vector")
	ErrInvalidHash          = errors.New("failed to decode SHA-256 hash")
	ErrReaderClosed         = errors.New("encrypting reader was already closed")
)

// Deprecated: use variables prefixed with Err
var (
	HashMismatch         = ErrHashMismatch
	UnsupportedVersion   = ErrUnsupportedVersion
	UnsupportedAlgorithm = ErrUnsupportedAlgorithm
	InvalidKey           = ErrInvalidKey
	InvalidInitVector    = ErrInvalidInitVector
	InvalidHash          = ErrInvalidHash
	ReaderClosed         = ErrReaderClosed
)

var (
	keyBase64Length  = base64.RawURLEncoding.EncodedLen(utils.AESCTRKeyLength)
	ivBase64Length   = base64.RawStdEncoding.EncodedLen(utils.AESCTRIVLength)
	hashBase64Length = base64.RawStdEncoding.EncodedLen(utils.SHAHashLength)
)

type JSONWebKey struct {
	Key         string   `json:"k"`
	Algorithm   string   `json:"alg"`
	Extractable bool     `json:"ext"`
	KeyType     string   `json:"kty"`
	KeyOps      []string `json:"key_ops"`
}

type EncryptedFileHashes struct {
	SHA256 string `json:"sha256"`
}

type decodedKeys struct {
	key [utils.AESCTRKeyLength]byte
	iv  [utils.AESCTRIVLength]byte

	sha256 [utils.SHAHashLength]byte
}

type EncryptedFile struct {
	Key        JSONWebKey          `json:"key"`
	InitVector string              `json:"iv"`
	Hashes     EncryptedFileHashes `json:"hashes"`
	Version    string              `json:"v"`

	decoded *decodedKeys
}

func NewEncryptedFile() *EncryptedFile {
	key, iv := utils.GenAttachmentA256CTR()
	return &EncryptedFile{
		Key: JSONWebKey{
			Key:         base64.RawURLEncoding.EncodeToString(key[:]),
			Algorithm:   "A256CTR",
			Extractable: true,
			KeyType:     "oct",
			KeyOps:      []string{"encrypt", "decrypt"},
		},
		InitVector: base64.RawStdEncoding.EncodeToString(iv[:]),
		Version:    "v2",

		decoded: &decodedKeys{key: key, iv: iv},
	}
}

func (ef *EncryptedFile) decodeKeys(includeHash bool) error {
	if ef.decoded != nil {
		return nil
	} else if len(ef.Key.Key) != keyBase64Length {
		return ErrInvalidKey
	} else if len(ef.InitVector) != ivBase64Length {
		return ErrInvalidInitVector
	} else if includeHash && len(ef.Hashes.SHA256) != hashBase64Length {
		return ErrInvalidHash
	}
	ef.decoded = &decodedKeys{}
	_, err := base64.RawURLEncoding.Decode(ef.decoded.key[:], []byte(ef.Key.Key))
	if err != nil {
		return ErrInvalidKey
	}
	_, err = base64.RawStdEncoding.Decode(ef.decoded.iv[:], []byte(ef.InitVector))
	if err != nil {
		return ErrInvalidInitVector
	}
	if includeHash {
		_, err = base64.RawStdEncoding.Decode(ef.decoded.sha256[:], []byte(ef.Hashes.SHA256))
		if err != nil {
			return ErrInvalidHash
		}
	}
	return nil
}

// Encrypt encrypts the given data, updates the SHA256 hash in the EncryptedFile struct and returns the ciphertext.
//
// Deprecated: this makes a copy for the ciphertext, which means 2x memory usage. EncryptInPlace is recommended.
func (ef *EncryptedFile) Encrypt(plaintext []byte) []byte {
	ciphertext := make([]byte, len(plaintext))
	copy(ciphertext, plaintext)
	ef.EncryptInPlace(ciphertext)
	return ciphertext
}

// EncryptInPlace encrypts the given data in-place (i.e. the provided data is overridden with the ciphertext)
// and updates the SHA256 hash in the EncryptedFile struct.
func (ef *EncryptedFile) EncryptInPlace(data []byte) {
	ef.decodeKeys(false)
	utils.XorA256CTR(data, ef.decoded.key, ef.decoded.iv)
	checksum := sha256.Sum256(data)
	ef.Hashes.SHA256 = base64.RawStdEncoding.EncodeToString(checksum[:])
}

type ReadWriterAt interface {
	io.WriterAt
	io.Reader
}

// EncryptFile encrypts the given file in-place and updates the SHA256 hash in the EncryptedFile struct.
func (ef *EncryptedFile) EncryptFile(file ReadWriterAt) error {
	err := ef.decodeKeys(false)
	if err != nil {
		return err
	}
	block, _ := aes.NewCipher(ef.decoded.key[:])
	stream := cipher.NewCTR(block, ef.decoded.iv[:])
	hasher := sha256.New()
	buf := make([]byte, 32*1024)
	var writePtr int64
	var n int
	for {
		n, err = file.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if n == 0 {
			break
		}
		stream.XORKeyStream(buf[:n], buf[:n])
		_, err = file.WriteAt(buf[:n], writePtr)
		if err != nil {
			return err
		}
		writePtr += int64(n)
		hasher.Write(buf[:n])
	}
	ef.Hashes.SHA256 = base64.RawStdEncoding.EncodeToString(hasher.Sum(nil))
	return nil
}

type encryptingReader struct {
	stream cipher.Stream
	hash   hash.Hash
	source io.Reader
	file   *EncryptedFile
	closed bool

	isDecrypting bool
}

var _ io.ReadSeekCloser = (*encryptingReader)(nil)

func (r *encryptingReader) Seek(offset int64, whence int) (int64, error) {
	if r.closed {
		return 0, ErrReaderClosed
	}
	if offset != 0 || whence != io.SeekStart {
		return 0, fmt.Errorf("attachments.EncryptStream: only seeking to the beginning is supported")
	}
	seeker, ok := r.source.(io.ReadSeeker)
	if !ok {
		return 0, fmt.Errorf("attachments.EncryptStream: source reader (%T) is not an io.ReadSeeker", r.source)
	}
	n, err := seeker.Seek(offset, whence)
	if err != nil {
		return 0, err
	}
	block, _ := aes.NewCipher(r.file.decoded.key[:])
	r.stream = cipher.NewCTR(block, r.file.decoded.iv[:])
	r.hash.Reset()
	return n, nil
}

func (r *encryptingReader) Read(dst []byte) (n int, err error) {
	if r.closed {
		return 0, ErrReaderClosed
	} else if r.isDecrypting && r.file.decoded == nil {
		if err = r.file.PrepareForDecryption(); err != nil {
			return
		}
	}
	n, err = r.source.Read(dst)
	if r.isDecrypting {
		r.hash.Write(dst[:n])
	}
	r.stream.XORKeyStream(dst[:n], dst[:n])
	if !r.isDecrypting {
		r.hash.Write(dst[:n])
	}
	return
}

func (r *encryptingReader) Close() (err error) {
	closer, ok := r.source.(io.ReadCloser)
	if ok {
		err = closer.Close()
	}
	if r.isDecrypting {
		if !hmac.Equal(r.hash.Sum(nil), r.file.decoded.sha256[:]) {
			return ErrHashMismatch
		}
	} else {
		r.file.Hashes.SHA256 = base64.RawStdEncoding.EncodeToString(r.hash.Sum(nil))
	}
	r.closed = true
	return
}

// EncryptStream wraps the given io.Reader in order to encrypt the data.
//
// The Close() method of the returned io.ReadCloser must be called for the SHA256 hash
// in the EncryptedFile struct to be updated. The metadata is not valid before the hash
// is filled.
func (ef *EncryptedFile) EncryptStream(reader io.Reader) io.ReadSeekCloser {
	ef.decodeKeys(false)
	block, _ := aes.NewCipher(ef.decoded.key[:])
	return &encryptingReader{
		stream: cipher.NewCTR(block, ef.decoded.iv[:]),
		hash:   sha256.New(),
		source: reader,
		file:   ef,
	}
}

// Decrypt decrypts the given data and returns the plaintext.
//
// Deprecated: this makes a copy for the plaintext data, which means 2x memory usage. DecryptInPlace is recommended.
func (ef *EncryptedFile) Decrypt(ciphertext []byte) ([]byte, error) {
	plaintext := make([]byte, len(ciphertext))
	copy(plaintext, ciphertext)
	return plaintext, ef.DecryptInPlace(plaintext)
}

// PrepareForDecryption checks that the version and algorithm are supported and decodes the base64 keys
//
// DecryptStream will call this with the first Read() call if this hasn't been called manually.
//
// DecryptInPlace will always call this automatically, so calling this manually is not necessary when using that function.
func (ef *EncryptedFile) PrepareForDecryption() error {
	if ef.Version != "v2" {
		return ErrUnsupportedVersion
	} else if ef.Key.Algorithm != "A256CTR" {
		return ErrUnsupportedAlgorithm
	} else if err := ef.decodeKeys(true); err != nil {
		return err
	}
	return nil
}

// DecryptInPlace decrypts the given data in-place (i.e. the provided data is overridden with the plaintext).
func (ef *EncryptedFile) DecryptInPlace(data []byte) error {
	if err := ef.PrepareForDecryption(); err != nil {
		return err
	}
	dataHash := sha256.Sum256(data)
	if !hmac.Equal(ef.decoded.sha256[:], dataHash[:]) {
		return ErrHashMismatch
	}
	utils.XorA256CTR(data, ef.decoded.key, ef.decoded.iv)
	return nil
}

// DecryptStream wraps the given io.Reader in order to decrypt the data.
//
// The first Read call will check the algorithm and decode keys, so it might return an error before actually reading anything.
// If you want to validate the file before opening the stream, call PrepareForDecryption manually and check for errors.
//
// The Close call will validate the hash and return an error if it doesn't match.
// In this case, the written data should be considered compromised and should not be used further.
func (ef *EncryptedFile) DecryptStream(reader io.Reader) io.ReadSeekCloser {
	block, _ := aes.NewCipher(ef.decoded.key[:])
	return &encryptingReader{
		isDecrypting: true,
		stream:       cipher.NewCTR(block, ef.decoded.iv[:]),
		hash:         sha256.New(),
		source:       reader,
		file:         ef,
	}
}
