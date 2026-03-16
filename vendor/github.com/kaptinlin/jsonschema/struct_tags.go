package jsonschema

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/kaptinlin/jsonschema/pkg/tagparser"
)

// StructTagError represents an error that occurred during struct tag processing.
// It provides detailed context about which struct, field, and tag rule caused the error.
type StructTagError struct {
	StructType string // The type name of the struct being processed
	FieldName  string // The name of the field with the error
	TagRule    string // The tag rule that failed (e.g., "pattern=...")
	Message    string // Human-readable error message
	Err        error  // Underlying error (renamed from Cause for consistency with UnmarshalError)
}

// Error returns a formatted error message with full context.
func (e *StructTagError) Error() string {
	var sb strings.Builder
	sb.WriteString("struct tag error")

	var parts []string
	if e.StructType != "" {
		parts = append(parts, "struct="+e.StructType)
	}
	if e.FieldName != "" {
		parts = append(parts, "field="+e.FieldName)
	}
	if e.TagRule != "" {
		parts = append(parts, "rule="+e.TagRule)
	}
	if len(parts) > 0 {
		sb.WriteString(" (")
		sb.WriteString(strings.Join(parts, ", "))
		sb.WriteByte(')')
	}

	if e.Message != "" {
		sb.WriteString(": ")
		sb.WriteString(e.Message)
	}

	if e.Err != nil {
		sb.WriteString(": ")
		sb.WriteString(e.Err.Error())
	}

	return sb.String()
}

// Unwrap returns the underlying error, allowing error chain inspection with errors.Is/As.
func (e *StructTagError) Unwrap() error {
	return e.Err
}

// Import schemagen components for reuse
// Note: Since schemagen is in cmd/schemagen and we're in the main package,
// we'll reimplement the required components adapted for runtime use

// RequiredSort controls how required field names are ordered
type RequiredSort string

const (
	// RequiredSortAlphabetical sorts required fields alphabetically for deterministic output
	RequiredSortAlphabetical RequiredSort = "alphabetical"

	// RequiredSortNone does not sort required fields, preserving the order from struct field iteration
	// Note: May be non-deterministic due to map iteration in TagParser
	RequiredSortNone RequiredSort = "none"
)

// StructTagOptions holds configuration for struct tag schema generation
type StructTagOptions struct {
	TagName             string              // tag name to parse (default: "jsonschema")
	AllowUntaggedFields bool                // whether to include fields without tags (default: false)
	DefaultRequired     bool                // whether fields are required by default (default: false)
	FieldNameMapper     func(string) string // function to map Go field names to JSON names
	CustomValidators    map[string]any      // custom validators (for future extension)
	CacheEnabled        bool                // whether to enable schema caching (default: true)
	SchemaVersion       string              // $schema URI to include in generated schemas (empty string = omit $schema, default = Draft 2020-12)
	RequiredSort        RequiredSort        // controls ordering of required fields (default: RequiredSortAlphabetical)

	// Schema-level properties using map approach
	SchemaProperties map[string]any // flexible configuration for any schema property
}

// CustomValidatorFunc represents a custom validator function
type CustomValidatorFunc func(_ reflect.Type, params []string) []Keyword

// ValidatorRegistry manages custom validators
type ValidatorRegistry struct {
	validators map[string]CustomValidatorFunc
	mutex      sync.RWMutex
}

var globalValidatorRegistry = &ValidatorRegistry{
	validators: make(map[string]CustomValidatorFunc),
}

// RegisterCustomValidator registers a custom validator globally
func RegisterCustomValidator(name string, validator CustomValidatorFunc) {
	globalValidatorRegistry.mutex.Lock()
	defer globalValidatorRegistry.mutex.Unlock()
	globalValidatorRegistry.validators[name] = validator
}

// GetCustomValidator retrieves a custom validator by name
func (r *ValidatorRegistry) GetCustomValidator(name string) (CustomValidatorFunc, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	validator, exists := r.validators[name]
	return validator, exists
}

// DefaultStructTagOptions returns the default configuration for struct tag processing
func DefaultStructTagOptions() *StructTagOptions {
	return &StructTagOptions{
		TagName:             "jsonschema",
		AllowUntaggedFields: false,
		DefaultRequired:     false,
		FieldNameMapper:     nil, // use default field naming
		CustomValidators:    make(map[string]any),
		CacheEnabled:        true,
		SchemaVersion:       "https://json-schema.org/draft/2020-12/schema", // default to JSON Schema Draft 2020-12
		RequiredSort:        RequiredSortAlphabetical,                       // default to alphabetical sorting for determinism

		// Schema-level properties - empty by default (not set)
		SchemaProperties: nil, // nil = no schema properties set
	}
}

// normalizeOptions ensures options fields have valid defaults. Returns new options if nil.
// Creates a copy to avoid mutating the input.
func normalizeOptions(options *StructTagOptions) *StructTagOptions {
	if options == nil {
		return DefaultStructTagOptions()
	}

	// Create a copy to avoid mutating the input
	normalized := *options

	// Set defaults for empty fields
	if normalized.TagName == "" {
		normalized.TagName = "jsonschema"
	}
	if normalized.CustomValidators == nil {
		normalized.CustomValidators = make(map[string]any)
	}
	if normalized.RequiredSort == "" {
		normalized.RequiredSort = RequiredSortAlphabetical
	}

	return &normalized
}

// structTagGenerator handles runtime struct tag schema generation with reused schemagen logic
type structTagGenerator struct {
	options      *StructTagOptions
	tagParser    *tagparser.TagParser // Use the real tagparser
	typeMapping  map[string]func(...Keyword) *Schema
	validatorMap map[string]validatorFunc
	// Dependency tracking (simplified from schemagen ReferenceAnalyzer)
	visited       map[reflect.Type]int    // 0=unvisited, 1=visiting, 2=completed
	definitions   map[string]*Schema      // $defs storage
	generatedRefs map[reflect.Type]string // track generated $refs
}

