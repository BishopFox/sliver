package jsonpointer

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// fastAtoi converts a string to an integer quickly.
// Returns -1 if the string is not a valid non-negative integer.
func fastAtoi(s string) int {
	if len(s) == 0 {
		return -1
	}

	// Handle special case for "0"
	if s == "0" {
		return 0
	}

	// Check for leading zeros (invalid except for "0")
	if s[0] == '0' {
		return -1
	}

	var n int
	for _, r := range s {
		if r < '0' || r > '9' {
			return -1 // non-digit character
		}
		t := n*10 + int(r-'0')
		if t < n {
			return -1 // overflow
		}
		n = t
	}
	return n
}

// derefValue dereferences pointer values until reaching a non-pointer value.
// Returns an error if any pointer in the chain is nil.
// This is a helper function to eliminate duplicated pointer dereferencing logic.
func derefValue(v reflect.Value) (reflect.Value, error) {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}, ErrNilPointer
		}
		v = v.Elem()
	}
	return v, nil
}

// unescapeComponent un-escapes a JSON pointer path component.
// Returns the unescaped component string.
//
// TypeScript Original:
//
//	export function unescapeComponent(component: string): string {
//	  if (component.indexOf('~') === -1) return component;
//	  return component.replace(r1, '/').replace(r2, '~');
//	}
func unescapeComponent(component string) string {
	// Use strings.IndexByte for fast check if escaping is needed
	if strings.IndexByte(component, '~') == -1 {
		return component
	}

	// Pre-allocate result string capacity
	result := make([]byte, 0, len(component))
	for i := 0; i < len(component); i++ {
		if component[i] == '~' && i+1 < len(component) {
			switch component[i+1] {
			case '0':
				result = append(result, '~')
				i++ // Skip next character
			case '1':
				result = append(result, '/')
				i++ // Skip next character
			default:
				result = append(result, component[i])
			}
		} else {
			result = append(result, component[i])
		}
	}
	return string(result)
}

// EscapeComponent escapes a JSON pointer path component.
// Returns the escaped component string.
//
// TypeScript Original:
//
//	export function escapeComponent(component: string): string {
//	  if (component.indexOf('/') === -1 && component.indexOf('~') === -1) return component;
//	  return component.replace(r3, '~0').replace(r4, '~1');
//	}
func escapeComponent(component string) string {
	// Use strings.IndexByte for fast check
	if strings.IndexByte(component, '/') == -1 && strings.IndexByte(component, '~') == -1 {
		return component
	}

	// Pre-allocate result string capacity (worst case: every character needs escaping)
	result := make([]byte, 0, len(component)*2)
	for i := 0; i < len(component); i++ {
		switch component[i] {
		case '~':
			result = append(result, '~', '0')
		case '/':
			result = append(result, '~', '1')
		default:
			result = append(result, component[i])
		}
	}
	return string(result)
}

// ParseJsonPointer converts JSON pointer like "/foo/bar" to path slice like []string{"foo", "bar"},
// while also un-escaping reserved characters.
//
// TypeScript Original:
//
//	export function parseJsonPointer(pointer: string): Path {
//	  if (!pointer) return [];
//	  return pointer.slice(1).split('/').map(unescapeComponent);
//	}
//
// Note: The Go implementation uses optimized string processing without split/map for better performance.
func parseJSONPointer(pointer string) Path {
	if pointer == "" {
		return Path{}
	}

	// Pre-calculate number of path segments
	segmentCount := 1
	for i := 1; i < len(pointer); i++ {
		if pointer[i] == '/' {
			segmentCount++
		}
	}

	// Pre-allocate result slice
	result := make(Path, 0, segmentCount)
	start := 1 // Skip the first '/'

	for i := 1; i <= len(pointer); i++ {
		if i == len(pointer) || pointer[i] == '/' {
			// Include empty string segments (like empty segments in "/foo///")
			segment := pointer[start:i]
			result = append(result, unescapeComponent(segment))
			start = i + 1
		}
	}

	return result
}

// FormatJsonPointer escapes and formats a path slice like []string{"foo", "bar"}
// to JSON pointer like "/foo/bar".
// Optimized with strings.Builder pre-allocation for zero intermediate allocations.
//
// TypeScript Original:
//
//	export function formatJsonPointer(path: Path): string {
//	  if (isRoot(path)) return '';
//	  return '/' + path.map((component) => escapeComponent(String(component))).join('/');
//	}
func formatJSONPointer(path Path) string {
	if IsRoot(path) {
		return ""
	}

	// Pre-calculate capacity for single allocation
	// Each component needs: '/' separator + component length + potential escaping (max 2 chars per original char)
	capacity := len(path) // '/' separators
	for _, comp := range path {
		capacity += len(comp) + 2 // component + max 2 chars for potential escaping
	}

	var b strings.Builder
	b.Grow(capacity)

	for _, component := range path {
		b.WriteByte('/')
		b.WriteString(escapeComponent(component))
	}
	return b.String()
}

