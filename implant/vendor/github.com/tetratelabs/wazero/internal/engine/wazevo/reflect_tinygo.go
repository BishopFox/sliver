//go:build tinygo

package wazevo

import "reflect"

// setSliceLimits sets both Cap and Len for the given reflected slice.
func setSliceLimits(s *reflect.SliceHeader, l, c uintptr) {
	s.Len = l
	s.Cap = c
}
