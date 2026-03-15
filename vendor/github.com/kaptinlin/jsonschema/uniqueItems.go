package jsonschema

import (
	"encoding/binary"
	"fmt"
	"hash/maphash"
	"math"
	"reflect"
	"slices"
	"strings"
)

// evaluateUniqueItems checks if all elements in the array are unique when the "uniqueItems" property is set to true.
// According to the JSON Schema Draft 2020-12:
//   - If "uniqueItems" is false, the data always validates successfully.
//   - If "uniqueItems" is true, the data validates successfully only if all elements in the array are unique.
//
// This implementation uses hash-based comparison for O(n) average complexity.
// Each item is hashed using maphash, and deep equality is only checked on hash collisions.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-uniqueitems
func evaluateUniqueItems(schema *Schema, data []any) *EvaluationError {
	// If uniqueItems is false or not set, no validation is needed
	if schema.UniqueItems == nil || !*schema.UniqueItems {
		return nil
	}

	// Determine the array length to validate
	maxLength := len(data)

	// If items is false, only validate items defined by prefixItems
	if schema.Items != nil && schema.Items.Boolean != nil && !*schema.Items.Boolean {
		if schema.PrefixItems != nil {
			maxLength = min(len(schema.PrefixItems), len(data))
		} else {
			maxLength = 0
		}
	}

	// If there are 0 or 1 items, they are always unique
	if maxLength <= 1 {
		return nil
	}

	// Use hash-based uniqueness check
	hashes := make(map[uint64][]int, maxLength) // hash -> indices
	seed := maphash.MakeSeed()

	for i := 0; i < maxLength; i++ {
		item := data[i]
		var h maphash.Hash
		h.SetSeed(seed)
		hashJSONValue(&h, item)
		hashValue := h.Sum64()

		// Check for hash collisions
		if indices := hashes[hashValue]; len(indices) > 0 {
			for _, j := range indices {
				if deepEqualJSON(item, data[j]) {
					return NewEvaluationError("uniqueItems", "unique_items_mismatch",
						"Array items at indices {index1} and {index2} are not unique", map[string]any{
							"index1": j,
							"index2": i,
						})
				}
			}
		}
		hashes[hashValue] = append(hashes[hashValue], i)
	}

	return nil
}

// hashJSONValue writes a deterministic hash of a JSON value to the hash.
func hashJSONValue(h *maphash.Hash, v any) {
	switch val := v.(type) {
	case nil:
		_ = h.WriteByte(0)

	case bool:
		if val {
			_ = h.WriteByte(1)
		} else {
			_ = h.WriteByte(0)
		}

	case float64:
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], math.Float64bits(val))
		_, _ = h.Write(buf[:])

	case int:
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(val)) //nolint:gosec // Overflow is acceptable for hashing
		_, _ = h.Write(buf[:])

	case int64:
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(val)) //nolint:gosec // Overflow is acceptable for hashing
		_, _ = h.Write(buf[:])

	case string:
		_, _ = h.WriteString(val)

	case []any:
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(len(val)))
		_, _ = h.Write(buf[:])
		for _, item := range val {
			hashJSONValue(h, item)
		}

	case map[string]any:
		// Sort keys for deterministic hashing
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		slices.Sort(keys)

		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(len(keys)))
		_, _ = h.Write(buf[:])

		for _, k := range keys {
			_, _ = h.WriteString(k)
			hashJSONValue(h, val[k])
		}

	default:
		// Fallback to reflection for other types
		hashJSONValueReflect(h, reflect.ValueOf(v))
	}
}

