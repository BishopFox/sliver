// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exslices

import (
	"iter"
)

type Stack[T comparable] []T

func (s *Stack[T]) Push(v ...T) {
	*s = append(*s, v...)
}

// Peek returns the top item from the stack without removing it.
func (s *Stack[T]) Peek() (v T, ok bool) {
	return s.PeekN(1)
}

// PeekN returns the nth item from the top of the stack (1-based).
func (s *Stack[T]) PeekN(n int) (v T, ok bool) {
	if len(*s) < n {
		return
	}
	v = (*s)[len(*s)-n]
	ok = true
	return
}

// Pop removes and returns the top item from the stack.
func (s *Stack[T]) Pop() (v T, ok bool) {
	v, ok = s.Peek()
	if ok {
		*s = (*s)[:len(*s)-1]
	}
	return
}

func (s *Stack[T]) PeekValue() T {
	v, _ := s.Peek()
	return v
}

func (s *Stack[T]) PopValue() T {
	v, _ := s.Pop()
	return v
}

// Index returns the highest index of the given value in the stack, or -1 if not found.
func (s *Stack[T]) Index(val T) int {
	for i := len(*s) - 1; i >= 0; i-- {
		if (*s)[i] == val {
			return i
		}
	}
	return -1
}

// Has returns whether the given value is in the stack.
func (s *Stack[T]) Has(val T) bool {
	return s.Index(val) != -1
}

func (s *Stack[T]) PopIter() iter.Seq[T] {
	return func(yield func(T) bool) {
		for {
			v, ok := s.Pop()
			if !ok {
				return
			}
			if !yield(v) {
				return
			}
		}
	}
}
