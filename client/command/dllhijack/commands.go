package dllhijack

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
	dllhijackCmd := &cobra.Command{
		Use:         consts.DLLHijackStr,
		Short:       "Plant a DLL for a hijack scenario",
		Long:        help.GetHelpFor([]string{consts.DLLHijackStr}),
		GroupID:     consts.ExecutionHelpGroup,
		Annotations: flags.RestrictTargets(consts.WindowsCmdsFilter),
		Args:        cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			DllHijackCmd(cmd, con, args)
		},
	}
	flags.Bind("", false, dllhijackCmd, func(f *pflag.FlagSet) {
		f.StringP("reference-path", "r", "", "Path to the reference DLL on the remote system")
		f.StringP("reference-file", "R", "", "Path to the reference DLL on the local system")
		f.StringP("file", "f", "", "Local path to the DLL to plant for the hijack")
		f.StringP("profile", "p", "", "Profile name to use as a base DLL")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.BindFlagCompletions(dllhijackCmd, func(comp *carapace.ActionMap) {
		(*comp)["reference-file"] = carapace.ActionFiles()
		(*comp)["file"] = carapace.ActionFiles()
		(*comp)["profile"] = generate.ProfileNameCompleter(con)
	})
	carapace.Gen(dllhijackCmd).PositionalCompletion(carapace.ActionValues().Usage("Path to upload the DLL to on the remote system"))

	return []*cobra.Command{dllhijackCmd}
}
