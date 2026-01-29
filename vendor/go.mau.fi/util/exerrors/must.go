// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exerrors

func Must[T any](val T, err error) T {
	PanicIfNotNil(err)
	return val
}

func Must2[T any, T2 any](val T, val2 T2, err error) (T, T2) {
	PanicIfNotNil(err)
	return val, val2
}

func PanicIfNotNil(err error) {
	if err != nil {
		panic(err)
	}
}
