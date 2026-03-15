package jsonschema

import (
	"bytes"
	"errors"
	"maps"
	"regexp"
	"slices"
	"strconv"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/kaptinlin/jsonpointer"
)

// knownSchemaFields contains all known JSON Schema keywords.
// Used to filter out known fields when collecting extra/extension fields.
var knownSchemaFields = map[string]struct{}{
	// Core keywords
	"$id":            {},
	"$schema":        {},
	"$ref":           {},
	"$dynamicRef":    {},
	"$anchor":        {},
	"$dynamicAnchor": {},
	"$defs":          {},
	"definitions":    {}, // Draft-7 compatibility
	"$comment":       {},

	// Applicator keywords
	"allOf":                 {},
	"anyOf":                 {},
	"oneOf":                 {},
	"not":                   {},
	"if":                    {},
	"then":                  {},
	"else":                  {},
	"dependentSchemas":      {},
	"prefixItems":           {},
	"items":                 {},
	"contains":              {},
	"properties":            {},
	"patternProperties":     {},
	"additionalProperties":  {},
	"propertyNames":         {},
	"unevaluatedItems":      {},
	"unevaluatedProperties": {},

	// Validation keywords
	"type":              {},
	"enum":              {},
	"const":             {},
	"multipleOf":        {},
	"maximum":           {},
	"exclusiveMaximum":  {},
	"minimum":           {},
	"exclusiveMinimum":  {},
	"maxLength":         {},
	"minLength":         {},
	"pattern":           {},
	"maxItems":          {},
	"minItems":          {},
	"uniqueItems":       {},
	"maxContains":       {},
	"minContains":       {},
	"maxProperties":     {},
	"minProperties":     {},
	"required":          {},
	"dependentRequired": {},

	// Format keyword
	"format": {},

	// Content keywords
	"contentEncoding":  {},
	"contentMediaType": {},
	"contentSchema":    {},

	// Meta-data keywords
	"title":       {},
	"description": {},
	"default":     {},
	"deprecated":  {},
	"readOnly":    {},
	"writeOnly":   {},
	"examples":    {},
}

