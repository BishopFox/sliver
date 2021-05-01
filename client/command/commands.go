package command

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"github.com/maxlandon/gonsole"

	"github.com/bishopfox/sliver/client/command/server"
	// "github.com/bishopfox/sliver/client/commands/sliver"
	"github.com/bishopfox/sliver/client/command/c2"
	"github.com/bishopfox/sliver/client/completion"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/core"
)

// GlobalOptions - Options appended directly to a command parser. These options will not explicitely
// be shown in option completions, but they are still available for use.
type GlobalOptions struct {
	Timeout int64 `long:"timeout" short:"t" description:"command timeout in seconds" default:"60" required:"1-1"`
}

// BindCommands - Register all commands for the different menu contexts (server or sliver).
// Commands also declare their completions in the same place as where they're registered.
func BindCommands(console *gonsole.Console) {

	// We have two different context (menus)
	serverMenu := console.GetMenu(constants.ServerMenu)
	sliverMenu := console.GetMenu(constants.SliverMenu)

	// Pass the console to the various packages needing it
	// 1 - Commands packages
	server.Console = console
	c2.Console = console
	// sliver.Console = console

	// 2 - Utility packages
	core.Console = console
	completion.Console = console

	// There are some completions that apply to Environment variables for each context,
	// register them now. These completers will also be called when parsing the input
	// line and evaluating the expansions.
	serverMenu.AddExpansionCompletion('$', console.Completer.EnvironmentVariables)
	sliverMenu.AddExpansionCompletion('%', completion.CompleteSliverEnv)
	sliverMenu.AddExpansionCompletion('$', console.Completer.EnvironmentVariables)

	// Configuration command, to which we bind a special 'save' subcommand,
	// which allows us to save console configurations for our user, on the server.
	// These commands are available in all menus, and their arguments may thus vary.
	console.AddConfigCommand(constants.ConfigStr, constants.CoreServerGroup)
	console.AddConfigSubCommand(constants.ConfigSaveStr,
		"save the current console configuration on the Sliver server, to be used by all user clients",
		"",
		"sliver commands",
		[]string{""},
		func() interface{} { return &server.SaveConfig{} })

	// The gonsole library gives a help command as well.
	console.AddHelpCommand(constants.CoreServerGroup)

	// We first bind commands and options available/needed in both menus.
	// Add global options, also applying to all commands in all menus.
	var menus = []*gonsole.Menu{serverMenu, sliverMenu}

	// For each menu, register the commands we want to be available.
	for _, menu := range menus {

		// Options applying to all commands.
		menu.AddGlobalOptions("global options",
			"these options are available to every command",
			func() interface{} { return &GlobalOptions{} },
		)

		// Commands: server commands can be used in both menus.
		server.BindCommands(menu)

		// Transports: some of them can be used only in a given menu,
		// depending on what pivoting capabilities the implant has.
		c2.BindCommands(menu)
	}

	// We then register Sliver session commands to their own menu.
	// This also takes care of registering/filtering Windows commands.
	// sliver.BindCommands(sliverMenu)
}
