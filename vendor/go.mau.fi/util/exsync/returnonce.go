// Copyright (c) 2023 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exsync

import "sync"

// ReturnableOnce is a wrapper for sync.Once that can return a value
//
// Deprecated: Use [sync.OnceValues] instead.
type ReturnableOnce[Value any] struct {
	once   sync.Once
	output Value
	err    error
}

func (ronce *ReturnableOnce[Value]) Do(fn func() (Value, error)) (Value, error) {
	ronce.once.Do(func() {
		ronce.output, ronce.err = fn()
	})
	return ronce.output, ronce.err
}
