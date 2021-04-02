package commands

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
	"github.com/jessevdk/go-flags"
	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/bishopfox/sliver/client/commands/server"
	"github.com/bishopfox/sliver/client/commands/sliver"
	"github.com/bishopfox/sliver/client/commands/transports"
	cctx "github.com/bishopfox/sliver/client/context"
)

const (
	// ANSI Colors
	normal    = "\033[0m"
	black     = "\033[30m"
	red       = "\033[31m"
	green     = "\033[32m"
	orange    = "\033[33m"
	blue      = "\033[34m"
	purple    = "\033[35m"
	cyan      = "\033[36m"
	gray      = "\033[37m"
	bold      = "\033[1m"
	clearln   = "\r\x1b[2K"
	upN       = "\033[%dA"
	downN     = "\033[%dB"
	underline = "\033[4m"
)

// BindCommands - Passes along a go-flags Command Parser, to which are bound commands
// depending on the current context, OS-specific session commands, etc. This parser
// is returned because the console might need it to load additional things, like completions.
func BindCommands() (parser *flags.Parser, err error) {

	// Not fundamentally useful, but might be clearer sometimes.
	switch cctx.Context.Menu {
	case cctx.Server:
		parser = flags.NewNamedParser("server", flags.HelpFlag)
	case cctx.Sliver:
		parser = flags.NewNamedParser("sliver", flags.HelpFlag)
	}

	// Stack up parsing options :
	// 1 - Add help options to all commands
	// 2 - Ignore unknown options (some commands needs args that are flags, ex: sideload)
	parser.Options = flags.IgnoreUnknown | flags.HelpFlag

	// Add global option flags, applying to all commands (ex: timeouts)
	parser.AddGroup("global options", "these options are available to every command", &GlobalOptions{})

	// First register all server commands if in server context
	server.BindCommands(parser)

	// Register the transports: the commands will be bound
	// only if the current server or sliver supports them.
	transports.BindCommands(parser)

	// Add sliver commands if in sliver context. This will also
	// automatically bind OS-specific commands if there are some.
	if cctx.Context.Menu == cctx.Sliver {
		sliver.BindCommands(parser)
	}

	// Pass the parser to the context package: some commands will
	// request this package to forge protobuf Request fields with
	// some of the parser settings (like default/specified option timeouts)
	cctx.Commands.Initialize(parser)

	return
}

// This should be called for any dangerous (OPSEC-wise) functions
func isUserAnAdult() bool {
	confirm := false
	prompt := &survey.Confirm{Message: "This action is bad OPSEC, are you an adult?"}
	survey.AskOne(prompt, &confirm, nil)
	return confirm
}

// GlobalOptions - Options appended directly to a command parser. These options will not explicitely
// be shown in option completions, but they are still available for use.
type GlobalOptions struct {
	Timeout int64 `long:"timeout" short:"t" description:"command timeout in seconds" default:"60"`
}

// ArgumentByName Get the name of a detected command's argument
func ArgumentByName(command *flags.Command, name string) *flags.Arg {
	args := command.Args()
	for _, arg := range args {
		if arg.Name == name {
			return arg
		}
	}

	// Maybe we can check for aliases, later...
	// Might sometimes push interesting things...

	return nil
}

// OptionByName - Returns an option for a command or a subcommand, identified by name
func OptionByName(cmd *flags.Command, option string) *flags.Option {

	if cmd == nil {
		return nil
	}
	// Get all (root) option groups.
	groups := cmd.Groups()

	// For each group, build completions
	for _, grp := range groups {
		// Add each option to completion group
		for _, opt := range grp.Options() {
			if opt.LongName == option {
				return opt
			}
		}
	}
	return nil
}
