package sessions

import (
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
)

// BackgroundCmd - Background the active session
func BackgroundCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	con.ActiveTarget.Background()
	con.PrintInfof("Background ...\n")
}
