package jobs

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

// Commands returns the `jobs` command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	jobsCmd := &cobra.Command{
		Use:   consts.JobsStr,
		Short: "Job control",
		Long:  help.GetHelpFor([]string{consts.JobsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			JobsCmd(cmd, con, args)
		},
		GroupID: consts.NetworkHelpGroup,
	}
	flags.Bind("jobs", true, jobsCmd, func(f *pflag.FlagSet) {
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.Bind("jobs", false, jobsCmd, func(f *pflag.FlagSet) {
		f.Int32P("kill", "k", -1, "kill a background job")
		f.BoolP("kill-all", "K", false, "kill all jobs")
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	completers.NewFlagCompsFor(jobsCmd, func(comp *carapace.ActionMap) {
		(*comp)["kill"] = JobsIDCompleter(con)
	})

	return []*cobra.Command{jobsCmd}
}
