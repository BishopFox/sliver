package kill

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	killCmd := &cobra.Command{
		Use:   consts.KillStr,
		Short: "Kill a session",
		Long:  help.GetHelpFor([]string{consts.KillStr}),
		Run: func(cmd *cobra.Command, args []string) {
			KillCmd(cmd, con, args)
		},
		GroupID: consts.SliverCoreHelpGroup,
	}
	flags.Bind("use", false, killCmd, func(f *pflag.FlagSet) {
		f.BoolP("force", "F", false, "Force kill,  does not clean up")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	return []*cobra.Command{killCmd}
}
