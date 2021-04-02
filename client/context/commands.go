package context

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

	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

var (
	// Commands - All commands for the current context.
	// Used to produce Protobuf requests with some specific fields
	// commands *flags.Parser, and to structure completion & help printing.
	Commands = &commands{
		ServerGroups:    &map[string][]*flags.Command{},
		serverGroupList: []string{},
		SliverGroups:    &map[string][]*flags.Command{},
		sliverGroupList: []string{},
	}
)

// commands - The go-flags library only support option grouping, not commands.
// We need to structure these groups for completion & help printing.
type commands struct {
	// The parser is being passed around, (even to other packages)
	// Holds all commands for a given context, but lacks a bit of classification.
	parser *flags.Parser

	// All groups of commands available in Server context
	ServerGroups    *map[string][]*flags.Command
	serverGroupList []string

	// All groups of commands available in Sliver context
	SliverGroups    *map[string][]*flags.Command
	sliverGroupList []string
}

// Initialize - Keeps a reference to the command parser, which may
// be used to forge Protobuf requests fields with some of its info.
func (c *commands) Initialize(parser *flags.Parser) {
	c.parser = parser
}

// Initialize - Keeps a reference to the command parser, which may
// be used to forge Protobuf requests fields with some of its info.
func (c *commands) ResetGroups() {
	// Make blank lists each time
	Commands = &commands{
		ServerGroups:    &map[string][]*flags.Command{},
		serverGroupList: []string{},
		SliverGroups:    &map[string][]*flags.Command{},
		sliverGroupList: []string{},
	}
}

// GetCommands - Some commands need to access the
// current command parser, which we serve here.
func (c *commands) GetCommands() *flags.Parser {
	return c.parser
}

// GetServerGroups - Completions and help call this to be able to restructure commands in ordered groups.
func (c *commands) GetCommandGroups() (groups []string, cmds map[string][]*flags.Command, parser *flags.Parser) {
	// We always return the same parser
	parser = c.parser

	// Depending on the current context we cumulate both context groups or not
	switch Context.Menu {
	case Server:
		groups = c.serverGroupList
		cmds = *c.ServerGroups
	case Sliver:
		// Merge both lists of commands into a temporary one
		var combined = c.ServerGroups
		for name, group := range *c.SliverGroups {
			(*combined)[name] = group
		}
		cmds = *combined
		groups = append(c.serverGroupList, c.sliverGroupList...)
	}

	return
}

// GetServerGroups - Help menus generally want only a subset of the available commands, for a given context
func (c *commands) GetServerGroups() (groups []string, cmds map[string][]*flags.Command) {
	return c.serverGroupList, *c.ServerGroups
}

// GetSliverGroups - Help menus generally want only a subset of the available commands, for a given context
func (c *commands) GetSliverGroups() (groups []string, cmds map[string][]*flags.Command) {
	return c.sliverGroupList, *c.SliverGroups
}

// RegisterServerCommand - Because Go cannot give ordered maps, we need to add a  list of
// groups with which we will structure the commands, when being used by help & completions
func (c *commands) RegisterServerCommand(err error, cmd *flags.Command, group string) {

	// If the command is nil we return.
	if cmd == nil && err != nil {
		fmt.Printf(util.CommandError+" %s\n", err.Error())
		return
	} else if err != nil {
		fmt.Printf(util.CommandError+" %s\n", err.Error())
	} else if cmd == nil {
		return
	}

	if g, exists := (*c.ServerGroups)[group]; exists {
		(*c.ServerGroups)[group] = append(g, cmd)
	} else if group == "" {
		if others, exist := (*c.ServerGroups)["others"]; exist {
			others = append(others, cmd)
		} else {
			(*c.ServerGroups)["others"] = []*flags.Command{cmd}
			c.serverGroupList = append(c.serverGroupList, "others")
		}
	} else {
		(*c.ServerGroups)[group] = []*flags.Command{cmd}
		c.serverGroupList = append(c.serverGroupList, group)
	}
}

// RegisterSliverCommand - Same as RegisterServerCommand, for Sliver context commands.
func (c *commands) RegisterSliverCommand(err error, cmd *flags.Command, group string) {

	// If the command is nil we return.
	if cmd == nil && err != nil {
		fmt.Printf(util.CommandError+" %s\n", err.Error())
		return
	} else if err != nil {
		fmt.Printf(util.CommandError+" %s\n", err.Error())
	} else if cmd == nil {
		return
	}

	if g, exists := (*c.SliverGroups)[group]; exists {
		(*c.SliverGroups)[group] = append(g, cmd)
	} else if group == "" {
		if others, exist := (*c.SliverGroups)["others"]; exist {
			others = append(others, cmd)
		} else {
			(*c.SliverGroups)["others"] = []*flags.Command{cmd}
			c.sliverGroupList = append(c.sliverGroupList, "others")
		}
	} else {
		(*c.SliverGroups)[group] = []*flags.Command{cmd}
		c.sliverGroupList = append(c.sliverGroupList, group)
	}
}

// GetCommandGroup - Get the group for a command.
func (c *commands) GetCommandGroup(cmd *flags.Command) string {
	// Server commands are accessible no matter the context
	for name, group := range *c.ServerGroups {
		for _, c := range group {
			if c.Name == cmd.Name {
				return name
			}
		}
	}

	// Sliver commands are searched for if we are in this context
	if Context.Menu == Sliver {
		for name, group := range *c.SliverGroups {
			for _, c := range group {
				if c.Name == cmd.Name {
					return name
				}
			}
		}
	}
	return ""
}

// Request - Forge a Request Protobuf metadata to be sent in a RPC request.
func Request(sess *clientpb.Session) (req *commonpb.Request) {
	req = &commonpb.Request{}

	if sess != nil {
		req.SessionID = sess.ID
	}

	// The current parser holds some data we want
	var parser = Commands.parser
	if parser == nil {
		return req
	}

	// Get timeout
	if opt := parser.FindOptionByLongName("timeout"); opt != nil {
		// All timeout options are int64
		if val, ok := opt.Value().(int64); ok {
			req.Timeout = val
		}
	}

	return
}
