package jsonschema

import (
	"reflect"
	"strings"
)

// Validate checks if the given instance conforms to the schema.
// This method automatically detects the input type and delegates to the appropriate validation method.
func (s *Schema) Validate(instance any) *EvaluationResult {
	switch data := instance.(type) {
	case []byte:
		return s.ValidateJSON(data)
	case map[string]any:
		return s.ValidateMap(data)
	default:
		// Check if it's a []byte type definition (like json.RawMessage)
		if isByteSlice(instance) {
			if bytes, ok := convertToByteSlice(instance); ok {
				return s.ValidateJSON(bytes)
			}
		}
		return s.ValidateStruct(instance)
	}
}

// ValidateJSON validates JSON data provided as []byte.
// The input is guaranteed to be treated as JSON data and parsed accordingly.
func (s *Schema) ValidateJSON(data []byte) *EvaluationResult {
	parsed, err := s.parseJSONData(data)
	if err != nil {
		result := NewEvaluationResult(s)
		//nolint:errcheck
		result.AddError(NewEvaluationError("format", "invalid_json", "Invalid JSON format"))
		return result
	}

	dynamicScope := NewDynamicScope()
	result, _, _ := s.evaluate(parsed, dynamicScope)
	return result
}

// ValidateStruct validates Go struct data directly using reflection.
// This method uses cached reflection data for optimal performance.
func (s *Schema) ValidateStruct(instance any) *EvaluationResult {
	dynamicScope := NewDynamicScope()
	result, _, _ := s.evaluate(instance, dynamicScope)
	return result
}

// ValidateMap validates map[string]any data directly.
// This method provides optimal performance for pre-parsed JSON data.
func (s *Schema) ValidateMap(data map[string]any) *EvaluationResult {
	dynamicScope := NewDynamicScope()
	result, _, _ := s.evaluate(data, dynamicScope)
	return result
}

// parseJSONData safely parses []byte data as JSON
func (s *Schema) parseJSONData(data []byte) (any, error) {
	var parsed any
	return parsed, s.GetCompiler().jsonDecoder(data, &parsed)
}

// processJSONBytes handles []byte input with smart JSON parsing
func (s *Schema) processJSONBytes(jsonBytes []byte) (any, error) {
	var parsed any
	if err := s.GetCompiler().jsonDecoder(jsonBytes, &parsed); err == nil {
		return parsed, nil
	}

	// Only return error if it looks like intended JSON
	if len(jsonBytes) > 0 && (jsonBytes[0] == '{' || jsonBytes[0] == '[') {
		return nil, s.GetCompiler().jsonDecoder(jsonBytes, &parsed)
	}

	// Otherwise, keep original bytes for validation as byte array
	return jsonBytes, nil
}

func (s *Schema) evaluate(instance any, dynamicScope *DynamicScope) (*EvaluationResult, map[string]bool, map[int]bool) {
	// Handle []byte input
	instance = s.preprocessByteInput(instance)

	// Check for circular reference before processing
	if dynamicScope.Contains(s) {
		// Determine if this is a problematic circular reference
		if s.isProblematicCircularReference(dynamicScope) {
			result := NewEvaluationResult(s)
			// For problematic circular references, we perform basic validation without following references
			// This prevents infinite recursion while still validating according to schema constraints
			evaluatedProps := make(map[string]bool)
			evaluatedItems := make(map[int]bool)

			// Process basic validation without references to avoid infinite loop
			s.processBasicValidationWithoutRefs(instance, result, evaluatedProps, evaluatedItems)

			return result, evaluatedProps, evaluatedItems
		}
	}

	dynamicScope.Push(s)
	defer dynamicScope.Pop()

	result := NewEvaluationResult(s)
	evaluatedProps := make(map[string]bool)
	evaluatedItems := make(map[int]bool)

	// Process schema types
	if s.Boolean != nil {
		if err := s.evaluateBoolean(instance, evaluatedProps, evaluatedItems); err != nil {
			//nolint:errcheck
			result.AddError(err)
		}
		return result, evaluatedProps, evaluatedItems
	}

	// Compile patterns if needed
	if s.PatternProperties != nil {
		s.compilePatterns()
	}

	// Process references
	s.processReferences(instance, dynamicScope, result, evaluatedProps, evaluatedItems)

	// Process validation keywords
	s.processValidationKeywords(instance, dynamicScope, result, evaluatedProps, evaluatedItems)

	return result, evaluatedProps, evaluatedItems
}

