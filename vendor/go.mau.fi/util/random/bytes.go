// Copyright (c) 2023 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package random

import (
	"crypto/rand"
)

// Bytes generates the given amount of random bytes using crypto/rand, and panics if it fails.
func Bytes(n int) []byte {
	data := make([]byte, n)
	_, err := rand.Read(data)
	if err != nil {
		panic(err)
	}
	return data
}
