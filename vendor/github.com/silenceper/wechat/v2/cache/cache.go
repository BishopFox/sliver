package cache

import (
	"context"
	"time"
)

// Cache interface
type Cache interface {
	Get(key string) interface{}
	Set(key string, val interface{}, timeout time.Duration) error
	IsExist(key string) bool
	Delete(key string) error
}

// ContextCache interface
type ContextCache interface {
	Cache
	GetContext(ctx context.Context, key string) interface{}
	SetContext(ctx context.Context, key string, val interface{}, timeout time.Duration) error
	IsExistContext(ctx context.Context, key string) bool
	DeleteContext(ctx context.Context, key string) error
}

// GetContext get value from cache
func GetContext(ctx context.Context, cache Cache, key string) interface{} {
	if cache, ok := cache.(ContextCache); ok {
		return cache.GetContext(ctx, key)
	}
	return cache.Get(key)
}

// SetContext set value to cache
func SetContext(ctx context.Context, cache Cache, key string, val interface{}, timeout time.Duration) error {
	if cache, ok := cache.(ContextCache); ok {
		return cache.SetContext(ctx, key, val, timeout)
	}
	return cache.Set(key, val, timeout)
}

// IsExistContext check value exists in cache.
func IsExistContext(ctx context.Context, cache Cache, key string) bool {
	if cache, ok := cache.(ContextCache); ok {
		return cache.IsExistContext(ctx, key)
	}
	return cache.IsExist(key)
}

// DeleteContext delete value in cache.
func DeleteContext(ctx context.Context, cache Cache, key string) error {
	if cache, ok := cache.(ContextCache); ok {
		return cache.DeleteContext(ctx, key)
	}
	return cache.Delete(key)
}