// preprocessByteInput handles []byte input intelligently
func (s *Schema) preprocessByteInput(instance any) any {
	jsonBytes, ok := instance.([]byte)
	if !ok {
		return instance
	}

	parsed, err := s.processJSONBytes(jsonBytes)
	if err != nil {
		// Create a temporary result to hold the JSON parsing error
		// Return the error as part of the instance for downstream handling
		return &jsonParseError{data: jsonBytes, err: err}
	}

	return parsed
}

// jsonParseError wraps JSON parsing errors for downstream handling
type jsonParseError struct {
	data []byte
	err  error
}

// processReferences handles $ref and $dynamicRef evaluation
func (s *Schema) processReferences(instance any, dynamicScope *DynamicScope, result *EvaluationResult, evaluatedProps map[string]bool, evaluatedItems map[int]bool) {
	// Handle JSON parse errors
	if _, ok := instance.(*jsonParseError); ok {
		//nolint:errcheck
		result.AddError(NewEvaluationError("format", "invalid_json", "Invalid JSON format in byte array"))
		return
	}

	// Process $ref
	if s.ResolvedRef != nil {
		refResult, props, items := s.ResolvedRef.evaluate(instance, dynamicScope)
		if refResult != nil {
			//nolint:errcheck
			result.AddDetail(refResult)
			if !refResult.IsValid() {
				//nolint:errcheck
				result.AddError(NewEvaluationError("$ref", "ref_mismatch", "Value does not match the reference schema"))
			}
		}
		mergeStringMaps(evaluatedProps, props)
		mergeIntMaps(evaluatedItems, items)
	}

	// Process $dynamicRef
	if s.ResolvedDynamicRef != nil {
		s.processDynamicRef(instance, dynamicScope, result, evaluatedProps, evaluatedItems)
	}
}

// processDynamicRef handles $dynamicRef evaluation
func (s *Schema) processDynamicRef(instance any, dynamicScope *DynamicScope, result *EvaluationResult, evaluatedProps map[string]bool, evaluatedItems map[int]bool) {
	anchorSchema := s.ResolvedDynamicRef
	_, anchor := splitRef(s.DynamicRef)

	if !isJSONPointer(anchor) {
		if dynamicAnchor := s.ResolvedDynamicRef.DynamicAnchor; dynamicAnchor != "" {
			if schema := dynamicScope.LookupDynamicAnchor(dynamicAnchor); schema != nil {
				anchorSchema = schema
			}
		}
	}

	dynamicRefResult, props, items := anchorSchema.evaluate(instance, dynamicScope)
	if dynamicRefResult != nil {
		//nolint:errcheck
		result.AddDetail(dynamicRefResult)
		if !dynamicRefResult.IsValid() {
			//nolint:errcheck
			result.AddError(NewEvaluationError("$dynamicRef", "dynamic_ref_mismatch", "Value does not match the dynamic reference schema"))
		}
	}

	mergeStringMaps(evaluatedProps, props)
	mergeIntMaps(evaluatedItems, items)
}

// processValidationKeywords handles all validation keywords
func (s *Schema) processValidationKeywords(instance any, dynamicScope *DynamicScope, result *EvaluationResult, evaluatedProps map[string]bool, evaluatedItems map[int]bool) {
	// Basic type validation
	s.processBasicValidation(instance, result)

	// Logical operations
	s.processLogicalOperations(instance, dynamicScope, result, evaluatedProps, evaluatedItems)

	// Conditional logic
	s.processConditionalLogic(instance, dynamicScope, result, evaluatedProps, evaluatedItems)

	// Type-specific validation
	s.processTypeSpecificValidation(instance, dynamicScope, result, evaluatedProps, evaluatedItems)

	// Content validation
	s.processContentValidation(instance, dynamicScope, result, evaluatedProps, evaluatedItems)
}