// validatorFunc represents a function that converts tag parameters to Schema keywords
type validatorFunc func(_ reflect.Type, params []string) []Keyword

var (
	// Global schema cache for improved performance across multiple calls
	globalSchemaCache sync.Map // map[cacheKey]*Schema
)

// cacheKey represents a unique key for caching schemas
type cacheKey struct {
	structType          reflect.Type
	tagName             string
	allowUntaggedFields bool
	defaultRequired     bool
	cacheEnabled        bool
	schemaVersion       string // include schema version in cache key to ensure different versions are cached separately
	// Note: we don't include function pointers in the cache key as they can't be compared
}

// newStructTagGenerator creates a new struct tag generator with the given options
func newStructTagGenerator(options *StructTagOptions) *structTagGenerator {
	options = normalizeOptions(options)

	return &structTagGenerator{
		options:       options,
		tagParser:     tagparser.NewWithTagName(options.TagName), // Use real tagparser
		typeMapping:   createRuntimeTypeMapping(),
		validatorMap:  createRuntimeValidatorMapping(),
		visited:       make(map[reflect.Type]int),
		definitions:   make(map[string]*Schema),
		generatedRefs: make(map[reflect.Type]string),
	}
}

// FromStruct generates a JSON Schema from a struct type with jsonschema tags.
func FromStruct[T any]() (*Schema, error) {
	return FromStructWithOptions[T](nil)
}

// FromStructWithOptions generates a JSON Schema from a struct type with custom options.
func FromStructWithOptions[T any](options *StructTagOptions) (*Schema, error) {
	var zero T
	structType := reflect.TypeOf(zero)

	// Normalize options with defaults
	options = normalizeOptions(options)

	// Check global cache first if enabled
	if options.CacheEnabled {
		key := cacheKey{
			structType:          structType,
			tagName:             options.TagName,
			allowUntaggedFields: options.AllowUntaggedFields,
			defaultRequired:     options.DefaultRequired,
			cacheEnabled:        options.CacheEnabled,
			schemaVersion:       options.SchemaVersion,
		}
		if cached, ok := globalSchemaCache.Load(key); ok {
			return cached.(*Schema), nil
		}
	}

	// Create generator for this call (allows different options per call)
	generator := newStructTagGenerator(options)

	// Generate schema using reused schemagen logic
	schema, err := generator.generateSchemaWithDependencyAnalysis(structType)
	if err != nil {
		return nil, err
	}

	// Set $schema if specified in options (empty string means omit $schema)
	if options.SchemaVersion != "" {
		schema.Schema = options.SchemaVersion
	}

	// Apply schema-level properties
	applySchemaProperties(schema, options)

	// Add $defs if there are any circular references
	if len(generator.definitions) > 0 {
		defsMap := make(SchemaMap)
		for name, defSchema := range generator.definitions {
			// Create a clean copy of the definition schema to avoid circular references
			cleanDef := generator.createCleanDefinition(defSchema)
			defsMap[name] = cleanDef
		}
		schema.Defs = defsMap
	}

	// Resolve all references to ensure ResolvedRef fields are populated
	// This is critical for validation to work correctly with nested structs
	schema.resolveReferences()

	if err := schema.validateRegexSyntax(); err != nil {
		return nil, err
	}

	// Clean up visited state
	generator.visited = make(map[reflect.Type]int)

	// Cache the result globally if caching is enabled
	if options.CacheEnabled {
		key := cacheKey{
			structType:          structType,
			tagName:             options.TagName,
			allowUntaggedFields: options.AllowUntaggedFields,
			defaultRequired:     options.DefaultRequired,
			cacheEnabled:        options.CacheEnabled,
			schemaVersion:       options.SchemaVersion,
		}
		globalSchemaCache.Store(key, schema)
	}

	return schema, nil
}

// ClearSchemaCache clears the global schema cache - useful for testing and memory management
func ClearSchemaCache() {
	globalSchemaCache.Range(func(key, _ any) bool {
		globalSchemaCache.Delete(key)
		return true
	})
}

// GetCacheStats returns statistics about the global schema cache - useful for monitoring
func GetCacheStats() map[string]int {
	stats := map[string]int{
		"cached_schemas": 0,
	}

	globalSchemaCache.Range(func(_, _ any) bool {
		stats["cached_schemas"]++
		return true
	})

	return stats
}

