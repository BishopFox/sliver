package opfor

import (
	"context"
	"errors"
)

// AggressorBOFDefaultEntryPoint is the documented entry point used by
// bof_extract when its optional second argument is omitted.
const AggressorBOFDefaultEntryPoint = "sleep_mask"

// AggressorBOFExtractionRequest is one resolved bof_extract request. Name is
// the exact normalized function spelling used by the script. RuntimeID is the
// nonzero process-local identity of the originating Runtime; Script and Span
// identify the call site without exposing a *Runtime.
//
// Data is a detached low-byte snapshot of the first Sleep string argument,
// including embedded NULs. EntryPoint is the exact second string argument, or
// AggressorBOFDefaultEntryPoint when that argument was omitted. An extractor
// may retain or mutate Data without changing the source scalar.
type AggressorBOFExtractionRequest struct {
	Name string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span

	Data       []byte
	EntryPoint string
}

// AggressorBOFExtractor supplies Cobalt-owned bof_extract behavior. OPFOR
// deliberately does not guess Cobalt Strike's unpublished extracted-BOF byte
// envelope and never parses, links, relocates, or executes the object locally.
// ExtractAggressorBOF is called synchronously exactly once for each valid
// invocation when an extractor is configured.
//
// Successful bytes are copied into a script-visible BinaryString. A nil or
// zero-length byte slice is still a successful zero-length string; this keeps
// the public strlen(result) failure check observable without conflating it
// with an extractor error. A returned error rejects the invocation with
// $null and is authoritative: OPFOR never retries through Host. Implementations
// may be called concurrently, should observe ctx, and must not retain ctx after
// ExtractAggressorBOF returns.
type AggressorBOFExtractor interface {
	ExtractAggressorBOF(context.Context, AggressorBOFExtractionRequest) ([]byte, error)
}

// AggressorBOFExtractorFunc adapts a function to AggressorBOFExtractor.
type AggressorBOFExtractorFunc func(context.Context, AggressorBOFExtractionRequest) ([]byte, error)

// ExtractAggressorBOF calls function.
func (function AggressorBOFExtractorFunc) ExtractAggressorBOF(
	ctx context.Context,
	request AggressorBOFExtractionRequest,
) ([]byte, error) {
	if function == nil {
		return nil, errors.New("opfor: Aggressor BOF extractor is nil")
	}
	return function(ctx, request)
}

// WithAggressorBOFExtractor installs the typed importer boundary for
// bof_extract. With no extractor, the native wrapper forwards the original
// reference-bearing Invocation to Host. Extractor errors are authoritative and
// never fall back to Host. Importer-defined WithFunction callbacks retain
// precedence over the native wrapper.
func WithAggressorBOFExtractor(extractor AggressorBOFExtractor) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(extractor) {
			return errors.New("opfor: Aggressor BOF extractor is nil")
		}
		config.aggressorBOFExtractor = extractor
		return nil
	}
}