// processBasicValidationWithoutRefs handles basic validation without following references (for circular reference cases)
func (s *Schema) processBasicValidationWithoutRefs(instance any, result *EvaluationResult, evaluatedProps map[string]bool, evaluatedItems map[int]bool) {
	// Process basic validation that doesn't involve references
	s.processBasicValidation(instance, result)

	// Process type-specific validation but without following schema references
	if s.hasNumericValidation() {
		errors := evaluateNumeric(s, instance)
		s.addErrors(result, errors)
	}

	if s.hasStringValidation() {
		errors := evaluateString(s, instance)
		s.addErrors(result, errors)
	}

	if s.Format != nil {
		if err := evaluateFormat(s, instance); err != nil {
			//nolint:errcheck
			result.AddError(err)
		}
	}

	// For object validation, only validate basic constraints without following references
	if s.hasObjectValidation() {
		s.processObjectValidationWithoutRefs(instance, result, evaluatedProps)
	}

	// For array validation, validate basic constraints without following item references
	if s.hasArrayValidation() {
		s.processArrayValidationWithoutRefs(instance, result, evaluatedItems)
	}
}

// processBasicValidation handles basic validation keywords
func (s *Schema) processBasicValidation(instance any, result *EvaluationResult) {
	if s.Type != nil {
		if err := evaluateType(s, instance); err != nil {
			//nolint:errcheck
			result.AddError(err)
		}
	}

	if s.Enum != nil {
		if err := evaluateEnum(s, instance); err != nil {
			//nolint:errcheck
			result.AddError(err)
		}
	}

	if s.Const != nil {
		if err := evaluateConst(s, instance); err != nil {
			//nolint:errcheck
			result.AddError(err)
		}
	}
}

// processLogicalOperations handles allOf, anyOf, oneOf, not
func (s *Schema) processLogicalOperations(instance any, dynamicScope *DynamicScope, result *EvaluationResult, evaluatedProps map[string]bool, evaluatedItems map[int]bool) {
	if s.AllOf != nil {
		results, err := evaluateAllOf(s, instance, evaluatedProps, evaluatedItems, dynamicScope)
		s.addResultsAndError(result, results, err)
	}

	if s.AnyOf != nil {
		results, err := evaluateAnyOf(s, instance, evaluatedProps, evaluatedItems, dynamicScope)
		s.addResultsAndError(result, results, err)
	}

	if s.OneOf != nil {
		results, err := evaluateOneOf(s, instance, evaluatedProps, evaluatedItems, dynamicScope)
		s.addResultsAndError(result, results, err)
	}

	if s.Not != nil {
		evalResult, err := evaluateNot(s, instance, evaluatedProps, evaluatedItems, dynamicScope)
		if evalResult != nil {
			//nolint:errcheck
			result.AddDetail(evalResult)
		}
		if err != nil {
			//nolint:errcheck
			result.AddError(err)
		}
	}
}

// processConditionalLogic handles if/then/else
func (s *Schema) processConditionalLogic(instance any, dynamicScope *DynamicScope, result *EvaluationResult, evaluatedProps map[string]bool, evaluatedItems map[int]bool) {
	if s.If != nil || s.Then != nil || s.Else != nil {
		results, err := evaluateConditional(s, instance, evaluatedProps, evaluatedItems, dynamicScope)
		s.addResultsAndError(result, results, err)
	}
}