// Schema represents a JSON Schema as per the 2020-12 draft, containing all
// necessary metadata and validation properties defined by the specification.
type Schema struct {
	compiledPatterns      map[string]*regexp.Regexp // Cached compiled regular expressions for pattern properties.
	compiler              *Compiler                 // Reference to the associated Compiler instance.
	parent                *Schema                   // Parent schema for hierarchical resolution.
	uri                   string                    // Internal schema identifier resolved during compilation.
	baseURI               string                    // Base URI for resolving relative references within the schema.
	anchors               map[string]*Schema        // Anchors for quick lookup of internal schema references.
	dynamicAnchors        map[string]*Schema        // Dynamic anchors for more flexible schema references.
	schemas               map[string]*Schema        // Cache of compiled schemas.
	compiledStringPattern *regexp.Regexp            // Cached compiled regular expressions for string patterns.

	ID     string  `json:"$id,omitempty"`     // Public identifier for the schema.
	Schema string  `json:"$schema,omitempty"` // URI indicating the specification the schema conforms to.
	Format *string `json:"format,omitempty"`  // Format hint for string data, e.g., "email" or "date-time".

	// Schema reference keywords, see https://json-schema.org/draft/2020-12/json-schema-core#ref
	Ref                string             `json:"$ref,omitempty"`           // Reference to another schema.
	DynamicRef         string             `json:"$dynamicRef,omitempty"`    // Reference to another schema that can be dynamically resolved.
	Anchor             string             `json:"$anchor,omitempty"`        // Anchor for resolving relative JSON Pointers.
	DynamicAnchor      string             `json:"$dynamicAnchor,omitempty"` // Anchor for dynamic resolution
	Defs               map[string]*Schema `json:"$defs,omitempty"`          // An object containing schema definitions.
	ResolvedRef        *Schema            `json:"-"`                        // Resolved schema for $ref
	ResolvedDynamicRef *Schema            `json:"-"`                        // Resolved schema for $dynamicRef

	// Boolean JSON Schemas, see https://json-schema.org/draft/2020-12/json-schema-core#name-boolean-json-schemas
	Boolean *bool `json:"-"` // Boolean schema, used for quick validation.

	// Applying subschemas with logical keywords, see https://json-schema.org/draft/2020-12/json-schema-core#name-keywords-for-applying-subsch
	AllOf []*Schema `json:"allOf,omitempty"` // Array of schemas for validating the instance against all of them.
	AnyOf []*Schema `json:"anyOf,omitempty"` // Array of schemas for validating the instance against any of them.
	OneOf []*Schema `json:"oneOf,omitempty"` // Array of schemas for validating the instance against exactly one of them.
	Not   *Schema   `json:"not,omitempty"`   // Schema for validating the instance against the negation of it.

	// Applying subschemas conditionally, see https://json-schema.org/draft/2020-12/json-schema-core#name-keywords-for-applying-subsche
	If               *Schema            `json:"if,omitempty"`               // Schema to be evaluated as a condition
	Then             *Schema            `json:"then,omitempty"`             // Schema to be evaluated if 'if' is successful
	Else             *Schema            `json:"else,omitempty"`             // Schema to be evaluated if 'if' is not successful
	DependentSchemas map[string]*Schema `json:"dependentSchemas,omitempty"` // Dependent schemas based on property presence

	// Applying subschemas to array keywords, see https://json-schema.org/draft/2020-12/json-schema-core#name-keywords-for-applying-subschem
	PrefixItems []*Schema `json:"prefixItems,omitempty"` // Array of schemas for validating the array items' prefix.
	Items       *Schema   `json:"items,omitempty"`       // Schema for items in an array.
	Contains    *Schema   `json:"contains,omitempty"`    // Schema for validating items in the array.

	// Applying subschemas to objects keywords, see https://json-schema.org/draft/2020-12/json-schema-core#name-keywords-for-applying-subschemas
	Properties           *SchemaMap `json:"properties,omitempty"`           // Definitions of properties for object types.
	PatternProperties    *SchemaMap `json:"patternProperties,omitempty"`    // Definitions of properties for object types matched by specific patterns.
	AdditionalProperties *Schema    `json:"additionalProperties,omitempty"` // Can be a boolean or a schema, controls additional properties handling.
	PropertyNames        *Schema    `json:"propertyNames,omitempty"`        // Can be a boolean or a schema, controls property names validation.

	// Any validation keywords, see https://json-schema.org/draft/2020-12/json-schema-validation#section-6.1
	Type  SchemaType  `json:"type,omitempty"`  // Can be a single type or an array of types.
	Enum  []any       `json:"enum,omitempty"`  // Enumerated values for the property.
	Const *ConstValue `json:"const,omitempty"` // Constant value for the property.

	// Numeric validation keywords, see https://json-schema.org/draft/2020-12/json-schema-validation#section-6.2
	MultipleOf       *Rat `json:"multipleOf,omitempty"`       // Number must be a multiple of this value, strictly greater than 0.
	Maximum          *Rat `json:"maximum,omitempty"`          // Maximum value of the number.
	ExclusiveMaximum *Rat `json:"exclusiveMaximum,omitempty"` // Number must be less than this value.
	Minimum          *Rat `json:"minimum,omitempty"`          // Minimum value of the number.
	ExclusiveMinimum *Rat `json:"exclusiveMinimum,omitempty"` // Number must be greater than this value.

	// String validation keywords, see https://json-schema.org/draft/2020-12/json-schema-validation#section-6.3
	MaxLength *float64 `json:"maxLength,omitempty"` // Maximum length of a string.
	MinLength *float64 `json:"minLength,omitempty"` // Minimum length of a string.
	Pattern   *string  `json:"pattern,omitempty"`   // Regular expression pattern to match the string against.

	// Array validation keywords, see https://json-schema.org/draft/2020-12/json-schema-validation#section-6.4
	MaxItems    *float64 `json:"maxItems,omitempty"`    // Maximum number of items in an array.
	MinItems    *float64 `json:"minItems,omitempty"`    // Minimum number of items in an array.
	UniqueItems *bool    `json:"uniqueItems,omitempty"` // Whether the items in the array must be unique.
	MaxContains *float64 `json:"maxContains,omitempty"` // Maximum number of items in the array that can match the contains schema.
	MinContains *float64 `json:"minContains,omitempty"` // Minimum number of items in the array that must match the contains schema.

	// https://json-schema.org/draft/2020-12/json-schema-core#name-unevaluateditems
	UnevaluatedItems *Schema `json:"unevaluatedItems,omitempty"` // Schema for unevaluated items in an array.

	// Object validation keywords, see https://json-schema.org/draft/2020-12/json-schema-validation#section-6.5
	MaxProperties     *float64            `json:"maxProperties,omitempty"`     // Maximum number of properties in an object.
	MinProperties     *float64            `json:"minProperties,omitempty"`     // Minimum number of properties in an object.
	Required          []string            `json:"required,omitempty"`          // List of required property names for object types.
	DependentRequired map[string][]string `json:"dependentRequired,omitempty"` // Properties required when another property is present.

	// https://json-schema.org/draft/2020-12/json-schema-core#name-unevaluatedproperties
	UnevaluatedProperties *Schema `json:"unevaluatedProperties,omitempty"` // Schema for unevaluated properties in an object.

	// Content validation keywords, see https://json-schema.org/draft/2020-12/json-schema-validation#name-a-vocabulary-for-the-conten
	ContentEncoding  *string `json:"contentEncoding,omitempty"`  // Encoding format of the content.
	ContentMediaType *string `json:"contentMediaType,omitempty"` // Media type of the content.
	ContentSchema    *Schema `json:"contentSchema,omitempty"`    // Schema for validating the content.

	// Meta-data for schema and instance description, see https://json-schema.org/draft/2020-12/json-schema-validation#name-a-vocabulary-for-basic-meta
	Title       *string `json:"title,omitempty"`       // A short summary of the schema.
	Description *string `json:"description,omitempty"` // A detailed description of the purpose of the schema.
	Default     any     `json:"default,omitempty"`     // Default value of the instance.
	Deprecated  *bool   `json:"deprecated,omitempty"`  // Indicates that the schema is deprecated.
	ReadOnly    *bool   `json:"readOnly,omitempty"`    // Indicates that the property is read-only.
	WriteOnly   *bool   `json:"writeOnly,omitempty"`   // Indicates that the property is write-only.
	Examples    []any   `json:"examples,omitempty"`    // Examples of the instance data that validates against this schema.

	// Extra keywords not in specification
	Extra map[string]any `json:"-"`
}

