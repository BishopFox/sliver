package cli

import (
	"errors"
	"os"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/exp/slices"

	"github.com/reeflective/console"

	"github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/client/command/completers"
	"github.com/bishopfox/sliver/client/command/use"
	client "github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
)

// implantCmd returns the command tree for Sliver active targets.
// This function has not yet found its place into one of the commands subdirectories,
// since I'm not really sure how to do it in a way that would be optimal for everyone.
func implantCmd(con *client.SliverClient, sliverCmds console.Commands) *cobra.Command {
	// Generate a not-yet filtered tree of all commands
	// usable in the context of an active target implant.
	implantCmd := sliverCmds()
	implantCmd.Use = constants.ImplantMenu
	implantCmd.Short = "Implant target command tree (equivalent of the sliver menu)"
	implantCmd.GroupID = constants.SliverHelpGroup

	// But let the user set this implant with a flag.
	implantFlags := pflag.NewFlagSet(constants.ImplantMenu, pflag.ContinueOnError)
	implantFlags.StringP("use", "s", "", "Set the active target session/beacon")
	implantCmd.Flags().AddFlagSet(implantFlags)

	// And when pre-running any of the commands in this tree,
	// connect to the server as we always do, but also set the
	// active target for this binary run.
	implantCmd.PersistentPreRunE = preRunImplant(implantCmd, con)
	implantCmd.PersistentPostRunE = postRunImplant(implantCmd, con)

	// Completions.
	// Unlike the server-only command tree, we need to unconditionally
	// pre-connect when completing commands, so that we can filter commands.
	comps := carapace.Gen(implantCmd)
	comps.PreRun(func(cmd *cobra.Command, args []string) {
		err := implantCmd.PersistentPreRunE(cmd, args)
		if err != nil {
			return
		}

		// And let the console and its active target decide
		// what should be available to us, and what should not.
		con.ActiveTarget.FilterCommands(implantCmd)
	})

	// This completer will try connect to the server anyway, if not done already.
	completers.NewFlagCompsFor(implantCmd, func(comp *carapace.ActionMap) {
		(*comp)["use"] = carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			return use.BeaconAndSessionIDCompleter(con)
		})
	})

	return implantCmd
}

// preRunImplant returns the pre-runner to be ran before any implant-targeting command,
// to connect to the server, query the sessions/beacons and set one as the active target.
// Like implantCmd, this function can moved from here down the road, either integrated as
// a console.Client method to be used easily by the users using "custom" Go clients for some things.
func preRunImplant(implantCmd *cobra.Command, con *client.SliverClient) command.CobraRunnerE {
	return func(cmd *cobra.Command, args []string) error {
		if err := con.PreRunConnect(cmd, args); err != nil {
			return err
		}

		// Pre-parse the flags and get our active target.
		if err := implantCmd.ParseFlags(args); err != nil {
			return err
		}

		target, _ := implantCmd.Flags().GetString("use")
		if target == "" {
			return errors.New("no active implant target to run command")
		}

		// Load either the session or the beacon.
		// This also sets the correct console menu.
		session := con.GetSession(target)
		if session != nil {
			con.ActiveTarget.Set(session, nil)
		} else {
			beacon := con.GetBeacon(target)
			if beacon != nil {
				con.ActiveTarget.Set(nil, beacon)
			}
		}

		// If the command is marked filtered (should not be ran in
		// the current context/target), don't do anything and return.
		// This is identical to the filtering behavior in the console.
		if err := con.App.ActiveMenu().CheckIsAvailable(cmd); err != nil {
			return err
		}

		// Keep a copy of the command-line: used for beacon task command-line info.
		con.Args = os.Args[1:]
		for i, arg := range os.Args {
			if arg == cmd.Name() {
				con.Args = os.Args[i:]
			} else if slices.Contains(cmd.Aliases, arg) {
				con.Args = os.Args[i:]
			}
		}

		return nil
	}
}

// postRunImplant saves the current CLI os.Args command line to the server, so that all interactions
// with an implant from a system shell will be logged and accessible just the same as in the console.
func postRunImplant(implantCmd *cobra.Command, con *client.SliverClient) command.CobraRunnerE {
	return func(cmd *cobra.Command, args []string) error {
		var saveArgs []string

		// Save only the subset of the command line starting
		// at the implant root (not the server one). This
		// is quite hackish, but I could not come up with
		// a better solution.
		for i, arg := range os.Args {
			if arg == cmd.Name() {
				saveArgs = os.Args[i:]
			} else if slices.Contains(cmd.Aliases, arg) {
				saveArgs = os.Args[i:]
			}
		}

		con.ActiveTarget.SaveCommandLine(saveArgs)

		// And disconnect from the server like for other commands.
		return con.PostRunDisconnect(cmd, args)
	}
}
