package processes

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/completers"
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	psCmd := &cobra.Command{
		Use:   consts.PsStr,
		Short: "List remote processes",
		Long:  help.GetHelpFor([]string{consts.PsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			PsCmd(cmd, con, args)
		},
		GroupID: consts.ProcessHelpGroup,
	}
	flags.Bind("", false, psCmd, func(f *pflag.FlagSet) {
		f.IntP("pid", "p", -1, "filter based on pid")
		f.StringP("exe", "e", "", "filter based on executable name")
		f.StringP("owner", "o", "", "filter based on owner")
		f.BoolP("print-cmdline", "c", false, "print command line arguments")
		f.BoolP("overflow", "O", false, "overflow terminal width (display truncated rows)")
		f.IntP("skip-pages", "S", 0, "skip the first n page(s)")
		f.BoolP("tree", "T", false, "print process tree")

		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	procdumpCmd := &cobra.Command{
		Use:   consts.ProcdumpStr,
		Short: "Dump process memory",
		Long:  help.GetHelpFor([]string{consts.ProcdumpStr}),
		Run: func(cmd *cobra.Command, args []string) {
			ProcdumpCmd(cmd, con, args)
		},
		GroupID: consts.ProcessHelpGroup,
	}
	flags.Bind("", false, procdumpCmd, func(f *pflag.FlagSet) {
		f.IntP("pid", "p", -1, "target pid")
		f.StringP("name", "n", "", "target process name")
		f.StringP("save", "s", "", "save to file (will overwrite if exists)")
		f.BoolP("loot", "X", false, "save output as loot")
		f.StringP("loot-name", "N", "", "name to assign when adding the memory dump to the loot store (optional)")

		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	completers.NewFlagCompsFor(procdumpCmd, func(comp *carapace.ActionMap) {
		(*comp)["save"] = carapace.ActionFiles()
	})

	terminateCmd := &cobra.Command{
		Use:   consts.TerminateStr,
		Short: "Terminate a process on the remote system",
		Long:  help.GetHelpFor([]string{consts.TerminateStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			TerminateCmd(cmd, con, args)
		},
		GroupID: consts.ProcessHelpGroup,
	}
	flags.Bind("", false, terminateCmd, func(f *pflag.FlagSet) {
		f.BoolP("force", "F", false, "disregard safety and kill the PID")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	carapace.Gen(terminateCmd).PositionalCompletion(carapace.ActionValues().Usage("process ID"))

	return []*cobra.Command{psCmd, procdumpCmd, terminateCmd}
}
