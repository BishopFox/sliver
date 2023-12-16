package operator

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	operatorCmd := &cobra.Command{
		Use:     consts.PreludeOperatorStr,
		Short:   "Manage connection to Prelude's Operator",
		Long:    help.GetHelpFor([]string{consts.PreludeOperatorStr}),
		GroupID: consts.GenericHelpGroup,
		Run: func(cmd *cobra.Command, args []string) {
			OperatorCmd(cmd, con, args)
		},
	}

	operatorConnectCmd := &cobra.Command{
		Use:   consts.ConnectStr,
		Short: "Connect with Prelude's Operator",
		Long:  help.GetHelpFor([]string{consts.PreludeOperatorStr, consts.ConnectStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ConnectCmd(cmd, con, args)
		},
		Args: cobra.ExactArgs(1),
	}
	operatorCmd.AddCommand(operatorConnectCmd)
	flags.Bind("operator", false, operatorConnectCmd, func(f *pflag.FlagSet) {
		f.BoolP("skip-existing", "s", false, "Do not add existing sessions as Operator Agents")
		f.StringP("aes-key", "a", "abcdefghijklmnopqrstuvwxyz012345", "AES key for communication encryption")
		f.StringP("range", "r", "sliver", "Agents range")
	})
	carapace.Gen(operatorConnectCmd).PositionalCompletion(
		carapace.ActionValues().Usage("connection string to the Operator Host (e.g. 127.0.0.1:1234)"))

	return []*cobra.Command{operatorCmd}
}