// newSchema parses JSON schema data and returns a Schema object.
func newSchema(jsonSchema []byte) (*Schema, error) {
	schema := &Schema{}

	// Parse schema
	if err := json.Unmarshal(jsonSchema, schema); err != nil {
		return nil, err
	}

	return schema, nil
}

// initializeSchema sets up the schema structure, resolves URIs, and initializes nested schemas.
// It populates schema properties from the compiler settings and the parent schema context.
func (s *Schema) initializeSchema(compiler *Compiler, parent *Schema) {
	s.initializeSchemaCore(compiler, parent, true)
}

// initializeSchemaWithoutReferences sets up the schema structure without resolving references.
// This is used by CompileBatch to defer reference resolution until all schemas are compiled.
func (s *Schema) initializeSchemaWithoutReferences(compiler *Compiler, parent *Schema) {
	s.initializeSchemaCore(compiler, parent, false)
}

// initializeSchemaCore contains the shared initialization logic.
// When resolveRefs is true, references are resolved immediately after nested schema initialization.
// When resolveRefs is false, reference resolution is deferred (used by CompileBatch).
func (s *Schema) initializeSchemaCore(compiler *Compiler, parent *Schema, resolveRefs bool) {
	if compiler != nil {
		s.compiler = compiler
	}
	s.parent = parent

	effectiveCompiler := s.Compiler()

	// Resolve base URI
	s.resolveBaseURI(effectiveCompiler)

	// Set anchors
	if s.Anchor != "" {
		s.setAnchor(s.Anchor)
	}
	if s.DynamicAnchor != "" {
		s.setDynamicAnchor(s.DynamicAnchor)
	}

	// Register schema in root
	if s.uri != "" && isValidURI(s.uri) {
		root := s.rootSchema()
		root.setSchema(s.uri, s)
	}

	// Initialize nested schemas
	initializeNestedSchemasCore(s, compiler, resolveRefs)
	if resolveRefs {
		s.resolveReferences()
	}

	// Handle PreserveExtra option
	if effectiveCompiler != nil && !effectiveCompiler.PreserveExtra {
		s.Extra = nil
	}
}

