package backdoor

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	backdoorCmd := &cobra.Command{
		Use:         consts.BackdoorStr,
		Short:       "Infect a remote file with a sliver shellcode",
		Long:        help.GetHelpFor([]string{consts.BackdoorStr}),
		Args:        cobra.ExactArgs(1),
		GroupID:     consts.ExecutionHelpGroup,
		Annotations: flags.RestrictTargets(consts.WindowsCmdsFilter),
		Run: func(cmd *cobra.Command, args []string) {
			BackdoorCmd(cmd, con, args)
		},
	}
	flags.Bind("", false, backdoorCmd, func(f *pflag.FlagSet) {
		f.StringP("profile", "p", "", "profile to use for service binary")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.BindFlagCompletions(backdoorCmd, func(comp *carapace.ActionMap) {
		(*comp)["profile"] = generate.ProfileNameCompleter(con)
	})
	carapace.Gen(backdoorCmd).PositionalCompletion(carapace.ActionValues().Usage("path to the remote file to backdoor"))

	return []*cobra.Command{backdoorCmd}
}
