package opfor

import (
	"context"
	"errors"
)

// AggressorPEOperation identifies one Cobalt-owned PE inspection or
// transformation operation. String values are the exact documented Aggressor
// function names so they remain stable in importer logs and adapters.
type AggressorPEOperation string

const (
	// AggressorPEInsertRichHeader identifies pe_insert_rich_header.
	AggressorPEInsertRichHeader AggressorPEOperation = "pe_insert_rich_header"
	// AggressorPEMaskSection identifies pe_mask_section.
	AggressorPEMaskSection AggressorPEOperation = "pe_mask_section"
	// AggressorPEPatchCode identifies pe_patch_code.
	AggressorPEPatchCode AggressorPEOperation = "pe_patch_code"
	// AggressorPERemoveRichHeader identifies pe_remove_rich_header.
	AggressorPERemoveRichHeader AggressorPEOperation = "pe_remove_rich_header"
	// AggressorPESetCompileTimeWithString identifies
	// pe_set_compile_time_with_string.
	AggressorPESetCompileTimeWithString AggressorPEOperation = "pe_set_compile_time_with_string"
	// AggressorPESetExportName identifies pe_set_export_name.
	AggressorPESetExportName AggressorPEOperation = "pe_set_export_name"
	// AggressorPESetValueAt identifies pe_set_value_at.
	AggressorPESetValueAt AggressorPEOperation = "pe_set_value_at"
	// AggressorPEDump identifies pedump.
	AggressorPEDump AggressorPEOperation = "pedump"
)

// AggressorPERequest is one resolved Cobalt-owned PE request. Name is the
// exact normalized function spelling used by the script. RuntimeID is the
// nonzero process-local identity of the originating Runtime; Script and Span
// identify the call site without exposing a *Runtime.
//
// Arguments is an exact positional snapshot resolved once before provider
// dispatch. The slice is detached from Invocation, while its Values retain
// ordinary binary, compound, function, and object identity and provenance.
// Providers which retain a request therefore retain any capabilities reachable
// through those Values and should snapshot or detach them when appropriate.
type AggressorPERequest struct {
	Operation AggressorPEOperation
	Name      string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span

	Arguments []Value
}

// Arg returns a resolved positional argument or $null when index is absent.
func (request AggressorPERequest) Arg(index int) Value {
	if index < 0 || index >= len(request.Arguments) {
		return Null()
	}
	return request.Arguments[index]
}

// HasArgument reports whether a positional argument was supplied.
func (request AggressorPERequest) HasArgument(index int) bool {
	return index >= 0 && index < len(request.Arguments)
}

// AggressorPEProvider supplies Cobalt-owned PE parsing and transformations.
// OPFOR calls HandleAggressorPE synchronously exactly once for each valid
// invocation when a provider is configured. A successful returned Value is
// transferred directly to script code without coercion, validation, cloning,
// or serialization. OPFOR validates only the documented arity and does not
// parse, modify, or otherwise infer the PE algorithm on this route.
// pe_set_export_name is the one exception to exact-arity validation: its
// current argument table lists one content argument while both executable
// examples supply content and an export name. OPFOR accepts the evidence union
// of one or two arguments; HasArgument(1) preserves which form was used for the
// provider rather than inventing a missing name.
//
// A returned error rejects the invocation with $null and is authoritative:
// OPFOR never retries through Host after a provider may have done work.
// Implementations may be called concurrently and should observe ctx. They may
// retain request Values subject to the capability lifetime above, but must not
// retain ctx after HandleAggressorPE returns.
type AggressorPEProvider interface {
	HandleAggressorPE(context.Context, AggressorPERequest) (Value, error)
}

// AggressorPEProviderFunc adapts a function to AggressorPEProvider.
type AggressorPEProviderFunc func(context.Context, AggressorPERequest) (Value, error)

// HandleAggressorPE calls function.
func (function AggressorPEProviderFunc) HandleAggressorPE(
	ctx context.Context,
	request AggressorPERequest,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor PE provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorPEProvider installs the typed importer boundary for the
// documentation-complete Cobalt-owned PE helpers. WithFunction overrides
// retain precedence over these native wrappers.
func WithAggressorPEProvider(provider AggressorPEProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor PE provider is nil")
		}
		config.aggressorPEProvider = provider
		return nil
	}
}
