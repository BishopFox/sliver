package opfor

import (
	"context"
	"errors"
)

// AggressorSessionQueryKind identifies one canonical Beacon/session metadata
// query. Legacy and current aliases share a Kind while Name retains the exact
// Aggressor function spelling used by the script.
type AggressorSessionQueryKind string

const (
	// AggressorSessionQueryBeacons identifies beacons().
	AggressorSessionQueryBeacons AggressorSessionQueryKind = "beacons"
	// AggressorSessionQueryBeaconIDs identifies beacon_ids().
	AggressorSessionQueryBeaconIDs AggressorSessionQueryKind = "beacon_ids"
	// AggressorSessionQueryBeaconData identifies bdata() and beacon_data().
	AggressorSessionQueryBeaconData AggressorSessionQueryKind = "beacon_data"
	// AggressorSessionQueryBeaconInfo identifies binfo() and beacon_info().
	AggressorSessionQueryBeaconInfo AggressorSessionQueryKind = "beacon_info"
	// AggressorSessionQueryBeaconArchitecture identifies barch().
	AggressorSessionQueryBeaconArchitecture AggressorSessionQueryKind = "barch"
	// AggressorSessionQueryIs64 identifies -is64.
	AggressorSessionQueryIs64 AggressorSessionQueryKind = "-is64"
	// AggressorSessionQueryIsActive identifies -isactive.
	AggressorSessionQueryIsActive AggressorSessionQueryKind = "-isactive"
	// AggressorSessionQueryIsAdmin identifies -isadmin.
	AggressorSessionQueryIsAdmin AggressorSessionQueryKind = "-isadmin"
	// AggressorSessionQueryIsBeacon identifies -isbeacon.
	AggressorSessionQueryIsBeacon AggressorSessionQueryKind = "-isbeacon"
	// AggressorSessionQueryIsSSH identifies -isssh.
	AggressorSessionQueryIsSSH AggressorSessionQueryKind = "-isssh"
)

// AggressorSessionQuery is one resolved read-only Beacon/session metadata
// request. Name is the exact normalized Aggressor name, including whether a
// script used bdata/beacon_data or binfo/beacon_info. Kind is the canonical
// operation shared by aliases.
//
// RuntimeID is the nonzero process-local identity of the originating runtime;
// that field alone does not expose or retain a *Runtime. Script and Span
// identify the call site.
// SessionID and Key are resolved Value snapshots observed exactly once before
// the provider is called. They are $null for operations whose documented ABI
// omits them. Scalars are immutable, while compound Values deliberately retain
// their mutable reference identity; the documented metadata functions accept
// scalar IDs and do not cause OPFOR to expand an array-valued SessionID. A
// provider which retains the whole query also retains any capability-bearing
// function, object, or nested compound graph supplied in those Value fields;
// snapshot their scalar coercions instead when that lifetime is undesirable.
type AggressorSessionQuery struct {
	Kind AggressorSessionQueryKind
	Name string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span

	SessionID Value
	Key       Value
}

// AggressorSessionQueryProvider supplies the Cobalt-owned data for the native
// Beacon/session metadata wrappers. QueryAggressorSession is called once for a
// successful wrapper invocation. Returning an error rejects that invocation
// with a $null result; OPFOR never retries it through Host.
//
// The documented return shapes are an array for beacons and beacon_ids, a hash
// for bdata/beacon_data, and a string for binfo/beacon_info and barch. OPFOR
// returns those provider Values without coercion or validation, except that it
// normalizes the five predicate results through Sleep truthiness and maps a
// null or empty barch result to the documented x86 fallback. A provider owns
// compatibility for missing IDs/keys and other Cobalt-specific edge cases.
//
// A compound Value returned by the provider is transferred directly to the
// script and may be mutated or retained there. Providers must therefore return
// a fresh, detached array/hash graph whenever their backing data must remain
// isolated. Implementations may be called concurrently and should observe ctx.
type AggressorSessionQueryProvider interface {
	QueryAggressorSession(context.Context, AggressorSessionQuery) (Value, error)
}

// AggressorSessionQueryProviderFunc adapts a function to
// AggressorSessionQueryProvider.
type AggressorSessionQueryProviderFunc func(context.Context, AggressorSessionQuery) (Value, error)

// QueryAggressorSession calls function.
func (function AggressorSessionQueryProviderFunc) QueryAggressorSession(
	ctx context.Context,
	query AggressorSessionQuery,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor session query provider is nil")
	}
	return function(ctx, query)
}

// WithAggressorSessionQueryProvider installs the read-only importer boundary
// for beacons, beacon_ids, bdata/beacon_data, binfo/beacon_info, barch, and the
// five documented session predicates. Importer-defined WithFunction callbacks
// retain precedence over the native wrappers.
func WithAggressorSessionQueryProvider(provider AggressorSessionQueryProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor session query provider is nil")
		}
		config.aggressorSessionQueryProvider = provider
		return nil
	}
}
