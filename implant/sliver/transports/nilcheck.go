package transports

import "reflect"

// isNilInterface returns true if v is nil, or if v is an interface value holding a nil pointer.
func isNilInterface(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.Interface, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}