// resolveBaseURI resolves the base URI for the schema
func (s *Schema) resolveBaseURI(compiler *Compiler) {
	parentBaseURI := s.parentBaseURI()
	if parentBaseURI == "" {
		parentBaseURI = compiler.DefaultBaseURI
	}

	if s.ID != "" {
		if isValidURI(s.ID) {
			s.uri = s.ID
			s.baseURI = getBaseURI(s.ID)
		} else {
			resolvedURL := resolveRelativeURI(parentBaseURI, s.ID)
			s.uri = resolvedURL
			s.baseURI = getBaseURI(resolvedURL)
		}
	} else {
		s.baseURI = parentBaseURI
	}

	if s.baseURI == "" && s.uri != "" && isValidURI(s.uri) {
		s.baseURI = getBaseURI(s.uri)
	}
}

// initializeNestedSchemasCore initializes all nested or related schemas as defined in the structure.
// When resolveRefs is true, schemas are initialized with full reference resolution.
// When resolveRefs is false, reference resolution is deferred (used by CompileBatch).
func initializeNestedSchemasCore(s *Schema, compiler *Compiler, resolveRefs bool) {
	initChild := func(child *Schema) {
		child.initializeSchemaCore(compiler, s, resolveRefs)
	}

	if s.Defs != nil {
		for _, def := range s.Defs {
			initChild(def)
		}
	}
	// Initialize logical schema groupings
	for _, schemas := range [][]*Schema{s.AllOf, s.AnyOf, s.OneOf} {
		for _, schema := range schemas {
			if schema != nil {
				initChild(schema)
			}
		}
	}

	// Initialize conditional schemas
	for _, schema := range []*Schema{s.Not, s.If, s.Then, s.Else} {
		if schema != nil {
			initChild(schema)
		}
	}
	if s.DependentSchemas != nil {
		for _, depSchema := range s.DependentSchemas {
			initChild(depSchema)
		}
	}

	// Initialize array and object schemas
	if s.PrefixItems != nil {
		for _, item := range s.PrefixItems {
			initChild(item)
		}
	}
	for _, schema := range []*Schema{s.Items, s.Contains, s.AdditionalProperties} {
		if schema != nil {
			initChild(schema)
		}
	}
	if s.Properties != nil {
		for _, prop := range *s.Properties {
			initChild(prop)
		}
	}
	if s.PatternProperties != nil {
		for _, prop := range *s.PatternProperties {
			initChild(prop)
		}
	}
	for _, schema := range []*Schema{s.UnevaluatedProperties, s.UnevaluatedItems, s.ContentSchema, s.PropertyNames} {
		if schema != nil {
			initChild(schema)
		}
	}
}

// validateRegexSyntax validates that all regex patterns in the schema are valid Go RE2 syntax.
// It recursively checks pattern and patternProperties in the schema and all nested schemas.
func (s *Schema) validateRegexSyntax() error {
	if s == nil {
		return nil
	}

	visited := make(map[*Schema]bool)
	errs := s.collectRegexErrors(nil, visited)
	if len(errs) == 0 {
		return nil
	}

	combined := append([]error{ErrRegexValidation}, errs...)
	return errors.Join(combined...)
}