// generateSchemaWithDependencyAnalysis generates schema using schemagen-style dependency analysis
func (g *structTagGenerator) generateSchemaWithDependencyAnalysis(structType reflect.Type) (*Schema, error) {
	// Handle pointers
	for structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	// Ensure it's a struct
	if structType.Kind() != reflect.Struct {
		return nil, ErrExpectedStructType
	}

	// Check current visiting state using schemagen-style three-state tracking
	state := g.visited[structType]
	switch state {
	case 1: // Currently visiting - circular reference detected
		return g.createRefSchema(structType), nil
	case 2: // Already completed
		// Return reference to existing schema if available in definitions
		refName := g.getRefName(structType)
		if _, exists := g.definitions[refName]; exists {
			return g.createRefSchema(structType), nil
		}
	}

	// Mark as visiting
	g.visited[structType] = 1

	// Generate unique reference name
	refName := g.getRefName(structType)

	// Always create a placeholder in definitions to ensure we can reference it
	g.definitions[refName] = &Schema{Type: SchemaType{"object"}}

	// Parse struct fields using reflection (adapted from schemagen analyzer logic)
	var properties []Property
	var required []string

	// Use the real tagparser to parse struct fields
	fieldInfos, err := g.tagParser.ParseStructTags(structType)
	if err != nil {
		g.visited[structType] = 0 // Reset on error
		return nil, fmt.Errorf("%w: %w", ErrStructTagParsing, err)
	}

	for _, fieldInfo := range fieldInfos {
		// Skip fields without tags unless explicitly allowed or promoted from embedding
		if !g.options.AllowUntaggedFields && fieldInfo.Tag == "" && !fieldInfo.IsPromoted {
			continue
		}

		// Generate schema for this field using reused schemagen logic
		fieldSchema, err := g.generateFieldSchemaWithValidators(structType, &fieldInfo)
		if err != nil {
			return nil, err
		}

		if fieldSchema != nil {
			properties = append(properties, Prop(fieldInfo.JSONName, fieldSchema))

			// Add to required if field is marked as required or default required is true
			if fieldInfo.Required || (g.options.DefaultRequired && !fieldInfo.Optional) {
				required = append(required, fieldInfo.JSONName)
			}
		}
	}

	// Sort required fields based on RequiredSort option
	if len(required) > 0 {
		if g.options.RequiredSort == RequiredSortAlphabetical {
			sort.Strings(required)
		}
		// For RequiredSortNone, keep the order as-is from struct field iteration
	}

	// Create the object schema
	var keywords []Keyword
	if len(required) > 0 {
		keywords = append(keywords, Required(required...))
	}

	// Convert properties to any slice
	items := make([]any, len(properties))
	for i, prop := range properties {
		items[i] = prop
	}

	// Add keywords to items slice
	for _, keyword := range keywords {
		items = append(items, keyword)
	}

	schema := Object(items...)

	// Store the final schema in definitions
	g.definitions[refName] = schema
	g.generatedRefs[structType] = refName

	// Mark as completed
	g.visited[structType] = 2

	// Always return the actual schema (not a reference) from this function
	// References are handled by the handleStructType function when needed
	return schema, nil
}

// generateFieldSchemaWithValidators generates schema for a field using reused schemagen validator logic
func (g *structTagGenerator) generateFieldSchemaWithValidators(structType reflect.Type, fieldInfo *tagparser.FieldInfo) (*Schema, error) {
	fieldType := fieldInfo.Type
	rules := fieldInfo.Rules

	if err := g.validateFieldRules(structType, fieldInfo); err != nil {
		return nil, err
	}

	// Handle pointer types - make nullable
	var isNullable bool
	for fieldType.Kind() == reflect.Ptr {
		isNullable = true
		fieldType = fieldType.Elem()
	}

	// Get base schema from type using reused schemagen type mapping
	baseSchema, err := g.getSchemaFromTypeWithMapping(fieldType)
	if err != nil {
		return nil, err
	}

	// Parse validation rules from tag using real tagparser rules
	keywords := g.applyValidationRules(rules, fieldType)

	// Handle array schemas with object-level constraints
	if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
		arrayKeywords, itemKeywords := g.separateArrayAndItemKeywords(keywords, fieldType)

		// Apply item-level constraints to the items schema if it exists
		if len(itemKeywords) > 0 && baseSchema.Items != nil {
			enhancedItemSchema := g.cloneSchemaWithKeywords(baseSchema.Items, itemKeywords)
			baseSchema = g.cloneSchemaAndUpdateItems(baseSchema, enhancedItemSchema)
		}

		// Apply array-level constraints to the array schema
		if len(arrayKeywords) > 0 {
			baseSchema = g.cloneSchemaWithKeywords(baseSchema, arrayKeywords)
		}

		// Handle nullable array
		if isNullable {
			nullSchema := Null()
			return AnyOf(baseSchema, nullSchema), nil
		}

		return baseSchema, nil
	}

	// Handle nullable fields
	if isNullable {
		// Apply keywords to base schema first before creating anyOf
		schemaWithRules := baseSchema
		if len(keywords) > 0 {
			schemaWithRules = g.cloneSchemaWithKeywords(baseSchema, keywords)
		}

		// Create anyOf with the enhanced schema and null schema
		nullSchema := Null()
		return AnyOf(schemaWithRules, nullSchema), nil
	}

	// Apply keywords to base schema
	if len(keywords) > 0 {
		// Clone the schema and apply keywords
		newSchema := g.cloneSchemaWithKeywords(baseSchema, keywords)
		return newSchema, nil
	}

	return baseSchema, nil
}

func (g *structTagGenerator) validateFieldRules(structType reflect.Type, fieldInfo *tagparser.FieldInfo) error {
	for _, rule := range fieldInfo.Rules {
		switch rule.Name {
		case "pattern":
			if len(rule.Params) == 0 {
				continue
			}
			if err := compilePattern(rule.Params[0]); err != nil {
				return &StructTagError{
					StructType: structType.String(),
					FieldName:  fieldInfo.Name,
					TagRule:    fmt.Sprintf("pattern=%s", rule.Params[0]),
					Message:    "invalid regular expression pattern",
					Err:        fmt.Errorf("%w: %w", ErrRegexValidation, err),
				}
			}
		case "patternProperties":
			if len(rule.Params) == 0 {
				continue
			}
			if err := compilePattern(rule.Params[0]); err != nil {
				return &StructTagError{
					StructType: structType.String(),
					FieldName:  fieldInfo.Name,
					TagRule:    fmt.Sprintf("patternProperties=%s", rule.Params[0]),
					Message:    "invalid regular expression pattern",
					Err:        fmt.Errorf("%w: %w", ErrRegexValidation, err),
				}
			}
		}
	}

	return nil
}

