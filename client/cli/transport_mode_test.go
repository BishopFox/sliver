package cli

import (
	"strings"
	"testing"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/spf13/cobra"
)

func TestWireGuardOverrideFlagsDefaultToFalse(t *testing.T) {
	for _, name := range []string{enableWGFlag, disableWGFlag} {
		flag := rootCmd.PersistentFlags().Lookup(name)
		if flag == nil {
			t.Fatalf("expected --%s persistent flag", name)
		}
		if flag.DefValue != "false" {
			t.Fatalf("expected --%s to default to false, got %q", name, flag.DefValue)
		}
	}
}

func TestApplyMultiplayerConnectModeDefaultsToAuto(t *testing.T) {
	resetMultiplayerConnectModeForTest(t)
	cmd := newTransportModeTestCommand()

	if err := applyMultiplayerConnectMode(cmd); err != nil {
		t.Fatalf("apply multiplayer mode: %v", err)
	}
	if !transport.MultiplayerConnectUsesWireGuard(enabledWGClientConfig()) {
		t.Fatal("expected automatic mode to honor an explicitly enabled WireGuard config")
	}
	if transport.MultiplayerConnectUsesWireGuard(legacyWGClientConfig()) {
		t.Fatal("expected automatic mode to ignore a legacy WireGuard config without the opt-in marker")
	}
}

func TestApplyMultiplayerConnectModeEnableWGForcesLegacyConfig(t *testing.T) {
	resetMultiplayerConnectModeForTest(t)
	cmd := newTransportModeTestCommand()
	if err := cmd.Flags().Set(enableWGFlag, "true"); err != nil {
		t.Fatalf("set --%s: %v", enableWGFlag, err)
	}

	if err := applyMultiplayerConnectMode(cmd); err != nil {
		t.Fatalf("apply multiplayer mode: %v", err)
	}
	if !transport.MultiplayerConnectUsesWireGuard(legacyWGClientConfig()) {
		t.Fatal("expected --enable-wg to force a complete legacy config through WireGuard")
	}
}

func TestApplyMultiplayerConnectModeDisableWGOverridesEnabledConfig(t *testing.T) {
	resetMultiplayerConnectModeForTest(t)
	cmd := newTransportModeTestCommand()
	if err := cmd.Flags().Set(disableWGFlag, "true"); err != nil {
		t.Fatalf("set --%s: %v", disableWGFlag, err)
	}

	if err := applyMultiplayerConnectMode(cmd); err != nil {
		t.Fatalf("apply multiplayer mode: %v", err)
	}
	if transport.MultiplayerConnectUsesWireGuard(enabledWGClientConfig()) {
		t.Fatal("expected --disable-wg to force direct mTLS")
	}
}

func TestApplyMultiplayerConnectModeRejectsConflictingOverrides(t *testing.T) {
	resetMultiplayerConnectModeForTest(t)
	cmd := newTransportModeTestCommand()
	if err := cmd.Flags().Set(enableWGFlag, "true"); err != nil {
		t.Fatalf("set --%s: %v", enableWGFlag, err)
	}
	if err := cmd.Flags().Set(disableWGFlag, "true"); err != nil {
		t.Fatalf("set --%s: %v", disableWGFlag, err)
	}

	err := applyMultiplayerConnectMode(cmd)
	if err == nil {
		t.Fatal("expected conflicting WireGuard overrides to fail")
	}
	if !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("unexpected conflict error: %v", err)
	}
}

func TestApplyMultiplayerConnectModeRejectsBothOverridesWhenExplicitlyFalse(t *testing.T) {
	resetMultiplayerConnectModeForTest(t)
	cmd := newTransportModeTestCommand()
	if err := cmd.Flags().Set(enableWGFlag, "false"); err != nil {
		t.Fatalf("set --%s: %v", enableWGFlag, err)
	}
	if err := cmd.Flags().Set(disableWGFlag, "false"); err != nil {
		t.Fatalf("set --%s: %v", disableWGFlag, err)
	}

	err := applyMultiplayerConnectMode(cmd)
	if err == nil {
		t.Fatal("expected explicitly combining both WireGuard override flags to fail")
	}
	if !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("unexpected conflict error: %v", err)
	}
}

func newTransportModeTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool(enableWGFlag, false, "")
	cmd.Flags().Bool(disableWGFlag, false, "")
	return cmd
}

func resetMultiplayerConnectModeForTest(t *testing.T) {
	t.Helper()
	transport.SetMultiplayerConnectMode(transport.MultiplayerConnectAuto)
	t.Cleanup(func() {
		transport.SetMultiplayerConnectMode(transport.MultiplayerConnectAuto)
	})
}

func enabledWGClientConfig() *assets.ClientConfig {
	config := legacyWGClientConfig()
	config.WG.Enabled = true
	return config
}

func legacyWGClientConfig() *assets.ClientConfig {
	return &assets.ClientConfig{
		WG: &assets.ClientWGConfig{
			ServerPubKey:     "server-pub",
			ClientPrivateKey: "client-priv",
			ClientIP:         "100.64.0.2",
		},
	}
}
