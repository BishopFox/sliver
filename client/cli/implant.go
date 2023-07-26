package cli

import (
	"errors"
	"strings"

	"github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/client/command/use"
	client "github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/reeflective/console"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func implantCmd(con *client.SliverClient, sliverCmds console.Commands) *cobra.Command {
	implantCmd := sliverCmds()
	implantCmd.Use = constants.ImplantMenu

	implantFlags := pflag.NewFlagSet(constants.ImplantMenu, pflag.ContinueOnError)
	implantFlags.StringP("use", "s", "", "Set the active target session/beacon")
	implantCmd.Flags().AddFlagSet(implantFlags)

	// Setup the active target before going down the implant command tree.
	implantCmd.PersistentPreRunE = sliverPrerun(implantCmd, con)

	// Completions:
	command.BindFlagCompletions(implantCmd, func(comp *carapace.ActionMap) {
		(*comp)["use"] = carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			return use.BeaconAndSessionIDCompleter(con)
		})
	})

	// Hide all commands not available for the current target.
	comps := carapace.Gen(implantCmd)
	comps.PreRun(func(cmd *cobra.Command, args []string) {
		err := implantCmd.PersistentPreRunE(cmd, args)
		if err != nil {
			return
		}

		hideUnavailableCommands(implantCmd, con)
	})

	return implantCmd
}

// hide commands that are filtered so that they are not
// shown in the help strings or proposed as completions.
func hideUnavailableCommands(rootCmd *cobra.Command, con *client.SliverClient) {
	targetFilters := con.ActiveTarget.Filters()

	for _, cmd := range rootCmd.Commands() {
		// Don't override commands if they are already hidden
		if cmd.Hidden {
			continue
		}

		if isFiltered(cmd, targetFilters) {
			cmd.Hidden = true
		}
	}
}

func isFiltered(cmd *cobra.Command, targetFilters []string) bool {
	if cmd.Annotations == nil {
		return false
	}

	// Get the filters on the command
	filterStr := cmd.Annotations[console.CommandFilterKey]
	filters := strings.Split(filterStr, ",")

	for _, cmdFilter := range filters {
		for _, filter := range targetFilters {
			if cmdFilter != "" && cmdFilter == filter {
				return true
			}
		}
	}

	return false
}

func sliverPrerun(implantCmd *cobra.Command, con *client.SliverClient) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := preRunClient(con)(cmd, args); err != nil {
			return err
		}

		if err := implantCmd.ParseFlags(args); err != nil {
			return err
		}

		target, _ := implantCmd.Flags().GetString("use")
		if target == "" {
			return errors.New("no target implant to run command on")
		}

		// Load either the session or the beacon.
		session := con.GetSession(target)
		if session != nil {
			con.ActiveTarget.Set(session, nil)
			return nil
		}

		beacon := con.GetBeacon(target)
		if beacon != nil {
			con.ActiveTarget.Set(nil, beacon)
			return nil
		}

		return nil
	}
}