// getSchemaFromTypeWithMapping converts Go types to JSON Schema using reused schemagen logic
func (g *structTagGenerator) getSchemaFromTypeWithMapping(fieldType reflect.Type) (*Schema, error) {
	kind := fieldType.Kind()

	// Handle basic types using reused type mapping
	if constructor, exists := g.typeMapping[fieldType.String()]; exists {
		return constructor(), nil
	}

	// Handle by kind if specific type not found
	//exhaustive:ignore - we only handle types that are relevant for schema generation
	switch kind {
	case reflect.String:
		return String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Integer(), nil
	case reflect.Float32, reflect.Float64:
		return Number(), nil
	case reflect.Bool:
		return Boolean(), nil
	case reflect.Slice, reflect.Array:
		return g.handleArrayType(fieldType)
	case reflect.Map:
		return g.handleMapType(fieldType)
	case reflect.Struct:
		return g.handleStructType(fieldType)
	case reflect.Interface:
		return Any(), nil
	default:
		return nil, ErrUnsupportedType
	}
}

// handleArrayType handles array/slice types with potential circular references
func (g *structTagGenerator) handleArrayType(fieldType reflect.Type) (*Schema, error) {
	elemType := fieldType.Elem()

	// Handle pointer element types
	for elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	// If element is a struct, handle potential circular reference
	if elemType.Kind() == reflect.Struct {
		// Check for circular reference before generating schema
		if g.visited[elemType] == 1 {
			// For circular reference in array, create a $ref
			refName := g.getRefName(elemType)

			// Store a placeholder in definitions if not already there
			if _, exists := g.definitions[refName]; !exists {
				g.definitions[refName] = &Schema{Type: SchemaType{"object"}}
			}

			// Create array with $ref to the circular type
			refSchema := Ref(fmt.Sprintf("#/$defs/%s", refName))
			return Array(Items(refSchema)), nil
		}

		elemSchema, err := g.generateSchemaWithDependencyAnalysis(elemType)
		if err != nil {
			// Fall back to basic array schema if struct schema fails
			// Return the error instead of ignoring it
			return nil, err
		}

		// Create array schema with proper items constraint
		return Array(Items(elemSchema)), nil
	}

	// For non-struct elements, generate appropriate type schema to ensure array elements have correct type constraints
	elemSchema, err := g.getSchemaFromTypeWithMapping(elemType)
	if err != nil {
		// If unable to generate element schema, fallback to basic array
		return Array(), err
	}
	// Create array schema with type constraints
	return Array(Items(elemSchema)), nil
}

// handleMapType handles map types with additionalProperties for value types
func (g *structTagGenerator) handleMapType(fieldType reflect.Type) (*Schema, error) {
	valueType := fieldType.Elem()

	// Handle pointer value types
	for valueType.Kind() == reflect.Ptr {
		valueType = valueType.Elem()
	}

	// If value is a struct, use handleStructType to get proper $ref
	if valueType.Kind() == reflect.Struct {
		valueSchema, err := g.handleStructType(valueType)
		if err != nil {
			// Fall back to basic object schema if struct schema fails
			return nil, err
		}

		// Create object schema with additionalProperties as $ref to the struct type
		return Object(AdditionalPropsSchema(valueSchema)), nil
	}

	// For non-struct values, generate appropriate type schema
	valueSchema, err := g.getSchemaFromTypeWithMapping(valueType)
	if err != nil {
		// If unable to generate value schema, fallback to basic object
		return Object(), err
	}
	// Create object schema with additionalProperties for the value type
	return Object(AdditionalPropsSchema(valueSchema)), nil
}

// handleStructType handles struct types with circular reference detection and deduplication
func (g *structTagGenerator) handleStructType(fieldType reflect.Type) (*Schema, error) {
	state := g.visited[fieldType]
	refName := g.getRefName(fieldType)

	switch state {
	case 1: // Currently visiting - circular reference detected
		// Store a placeholder in definitions if not already there
		if _, exists := g.definitions[refName]; !exists {
			g.definitions[refName] = &Schema{Type: SchemaType{"object"}}
		}
		// Return $ref to the circular type
		return Ref(fmt.Sprintf("#/$defs/%s", refName)), nil

	case 2: // Already completed - reuse existing definition
		// Struct has already been processed, use a reference
		if _, exists := g.definitions[refName]; exists {
			return Ref(fmt.Sprintf("#/$defs/%s", refName)), nil
		}
		// Fallback: regenerate if definition doesn't exist (shouldn't happen)
		return g.generateSchemaWithDependencyAnalysis(fieldType)

	default: // 0 or unvisited
		// Generate the schema first to populate definitions
		_, err := g.generateSchemaWithDependencyAnalysis(fieldType)
		if err != nil {
			return nil, err
		}

		// After generation, always return a reference to avoid duplication
		// This ensures all struct types are referenced from $defs
		return Ref(fmt.Sprintf("#/$defs/%s", refName)), nil
	}
}

// applyValidationRules converts tagparser rules to Schema keywords using reused schemagen logic
func (g *structTagGenerator) applyValidationRules(rules []tagparser.TagRule, fieldType reflect.Type) []Keyword {
	var keywords []Keyword

	for _, rule := range rules {
		if rule.Name == "required" {
			continue // Required is handled at the object level
		}

		// Check built-in validators first
		if validator, exists := g.validatorMap[rule.Name]; exists {
			ruleKeywords := validator(fieldType, rule.Params)
			keywords = append(keywords, ruleKeywords...)
			continue
		}

		// Check custom validators
		if customValidator, exists := globalValidatorRegistry.GetCustomValidator(rule.Name); exists {
			ruleKeywords := customValidator(fieldType, rule.Params)
			keywords = append(keywords, ruleKeywords...)
			continue
		}

		// Check options-specific custom validators
		if g.options.CustomValidators != nil {
			if customFunc, exists := g.options.CustomValidators[rule.Name]; exists {
				if validatorFunc, ok := customFunc.(CustomValidatorFunc); ok {
					ruleKeywords := validatorFunc(fieldType, rule.Params)
					keywords = append(keywords, ruleKeywords...)
				} else if validatorFuncLegacy, ok := customFunc.(func(reflect.Type, []string) []Keyword); ok {
					// Support legacy validator function signature
					ruleKeywords := validatorFuncLegacy(fieldType, rule.Params)
					keywords = append(keywords, ruleKeywords...)
				}
			}
		}
	}

	return keywords
}

