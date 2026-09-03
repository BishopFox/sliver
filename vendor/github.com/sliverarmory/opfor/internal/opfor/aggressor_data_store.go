package opfor

import (
	"context"
	"errors"
)

// AggressorDataStoreOperation identifies one documented Cobalt data-store
// query or mutation. The string values are the exact Aggressor function names
// so they remain stable in importer logs and adapters.
type AggressorDataStoreOperation string

const (
	// AggressorDataStoreCredentialAdd identifies credential_add.
	AggressorDataStoreCredentialAdd AggressorDataStoreOperation = "credential_add"
	// AggressorDataStoreCredentials identifies credentials.
	AggressorDataStoreCredentials AggressorDataStoreOperation = "credentials"
	// AggressorDataStoreTokenToEmail identifies tokenToEmail.
	AggressorDataStoreTokenToEmail AggressorDataStoreOperation = "tokenToEmail"
	// AggressorDataStoreApplications identifies applications.
	AggressorDataStoreApplications AggressorDataStoreOperation = "applications"
	// AggressorDataStoreArchives identifies archives.
	AggressorDataStoreArchives AggressorDataStoreOperation = "archives"
	// AggressorDataStoreDownloads identifies downloads.
	AggressorDataStoreDownloads AggressorDataStoreOperation = "downloads"
	// AggressorDataStoreHighlight identifies highlight.
	AggressorDataStoreHighlight AggressorDataStoreOperation = "highlight"
	// AggressorDataStoreKeystrokes identifies keystrokes.
	AggressorDataStoreKeystrokes AggressorDataStoreOperation = "keystrokes"
	// AggressorDataStoreScreenshots identifies screenshots.
	AggressorDataStoreScreenshots AggressorDataStoreOperation = "screenshots"
	// AggressorDataStoreServices identifies services.
	AggressorDataStoreServices AggressorDataStoreOperation = "services"
	// AggressorDataStoreTargets identifies targets.
	AggressorDataStoreTargets AggressorDataStoreOperation = "targets"
	// AggressorDataStoreHosts identifies hosts.
	AggressorDataStoreHosts AggressorDataStoreOperation = "hosts"
	// AggressorDataStoreHostInfo identifies host_info.
	AggressorDataStoreHostInfo AggressorDataStoreOperation = "host_info"
	// AggressorDataStoreHostUpdate identifies host_update.
	AggressorDataStoreHostUpdate AggressorDataStoreOperation = "host_update"
	// AggressorDataStoreHostDelete identifies host_delete.
	AggressorDataStoreHostDelete AggressorDataStoreOperation = "host_delete"
	// AggressorDataStoreResetData identifies resetData.
	AggressorDataStoreResetData AggressorDataStoreOperation = "resetData"
	// AggressorDataStoreRedactObject identifies redactobject.
	AggressorDataStoreRedactObject AggressorDataStoreOperation = "redactobject"
)

// AggressorDataStoreRequest is one resolved request for Cobalt's application
// data stores. Name is the exact normalized Aggressor function spelling used
// by the script. RuntimeID is the nonzero process-local identity of the
// originating Runtime; Script and Span identify the call site without
// exposing a *Runtime.
//
// Arguments contains an exact positional snapshot, resolved once before the
// provider call. Its documented shapes are:
//
//   - credential_add: username, secret, then optional realm, source, host,
//     secret type, and notes (two through seven positions)
//   - tokenToEmail: token
//   - highlight: model, rows array, and accent
//   - host_info: host and an optional key
//   - host_update: host, DNS name, OS, version, and an optional note
//   - host_delete: one host or array of hosts
//   - redactobject: one post-exploitation object ID
//   - credentials, applications, archives, downloads, keystrokes,
//     screenshots, services, targets, hosts, and resetData: no positions
//
// The length of Arguments distinguishes omission from an explicitly supplied
// $null. The slice itself is detached from Invocation, but each Value is
// transferred without coercion or cloning. Compound and object Values retain
// their ordinary identity, so a provider retaining this request also retains
// any capabilities reachable through those Values.
type AggressorDataStoreRequest struct {
	Operation AggressorDataStoreOperation
	Name      string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span

	Arguments []Value
}

// Arg returns a resolved positional argument or $null when index is absent.
// Use HasArgument when omission must be distinguished from an explicit $null.
func (request AggressorDataStoreRequest) Arg(index int) Value {
	if index < 0 || index >= len(request.Arguments) {
		return Null()
	}
	return request.Arguments[index]
}

// HasArgument reports whether a positional argument was supplied.
func (request AggressorDataStoreRequest) HasArgument(index int) bool {
	return index >= 0 && index < len(request.Arguments)
}

// AggressorDataStoreProvider supplies the Cobalt-owned records behind the
// documented data-store queries and mutations. HandleAggressorDataStore is
// called synchronously exactly once for every valid invocation when a provider
// is installed. A returned error rejects the invocation with a $null result
// and is authoritative: OPFOR never retries through Host because a mutation
// may already have taken effect.
//
// OPFOR transfers successful provider Values directly to the script without
// coercion, validation, cloning, or serialization, except for RedactObject.
// The documented query results are arrays of model dictionaries, except
// tokenToEmail (an email string or the literal "unknown") and host_info (a host
// dictionary or one selected value). The current redactobject entry documents
// a removal effect and no Returns section, so the typed route discards its
// successful provider Value and returns $null. The public reference does not
// fully specify record schemas, ordering, freshness, missing-record behavior,
// or other mutation return values; those remain provider-owned. Providers
// should return detached graphs when script mutation must not affect backing
// state.
//
// Implementations may be called concurrently and should observe ctx. They may
// retain request Values subject to the capability lifetime above, but must not
// retain ctx after HandleAggressorDataStore returns.
type AggressorDataStoreProvider interface {
	HandleAggressorDataStore(context.Context, AggressorDataStoreRequest) (Value, error)
}

// AggressorDataStoreProviderFunc adapts a function to
// AggressorDataStoreProvider.
type AggressorDataStoreProviderFunc func(context.Context, AggressorDataStoreRequest) (Value, error)

// HandleAggressorDataStore calls function.
func (function AggressorDataStoreProviderFunc) HandleAggressorDataStore(
	ctx context.Context,
	request AggressorDataStoreRequest,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor data-store provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorDataStoreProvider installs the typed importer boundary for the
// documented Cobalt application data-store functions. Provider errors are
// authoritative and never fall back to Host. Importer-defined WithFunction
// callbacks retain precedence over the native wrappers.
func WithAggressorDataStoreProvider(provider AggressorDataStoreProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor data-store provider is nil")
		}
		config.aggressorDataStoreProvider = provider
		return nil
	}
}
