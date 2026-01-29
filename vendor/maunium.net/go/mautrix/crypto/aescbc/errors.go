// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package aescbc

import "errors"

var (
	ErrNoKeyProvided        = errors.New("no key")
	ErrIVNotBlockSize       = errors.New("IV length does not match AES block size")
	ErrNotMultipleBlockSize = errors.New("ciphertext length is not a multiple of the AES block size")
)
