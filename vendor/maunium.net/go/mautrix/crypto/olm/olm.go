// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package olm

var GetVersion func() (major, minor, patch uint8)
var SetPickleKeyImpl func(key []byte)

// Version returns the version number of the olm library.
func Version() (major, minor, patch uint8) {
	return GetVersion()
}

// SetPickleKey sets the global pickle key used when encoding structs with Gob or JSON.
func SetPickleKey(key []byte) {
	SetPickleKeyImpl(key)
}
