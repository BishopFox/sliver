package opfor

import (
	"context"
	"errors"
)

// AggressorBeaconExecutionKind identifies one low-level Beacon execution or
// related Postex Kit operation. The values are the exact Aggressor function
// names so importer logs do not need a second name mapping.
type AggressorBeaconExecutionKind string

const (
	// AggressorBeaconExecuteJob identifies beacon_execute_job.
	AggressorBeaconExecuteJob AggressorBeaconExecutionKind = "beacon_execute_job"
	// AggressorBeaconExecutePostexJob identifies beacon_execute_postex_job.
	AggressorBeaconExecutePostexJob AggressorBeaconExecutionKind = "beacon_execute_postex_job"
	// AggressorBeaconInlineExecute identifies beacon_inline_execute.
	AggressorBeaconInlineExecute AggressorBeaconExecutionKind = "beacon_inline_execute"
	// AggressorBeaconInlineExecutePE identifies beacon_inline_execute_pe.
	AggressorBeaconInlineExecutePE AggressorBeaconExecutionKind = "beacon_inline_execute_pe"
	// AggressorBeaconHostImportedScript identifies beacon_host_imported_script.
	AggressorBeaconHostImportedScript AggressorBeaconExecutionKind = "beacon_host_imported_script"
	// AggressorBeaconHostScript identifies beacon_host_script.
	AggressorBeaconHostScript AggressorBeaconExecutionKind = "beacon_host_script"
	// AggressorBeaconPostexKitCallbackID identifies
	// get_postex_kit_callback_id.
	AggressorBeaconPostexKitCallbackID AggressorBeaconExecutionKind = "get_postex_kit_callback_id"
)

// AggressorBeaconExecutionRequest is one resolved low-level Beacon execution
// request. Name is the exact normalized function spelling used by the script.
// RuntimeID is the nonzero process-local identity of the originating Runtime;
// Script and Span identify its call site without exposing a *Runtime.
// Bindings is the opaque callback capability for that exact Runtime; retaining
// it retains the Runtime but does not expose its evaluator or binding registry.
//
// Fields have these documented shapes:
//
//   - beacon_execute_job: BeaconID, Command, CommandArguments, and Flags.
//   - beacon_execute_postex_job: BeaconID, PID, Content, and optional
//     PackedArguments, Callback, and MessageID.
//   - beacon_inline_execute and beacon_inline_execute_pe: BeaconID, Content,
//     EntryPoint, PackedArguments, and an optional Callback. For
//     beacon_inline_execute, Content is the BOF byte string after OPFOR applies
//     the active BEACON_INLINE_EXECUTE hook.
//   - beacon_host_imported_script: BeaconID. The provider result is the hosted
//     PowerShell download cradle.
//   - beacon_host_script: BeaconID and Content containing the PowerShell script
//     to host. The provider result is the hosted-script download cradle.
//   - get_postex_kit_callback_id: no argument fields.
//
// HasPackedArguments and HasMessageID distinguish omission from an explicitly
// supplied $null Value. CallbackState likewise distinguishes omission,
// explicit $null, and a retained callable. Every source argument is resolved
// exactly once before a configured provider is called. Compound Values retain
// their ordinary identity and binary provenance; providers that retain a
// request also retain any capabilities reachable through those Values.
//
// Results from the two beacon_host functions retain their exact Value shape;
// OPFOR does not coerce them to strings. A non-nil Callback is a script-owned
// multi-shot capability. It honors the
// invocation context supplied by the provider and rejects calls after the
// creating Script generation retires, its Script unloads, or its Runtime closes. The documented callback ABI
// is (Beacon ID, result, information map); OPFOR deliberately leaves the
// provider responsible for supplying those exact Values.
type AggressorBeaconExecutionRequest struct {
	Kind AggressorBeaconExecutionKind
	Name string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span
	Bindings  AggressorBindings

	BeaconID Value

	Command          Value
	CommandArguments Value
	Flags            Value

	PID        Value
	Content    Value
	EntryPoint Value

	PackedArguments    Value
	HasPackedArguments bool

	Callback      Callable
	CallbackState AggressorCallbackState

	MessageID    Value
	HasMessageID bool
}

// AggressorBeaconExecutionProvider supplies Cobalt-owned low-level execution
// and Postex Kit behavior. HandleAggressorBeaconExecution is called
// synchronously exactly once for each valid invocation when a provider is
// configured. The four tasking operations are side-effect-only: OPFOR discards
// the returned Value and returns $null. For beacon_host_imported_script,
// beacon_host_script, and get_postex_kit_callback_id, the returned Value is
// transferred directly to the script.
//
// A returned error rejects the invocation with $null and is authoritative:
// OPFOR never retries through Host because the provider may already have
// issued a Beacon task. Implementations may be called concurrently and should
// observe ctx. They may retain request Values and Callback subject to the
// documented capability lifetimes, but must not retain ctx after this method
// returns.
type AggressorBeaconExecutionProvider interface {
	HandleAggressorBeaconExecution(context.Context, AggressorBeaconExecutionRequest) (Value, error)
}

// AggressorBeaconExecutionProviderFunc adapts a function to
// AggressorBeaconExecutionProvider.
type AggressorBeaconExecutionProviderFunc func(context.Context, AggressorBeaconExecutionRequest) (Value, error)

// HandleAggressorBeaconExecution calls function.
func (function AggressorBeaconExecutionProviderFunc) HandleAggressorBeaconExecution(
	ctx context.Context,
	request AggressorBeaconExecutionRequest,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor Beacon execution provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorBeaconExecutionProvider installs the typed importer boundary
// for beacon_execute_job, beacon_execute_postex_job, beacon_inline_execute,
// beacon_inline_execute_pe, beacon_host_imported_script, beacon_host_script,
// and get_postex_kit_callback_id. Provider errors
// are authoritative and never fall back to Host. Importer-defined
// WithFunction callbacks retain precedence over all native wrappers.
func WithAggressorBeaconExecutionProvider(provider AggressorBeaconExecutionProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor Beacon execution provider is nil")
		}
		config.aggressorBeaconExecutionProvider = provider
		return nil
	}
}
