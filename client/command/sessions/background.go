package sessions

import (
	"github.com/bishopfox/sliver/client/console"
	"github.com/spf13/cobra"
)

// BackgroundCmd - Background the active session.
// BackgroundCmd - Background 活跃 session.
func BackgroundCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	con.ActiveTarget.Background()
	con.PrintInfof("Background ...\n")
}
