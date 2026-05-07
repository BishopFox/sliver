package jsonschema

// Struct Validation Semantics
//
// This file implements JSON Schema validation for Go structs with special handling
// for omitempty, omitzero, and required field semantics.
//
// Key Concepts:
//
// 1. isEmptyValue: Used for `omitempty` tag behavior
//    - Determines if a field should be omitted from JSON serialization
//    - Empty values: nil pointers, zero-length collections, zero numbers, false booleans, zero time.Time
//    - Follows standard JSON encoding/json package semantics
//
// 2. isMissingValue: Used for `required` field validation
//    - Determines if a required field is present with a meaningful value
//    - Missing values: nil pointers, empty strings, zero-length collections, zero time.Time
//    - Non-missing: All numeric values (including 0), false for booleans
//    - Rationale: Required numeric/boolean fields can legitimately have zero/false values
//
// 3. isZeroValue: Used for `omitzero` tag behavior (Go 1.24+)
//    - Uses IsZero() method when available (time.Time, custom types)
//    - Falls back to reflect.Value.IsZero() for built-in types
//
// The distinction ensures that:
// - `required` validation allows zero values for numbers/booleans (0, false are valid)
// - `omitempty` follows JSON marshaling behavior
// - `omitzero` provides strict zero-value checking

import (
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"
)

// FieldCache stores parsed field information for a struct type
type FieldCache struct {
	FieldsByName map[string]FieldInfo
	FieldCount   int
}

// FieldInfo contains metadata for a struct field
type FieldInfo struct {
	Index     int          // Field index in the struct
	JSONName  string       // JSON field name (after processing tags)
	Omitempty bool         // Whether the field has omitempty tag
	Omitzero  bool         // Whether the field has omitzero tag
	Type      reflect.Type // Field type
}

// Global cache for struct field information
var fieldCacheMap sync.Map

// jsonTagIgnore is the special value used in JSON tags to skip a field
const jsonTagIgnore = "-"

// getFieldCache retrieves or creates cached field information for a struct type
func getFieldCache(structType reflect.Type) *FieldCache {
	if cached, ok := fieldCacheMap.Load(structType); ok {
		return cached.(*FieldCache)
	}

	cache := parseStructType(structType)
	fieldCacheMap.Store(structType, cache)
	return cache
}

// parseStructType analyzes a struct type and extracts field information
func parseStructType(structType reflect.Type) *FieldCache {
	cache := &FieldCache{
		FieldsByName: make(map[string]FieldInfo),
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		jsonName, omitempty, omitzero := parseJSONTag(field.Tag.Get("json"), field.Name)
		if jsonName == jsonTagIgnore {
			continue // Skip fields marked with json:"-"
		}

		cache.FieldsByName[jsonName] = FieldInfo{
			Index:     i,
			JSONName:  jsonName,
			Omitempty: omitempty,
			Omitzero:  omitzero,
			Type:      field.Type,
		}
		cache.FieldCount++
	}

	return cache
}

// parseJSONTag parses a JSON struct tag and returns the field name, omitempty and omitzero flags
func parseJSONTag(tag, defaultName string) (string, bool, bool) {
	if tag == "" {
		return defaultName, false, false
	}

	if commaIdx := strings.IndexByte(tag, ','); commaIdx >= 0 {
		name := tag[:commaIdx]
		if name == "" {
			name = defaultName
		}
		options := tag[commaIdx:]
		omitempty := strings.Contains(options, "omitempty")
		omitzero := strings.Contains(options, "omitzero")
		return name, omitempty, omitzero
	}

	return tag, false, false
}

// isZeroValue checks if a reflect.Value represents a zero value for omitzero behavior
// This uses the IsZero() method when available, following Go 1.24 omitzero semantics
func isZeroValue(rv reflect.Value) bool {
	// Check if the value has an IsZero method (like time.Time, custom types)
	if rv.CanInterface() {
		if zeroChecker, ok := rv.Interface().(interface{ IsZero() bool }); ok {
			return zeroChecker.IsZero()
		}
	}

	// Fall back to reflect.Value.IsZero() for built-in types
	return rv.IsZero()
}

