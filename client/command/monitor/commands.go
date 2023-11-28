package monitor

import (
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	monitorCmd := &cobra.Command{
		Use:     consts.MonitorStr,
		Short:   "Monitor threat intel platforms for Sliver implants",
		GroupID: consts.SliverHelpGroup,
	}

	// Monitory run management
	monitorCmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Start the monitoring loops",
		Run: func(cmd *cobra.Command, args []string) {
			MonitorStartCmd(cmd, con, args)
		},
	})
	monitorCmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop the monitoring loops",
		Run: func(cmd *cobra.Command, args []string) {
			MonitorStopCmd(cmd, con, args)
		},
	})

	// Configuration management
	configCmd := &cobra.Command{
		Use:   consts.MonitorConfigStr,
		Short: "Configure monitor API keys",
		Run: func(cmd *cobra.Command, args []string) {
			MonitorConfigCmd(cmd, con, args)
		},
	}
	monitorCmd.AddCommand(configCmd)

	configCmd.AddCommand(&cobra.Command{
		Use:   "add",
		Short: "Add API key configuration",
		Run: func(cmd *cobra.Command, args []string) {
			MonitorAddConfigCmd(cmd, con, args)
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "del",
		Short: "Remove API key configuration",
		Run: func(cmd *cobra.Command, args []string) {
			MonitorDelConfigCmd(cmd, con, args)
		},
	})

	return []*cobra.Command{monitorCmd}
}
