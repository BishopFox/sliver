// Copyright (c) 2023 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exsync

import (
	"errors"
	"sync"
)

type pair[Key comparable, Value any] struct {
	Set   bool
	Key   Key
	Value Value
}

type RingBuffer[Key comparable, Value any] struct {
	ptr  int
	data []pair[Key, Value]
	lock sync.RWMutex
	size int
}

func NewRingBuffer[Key comparable, Value any](size int) *RingBuffer[Key, Value] {
	return &RingBuffer[Key, Value]{
		data: make([]pair[Key, Value], size),
	}
}

var (
	// StopIteration can be returned by the RingBuffer.Iter or MapRingBuffer callbacks to stop iteration immediately.
	StopIteration = errors.New("stop iteration") //lint:ignore ST1012 not an error

	// SkipItem can be returned by the MapRingBuffer callback to skip adding a specific item.
	SkipItem = errors.New("skip item") //lint:ignore ST1012 not an error
)

func (rb *RingBuffer[Key, Value]) unlockedIter(callback func(key Key, val Value) error) error {
	end := rb.ptr
	for i := clamp(end-1, len(rb.data)); i != end; i = clamp(i-1, len(rb.data)) {
		entry := rb.data[i]
		if !entry.Set {
			break
		}
		err := callback(entry.Key, entry.Value)
		if err != nil {
			if errors.Is(err, StopIteration) {
				return nil
			}
			return err
		}
	}
	return nil
}

func (rb *RingBuffer[Key, Value]) Iter(callback func(key Key, val Value) error) error {
	rb.lock.RLock()
	defer rb.lock.RUnlock()
	return rb.unlockedIter(callback)
}

func MapRingBuffer[Key comparable, Value, Output any](rb *RingBuffer[Key, Value], callback func(key Key, val Value) (Output, error)) ([]Output, error) {
	rb.lock.RLock()
	defer rb.lock.RUnlock()
	output := make([]Output, 0, rb.size)
	err := rb.unlockedIter(func(key Key, val Value) error {
		item, err := callback(key, val)
		if err != nil {
			if errors.Is(err, SkipItem) {
				return nil
			}
			return err
		}
		output = append(output, item)
		return nil
	})
	return output, err
}

func (rb *RingBuffer[Key, Value]) Size() int {
	rb.lock.RLock()
	defer rb.lock.RUnlock()
	return rb.size
}

func (rb *RingBuffer[Key, Value]) Contains(val Key) bool {
	_, ok := rb.Get(val)
	return ok
}

func (rb *RingBuffer[Key, Value]) Get(key Key) (val Value, found bool) {
	rb.lock.RLock()
	end := rb.ptr
	for i := clamp(end-1, len(rb.data)); i != end; i = clamp(i-1, len(rb.data)) {
		if rb.data[i].Key == key {
			val = rb.data[i].Value
			found = true
			break
		}
	}
	rb.lock.RUnlock()
	return
}

func (rb *RingBuffer[Key, Value]) Replace(key Key, val Value) bool {
	rb.lock.Lock()
	defer rb.lock.Unlock()
	end := rb.ptr
	for i := clamp(end-1, len(rb.data)); i != end; i = clamp(i-1, len(rb.data)) {
		if rb.data[i].Key == key {
			rb.data[i].Value = val
			return true
		}
	}
	return false
}

func (rb *RingBuffer[Key, Value]) Push(key Key, val Value) {
	rb.lock.Lock()
	rb.data[rb.ptr] = pair[Key, Value]{Key: key, Value: val, Set: true}
	rb.ptr = (rb.ptr + 1) % len(rb.data)
	if rb.size < len(rb.data) {
		rb.size++
	}
	rb.lock.Unlock()
}

func clamp(index, len int) int {
	if index < 0 {
		return len + index
	} else if index >= len {
		return len - index
	} else {
		return index
	}
}
