package jsonpointer

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// fastAtoi converts a string to an integer quickly without allocations.
// Returns -1 if the string is not a valid non-negative integer.
// This is optimized for JSON Pointer array index parsing.
func fastAtoi(s string) int {
	if len(s) == 0 {
		return -1
	}

	// Special case: "0" is valid
	if s == "0" {
		return 0
	}

	// Leading zeros are invalid per RFC 6901
	if s[0] == '0' {
		return -1
	}

	var n int
	for i := range len(s) {
		c := s[i]
		if c < '0' || c > '9' {
			return -1 // Non-digit character
		}
		next := n*10 + int(c-'0')
		if next < n {
			return -1 // Integer overflow detected
		}
		n = next
	}
	return n
}

// derefValue dereferences pointer values until reaching a non-pointer value.
// Returns an error if any pointer in the chain is nil.
// This is a helper function to eliminate duplicated pointer dereferencing logic.
func derefValue(v reflect.Value) (reflect.Value, error) {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}, ErrNilPointer
		}
		v = v.Elem()
	}
	return v, nil
}

// unescapeComponent un-escapes a JSON pointer path component.
//
// TypeScript Original:
//
//	export function unescapeComponent(component: string): string {
//	  if (component.indexOf('~') === -1) return component;
//	  return component.replace(r1, '/').replace(r2, '~');
//	}
func unescapeComponent(component string) string {
	if !strings.Contains(component, "~") {
		return component
	}

	result := make([]byte, 0, len(component))
	for i := 0; i < len(component); i++ {
		if component[i] == '~' && i+1 < len(component) {
			switch component[i+1] {
			case '0':
				result = append(result, '~')
				i++
			case '1':
				result = append(result, '/')
				i++
			default:
				result = append(result, component[i])
			}
		} else {
			result = append(result, component[i])
		}
	}
	return string(result)
}

// escapeComponent escapes a JSON pointer path component.
//
// TypeScript Original:
//
//	export function escapeComponent(component: string): string {
//	  if (component.indexOf('/') === -1 && component.indexOf('~') === -1) return component;
//	  return component.replace(r3, '~0').replace(r4, '~1');
//	}
func escapeComponent(component string) string {
	if !strings.Contains(component, "/") && !strings.Contains(component, "~") {
		return component
	}

	result := make([]byte, 0, len(component)*2)
	for i := range len(component) {
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

// parseJSONPointer converts JSON pointer like "/foo/bar" to path slice
// like []string{"foo", "bar"}, while also un-escaping reserved characters.
//
// TypeScript Original:
//
//	export function parseJsonPointer(pointer: string): Path {
//	  if (!pointer) return [];
//	  return pointer.slice(1).split('/').map(unescapeComponent);
//	}
func parseJSONPointer(pointer string) Path {
	if pointer == "" {
		return Path{}
	}

	segmentCount := 1
	for i := range len(pointer) - 1 {
		if pointer[i+1] == '/' {
			segmentCount++
		}
	}

	result := make(Path, 0, segmentCount)
	start := 1

	for i := 1; i <= len(pointer); i++ {
		if i == len(pointer) || pointer[i] == '/' {
			segment := pointer[start:i]
			result = append(result, unescapeComponent(segment))
			start = i + 1
		}
	}

	return result
}

// formatJSONPointer escapes and formats a path slice like []string{"foo", "bar"}
// to JSON pointer like "/foo/bar".
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

	capacity := len(path)
	for _, comp := range path {
		capacity += len(comp) + 2
	}

	var b strings.Builder
	b.Grow(capacity)

	for _, component := range path {
		b.WriteByte('/')
		b.WriteString(escapeComponent(component))
	}
	return b.String()
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
	return slices.Equal(parent, child[:len(parent)])
}

// IsPathEqual returns true if two paths are equal, false otherwise.
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

// Parent returns parent path, e.g. for []string{"foo", "bar", "baz"}
// returns []string{"foo", "bar"}.
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
// Preserves RFC 6901 semantics for array end marker and bounds checking.
func validateArrayIndex(key string, length int) (int, error) {
	if key == "-" {
		return -1, ErrIndexOutOfBounds
	}
	index := fastAtoi(key)
	if index < 0 {
		return -1, ErrInvalidIndex
	}
	if index > length {
		return -1, ErrIndexOutOfBounds
	}
	return index, nil
}

// validateAndAccessArray validates array index and checks for array end marker.
// Returns ErrIndexOutOfBounds if index equals array length per RFC 6901.
func validateAndAccessArray(key string, length int) (int, error) {
	index, err := validateArrayIndex(key, length)
	if err != nil {
		return -1, err
	}
	if index == length {
		return -1, ErrIndexOutOfBounds
	}
	return index, nil
}
