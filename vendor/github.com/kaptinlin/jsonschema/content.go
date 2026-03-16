package jsonschema

// evaluateContent checks if the given data conforms to the encoding, media type, and content schema specified in the schema.
// According to the JSON Schema Draft 2020-12:
//   - The "contentEncoding" property defines how a string should be decoded from encoded binary data.
//   - The "contentMediaType" describes the media type that the decoded data should conform to.
//   - The "contentSchema" provides a schema to validate the structure of the decoded and unmarshalled data.
//
// This method ensures that the data instance conforms to the encoding, media type, and content schema constraints defined in the schema.
// If any stage fails, it returns a EvaluationError detailing the specific failure.
//
// References:
//   - https://json-schema.org/draft/2020-12/json-schema-validation#name-contentencoding
//   - https://json-schema.org/draft/2020-12/json-schema-validation#name-contentmediatype
//   - https://json-schema.org/draft/2020-12/json-schema-validation#name-contentschema
func evaluateContent(schema *Schema, instance any, _ map[string]bool, _ map[int]bool, dynamicScope *DynamicScope) (*EvaluationResult, *EvaluationError) {
	value, isString := instance.(string)
	if !isString {
		return nil, nil // If instance is not a string, content validation is not applicable.
	}

	var content []byte
	var parsedValue any
	var err error

	// Decode the content if encoding is specified
	if schema.ContentEncoding != nil {
		decoder, exists := schema.compiler.Decoders[*schema.ContentEncoding]
		if !exists {
			return nil, NewEvaluationError("contentEncoding", "unsupported_encoding", "Encoding '{encoding}' is not supported", map[string]any{
				"encoding": *schema.ContentEncoding,
			})
		}
		content, err = decoder(value)
		if err != nil {
			return nil, NewEvaluationError("contentEncoding", "invalid_encoding", "Error decoding data with '{encoding}'", map[string]any{
				"encoding": *schema.ContentEncoding,
				"error":    err.Error(),
			})
		}
	} else {
		content = []byte(value) // Assume the content is the raw string if no encoding is specified
	}

	// Handle content media type validation
	if schema.ContentMediaType != nil {
		unmarshal, exists := schema.compiler.MediaTypes[*schema.ContentMediaType]
		if !exists {
			return nil, NewEvaluationError("contentMediaType", "unsupported_media_type", "Media type '{media_type}' is not supported", map[string]any{
				"media_type": *schema.ContentMediaType,
			})
		}
		parsedValue, err = unmarshal(content)
		if err != nil {
			return nil, NewEvaluationError("contentMediaType", "invalid_media_type", "Error unmarshalling data with media type '{media_type}'", map[string]any{
				"media_type": *schema.ContentMediaType,
				"error":      err.Error(),
			})
		}
	} else {
		parsedValue = content // If no media type is specified, pass the raw content
	}

	// Evaluate against the content schema if specified and value was decoded
	if schema.ContentSchema != nil {
		result, _, _ := schema.ContentSchema.evaluate(parsedValue, dynamicScope)
		if result != nil {
			//nolint:errcheck
			result.SetEvaluationPath("/contentSchema").
				SetSchemaLocation(schema.GetSchemaLocation("/contentSchema"))

			if !result.IsValid() {
				return result, NewEvaluationError("contentSchema", "content_schema_mismatch", "Content does not match the schema")
			}
			return result, nil
		}
	}

	return nil, nil
}
