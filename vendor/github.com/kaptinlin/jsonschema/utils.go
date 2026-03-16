package jsonschema

import (
	"fmt"
	"math/big"
	"net/url"
	"path"
	"reflect"
	"strings"
)

// replace substitutes placeholders in a template string with actual parameter values.
func replace(template string, params map[string]any) string {
	for key, value := range params {
		placeholder := "{" + key + "}"
		template = strings.ReplaceAll(template, placeholder, fmt.Sprint(value))
	}

	return template
}

// mergeIntMaps merges two integer maps. The values in the second map overwrite the first where keys overlap.
func mergeIntMaps(map1, map2 map[int]bool) map[int]bool {
	for key, value := range map2 {
		map1[key] = value
	}
	return map1
}

// mergeStringMaps merges two string maps. The values in the second map overwrite the first where keys overlap.
func mergeStringMaps(map1, map2 map[string]bool) map[string]bool {
	for key, value := range map2 {
		map1[key] = value
	}
	return map1
}

// getDataType identifies the JSON schema type for a given Go value.
func getDataType(v any) string {
	switch v := v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	// json.Number is not available in go-json-experiment/json
	// Numbers are handled as float64 by default
	case float32, float64:
		// Convert to big.Float to check if it can be considered an integer
		bigFloat := new(big.Float).SetFloat64(reflect.ValueOf(v).Float())
		if _, acc := bigFloat.Int(nil); acc == big.Exact {
			return "integer" // Treated as integer if no fractional part
		}
		return "number"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "integer"
	case string:
		return "string"
	case []any:
		return "array"
	case []bool, []float32, []float64, []int, []int8, []int16, []int32, []int64, []uint, []uint8, []uint16, []uint32, []uint64, []string:
		return "array"
	case map[string]any:
		return "object"
	default:
		// Use reflection for other types (struct, slices, maps, etc.)
		return getDataTypeReflect(v)
	}
}

// getDataTypeReflect uses reflection to determine the JSON schema type for complex Go types.
func getDataTypeReflect(v any) string {
	rv := reflect.ValueOf(v)

	// Handle pointers by dereferencing them
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return "null"
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Invalid:
		return "null"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		f := rv.Float()
		if f == float64(int64(f)) {
			return "integer"
		}
		return "number"
	case reflect.String:
		return "string"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map:
		// Only consider string-keyed maps as objects
		if rv.Type().Key().Kind() == reflect.String {
			return "object"
		}
		return "unknown"
	case reflect.Struct:
		return "object"
	case reflect.Interface:
		if rv.IsNil() {
			return "null"
		}
		return getDataType(rv.Interface())
	case reflect.Uintptr, reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func, reflect.Ptr, reflect.UnsafePointer:
		return "unknown"
	default:
		return "unknown"
	}
}

// getURLScheme extracts the scheme component of a URL string.
func getURLScheme(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return parsedURL.Scheme
}

// isValidURI verifies if the provided string is a valid URI.
func isValidURI(s string) bool {
	_, err := url.ParseRequestURI(s)
	return err == nil
}

// resolveRelativeURI resolves a relative URI against a base URI.
func resolveRelativeURI(baseURI, relativeURL string) string {
	if isAbsoluteURI(relativeURL) {
		return relativeURL
	}
	base, err := url.Parse(baseURI)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return relativeURL // Return the original if there's a base URL parsing error
	}
	rel, err := url.Parse(relativeURL)
	if err != nil {
		return relativeURL // Return the original if there's a relative URL parsing error
	}
	return base.ResolveReference(rel).String()
}

// isAbsoluteURI checks if the given URL is absolute.
func isAbsoluteURI(urlStr string) bool {
	u, err := url.Parse(urlStr)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// getBaseURI extracts the base URL from an $id URI, falling back if not valid.
func getBaseURI(id string) string {
	if id == "" {
		return ""
	}
	u, err := url.Parse(id)
	if err != nil {
		return ""
	}
	if strings.HasSuffix(u.Path, "/") {
		return u.String()
	}
	u.Path = path.Dir(u.Path)
	if u.Path == "." {
		u.Path = "/"
	}
	if u.Path != "/" && !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	if u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.String()
}

// splitRef separates a URI into its base URI and anchor parts.
func splitRef(ref string) (baseURI string, anchor string) {
	parts := strings.SplitN(ref, "#", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return ref, ""
}

// isJSONPointer checks if a string is a JSON Pointer.
func isJSONPointer(s string) bool {
	return strings.HasPrefix(s, "/")
}
