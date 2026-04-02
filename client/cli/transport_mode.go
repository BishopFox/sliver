package cli

import (
	"github.com/bishopfox/sliver/client/transport"
	"github.com/spf13/cobra"
)

const (
	disableWGFlag = "disable-wg"
)

func applyMultiplayerConnectMode(cmd *cobra.Command) error {
	if cmd == nil {
		transport.SetMultiplayerConnectMode(transport.MultiplayerConnectAuto)
		return nil
	}

	disableWG, err := cmd.Flags().GetBool(disableWGFlag)
	if err != nil {
		return err
	}

	mode := transport.MultiplayerConnectAuto
	if disableWG {
		mode = transport.MultiplayerConnectDisableWG
	}
	transport.SetMultiplayerConnectMode(mode)
	return nil
}