// shouldOmitField determines if a field should be omitted based on omitempty/omitzero tags
func shouldOmitField(fieldInfo FieldInfo, fieldValue reflect.Value) bool {
	if fieldInfo.Omitzero && isZeroValue(fieldValue) {
		return true
	}
	if fieldInfo.Omitempty && isEmptyValue(fieldValue) {
		return true
	}
	return false
}

// isEmptyValue checks if a reflect.Value represents an empty value for omitempty behavior
func isEmptyValue(rv reflect.Value) bool {
	switch rv.Kind() {
	case reflect.Invalid:
		return true
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return rv.Len() == 0
	case reflect.Bool:
		return !rv.Bool() // For omitempty, false is considered empty
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return rv.IsNil()
	case reflect.Struct:
		// Use IsZero method if available (time.Time, custom types)
		if rv.CanInterface() {
			if zeroChecker, ok := rv.Interface().(interface{ IsZero() bool }); ok {
				return zeroChecker.IsZero()
			}
		}
		return rv.IsZero()
	case reflect.Uintptr, reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return false
	default:
		return false
	}
}

// isMissingValue checks if a reflect.Value represents a missing value for required validation
func isMissingValue(rv reflect.Value) bool {
	switch rv.Kind() {
	case reflect.Invalid:
		return true
	case reflect.Interface, reflect.Ptr:
		return rv.IsNil()
	case reflect.Struct:
		// Use IsZero method if available (time.Time, custom types)
		if rv.CanInterface() {
			if zeroChecker, ok := rv.Interface().(interface{ IsZero() bool }); ok {
				return zeroChecker.IsZero()
			}
		}
		return rv.IsZero()
	case reflect.String:
		// For required fields, empty string is considered missing
		return rv.String() == ""
	case reflect.Slice, reflect.Map, reflect.Array:
		// For required fields, empty collections are considered missing
		return rv.Len() == 0
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Uintptr, reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.UnsafePointer:
		// Numeric types and special types: non-zero values are considered present
		return false
	default:
		// Fallback for any other types
		return false
	}
}

// extractValue safely gets the any value from a reflect.Value
func extractValue(rv reflect.Value) any {
	// Handle pointers by dereferencing them first
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	// Special handling for time.Time - convert to string for JSON schema validation
	if rv.Type() == reflect.TypeOf(time.Time{}) {
		t, ok := rv.Interface().(time.Time)
		if !ok {
			return nil
		}
		return t.Format(time.RFC3339)
	}

	// Convert slices and arrays to []any for proper array validation
	if rv.Kind() == reflect.Slice {
		if rv.IsNil() {
			return nil
		}
		return convertSliceToAny(rv)
	}
	if rv.Kind() == reflect.Array {
		return convertSliceToAny(rv)
	}

	if rv.CanInterface() {
		return rv.Interface()
	}

	return nil
}

// convertSliceToAny converts a reflect.Value slice/array to []any
func convertSliceToAny(rv reflect.Value) []any {
	length := rv.Len()
	result := make([]any, length)
	for i := 0; i < length; i++ {
		elem := rv.Index(i)
		// Recursively extract values to handle nested pointers and special types
		result[i] = extractValue(elem)
	}
	return result
}

// appendValidationResult appends a validation result and tracks invalid properties
func appendValidationResult(results *[]*EvaluationResult, invalidProps *[]string, propName string, result *EvaluationResult) {
	if result == nil {
		return
	}
	*results = append(*results, result)
	if !result.IsValid() {
		*invalidProps = append(*invalidProps, propName)
	}
}

