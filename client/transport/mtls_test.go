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

func TestSelectMultiplayerDialStrategy(t *testing.T) {
	for _, test := range []struct {
		name        string
		mode        MultiplayerConnectMode
		config      *assets.ClientConfig
		want        multiplayerDialStrategy
		wantErr     error
		wantErrText string
	}{
		{
			name:   "auto without WireGuard config",
			mode:   MultiplayerConnectAuto,
			config: &assets.ClientConfig{},
			want:   multiplayerDialDirect,
		},
		{
			name:   "auto with legacy WireGuard config",
			mode:   MultiplayerConnectAuto,
			config: completeWGClientConfig(false),
			want:   multiplayerDialDirect,
		},
		{
			name: "auto with incomplete enabled config",
			mode: MultiplayerConnectAuto,
			config: &assets.ClientConfig{WG: &assets.ClientWGConfig{
				Enabled: true, ServerPubKey: "server-pub", ClientIP: "100.64.0.2",
			}},
			want:    multiplayerDialDirect,
			wantErr: ErrIncompleteWireGuardConfig,
		},
		{
			name:   "auto with enabled WireGuard config",
			mode:   MultiplayerConnectAuto,
			config: completeWGClientConfig(true),
			want:   multiplayerDialWireGuard,
		},
		{
			name:    "enable with missing WireGuard config",
			mode:    MultiplayerConnectEnableWG,
			config:  &assets.ClientConfig{},
			want:    multiplayerDialDirect,
			wantErr: ErrMissingWireGuardConfig,
		},
		{
			name: "enable with incomplete WireGuard config",
			mode: MultiplayerConnectEnableWG,
			config: &assets.ClientConfig{WG: &assets.ClientWGConfig{
				ServerPubKey: "server-pub", ClientIP: "100.64.0.2",
			}},
			want:    multiplayerDialDirect,
			wantErr: ErrIncompleteWireGuardConfig,
		},
		{
			name:   "enable with complete legacy config",
			mode:   MultiplayerConnectEnableWG,
			config: completeWGClientConfig(false),
			want:   multiplayerDialWireGuard,
		},
		{
			name:   "disable overrides enabled config",
			mode:   MultiplayerConnectDisableWG,
			config: completeWGClientConfig(true),
			want:   multiplayerDialDirect,
		},
		{
			name:        "invalid mode",
			mode:        MultiplayerConnectMode(255),
			config:      completeWGClientConfig(true),
			want:        multiplayerDialDirect,
			wantErrText: "invalid multiplayer connect mode 255",
		},
		{
			name:        "nil config",
			mode:        MultiplayerConnectAuto,
			want:        multiplayerDialDirect,
			wantErrText: "client config is required",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			setTestMultiplayerConnectMode(t, test.mode)
			strategy, err := selectMultiplayerDialStrategy(test.config)
			if strategy != test.want {
				t.Errorf("dial strategy = %v, want %v", strategy, test.want)
			}
			switch {
			case test.wantErr != nil:
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("select dial strategy error = %v, want %v", err, test.wantErr)
				}
			case test.wantErrText != "":
				if err == nil || err.Error() != test.wantErrText {
					t.Fatalf("select dial strategy error = %v, want %q", err, test.wantErrText)
				}
			case err != nil:
				t.Fatalf("select dial strategy: %v", err)
			}
		})
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
