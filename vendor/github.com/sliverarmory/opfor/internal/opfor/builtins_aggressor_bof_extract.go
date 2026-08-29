package opfor

import (
	"context"
	"errors"
	"fmt"
)

const (
	aggressorBOFExtractionMinimumArguments = 1
	aggressorBOFExtractionMaximumArguments = 2
)

// aggressorBOFExtractionFunctions returns the native wrapper around the
// importer-owned extracted-BOF byte contract. With no extractor, a valid call
// preserves the original reference-bearing Host invocation exactly once.
func (r *Runtime) aggressorBOFExtractionFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{
		"bof_extract": r.aggressorBOFExtract,
	}
}

func (r *Runtime) aggressorBOFExtract(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireAggressorBOFExtractionArguments(invocation); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	extractor := r.aggressorBOFExtractor
	if isNilInterface(extractor) {
		// Do not resolve, validate, copy, or replace Arguments on this
		// compatibility path. Existing Hosts retain the exact pass-by-name Cells,
		// input provenance, context, synchronous result, and error behavior.
		result, err := r.host.Call(ctx, invocation)
		return result, preserveNativeBoundaryError(ctx, err)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	// Resolve both source references exactly once before validation so request
	// fields describe one argument snapshot and no source Cell reaches the
	// extractor.
	values := invocation.Values()
	if err := requireAggressorBOFExtractionString(invocation, values[0], 1, "BOF data"); err != nil {
		return Null(), err
	}
	if len(values) == 2 {
		if err := requireAggressorBOFExtractionString(invocation, values[1], 2, "entry point"); err != nil {
			return Null(), err
		}
	}

	data, err := aggressorBOFExtractionLowBytes(ctx, values[0])
	if err != nil {
		return Null(), err
	}
	entryPoint := AggressorBOFDefaultEntryPoint
	if len(values) == 2 {
		entryPoint = values[1].String()
	}
	request := AggressorBOFExtractionRequest{
		Name:       invocation.Name,
		RuntimeID:  r.ID(),
		Script:     invocation.Script,
		Span:       invocation.Span,
		Data:       data,
		EntryPoint: entryPoint,
	}

	extracted, err := extractor.ExtractAggressorBOF(ctx, request)
	if err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	// BinaryString copies extracted, so extractor-owned storage cannot mutate a
	// completed script result. A nil slice intentionally becomes an empty string,
	// not $null.
	return BinaryString(extracted), nil
}

func requireAggressorBOFExtractionArguments(invocation Invocation) error {
	count := len(invocation.Arguments)
	if count >= aggressorBOFExtractionMinimumArguments && count <= aggressorBOFExtractionMaximumArguments {
		return nil
	}
	return fmt.Errorf("&%s: expected %d or %d arguments, received %d",
		builtinName(invocation.Name), aggressorBOFExtractionMinimumArguments,
		aggressorBOFExtractionMaximumArguments, count)
}

func requireAggressorBOFExtractionString(
	invocation Invocation,
	value Value,
	position int,
	description string,
) error {
	if value.Kind() == KindString {
		return nil
	}
	return fmt.Errorf("&%s: argument %d (%s) must be a string, received %s",
		builtinName(invocation.Name), position, description, value.Kind())
}

func aggressorBOFExtractionLowBytes(ctx context.Context, value Value) ([]byte, error) {
	units := sleepStringUnits(value)
	result := make([]byte, len(units))
	for index, unit := range units {
		if index%aggressorUtilityChunkSize == 0 {
			if err := executionContextError(ctx); err != nil {
				return nil, err
			}
		}
		result[index] = byte(unit)
	}
	if err := executionContextError(ctx); err != nil {
		return nil, err
	}
	return result, nil
}