// evaluateObjectStruct handles validation for Go structs
func evaluateObjectStruct(schema *Schema, structValue reflect.Value, evaluatedProps map[string]bool, _ map[int]bool, dynamicScope *DynamicScope) ([]*EvaluationResult, []*EvaluationError) {
	var results []*EvaluationResult
	var errors []*EvaluationError

	structType := structValue.Type()
	fieldCache := getFieldCache(structType)

	// Validate properties
	if schema.Properties != nil {
		propertiesResults, propertiesErrors := evaluatePropertiesStruct(schema, structValue, fieldCache, evaluatedProps, dynamicScope)
		results = append(results, propertiesResults...)
		errors = append(errors, propertiesErrors...)
	}

	// Validate patternProperties
	if schema.PatternProperties != nil {
		patternResults, patternError := evaluatePatternPropertiesStruct(schema, structValue, fieldCache, evaluatedProps, dynamicScope)
		results = append(results, patternResults...)
		if patternError != nil {
			errors = append(errors, patternError)
		}
	}

	// Validate additionalProperties
	if schema.AdditionalProperties != nil {
		additionalResults, additionalError := evaluateAdditionalPropertiesStruct(schema, structValue, fieldCache, evaluatedProps, dynamicScope)
		results = append(results, additionalResults...)
		if additionalError != nil {
			errors = append(errors, additionalError)
		}
	}

	// Validate propertyNames
	if schema.PropertyNames != nil {
		propertyNamesResults, propertyNamesError := evaluatePropertyNamesStruct(schema, structValue, fieldCache, evaluatedProps, dynamicScope)
		results = append(results, propertyNamesResults...)
		if propertyNamesError != nil {
			errors = append(errors, propertyNamesError)
		}
	}

	// Validate required fields
	if len(schema.Required) > 0 {
		if err := evaluateRequiredStruct(schema, structValue, fieldCache); err != nil {
			errors = append(errors, err)
		}
	}

	// Validate dependentRequired
	if len(schema.DependentRequired) > 0 {
		if err := evaluateDependentRequiredStruct(schema, structValue, fieldCache); err != nil {
			errors = append(errors, err)
		}
	}

	// Validate property count constraints
	if schema.MaxProperties != nil || schema.MinProperties != nil {
		if err := evaluatePropertyCountStruct(schema, structValue, fieldCache); err != nil {
			errors = append(errors, err)
		}
	}

	return results, errors
}

// evaluateObjectReflectMap handles validation for reflect map types
func evaluateObjectReflectMap(schema *Schema, mapValue reflect.Value, evaluatedProps map[string]bool, evaluatedItems map[int]bool, dynamicScope *DynamicScope) ([]*EvaluationResult, []*EvaluationError) {
	// Convert reflect map to map[string]any and use existing logic
	object := make(map[string]any)

	for _, key := range mapValue.MapKeys() {
		if key.Kind() == reflect.String {
			value := mapValue.MapIndex(key)
			if value.CanInterface() {
				object[key.String()] = value.Interface()
			}
		}
	}

	return evaluateObjectMap(schema, object, evaluatedProps, evaluatedItems, dynamicScope)
}

// evaluatePropertiesStruct validates struct properties against schema properties
func evaluatePropertiesStruct(schema *Schema, structValue reflect.Value, fieldCache *FieldCache, evaluatedProps map[string]bool, dynamicScope *DynamicScope) ([]*EvaluationResult, []*EvaluationError) {
	var results []*EvaluationResult
	var errors []*EvaluationError
	var invalidProperties []string

	for propName, propSchema := range *schema.Properties {
		evaluatedProps[propName] = true

		fieldInfo, exists := fieldCache.FieldsByName[propName]
		if !exists {
			// Field doesn't exist in struct, only validate as nil if required and no default
			if isRequired(schema, propName) && !defaultIsSpecified(propSchema) {
				result, _, _ := propSchema.evaluate(nil, dynamicScope)
				appendValidationResult(&results, &invalidProperties, propName, result)
			}
			continue
		}

		// Get field value
		fieldValue := structValue.Field(fieldInfo.Index)

		// Handle omitempty/omitzero: skip validation if field should be omitted
		if shouldOmitField(fieldInfo, fieldValue) {
			continue
		}

		// Get the interface value for validation
		valueToValidate := extractValue(fieldValue)

		result, _, _ := propSchema.evaluate(valueToValidate, dynamicScope)
		appendValidationResult(&results, &invalidProperties, propName, result)
	}

	// Handle errors for invalid properties
	if len(invalidProperties) > 0 {
		errors = append(errors, createPropertyValidationError(invalidProperties))
	}

	return results, errors
}