// collectRegexErrors recursively collects regex compilation errors from the schema tree.
// It uses a token slice to track the JSON Pointer path, avoiding string parsing overhead.
func (s *Schema) collectRegexErrors(pathTokens []string, visited map[*Schema]bool) []error {
	if s == nil || visited[s] {
		return nil
	}
	visited[s] = true

	var errs []error

	// Validate pattern field
	if s.Pattern != nil {
		if err := compilePattern(*s.Pattern); err != nil {
			patternTokens := slices.Concat(pathTokens, []string{"pattern"})
			errs = append(errs, &RegexPatternError{
				Keyword:  "pattern",
				Location: "#" + jsonpointer.Format(patternTokens...),
				Pattern:  *s.Pattern,
				Err:      err,
			})
		}
	}

	// Validate patternProperties keys and recurse into values
	if s.PatternProperties != nil {
		for pattern, schema := range *s.PatternProperties {
			patternPropTokens := slices.Concat(pathTokens, []string{"patternProperties", pattern})
			if err := compilePattern(pattern); err != nil {
				errs = append(errs, &RegexPatternError{
					Keyword:  "patternProperties",
					Location: "#" + jsonpointer.Format(patternPropTokens...),
					Pattern:  pattern,
					Err:      err,
				})
				continue
			}
			errs = append(errs, schema.collectRegexErrors(patternPropTokens, visited)...)
		}
	}

	// Helper to recurse into a single schema
	addSchema := func(child *Schema, token string) {
		childTokens := slices.Concat(pathTokens, []string{token})
		errs = append(errs, child.collectRegexErrors(childTokens, visited)...)
	}

	// Helper to recurse into a map of schemas
	addSchemaMap := func(m map[string]*Schema, prefix string) {
		if len(m) == 0 {
			return
		}
		for key, schema := range m {
			mapTokens := slices.Concat(pathTokens, []string{prefix, key})
			errs = append(errs, schema.collectRegexErrors(mapTokens, visited)...)
		}
	}

	// Helper to recurse into a slice of schemas
	addSchemaSlice := func(children []*Schema, prefix string) {
		if len(children) == 0 {
			return
		}
		for i, child := range children {
			sliceTokens := slices.Concat(pathTokens, []string{prefix, strconv.Itoa(i)})
			errs = append(errs, child.collectRegexErrors(sliceTokens, visited)...)
		}
	}

	// Recurse into all nested schemas
	if s.Properties != nil {
		addSchemaMap(map[string]*Schema(*s.Properties), "properties")
	}
	if s.Defs != nil {
		addSchemaMap(s.Defs, "$defs")
	}
	if s.DependentSchemas != nil {
		addSchemaMap(s.DependentSchemas, "dependentSchemas")
	}

	addSchema(s.AdditionalProperties, "additionalProperties")
	addSchema(s.UnevaluatedProperties, "unevaluatedProperties")
	addSchema(s.UnevaluatedItems, "unevaluatedItems")
	addSchema(s.PropertyNames, "propertyNames")
	addSchema(s.ContentSchema, "contentSchema")
	addSchema(s.Items, "items")
	addSchema(s.Contains, "contains")
	addSchema(s.Not, "not")
	addSchema(s.If, "if")
	addSchema(s.Then, "then")
	addSchema(s.Else, "else")
	addSchema(s.ResolvedRef, "$ref")
	addSchema(s.ResolvedDynamicRef, "$dynamicRef")

	addSchemaSlice(s.PrefixItems, "prefixItems")
	addSchemaSlice(s.AllOf, "allOf")
	addSchemaSlice(s.AnyOf, "anyOf")
	addSchemaSlice(s.OneOf, "oneOf")

	return errs
}

// compilePattern validates that a regex pattern is valid Go RE2 syntax.
// Returns nil if the pattern is valid, or the regexp compilation error if invalid.
func compilePattern(pattern string) error {
	if pattern == "" {
		return nil
	}
	_, err := regexp.Compile(pattern)
	return err
}

// setAnchor creates or updates the anchor mapping for the current schema and propagates it to parent schemas.
func (s *Schema) setAnchor(anchor string) {
	if s.anchors == nil {
		s.anchors = make(map[string]*Schema)
	}
	s.anchors[anchor] = s

	root := s.rootSchema()
	if root.anchors == nil {
		root.anchors = make(map[string]*Schema)
	}

	// Only set anchor at root level if it's in the same scope as root
	if (s.ID == "" || s.ID == root.ID) && root.anchors[anchor] == nil {
		root.anchors[anchor] = s
	}
}

// setDynamicAnchor sets or updates a dynamic anchor for the current schema and propagates it to parents in the same scope.
func (s *Schema) setDynamicAnchor(anchor string) {
	if s.dynamicAnchors == nil {
		s.dynamicAnchors = make(map[string]*Schema)
	}
	if s.dynamicAnchors[anchor] == nil {
		s.dynamicAnchors[anchor] = s
	}

	scope := s.scopeSchema()
	if scope.dynamicAnchors == nil {
		scope.dynamicAnchors = make(map[string]*Schema)
	}

	if scope.dynamicAnchors[anchor] == nil {
		scope.dynamicAnchors[anchor] = s
	}
}

// setSchema adds a schema to the internal schema cache, using the provided URI as the key.
func (s *Schema) setSchema(uri string, schema *Schema) *Schema {
	if s.schemas == nil {
		s.schemas = make(map[string]*Schema)
	}

	s.schemas[uri] = schema
	return s
}

