// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exslices

// DeduplicateUnsorted removes duplicates from the given slice without requiring that the input slice is sorted.
// The order of the output will be the same as the input. The input slice will not be modified.
//
// If you don't care about the order of the output, it's recommended to sort the list and then use [slices.Compact].
func DeduplicateUnsorted[T comparable](s []T) []T {
	return deduplicateUnsortedInto(s, make([]T, 0, len(s)))
}

// DeduplicateUnsortedOverwrite removes duplicates from the given slice without requiring that the input slice is sorted.
// The input slice will be modified and used as the output slice to avoid extra allocations.
//
// If you don't care about the order of the output, it's recommended to sort the list and then use [slices.Compact].
func DeduplicateUnsortedOverwrite[T comparable](s []T) []T {
	out := deduplicateUnsortedInto(s, s[:0])
	clear(s[len(out):])
	return out
}

func deduplicateUnsortedInto[T comparable](s, result []T) []T {
	seen := make(map[T]struct{}, len(s))
	for _, item := range s {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// DeduplicateUnsortedFunc removes duplicates from the given slice using the given key function without requiring
// that the input slice is sorted. The order of the output will be the same as the input.
//
// If you don't care about the order of the output, it's recommended to sort the list and then use [slices.CompactFunc].
func DeduplicateUnsortedFunc[T any, K comparable](s []T, getKey func(T) K) []T {
	return deduplicateUnsortedFuncInto(s, make([]T, 0, len(s)), getKey)
}

// DeduplicateUnsortedOverwriteFunc removes duplicates from the given slice using the given key function
// without requiring that the input slice is sorted. The order of the output will be the same as the input.
// The input slice will be modified and used as the output slice to avoid extra allocations.
//
// If you don't care about the order of the output, it's recommended to sort the list and then use [slices.CompactFunc].
func DeduplicateUnsortedOverwriteFunc[T any, K comparable](s []T, getKey func(T) K) []T {
	out := deduplicateUnsortedFuncInto(s, s[:0], getKey)
	clear(s[len(out):])
	return out
}

func deduplicateUnsortedFuncInto[T any, K comparable](s, result []T, getKey func(T) K) []T {
	seen := make(map[K]struct{}, len(s))
	for _, item := range s {
		key := getKey(item)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}
