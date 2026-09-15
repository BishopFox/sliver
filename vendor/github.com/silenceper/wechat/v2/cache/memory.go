package cache

import (
	"sync"
	"time"
)

// Memory is an concurrent-safe in-process cache backed by a plain map.
type Memory struct {
	mu   sync.RWMutex
	data map[string]*data
}

// data holds a single cached value together with its expiry time.
type data struct {
	Data    interface{}
	Expired time.Time
}

// NewMemory returns a ready-to-use in-memory cache.
func NewMemory() *Memory {
	return &Memory{
		data: map[string]*data{},
	}
}

// Get returns the cached value for key.
// Returns nil if the key does not exist or has expired.
func (mem *Memory) Get(key string) interface{} {
	mem.mu.RLock()
	ret, ok := mem.data[key]
	expired := ok && ret.Expired.Before(time.Now())
	mem.mu.RUnlock()

	if !ok {
		return nil
	}
	if expired {
		mem.deleteExpiredKey(key)
		return nil
	}
	return ret.Data
}

// IsExist check value exists in memcache.
func (mem *Memory) IsExist(key string) bool {
	mem.mu.RLock()
	ret, ok := mem.data[key]
	expired := ok && ret.Expired.Before(time.Now())
	mem.mu.RUnlock()

	if !ok {
		return false
	}
	if expired {
		mem.deleteExpiredKey(key)
		return false
	}
	return true
}

// Set cached value with key and expire time.
func (mem *Memory) Set(key string, val interface{}, timeout time.Duration) (err error) {
	mem.mu.Lock()
	defer mem.mu.Unlock()

	mem.data[key] = &data{
		Data:    val,
		Expired: time.Now().Add(timeout),
	}
	return nil
}

// Delete delete value in memcache.
func (mem *Memory) Delete(key string) error {
	mem.mu.Lock()
	defer mem.mu.Unlock()
	delete(mem.data, key)
	return nil
}

// deleteExpiredKey deletes a key only if it has expired.
// Caller must NOT hold any lock.
func (mem *Memory) deleteExpiredKey(key string) {
	mem.mu.Lock()
	defer mem.mu.Unlock()
	if d, ok := mem.data[key]; ok && d.Expired.Before(time.Now()) {
		delete(mem.data, key)
	}
}