// processTypeSpecificValidation handles array, object, string, and numeric validation
func (s *Schema) processTypeSpecificValidation(instance any, dynamicScope *DynamicScope, result *EvaluationResult, evaluatedProps map[string]bool, evaluatedItems map[int]bool) {
	// Array validation
	if s.hasArrayValidation() {
		results, errors := evaluateArray(s, instance, evaluatedProps, evaluatedItems, dynamicScope)
		s.addResultsAndErrors(result, results, errors)
	}

	// Numeric validation
	if s.hasNumericValidation() {
		errors := evaluateNumeric(s, instance)
		s.addErrors(result, errors)
	}

	// String validation
	if s.hasStringValidation() {
		errors := evaluateString(s, instance)
		s.addErrors(result, errors)
	}

	if s.Format != nil {
		if err := evaluateFormat(s, instance); err != nil {
			//nolint:errcheck
			result.AddError(err)
		}
	}

	// Object validation
	if s.hasObjectValidation() {
		results, errors := evaluateObject(s, instance, evaluatedProps, evaluatedItems, dynamicScope)
		s.addResultsAndErrors(result, results, errors)
	}

	// Dependent schemas
	if s.DependentSchemas != nil {
		results, err := evaluateDependentSchemas(s, instance, evaluatedProps, evaluatedItems, dynamicScope)
		s.addResultsAndError(result, results, err)
	}

	// Unevaluated properties and items
	s.processUnevaluatedValidation(instance, dynamicScope, result, evaluatedProps, evaluatedItems)
}

// processContentValidation handles content encoding/media type/schema
func (s *Schema) processContentValidation(instance any, dynamicScope *DynamicScope, result *EvaluationResult, evaluatedProps map[string]bool, evaluatedItems map[int]bool) {
	if s.ContentEncoding != nil || s.ContentMediaType != nil || s.ContentSchema != nil {
		contentResult, err := evaluateContent(s, instance, evaluatedProps, evaluatedItems, dynamicScope)
		if contentResult != nil {
			//nolint:errcheck
			result.AddDetail(contentResult)
		}
		if err != nil {
			//nolint:errcheck
			result.AddError(err)
		}
	}
}

// processUnevaluatedValidation handles unevaluated properties and items
func (s *Schema) processUnevaluatedValidation(instance any, dynamicScope *DynamicScope, result *EvaluationResult, evaluatedProps map[string]bool, evaluatedItems map[int]bool) {
	if s.UnevaluatedProperties != nil {
		results, err := evaluateUnevaluatedProperties(s, instance, evaluatedProps, evaluatedItems, dynamicScope)
		s.addResultsAndError(result, results, err)
	}

	if s.UnevaluatedItems != nil {
		results, err := evaluateUnevaluatedItems(s, instance, evaluatedProps, evaluatedItems, dynamicScope)
		s.addResultsAndError(result, results, err)
	}
}

// Helper methods for checking if schema has specific validation types.
func (s *Schema) hasArrayValidation() bool {
	return len(s.PrefixItems) > 0 || s.Items != nil || s.Contains != nil ||
		s.MaxContains != nil || s.MinContains != nil || s.MaxItems != nil ||
		s.MinItems != nil || s.UniqueItems != nil
}

func (s *Schema) hasNumericValidation() bool {
	return s.MultipleOf != nil || s.Maximum != nil || s.ExclusiveMaximum != nil ||
		s.Minimum != nil || s.ExclusiveMinimum != nil
}

func (s *Schema) hasStringValidation() bool {
	return s.MaxLength != nil || s.MinLength != nil || s.Pattern != nil
}

func (s *Schema) hasObjectValidation() bool {
	return s.Properties != nil || s.PatternProperties != nil || s.AdditionalProperties != nil ||
		s.PropertyNames != nil || s.MaxProperties != nil || s.MinProperties != nil ||
		len(s.Required) > 0 || len(s.DependentRequired) > 0
}

// Helper methods for adding results and errors.
func (s *Schema) addResultsAndError(result *EvaluationResult, results []*EvaluationResult, err *EvaluationError) {
	for _, res := range results {
		//nolint:errcheck
		result.AddDetail(res)
	}
	if err != nil {
		//nolint:errcheck
		result.AddError(err)
	}
}

func (s *Schema) addResultsAndErrors(result *EvaluationResult, results []*EvaluationResult, errors []*EvaluationError) {
	for _, res := range results {
		//nolint:errcheck
		result.AddDetail(res)
	}
	s.addErrors(result, errors)
}

func (s *Schema) addErrors(result *EvaluationResult, errors []*EvaluationError) {
	for _, err := range errors {
		//nolint:errcheck
		result.AddError(err)
	}
}

