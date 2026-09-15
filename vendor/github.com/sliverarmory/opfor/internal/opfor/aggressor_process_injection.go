package opfor

import (
	"context"
	"errors"
)

// AggressorProcessInjectionOperation identifies one documented process-
// injection selection or inventory operation. String values are the exact
// Aggressor function names so they remain stable in importer logs and
// adapters.
type AggressorProcessInjectionOperation string

const (
	// AggressorProcessInjectionExplicitGet identifies pi_explicit_get.
	AggressorProcessInjectionExplicitGet AggressorProcessInjectionOperation = "pi_explicit_get"
	// AggressorProcessInjectionExplicitInfo identifies pi_explicit_info.
	AggressorProcessInjectionExplicitInfo AggressorProcessInjectionOperation = "pi_explicit_info"
	// AggressorProcessInjectionExplicitSet identifies pi_explicit_set.
	AggressorProcessInjectionExplicitSet AggressorProcessInjectionOperation = "pi_explicit_set"
	// AggressorProcessInjectionSpawnGet identifies pi_spawn_get.
	AggressorProcessInjectionSpawnGet AggressorProcessInjectionOperation = "pi_spawn_get"
	// AggressorProcessInjectionSpawnInfo identifies pi_spawn_info.
	AggressorProcessInjectionSpawnInfo AggressorProcessInjectionOperation = "pi_spawn_info"
	// AggressorProcessInjectionSpawnSet identifies pi_spawn_set.
	AggressorProcessInjectionSpawnSet AggressorProcessInjectionOperation = "pi_spawn_set"
	// AggressorProcessInjectionUserExplicitClear identifies
	// pi_user_explicit_clear.
	AggressorProcessInjectionUserExplicitClear AggressorProcessInjectionOperation = "pi_user_explicit_clear"
	// AggressorProcessInjectionUserExplicitGet identifies pi_user_explicit_get.
	AggressorProcessInjectionUserExplicitGet AggressorProcessInjectionOperation = "pi_user_explicit_get"
	// AggressorProcessInjectionUserExplicitGetMap identifies
	// pi_user_explicit_get_map.
	AggressorProcessInjectionUserExplicitGetMap AggressorProcessInjectionOperation = "pi_user_explicit_get_map"
	// AggressorProcessInjectionUserExplicitGetNames identifies
	// pi_user_explicit_get_names.
	AggressorProcessInjectionUserExplicitGetNames AggressorProcessInjectionOperation = "pi_user_explicit_get_names"
	// AggressorProcessInjectionUserExplicitSet identifies pi_user_explicit_set.
	AggressorProcessInjectionUserExplicitSet AggressorProcessInjectionOperation = "pi_user_explicit_set"
	// AggressorProcessInjectionUserSpawnClear identifies pi_user_spawn_clear.
	AggressorProcessInjectionUserSpawnClear AggressorProcessInjectionOperation = "pi_user_spawn_clear"
	// AggressorProcessInjectionUserSpawnGet identifies pi_user_spawn_get.
	AggressorProcessInjectionUserSpawnGet AggressorProcessInjectionOperation = "pi_user_spawn_get"
	// AggressorProcessInjectionUserSpawnGetMap identifies pi_user_spawn_get_map.
	AggressorProcessInjectionUserSpawnGetMap AggressorProcessInjectionOperation = "pi_user_spawn_get_map"
	// AggressorProcessInjectionUserSpawnGetNames identifies
	// pi_user_spawn_get_names.
	AggressorProcessInjectionUserSpawnGetNames AggressorProcessInjectionOperation = "pi_user_spawn_get_names"
	// AggressorProcessInjectionUserSpawnSet identifies pi_user_spawn_set.
	AggressorProcessInjectionUserSpawnSet AggressorProcessInjectionOperation = "pi_user_spawn_set"
)

// AggressorProcessInjectionRequest is one resolved request for Cobalt's
// process-injection configuration. Name is the exact normalized function
// spelling used by the script. RuntimeID is the nonzero process-local identity
// of the originating Runtime; Script and Span identify the call site without
// exposing a *Runtime.
// Bindings is the opaque callback capability for that exact Runtime; retaining
// it retains the Runtime but does not expose its evaluator or binding registry.
//
// SelectionName is populated only for pi_explicit_set, pi_spawn_set,
// pi_user_explicit_set, and pi_user_spawn_set. It is resolved exactly once and
// transferred without coercion or cloning. Although the public reference
// describes a string name, it does not document Cobalt's coercion behavior;
// importers may therefore enforce their own accepted representation.
type AggressorProcessInjectionRequest struct {
	Operation AggressorProcessInjectionOperation
	Name      string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span
	Bindings  AggressorBindings

	SelectionName Value
}

// AggressorProcessInjectionProvider supplies Cobalt's built-in and
// user-defined process-injection inventories and active selections. OPFOR
// calls HandleAggressorProcessInjection synchronously exactly once for each
// valid invocation when a provider is configured.
//
// Successful query results are transferred directly to script code without
// coercion, validation, or cloning. The public reference describes strings
// for active selections, arrays for available-name queries, maps for
// user-defined inventory queries, and $null when no user-defined selection is
// active. OPFOR deliberately leaves those concrete result policies to the
// importer. Set and clear operations are documented only as effects, so their
// provider result is discarded and the script receives $null.
//
// A returned error rejects the invocation with $null and is authoritative:
// OPFOR never retries through Host because a selection mutation may already
// have taken effect. Implementations may be called concurrently and should
// observe ctx. They may retain SelectionName subject to its ordinary Value
// capability lifetime, but must not retain ctx after this method returns.
type AggressorProcessInjectionProvider interface {
	HandleAggressorProcessInjection(context.Context, AggressorProcessInjectionRequest) (Value, error)
}

// AggressorProcessInjectionProviderFunc adapts a function to
// AggressorProcessInjectionProvider.
type AggressorProcessInjectionProviderFunc func(context.Context, AggressorProcessInjectionRequest) (Value, error)

// HandleAggressorProcessInjection calls function.
func (function AggressorProcessInjectionProviderFunc) HandleAggressorProcessInjection(
	ctx context.Context,
	request AggressorProcessInjectionRequest,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor process-injection provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorProcessInjectionProvider installs the typed importer boundary
// for the documented pi_explicit_*, pi_spawn_*, pi_user_explicit_*, and
// pi_user_spawn_* functions. Provider errors are authoritative and never fall
// back to Host. WithFunction overrides retain precedence over the native
// wrappers.
func WithAggressorProcessInjectionProvider(provider AggressorProcessInjectionProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor process-injection provider is nil")
		}
		config.aggressorProcessInjectionProvider = provider
		return nil
	}
}
