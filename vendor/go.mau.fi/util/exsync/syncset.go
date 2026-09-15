// Copyright (c) 2023 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exsync

import (
	"iter"
	"sync"

	"go.mau.fi/util/exmaps"
)

type empty struct{}

var emptyVal = empty{}

// Set is a wrapper around a map[T]struct{} with a built-in mutex.
type Set[T comparable] struct {
	m map[T]empty
	l sync.RWMutex
}

var _ exmaps.AbstractSet[int] = (*Set[int])(nil)

// NewSet constructs a Set with an empty map.
func NewSet[T comparable]() *Set[T] {
	return NewSetWithMap[T](make(map[T]empty))
}

// NewSetWithSize constructs a Set with a map that has been allocated the given amount of space.
func NewSetWithSize[T comparable](size int) *Set[T] {
	return NewSetWithMap[T](make(map[T]empty, size))
}

// NewSetWithMap constructs a Set with the given map. Accessing the map directly after passing it here is not safe.
func NewSetWithMap[T comparable](m map[T]empty) *Set[T] {
	return &Set[T]{m: m}
}

// NewSetWithItems constructs a Set with items from the given slice pre-filled.
// The slice is not modified or used after the function returns, so using it after this is safe.
func NewSetWithItems[T comparable](items []T) *Set[T] {
	s := NewSetWithSize[T](len(items))
	for _, item := range items {
		s.m[item] = emptyVal
	}
	return s
}

// Add adds an item to the set. The return value is true if the item was added to the set, or false otherwise.
func (s *Set[T]) Add(item T) bool {
	if s == nil {
		return false
	}
	s.l.Lock()
	_, exists := s.m[item]
	if exists {
		s.l.Unlock()
		return false
	}
	s.m[item] = emptyVal
	s.l.Unlock()
	return true
}

func (s *Set[T]) AddSeq(seq iter.Seq[T]) {
	if s == nil {
		return
	}
	s.l.Lock()
	defer s.l.Unlock()
	for item := range seq {
		s.m[item] = emptyVal
	}
}

// Has checks if the given item is in the set.
func (s *Set[T]) Has(item T) bool {
	if s == nil {
		return false
	}
	s.l.RLock()
	_, exists := s.m[item]
	s.l.RUnlock()
	return exists
}

// Pop removes the given item from the set. The return value is true if the item was in the set, or false otherwise.
func (s *Set[T]) Pop(item T) bool {
	if s == nil {
		return false
	}
	s.l.Lock()
	_, exists := s.m[item]
	if exists {
		delete(s.m, item)
	}
	s.l.Unlock()
	return exists
}

// Remove removes the given item from the set.
func (s *Set[T]) Remove(item T) {
	if s == nil {
		return
	}
	s.l.Lock()
	delete(s.m, item)
	s.l.Unlock()
}

// ReplaceAll replaces this set with the given set. If the given set is nil, the set is cleared.
func (s *Set[T]) ReplaceAll(newSet *Set[T]) {
	if s == nil {
		return
	}
	s.l.Lock()
	if newSet == nil {
		s.m = make(map[T]empty)
	} else {
		s.m = newSet.m
	}
	s.l.Unlock()
}

func (s *Set[T]) Clear() {
	if s == nil {
		return
	}
	s.l.Lock()
	clear(s.m)
	s.l.Unlock()
}

func (s *Set[T]) Size() int {
	if s == nil {
		return 0
	}
	s.l.RLock()
	size := len(s.m)
	s.l.RUnlock()
	return size
}

func (s *Set[T]) AsList() []T {
	if s == nil {
		return nil
	}
	s.l.RLock()
	list := make([]T, len(s.m))
	i := 0
	for item := range s.m {
		list[i] = item
		i++
	}
	s.l.RUnlock()
	return list
}

func (s *Set[T]) Iter() iter.Seq[T] {
	return func(yield func(T) bool) {
		s.l.RLock()
		defer s.l.RUnlock()
		for item := range s.m {
			if !yield(item) {
				return
			}
		}
	}
}
