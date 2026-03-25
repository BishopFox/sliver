package jsonschema

// evaluateConditional evaluates the data against conditional subschemas defined by 'if', 'then', and 'else'.
// According to the JSON Schema Draft 2020-12:
//   - The "if" keyword specifies a subschema to conditionally validate data.
//   - If data validates against the "if" subschema, "then" subschema must also validate the data if "then" is present.
//   - If data does not validate against the "if" subschema, "else" subschema must validate the data if "else" is present.
//   - This function ensures data conformity based on the provided conditional subschema.
//   - The function ignores "then" and "else" if "if" is not present.
//
// This function serves as a central feature for conditional logic application in JSON Schema validation.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-core#name-if
func evaluateConditional(
	schema *Schema, instance any, evaluatedProps map[string]bool,
	evaluatedItems map[int]bool, dynamicScope *DynamicScope,
) ([]*EvaluationResult, *EvaluationError) {
	if schema.If == nil {
		// If there's no 'if' condition defined, nothing to validate conditionally.
		return nil, nil
	}

	// Evaluate the 'if' condition
	ifResult, ifEvaluatedProps, ifEvaluatedItems := schema.If.evaluate(instance, dynamicScope)

	var results []*EvaluationResult

	if ifResult != nil {
		//nolint:errcheck
		ifResult.SetEvaluationPath("/if").
			SetSchemaLocation(schema.GetSchemaLocation("/if"))

		results = append(results, ifResult)

		if ifResult.IsValid() {
			// Merge maps only if 'if' condition is successfully validated
			mergeStringMaps(evaluatedProps, ifEvaluatedProps)
			mergeIntMaps(evaluatedItems, ifEvaluatedItems)

			if schema.Then != nil {
				thenResult, thenEvaluatedProps, thenEvaluatedItems := schema.Then.evaluate(instance, dynamicScope)

				if thenResult != nil {
					//nolint:errcheck
					thenResult.SetEvaluationPath("/then").
						SetSchemaLocation(schema.GetSchemaLocation("/then"))

					results = append(results, thenResult)

					if !thenResult.IsValid() {
						return results, NewEvaluationError("then", "if_then_mismatch",
							"Value meets the 'if' condition but does not match the 'then' schema")
					}
					// Merge maps only if 'then' condition is successfully validated
					mergeStringMaps(evaluatedProps, thenEvaluatedProps)
					mergeIntMaps(evaluatedItems, thenEvaluatedItems)
				}
			}
		} else if schema.Else != nil {
			elseResult, elseEvaluatedProps, elseEvaluatedItems := schema.Else.evaluate(instance, dynamicScope)
			if elseResult != nil {
				results = append(results, elseResult)

				if !elseResult.IsValid() {
					return results, NewEvaluationError("else", "if_else_mismatch",
						"Value fails the 'if' condition and does not match the 'else' schema")
				}
				// Merge maps only if 'else' condition is successfully validated
				mergeStringMaps(evaluatedProps, elseEvaluatedProps)
				mergeIntMaps(evaluatedItems, elseEvaluatedItems)
			}
		}
	}

	return results, nil
}