func (s *Schema) getSchema(ref string) (*Schema, error) {
	baseURI, anchor := splitRef(ref)

	if schema, exists := s.schemas[baseURI]; exists {
		if baseURI == ref {
			return schema, nil
		}
		return schema.resolveAnchor(anchor)
	}

	return nil, ErrReferenceResolution
}

// SchemaURI returns the resolved URI for the schema, or an empty string if no URI is defined.
func (s *Schema) SchemaURI() string {
	if s.uri != "" {
		return s.uri
	}
	root := s.rootSchema()
	if root.uri != "" {
		return root.uri
	}

	return ""
}

// SchemaLocation returns the schema location with the given anchor
func (s *Schema) SchemaLocation(anchor string) string {
	uri := s.SchemaURI()

	return uri + "#" + anchor
}

// rootSchema returns the highest-level parent schema, serving as the root in the schema tree.
func (s *Schema) rootSchema() *Schema {
	if s.parent != nil {
		return s.parent.rootSchema()
	}

	return s
}

func (s *Schema) scopeSchema() *Schema {
	if s.ID != "" {
		return s
	}
	if s.parent != nil {
		return s.parent.scopeSchema()
	}

	return s
}

// parentBaseURI returns the base URI from the nearest parent schema that has one defined,
// or an empty string if none of the parents up to the root define a base URI.
func (s *Schema) parentBaseURI() string {
	for p := s.parent; p != nil; p = p.parent {
		if p.baseURI != "" {
			return p.baseURI
		}
	}
	return ""
}

// MarshalJSON implements json.Marshaler
func (s *Schema) MarshalJSON() ([]byte, error) {
	if s.Boolean != nil {
		return json.Marshal(s.Boolean, json.Deterministic(true))
	}

	// Custom marshaling to handle the const field properly
	type Alias Schema
	alias := (*Alias)(s)

	// Marshal to a map to handle const field manually with deterministic ordering
	data, err := json.Marshal(alias, json.Deterministic(true))
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	// Handle the const field manually
	if s.Const != nil {
		result["const"] = s.Const.Value
	}

	maps.Copy(result, s.Extra)

	// Use deterministic marshaling to ensure consistent key ordering
	// Note: Required and DependentRequired arrays maintain their order from generation/parsing
	return json.Marshal(result, json.Deterministic(true))
}

// MarshalJSONTo implements json.MarshalerTo for JSON v2 with proper option support
func (s *Schema) MarshalJSONTo(enc *jsontext.Encoder, opts json.Options) error {
	// Ensure deterministic ordering is always enabled
	opts = json.JoinOptions(opts, json.Deterministic(true))

	if s.Boolean != nil {
		return json.MarshalEncode(enc, s.Boolean, opts)
	}

	// Use the existing MarshalJSON method which already handles the const field properly
	// and then ensure the result is marshaled with the provided options
	data, err := s.MarshalJSON()
	if err != nil {
		return err
	}

	// Parse and re-marshal with deterministic options
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	return json.MarshalEncode(enc, result, opts)
}

