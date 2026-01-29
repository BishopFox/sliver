// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exbytes

import (
	"unsafe"
)

// UnsafeString returns a string that points to the same memory as the input byte slice.
//
// The input byte slice must not be modified after this function is called.
//
// See [go.mau.fi/util/exstrings.UnsafeBytes] for the reverse operation.
func UnsafeString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
