// Copyright (c) 2024 Sumner Evans
// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package pkcs7 implements PKCS#7 padding and unpadding as defined in [RFC2315].
// This is normally used for AES-CBC encryption.
//
// [RFC2315]: https://www.ietf.org/rfc/rfc2315.txt
package pkcs7

import (
	"bytes"
	"errors"
	"fmt"
)

var (
	ErrInvalidBlockSize = errors.New("pkcs7: invalid block size")
	ErrEmptyData        = errors.New("pkcs7: empty data")
	ErrInvalidPadding   = errors.New("pkcs7: invalid padding")
)

// Pad pads the given data to the given block size.
// This will always clone the input; use PadAppend to use existing capacity at the end of the slice.
func Pad(data []byte, blockSize int) []byte {
	padding := calculatePadding(len(data), blockSize)
	output := make([]byte, len(data)+padding)
	copy(output, data)
	for i := len(data); i < len(output); i++ {
		output[i] = byte(padding)
	}
	return output
}

// PadAppend pads the given data to the given block size.
// This will reuse the existing slice if it has sufficient capacity; use Pad to always clone the input.
func PadAppend(data []byte, blockSize int) []byte {
	padding := calculatePadding(len(data), blockSize)
	return append(data, bytes.Repeat([]byte{byte(padding)}, padding)...)
}

// PadSplit pads the given data to the given block size, then returns the last block as a separate slice.
func PadSplit(data []byte, blockSize int) (mainData []byte, lastBlock []byte) {
	lastBlockSize := len(data) % blockSize
	mainData = data[:len(data)-lastBlockSize]
	lastBlock = Pad(data[len(data)-lastBlockSize:], blockSize)
	return
}

func calculatePadding(length, blockSize int) int {
	if blockSize < 1 || blockSize > 255 {
		panic(fmt.Errorf("%w %d", ErrInvalidBlockSize, blockSize))
	}
	return blockSize - length%blockSize
}

// Unpad unpads the given data by reading the padding length from the last byte of the data.
// This will also verify that each byte of padding is equal to the last byte.
// A non-error return value will always be a slice of the input data, never a copy.
func Unpad(data []byte) ([]byte, error) {
	return UnpadCustom(data, true, 255)
}

// UnpadCustom is a more flexible version of Unpad that ignores skipping validation of padding bytes
// and/or specifying a maximum block size that the padding must not exceed.
func UnpadCustom(data []byte, strict bool, maxBlockSize byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, ErrEmptyData
	}
	unpadding := data[length-1]
	if unpadding == 0 || int(unpadding) > length || unpadding > maxBlockSize {
		return nil, fmt.Errorf("%w: length %d", ErrInvalidPadding, unpadding)
	}
	if strict {
		for _, b := range data[length-int(unpadding) : length-1] {
			if b != unpadding {
				return nil, fmt.Errorf("%w: got byte %d (expected only %d)", ErrInvalidPadding, b, unpadding)
			}
		}
	}
	return data[:length-int(unpadding)], nil
}
