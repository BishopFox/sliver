package cli

import (
	"github.com/bishopfox/sliver/server/daemon"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start server in daemon mode",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		daemon.Start()
	},
}
