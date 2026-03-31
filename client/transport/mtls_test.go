package transport

import (
	"errors"
	"testing"

	"github.com/bishopfox/sliver/client/assets"
)

func TestSelectMultiplayerDialStrategyLegacyConfigUsesDirectMTLS(t *testing.T) {
	setTestMultiplayerConnectMode(t, MultiplayerConnectAuto)

	strategy, err := selectMultiplayerDialStrategy(&assets.ClientConfig{})
	if err != nil {
		t.Fatalf("select dial strategy: %v", err)
	}
	if strategy != multiplayerDialDirect {
		t.Fatalf("expected direct mTLS strategy, got %v", strategy)
	}
}

func TestSelectMultiplayerDialStrategyRejectsIncompleteWGConfig(t *testing.T) {
	setTestMultiplayerConnectMode(t, MultiplayerConnectAuto)

	_, err := selectMultiplayerDialStrategy(&assets.ClientConfig{
		WG: &assets.ClientWGConfig{
			ServerPubKey: "server-pub",
			ClientIP:     "100.64.0.2",
		},
	})
	if !errors.Is(err, ErrIncompleteWireGuardConfig) {
		t.Fatalf("expected incomplete WG config error, got %v", err)
	}
}

func TestSelectMultiplayerDialStrategyUsesWireGuardWhenConfigComplete(t *testing.T) {
	setTestMultiplayerConnectMode(t, MultiplayerConnectAuto)

	strategy, err := selectMultiplayerDialStrategy(&assets.ClientConfig{
		WG: &assets.ClientWGConfig{
			ServerPubKey:     "server-pub",
			ClientPrivateKey: "client-priv",
			ClientIP:         "100.64.0.2",
		},
	})
	if err != nil {
		t.Fatalf("select dial strategy: %v", err)
	}
	if strategy != multiplayerDialWireGuard {
		t.Fatalf("expected WireGuard strategy, got %v", strategy)
	}
}

func TestSelectMultiplayerDialStrategyDisableWGOverridesCompleteConfig(t *testing.T) {
	setTestMultiplayerConnectMode(t, MultiplayerConnectDisableWG)

	strategy, err := selectMultiplayerDialStrategy(&assets.ClientConfig{
		WG: &assets.ClientWGConfig{
			ServerPubKey:     "server-pub",
			ClientPrivateKey: "client-priv",
			ClientIP:         "100.64.0.2",
		},
	})
	if err != nil {
		t.Fatalf("select dial strategy: %v", err)
	}
	if strategy != multiplayerDialDirect {
		t.Fatalf("expected direct strategy when WG is disabled, got %v", strategy)
	}
}

func setTestMultiplayerConnectMode(t *testing.T, mode MultiplayerConnectMode) {
	t.Helper()

	previous := getMultiplayerConnectMode()
	SetMultiplayerConnectMode(mode)
	t.Cleanup(func() {
		SetMultiplayerConnectMode(previous)
	})
}
