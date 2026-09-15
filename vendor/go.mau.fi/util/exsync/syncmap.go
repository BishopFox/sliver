// Copyright (c) 2023 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exsync

import (
	"iter"
	"maps"
	"sync"
)

// Map is a simple map with a built-in mutex.
type Map[Key comparable, Value any] struct {
	data map[Key]Value
	lock sync.RWMutex
}

func NewMap[Key comparable, Value any]() *Map[Key, Value] {
	return NewMapWithData(make(map[Key]Value))
}

// NewMapWithData constructs a Map with the given map as data. Accessing the map directly after passing it here is not safe.
func NewMapWithData[Key comparable, Value any](data map[Key]Value) *Map[Key, Value] {
	return &Map[Key, Value]{
		data: data,
	}
}

// Set stores a value in the map.
func (sm *Map[Key, Value]) Set(key Key, value Value) {
	sm.Swap(key, value)
}

// Swap sets a value in the map and returns the old value.
//
// The boolean return parameter is true if the value already existed, false if not.
func (sm *Map[Key, Value]) Swap(key Key, value Value) (oldValue Value, wasReplaced bool) {
	sm.lock.Lock()
	oldValue, wasReplaced = sm.data[key]
	sm.data[key] = value
	sm.lock.Unlock()
	return
}

// Delete removes a key from the map.
func (sm *Map[Key, Value]) Delete(key Key) {
	sm.Pop(key)
}

// Pop removes a key from the map and returns the old value.
//
// The boolean return parameter is the same as with normal Go map access (true if the key exists, false if not).
func (sm *Map[Key, Value]) Pop(key Key) (value Value, ok bool) {
	sm.lock.Lock()
	value, ok = sm.data[key]
	delete(sm.data, key)
	sm.lock.Unlock()
	return
}

// Get gets a value in the map.
//
// The boolean return parameter is the same as with normal Go map access (true if the key exists, false if not).
func (sm *Map[Key, Value]) Get(key Key) (value Value, ok bool) {
	sm.lock.RLock()
	value, ok = sm.data[key]
	sm.lock.RUnlock()
	return
}

// GetDefault gets a value in the map, or the given default value if there's no such key.
func (sm *Map[Key, Value]) GetDefault(key Key, def Value) Value {
	sm.lock.RLock()
	value, ok := sm.data[key]
	sm.lock.RUnlock()
	if ok {
		return value
	}
	return def
}

// GetOrSetFactory gets a value in the map if the key already exists,
// otherwise inserts a new value from the given function and returns it.
func (sm *Map[Key, Value]) GetOrSetFactory(key Key, def func() Value) Value {
	val, ok := sm.Get(key)
	if ok {
		return val
	}
	sm.lock.Lock()
	defer sm.lock.Unlock()
	value, ok := sm.data[key]
	if !ok {
		value = def()
		sm.data[key] = value
	}
	return value
}

// GetOrSet gets a value in the map if the key already exists, otherwise inserts the given value and returns it.
//
// The boolean return parameter is true if the key already exists, and false if the given value was inserted.
func (sm *Map[Key, Value]) GetOrSet(key Key, value Value) (actual Value, wasGet bool) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	actual, wasGet = sm.data[key]
	if wasGet {
		return
	}
	sm.data[key] = value
	actual = value
	return
}

// Has returns true if the map contains the given key.
func (sm *Map[Key, Value]) Has(key Key) bool {
	sm.lock.RLock()
	_, ok := sm.data[key]
	sm.lock.RUnlock()
	return ok
}

// Clear removes all items from the map.
func (sm *Map[Key, Value]) Clear() {
	sm.lock.Lock()
	clear(sm.data)
	sm.lock.Unlock()
}

// Len returns the number of items in the map.
func (sm *Map[Key, Value]) Len() int {
	sm.lock.RLock()
	l := len(sm.data)
	sm.lock.RUnlock()
	return l
}

// CopyFrom copies all key/value pairs from the given map into this map, overriding any existing keys.
// Keys present in this map but not in the given map are not removed.
func (sm *Map[Key, Value]) CopyFrom(other map[Key]Value) {
	sm.lock.Lock()
	maps.Copy(sm.data, other)
	sm.lock.Unlock()
}

// SwapData replaces the internal map with the given map, returning the previous map.
// If the given map is nil, a new empty map is created.
// The given map must not be modified after passing it to this function.
func (sm *Map[Key, Value]) SwapData(other map[Key]Value) map[Key]Value {
	sm.lock.Lock()
	prev := sm.data
	if other == nil {
		sm.data = make(map[Key]Value)
	} else {
		sm.data = other
	}
	sm.lock.Unlock()
	return prev
}

// Clone returns a copy of the map.
func (sm *Map[Key, Value]) Clone() *Map[Key, Value] {
	return NewMapWithData(sm.CopyData())
}

// CopyData returns a copy of the data in the map as a normal (non-atomic) map.
func (sm *Map[Key, Value]) CopyData() map[Key]Value {
	sm.lock.RLock()
	copied := maps.Clone(sm.data)
	sm.lock.RUnlock()
	return copied
}

func (sm *Map[Key, Value]) Iter() iter.Seq2[Key, Value] {
	return func(yield func(Key, Value) bool) {
		sm.lock.RLock()
		defer sm.lock.RUnlock()
		for k, v := range sm.data {
			if !yield(k, v) {
				return
			}
		}
	}
}