// getRefName generates a reference name for a struct type
func (g *structTagGenerator) getRefName(structType reflect.Type) string {
	if structType.Name() != "" {
		return structType.Name()
	}
	return fmt.Sprintf("Type%p", structType) // fallback for anonymous structs
}

// createRefSchema creates a $ref schema for circular references
func (g *structTagGenerator) createRefSchema(structType reflect.Type) *Schema {
	refName := g.getRefName(structType)

	// Store a placeholder in definitions if not already there
	if _, exists := g.definitions[refName]; !exists {
		g.definitions[refName] = &Schema{Type: SchemaType{"object"}}
	}

	// Store the ref name for this type
	g.generatedRefs[structType] = refName

	// Return a $ref schema
	return Ref(fmt.Sprintf("#/$defs/%s", refName))
}

// cloneSchemaWithKeywords creates a new schema by applying keywords to an existing schema
func (g *structTagGenerator) cloneSchemaWithKeywords(baseSchema *Schema, keywords []Keyword) *Schema {
	// Start with a copy of the base schema
	newSchema := &Schema{}
	*newSchema = *baseSchema // Copy all fields

	// Apply the additional keywords to the cloned schema
	for _, keyword := range keywords {
		keyword(newSchema)
	}

	return newSchema
}

// createCleanDefinition creates a clean copy of a schema for $defs to avoid circular references
func (g *structTagGenerator) createCleanDefinition(schema *Schema) *Schema {
	cleanSchema := &Schema{}
	*cleanSchema = *schema // Copy all fields

	// Ensure no circular references by not copying $defs in the definition
	cleanSchema.Defs = nil

	return cleanSchema
}

// createSchemaFromParam creates a Schema from a parameter string, handling primitive types and custom types
// This is a standalone function that doesn't depend on generator instance for use in validator mappings
func createSchemaFromParam(param string) *Schema {
	// Handle primitive types
	primitiveTypes := map[string]func() *Schema{
		"string":  func() *Schema { return String() },
		"integer": func() *Schema { return Integer() },
		"number":  func() *Schema { return Number() },
		"boolean": func() *Schema { return Boolean() },
		"null":    func() *Schema { return Null() },
		"object":  func() *Schema { return Object() },
		"array":   func() *Schema { return Array() },
	}

	if constructor, exists := primitiveTypes[param]; exists {
		return constructor()
	}

	// Handle boolean values
	if param == "true" || param == "false" {
		if param == "true" {
			return &Schema{Type: SchemaType{"boolean"}, Const: &ConstValue{Value: true, IsSet: true}}
		}
		return &Schema{Type: SchemaType{"boolean"}, Const: &ConstValue{Value: false, IsSet: true}}
	}

	// Handle numeric values
	if num, err := strconv.ParseFloat(param, 64); err == nil {
		if num == float64(int64(num)) {
			return &Schema{Type: SchemaType{"integer"}, Const: &ConstValue{Value: int64(num), IsSet: true}}
		}
		return &Schema{Type: SchemaType{"number"}, Const: &ConstValue{Value: num, IsSet: true}}
	}

	// Check if it's a custom struct type
	if isCustomStructType(param) {
		// Create a basic object schema for custom types.
		// Full schema generation for referenced types is handled separately via the Compiler.
		return &Schema{Type: SchemaType{"object"}, Description: &param}
	}

	// Default: treat as string constant
	return &Schema{Type: SchemaType{"string"}, Const: &ConstValue{Value: param, IsSet: true}}
}

// isCustomStructType checks if a parameter string represents a custom struct type
func isCustomStructType(typeName string) bool {
	// Check if it's not a built-in type
	builtinTypes := map[string]bool{
		"string": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true, "bool": true, "any": true,
		"integer": true, "number": true, "boolean": true, "null": true, "object": true, "array": true,
		"true": true, "false": true,
	}

	// Check if it starts with a package name (contains a dot)
	if strings.Contains(typeName, ".") {
		// External package type (like time.Time)
		return !builtinTypes[typeName]
	}

	// Local type - assume it's a custom struct if it's capitalized and not builtin
	if len(typeName) > 0 && typeName[0] >= 'A' && typeName[0] <= 'Z' {
		return !builtinTypes[typeName]
	}

	return false
}

// createRuntimeTypeMapping creates the mapping from Go types to Schema constructors (reused from schemagen)
func createRuntimeTypeMapping() map[string]func(...Keyword) *Schema {
	return map[string]func(...Keyword) *Schema{
		"string":    String,
		"int":       Integer,
		"int8":      Integer,
		"int16":     Integer,
		"int32":     Integer,
		"int64":     Integer,
		"uint":      Integer,
		"uint8":     Integer,
		"uint16":    Integer,
		"uint32":    Integer,
		"uint64":    Integer,
		"float32":   Number,
		"float64":   Number,
		"bool":      Boolean,
		"time.Time": func(keywords ...Keyword) *Schema { return String(append(keywords, Format("date-time"))...) },
	}
}

