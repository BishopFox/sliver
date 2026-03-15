package jsonschema

import "regexp"

// evaluatePattern checks if the string data matches the regular expression specified in the "pattern" schema attribute.
// According to the JSON Schema Draft 2020-12:
//   - The value of "pattern" must be a string that should be a valid regular expression, according to the ECMA-262 regular expression dialect.
//   - A string instance is considered valid if the regular expression matches the instance successfully.
//     Note: Regular expressions are not implicitly anchored.
//
// This method ensures that the string data instance conforms to the pattern constraints defined in the schema.
// If the instance does not match the pattern, it returns a EvaluationError detailing the expected pattern and the actual string.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-pattern
func evaluatePattern(schema *Schema, instance string) *EvaluationError {
	if schema.Pattern != nil {
		// Gets a compiled regular expression, or compiles and caches it if not already present.
		regExp, err := getCompiledPattern(schema)
		if err != nil {
			// Handle regular expression compilation errors.
			return NewEvaluationError("pattern", "invalid_pattern", "Invalid regular expression pattern {pattern}", map[string]any{
				"pattern": *schema.Pattern,
			})
		}

		// Check if the regular expression matches the string value.
		if !regExp.MatchString(instance) {
			// Data does not match the pattern.
			return NewEvaluationError("pattern", "pattern_mismatch", "Value does not match the required pattern {pattern}", map[string]any{
				"pattern": *schema.Pattern,
				"value":   instance,
			})
		}
	}
	return nil
}

func getCompiledPattern(schema *Schema) (*regexp.Regexp, error) {
	if schema.compiledStringPattern == nil {
		regExp, err := regexp.Compile(*schema.Pattern)
		if err != nil {
			return nil, err
		}
		schema.compiledStringPattern = regExp
	}

	return schema.compiledStringPattern, nil
}
