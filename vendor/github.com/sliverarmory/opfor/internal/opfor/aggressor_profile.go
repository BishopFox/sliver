package opfor

import (
	"context"
	"errors"
)

// AggressorProfileOperation identifies one operation backed by Team Server
// configuration or the active Malleable C2 profile. String values are the
// exact Aggressor function names so they remain stable in importer logs and
// adapters.
type AggressorProfileOperation string

const (
	// AggressorProfileKillDate identifies killdate.
	AggressorProfileKillDate AggressorProfileOperation = "killdate"
	// AggressorProfileSetupStrings identifies setup_strings.
	AggressorProfileSetupStrings AggressorProfileOperation = "setup_strings"
	// AggressorProfileSetupTransformations identifies setup_transformations.
	AggressorProfileSetupTransformations AggressorProfileOperation = "setup_transformations"
)

// AggressorProfileRequest is one resolved Team Server/profile request. Name is
// the exact normalized function spelling used by the script. RuntimeID is the
// nonzero process-local identity of the originating Runtime; Script and Span
// identify the call site without exposing a *Runtime.
//
// Payload is populated for setup_strings and setup_transformations.
// Architecture is additionally populated for setup_transformations. Values
// are resolved exactly once before the provider call. Scalars are immutable,
// while compound, binary, function, and object Values retain their ordinary
// identity and provenance. A provider which retains a request therefore also
// retains capabilities reachable through those Values and should snapshot or
// detach them when that lifetime is undesirable.
type AggressorProfileRequest struct {
	Operation AggressorProfileOperation
	Name      string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span

	Payload      Value
	Architecture Value
}

// AggressorProfileProvider supplies the connected Team Server kill date and
// Malleable C2 payload transformations. OPFOR calls HandleAggressorProfile
// synchronously exactly once for each valid invocation when a provider is
// configured. Its successful returned Value is transferred directly to script
// code without coercion, validation, or cloning.
//
// A returned error rejects the invocation with $null and is authoritative:
// OPFOR never retries through Host because a provider may already have done
// work. Implementations may be called concurrently and should observe ctx. A
// provider may retain request Values subject to the capability lifetime above,
// but must not retain ctx after HandleAggressorProfile returns.
type AggressorProfileProvider interface {
	HandleAggressorProfile(context.Context, AggressorProfileRequest) (Value, error)
}

// AggressorProfileProviderFunc adapts a function to AggressorProfileProvider.
type AggressorProfileProviderFunc func(context.Context, AggressorProfileRequest) (Value, error)

// HandleAggressorProfile calls function.
func (function AggressorProfileProviderFunc) HandleAggressorProfile(
	ctx context.Context,
	request AggressorProfileRequest,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor profile provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorProfileProvider installs the typed importer boundary for
// killdate, setup_strings, and setup_transformations. WithFunction overrides
// retain precedence over the native wrappers.
func WithAggressorProfileProvider(provider AggressorProfileProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor profile provider is nil")
		}
		config.aggressorProfileProvider = provider
		return nil
	}
}
