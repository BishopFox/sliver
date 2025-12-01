package table

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
)

// AutoIndexColumnID returns a unique Column ID/Name for the given Column Number.
// The functionality is similar to what you get in an Excel spreadsheet w.r.t.
// the Column ID/Name.
func AutoIndexColumnID(colIdx int) string {
	charIdx := colIdx % 26
	out := string(rune(65 + charIdx))
	colIdx = colIdx / 26
	if colIdx > 0 {
		return AutoIndexColumnID(colIdx-1) + out
	}
	return out
}

// WidthEnforcer is a function that helps enforce a width condition on a string.
type WidthEnforcer func(col string, maxLen int) string

// widthEnforcerNone returns the input string as is without any modifications.
func widthEnforcerNone(col string, _ int) string {
	return col
}

// convertValueToString converts a value to string using fast type assertions
// for common numeric types before falling back to fmt.Sprint.
//
//gocyclo:ignore
func convertValueToString(v interface{}) string {
	switch val := v.(type) {
	case int:
		return strconv.FormatInt(int64(val), 10)
	case int8:
		return strconv.FormatInt(int64(val), 10)
	case int16:
		return strconv.FormatInt(int64(val), 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint8:
		return strconv.FormatUint(uint64(val), 10)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float32:
		return strconv.FormatFloat(float64(val), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'g', -1, 64)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case string:
		return val
	default:
		return fmt.Sprint(v)
	}
}

// isNumber returns true if the argument is a numeric type; false otherwise.
func isNumber(x interface{}) bool {
	if x == nil {
		return false
	}

	switch reflect.TypeOf(x).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

type mergedColumnIndices map[int]int

func objAsSlice(in interface{}) []interface{} {
	var out []interface{}
	if in != nil {
		// dereference pointers
		val := reflect.ValueOf(in)
		if val.Kind() == reflect.Ptr && !val.IsNil() {
			in = val.Elem().Interface()
		}

		if objIsSlice(in) {
			v := reflect.ValueOf(in)
			for i := 0; i < v.Len(); i++ {
				// dereference pointers
				v2 := v.Index(i)
				if v2.Kind() == reflect.Ptr && !v2.IsNil() {
					v2 = reflect.ValueOf(v2.Elem().Interface())
				}

				out = append(out, v2.Interface())
			}
		}
	}

	// remove trailing nil pointers
	tailIdx := len(out)
	for i := len(out) - 1; i >= 0; i-- {
		val := reflect.ValueOf(out[i])
		if val.Kind() != reflect.Ptr || !val.IsNil() {
			break
		}
		tailIdx = i
	}
	return out[:tailIdx]
}

func objIsSlice(in interface{}) bool {
	if in == nil {
		return false
	}
	k := reflect.TypeOf(in).Kind()
	return k == reflect.Slice || k == reflect.Array
}

func getSortedKeys(input map[int]map[int]int) ([]int, map[int][]int) {
	keys := make([]int, 0, len(input))
	subkeysMap := make(map[int][]int)
	for key, subMap := range input {
		keys = append(keys, key)
		subkeys := make([]int, 0, len(subMap))
		for subkey := range subMap {
			subkeys = append(subkeys, subkey)
		}
		sort.Ints(subkeys)
		subkeysMap[key] = subkeys
	}
	sort.Ints(keys)
	return keys, subkeysMap
}
