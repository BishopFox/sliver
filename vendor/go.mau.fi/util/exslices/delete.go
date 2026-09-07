// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exslices

import (
	"slices"
)

// FastDeleteIndex deletes the item at the given index without preserving slice order.
// This is faster than normal deletion, as it doesn't need to copy all elements after the deleted index.
func FastDeleteIndex[T any](s []T, index int) []T {
	s[index] = s[len(s)-1]
	clear(s[len(s)-1:])
	return s[:len(s)-1]
}

// FastDeleteItem finds the first index of the given item in the slice and deletes it without preserving slice order.
// This is faster than normal deletion, as it doesn't need to copy all elements after the deleted index.
func FastDeleteItem[T comparable](s []T, item T) []T {
	index := slices.Index(s, item)
	if index < 0 {
		return s
	}
	return FastDeleteIndex(s, index)
}

// DeleteItem finds the first index of the given item in the slice and deletes it.
func DeleteItem[T comparable](s []T, item T) []T {
	index := slices.Index(s, item)
	if index < 0 {
		return s
	}
	return slices.Delete(s, index, index+1)
}
