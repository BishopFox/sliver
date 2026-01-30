// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package aescbc

import (
	"crypto/aes"
	"crypto/cipher"

	"maunium.net/go/mautrix/crypto/pkcs7"
)

// Encrypt encrypts the plaintext with the key and IV. The IV length must be
// equal to the AES block size.
//
// This function might mutate the plaintext.
func Encrypt(key, iv, plaintext []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, ErrNoKeyProvided
	}
	if len(iv) != aes.BlockSize {
		return nil, ErrIVNotBlockSize
	}
	plaintext = pkcs7.Pad(plaintext, aes.BlockSize)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	cipher.NewCBCEncrypter(block, iv).CryptBlocks(plaintext, plaintext)
	return plaintext, nil
}

// Decrypt decrypts the ciphertext with the key and IV. The IV length must be
// equal to the block size.
//
// This function mutates the ciphertext.
func Decrypt(key, iv, ciphertext []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, ErrNoKeyProvided
	}
	if len(iv) != aes.BlockSize {
		return nil, ErrIVNotBlockSize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < aes.BlockSize {
		return nil, ErrNotMultipleBlockSize
	}

	cipher.NewCBCDecrypter(block, iv).CryptBlocks(ciphertext, ciphertext)
	return pkcs7.Unpad(ciphertext), nil
}
