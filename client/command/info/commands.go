package info

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/command/use"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverConsoleClient) []*cobra.Command {
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