// hashJSONValueReflect handles hashing for types that need reflection.
func hashJSONValueReflect(h *maphash.Hash, rv reflect.Value) {
	if !rv.IsValid() {
		_ = h.WriteByte(0)
		return
	}

	switch rv.Kind() {
	case reflect.Bool:
		if rv.Bool() {
			_ = h.WriteByte(1)
		} else {
			_ = h.WriteByte(0)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(rv.Int())) //nolint:gosec // Overflow is acceptable for hashing
		_, _ = h.Write(buf[:])

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], rv.Uint())
		_, _ = h.Write(buf[:])

	case reflect.Float32, reflect.Float64:
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], math.Float64bits(rv.Float()))
		_, _ = h.Write(buf[:])

	case reflect.String:
		_, _ = h.WriteString(rv.String())

	case reflect.Slice, reflect.Array:
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(rv.Len())) //nolint:gosec // Overflow is acceptable for hashing
		_, _ = h.Write(buf[:])
		for i := 0; i < rv.Len(); i++ {
			hashJSONValueReflect(h, rv.Index(i))
		}

	case reflect.Map:
		keys := rv.MapKeys()
		slices.SortFunc(keys, func(a, b reflect.Value) int {
			return strings.Compare(fmt.Sprint(a.Interface()), fmt.Sprint(b.Interface()))
		})

		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(len(keys)))
		_, _ = h.Write(buf[:])

		for _, k := range keys {
			hashJSONValueReflect(h, k)
			hashJSONValueReflect(h, rv.MapIndex(k))
		}

	case reflect.Interface, reflect.Pointer:
		if rv.IsNil() {
			_ = h.WriteByte(0)
		} else {
			hashJSONValueReflect(h, rv.Elem())
		}

	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.Struct, reflect.UnsafePointer:
		// For unsupported types, use string representation as fallback
		_, _ = fmt.Fprint(h, rv.Interface())
	}
}

// deepEqualJSON performs deep equality comparison for JSON values.
func deepEqualJSON(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch va := a.(type) {
	case bool:
		vb, ok := b.(bool)
		return ok && va == vb

	case float64:
		vb, ok := b.(float64)
		return ok && va == vb

	case int:
		vb, ok := b.(int)
		return ok && va == vb

	case int64:
		vb, ok := b.(int64)
		return ok && va == vb

	case string:
		vb, ok := b.(string)
		return ok && va == vb

	case []any:
		vb, ok := b.([]any)
		if !ok || len(va) != len(vb) {
			return false
		}
		for i := range va {
			if !deepEqualJSON(va[i], vb[i]) {
				return false
			}
		}
		return true

	case map[string]any:
		vb, ok := b.(map[string]any)
		if !ok || len(va) != len(vb) {
			return false
		}
		for k, v := range va {
			vbVal, exists := vb[k]
			if !exists || !deepEqualJSON(v, vbVal) {
				return false
			}
		}
		return true
	}

	// Fallback to reflection-based comparison
	return deepEqualJSONReflect(reflect.ValueOf(a), reflect.ValueOf(b))
}

// deepEqualJSONReflect performs reflection-based deep equality.
func deepEqualJSONReflect(a, b reflect.Value) bool {
	if !a.IsValid() || !b.IsValid() {
		return a.IsValid() == b.IsValid()
	}

	if a.Kind() != b.Kind() {
		return false
	}

	switch a.Kind() {
	case reflect.Bool:
		return a.Bool() == b.Bool()

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() == b.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return a.Uint() == b.Uint()

	case reflect.Float32, reflect.Float64:
		return a.Float() == b.Float()

	case reflect.String:
		return a.String() == b.String()

	case reflect.Slice, reflect.Array:
		if a.Len() != b.Len() {
			return false
		}
		for i := 0; i < a.Len(); i++ {
			if !deepEqualJSONReflect(a.Index(i), b.Index(i)) {
				return false
			}
		}
		return true

	case reflect.Map:
		if a.Len() != b.Len() {
			return false
		}
		for _, k := range a.MapKeys() {
			aVal := a.MapIndex(k)
			bVal := b.MapIndex(k)
			if !bVal.IsValid() || !deepEqualJSONReflect(aVal, bVal) {
				return false
			}
		}
		return true

	case reflect.Interface, reflect.Pointer:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() == b.IsNil()
		}
		return deepEqualJSONReflect(a.Elem(), b.Elem())

	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.Struct, reflect.UnsafePointer:
		return false
	}

	return false
}
