package commands

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"fmt"

	"github.com/reeflective/team/client"
	cli "github.com/reeflective/team/client/commands"
	"github.com/reeflective/team/internal/command"
	"github.com/reeflective/team/server"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Generate returns a "teamserver" command root and its tree for teamserver (server-side) management.
// It requires a teamclient so as to bind its "teamclient" tree as a subcommand of the server root.
// This is so that all CLI applications which can be a teamserver can also be a client of their own.
//
// ** Commands do:
//   - Work even if the teamserver/client returns errors: those are returned &| printed &| logged.
//   - Use the cobra utilities OutOrStdout(), ErrOrStdErr(), ... for all and every command output.
//   - Have attached completions for users/listeners/config files of all sorts, and other things.
//   - Have the ability to be ran in closed-loop console applications ("single runtime shell").
//
// ** Commands do NOT:
//   - Ensure they are connected to a server instance before running (in memory).
//   - Call os.Exit() anywhere, thus will not exit the program embedding them.
//   - Ignite/start the teamserver core/filesystem/backends before they absolutely need to.
//     Consequently, do not touch the filesystem until they absolutely need to.
//   - Connect the client more than once to the teamserver.
//   - Start persistent listeners, excluding the daemon command.
func Generate(teamserver *server.Server, teamclient *client.Client) *cobra.Command {
	// Server-only commands always need to have open log
	// files, most of the time access to the database, etc.
	// On top, they need a listener in memory.
	servCmds := serverCommands(teamserver, teamclient)

	// We bind the same runners to the client-side commands.
	cliCmds := cli.Generate(teamclient)
	cliCmds.Use = "client"
	cliCmds.GroupID = command.TeamServerGroup

	servCmds.AddCommand(cliCmds)

	return servCmds
}

