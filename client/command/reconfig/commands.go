package reconfig

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
	renameCmd := &cobra.Command{
		Use:   consts.RenameStr,
		Short: "Rename the active beacon/session",
		Long:  help.GetHelpFor([]string{consts.RenameStr}),
		Run: func(cmd *cobra.Command, args []string) {
			RenameCmd(cmd, con, args)
		},
		GroupID: consts.SliverCoreHelpGroup,
	}
	flags.Bind("rename", false, renameCmd, func(f *pflag.FlagSet) {
		f.StringP("name", "n", "", "change implant name to")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	return []*cobra.Command{renameCmd}
}