// ToPath converts a pointer (string or Path) to Path.
// If the input is a string, it parses it as JSON pointer.
// If the input is already a Path, it returns it as-is.
//
// TypeScript Original:
// export const toPath = (pointer: string | Path) => (typeof pointer === 'string' ? parseJsonPointer(pointer) : pointer);
func ToPath(pointer any) Path {
	switch p := pointer.(type) {
	case string:
		return parseJSONPointer(p)
	case Path:
		return p
	case []string:
		result := make(Path, len(p))
		copy(result, p)
		return result
	default:
		// For other types, return empty path
		return Path{}
	}
}

// IsChild returns true if parent contains child path, false otherwise.
//
// TypeScript Original:
//
//	export function isChild(parent: Path, child: Path): boolean {
//	  if (parent.length >= child.length) return false;
//	  for (let i = 0; i < parent.length; i++) if (parent[i] !== child[i]) return false;
//	  return true;
//	}
func IsChild(parent, child Path) bool {
	if len(parent) >= len(child) {
		return false
	}
	for i := 0; i < len(parent); i++ {
		if parent[i] != child[i] {
			return false
		}
	}
	return true
}

// IsPathEqual returns true if two paths are equal, false otherwise.
// Optimized with slices.Equal (Go 1.21+) using SIMD instructions.
//
// TypeScript Original:
//
//	export function isPathEqual(p1: Path, p2: Path): boolean {
//	  if (p1.length !== p2.length) return false;
//	  for (let i = 0; i < p1.length; i++) if (p1[i] !== p2[i]) return false;
//	  return true;
//	}
func IsPathEqual(p1, p2 Path) bool {
	return slices.Equal(p1, p2)
}

// IsRoot returns true if JSON Pointer points to root value, false otherwise.
//
// TypeScript Original:
// export const isRoot = (path: Path): boolean => !path.length;
func IsRoot(path Path) bool {
	return len(path) == 0
}

// Parent returns parent path, e.g. for []string{"foo", "bar", "baz"} returns []string{"foo", "bar"}.
// Returns ErrNoParent if the path has no parent (empty or root path).
//
// TypeScript Original:
//
//	export function parent(path: Path): Path {
//	  if (path.length < 1) throw new Error('NO_PARENT');
//	  return path.slice(0, path.length - 1);
//	}
func Parent(path Path) (Path, error) {
	if len(path) < 1 {
		return nil, ErrNoParent
	}
	return path[:len(path)-1], nil
}

// IsValidIndex checks if path component can be a valid array index.
//
// TypeScript Original:
//
//	export function isValidIndex(index: string | number): boolean {
//	  if (typeof index === 'number') return true;
//	  const n = Number.parseInt(index, 10);
//	  return String(n) === index && n >= 0;
//	}
func IsValidIndex(index string) bool {
	if index == "-" {
		return true // Special case for array end marker
	}
	n, err := strconv.ParseInt(index, 10, 64)
	if err != nil {
		return false
	}
	// Check if string representation matches parsed value and is non-negative
	return strconv.FormatInt(n, 10) == index && n >= 0
}

// IsInteger checks if a string contains only digit characters (0-9).
//
// TypeScript Original:
//
//	export const isInteger = (str: string): boolean => {
//	  const len = str.length;
//	  let i = 0;
//	  let charCode: any;
//	  while (i < len) {
//	    charCode = str.charCodeAt(i);
//	    if (charCode >= 48 && charCode <= 57) {
//	      i++;
//	      continue;
//	    }
//	    return false;
//	  }
//	  return true;
//	};
func IsInteger(str string) bool {
	if len(str) == 0 {
		return false
	}
	for _, r := range str {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// validateArrayIndex validates and parses array index from string key.
// Returns the parsed index and error if validation fails.
// Preserves RFC 6901 semantics: distinguishes between index == length (array end) and index > length (out of bounds).
func validateArrayIndex(key string, length int) (int, error) {
	if key == "-" {
		return -1, ErrIndexOutOfBounds // "-" refers to nonexistent element
	}
	index := fastAtoi(key)
	if index < 0 {
		return -1, ErrInvalidIndex
	}
	// Note: Caller should handle the distinction between index == length and index > length
	// to maintain RFC 6901 semantics
	if index > length {
		return -1, ErrIndexOutOfBounds
	}
	return index, nil
}

// validateAndAccessArray validates array index and checks for array end marker.
// Returns ErrIndexOutOfBounds if index equals array length (array end marker per RFC 6901).
// Returns the validated index ready for array access.
// This helper eliminates repeated validation + end-check logic across get.go and find.go.
func validateAndAccessArray(key string, length int) (int, error) {
	index, err := validateArrayIndex(key, length)
	if err != nil {
		return -1, err
	}
	if index == length {
		// Array end position is nonexistent element (JSON Pointer spec)
		return -1, ErrIndexOutOfBounds
	}
	return index, nil
}
