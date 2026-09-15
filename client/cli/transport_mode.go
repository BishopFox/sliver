package cli

import (
	"fmt"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/spf13/cobra"
)

const (
	enableWGFlag  = "enable-wg"
	disableWGFlag = "disable-wg"
)

func applyMultiplayerConnectMode(cmd *cobra.Command) error {
	if cmd == nil {
		transport.SetMultiplayerConnectMode(transport.MultiplayerConnectAuto)
		return nil
	}

	enableWG, err := cmd.Flags().GetBool(enableWGFlag)
	if err != nil {
		return err
	}
	disableWG, err := cmd.Flags().GetBool(disableWGFlag)
	if err != nil {
		return err
	}
	if cmd.Flags().Changed(enableWGFlag) && cmd.Flags().Changed(disableWGFlag) {
		return fmt.Errorf("--%s and --%s cannot be used together", enableWGFlag, disableWGFlag)
	}

	mode := transport.MultiplayerConnectAuto
	switch {
	case enableWG:
		mode = transport.MultiplayerConnectEnableWG
	case disableWG:
		mode = transport.MultiplayerConnectDisableWG
	}
	transport.SetMultiplayerConnectMode(mode)
	return nil
}
