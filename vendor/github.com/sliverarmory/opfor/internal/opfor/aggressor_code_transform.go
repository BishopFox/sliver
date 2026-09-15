package opfor

import (
	"context"
	"errors"
)

// AggressorCodeTransformOperation identifies one documented Cobalt-owned
// code or script transformation. String values are the exact Aggressor
// function names so they remain stable in importer logs and adapters.
type AggressorCodeTransformOperation string

const (
	// AggressorCodeTransformEncode identifies encode.
	AggressorCodeTransformEncode AggressorCodeTransformOperation = "encode"
	// AggressorCodeTransformPowerShellCompress identifies powershell_compress.
	AggressorCodeTransformPowerShellCompress AggressorCodeTransformOperation = "powershell_compress"
	// AggressorCodeTransformVBS identifies transform_vbs.
	AggressorCodeTransformVBS AggressorCodeTransformOperation = "transform_vbs"
)

// AggressorCodeTransformRequest is one resolved Cobalt-owned transformation
// request. Name is the exact normalized function spelling used by the script.
// RuntimeID is the nonzero process-local identity of the originating Runtime;
// Script and Span identify the call site without exposing a *Runtime.
//
// Arguments is an exact positional snapshot resolved once before provider
// dispatch. Its documented shapes are:
//
//   - encode: position-independent code, encoder name, architecture
//   - powershell_compress: PowerShell script
//   - transform_vbs: shellcode, maximum plaintext-run length
//
// The slice is detached from Invocation, while its Values retain ordinary
// binary, compound, function, and object identity and provenance. Providers
// which retain a request therefore retain any capabilities reachable through
// those Values and should snapshot or detach them when appropriate.
type AggressorCodeTransformRequest struct {
	Operation AggressorCodeTransformOperation
	Name      string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span

	Arguments []Value
}

// Arg returns a resolved positional argument or $null when index is absent.
func (request AggressorCodeTransformRequest) Arg(index int) Value {
	if index < 0 || index >= len(request.Arguments) {
		return Null()
	}
	return request.Arguments[index]
}

// HasArgument reports whether a positional argument was supplied.
func (request AggressorCodeTransformRequest) HasArgument(index int) bool {
	return index >= 0 && index < len(request.Arguments)
}

// AggressorCodeTransformProvider supplies Cobalt-owned position-independent
// code encoding, PowerShell compression, and VBS shellcode transformation.
// OPFOR calls HandleAggressorCodeTransform synchronously exactly once for each
// valid invocation when a provider is configured and no applicable script hook
// handled the request. OPFOR validates only documented arity and does not
// reproduce or infer the unpublished output algorithms.
//
// A successful returned Value is transferred directly to script code without
// coercion, validation, cloning, or serialization. A returned error rejects
// the invocation with $null and is authoritative: OPFOR never retries through
// Host after a provider may have done work. Implementations may be called
// concurrently and should observe ctx. They may retain request Values subject
// to the capability lifetime above, but must not retain ctx after this method
// returns.
type AggressorCodeTransformProvider interface {
	HandleAggressorCodeTransform(context.Context, AggressorCodeTransformRequest) (Value, error)
}

// AggressorCodeTransformProviderFunc adapts a function to
// AggressorCodeTransformProvider.
type AggressorCodeTransformProviderFunc func(context.Context, AggressorCodeTransformRequest) (Value, error)

// HandleAggressorCodeTransform calls function.
func (function AggressorCodeTransformProviderFunc) HandleAggressorCodeTransform(
	ctx context.Context,
	request AggressorCodeTransformRequest,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor code-transform provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorCodeTransformProvider installs the typed importer boundary for
// encode, powershell_compress, and transform_vbs. A script-defined
// POWERSHELL_COMPRESS hook takes precedence over the provider for
// powershell_compress. Provider errors are authoritative and never fall back
// to Host. WithFunction overrides retain precedence over every native wrapper.
func WithAggressorCodeTransformProvider(provider AggressorCodeTransformProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor code-transform provider is nil")
		}
		config.aggressorCodeTransformProvider = provider
		return nil
	}
}