// UnmarshalJSON handles unmarshaling JSON data into the Schema type.
func (s *Schema) UnmarshalJSON(data []byte) error {
	// First try to parse as a boolean
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		s.Boolean = &b
		return nil
	}

	// Use a temporary struct to intercept "items" and "additionalItems"
	type Alias Schema
	aux := &struct {
		Items           jsontext.Value `json:"items,omitempty"`
		AdditionalItems *Schema        `json:"additionalItems,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Smart handling for "items" polymorphism (Draft 07 vs 2020-12)
	if len(aux.Items) > 0 {
		trimmed := bytes.TrimSpace(aux.Items)
		if len(trimmed) > 0 && trimmed[0] == '[' {
			// Case 1: items is an array (Draft 07 Tuple Validation)
			if err := json.Unmarshal(aux.Items, &s.PrefixItems); err != nil {
				return err
			}
			// In Draft 07, "additionalItems" validates the rest
			if aux.AdditionalItems != nil {
				s.Items = aux.AdditionalItems
			}
		} else {
			// Case 2: items is a schema object (Draft 2020-12 List Validation)
			if err := json.Unmarshal(aux.Items, &s.Items); err != nil {
				return err
			}
		}
	}

	// Special handling for backward compatibility and const field
	var raw map[string]jsontext.Value
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Handle backward compatibility: "definitions" (Draft-7) -> "$defs" (Draft 2020-12)
	if defsData, ok := raw["definitions"]; ok && s.Defs == nil {
		var defs map[string]*Schema
		if err := json.Unmarshal(defsData, &defs); err != nil {
			return err
		}
		s.Defs = defs
	}

	// Special handling for the const field
	if constData, ok := raw["const"]; ok {
		if s.Const == nil {
			s.Const = &ConstValue{}
		}
		if err := s.Const.UnmarshalJSON(constData); err != nil {
			return err
		}
	}

	return s.collectExtraFields(data)
}

func (s *Schema) collectExtraFields(raw []byte) error {
	var allFields map[string]any
	if err := json.Unmarshal(raw, &allFields); err != nil {
		return err
	}

	// Remove all known schema fields
	for key := range knownSchemaFields {
		delete(allFields, key)
	}

	if len(allFields) > 0 {
		s.Extra = allFields
	}
	return nil
}

// SchemaMap represents a map of string keys to *Schema values, used primarily for properties and patternProperties.
type SchemaMap map[string]*Schema

// MarshalJSON ensures that SchemaMap serializes properly as a JSON object.
func (sm SchemaMap) MarshalJSON() ([]byte, error) {
	m := make(map[string]*Schema)
	maps.Copy(m, sm)
	// Use deterministic marshaling to ensure consistent key ordering
	return json.Marshal(m, json.Deterministic(true))
}

// MarshalJSONTo implements json.MarshalerTo for JSON v2 with proper option support
func (sm *SchemaMap) MarshalJSONTo(enc *jsontext.Encoder, opts json.Options) error {
	// Ensure deterministic ordering is always enabled
	opts = json.JoinOptions(opts, json.Deterministic(true))

	if sm == nil {
		return json.MarshalEncode(enc, nil, opts)
	}
	m := make(map[string]*Schema)
	maps.Copy(m, *sm)
	return json.MarshalEncode(enc, m, opts)
}

// UnmarshalJSON ensures that JSON objects are correctly parsed into SchemaMap,
// supporting the detailed structure required for nested schema definitions.
func (sm *SchemaMap) UnmarshalJSON(data []byte) error {
	m := make(map[string]*Schema)
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*sm = SchemaMap(m)
	return nil
}

// SchemaType holds a set of SchemaType values, accommodating complex schema definitions that permit multiple types.
type SchemaType []string

// MarshalJSON customizes the JSON serialization of SchemaType.
func (st SchemaType) MarshalJSON() ([]byte, error) {
	if len(st) == 1 {
		return json.Marshal(st[0])
	}
	return json.Marshal([]string(st))
}

// UnmarshalJSON customizes the JSON deserialization into SchemaType.
func (st *SchemaType) UnmarshalJSON(data []byte) error {
	var singleType string
	if err := json.Unmarshal(data, &singleType); err == nil {
		*st = SchemaType{singleType}
		return nil
	}

	var multiType []string
	if err := json.Unmarshal(data, &multiType); err == nil {
		*st = SchemaType(multiType)
		return nil
	}

	return ErrInvalidSchemaType
}

// ConstValue represents a constant value in a JSON Schema.
type ConstValue struct {
	Value any
	IsSet bool
}

// UnmarshalJSON handles unmarshaling a JSON value into the ConstValue type.
func (cv *ConstValue) UnmarshalJSON(data []byte) error {
	if cv == nil {
		return ErrNilConstValue
	}

	cv.IsSet = true

	// If the input is "null", explicitly set Value to nil
	if string(data) == "null" {
		cv.Value = nil
		return nil
	}

	return json.Unmarshal(data, &cv.Value)
}

// MarshalJSON handles marshaling the ConstValue type back to JSON.
func (cv ConstValue) MarshalJSON() ([]byte, error) {
	if cv.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(cv.Value)
}

// SetCompiler sets a custom Compiler for the Schema and returns the Schema itself to support method chaining
func (s *Schema) SetCompiler(compiler *Compiler) *Schema {
	s.compiler = compiler
	return s
}

// Compiler gets the effective Compiler for the Schema
// Lookup order: current Schema -> parent Schema -> defaultCompiler
func (s *Schema) Compiler() *Compiler {
	if s.compiler != nil {
		return s.compiler
	}

	// Look up parent Schema's compiler
	if s.parent != nil {
		return s.parent.Compiler()
	}

	return defaultCompiler
}
