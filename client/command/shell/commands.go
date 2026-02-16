package shell

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	shellCmd := &cobra.Command{
		Use:         consts.ShellStr,
		Short:       "Start an interactive shell",
		Long:        help.GetHelpFor([]string{consts.ShellStr}),
		GroupID:     consts.ExecutionHelpGroup,
		Annotations: flags.RestrictTargets(consts.SessionCmdsFilter),
		Run: func(cmd *cobra.Command, args []string) {
			ShellCmd(cmd, con, args)
		},
	}
	flags.Bind("", false, shellCmd, func(f *pflag.FlagSet) {
		f.BoolP("no-pty", "y", false, "disable use of pty on macos/linux")
		f.StringP("shell-path", "s", "", "path to shell interpreter")

		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	shellLsCmd := &cobra.Command{
		Use:   consts.LsStr,
		Short: "List managed local shell tunnels",
		Long:  help.GetHelpFor([]string{consts.ShellStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ShellLsCmd(cmd, con, args)
		},
	}
	shellCmd.AddCommand(shellLsCmd)

	shellAttachCmd := &cobra.Command{
		Use:   "attach <id>",
		Short: "Attach to a managed local shell tunnel",
		Long:  help.GetHelpFor([]string{consts.ShellStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ShellAttachCmd(cmd, con, args)
		},
	}
	shellCmd.AddCommand(shellAttachCmd)
	carapace.Gen(shellAttachCmd).PositionalCompletion(ShellIDCompleter(con).Usage("managed shell ID"))

	return []*cobra.Command{shellCmd}
}
