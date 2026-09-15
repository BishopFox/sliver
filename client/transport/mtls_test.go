package transport

import (
	"errors"
	"testing"

	"github.com/bishopfox/sliver/client/assets"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

func TestMultiplayerConnectModeDefaultsToAuto(t *testing.T) {
	if mode := getMultiplayerConnectMode(); mode != MultiplayerConnectAuto {
		t.Fatalf("expected auto to be the default multiplayer mode, got %v", mode)
	}
}

func TestSelectMultiplayerDialStrategyAutoWithoutWGConfigUsesDirectMTLS(t *testing.T) {
	setTestMultiplayerConnectMode(t, MultiplayerConnectAuto)

	strategy, err := selectMultiplayerDialStrategy(&assets.ClientConfig{})
	if err != nil {
		t.Fatalf("select dial strategy: %v", err)
	}
	if strategy != multiplayerDialDirect {
		t.Fatalf("expected direct mTLS strategy, got %v", strategy)
	}
}

func TestSelectMultiplayerDialStrategyAutoLegacyWGConfigUsesDirectMTLS(t *testing.T) {
	setTestMultiplayerConnectMode(t, MultiplayerConnectAuto)

	strategy, err := selectMultiplayerDialStrategy(completeWGClientConfig(false))
	if err != nil {
		t.Fatalf("select dial strategy: %v", err)
	}
	if strategy != multiplayerDialDirect {
		t.Fatalf("expected direct mTLS strategy, got %v", strategy)
	}
}

func TestSelectMultiplayerDialStrategyAutoEnabledWGConfigRequiresCompleteConfig(t *testing.T) {
	setTestMultiplayerConnectMode(t, MultiplayerConnectAuto)

	_, err := selectMultiplayerDialStrategy(&assets.ClientConfig{
		WG: &assets.ClientWGConfig{
			Enabled:      true,
			ServerPubKey: "server-pub",
			ClientIP:     "100.64.0.2",
		},
	})
	if !errors.Is(err, ErrIncompleteWireGuardConfig) {
		t.Fatalf("expected incomplete WG config error, got %v", err)
	}
}

func TestSelectMultiplayerDialStrategyAutoEnabledWGConfigUsesWireGuard(t *testing.T) {
	setTestMultiplayerConnectMode(t, MultiplayerConnectAuto)

	strategy, err := selectMultiplayerDialStrategy(completeWGClientConfig(true))
	if err != nil {
		t.Fatalf("select dial strategy: %v", err)
	}
	if strategy != multiplayerDialWireGuard {
		t.Fatalf("expected WireGuard strategy, got %v", strategy)
	}
}

func TestSelectMultiplayerDialStrategyEnableWGRejectsMissingWGConfig(t *testing.T) {
	setTestMultiplayerConnectMode(t, MultiplayerConnectEnableWG)

	_, err := selectMultiplayerDialStrategy(&assets.ClientConfig{})
	if !errors.Is(err, ErrMissingWireGuardConfig) {
		t.Fatalf("expected missing WG config error, got %v", err)
	}
}

func TestSelectMultiplayerDialStrategyEnableWGRejectsIncompleteWGConfig(t *testing.T) {
	setTestMultiplayerConnectMode(t, MultiplayerConnectEnableWG)

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

func TestSelectMultiplayerDialStrategyEnableWGUsesCompleteLegacyConfig(t *testing.T) {
	setTestMultiplayerConnectMode(t, MultiplayerConnectEnableWG)

	strategy, err := selectMultiplayerDialStrategy(completeWGClientConfig(false))
	if err != nil {
		t.Fatalf("select dial strategy: %v", err)
	}
	if strategy != multiplayerDialWireGuard {
		t.Fatalf("expected WireGuard strategy, got %v", strategy)
	}
}

func TestSelectMultiplayerDialStrategyDisableWGOverridesEnabledConfig(t *testing.T) {
	setTestMultiplayerConnectMode(t, MultiplayerConnectDisableWG)

	strategy, err := selectMultiplayerDialStrategy(completeWGClientConfig(true))
	if err != nil {
		t.Fatalf("select dial strategy: %v", err)
	}
	if strategy != multiplayerDialDirect {
		t.Fatalf("expected direct strategy, got %v", strategy)
	}
}

func TestSelectMultiplayerDialStrategyRejectsInvalidMode(t *testing.T) {
	setTestMultiplayerConnectMode(t, MultiplayerConnectMode(255))

	strategy, err := selectMultiplayerDialStrategy(completeWGClientConfig(true))
	if strategy != multiplayerDialDirect {
		t.Fatalf("expected invalid mode to fall back to direct strategy, got %v", strategy)
	}
	if err == nil || err.Error() != "invalid multiplayer connect mode 255" {
		t.Fatalf("expected invalid mode error, got %v", err)
	}
}

func completeWGClientConfig(enabled bool) *assets.ClientConfig {
	return &assets.ClientConfig{
		WG: &assets.ClientWGConfig{
			Enabled:          enabled,
			ServerPubKey:     "server-pub",
			ClientPrivateKey: "client-priv",
			ClientIP:         "100.64.0.2",
		},
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

func TestCloseGRPCConnectionClosesConnBeforeTransportCloser(t *testing.T) {
	conn, err := grpc.NewClient("passthrough:///sliver-test", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("create grpc client: %v", err)
	}

	var stateSeen connectivity.State
	registerConnCloser(conn, testConnectionCloser(func() error {
		stateSeen = conn.GetState()
		return nil
	}))

	if err := CloseGRPCConnection(conn); err != nil {
		t.Fatalf("close grpc connection: %v", err)
	}
	if stateSeen != connectivity.Shutdown {
		t.Fatalf("expected transport closer to run after grpc shutdown, got state %v", stateSeen)
	}
}

type testConnectionCloser func() error

func (fn testConnectionCloser) Close() error {
	return fn()
}
