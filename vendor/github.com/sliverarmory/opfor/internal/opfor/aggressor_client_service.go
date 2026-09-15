package opfor

import (
	"context"
	"errors"
)

// AggressorClientServiceOperation identifies one operation owned by the
// connected Cobalt Strike client or Team Server. String values are the exact
// Aggressor function names so they remain stable in importer logs and
// adapters.
type AggressorClientServiceOperation string

const (
	// AggressorClientServiceGetAggressorClient identifies getAggressorClient.
	AggressorClientServiceGetAggressorClient AggressorClientServiceOperation = "getAggressorClient"
	// AggressorClientServiceGetCSVersion identifies get_cs_version.
	AggressorClientServiceGetCSVersion AggressorClientServiceOperation = "get_cs_version"
	// AggressorClientServiceMyNick identifies mynick.
	AggressorClientServiceMyNick AggressorClientServiceOperation = "mynick"
	// AggressorClientServiceUsers identifies users.
	AggressorClientServiceUsers AggressorClientServiceOperation = "users"
	// AggressorClientServiceAction identifies action.
	AggressorClientServiceAction AggressorClientServiceOperation = "action"
	// AggressorClientServiceEventLog identifies elog.
	AggressorClientServiceEventLog AggressorClientServiceOperation = "elog"
	// AggressorClientServiceSay identifies say.
	AggressorClientServiceSay AggressorClientServiceOperation = "say"
	// AggressorClientServicePrivateMessage identifies privmsg.
	AggressorClientServicePrivateMessage AggressorClientServiceOperation = "privmsg"
	// AggressorClientServiceCustomEvent identifies custom_event.
	AggressorClientServiceCustomEvent AggressorClientServiceOperation = "custom_event"
	// AggressorClientServiceCustomEventPrivate identifies custom_event_private.
	AggressorClientServiceCustomEventPrivate AggressorClientServiceOperation = "custom_event_private"
	// AggressorClientServiceCloseClient identifies closeClient.
	AggressorClientServiceCloseClient AggressorClientServiceOperation = "closeClient"
	// AggressorClientServiceSyncDownload identifies sync_download.
	AggressorClientServiceSyncDownload AggressorClientServiceOperation = "sync_download"
)

// AggressorClientServiceRequest is one resolved client-service request. Name
// is the exact normalized function spelling used by the script. RuntimeID is
// the nonzero process-local identity of the originating Runtime; Script and
// Span identify the call site without exposing a *Runtime.
// Bindings is the opaque callback capability for that exact Runtime; retaining
// it retains the Runtime but does not expose its evaluator or binding registry.
//
// Arguments is a detached top-level positional snapshot. Every source
// argument is resolved exactly once before the provider call. Scalar Values
// are immutable, while compound, binary, function, and object Values retain
// their ordinary identity and provenance. A provider which retains a request
// therefore also retains capabilities reachable through those Values and
// should snapshot or detach them when that lifetime is undesirable.
//
// Arguments has these exact documented shapes:
//
//   - action, elog, and say: message
//   - privmsg: recipient and message
//   - custom_event: topic and data
//   - custom_event_private: recipient, topic, and data
//   - sync_download: remote path and local destination; its optional third
//     callback argument is represented by CallbackState and Callback
//   - getAggressorClient, get_cs_version, mynick, users, and closeClient: no
//     positions
//
// Callback is non-nil only when sync_download's third position was callable.
// It is a retained, script-owned, multi-shot capability. The importer invokes
// it with the synced local path as its first positional argument. CallbackState
// distinguishes omission, explicit $null, and a callable. Invocation honors
// the supplied context and is rejected after generation retirement, Script
// unload, or Runtime close.
type AggressorClientServiceRequest struct {
	Operation AggressorClientServiceOperation
	Name      string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span
	Bindings  AggressorBindings

	Arguments     []Value
	CallbackState AggressorCallbackState
	Callback      Callable
}

// AggressorClientServiceProvider supplies client identity, connected-user,
// event-log, chat, custom-event, and connection-lifecycle behavior. OPFOR
// calls HandleAggressorClientService synchronously exactly once for each valid
// invocation when a provider is configured. A returned error rejects the
// invocation with $null and is authoritative: OPFOR never retries through
// Host because a client or Team Server effect may already have occurred.
//
// getAggressorClient, get_cs_version, mynick, and users transfer the returned
// Value directly to script code without coercion, validation, or cloning. In
// particular, getAggressorClient may return an opaque Object Value which stays
// usable through ObjectHost. The remaining operations, including
// sync_download, are documented only for their side effects; OPFOR discards
// the provider result and returns $null.
//
// Implementations may be called concurrently and should observe ctx. A
// provider may retain request Values and Callback subject to the capability
// lifetimes above, but must not retain ctx after HandleAggressorClientService
// returns.
type AggressorClientServiceProvider interface {
	HandleAggressorClientService(context.Context, AggressorClientServiceRequest) (Value, error)
}

// AggressorClientServiceProviderFunc adapts a function to
// AggressorClientServiceProvider.
type AggressorClientServiceProviderFunc func(context.Context, AggressorClientServiceRequest) (Value, error)

// HandleAggressorClientService calls function.
func (function AggressorClientServiceProviderFunc) HandleAggressorClientService(
	ctx context.Context,
	request AggressorClientServiceRequest,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor client service provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorClientServiceProvider installs the typed importer boundary for
// getAggressorClient, get_cs_version, mynick, users, action, elog, say,
// privmsg, custom_event, custom_event_private, closeClient, and sync_download.
// WithFunction overrides retain precedence over the native wrappers.
func WithAggressorClientServiceProvider(provider AggressorClientServiceProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor client service provider is nil")
		}
		config.aggressorClientServiceProvider = provider
		return nil
	}
}
