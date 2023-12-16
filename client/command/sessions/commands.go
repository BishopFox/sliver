package sessions

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

// Commands returns the `sessions` command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	sessionsCmd := &cobra.Command{
		Use:   consts.SessionsStr,
		Short: "Session management",
		Long:  help.GetHelpFor([]string{consts.SessionsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			SessionsCmd(cmd, con, args)
		},
		GroupID: consts.SliverHelpGroup,
	}
	flags.Bind("sessions", true, sessionsCmd, func(f *pflag.FlagSet) {
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.Bind("sessions", false, sessionsCmd, func(f *pflag.FlagSet) {
		f.StringP("interact", "i", "", "interact with a session")
		f.StringP("kill", "k", "", "kill the designated session")
		f.BoolP("kill-all", "K", false, "kill all the sessions")
		f.BoolP("clean", "C", false, "clean out any sessions marked as [DEAD]")
		f.BoolP("force", "F", false, "force session action without waiting for results")

		f.StringP("filter", "f", "", "filter sessions by substring")
		f.StringP("filter-re", "e", "", "filter sessions by regular expression")
	})
	completers.NewFlagCompsFor(sessionsCmd, func(comp *carapace.ActionMap) {
		(*comp)["interact"] = SessionIDCompleter(con)
		(*comp)["kill"] = SessionIDCompleter(con)
	})

	sessionsPruneCmd := &cobra.Command{
		Use:   consts.PruneStr,
		Short: "Kill all stale/dead sessions",
		Long:  help.GetHelpFor([]string{consts.SessionsStr, consts.PruneStr}),
		Run: func(cmd *cobra.Command, args []string) {
			SessionsPruneCmd(cmd, con, args)
		},
	}
	flags.Bind("prune", false, sessionsPruneCmd, func(f *pflag.FlagSet) {
		f.BoolP("force", "F", false, "Force the killing of stale/dead sessions")
	})
	sessionsCmd.AddCommand(sessionsPruneCmd)

	return []*cobra.Command{sessionsCmd}
}

// SliverCommands returns all session control commands for the active target.
func SliverCommands(con *console.SliverClient) []*cobra.Command {
	backgroundCmd := &cobra.Command{
		Use:         consts.BackgroundStr,
		Short:       "Background an active session",
		Long:        help.GetHelpFor([]string{consts.BackgroundStr}),
		Annotations: flags.RestrictTargets(consts.ConsoleCmdsFilter),
		Run: func(cmd *cobra.Command, args []string) {
			BackgroundCmd(cmd, con, args)
		},
		GroupID: consts.SliverCoreHelpGroup,
	}
	flags.Bind("use", false, backgroundCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	openSessionCmd := &cobra.Command{
		Use:   consts.InteractiveStr,
		Short: "Task a beacon to open an interactive session (Beacon only)",
		Long:  help.GetHelpFor([]string{consts.InteractiveStr}),
		Run: func(cmd *cobra.Command, args []string) {
			InteractiveCmd(cmd, con, args)
		},
		GroupID:     consts.SliverCoreHelpGroup,
		Annotations: flags.RestrictTargets(consts.BeaconCmdsFilter),
	}
	flags.Bind("interactive", false, openSessionCmd, func(f *pflag.FlagSet) {
		f.StringP("mtls", "m", "", "mtls connection strings")
		f.StringP("wg", "g", "", "wg connection strings")
		f.StringP("http", "b", "", "http(s) connection strings")
		f.StringP("dns", "n", "", "dns connection strings")
		f.StringP("named-pipe", "p", "", "namedpipe connection strings")
		f.StringP("tcp-pivot", "i", "", "tcppivot connection strings")

		f.StringP("delay", "d", "0s", "delay opening the session (after checkin) for a given period of time")

		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	closeSessionCmd := &cobra.Command{
		Use:   consts.CloseStr,
		Short: "Close an interactive session without killing the remote process",
		Long:  help.GetHelpFor([]string{consts.CloseStr}),
		Run: func(cmd *cobra.Command, args []string) {
			CloseSessionCmd(cmd, con, args)
		},
		GroupID: consts.SliverCoreHelpGroup,
	}
	flags.Bind("", false, closeSessionCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	return []*cobra.Command{backgroundCmd, openSessionCmd, closeSessionCmd}
}
