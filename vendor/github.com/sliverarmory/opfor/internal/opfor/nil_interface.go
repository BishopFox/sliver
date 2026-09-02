package opfor

import "reflect"

// isNilInterface recognizes nil values hidden behind an interface. Public
// option boundaries accept interface implementations, where a typed nil
// pointer or function would otherwise pass an ordinary value == nil check and
// panic only when the runtime later invokes it.
func isNilInterface(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
