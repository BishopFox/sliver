package jsonschema

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/kaptinlin/jsonpointer"
)

// resolveRef resolves a reference to another schema, either locally or globally, supporting both $ref and $dynamicRef.
func (s *Schema) resolveRef(ref string) (*Schema, error) {
	if ref == "#" {
		return s.getRootSchema(), nil
	}

	if strings.HasPrefix(ref, "#") {
		return s.resolveAnchor(ref[1:])
	}

	// Resolve the full URL if ref is a relative URL
	if !isAbsoluteURI(ref) && s.baseURI != "" {
		ref = resolveRelativeURI(s.baseURI, ref)
	}

	// Handle full URL references
	return s.resolveRefWithFullURL(ref)
}

func (s *Schema) resolveAnchor(anchorName string) (*Schema, error) {
	var schema *Schema
	var err error

	if strings.HasPrefix(anchorName, "/") {
		schema, err = s.resolveJSONPointer(anchorName)
	} else {
		if schema, ok := s.anchors[anchorName]; ok {
			return schema, nil
		}

		if schema, ok := s.dynamicAnchors[anchorName]; ok {
			return schema, nil
		}
	}

	if schema == nil && s.parent != nil {
		return s.parent.resolveAnchor(anchorName)
	}

	return schema, err
}

// resolveRefWithFullURL resolves a full URL reference to another schema.
func (s *Schema) resolveRefWithFullURL(ref string) (*Schema, error) {
	root := s.getRootSchema()
	if resolved, err := root.getSchema(ref); err == nil {
		return resolved, nil
	}

	// If not found in the current schema or its parents, look for the reference in the compiler
	resolved, err := s.GetCompiler().GetSchema(ref)
	if err != nil {
		return nil, ErrGlobalReferenceResolution
	}
	return resolved, nil
}

// resolveJSONPointer resolves a JSON Pointer within the schema based on JSON Schema structure.
func (s *Schema) resolveJSONPointer(pointer string) (*Schema, error) {
	if pointer == "/" {
		return s, nil
	}

	// Parse JSON Pointer using the jsonpointer library
	// This handles ~ escaping (~ -> ~0, / -> ~1) automatically
	segments := jsonpointer.Parse(pointer)
	currentSchema := s
	previousSegment := ""

	for i, segment := range segments {
		// jsonpointer.Parse handles ~0 and ~1 escaping, but not URL percent encoding
		// We need to handle URL percent encoding separately for JSON Schema compatibility
		decodedSegment, err := url.PathUnescape(segment)
		if err != nil {
			return nil, ErrJSONPointerSegmentDecode
		}

		nextSchema, found := findSchemaInSegment(currentSchema, decodedSegment, previousSegment)
		if found {
			currentSchema = nextSchema
			previousSegment = decodedSegment
			continue
		}

		if !found && i == len(segments)-1 {
			// If no schema is found and it's the last segment, throw error
			return nil, ErrJSONPointerSegmentNotFound
		}

		previousSegment = decodedSegment
	}

	return currentSchema, nil
}

// findSchemaInSegment finds a schema within a given segment.
func findSchemaInSegment(currentSchema *Schema, segment string, previousSegment string) (*Schema, bool) {
	switch previousSegment {
	case "properties":
		if currentSchema.Properties != nil {
			if schema, exists := (*currentSchema.Properties)[segment]; exists {
				return schema, true
			}
		}
	case "prefixItems":
		index, err := strconv.Atoi(segment)

		if err == nil && currentSchema.PrefixItems != nil && index < len(currentSchema.PrefixItems) {
			return currentSchema.PrefixItems[index], true
		}
	case "$defs", "definitions": // Support both $defs (2020-12) and definitions (Draft-7) for backward compatibility
		if defSchema, exists := currentSchema.Defs[segment]; exists {
			return defSchema, true
		}
	case "items":
		if currentSchema.Items != nil {
			return currentSchema.Items, true
		}
	}
	return nil, false
}

// ResolveUnresolvedReferences tries to resolve any previously unresolved references.
// This is called after new schemas are added to the compiler.
func (s *Schema) ResolveUnresolvedReferences() {
	// Try to resolve unresolved $ref
	if s.Ref != "" && s.ResolvedRef == nil {
		if resolved, err := s.resolveRef(s.Ref); err == nil {
			s.ResolvedRef = resolved
		}
	}

	// Try to resolve unresolved $dynamicRef
	if s.DynamicRef != "" && s.ResolvedDynamicRef == nil {
		if resolved, err := s.resolveRef(s.DynamicRef); err == nil {
			s.ResolvedDynamicRef = resolved
		}
	}

	s.walkNestedSchemas((*Schema).ResolveUnresolvedReferences)
}

