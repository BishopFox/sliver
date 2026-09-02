package opfor

import (
	"context"
	"errors"
)

// AggressorPayloadOperation identifies one documented payload, artifact, or
// stager operation. String values are the exact Aggressor function names so
// they remain stable in importer logs and adapters.
type AggressorPayloadOperation string

const (
	// AggressorPayloadHasBootstrapHint identifies -hasbootstraphint.
	AggressorPayloadHasBootstrapHint AggressorPayloadOperation = "-hasbootstraphint"
	// AggressorPayloadGenerateAll identifies all_payloads.
	AggressorPayloadGenerateAll AggressorPayloadOperation = "all_payloads"
	// AggressorPayloadArtifact identifies the deprecated artifact function.
	AggressorPayloadArtifact AggressorPayloadOperation = "artifact"
	// AggressorPayloadArtifactGeneral identifies artifact_general.
	AggressorPayloadArtifactGeneral AggressorPayloadOperation = "artifact_general"
	// AggressorPayloadArtifactSign identifies artifact_sign.
	AggressorPayloadArtifactSign AggressorPayloadOperation = "artifact_sign"
	// AggressorPayloadArtifactStager identifies artifact_stager.
	AggressorPayloadArtifactStager AggressorPayloadOperation = "artifact_stager"
	// AggressorPayloadExport identifies payload.
	AggressorPayloadExport AggressorPayloadOperation = "payload"
	// AggressorPayloadBootstrapHint identifies payload_bootstrap_hint.
	AggressorPayloadBootstrapHint AggressorPayloadOperation = "payload_bootstrap_hint"
	// AggressorPayloadExportLocal identifies payload_local.
	AggressorPayloadExportLocal AggressorPayloadOperation = "payload_local"
	// AggressorPayloadPowerShell identifies the deprecated powershell function.
	AggressorPayloadPowerShell AggressorPayloadOperation = "powershell"
	// AggressorPayloadShellcode identifies the deprecated shellcode function.
	AggressorPayloadShellcode AggressorPayloadOperation = "shellcode"
	// AggressorPayloadStager identifies stager.
	AggressorPayloadStager AggressorPayloadOperation = "stager"
	// AggressorPayloadStagerBindPipe identifies stager_bind_pipe.
	AggressorPayloadStagerBindPipe AggressorPayloadOperation = "stager_bind_pipe"
	// AggressorPayloadStagerBindTCP identifies stager_bind_tcp.
	AggressorPayloadStagerBindTCP AggressorPayloadOperation = "stager_bind_tcp"
)

// AggressorPayloadRequest is one resolved importer-owned payload operation.
// Name is the exact normalized function spelling used by the script.
// RuntimeID is the nonzero process-local identity of the originating Runtime;
// Script and Span identify the call site without exposing a *Runtime.
// Bindings is the opaque callback capability for that exact Runtime; retaining
// it retains the Runtime but does not expose its evaluator or binding registry.
//
// Arguments is an exact positional snapshot resolved once before the provider
// call. Its documented shapes are:
//
//   - -hasbootstraphint: payload bytes
//   - all_payloads: destination folder, sign flag, system-call method, then
//     optional HTTP library, DNS communication mode, and profile override
//   - artifact: listener, artifact type, then the optional deprecated value
//     and architecture (the reference's example supplies only the first two)
//   - artifact_general: shellcode, artifact type, architecture
//   - artifact_sign: artifact bytes
//   - artifact_stager: listener, artifact type, architecture, then an optional
//     Payload Store information map
//   - payload: listener, architecture, exit method, system-call method, then
//     optional HTTP library, DNS communication mode, and profile override
//   - payload_bootstrap_hint: payload bytes and function name
//   - payload_local: parent Beacon ID, listener, architecture, exit method,
//     system-call method, then an optional HTTP library
//   - powershell: listener, local-host flag, then an optional architecture;
//     omission is preserved because the public reference's example omits it
//   - shellcode: listener, remote-target flag, architecture
//   - stager: listener and architecture
//   - stager_bind_pipe: listener
//   - stager_bind_tcp: listener, architecture, and port
//
// The slice is detached from Invocation, but Values are neither replaced nor
// cloned. Its length therefore distinguishes omission from an explicitly
// supplied $null, and compound/object/function Values retain their ordinary
// reference identity. The all_payloads typed route applies Sleep string
// coercion only while checking the function reference's exact system-call,
// HTTP-library, and DNS-mode enumerations; the provider still receives the
// original Values. Providers that retain a request also retain any capabilities
// reachable through those Values.
type AggressorPayloadRequest struct {
	Operation AggressorPayloadOperation
	Name      string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span
	Bindings  AggressorBindings

	Arguments []Value
}

// Arg returns a resolved positional argument or $null when index is absent.
// Use HasArgument when omission must be distinguished from explicit $null.
func (request AggressorPayloadRequest) Arg(index int) Value {
	if index < 0 || index >= len(request.Arguments) {
		return Null()
	}
	return request.Arguments[index]
}

// HasArgument reports whether a positional argument was supplied.
func (request AggressorPayloadRequest) HasArgument(index int) bool {
	return index >= 0 && index < len(request.Arguments)
}

// AggressorPayloadProvider supplies Cobalt-owned payload, artifact, and
// stager generation. OPFOR never generates these payloads itself. The method
// is called synchronously exactly once for each valid invocation when a
// provider is configured. A returned error is authoritative and is never
// retried through Host because generation or signing may already have caused
// an external effect.
//
// Successful provider Values are transferred directly to script code without
// coercion, validation, cloning, or serialization, except that
// -hasbootstraphint is normalized through Sleep truthiness to Bool. The public
// reference describes all operations in this provider as returning a value;
// OPFOR consequently preserves the provider result for each of them. Compound
// results remain provider-owned references, so providers should return
// detached graphs when script mutation must not affect backing state.
//
// Implementations may be called concurrently and should observe ctx. They may
// retain request Values subject to the capability lifetime above, but must not
// retain ctx after HandleAggressorPayload returns.
type AggressorPayloadProvider interface {
	HandleAggressorPayload(context.Context, AggressorPayloadRequest) (Value, error)
}

// AggressorPayloadProviderFunc adapts a function to AggressorPayloadProvider.
type AggressorPayloadProviderFunc func(context.Context, AggressorPayloadRequest) (Value, error)

// HandleAggressorPayload calls function.
func (function AggressorPayloadProviderFunc) HandleAggressorPayload(
	ctx context.Context,
	request AggressorPayloadRequest,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor payload provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorPayloadProvider installs the typed importer boundary for the
// documented payload, legacy artifact, and stager functions. The separate
// AggressorArtifactProvider remains the boundary for artifact_payload and
// artifact_stageless. Provider errors are authoritative; importer-defined
// WithFunction callbacks retain precedence over every native wrapper.
func WithAggressorPayloadProvider(provider AggressorPayloadProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor payload provider is nil")
		}
		config.aggressorPayloadProvider = provider
		return nil
	}
}
