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
	"sort"
)

// Commands collection.
type Commands struct {
	list []*Command
}

// Add the command to the slice.
// Duplicates are ignored.
func (c *Commands) Add(cmd *Command) {
	c.list = append(c.list, cmd)
}

// All returns a slice of all commands.
func (c *Commands) All() []*Command {
	return c.list
}

// Get the command by the name. Aliases are also checked.
// Returns nil if not found.
func (c *Commands) Get(name string) *Command {
	for _, cmd := range c.list {
		if cmd.Name == name {
			return cmd
		}
		for _, a := range cmd.Aliases {
			if a == name {
				return cmd
			}
		}
	}
	return nil
}

// FindCommand searches for the final command through all children.
// Returns a slice of non processed following command args.
// Returns cmd=nil if not found.
func (c *Commands) FindCommand(args []string) (cmd *Command, rest []string, err error) {
	var cmds []*Command
	cmds, _, rest, err = c.parse(args, nil, true)
	if err != nil {
		return
	}

	if len(cmds) > 0 {
		cmd = cmds[len(cmds)-1]
	}

	return
}

// Sort the commands by their name.
func (c *Commands) Sort() {
	sort.Slice(c.list, func(i, j int) bool {
		return c.list[i].Name < c.list[j].Name
	})
}

// SortRecursive sorts the commands by their name including all sub commands.
func (c *Commands) SortRecursive() {
	c.Sort()
	for _, cmd := range c.list {
		cmd.commands.SortRecursive()
	}
}

// parse the args and return a command path to the root.
// cmds slice is empty, if no command was found.
func (c *Commands) parse(
	args []string,
	parentFlagMap FlagMap,
	skipFlagMaps bool,
) (
	cmds []*Command,
	flagsMap FlagMap,
	rest []string,
	err error,
) {
	var fgs []FlagMap
	cur := c

	for len(args) > 0 && cur != nil {
		// Extract the command name from the arguments.
		name := args[0]

		// Try to find the command.
		cmd := cur.Get(name)
		if cmd == nil {
			break
		}

		args = args[1:]
		cmds = append(cmds, cmd)
		cur = &cmd.commands

		// Parse the command flags.
		fg := make(FlagMap)
		args, err = cmd.flags.parse(args, fg)
		if err != nil {
			return
		}

		if !skipFlagMaps {
			fgs = append(fgs, fg)
		}
	}

	if !skipFlagMaps {
		// Merge all the flag maps without default values.
		flagsMap = make(FlagMap)
		for i := len(fgs) - 1; i >= 0; i-- {
			flagsMap.copyMissingValues(fgs[i], false)
		}
		flagsMap.copyMissingValues(parentFlagMap, false)

		// Now include default values. This will ensure, that default values have
		// lower rank.
		for i := len(fgs) - 1; i >= 0; i-- {
			flagsMap.copyMissingValues(fgs[i], true)
		}
		flagsMap.copyMissingValues(parentFlagMap, true)
	}

	rest = args
	return
}