// createRuntimeValidatorMapping creates the mapping from validator names to runtime functions (reused from schemagen)
func createRuntimeValidatorMapping() map[string]validatorFunc {
	return map[string]validatorFunc{
		// String validators
		"minLength": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			if length, err := strconv.Atoi(params[0]); err == nil {
				return []Keyword{MinLen(length)}
			}
			return nil
		},
		"maxLength": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			if length, err := strconv.Atoi(params[0]); err == nil {
				return []Keyword{MaxLen(length)}
			}
			return nil
		},
		"pattern": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			return []Keyword{Pattern(params[0])}
		},
		"format": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			return []Keyword{Format(params[0])}
		},

		// Numeric validators
		"minimum": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			if value, err := strconv.ParseFloat(params[0], 64); err == nil {
				return []Keyword{Min(value)}
			}
			return nil
		},
		"maximum": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			if value, err := strconv.ParseFloat(params[0], 64); err == nil {
				return []Keyword{Max(value)}
			}
			return nil
		},
		"exclusiveMinimum": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			if value, err := strconv.ParseFloat(params[0], 64); err == nil {
				return []Keyword{ExclusiveMin(value)}
			}
			return nil
		},
		"exclusiveMaximum": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			if value, err := strconv.ParseFloat(params[0], 64); err == nil {
				return []Keyword{ExclusiveMax(value)}
			}
			return nil
		},
		"multipleOf": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			if value, err := strconv.ParseFloat(params[0], 64); err == nil {
				return []Keyword{MultipleOf(value)}
			}
			return nil
		},

		// Array validators
		"minItems": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			if count, err := strconv.Atoi(params[0]); err == nil {
				return []Keyword{MinItems(count)}
			}
			return nil
		},
		"maxItems": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			if count, err := strconv.Atoi(params[0]); err == nil {
				return []Keyword{MaxItems(count)}
			}
			return nil
		},
		"uniqueItems": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 || params[0] == "true" {
				return []Keyword{UniqueItems(true)}
			}
			if unique, err := strconv.ParseBool(params[0]); err == nil {
				return []Keyword{UniqueItems(unique)}
			}
			return nil
		},

		// Object validators
		"additionalProperties": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			if allowed, err := strconv.ParseBool(params[0]); err == nil {
				return []Keyword{AdditionalProps(allowed)}
			}
			return nil
		},
		"minProperties": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			if count, err := strconv.Atoi(params[0]); err == nil {
				return []Keyword{MinProps(count)}
			}
			return nil
		},
		"maxProperties": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			if count, err := strconv.Atoi(params[0]); err == nil {
				return []Keyword{MaxProps(count)}
			}
			return nil
		},

		// Enum and const validators
		"enum": func(fieldType reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			values := make([]any, len(params))
			for i, param := range params {
				// Convert based on field type
				//exhaustive:ignore - we only handle types that need conversion for enum values
				switch fieldType.Kind() {
				case reflect.String:
					values[i] = param
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					if intVal, err := strconv.Atoi(param); err == nil {
						values[i] = intVal
					} else {
						values[i] = param
					}
				case reflect.Float32, reflect.Float64:
					if floatVal, err := strconv.ParseFloat(param, 64); err == nil {
						values[i] = floatVal
					} else {
						values[i] = param
					}
				case reflect.Bool:
					if boolVal, err := strconv.ParseBool(param); err == nil {
						values[i] = boolVal
					} else {
						values[i] = param
					}
				default:
					values[i] = param
				}
			}
			return []Keyword{func(schema *Schema) {
				schema.Enum = values
			}}
		},

		"const": func(fieldType reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			value := params[0]

			// Convert value based on field type
			var constValue any = value
			//exhaustive:ignore - we only handle types that need conversion for const values
			switch fieldType.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if intVal, err := strconv.Atoi(value); err == nil {
					constValue = intVal
				}
			case reflect.Float32, reflect.Float64:
				if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
					constValue = floatVal
				}
			case reflect.Bool:
				if boolVal, err := strconv.ParseBool(value); err == nil {
					constValue = boolVal
				}
			}

			return []Keyword{func(schema *Schema) {
				schema.Const = &ConstValue{Value: constValue, IsSet: true}
			}}
		},

		// Metadata validators
		"title": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			return []Keyword{Title(params[0])}
		},
		"description": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			return []Keyword{Description(params[0])}
		},
		"default": func(fieldType reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}

			// Convert default value based on field type
			value := params[0]

			// Unwrap pointer type to get the underlying type
			actualType := fieldType
			if fieldType.Kind() == reflect.Ptr {
				actualType = fieldType.Elem()
			}

			//exhaustive:ignore - we only handle types that need conversion for default values
			switch actualType.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if intVal, err := strconv.Atoi(value); err == nil {
					return []Keyword{Default(intVal)}
				}
			case reflect.Float32, reflect.Float64:
				if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
					return []Keyword{Default(floatVal)}
				}
			case reflect.Bool:
				if boolVal, err := strconv.ParseBool(value); err == nil {
					return []Keyword{Default(boolVal)}
				}
			}
			return []Keyword{Default(value)}
		},
		"examples": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			examples := make([]any, len(params))
			for i, param := range params {
				examples[i] = param // Could add type conversion here
			}
			return []Keyword{Examples(examples...)}
		},
		"deprecated": func(_ reflect.Type, params []string) []Keyword {
			deprecated := true
			if len(params) > 0 {
				if val, err := strconv.ParseBool(params[0]); err == nil {
					deprecated = val
				}
			}
			return []Keyword{Deprecated(deprecated)}
		},
		"readOnly": func(_ reflect.Type, params []string) []Keyword {
			readOnly := true
			if len(params) > 0 {
				if val, err := strconv.ParseBool(params[0]); err == nil {
					readOnly = val
				}
			}
			return []Keyword{ReadOnly(readOnly)}
		},
		"writeOnly": func(_ reflect.Type, params []string) []Keyword {
			writeOnly := true
			if len(params) > 0 {
				if val, err := strconv.ParseBool(params[0]); err == nil {
					writeOnly = val
				}
			}
			return []Keyword{WriteOnly(writeOnly)}
		},

		// Content validators
		"contentEncoding": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			return []Keyword{ContentEncoding(params[0])}
		},
		"contentMediaType": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			return []Keyword{ContentMediaType(params[0])}
		},

		// Logical combination validators
		"allOf": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			var schemas []*Schema
			for _, schemaType := range params {
				schema := createSchemaFromParam(schemaType)
				if schema != nil {
					schemas = append(schemas, schema)
				}
			}
			if len(schemas) == 0 {
				return nil
			}
			return []Keyword{func(s *Schema) {
				s.AllOf = schemas
			}}
		},
		"anyOf": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			var schemas []*Schema
			for _, schemaType := range params {
				schema := createSchemaFromParam(schemaType)
				if schema != nil {
					schemas = append(schemas, schema)
				}
			}
			if len(schemas) == 0 {
				return nil
			}
			return []Keyword{func(s *Schema) {
				s.AnyOf = schemas
			}}
		},
		"oneOf": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			var schemas []*Schema
			for _, schemaType := range params {
				schema := createSchemaFromParam(schemaType)
				if schema != nil {
					schemas = append(schemas, schema)
				}
			}
			if len(schemas) == 0 {
				return nil
			}
			return []Keyword{func(s *Schema) {
				s.OneOf = schemas
			}}
		},
		"not": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			schema := createSchemaFromParam(params[0])
			if schema == nil {
				return nil
			}
			return []Keyword{func(s *Schema) {
				s.Not = schema
			}}
		},

		// Array advanced validators
		"contains": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			schema := createSchemaFromParam(params[0])
			if schema == nil {
				return nil
			}
			return []Keyword{func(s *Schema) {
				s.Contains = schema
			}}
		},
		"minContains": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			if count, err := strconv.Atoi(params[0]); err == nil {
				return []Keyword{func(s *Schema) {
					countFloat := float64(count)
					s.MinContains = &countFloat
				}}
			}
			return nil
		},
		"maxContains": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			if count, err := strconv.Atoi(params[0]); err == nil {
				return []Keyword{func(s *Schema) {
					countFloat := float64(count)
					s.MaxContains = &countFloat
				}}
			}
			return nil
		},

		// Array advanced validators - prefixItems
		"prefixItems": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			var schemas []*Schema
			for _, schemaType := range params {
				schema := createSchemaFromParam(schemaType)
				if schema != nil {
					schemas = append(schemas, schema)
				}
			}
			if len(schemas) == 0 {
				return nil
			}
			return []Keyword{func(s *Schema) {
				s.PrefixItems = schemas
			}}
		},

		// Object advanced validators
		"patternProperties": func(_ reflect.Type, params []string) []Keyword {
			if len(params) < 2 {
				return nil
			}
			pattern := params[0]
			schemaType := params[1]

			schema := createSchemaFromParam(schemaType)
			if schema == nil {
				return nil
			}

			return []Keyword{func(s *Schema) {
				if s.PatternProperties == nil {
					s.PatternProperties = &SchemaMap{}
				}
				(*s.PatternProperties)[pattern] = schema
			}}
		},
		"propertyNames": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			schema := createSchemaFromParam(params[0])
			if schema == nil {
				return nil
			}
			return []Keyword{func(s *Schema) {
				s.PropertyNames = schema
			}}
		},
		"dependentRequired": func(_ reflect.Type, params []string) []Keyword {
			if len(params) < 2 {
				return nil
			}
			property := params[0]
			dependentFields := params[1:]

			return []Keyword{func(s *Schema) {
				if s.DependentRequired == nil {
					s.DependentRequired = make(map[string][]string)
				}
				s.DependentRequired[property] = dependentFields
			}}
		},
		"dependentSchemas": func(_ reflect.Type, params []string) []Keyword {
			if len(params) < 2 {
				return nil
			}
			property := params[0]
			schemaType := params[1]

			schema := createSchemaFromParam(schemaType)
			if schema == nil {
				return nil
			}

			return []Keyword{func(s *Schema) {
				if s.DependentSchemas == nil {
					s.DependentSchemas = make(map[string]*Schema)
				}
				s.DependentSchemas[property] = schema
			}}
		},

		// Conditional logic validators
		"if": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			schema := createSchemaFromParam(params[0])
			if schema == nil {
				return nil
			}
			return []Keyword{func(s *Schema) {
				s.If = schema
			}}
		},
		"then": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			schema := createSchemaFromParam(params[0])
			if schema == nil {
				return nil
			}
			return []Keyword{func(s *Schema) {
				s.Then = schema
			}}
		},
		"else": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			schema := createSchemaFromParam(params[0])
			if schema == nil {
				return nil
			}
			return []Keyword{func(s *Schema) {
				s.Else = schema
			}}
		},

		// Advanced array validators
		"unevaluatedItems": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return []Keyword{func(s *Schema) {
					s.UnevaluatedItems = &Schema{Not: &Schema{}}
				}}
			}

			switch params[0] {
			case "false":
				return []Keyword{func(s *Schema) {
					s.UnevaluatedItems = &Schema{Not: &Schema{}}
				}}
			case "true":
				return []Keyword{func(s *Schema) {
					s.UnevaluatedItems = &Schema{}
				}}
			default:
				schema := createSchemaFromParam(params[0])
				if schema == nil {
					return nil
				}
				return []Keyword{func(s *Schema) {
					s.UnevaluatedItems = schema
				}}
			}
		},

		// Advanced object validators
		"unevaluatedProperties": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return []Keyword{func(s *Schema) {
					s.UnevaluatedProperties = &Schema{Not: &Schema{}}
				}}
			}

			switch params[0] {
			case "false":
				return []Keyword{func(s *Schema) {
					s.UnevaluatedProperties = &Schema{Not: &Schema{}}
				}}
			case "true":
				return []Keyword{func(s *Schema) {
					s.UnevaluatedProperties = &Schema{}
				}}
			default:
				schema := createSchemaFromParam(params[0])
				if schema == nil {
					return nil
				}
				return []Keyword{func(s *Schema) {
					s.UnevaluatedProperties = schema
				}}
			}
		},

		// Content validation
		"contentSchema": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			schema := createSchemaFromParam(params[0])
			if schema == nil {
				return nil
			}
			return []Keyword{func(s *Schema) {
				s.ContentSchema = schema
			}}
		},

		// Manual reference support - for advanced use cases
		"ref": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			refURI := params[0]
			return []Keyword{func(s *Schema) {
				s.Ref = refURI
			}}
		},
		"defs": func(_ reflect.Type, params []string) []Keyword {
			// This is more complex as it would need access to multiple schemas
			// For now, we'll implement a basic version that just sets a marker
			if len(params) == 0 {
				return nil
			}
			// This would typically be handled at the schema level, not field level
			// For now, return empty to indicate it's recognized but not implemented
			return nil
		},
		"anchor": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			anchor := params[0]
			return []Keyword{func(s *Schema) {
				s.Anchor = anchor
			}}
		},
		"dynamicRef": func(_ reflect.Type, params []string) []Keyword {
			if len(params) == 0 {
				return nil
			}
			dynamicRef := params[0]
			return []Keyword{func(s *Schema) {
				s.DynamicRef = dynamicRef
			}}
		},
	}
}

