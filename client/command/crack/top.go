package crack

import (
	cracktop "github.com/bishopfox/sliver/client/command/crack/top"
	"github.com/bishopfox/sliver/client/console"
	"github.com/spf13/cobra"
)

// CrackTopCmd launches the full-screen real-time crack cluster monitor.
// The implementation lives in the top subpackage; this facade preserves the
// existing command-handler API for callers of client/command/crack.
//
//nolint:revive // Preserve the established exported command-handler API.
func CrackTopCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	cracktop.CrackTopCmd(cmd, con, args)
}
