package command

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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
	"strings"

	client "github.com/bishopfox/sliver/client/console"
	"github.com/reeflective/console"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Bind is a convenience function to bind flags to a given command.
// name - The name of the flag set (can be empty).
// cmd  - The command to which the flags should be bound.
// flags - A function exposing the flag set through which flags are declared.
func Bind(name string, persistent bool, cmd *cobra.Command, flags func(f *pflag.FlagSet)) {
	flagSet := pflag.NewFlagSet(name, pflag.ContinueOnError) // Create the flag set.
	flags(flagSet)                                           // Let the user bind any number of flags to it.

	if persistent {
		cmd.PersistentFlags().AddFlagSet(flagSet)
	} else {
		cmd.Flags().AddFlagSet(flagSet)
	}
}

// BindFlagCompletions is a convenience function for adding completions to a command's flags.
// cmd - The command owning the flags to complete.
// bind - A function exposing a map["flag-name"]carapace.Action.
func BindFlagCompletions(cmd *cobra.Command, bind func(comp *carapace.ActionMap)) {
	comps := make(carapace.ActionMap)
	bind(&comps)

	carapace.Gen(cmd).FlagCompletion(comps)
}

// RestrictTargets generates a cobra annotation map with a single console.CommandHiddenFilter key
// to a comma-separated list of filters to use in order to expose/hide commands based on requirements.
// Ex: cmd.Annotations = RestrictTargets("windows") will only show the command if the target is Windows.
// Ex: cmd.Annotations = RestrictTargets("windows", "beacon") show the command if target is a beacon on Windows.
func RestrictTargets(filters ...string) map[string]string {
	if len(filters) == 0 {
		return nil
	}

	if len(filters) == 1 {
		return map[string]string{
			console.CommandFilterKey: filters[0],
		}
	}

	filts := strings.Join(filters, ",")

	return map[string]string{
		console.CommandFilterKey: filts,
	}
}

// makeBind returns a commandBinder helper function
// @menu  - The command menu to which the commands should be bound (either server or implant menu).
func makeBind(cmd *cobra.Command, con *client.SliverClient) func(group string, cmds ...func(con *client.SliverClient) []*cobra.Command) {
	return func(group string, cmds ...func(con *client.SliverClient) []*cobra.Command) {
		found := false

		// Ensure the given command group is available in the menu.
		if group != "" {
			for _, grp := range cmd.Groups() {
				if grp.Title == group {
					found = true
					break
				}
			}

			if !found {
				cmd.AddGroup(&cobra.Group{
					ID:    group,
					Title: group,
				})
			}
		}

		// Bind the command to the root
		for _, command := range cmds {
			cmd.AddCommand(command(con)...)
		}
	}
}

// commandBinder is a helper used to bind commands to a given menu, for a given "command help group".
//
// @group - Name of the group under which the command should be shown. Preferably use a string in the constants package.
// @ cmds - A list of functions returning a list of root commands to bind. See any package's `commands.go` file and function.
// type commandBinder func(group string, cmds ...func(con *client.SliverClient) []*cobra.Command)

// [ Core ]
// [ Sessions ]
// [ Execution ]
// [ Filesystem ]
// [ Info ]
// [ Network (C2)]
// [ Network tools ]
// [ Payloads ]
// [ Privileges ]
// [ Processes ]
// [ Aliases ]
// [ Extensions ]

// Commands not to bind in CLI:
// - portforwarders
// - Socks (and wg-socks ?)
// - shell ?

// Take care of:
// - double bind help command
// - double bind session commands
// - don't bind readline command in CLI.