// separateArrayAndItemKeywords separates validation keywords into array-level and item-level constraints
func (g *structTagGenerator) separateArrayAndItemKeywords(keywords []Keyword, fieldType reflect.Type) ([]Keyword, []Keyword) {
	var arrayKeywords []Keyword
	var itemKeywords []Keyword

	// Get the element type to determine if item-level constraints make sense
	elemType := fieldType.Elem()
	for elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	// For each keyword, determine if it should be applied to array or items
	for _, keyword := range keywords {
		keywordType := getKeywordType(keyword)

		switch {
		case isObjectConstraint(keywordType) && (elemType.Kind() == reflect.Struct || elemType.Kind() == reflect.Map):
			// Object-specific constraints should go on items if elements are objects
			itemKeywords = append(itemKeywords, keyword)
		case isArrayConstraint(keywordType):
			// Array-specific constraints go on the array
			arrayKeywords = append(arrayKeywords, keyword)
		default:
			// Default: apply to array level (backward compatibility)
			arrayKeywords = append(arrayKeywords, keyword)
		}
	}

	return arrayKeywords, itemKeywords
}

// cloneSchemaAndUpdateItems creates a copy of an array schema with updated items
func (g *structTagGenerator) cloneSchemaAndUpdateItems(arraySchema *Schema, newItemSchema *Schema) *Schema {
	newSchema := &Schema{}
	*newSchema = *arraySchema // Copy all fields
	newSchema.Items = newItemSchema
	return newSchema
}

