package opfor

import (
	"context"
	"errors"
)

// AggressorPreferenceOperation identifies one documented Cobalt preference
// query or mutation. The string values are the exact Aggressor function names
// so they remain stable in importer logs and adapters.
type AggressorPreferenceOperation string

const (
	// AggressorPreferenceGet identifies pref_get.
	AggressorPreferenceGet AggressorPreferenceOperation = "pref_get"
	// AggressorPreferenceGetList identifies pref_get_list.
	AggressorPreferenceGetList AggressorPreferenceOperation = "pref_get_list"
	// AggressorPreferenceSet identifies pref_set.
	AggressorPreferenceSet AggressorPreferenceOperation = "pref_set"
	// AggressorPreferenceSetList identifies pref_set_list.
	AggressorPreferenceSetList AggressorPreferenceOperation = "pref_set_list"
)

// AggressorPreferenceRequest is one resolved request for Cobalt's preference
// store. RuntimeID is the nonzero process-local identity of the originating
// Runtime; Script and Span identify the call site without exposing a *Runtime.
//
// PreferenceName is the exact resolved first argument. DefaultValue is the
// exact resolved second argument for pref_get and $null for the other three
// operations. PreferenceValue is the exact resolved second argument for
// pref_set and pref_set_list and $null for both query operations. No field is
// coerced or cloned. In particular, the pref_set_list array retains its normal
// reference identity, so a provider retaining the request also retains that
// mutable capability.
//
// The current public function reference specifies these shapes:
//
//   - pref_get: preference name and default value
//   - pref_get_list: preference name
//   - pref_set: preference name and preference value
//   - pref_set_list: preference name and an array of preference values
//
// The reference does not specify whether names and scalar values are coerced
// before Cobalt's store sees them. OPFOR therefore transfers their resolved
// Values unchanged. It enforces only the documented arities and the explicit
// array requirement for pref_set_list.
type AggressorPreferenceRequest struct {
	Operation AggressorPreferenceOperation

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span

	PreferenceName  Value
	DefaultValue    Value
	PreferenceValue Value
}

// AggressorPreferenceProvider supplies the importer-owned Cobalt preference
// store. HandleAggressorPreference is called synchronously exactly once for
// every valid invocation when a provider is installed. A returned error
// rejects the invocation with $null and is authoritative: OPFOR never retries
// through Host because a preference mutation may already have taken effect.
//
// For pref_get and pref_get_list, OPFOR transfers the successful returned
// Value directly to script code without coercion, validation, cloning, or
// serialization. This preserves an array returned by pref_get_list as the
// provider's exact array identity. For pref_set and pref_set_list, the public
// reference documents only a side effect; OPFOR discards the provider result
// and returns $null.
//
// Missing-key behavior for pref_get_list, preference persistence, concurrent
// update ordering, and whether returned arrays are live store views are not
// defined by the public reference and remain provider-owned. Implementations
// may be called concurrently and should observe ctx. They may retain request
// Values subject to the capability lifetime above, but must not retain ctx
// after HandleAggressorPreference returns.
type AggressorPreferenceProvider interface {
	HandleAggressorPreference(context.Context, AggressorPreferenceRequest) (Value, error)
}

// AggressorPreferenceProviderFunc adapts a function to
// AggressorPreferenceProvider.
type AggressorPreferenceProviderFunc func(context.Context, AggressorPreferenceRequest) (Value, error)

// HandleAggressorPreference calls function.
func (function AggressorPreferenceProviderFunc) HandleAggressorPreference(
	ctx context.Context,
	request AggressorPreferenceRequest,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor preference provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorPreferenceProvider installs the typed importer boundary for
// pref_get, pref_get_list, pref_set, and pref_set_list. Provider errors are
// authoritative and never fall back to Host. Importer-defined WithFunction
// callbacks retain precedence over the native wrappers.
func WithAggressorPreferenceProvider(provider AggressorPreferenceProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor preference provider is nil")
		}
		config.aggressorPreferenceProvider = provider
		return nil
	}
}
