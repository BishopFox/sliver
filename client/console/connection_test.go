package console

import (
	"testing"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/transport"
)

func TestDedicatedCommandConnectionUsesEffectiveWireGuardMode(t *testing.T) {
	transport.SetMultiplayerConnectMode(transport.MultiplayerConnectAuto)
	t.Cleanup(func() {
		transport.SetMultiplayerConnectMode(transport.MultiplayerConnectAuto)
	})

	legacyDetails := &ConnectionDetails{Config: completeWireGuardClientConfig(false)}
	if useDedicatedCommandConnection(legacyDetails) {
		t.Fatal("expected auto mode to keep an unmarked legacy config on direct mTLS")
	}

	enabledDetails := &ConnectionDetails{Config: completeWireGuardClientConfig(true)}
	if !useDedicatedCommandConnection(enabledDetails) {
		t.Fatal("expected auto mode to use a dedicated connection for an explicitly enabled config")
	}

	transport.SetMultiplayerConnectMode(transport.MultiplayerConnectEnableWG)
	if !useDedicatedCommandConnection(legacyDetails) {
		t.Fatal("expected force-enable mode to use a dedicated connection with a complete legacy config")
	}

	transport.SetMultiplayerConnectMode(transport.MultiplayerConnectDisableWG)
	if useDedicatedCommandConnection(enabledDetails) {
		t.Fatal("expected force-disable mode to keep an enabled config on direct mTLS")
	}
}

func TestDedicatedCommandConnectionRejectsIncompleteEnabledConfig(t *testing.T) {
	transport.SetMultiplayerConnectMode(transport.MultiplayerConnectAuto)
	t.Cleanup(func() {
		transport.SetMultiplayerConnectMode(transport.MultiplayerConnectAuto)
	})

	details := &ConnectionDetails{
		Config: &assets.ClientConfig{
			WG: &assets.ClientWGConfig{Enabled: true},
		},
	}
	if useDedicatedCommandConnection(details) {
		t.Fatal("expected an incomplete enabled config not to select a dedicated connection")
	}
}

func completeWireGuardClientConfig(enabled bool) *assets.ClientConfig {
	return &assets.ClientConfig{
		WG: &assets.ClientWGConfig{
			Enabled:          enabled,
			ServerPubKey:     "server-pub",
			ClientPrivateKey: "client-priv",
			ClientIP:         "100.64.0.2",
		},
	}
}