func (s *Schema) evaluateBoolean(instance any, evaluatedProps map[string]bool, evaluatedItems map[int]bool) *EvaluationError {
	if s.Boolean == nil {
		return nil
	}

	if *s.Boolean {
		switch v := instance.(type) {
		case map[string]any:
			for key := range v {
				evaluatedProps[key] = true
			}
		case []any:
			for index := range v {
				evaluatedItems[index] = true
			}
		}
		return nil
	}

	return NewEvaluationError("schema", "false_schema_mismatch", "No values are allowed because the schema is set to 'false'")
}

// evaluateObject groups the validation of all object-specific keywords.
func evaluateObject(schema *Schema, data any, evaluatedProps map[string]bool, evaluatedItems map[int]bool, dynamicScope *DynamicScope) ([]*EvaluationResult, []*EvaluationError) {
	// Fast path: Direct type assertions for common map types (5-10x faster than reflection)
	// This optimization follows the pattern from uniqueItems.go:94-164
	switch obj := data.(type) {
	case map[string]any:
		// Most common case: already map[string]any
		return evaluateObjectMap(schema, obj, evaluatedProps, evaluatedItems, dynamicScope)

	case map[string]string:
		// Common case: form data, query params, headers
		converted := make(map[string]any, len(obj))
		for k, v := range obj {
			converted[k] = v
		}
		return evaluateObjectMap(schema, converted, evaluatedProps, evaluatedItems, dynamicScope)

	case map[string]int:
		// Common case: counters, metrics, config with int values
		converted := make(map[string]any, len(obj))
		for k, v := range obj {
			converted[k] = v
		}
		return evaluateObjectMap(schema, converted, evaluatedProps, evaluatedItems, dynamicScope)

	case map[string]int64:
		// Common case: timestamps, IDs
		converted := make(map[string]any, len(obj))
		for k, v := range obj {
			converted[k] = v
		}
		return evaluateObjectMap(schema, converted, evaluatedProps, evaluatedItems, dynamicScope)

	case map[string]float64:
		// Common case: numeric data, coordinates
		converted := make(map[string]any, len(obj))
		for k, v := range obj {
			converted[k] = v
		}
		return evaluateObjectMap(schema, converted, evaluatedProps, evaluatedItems, dynamicScope)

	case map[string]bool:
		// Common case: feature flags, boolean configs
		converted := make(map[string]any, len(obj))
		for k, v := range obj {
			converted[k] = v
		}
		return evaluateObjectMap(schema, converted, evaluatedProps, evaluatedItems, dynamicScope)
	}

	// Slow path: Use reflection for uncommon types (structs, interfaces, other map types)
	rv := reflect.ValueOf(data)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, nil
		}
		rv = rv.Elem()
	}

	//nolint:exhaustive,nolintlint // Only handling Struct and Map kinds - other types use default fallback
	switch rv.Kind() {
	case reflect.Struct:
		return evaluateObjectStruct(schema, rv, evaluatedProps, evaluatedItems, dynamicScope)
	case reflect.Map:
		if rv.Type().Key().Kind() == reflect.String {
			return evaluateObjectReflectMap(schema, rv, evaluatedProps, evaluatedItems, dynamicScope)
		}
	default:
		// Handle other kinds by returning nil
	}

	return nil, nil
}

