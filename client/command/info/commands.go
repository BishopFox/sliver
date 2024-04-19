package info

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/command/use"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	infoCmd := &cobra.Command{
		Use:   consts.InfoStr,
		Short: "Get info about session",
		Long:  help.GetHelpFor([]string{consts.InfoStr}),
		Run: func(cmd *cobra.Command, args []string) {
			InfoCmd(cmd, con, args)
		},
		GroupID: consts.SliverHelpGroup,
	}
	flags.Bind("use", false, infoCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	carapace.Gen(infoCmd).PositionalCompletion(use.BeaconAndSessionIDCompleter(con))

	return []*cobra.Command{infoCmd}
}

// SliverCommands returns all info commands working on an active target.
func SliverCommands(con *console.SliverClient) []*cobra.Command {
	pingCmd := &cobra.Command{
		Use:   consts.PingStr,
		Short: "Send round trip message to implant (does not use ICMP)",
		Long:  help.GetHelpFor([]string{consts.PingStr}),
		Run: func(cmd *cobra.Command, args []string) {
			PingCmd(cmd, con, args)
		},
		GroupID: consts.InfoHelpGroup,
	}
	flags.Bind("", false, pingCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	getPIDCmd := &cobra.Command{
		Use:   consts.GetPIDStr,
		Short: "Get session pid",
		Long:  help.GetHelpFor([]string{consts.GetPIDStr}),
		Run: func(cmd *cobra.Command, args []string) {
			PIDCmd(cmd, con, args)
		},
		GroupID: consts.InfoHelpGroup,
	}
	flags.Bind("", false, getPIDCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	getUIDCmd := &cobra.Command{
		Use:   consts.GetUIDStr,
		Short: "Get session process UID",
		Long:  help.GetHelpFor([]string{consts.GetUIDStr}),
		Run: func(cmd *cobra.Command, args []string) {
			UIDCmd(cmd, con, args)
		},
		GroupID: consts.InfoHelpGroup,
	}
	flags.Bind("", false, getUIDCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	getGIDCmd := &cobra.Command{
		Use:   consts.GetGIDStr,
		Short: "Get session process GID",
		Long:  help.GetHelpFor([]string{consts.GetGIDStr}),
		Run: func(cmd *cobra.Command, args []string) {
			GIDCmd(cmd, con, args)
		},
		GroupID: consts.InfoHelpGroup,
	}
	flags.Bind("", false, getGIDCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	whoamiCmd := &cobra.Command{
		Use:   consts.WhoamiStr,
		Short: "Get session user execution context",
		Long:  help.GetHelpFor([]string{consts.WhoamiStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WhoamiCmd(cmd, con, args)
		},
		GroupID: consts.InfoHelpGroup,
	}
	flags.Bind("", false, whoamiCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	infoCmd := &cobra.Command{
		Use:   consts.InfoStr,
		Short: "Get session info",
		Long:  help.GetHelpFor([]string{consts.InfoStr}),
		Run: func(cmd *cobra.Command, args []string) {
			InfoCmd(cmd, con, args)
		},
		GroupID: consts.InfoHelpGroup,
	}
	flags.Bind("use", false, infoCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	return []*cobra.Command{pingCmd, getPIDCmd, getUIDCmd, getGIDCmd, whoamiCmd, infoCmd}
}
