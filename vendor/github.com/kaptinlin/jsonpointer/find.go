package jsonpointer

import (
	"reflect"
)

// find locates a reference in document using string path components.
// Optimized with inline fast paths and minimal allocations.
func find(val any, path Path) (*Reference, error) {
	pathLength := len(path)
	if pathLength == 0 {
		return &Reference{Val: val}, nil
	}

	var obj any
	var key string
	current := val

	for i := 0; i < pathLength; i++ {
		obj = current
		key = path[i] // key is already a string

		if current == nil {
			return nil, ErrNotFound
		}

		// Inline ultra-fast path - avoid function call overhead
		switch v := current.(type) {
		case map[string]any:
			// Most common case: map[string]any - direct string key access
			if result, exists := v[key]; exists {
				current = result
			} else {
				return nil, ErrKeyNotFound
			}

		case *map[string]any:
			// Pointer to map optimization
			if v == nil {
				return nil, ErrNilPointer
			}
			if result, exists := (*v)[key]; exists {
				current = result
			} else {
				return nil, ErrKeyNotFound
			}

		case []any:
			// Array access - optimized inline parsing
			index, err := validateAndAccessArray(key, len(v))
			if err != nil {
				return nil, err
			}
			current = v[index]

		case *[]any:
			// Pointer to slice optimization
			if v == nil {
				return nil, ErrNilPointer
			}
			index, err := validateAndAccessArray(key, len(*v))
			if err != nil {
				return nil, err
			}
			current = (*v)[index]

		default:
			// Reflection fallback for other types
			objVal, err := derefValue(reflect.ValueOf(current))
			if err != nil {
				return nil, err
			}

			switch objVal.Kind() {
			case reflect.Slice, reflect.Array:
				// Array access using reflection
				index, err := validateAndAccessArray(key, objVal.Len())
				if err != nil {
					return nil, err
				}
				current = objVal.Index(index).Interface()

			case reflect.Map:
				// Map access using reflection
				mapKey := reflect.ValueOf(key)
				mapVal := objVal.MapIndex(mapKey)
				if mapVal.IsValid() {
					current = mapVal.Interface()
				} else {
					return nil, ErrKeyNotFound
				}

			case reflect.Struct:
				// Struct field access using reflection
				if structField(key, &objVal) {
					current = objVal.Interface()
				} else {
					return nil, ErrFieldNotFound
				}

			case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
				reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
				reflect.Chan, reflect.Func, reflect.Interface, reflect.Ptr, reflect.String, reflect.UnsafePointer:
				// Handle all other reflect.Kind types not supported for JSON Pointer traversal
				return nil, ErrNotFound
			}
		}
	}

	return &Reference{Val: current, Obj: obj, Key: key}, nil
}
