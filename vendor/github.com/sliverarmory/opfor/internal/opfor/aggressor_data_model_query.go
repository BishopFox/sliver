package opfor

import (
	"context"
	"errors"
)

// AggressorDataModelQueryKind identifies one canonical Cobalt data-model
// operation. Name on AggressorDataModelQuery retains the exact function
// spelling used by the script.
type AggressorDataModelQueryKind string

const (
	// AggressorDataModelQueryKeys identifies data_keys().
	AggressorDataModelQueryKeys AggressorDataModelQueryKind = "data_keys"
	// AggressorDataModelQueryValue identifies data_query(key).
	AggressorDataModelQueryValue AggressorDataModelQueryKind = "data_query"
	// AggressorDataModelQueryPivots identifies pivots().
	AggressorDataModelQueryPivots AggressorDataModelQueryKind = "pivots"
)

// AggressorDataModelQuery is one resolved, read-only request for Cobalt's data
// model. RuntimeID is the nonzero process-local identity of the originating
// runtime; Script and Span identify the call site. Key is $null for data_keys
// and pivots, and is the data_query argument resolved exactly once otherwise.
//
// Scalar Values are immutable. A compound Key deliberately retains its
// mutable reference identity because the public contract calls for a key but
// does not define coercion. A provider retaining the whole request therefore
// also retains any object, callable, or nested compound capabilities carried
// by Key; snapshot a scalar coercion instead when that lifetime is undesirable.
type AggressorDataModelQuery struct {
	Kind AggressorDataModelQueryKind
	Name string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span

	Key Value
}

// AggressorDataModelQueryProvider supplies the Cobalt-owned data behind
// data_keys, data_query, and pivots. QueryAggressorDataModel is called exactly
// once for each valid invocation when a provider is installed. Returning an
// error rejects the invocation with a $null result; OPFOR never retries through
// Host.
//
// OPFOR transfers the returned Value directly to the script without sorting,
// coercion, validation, or cloning. Providers must therefore return a fresh,
// detached array or hash graph whenever script mutation must not affect their
// backing data. Implementations may be called concurrently and should observe
// ctx. Unknown keys, key ordering, result freshness, and missing-data behavior
// remain provider-owned because the public Aggressor reference does not define
// them.
type AggressorDataModelQueryProvider interface {
	QueryAggressorDataModel(context.Context, AggressorDataModelQuery) (Value, error)
}

// AggressorDataModelQueryProviderFunc adapts a function to
// AggressorDataModelQueryProvider.
type AggressorDataModelQueryProviderFunc func(context.Context, AggressorDataModelQuery) (Value, error)

// QueryAggressorDataModel calls function.
func (function AggressorDataModelQueryProviderFunc) QueryAggressorDataModel(
	ctx context.Context,
	query AggressorDataModelQuery,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor data-model query provider is nil")
	}
	return function(ctx, query)
}

// WithAggressorDataModelQueryProvider installs the read-only importer boundary
// for data_keys, data_query, and pivots. Importer-defined WithFunction
// callbacks retain precedence over the native wrappers.
func WithAggressorDataModelQueryProvider(provider AggressorDataModelQueryProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor data-model query provider is nil")
		}
		config.aggressorDataModelQueryProvider = provider
		return nil
	}
}
