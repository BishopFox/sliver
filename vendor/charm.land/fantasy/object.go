package fantasy

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"reflect"

	"charm.land/fantasy/schema"
)

// ObjectMode specifies how structured output should be generated.
type ObjectMode string

const (
	// ObjectModeAuto lets the provider choose the best approach.
	ObjectModeAuto ObjectMode = "auto"

	// ObjectModeJSON forces the use of native JSON mode (if supported).
	ObjectModeJSON ObjectMode = "json"

	// ObjectModeTool forces the use of tool-based approach.
	ObjectModeTool ObjectMode = "tool"

	// ObjectModeText uses text generation with schema in prompt (fallback for models without tool/JSON support).
	ObjectModeText ObjectMode = "text"
)

// ObjectCall represents a request to generate a structured object.
type ObjectCall struct {
	Prompt            Prompt
	Schema            Schema
	SchemaName        string
	SchemaDescription string

	MaxOutputTokens  *int64
	Temperature      *float64
	TopP             *float64
	TopK             *int64
	PresencePenalty  *float64
	FrequencyPenalty *float64

	ProviderOptions ProviderOptions

	RepairText schema.ObjectRepairFunc
}

// ObjectResponse represents the response from a structured object generation.
type ObjectResponse struct {
	Object           any
	RawText          string
	Usage            Usage
	FinishReason     FinishReason
	Warnings         []CallWarning
	ProviderMetadata ProviderMetadata
}

// ObjectStreamPartType indicates the type of stream part.
type ObjectStreamPartType string

const (
	// ObjectStreamPartTypeObject is emitted when a new partial object is available.
	ObjectStreamPartTypeObject ObjectStreamPartType = "object"

	// ObjectStreamPartTypeTextDelta is emitted for text deltas (if model generates text).
	ObjectStreamPartTypeTextDelta ObjectStreamPartType = "text-delta"

	// ObjectStreamPartTypeError is emitted when an error occurs.
	ObjectStreamPartTypeError ObjectStreamPartType = "error"

	// ObjectStreamPartTypeFinish is emitted when streaming completes.
	ObjectStreamPartTypeFinish ObjectStreamPartType = "finish"
)

// ObjectStreamPart represents a single chunk in the object stream.
type ObjectStreamPart struct {
	Type             ObjectStreamPartType
	Object           any
	Delta            string
	Error            error
	Usage            Usage
	FinishReason     FinishReason
	Warnings         []CallWarning
	ProviderMetadata ProviderMetadata
}

// ObjectStreamResponse is an iterator over ObjectStreamPart.
type ObjectStreamResponse = iter.Seq[ObjectStreamPart]

// ObjectResult is a typed result wrapper returned by GenerateObject[T].
type ObjectResult[T any] struct {
	Object           T
	RawText          string
	Usage            Usage
	FinishReason     FinishReason
	Warnings         []CallWarning
	ProviderMetadata ProviderMetadata
}

// StreamObjectResult provides typed access to a streaming object generation result.
type StreamObjectResult[T any] struct {
	stream ObjectStreamResponse
	ctx    context.Context
}

// NewStreamObjectResult creates a typed stream result from an untyped stream.
func NewStreamObjectResult[T any](ctx context.Context, stream ObjectStreamResponse) *StreamObjectResult[T] {
	return &StreamObjectResult[T]{
		stream: stream,
		ctx:    ctx,
	}
}

// PartialObjectStream returns an iterator that yields progressively more complete objects.
// Only emits when the object actually changes (deduplication).
func (s *StreamObjectResult[T]) PartialObjectStream() iter.Seq[T] {
	return func(yield func(T) bool) {
		var lastObject T
		var hasEmitted bool

		for part := range s.stream {
			if part.Type == ObjectStreamPartTypeObject && part.Object != nil {
				var current T
				if err := unmarshalObject(part.Object, &current); err != nil {
					continue
				}

				if !hasEmitted || !reflect.DeepEqual(current, lastObject) {
					if !yield(current) {
						return
					}
					lastObject = current
					hasEmitted = true
				}
			}
		}
	}
}

// TextStream returns an iterator that yields text deltas.
// Useful if the model generates explanatory text alongside the object.
func (s *StreamObjectResult[T]) TextStream() iter.Seq[string] {
	return func(yield func(string) bool) {
		for part := range s.stream {
			if part.Type == ObjectStreamPartTypeTextDelta && part.Delta != "" {
				if !yield(part.Delta) {
					return
				}
			}
		}
	}
}

// FullStream returns an iterator that yields all stream parts including errors and metadata.
func (s *StreamObjectResult[T]) FullStream() iter.Seq[ObjectStreamPart] {
	return s.stream
}

// Object waits for the stream to complete and returns the final object.
// Returns an error if streaming fails or no valid object was generated.
func (s *StreamObjectResult[T]) Object() (*ObjectResult[T], error) {
	var finalObject T
	var usage Usage
	var finishReason FinishReason
	var warnings []CallWarning
	var providerMetadata ProviderMetadata
	var rawText string
	var lastError error
	hasObject := false

	for part := range s.stream {
		switch part.Type {
		case ObjectStreamPartTypeObject:
			if part.Object != nil {
				if err := unmarshalObject(part.Object, &finalObject); err == nil {
					hasObject = true
					if jsonBytes, err := json.Marshal(part.Object); err == nil {
						rawText = string(jsonBytes)
					}
				}
			}

		case ObjectStreamPartTypeError:
			lastError = part.Error

		case ObjectStreamPartTypeFinish:
			usage = part.Usage
			finishReason = part.FinishReason
			if len(part.Warnings) > 0 {
				warnings = part.Warnings
			}
			if len(part.ProviderMetadata) > 0 {
				providerMetadata = part.ProviderMetadata
			}
		}
	}

	if lastError != nil {
		return nil, lastError
	}

	if !hasObject {
		return nil, &NoObjectGeneratedError{
			RawText:      rawText,
			ParseError:   fmt.Errorf("no valid object generated in stream"),
			Usage:        usage,
			FinishReason: finishReason,
		}
	}

	return &ObjectResult[T]{
		Object:           finalObject,
		RawText:          rawText,
		Usage:            usage,
		FinishReason:     finishReason,
		Warnings:         warnings,
		ProviderMetadata: providerMetadata,
	}, nil
}

func unmarshalObject(obj any, target any) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal object: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal into target type: %w", err)
	}

	return nil
}
