package cli

import (
	"fmt"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/spf13/cobra"
)

const (
	requireWGFlag = "require-wg"
	disableWGFlag = "disable-wg"
)

func applyMultiplayerConnectMode(cmd *cobra.Command) error {
	if cmd == nil {
		transport.SetMultiplayerConnectMode(transport.MultiplayerConnectAuto)
		return nil
	}

	requireWG, err := cmd.Flags().GetBool(requireWGFlag)
	if err != nil {
		return err
	}
	disableWG, err := cmd.Flags().GetBool(disableWGFlag)
	if err != nil {
		return err
	}
	if requireWG && disableWG {
		return fmt.Errorf("--%s and --%s cannot be used together", requireWGFlag, disableWGFlag)
	}

	mode := transport.MultiplayerConnectAuto
	switch {
	case requireWG:
		mode = transport.MultiplayerConnectRequireWG
	case disableWG:
		mode = transport.MultiplayerConnectDisableWG
	}
	transport.SetMultiplayerConnectMode(mode)
	return nil
}
