package cli

import (
	"testing"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/spf13/cobra"
)

func TestBuilderMultiplayerConnectMode(t *testing.T) {
	tests := []struct {
		name      string
		enableWG  bool
		disableWG bool
		want      transport.MultiplayerConnectMode
		wantErr   bool
	}{
		{name: "automatic", want: transport.MultiplayerConnectAuto},
		{name: "force wireguard", enableWG: true, want: transport.MultiplayerConnectEnableWG},
		{name: "force direct", disableWG: true, want: transport.MultiplayerConnectDisableWG},
		{name: "conflicting overrides", enableWG: true, disableWG: true, want: transport.MultiplayerConnectAuto, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool(enableWGFlagStr, false, "")
			cmd.Flags().Bool(disableWGFlagStr, false, "")
			if test.enableWG {
				if err := cmd.Flags().Set(enableWGFlagStr, "true"); err != nil {
					t.Fatalf("set --%s: %v", enableWGFlagStr, err)
				}
			}
			if test.disableWG {
				if err := cmd.Flags().Set(disableWGFlagStr, "true"); err != nil {
					t.Fatalf("set --%s: %v", disableWGFlagStr, err)
				}
			}

			mode, err := builderMultiplayerConnectMode(cmd)
			if (err != nil) != test.wantErr {
				t.Fatalf("builder multiplayer mode error = %v, wantErr %t", err, test.wantErr)
			}
			if mode != test.want {
				t.Fatalf("builder multiplayer mode = %v, want %v", mode, test.want)
			}
		})
	}
}