// evaluateObjectMap handles validation for map[string]any (original implementation)
func evaluateObjectMap(schema *Schema, object map[string]any, evaluatedProps map[string]bool, evaluatedItems map[int]bool, dynamicScope *DynamicScope) ([]*EvaluationResult, []*EvaluationError) {
	var results []*EvaluationResult
	var errors []*EvaluationError

	// Properties validation
	if schema.Properties != nil {
		if propResults, propError := evaluateProperties(schema, object, evaluatedProps, evaluatedItems, dynamicScope); propResults != nil || propError != nil {
			results = append(results, propResults...)
			if propError != nil {
				errors = append(errors, propError)
			}
		}
	}

	// Pattern properties validation
	if schema.PatternProperties != nil {
		if patResults, patError := evaluatePatternProperties(schema, object, evaluatedProps, evaluatedItems, dynamicScope); patResults != nil || patError != nil {
			results = append(results, patResults...)
			if patError != nil {
				errors = append(errors, patError)
			}
		}
	}

	// Additional properties validation
	if schema.AdditionalProperties != nil {
		if addResults, addError := evaluateAdditionalProperties(schema, object, evaluatedProps, evaluatedItems, dynamicScope); addResults != nil || addError != nil {
			results = append(results, addResults...)
			if addError != nil {
				errors = append(errors, addError)
			}
		}
	}

	// Property names validation
	if schema.PropertyNames != nil {
		if nameResults, nameError := evaluatePropertyNames(schema, object, evaluatedProps, evaluatedItems, dynamicScope); nameResults != nil || nameError != nil {
			results = append(results, nameResults...)
			if nameError != nil {
				errors = append(errors, nameError)
			}
		}
	}

	// Object constraint validation
	errors = append(errors, validateObjectConstraints(schema, object)...)

	return results, errors
}