// evaluateRequiredStruct validates required fields for structs
func evaluateRequiredStruct(schema *Schema, structValue reflect.Value, fieldCache *FieldCache) *EvaluationError {
	var missingFields []string

	for _, requiredField := range schema.Required {
		fieldInfo, exists := fieldCache.FieldsByName[requiredField]
		if !exists {
			missingFields = append(missingFields, requiredField)
			continue
		}

		fieldValue := structValue.Field(fieldInfo.Index)

		// Check if field is missing or empty
		if !fieldValue.IsValid() || isMissingValue(fieldValue) {
			missingFields = append(missingFields, requiredField)
		}
	}

	return createRequiredValidationError(missingFields)
}

// evaluatePropertyCountStruct validates maxProperties and minProperties for structs
func evaluatePropertyCountStruct(schema *Schema, structValue reflect.Value, fieldCache *FieldCache) *EvaluationError {
	// Count actual non-empty properties (considering omitempty)
	actualCount := 0
	for _, fieldInfo := range fieldCache.FieldsByName {
		fieldValue := structValue.Field(fieldInfo.Index)
		if !shouldOmitField(fieldInfo, fieldValue) {
			actualCount++
		}
	}

	if schema.MaxProperties != nil && float64(actualCount) > *schema.MaxProperties {
		return NewEvaluationError("maxProperties", "too_many_properties",
			"Value should have at most {max_properties} properties", map[string]any{
				"max_properties": *schema.MaxProperties,
			})
	}

	if schema.MinProperties != nil && float64(actualCount) < *schema.MinProperties {
		return NewEvaluationError("minProperties", "too_few_properties",
			"Value should have at least {min_properties} properties", map[string]any{
				"min_properties": *schema.MinProperties,
			})
	}

	return nil
}

// evaluatePatternPropertiesStruct validates struct properties against pattern properties
func evaluatePatternPropertiesStruct(schema *Schema, structValue reflect.Value, fieldCache *FieldCache, evaluatedProps map[string]bool, dynamicScope *DynamicScope) ([]*EvaluationResult, *EvaluationError) {
	var results []*EvaluationResult

	for jsonName, fieldInfo := range fieldCache.FieldsByName {
		if evaluatedProps[jsonName] {
			continue
		}

		fieldValue := structValue.Field(fieldInfo.Index)
		if shouldOmitField(fieldInfo, fieldValue) {
			continue
		}

		for pattern, patternSchema := range *schema.PatternProperties {
			if matched, _ := regexp.MatchString(pattern, jsonName); matched {
				evaluatedProps[jsonName] = true
				value := extractValue(fieldValue)

				// Reuse existing validation logic directly
				result, _, _ := patternSchema.evaluate(value, dynamicScope)
				if result != nil {
					results = append(results, result)
				}
				break
			}
		}
	}

	return results, nil
}

// evaluateAdditionalPropertiesStruct validates struct properties against additional properties
func evaluateAdditionalPropertiesStruct(schema *Schema, structValue reflect.Value, fieldCache *FieldCache, evaluatedProps map[string]bool, dynamicScope *DynamicScope) ([]*EvaluationResult, *EvaluationError) {
	var results []*EvaluationResult
	var invalidProperties []string

	// Check for unevaluated properties
	for jsonName, fieldInfo := range fieldCache.FieldsByName {
		if evaluatedProps[jsonName] {
			continue
		}

		fieldValue := structValue.Field(fieldInfo.Index)
		if shouldOmitField(fieldInfo, fieldValue) {
			continue
		}

		// This is an additional property, validate according to additionalProperties
		if schema.AdditionalProperties != nil {
			value := extractValue(fieldValue)
			result, _, _ := schema.AdditionalProperties.evaluate(value, dynamicScope)
			if result != nil {
				results = append(results, result)
				if !result.IsValid() {
					invalidProperties = append(invalidProperties, jsonName)
				}
			}
			// Mark property as evaluated
			evaluatedProps[jsonName] = true
		}
	}

	// Handle errors for invalid properties
	if len(invalidProperties) > 0 {
		return results, createValidationError(
			"additional_property_mismatch",
			"additionalProperties",
			"Additional property {property} does not match the schema",
			"Additional properties {properties} do not match the schema",
			invalidProperties,
		)
	}

	return results, nil
}

