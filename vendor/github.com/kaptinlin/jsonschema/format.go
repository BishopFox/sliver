package jsonschema

// evaluateFormat checks if the data conforms to the format specified in the schema.
// According to the JSON Schema Draft 2020-12:
//   - The "format" keyword defines the data format expected for a value.
//   - The format must be a string that names a specific format which the value should conform to.
//   - The function uses custom formats first, then falls back to the global `Formats` map.
//   - If the format is not supported or not found, it may fall back to a no-op validation depending on configuration.
//
// This method ensures that data matches the expected format as specified in the schema.
// It handles formats as annotations by default, but can assert format validation if configured.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-format
func evaluateFormat(schema *Schema, value any) *EvaluationError {
	if schema.Format == nil {
		return nil
	}

	formatName := *schema.Format
	var formatDef *FormatDef
	var customValidator func(any) bool

	// Get the effective compiler (may be from parent or defaultCompiler)
	compiler := schema.GetCompiler()

	// 1. Check compiler-specific custom formats first
	if compiler != nil {
		compiler.customFormatsRW.RLock()
		formatDef = compiler.customFormats[formatName]
		compiler.customFormatsRW.RUnlock()
	}

	if formatDef != nil {
		// Found in custom formats
		if formatDef.Type != "" {
			valueType := getDataType(value)
			if !matchesType(valueType, formatDef.Type) {
				return nil // Type doesn't match, so skip validation
			}
		}
		customValidator = formatDef.Validate
	} else if globalValidator, ok := Formats[formatName]; ok {
		// Fallback to global formats
		customValidator = globalValidator
	}

	// If a validator was found (either custom or global)
	if customValidator != nil {
		if !customValidator(value) {
			if compiler != nil && compiler.AssertFormat {
				return NewEvaluationError("format", "format_mismatch", "Value does not match format '{format}'", map[string]any{"format": formatName})
			}
		}
		return nil // Validation passed or not asserted
	}

	// If no validator was found and AssertFormat is true, fail
	if compiler != nil && compiler.AssertFormat {
		return NewEvaluationError("format", "unknown_format", "Unknown format '{format}'", map[string]any{"format": formatName})
	}

	return nil // Default behavior: ignore unknown formats
}

// matchesType checks if a value type matches the required type
func matchesType(valueType, requiredType string) bool {
	if requiredType == "" {
		return true // No type restriction
	}

	// Special handling: integer is also considered number
	if requiredType == "number" && valueType == "integer" {
		return true
	}

	return valueType == requiredType
}
