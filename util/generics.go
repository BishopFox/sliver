package util

import (
	"maps"
	"slices"
)

// Contains reports whether v is present in elements.
//
// Deprecated: use [slices.Contains] directly.
func Contains[T comparable](elements []T, v T) bool {
	return slices.Contains(elements, v)
}

// Keys returns the keys in m in an unspecified order.
//
// Deprecated: use [maps.Keys] directly, collecting the iterator when a slice
// is required.
func Keys[M ~map[K]V, K comparable, V any](m M) []K {
	return slices.Collect(maps.Keys(m))
}
