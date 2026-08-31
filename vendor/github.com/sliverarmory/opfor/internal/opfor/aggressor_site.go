package opfor

import (
	"context"
	"errors"
)

// AggressorSiteKind identifies one Team Server site-delivery operation. The
// string values are the exact Aggressor function names so they remain stable
// in importer logs and adapters.
type AggressorSiteKind string

const (
	// AggressorSiteLocalIP identifies localip().
	AggressorSiteLocalIP AggressorSiteKind = "localip"
	// AggressorSiteHost identifies site_host(...).
	AggressorSiteHost AggressorSiteKind = "site_host"
	// AggressorSiteKill identifies site_kill(...).
	AggressorSiteKill AggressorSiteKind = "site_kill"
	// AggressorSiteList identifies sites().
	AggressorSiteList AggressorSiteKind = "sites"
)

// AggressorSiteRequest is one resolved Team Server site-delivery request.
// Name is the exact normalized function spelling used by the script.
// RuntimeID is the nonzero process-local identity of the originating Runtime;
// Script and Span identify the call site without exposing a *Runtime.
// Bindings is the opaque callback capability for that exact Runtime; retaining
// it retains the Runtime but does not expose its evaluator or binding registry.
//
// Host, Port, URI, Content, MIMEType, Description, and SSL are populated only
// where the corresponding Aggressor contract supplies them. Every Value is
// resolved exactly once before the provider call. Scalars are immutable,
// while binary provenance and compound/object/function reference identity are
// retained. A provider which retains a request therefore also retains any
// capabilities reachable through those Values and should snapshot or detach
// them when that lifetime is undesirable.
//
// site_host accepts both the six-argument form used by the pinned official 3.7
// examples and the current seven-argument contract. HasSSL distinguishes an
// omitted seventh argument from an explicitly supplied $null. SSL retains the
// exact supplied Value and SSLTruth records its Sleep truth value; both are
// $null/false when the argument is omitted.
type AggressorSiteRequest struct {
	Kind AggressorSiteKind
	Name string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span
	Bindings  AggressorBindings

	Host        Value
	Port        Value
	URI         Value
	Content     Value
	MIMEType    Value
	Description Value

	SSL      Value
	HasSSL   bool
	SSLTruth bool
}

// AggressorSiteProvider supplies Team Server site-delivery queries and
// effects. HandleAggressorSite is called synchronously exactly once for each
// valid invocation when a provider is installed. LocalIP is a read-only query
// whose documented result is the Team Server IP. List is a read-only query
// whose documented result is an array of site dictionaries. Host creates or
// replaces hosted content and its documented result is the hosted URL. Kill
// removes hosted content; the current official entry has no Returns section,
// so OPFOR models it as an effect whose successful script result is $null.
//
// OPFOR transfers successful LocalIP, List, and Host provider Values directly
// to the script without coercion, validation, or cloning. A successful Kill
// provider Value is discarded and the wrapper returns $null. A returned error
// rejects the invocation with a $null result and is authoritative: OPFOR never
// retries through Host, because Host or Kill may already have performed an
// effect. Compound query/host results remain provider-owned references, so
// providers should return detached graphs when script mutation must not affect
// backing state.
//
// Implementations may be called concurrently and should observe ctx. The
// configured provider is Runtime-owned and may be shared with inherited child
// runtimes. The call itself is lifecycle-neutral: OPFOR does not retain the
// request or context beyond the synchronous call, although an importer may
// retain request Values subject to the capability lifetime described above. A
// provider must not retain ctx after HandleAggressorSite returns.
type AggressorSiteProvider interface {
	HandleAggressorSite(context.Context, AggressorSiteRequest) (Value, error)
}

// AggressorSiteProviderFunc adapts a function to AggressorSiteProvider.
type AggressorSiteProviderFunc func(context.Context, AggressorSiteRequest) (Value, error)

// HandleAggressorSite calls function.
func (function AggressorSiteProviderFunc) HandleAggressorSite(
	ctx context.Context,
	request AggressorSiteRequest,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor site provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorSiteProvider installs the typed importer boundary for localip,
// site_host, site_kill, and sites. Provider errors are authoritative and never
// fall back to Host. Importer-defined WithFunction callbacks retain precedence
// over the native wrappers.
func WithAggressorSiteProvider(provider AggressorSiteProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor site provider is nil")
		}
		config.aggressorSiteProvider = provider
		return nil
	}
}