func (s *Schema) resolveReferences() {
	// Resolve the root reference if this schema itself is a reference.
	if s.Ref != "" {
		resolved, err := s.resolveRef(s.Ref)
		if err == nil {
			s.ResolvedRef = resolved
		}
		// If resolution fails, leave ResolvedRef as nil and validation will handle this gracefully.
	}

	if s.DynamicRef != "" {
		resolved, err := s.resolveRef(s.DynamicRef)
		if err == nil {
			s.ResolvedDynamicRef = resolved
		}
		// If resolution fails, leave ResolvedDynamicRef as nil and validation will handle this gracefully.
	}

	s.walkNestedSchemas((*Schema).resolveReferences)
}

// walkNestedSchemas applies fn recursively to all nested subschemas.
func (s *Schema) walkNestedSchemas(fn func(*Schema)) {
	if s.Defs != nil {
		for _, defSchema := range s.Defs {
			fn(defSchema)
		}
	}

	if s.Properties != nil {
		for _, schema := range *s.Properties {
			if schema != nil {
				fn(schema)
			}
		}
	}

	for _, schemas := range [][]*Schema{s.AllOf, s.AnyOf, s.OneOf} {
		for _, schema := range schemas {
			if schema != nil {
				fn(schema)
			}
		}
	}

	if s.Not != nil {
		fn(s.Not)
	}
	if s.Items != nil {
		fn(s.Items)
	}
	for _, schema := range s.PrefixItems {
		fn(schema)
	}
	if s.AdditionalProperties != nil {
		fn(s.AdditionalProperties)
	}
	if s.Contains != nil {
		fn(s.Contains)
	}
	if s.PatternProperties != nil {
		for _, schema := range *s.PatternProperties {
			fn(schema)
		}
	}
}

// GetUnresolvedReferenceURIs returns a list of URIs that this schema references but are not yet resolved.
func (s *Schema) GetUnresolvedReferenceURIs() []string {
	var unresolvedURIs []string

	// Check direct references
	if s.Ref != "" && s.ResolvedRef == nil {
		unresolvedURIs = append(unresolvedURIs, s.Ref)
	}

	if s.DynamicRef != "" && s.ResolvedDynamicRef == nil {
		unresolvedURIs = append(unresolvedURIs, s.DynamicRef)
	}

	// Recursively check nested schemas
	if s.Defs != nil {
		for _, defSchema := range s.Defs {
			unresolvedURIs = append(unresolvedURIs, defSchema.GetUnresolvedReferenceURIs()...)
		}
	}

	if s.Properties != nil {
		for _, propSchema := range *s.Properties {
			if propSchema != nil {
				unresolvedURIs = append(unresolvedURIs, propSchema.GetUnresolvedReferenceURIs()...)
			}
		}
	}

	// Check other schema fields
	unresolvedURIs = append(unresolvedURIs, getUnresolvedFromList(s.AllOf)...)
	unresolvedURIs = append(unresolvedURIs, getUnresolvedFromList(s.AnyOf)...)
	unresolvedURIs = append(unresolvedURIs, getUnresolvedFromList(s.OneOf)...)

	if s.Not != nil {
		unresolvedURIs = append(unresolvedURIs, s.Not.GetUnresolvedReferenceURIs()...)
	}

	if s.Items != nil {
		unresolvedURIs = append(unresolvedURIs, s.Items.GetUnresolvedReferenceURIs()...)
	}

	if s.PrefixItems != nil {
		for _, schema := range s.PrefixItems {
			unresolvedURIs = append(unresolvedURIs, schema.GetUnresolvedReferenceURIs()...)
		}
	}

	if s.AdditionalProperties != nil {
		unresolvedURIs = append(unresolvedURIs, s.AdditionalProperties.GetUnresolvedReferenceURIs()...)
	}

	if s.Contains != nil {
		unresolvedURIs = append(unresolvedURIs, s.Contains.GetUnresolvedReferenceURIs()...)
	}

	if s.PatternProperties != nil {
		for _, schema := range *s.PatternProperties {
			unresolvedURIs = append(unresolvedURIs, schema.GetUnresolvedReferenceURIs()...)
		}
	}

	return unresolvedURIs
}

// getUnresolvedFromList returns unresolved references from a list of schemas.
func getUnresolvedFromList(schemas []*Schema) []string {
	var unresolvedURIs []string
	for _, schema := range schemas {
		if schema != nil {
			unresolvedURIs = append(unresolvedURIs, schema.GetUnresolvedReferenceURIs()...)
		}
	}
	return unresolvedURIs
}
