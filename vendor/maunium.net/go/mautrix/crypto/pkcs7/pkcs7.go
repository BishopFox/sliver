// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package pkcs7

import "bytes"

// Pad implements PKCS#7 padding as defined in [RFC2315]. It pads the data to
// the given blockSize in the range [1, 255]. This is normally used in AES-CBC
// encryption.
//
// [RFC2315]: https://www.ietf.org/rfc/rfc2315.txt
func Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	return append(data, bytes.Repeat([]byte{byte(padding)}, padding)...)
}

// Unpad implements PKCS#7 unpadding as defined in [RFC2315]. It unpads the
// data by reading the padding amount from the last byte of the data. This is
// normally used in AES-CBC decryption.
//
// [RFC2315]: https://www.ietf.org/rfc/rfc2315.txt
func Unpad(data []byte) []byte {
	length := len(data)
	unpadding := int(data[length-1])
	return data[:length-unpadding]
}