// validateObjectConstraints validates object-specific constraints.
func validateObjectConstraints(schema *Schema, object map[string]any) []*EvaluationError {
	var errors []*EvaluationError

	if schema.MaxProperties != nil {
		if err := evaluateMaxProperties(schema, object); err != nil {
			errors = append(errors, err)
		}
	}

	if schema.MinProperties != nil {
		if err := evaluateMinProperties(schema, object); err != nil {
			errors = append(errors, err)
		}
	}

	if len(schema.Required) > 0 {
		if err := evaluateRequired(schema, object); err != nil {
			errors = append(errors, err)
		}
	}

	if len(schema.DependentRequired) > 0 {
		if err := evaluateDependentRequired(schema, object); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// evaluateNumeric groups the validation of all numeric-specific keywords.
func evaluateNumeric(schema *Schema, data any) []*EvaluationError {
	dataType := getDataType(data)
	if dataType != "number" && dataType != "integer" {
		return nil
	}

	value := NewRat(data)
	if value == nil {
		return []*EvaluationError{
			NewEvaluationError("type", "invalid_numeric", "Value is {received} but should be numeric", map[string]any{
				"actual_type": dataType,
			}),
		}
	}

	var errors []*EvaluationError

	// Collect all numeric validation errors
	if schema.MultipleOf != nil {
		if err := evaluateMultipleOf(schema, value); err != nil {
			errors = append(errors, err)
		}
	}

	if schema.Maximum != nil {
		if err := evaluateMaximum(schema, value); err != nil {
			errors = append(errors, err)
		}
	}

	if schema.ExclusiveMaximum != nil {
		if err := evaluateExclusiveMaximum(schema, value); err != nil {
			errors = append(errors, err)
		}
	}

	if schema.Minimum != nil {
		if err := evaluateMinimum(schema, value); err != nil {
			errors = append(errors, err)
		}
	}

	if schema.ExclusiveMinimum != nil {
		if err := evaluateExclusiveMinimum(schema, value); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// evaluateString groups the validation of all string-specific keywords.
func evaluateString(schema *Schema, data any) []*EvaluationError {
	value, ok := data.(string)
	if !ok {
		return nil
	}

	var errors []*EvaluationError

	// Collect all string validation errors
	if schema.MaxLength != nil {
		if err := evaluateMaxLength(schema, value); err != nil {
			errors = append(errors, err)
		}
	}

	if schema.MinLength != nil {
		if err := evaluateMinLength(schema, value); err != nil {
			errors = append(errors, err)
		}
	}

	if schema.Pattern != nil {
		if err := evaluatePattern(schema, value); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// evaluateArray groups the validation of all array-specific keywords.
func evaluateArray(schema *Schema, data any, evaluatedProps map[string]bool, evaluatedItems map[int]bool, dynamicScope *DynamicScope) ([]*EvaluationResult, []*EvaluationError) {
	items, ok := data.([]any)
	if !ok {
		return nil, nil
	}

	var results []*EvaluationResult
	var errors []*EvaluationError

	// Process array schema validations
	arrayValidations := []func(*Schema, []any, map[string]bool, map[int]bool, *DynamicScope) ([]*EvaluationResult, *EvaluationError){
		evaluatePrefixItems,
		evaluateItems,
		evaluateContains,
	}

	for _, validate := range arrayValidations {
		if res, err := validate(schema, items, evaluatedProps, evaluatedItems, dynamicScope); res != nil || err != nil {
			if res != nil {
				results = append(results, res...)
			}
			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	// Array constraint validation
	errors = append(errors, validateArrayConstraints(schema, items)...)

	return results, errors
}

// validateArrayConstraints validates array-specific constraints.
func validateArrayConstraints(schema *Schema, items []any) []*EvaluationError {
	var errors []*EvaluationError

	if schema.MaxItems != nil {
		if err := evaluateMaxItems(schema, items); err != nil {
			errors = append(errors, err)
		}
	}

	if schema.MinItems != nil {
		if err := evaluateMinItems(schema, items); err != nil {
			errors = append(errors, err)
		}
	}

	if schema.UniqueItems != nil && *schema.UniqueItems {
		if err := evaluateUniqueItems(schema, items); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// DynamicScope struct defines a stack specifically for handling Schema types.
type DynamicScope struct {
	schemas []*Schema // Slice storing pointers to Schema
}

// NewDynamicScope creates and returns a new empty DynamicScope.
func NewDynamicScope() *DynamicScope {
	return &DynamicScope{schemas: make([]*Schema, 0)}
}

// Push adds a Schema to the dynamic scope.
func (ds *DynamicScope) Push(schema *Schema) {
	ds.schemas = append(ds.schemas, schema)
}

// Pop removes and returns the top Schema from the dynamic scope.
func (ds *DynamicScope) Pop() *Schema {
	if len(ds.schemas) == 0 {
		return nil
	}
	lastIndex := len(ds.schemas) - 1
	schema := ds.schemas[lastIndex]
	ds.schemas = ds.schemas[:lastIndex]
	return schema
}

// Peek returns the top Schema without removing it.
func (ds *DynamicScope) Peek() *Schema {
	if len(ds.schemas) == 0 {
		return nil
	}
	return ds.schemas[len(ds.schemas)-1]
}

// IsEmpty checks if the dynamic scope is empty.
func (ds *DynamicScope) IsEmpty() bool {
	return len(ds.schemas) == 0
}

// Size returns the number of Schemas in the dynamic scope.
func (ds *DynamicScope) Size() int {
	return len(ds.schemas)
}

// LookupDynamicAnchor searches for a dynamic anchor in the dynamic scope.
func (ds *DynamicScope) LookupDynamicAnchor(anchor string) *Schema {
	// use the first schema dynamic anchor matching the anchor
	for _, schema := range ds.schemas {
		if schema.dynamicAnchors != nil && schema.dynamicAnchors[anchor] != nil {
			return schema.dynamicAnchors[anchor]
		}
	}

	return nil
}

// Contains checks if a schema is already in the dynamic scope (circular reference detection).
func (ds *DynamicScope) Contains(schema *Schema) bool {
	for _, s := range ds.schemas {
		if s == schema {
			return true
		}
	}
	return false
}

// isByteSlice checks if the given value is a []byte type definition (like json.RawMessage)
func isByteSlice(v any) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8
}

// convertToByteSlice converts a []byte type definition to []byte
func convertToByteSlice(v any) ([]byte, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
		return rv.Bytes(), true
	}
	return nil, false
}

// processObjectValidationWithoutRefs validates object constraints without following schema references
func (s *Schema) processObjectValidationWithoutRefs(instance any, result *EvaluationResult, evaluatedProps map[string]bool) {
	// Fast path: direct map[string]any type
	if object, ok := instance.(map[string]any); ok {
		// Validate basic object constraints
		errors := validateObjectConstraints(s, object)
		s.addErrors(result, errors)

		// Check additional properties constraint for circular references
		if s.AdditionalProperties != nil {
			s.checkAdditionalPropertiesForCircular(object, result, evaluatedProps)
		} else {
			// Mark all properties as evaluated if no additional properties constraint
			for key := range object {
				evaluatedProps[key] = true
			}
		}
		return
	}

	// For struct types, validate basic constraints
	rv := reflect.ValueOf(instance)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}

	if rv.Kind() == reflect.Struct {
		// Convert struct to map for constraint validation
		objectMap := make(map[string]any)
		structType := rv.Type()
		for i := 0; i < rv.NumField(); i++ {
			field := structType.Field(i)
			if field.IsExported() {
				fieldValue := rv.Field(i)
				if fieldValue.CanInterface() {
					objectMap[field.Name] = fieldValue.Interface()
					evaluatedProps[field.Name] = true
				}
			}
		}

		errors := validateObjectConstraints(s, objectMap)
		s.addErrors(result, errors)

		// Check additional properties for struct as well
		if s.AdditionalProperties != nil {
			s.checkAdditionalPropertiesForCircular(objectMap, result, evaluatedProps)
		}
	}
}

// processArrayValidationWithoutRefs validates array constraints without following item schema references
func (s *Schema) processArrayValidationWithoutRefs(instance any, result *EvaluationResult, evaluatedItems map[int]bool) {
	items, ok := instance.([]any)
	if !ok {
		return
	}

	// Only validate basic array constraints, not item schemas
	errors := validateArrayConstraints(s, items)
	s.addErrors(result, errors)

	// Mark all items as evaluated to prevent further processing
	for i := range items {
		evaluatedItems[i] = true
	}
}

// checkAdditionalPropertiesForCircular validates additionalProperties constraint for circular references
func (s *Schema) checkAdditionalPropertiesForCircular(object map[string]any, result *EvaluationResult, evaluatedProps map[string]bool) {
	// Check if additionalProperties is false (no additional properties allowed)
	if s.AdditionalProperties.Boolean != nil && !*s.AdditionalProperties.Boolean {
		// Get list of properties defined in the schema
		allowedProps := make(map[string]bool)
		if s.Properties != nil {
			for prop := range *s.Properties {
				allowedProps[prop] = true
			}
		}

		// Check if all object properties are allowed
		for prop := range object {
			if !allowedProps[prop] {
				//nolint:errcheck
				result.AddError(NewEvaluationError("additionalProperties", "additional_property_false",
					"Additional property '{property}' not allowed", map[string]any{
						"property": prop,
					}))
			} else {
				evaluatedProps[prop] = true
			}
		}
	} else {
		// Mark all properties as evaluated if additional properties are allowed
		for key := range object {
			evaluatedProps[key] = true
		}
	}
}

// isProblematicCircularReference determines if this circular reference would cause infinite recursion
func (s *Schema) isProblematicCircularReference(scope *DynamicScope) bool {
	depth := 0
	for _, schema := range scope.schemas {
		if schema == s {
			depth++
		}
	}

	// Use different thresholds based on the type of reference and context

	// For metaschema validation (remote references), be very permissive
	// These are legitimate validation scenarios, not circular references
	if s.ID != "" && strings.Contains(s.ID, "json-schema.org") {
		return depth > 5 // High threshold for metaschema
	}

	// For schemas that are clearly self-referential by design, allow more depth
	if s.hasSelfReferentialPattern() {
		return depth > 10 // Allow reasonable nesting for self-referential schemas
	}

	// For other cases, use a moderate threshold
	return depth > 3
}

// hasSelfReferentialPattern checks if schema has patterns designed for self-reference
func (s *Schema) hasSelfReferentialPattern() bool {
	// Schema has a property that references itself
	if s.Properties != nil {
		for _, prop := range *s.Properties {
			if prop.Ref == "#" {
				return true
			}
		}
	}

	// Schema has array items that reference itself
	if s.Items != nil && s.Items.Ref == "#" {
		return true
	}

	// Schema has definitions that might create recursive patterns
	if s.Defs != nil {
		for _, def := range s.Defs {
			if def.Ref == "#" {
				return true
			}
		}
	}

	return false
}
