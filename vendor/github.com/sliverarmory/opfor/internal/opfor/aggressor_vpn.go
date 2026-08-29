package opfor

import (
	"context"
	"errors"
)

// AggressorVPNOperation identifies one Covert VPN interface operation owned by
// the connected Team Server. String values are the exact Aggressor function
// names so they remain stable in importer logs and adapters.
type AggressorVPNOperation string

const (
	// AggressorVPNInterfaceInfo identifies vpn_interface_info.
	AggressorVPNInterfaceInfo AggressorVPNOperation = "vpn_interface_info"
	// AggressorVPNInterfaces identifies vpn_interfaces.
	AggressorVPNInterfaces AggressorVPNOperation = "vpn_interfaces"
	// AggressorVPNTAPCreate identifies vpn_tap_create.
	AggressorVPNTAPCreate AggressorVPNOperation = "vpn_tap_create"
	// AggressorVPNTAPDelete identifies vpn_tap_delete.
	AggressorVPNTAPDelete AggressorVPNOperation = "vpn_tap_delete"
)

// AggressorVPNRequest is one resolved Covert VPN request. Name is the exact
// normalized function spelling used by the script. RuntimeID is the nonzero
// process-local identity of the originating Runtime; Script and Span identify
// the call site without exposing a *Runtime.
//
// Interface is populated for interface-info, create, and delete requests. An
// interface-info request optionally populates Key and HasKey. A create request
// additionally populates MACAddress, Reserved, Port, and Channel. Values are
// resolved exactly once. Scalars are immutable, while compound, object,
// binary, and function Values retain their ordinary identity and provenance.
// Providers which retain a request therefore retain any capabilities reachable
// through those Values and should snapshot or detach them when appropriate.
type AggressorVPNRequest struct {
	Operation AggressorVPNOperation
	Name      string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span

	Interface Value
	Key       Value
	HasKey    bool

	MACAddress Value
	Reserved   Value
	Port       Value
	Channel    Value
}

// AggressorVPNProvider supplies Team Server Covert VPN interface inventory,
// metadata, creation, and deletion. OPFOR calls HandleAggressorVPN
// synchronously exactly once for each valid invocation when a provider is
// configured. vpn_interface_info and vpn_interfaces transfer the returned
// Value directly to script code; vpn_tap_create and vpn_tap_delete discard the
// result and return $null because their documented contract is side-effect
// only.
//
// A returned error rejects the invocation with $null and is authoritative:
// OPFOR never retries through Host after a possible Team Server effect.
// Implementations may be called concurrently and should observe ctx. A
// provider may retain request Values subject to the capability lifetime above,
// but must not retain ctx after HandleAggressorVPN returns.
type AggressorVPNProvider interface {
	HandleAggressorVPN(context.Context, AggressorVPNRequest) (Value, error)
}

// AggressorVPNProviderFunc adapts a function to AggressorVPNProvider.
type AggressorVPNProviderFunc func(context.Context, AggressorVPNRequest) (Value, error)

// HandleAggressorVPN calls function.
func (function AggressorVPNProviderFunc) HandleAggressorVPN(
	ctx context.Context,
	request AggressorVPNRequest,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor VPN provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorVPNProvider installs the typed importer boundary for
// vpn_interface_info, vpn_interfaces, vpn_tap_create, and vpn_tap_delete.
// WithFunction overrides retain precedence over the native wrappers.
func WithAggressorVPNProvider(provider AggressorVPNProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor VPN provider is nil")
		}
		config.aggressorVPNProvider = provider
		return nil
	}
}
