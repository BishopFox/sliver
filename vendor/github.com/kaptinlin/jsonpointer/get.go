package jsonpointer

import (
	"reflect"
)

// fastGet implements ultra-fast path that avoids token allocation entirely.
// Optimized for string-only Path - direct access without intermediate token creation.
func fastGet(val any, step string) (any, bool) {
	switch v := val.(type) {
	case map[string]any:
		// Most common case: map[string]any - direct string key access
		result, exists := v[step]
		return result, exists

	case *map[string]any:
		// Pointer to map optimization
		if v == nil {
			return nil, false
		}
		result, exists := (*v)[step]
		return result, exists

	case []any:
		// Array access - parse index from string
		if step == "-" {
			return nil, false // array end marker
		}
		index := fastAtoi(step)
		if index < 0 || index >= len(v) {
			return nil, false // invalid or out of bounds
		}
		return v[index], true

	case *[]any:
		// Pointer to slice optimization
		if v == nil {
			return nil, false
		}
		if step == "-" {
			return nil, false // array end marker
		}
		index := fastAtoi(step)
		if index < 0 || index >= len(*v) {
			return nil, false // invalid or out of bounds
		}
		return (*v)[index], true

	case *any:
		// Interface pointer - recurse once
		if v == nil {
			return nil, false
		}
		return fastGet(*v, step)

	default:
		// Fast path failed, need reflection fallback
		return nil, false
	}
}

// getTokenAtIndex computes an internalToken for a specific path step without allocating a slice.
// Optimized for string-only Path - avoid allocating entire tokens slice, compute on-demand.
func getTokenAtIndex(path Path, index int) internalToken {
	if index >= len(path) {
		return internalToken{}
	}

	step := path[index] // step is already a string
	return internalToken{
		key:   step,
		index: fastAtoi(step),
	}
}

// tryArrayAccess attempts array access using type assertions for performance.
// Enhanced to handle all slice types efficiently.
func tryArrayAccess(current any, token internalToken) (any, bool, error) {
	// Fast type assertion path for common slice types
	switch arr := current.(type) {
	case []any:
		index, err := validateAndAccessArray(token.key, len(arr))
		if err != nil {
			return nil, true, err
		}
		return arr[index], true, nil

	case *[]any:
		if arr == nil {
			return nil, true, ErrNilPointer
		}
		index, err := validateAndAccessArray(token.key, len(*arr))
		if err != nil {
			return nil, true, err
		}
		return (*arr)[index], true, nil

	default:
		// Fallback to reflection for other array types (like []User, native arrays, and pointers to arrays)
		arrayVal, err := derefValue(reflect.ValueOf(current))
		if err != nil {
			return nil, true, err
		}

		// Check if the dereferenced value is an array/slice
		if arrayVal.Kind() != reflect.Slice && arrayVal.Kind() != reflect.Array {
			return nil, false, nil
		}

		index, err := validateAndAccessArray(token.key, arrayVal.Len())
		if err != nil {
			return nil, true, err
		}
		return arrayVal.Index(index).Interface(), true, nil
	}
}

// tryObjectAccess attempts object access using type assertions for performance.
// Enhanced with proper struct field handling.
func tryObjectAccess(current any, token internalToken) (any, bool, error) {
	// Fast type assertion path for common map types
	switch obj := current.(type) {
	case map[string]any:
		result, exists := obj[token.key]
		if !exists {
			return nil, true, ErrKeyNotFound // Key doesn't exist
		}
		return result, true, nil

	case *map[string]any:
		if obj == nil {
			return nil, true, ErrNilPointer
		}
		result, exists := (*obj)[token.key]
		if !exists {
			return nil, true, ErrKeyNotFound // Key doesn't exist
		}
		return result, true, nil

	default:
		// Fallback to reflection for other object types
		objVal, err := derefValue(reflect.ValueOf(current))
		if err != nil {
			return nil, false, err
		}

		switch objVal.Kind() { //nolint:exhaustive
		case reflect.Map:
			mapKey := reflect.ValueOf(token.key)
			mapVal := objVal.MapIndex(mapKey)
			if !mapVal.IsValid() {
				return nil, true, ErrKeyNotFound // Key doesn't exist
			}
			return mapVal.Interface(), true, nil
		case reflect.Struct:
			// Handle struct fields using optimized struct field lookup
			if field := findStructField(objVal, token.key); field.IsValid() {
				return field.Interface(), true, nil
			}
			return nil, true, ErrFieldNotFound // Field not found in struct

		default:
			// Handle all other reflect.Kind types not supported for JSON Pointer traversal
			return nil, false, nil
		}
	}
}

// get retrieves value at JSON pointer path, returns error if path cannot be traversed.
// Optimized for zero-allocation string-only paths with layered fallback strategy.
func get(val any, path Path) (any, error) {
	pathLength := len(path)
	if pathLength == 0 {
		return val, nil
	}

	// Zero-allocation fast path for common cases
	current := val
	fastPathDepth := 0

	// Ultra-fast path - direct access without token creation
	for i := 0; i < pathLength; i++ {
		step := path[i] // step is already a string

		// Try direct fast path first (zero allocations for map[string]any)
		if result, ok := fastGet(current, step); ok {
			current = result
			fastPathDepth = i + 1
		} else {
			// Direct fast path failed, break to optimized type assertion fallback
			break
		}
	}

	// Optimized type assertion fallback for remaining path (if any)
	if fastPathDepth < pathLength {
		// Use optimized type assertions for the remaining tokens
		for i := fastPathDepth; i < pathLength; i++ {
			// Compute token on-demand only when needed
			token := getTokenAtIndex(path, i)

			if current == nil {
				return nil, ErrNotFound
			}

			// Try optimized array access first
			if result, handled, err := tryArrayAccess(current, token); err != nil {
				return nil, err
			} else if handled {
				current = result
				continue
			}

			// Try optimized object access
			if result, handled, err := tryObjectAccess(current, token); err != nil {
				return nil, err
			} else if handled {
				current = result
				continue
			}

			// Neither array nor object, can't traverse further
			return nil, ErrNotFound
		}
	}

	return current, nil
}

// findStructField finds a struct field by JSON tag or field name using cached field mapping.
// Returns the field value if found, invalid reflect.Value otherwise.
func findStructField(structVal reflect.Value, key string) reflect.Value {
	// Use cached struct field mapping from struct.go
	fields := getStructFields(structVal.Type())
	if fieldIndex, ok := fields[key]; ok {
		return structVal.Field(fieldIndex)
	}
	return reflect.Value{} // Not found
}
