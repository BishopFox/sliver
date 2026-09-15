package opfor

import (
	"context"
	"errors"
)

// AggressorPayloadStoreOperation identifies one documented Payload Store
// query or mutation. String values are the exact Aggressor function names.
type AggressorPayloadStoreOperation string

const (
	// AggressorPayloadStoreAdd identifies payloadstore_add.
	AggressorPayloadStoreAdd AggressorPayloadStoreOperation = "payloadstore_add"
	// AggressorPayloadStoreFetch identifies payloadstore_fetch.
	AggressorPayloadStoreFetch AggressorPayloadStoreOperation = "payloadstore_fetch"
	// AggressorPayloadStoreList identifies payloadstore_list.
	AggressorPayloadStoreList AggressorPayloadStoreOperation = "payloadstore_list"
	// AggressorPayloadStoreMetadata identifies payloadstore_metadata.
	AggressorPayloadStoreMetadata AggressorPayloadStoreOperation = "payloadstore_metadata"
	// AggressorPayloadStoreRemove identifies payloadstore_remove.
	AggressorPayloadStoreRemove AggressorPayloadStoreOperation = "payloadstore_remove"
)

// AggressorPayloadStoreRequest is one resolved request against the importer's
// Cobalt Payload Store. RuntimeID, Script, and Span identify the call site.
//
// Arguments contains an exact positional snapshot:
//
//   - payloadstore_add: name, payload type, artifact type, architecture,
//     payload bytes, then an optional information hash
//   - payloadstore_fetch/payloadstore_metadata/payloadstore_remove: ID or name
//   - payloadstore_list: no positions
//
// The slice is detached from Invocation, while every Value is resolved once
// without coercion or cloning. Optional-position presence and compound
// identity are therefore preserved exactly.
type AggressorPayloadStoreRequest struct {
	Operation AggressorPayloadStoreOperation
	Name      string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span

	Arguments []Value
}

// Arg returns a resolved positional argument or $null when index is absent.
func (request AggressorPayloadStoreRequest) Arg(index int) Value {
	if index < 0 || index >= len(request.Arguments) {
		return Null()
	}
	return request.Arguments[index]
}

// HasArgument reports whether a positional argument was supplied.
func (request AggressorPayloadStoreRequest) HasArgument(index int) bool {
	return index >= 0 && index < len(request.Arguments)
}

// AggressorPayloadStoreProvider supplies Cobalt-owned Payload Store state.
// HandleAggressorPayloadStore is called synchronously exactly once for every
// valid invocation when configured. Errors are authoritative and never retry
// through Host because add/remove may already have changed external state.
//
// Successful add, fetch, list, and metadata results are transferred directly
// to the script without coercion, cloning, validation, or serialization.
// Remove is documented solely as an effect; OPFOR discards the provider result
// and returns $null. Entry schemas, ordering, duplicate-name behavior, missing
// entries, persistence, and freshness remain importer-owned.
//
// Implementations may be called concurrently and should observe ctx. They may
// retain request Values but must not retain ctx after the method returns.
type AggressorPayloadStoreProvider interface {
	HandleAggressorPayloadStore(context.Context, AggressorPayloadStoreRequest) (Value, error)
}

// AggressorPayloadStoreProviderFunc adapts a function to
// AggressorPayloadStoreProvider.
type AggressorPayloadStoreProviderFunc func(context.Context, AggressorPayloadStoreRequest) (Value, error)

// HandleAggressorPayloadStore calls function.
func (function AggressorPayloadStoreProviderFunc) HandleAggressorPayloadStore(
	ctx context.Context,
	request AggressorPayloadStoreRequest,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor Payload Store provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorPayloadStoreProvider installs the typed importer boundary for
// the five documented payloadstore_* functions. Provider errors are
// authoritative and importer-defined WithFunction callbacks retain precedence.
func WithAggressorPayloadStoreProvider(provider AggressorPayloadStoreProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor Payload Store provider is nil")
		}
		config.aggressorPayloadStoreProvider = provider
		return nil
	}
}
