//go:build !tinygo

package amd64

import "reflect"

// setSliceLimits sets both Cap and Len for the given reflected slice.
func setSliceLimits(s *reflect.SliceHeader, limit uintptr) {
	s.Len = int(limit)
	s.Cap = int(limit)
}
