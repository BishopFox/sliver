// Copyright (c) 2023 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exslices

// SortedDiff returns the difference between two already-sorted slices, with the help of the given comparison function.
// The output will be in the same order as the input, which means it'll be sorted.
func SortedDiff[T any](a, b []T, compare func(a, b T) int) (uniqueToA, uniqueToB []T) {
	uniqueToA = make([]T, 0, len(a))
	uniqueToB = make([]T, 0, len(b))

	var i, j int
	for {
		if j >= len(b) {
			uniqueToA = append(uniqueToA, a[i:]...)
			break
		} else if i >= len(a) {
			uniqueToB = append(uniqueToB, b[j:]...)
			break
		}
		c := compare(a[i], b[j])
		if c < 0 {
			uniqueToA = append(uniqueToA, a[i])
			i++
		} else if c > 0 {
			uniqueToB = append(uniqueToB, b[j])
			j++
		} else {
			i++
			j++
		}
	}
	return
}

// Diff returns the difference between two slices. The slices may contain duplicates and don't need to be sorted.
// The output will not be sorted, but is guaranteed to not contain any duplicates.
func Diff[T comparable](a, b []T) (uniqueToA, uniqueToB []T) {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	collector := make(map[T]uint8, maxLen)
	for _, item := range a {
		collector[item] |= 0b01
	}
	for _, item := range b {
		collector[item] |= 0b10
	}
	uniqueToA = make([]T, 0, maxLen)
	uniqueToB = make([]T, 0, maxLen)
	for item, mask := range collector {
		if mask == 0b01 {
			uniqueToA = append(uniqueToA, item)
		} else if mask == 0b10 {
			uniqueToB = append(uniqueToB, item)
		}
	}
	return
}
