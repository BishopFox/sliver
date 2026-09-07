// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exslices

// Chunk splits a slice into chunks of the given size.
//
// From https://github.com/golang/go/issues/53987#issuecomment-1224367139
//
// TODO remove this after slices.Chunk can be used (it'll probably be added in Go 1.23, so it can be used after 1.22 is EOL)
func Chunk[T any](slice []T, size int) (chunks [][]T) {
	if size < 1 {
		panic("chunk size cannot be less than 1")
	}
	for i := 0; ; i++ {
		next := i * size
		if len(slice[next:]) > size {
			end := next + size
			chunks = append(chunks, slice[next:end:end])
		} else {
			chunks = append(chunks, slice[i*size:])
			return
		}
	}
}