// evaluatePropertyNamesStruct validates struct property names
func evaluatePropertyNamesStruct(schema *Schema, structValue reflect.Value, fieldCache *FieldCache, _ map[string]bool, dynamicScope *DynamicScope) ([]*EvaluationResult, *EvaluationError) {
	if schema.PropertyNames == nil {
		return nil, nil
	}

	var results []*EvaluationResult
	var invalidProperties []string

	for jsonName, fieldInfo := range fieldCache.FieldsByName {
		fieldValue := structValue.Field(fieldInfo.Index)
		if shouldOmitField(fieldInfo, fieldValue) {
			continue
		}

		// Validate the property name itself
		result, _, _ := schema.PropertyNames.evaluate(jsonName, dynamicScope)
		if result != nil {
			results = append(results, result)
			if !result.IsValid() {
				invalidProperties = append(invalidProperties, jsonName)
			}
		}
	}

	// Handle errors for invalid properties
	if len(invalidProperties) > 0 {
		return results, createValidationError(
			"property_name_mismatch",
			"propertyNames",
			"Property name {property} does not match the schema",
			"Property names {properties} do not match the schema",
			invalidProperties,
		)
	}

	return results, nil
}

// evaluateDependentRequiredStruct validates dependent required properties for structs
func evaluateDependentRequiredStruct(schema *Schema, structValue reflect.Value, fieldCache *FieldCache) *EvaluationError {
	for propName, dependentRequired := range schema.DependentRequired {
		// Check if property exists
		fieldInfo, exists := fieldCache.FieldsByName[propName]
		if !exists {
			continue
		}

		fieldValue := structValue.Field(fieldInfo.Index)

		// If property exists and is not empty, check dependent properties
		if !isEmptyValue(fieldValue) {
			for _, requiredProp := range dependentRequired {
				depFieldInfo, depExists := fieldCache.FieldsByName[requiredProp]
				if !depExists {
					return NewEvaluationError("dependentRequired", "dependent_required_missing",
						"Property {property} is required when {dependent_property} is present", map[string]any{
							"property":           requiredProp,
							"dependent_property": propName,
						})
				}

				depFieldValue := structValue.Field(depFieldInfo.Index)
				if isMissingValue(depFieldValue) {
					return NewEvaluationError("dependentRequired", "dependent_required_missing",
						"Property {property} is required when {dependent_property} is present", map[string]any{
							"property":           requiredProp,
							"dependent_property": propName,
						})
				}
			}
		}
	}

	return nil
}

// createValidationError creates a validation error with proper formatting for single or multiple items
func createValidationError(errorType, keyword string, singleTemplate, multiTemplate string, invalidItems []string) *EvaluationError {
	if len(invalidItems) == 1 {
		return NewEvaluationError(keyword, errorType, singleTemplate, map[string]any{
			"property": invalidItems[0],
		})
	}
	if len(invalidItems) > 1 {
		return NewEvaluationError(keyword, errorType, multiTemplate, map[string]any{
			"properties": strings.Join(invalidItems, ", "),
		})
	}
	return nil
}

// createPropertyValidationError creates a validation error for property validation
func createPropertyValidationError(invalidProperties []string) *EvaluationError {
	return createValidationError(
		"property_mismatch",
		"properties",
		"Property {property} does not match the schema",
		"Properties {properties} do not match their schemas",
		invalidProperties,
	)
}

// createRequiredValidationError creates a validation error for required field validation
func createRequiredValidationError(missingFields []string) *EvaluationError {
	return createValidationError(
		"required_missing",
		"required",
		"Required property {property} is missing",
		"Required properties {properties} are missing",
		missingFields,
	)
}
