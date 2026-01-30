// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exmaps

import (
	"iter"
)

// AbstractSet is an interface implemented by [Set] and [exsync.Set]
type AbstractSet[T comparable] interface {
	Add(item T) bool
	Has(item T) bool
	Pop(item T) bool
	Remove(item T)
	Size() int
	AsList() []T
	Iter() iter.Seq[T]
}

type empty = struct{}

var emptyVal = empty{}

// Set is a generic set type based on a map. It is not thread-safe, use [exsync.Set] for a thread-safe variant.
type Set[T comparable] map[T]empty

var _ AbstractSet[int] = (Set[int])(nil)

func NewSetWithItems[T comparable](items []T) Set[T] {
	s := make(Set[T], len(items))
	for _, item := range items {
		s[item] = emptyVal
	}
	return s
}

func (s Set[T]) Size() int {
	return len(s)
}

func (s Set[T]) Add(item T) bool {
	_, exists := s[item]
	if !exists {
		s[item] = emptyVal
		return true
	}
	return false
}

func (s Set[T]) Remove(item T) {
	delete(s, item)
}

func (s Set[T]) Pop(item T) bool {
	_, exists := s[item]
	if exists {
		delete(s, item)
	}
	return exists
}

func (s Set[T]) Has(item T) bool {
	_, exists := s[item]
	return exists
}

func (s Set[T]) AsList() []T {
	result := make([]T, len(s))
	i := 0
	for item := range s {
		result[i] = item
		i++
	}
	return result
}

func (s Set[T]) Iter() iter.Seq[T] {
	return func(yield func(T) bool) {
		for item := range s {
			if !yield(item) {
				return
			}
		}
	}
}
