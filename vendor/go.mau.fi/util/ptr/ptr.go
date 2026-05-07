// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package ptr

// Clone creates a shallow copy of the given pointer.
func Clone[T any](val *T) *T {
	if val == nil {
		return nil
	}
	valCopy := *val
	return &valCopy
}

// Ptr returns a pointer to the given value.
func Ptr[T any](val T) *T {
	return &val
}

// NonZero returns a pointer to the given comparable value, unless the value is the type's zero value.
func NonZero[T comparable](val T) *T {
	var zero T
	return NonDefault(val, zero)
}

// NonDefault returns a pointer to the first parameter, unless it is equal to the second parameter.
func NonDefault[T comparable](val, def T) *T {
	if val == def {
		return nil
	}
	return &val
}

// Val returns the value of the given pointer, or the zero value of the type if the pointer is nil.
func Val[T any](ptr *T) (val T) {
	if ptr != nil {
		val = *ptr
	}
	return
}
