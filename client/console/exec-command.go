package console

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
	"fmt"

	"github.com/jessevdk/go-flags"
	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/context"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/help"
	"github.com/bishopfox/sliver/client/util"
)

// ExecuteCommand - Dispatches an input line to its appropriate command.
func (c *Client) ExecuteCommand(args []string) (err error) {

	// Get the current command parser, holding all commands for the current context.
	var parser = cctx.Commands.GetCommands()

	if _, parserErr := parser.ParseArgs(args); parserErr != nil {
		err = c.HandleParserErrors(parser, parserErr, args)
	}

	// Reset the current command groups, they will be bound
	// and computed again on the next readline loop.
	cctx.Commands.ResetGroups()

	return nil
}

// HandleParserErrors - The parsers may return various types of Errors, this function handles them.
func (c *Client) HandleParserErrors(parser *flags.Parser, in error, args []string) (err error) {

	// If there is an error, cast it to a parser error, else return
	var parserErr *flags.Error
	if in == nil {
		return
	}
	parserErr, ok := in.(*flags.Error)
	if !ok {
		return
	}
	if parserErr == nil {
		return
	}

	// If command is not found, handle special (either through OS shell, or exits, etc.)
	if parserErr.Type == flags.ErrUnknownCommand {
		return c.executeSpecialCommand(args)
	}

	// If the error type is a detected -h, --help flag, print custom help.
	if parserErr.Type == flags.ErrHelp {
		cmd := c.findHelpCommand(args, parser)

		// If command is nil, it means the help was requested as
		// the menu help: print all commands for the context.
		if cmd == nil {
			switch cctx.Context.Menu {
			case cctx.Server:
				help.PrintMenuHelp(cctx.Server)
			case cctx.Sliver:
				help.PrintMenuHelp(cctx.Server)
			}
			return
		}

		// Else print the help for a specific command
		help.PrintCommandHelp(cmd)
		return
	}

	// Else, we print the raw parser error
	fmt.Println(ParserError + parserErr.Error())

	return
}

// executeSpecialCommand - Handles all commands not registered to command parsers.
func (c *Client) executeSpecialCommand(args []string) error {

	// Check context for availability
	switch context.Context.Menu {
	case context.Server:
		switch args[0] {
		default:
			// Fallback: Use the system shell through the console
			return util.Shell(args)
		}
	}

	// We should not get here, so we print an error-like message
	fmt.Printf(CommandError+"Invalid command: %s%s \n", readline.YELLOW, args[0])

	return nil
}

// findHelpCommand - A -h, --help flag was invoked in the output.
// Find the root or any subcommand.
func (c *Client) findHelpCommand(args []string, parser *flags.Parser) *flags.Command {

	var root *flags.Command
	for _, cmd := range parser.Commands() {
		if cmd.Name == args[0] {
			root = cmd
		}
	}
	if root == nil {
		return nil
	}
	if len(args) == 1 || len(root.Commands()) == 0 {
		return root
	}

	var sub *flags.Command
	if len(args) > 1 {
		for _, s := range root.Commands() {
			if s.Name == args[1] {
				sub = s
			}
		}
	}
	if sub == nil {
		return root
	}
	if len(args) == 2 || len(sub.Commands()) == 0 {
		return sub
	}

	return nil
}
