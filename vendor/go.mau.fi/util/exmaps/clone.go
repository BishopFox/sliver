// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exmaps

import (
	"maps"
)

// NonNilClone returns a copy of the given map, or an empty map if the input is nil.
func NonNilClone[Key comparable, Value any](m map[Key]Value) map[Key]Value {
	if m == nil {
		return make(map[Key]Value)
	}
	return maps.Clone(m)
}
