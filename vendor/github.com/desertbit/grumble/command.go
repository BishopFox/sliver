/*
 * The MIT License (MIT)
 *
 * Copyright (c) 2018 Roland Singer [roland.singer@deserbit.com]
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package grumble

import (
	"fmt"
)

// Command is just that, a command for your application.
type Command struct {
	// Command name.
	// This field is required.
	Name string

	// Command name aliases.
	Aliases []string

	// One liner help message for the command.
	// This field is required.
	Help string

	// More descriptive help message for the command.
	LongHelp string

	// HelpGroup defines the help group headline.
	// Note: this is only used for primary top-level commands.
	HelpGroup string

	// Usage should define how to use the command.
	// Sample: start [OPTIONS] CONTAINER [CONTAINER...]
	Usage string

	// Define all command flags within this function.
	Flags func(f *Flags)

	// Define all command arguments within this function.
	Args func(a *Args)

	// Function to execute for the command.
	Run func(c *Context) error

	// Completer is custom autocompleter for command.
	// It takes in command arguments and returns autocomplete options.
	// By default all commands get autocomplete of subcommands.
	// A non-nil Completer overrides the default behaviour.
	Completer func(prefix string, args []string) []string

	parent   *Command
	flags    Flags
	args     Args
	commands Commands
}

func (c *Command) validate() error {
	if len(c.Name) == 0 {
		return fmt.Errorf("empty command name")
	} else if c.Name[0] == '-' {
		return fmt.Errorf("command name must not start with a '-'")
	} else if len(c.Help) == 0 {
		return fmt.Errorf("empty command help")
	}
	return nil
}

func (c *Command) registerFlagsAndArgs(addHelpFlag bool) {
	if addHelpFlag {
		// Add default help command.
		c.flags.Bool("h", "help", false, "display help")
	}

	if c.Flags != nil {
		c.Flags(&c.flags)
	}
	if c.Args != nil {
		c.Args(&c.args)
	}
}

// Parent returns the parent command or nil.
func (c *Command) Parent() *Command {
	return c.parent
}

// AddCommand adds a new command.
// Panics on error.
func (c *Command) AddCommand(cmd *Command) {
	err := cmd.validate()
	if err != nil {
		panic(err)
	}

	cmd.parent = c
	cmd.registerFlagsAndArgs(true)

	c.commands.Add(cmd)
}