// getKeywordType inspects a keyword function to determine its type
func getKeywordType(keyword Keyword) string {
	// This is a simplified implementation - in practice, we would need to inspect
	// the keyword function to determine what it does. For now, we'll use a test schema
	// to see what fields the keyword modifies.
	testSchema := &Schema{}
	keyword(testSchema)

	// Check which fields were modified
	if testSchema.MaxProperties != nil || testSchema.MinProperties != nil {
		return "object"
	}
	if testSchema.MaxItems != nil || testSchema.MinItems != nil || testSchema.UniqueItems != nil {
		return "array"
	}
	if testSchema.MaxLength != nil || testSchema.MinLength != nil || testSchema.Pattern != nil {
		return "string"
	}
	if testSchema.Maximum != nil || testSchema.Minimum != nil || testSchema.MultipleOf != nil {
		return "number"
	}
	if testSchema.Enum != nil || testSchema.Const != nil {
		return "value"
	}
	if testSchema.Description != nil || testSchema.Title != nil {
		return "metadata"
	}

	return "unknown"
}

// isObjectConstraint checks if a constraint type applies to objects
func isObjectConstraint(constraintType string) bool {
	return constraintType == "object"
}

// isArrayConstraint checks if a constraint type applies to arrays
func isArrayConstraint(constraintType string) bool {
	return constraintType == "array"
}

// applySchemaProperties applies schema-level properties from StructTagOptions to the schema
func applySchemaProperties(schema *Schema, options *StructTagOptions) {
	// Apply properties from SchemaProperties map - only set when explicitly provided
	if options.SchemaProperties != nil {
		for key, value := range options.SchemaProperties {
			switch key {
			case "additionalProperties":
				if boolVal, ok := value.(bool); ok {
					schema.AdditionalProperties = &Schema{Boolean: &boolVal}
				}
			case "title":
				if strVal, ok := value.(string); ok {
					schema.Title = &strVal
				}
			case "description":
				if strVal, ok := value.(string); ok {
					schema.Description = &strVal
				}
			case "minProperties":
				if intVal, ok := value.(int); ok {
					floatVal := float64(intVal)
					schema.MinProperties = &floatVal
				}
			case "maxProperties":
				if intVal, ok := value.(int); ok {
					floatVal := float64(intVal)
					schema.MaxProperties = &floatVal
				}
			case "default":
				schema.Default = value
			case "examples":
				if examples, ok := value.([]any); ok {
					schema.Examples = examples
				}
			case "deprecated":
				if deprecated, ok := value.(bool); ok {
					schema.Deprecated = &deprecated
				}
			case "readOnly":
				if readOnly, ok := value.(bool); ok {
					schema.ReadOnly = &readOnly
				}
			case "writeOnly":
				if writeOnly, ok := value.(bool); ok {
					schema.WriteOnly = &writeOnly
				}
			}
		}
	}
}
