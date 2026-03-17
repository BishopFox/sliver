package jsonpointer

import (
	"reflect"
	"strings"
	"sync"
)

// structFields caches field mapping for struct types
type structFields map[string]int

// structFieldsCache global cache that stores field mapping for each struct type
var structFieldsCache sync.Map

// structField looks up the specified field in struct, updates value to point to that field if found
// Returns true if field is found, false if not found
func structField(field string, value *reflect.Value) bool {
	// Dereference pointers
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return false
		}
		*value = value.Elem()
	}

	// Ensure it's a struct type
	if value.Kind() != reflect.Struct {
		return false
	}

	// Get field mapping
	fields := getStructFields(value.Type())
	fieldIndex, ok := fields[field]
	if !ok {
		return false
	}

	// Get field value
	*value = value.Field(fieldIndex)
	return true
}

// getStructFields gets field mapping for struct type with caching
func getStructFields(t reflect.Type) structFields {
	// Try to get from cache
	if cached, ok := structFieldsCache.Load(t); ok {
		return cached.(structFields)
	}

	// Build field mapping
	fields := make(structFields)
	numField := t.NumField()

	for i := 0; i < numField; i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get field name
		name := getFieldName(field)
		if name == "-" {
			continue // json:"-" means ignore field
		}

		fields[name] = i
	}

	// Store in cache
	structFieldsCache.Store(t, fields)
	return fields
}

// getFieldName gets the JSON name of field, supports basic JSON tags.
// Optimized with strings.Cut (Go 1.18+) for cleaner parsing.
func getFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}

	// Use strings.Cut for cleaner parsing
	// Extracts field name before comma: "name,omitempty" → "name"
	name, _, _ := strings.Cut(tag, ",")
	if name != "" {
		return name
	}

	// If only options (e.g., ",omitempty"), use field name
	return field.Name
}
