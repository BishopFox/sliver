package hosts

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	hostsCmd := &cobra.Command{
		Use:   consts.HostsStr,
		Short: "Manage the database of hosts",
		Long:  help.GetHelpFor([]string{consts.HostsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			HostsCmd(cmd, con, args)
		},
		GroupID: consts.SliverHelpGroup,
	}
	flags.Bind("hosts", true, hostsCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	hostsRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Remove a host from the database",
		Long:  help.GetHelpFor([]string{consts.HostsStr, consts.RmStr}),
		Run: func(cmd *cobra.Command, args []string) {
			HostsRmCmd(cmd, con, args)
		},
	}
	hostsCmd.AddCommand(hostsRmCmd)

	hostsIOCCmd := &cobra.Command{
		Use:   consts.IOCStr,
		Short: "Manage tracked IOCs on a given host",
		Long:  help.GetHelpFor([]string{consts.HostsStr, consts.IOCStr}),
		Run: func(cmd *cobra.Command, args []string) {
			HostsIOCCmd(cmd, con, args)
		},
	}
	hostsCmd.AddCommand(hostsIOCCmd)

	hostsIOCRmCmd := &cobra.Command{
		Use:   consts.RmStr,
		Short: "Delete IOCs from the database",
		Long:  help.GetHelpFor([]string{consts.HostsStr, consts.IOCStr, consts.RmStr}),
		Run: func(cmd *cobra.Command, args []string) {
			HostsIOCRmCmd(cmd, con, args)
		},
	}
	hostsIOCCmd.AddCommand(hostsIOCRmCmd)

	return []*cobra.Command{hostsCmd}
}