func serverCommands(server *server.Server, client *client.Client) *cobra.Command {
	teamCmd := &cobra.Command{
		Use:          "teamserver",
		Short:        fmt.Sprintf("Manage the %s teamserver and users", server.Name()),
		SilenceUsage: true,
	}

	// Groups
	teamCmd.AddGroup(
		&cobra.Group{ID: command.TeamServerGroup, Title: command.TeamServerGroup},
		&cobra.Group{ID: command.UserManagementGroup, Title: command.UserManagementGroup},
	)

	teamFlags := pflag.NewFlagSet("teamserver", pflag.ContinueOnError)
	teamFlags.CountP("verbosity", "v", "Counter flag (-vvv) to increase log verbosity on stdout (1:info-> 3:trace)")
	teamCmd.PersistentFlags().AddFlagSet(teamFlags)

	// [ Listeners and servers control commands ] ------------------------------------------

	// Start a listener
	listenCmd := &cobra.Command{
		Use:     "listen",
		Short:   "Start a teamserver listener (non-blocking)",
		GroupID: command.TeamServerGroup,
		RunE:    startListenerCmd(server),
	}

	lnFlags := pflag.NewFlagSet("listener", pflag.ContinueOnError)
	lnFlags.StringP("host", "H", "", "interface to bind server to")
	lnFlags.StringP("listener", "l", "", "listener stack to use instead of default (completed)")
	lnFlags.Uint16P("port", "P", 31337, "tcp listen port")
	lnFlags.BoolP("persistent", "p", false, "make listener persistent across restarts")
	listenCmd.Flags().AddFlagSet(lnFlags)

	listenComps := make(carapace.ActionMap)
	listenComps["host"] = interfacesCompleter()
	listenComps["listener"] = carapace.ActionCallback(listenerTypeCompleter(client, server))
	carapace.Gen(listenCmd).FlagCompletion(listenComps)

	teamCmd.AddCommand(listenCmd)

	// Close a listener
	closeCmd := &cobra.Command{
		Use:     "close",
		Short:   "Close a listener and remove it from persistent ones if it's one",
		Args:    cobra.MinimumNArgs(1),
		GroupID: command.TeamServerGroup,
		Run:     closeCmd(server),
	}

	closeComps := carapace.Gen(closeCmd)
	closeComps.PositionalAnyCompletion(carapace.ActionCallback(listenerIDCompleter(client, server)))

	closeComps.PreRun(func(cmd *cobra.Command, args []string) {
		cmd.PersistentPreRunE(cmd, args)
	})

	teamCmd.AddCommand(closeCmd)

	// Daemon (blocking listener and persistent jobs)
	daemonCmd := &cobra.Command{
		Use:     "daemon",
		Short:   "Start the teamserver in daemon mode (blocking)",
		GroupID: command.TeamServerGroup,
		RunE:    daemoncmd(server),
	}
	daemonCmd.Flags().StringP("host", "l", "-", "multiplayer listener host")
	daemonCmd.Flags().Uint16P("port", "p", uint16(0), "multiplayer listener port")

	daemonComps := make(carapace.ActionMap)
	daemonComps["host"] = interfacesCompleter()
	carapace.Gen(daemonCmd).FlagCompletion(daemonComps)

	teamCmd.AddCommand(daemonCmd)

	// Systemd configuration output
	systemdCmd := &cobra.Command{
		Use:     "systemd",
		Short:   "Print a systemd unit file for the application teamserver, with options",
		GroupID: command.TeamServerGroup,
		RunE:    systemdConfigCmd(server),
	}

	sFlags := pflag.NewFlagSet("systemd", pflag.ContinueOnError)
	sFlags.StringP("binpath", "b", "", "Specify the path of the teamserver application binary")
	sFlags.StringP("user", "u", "", "Specify the user for the systemd file to run with")
	sFlags.StringP("save", "s", "", "Directory/file in which to save config, instead of stdout")
	sFlags.StringP("host", "l", "", "Listen host to use in the systemd command line")
	sFlags.Uint16P("port", "p", 0, "Listen port in the systemd command line")
	systemdCmd.Flags().AddFlagSet(sFlags)

	sComps := make(carapace.ActionMap)
	sComps["save"] = carapace.ActionFiles()
	sComps["binpath"] = carapace.ActionFiles()
	sComps["host"] = interfacesCompleter()
	carapace.Gen(systemdCmd).FlagCompletion(sComps)

	teamCmd.AddCommand(systemdCmd)

	statusCmd := &cobra.Command{
		Use:     "status",
		Short:   "Show the status of the teamserver (listeners, configurations, health...)",
		GroupID: command.TeamServerGroup,
		Run:     statusCmd(server),
	}

	teamCmd.AddCommand(statusCmd)

	// [ Users and data control commands ] -------------------------------------------------

	// Add user
	userCmd := &cobra.Command{
		Use:     "user",
		Short:   "Create a user for this teamserver and generate its client configuration file",
		GroupID: command.UserManagementGroup,
		Run:     createUserCmd(server, client),
	}

	teamCmd.AddCommand(userCmd)

	userFlags := pflag.NewFlagSet("user", pflag.ContinueOnError)
	userFlags.StringP("host", "l", "", "listen host")
	userFlags.Uint16P("port", "p", 0, "listen port")
	userFlags.StringP("save", "s", "", "directory/file in which to save config")
	userFlags.StringP("name", "n", "", "user name")
	userFlags.BoolP("system", "U", false, "Use the current OS user, and save its configuration directly in client dir")
	userCmd.Flags().AddFlagSet(userFlags)

	userComps := make(carapace.ActionMap)
	userComps["save"] = carapace.ActionDirectories()
	userComps["host"] = interfacesCompleter()
	carapace.Gen(userCmd).FlagCompletion(userComps)

	// Delete and kick user
	rmUserCmd := &cobra.Command{
		Use:     "delete",
		Short:   "Remove a user from the teamserver, and revoke all its current tokens",
		GroupID: command.UserManagementGroup,
		Args:    cobra.ExactArgs(1),
		Run:     rmUserCmd(server),
	}

	teamCmd.AddCommand(rmUserCmd)

	rmUserComps := carapace.Gen(rmUserCmd)

	rmUserComps.PositionalCompletion(carapace.ActionCallback(userCompleter(client, server)))

	rmUserComps.PreRun(func(cmd *cobra.Command, args []string) {
		cmd.PersistentPreRunE(cmd, args)
	})

	// Import a list of users and their credentials.
	cmdImportCA := &cobra.Command{
		Use:     "import",
		Short:   "Import a certificate Authority file containing teamserver users",
		GroupID: command.UserManagementGroup,
		Args:    cobra.ExactArgs(1),
		Run:     importCACmd(server),
	}

	iComps := carapace.Gen(cmdImportCA)
	iComps.PositionalCompletion(
		carapace.Batch(
			carapace.ActionCallback(cli.ConfigsCompleter(client, "teamserver/certs", ".teamserver.pem", "other teamservers user CAs", true)),
			carapace.ActionFiles().Tag("teamserver user CAs"),
		).ToA(),
	)

	teamCmd.AddCommand(cmdImportCA)

	// Export the list of users and their credentials.
	cmdExportCA := &cobra.Command{
		Use:     "export",
		Short:   "Export a Certificate Authority file containing the teamserver users",
		GroupID: command.UserManagementGroup,
		Args:    cobra.RangeArgs(0, 1),
		Run:     exportCACmd(server),
	}

	carapace.Gen(cmdExportCA).PositionalCompletion(carapace.ActionFiles())
	teamCmd.AddCommand(cmdExportCA)

	return teamCmd
}
