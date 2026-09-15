package opfor

import (
	"context"
	"errors"
)

// AggressorArtifactKind identifies one documented stageless artifact
// generation function. The string values are the exact Aggressor function
// names so they remain stable in importer logs and adapters.
type AggressorArtifactKind string

const (
	// AggressorArtifactPayload identifies artifact_payload.
	AggressorArtifactPayload AggressorArtifactKind = "artifact_payload"
	// AggressorArtifactStageless identifies the deprecated
	// callback-based artifact_stageless function.
	AggressorArtifactStageless AggressorArtifactKind = "artifact_stageless"
)

// AggressorArtifactRequest is one resolved stageless artifact generation
// request. Name is the exact normalized function spelling used by the script.
// RuntimeID is the nonzero process-local identity of the originating Runtime;
// Script and Span identify the call site without exposing a *Runtime.
// Bindings is the opaque callback capability for that exact Runtime; retaining
// it retains the Runtime but does not expose its evaluator or binding registry.
//
// Listener, ArtifactType, and Architecture are populated for both Kinds.
// Payload requests also populate ExitMethod and SystemCallMethod. Each of the
// four later optional positions has a Has field which distinguishes omission
// from an explicitly supplied $null Value. Stageless requests instead populate
// ProxyConfiguration and Callback. Every Value is resolved exactly once before
// the provider call. Scalars are immutable, while compound Values retain their
// ordinary reference identity; providers which retain a request also retain
// capabilities reachable through those Values and should snapshot or detach
// them when that lifetime is undesirable.
//
// Callback is a retained, script-owned capability. A provider may invoke it
// asynchronously with the generated stageless content after
// GenerateAggressorArtifact returns. It honors the invocation context supplied
// by the provider and rejects calls after the creating Script generation
// retires, its Script unloads, or its Runtime closes.
type AggressorArtifactRequest struct {
	Kind AggressorArtifactKind
	Name string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span
	Bindings  AggressorBindings

	Listener     Value
	ArtifactType Value
	Architecture Value

	ExitMethod       Value
	SystemCallMethod Value

	HTTPLibrary    Value
	HasHTTPLibrary bool

	DNSCommMode    Value
	HasDNSCommMode bool

	MalleableProfileOverride    Value
	HasMalleableProfileOverride bool

	PayloadStoreInfo    Value
	HasPayloadStoreInfo bool

	ProxyConfiguration Value
	Callback           Callable
}

// AggressorArtifactProvider supplies Cobalt-owned stageless artifact
// generation. GenerateAggressorArtifact is called synchronously exactly once
// for each valid invocation when a provider is installed. For
// artifact_payload, OPFOR transfers the returned Value directly to the script.
// For artifact_stageless, the documented result is delivered through Callback
// and OPFOR discards the provider's returned Value, returning $null from the
// native wrapper.
//
// A returned error rejects the invocation with a $null result and is never
// retried through Host. Implementations may be called concurrently and should
// observe ctx. A provider may retain Request Values and Callback subject to the
// documented capability lifetimes, but must not retain ctx after
// GenerateAggressorArtifact returns.
type AggressorArtifactProvider interface {
	GenerateAggressorArtifact(context.Context, AggressorArtifactRequest) (Value, error)
}

// AggressorArtifactProviderFunc adapts a function to
// AggressorArtifactProvider.
type AggressorArtifactProviderFunc func(context.Context, AggressorArtifactRequest) (Value, error)

// GenerateAggressorArtifact calls function.
func (function AggressorArtifactProviderFunc) GenerateAggressorArtifact(
	ctx context.Context,
	request AggressorArtifactRequest,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor artifact provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorArtifactProvider installs the typed importer boundary for
// artifact_payload and artifact_stageless. Importer-defined WithFunction
// callbacks retain precedence over the native wrappers.
func WithAggressorArtifactProvider(provider AggressorArtifactProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor artifact provider is nil")
		}
		config.aggressorArtifactProvider = provider
		return nil
	}
}
