package gonsole

import (
	"fmt"
	"sync"

	"github.com/jessevdk/go-flags"
	"github.com/maxlandon/readline"
)

// Command - A struct storing basic command info, functions used for command
// instantiation, completion generation, and any number of subcommand groups.
type Command struct {
	// Name - The name of the command, as typed in the shell.
	Name string
	// ShortDescription - A short string to be used in console completions, hints, etc.
	ShortDescription string
	// LongDescription - A longer description text to be printed in the help menus.
	LongDescription string
	// Group - Commands can be specified a group, by which they will appear in completions, etc.
	Group string
	// Filters - A list of filters against which the commands might be shown/hidden.
	// For example, adding the "windows" filter and calling the console.HideCommands("windows"),
	// will hide commands from now on, until console.ShowCommands("windows") is called.
	Filters []string

	// SubcommandsOptional - If this is false, the help usage will be printed for this command.
	SubcommandsOptional bool

	// Data - A function that must yield a pointer to a struct (which is, and will become a command instance)
	// Compatible interfaces must match https://github.com/jessevdk/go-flags.git requirements. Please refer
	// to either the go-flags documentation, or this library's one.
	Data      func() interface{}
	generator func(cParser commandParser, subOptional bool) *flags.Command
	cmd       *flags.Command

	// global options generator. These options are available even when subcommands are being used.
	opts []*optionGroup

	// subcommands
	groups []*commandGroup

	// completions functions, used to match either arguments of this command, or its options.
	argComps        map[string]CompletionFunc
	argCompsDynamic map[string]CompletionFuncDynamic

	optComps        map[string]CompletionFunc
	optCompsDynamic map[string]CompletionFuncDynamic

	// concurrency management
	mutex *sync.RWMutex
}

// NewCommand - Any user wishing to add a command to its application by passing a *Command struct
// directly, should use this constructor, becaus it must initialize a few private fields first.
func NewCommand() *Command {
	c := &Command{
		argComps:        map[string]CompletionFunc{},
		argCompsDynamic: map[string]CompletionFuncDynamic{},
		optComps:        map[string]CompletionFunc{},
		optCompsDynamic: map[string]CompletionFuncDynamic{},
		mutex:           &sync.RWMutex{},
	}
	return c
}

// AddCommand - Add a command to the given command (the console Contexts embed a command for this matter). If you are
// calling this function directly like gonsole.Console.AddCommand(), be aware that this will bind the command to the
// default menu named "". If you don't intend to use multiple menus this is fine, but if you do, you should
// create and name each of your menus, and add commands to them, like Console.NewContext("name").AddCommand("", "", ...)
func (c *Command) AddCommand(name, short, long, group string, filters []string, data func() interface{}) *Command {

	if data == nil {
		return nil
	}

	// Check if the group exists within this menu, or create
	// it and attach to the specificed menu.if needed
	var grp *commandGroup
	for _, g := range c.groups {
		if g.Name == group {
			grp = g
		}
	}
	if grp == nil {
		grp = &commandGroup{Name: group}
		c.mutex.RLock()
		c.groups = append(c.groups, grp)
		c.mutex.RUnlock()
	}

	// Store the interface data in a command spawing funtion, which acts as an instantiator.
	// We use the command's go-flags struct, as opposed to the console root parser.
	var spawner = func(cmdParser commandParser, subOptional bool) *flags.Command {
		cmd, err := cmdParser.AddCommand(name, short, long, data())
		if err != nil {
			fmt.Printf("%s Command bind error:%s %s\n", readline.RED, readline.RESET, err.Error())
		}
		if cmd == nil {
			return nil
		}
		if subOptional {
			cmd.SubcommandsOptional = true
		}
		return cmd
	}

	// Make a new command struct with everything, and store it in the command tree
	command := &Command{
		Name:             name,
		ShortDescription: short,
		LongDescription:  long,
		Group:            group,
		Filters:          filters,
		generator:        spawner,
		argComps:         map[string]CompletionFunc{},
		argCompsDynamic:  map[string]CompletionFuncDynamic{},
		optComps:         map[string]CompletionFunc{},
		optCompsDynamic:  map[string]CompletionFuncDynamic{},
		mutex:            &sync.RWMutex{},
	}

	c.mutex.RLock()
	grp.cmds = append(grp.cmds, command)
	c.mutex.RUnlock()

	return command
}

// GoFlagsCommands - Returns the list of all GO-FLAGS subcommands for this command.
// This means that these commands in the list are temporary ones, they will be respawned
// at the next execution readline loop. Do NOT bind/assign anything to them, it will NOT persist.
func (c *Command) GoFlagsCommands() (cmds []*flags.Command) {
	for _, group := range c.groups {
		for _, cmd := range group.cmds {
			if cmd.cmd != nil {
				cmds = append(cmds, cmd.cmd)
			}
		}
	}
	return
}

// Commands - Returns the list of child gonsole.Commands for this command. You can set
// anything to them, these changes will persist for the lifetime of the application,
// or until you deregister this command or one of its childs.
func (c *Command) Commands() (cmds []*Command) {
	for _, group := range c.groups {
		for _, cmd := range group.cmds {
			cmds = append(cmds, cmd)
		}
	}
	return
}

// CommandGroups - Returns the command's child commands, structured in their respective groups.
// Commands having been assigned no specific group are the group named "".
func (c *Command) CommandGroups() (grps []*commandGroup) {
	return c.groups
}

// getCommandGroup - Get the group for a command.
func (c *Command) getCommandGroup(cmd *flags.Command) string {

	// Sliver commands are searched for if we are in this menu
	for _, group := range c.groups {
		for _, c := range group.cmds {
			if c.Name == cmd.Name {
				// We don't return the name if the command is not generated
				if c.cmd != nil {
					return group.Name
				}
			}
		}
	}
	return ""
}

// OptionGroups - Returns all groups of options that are bound to this command. These
// groups (and their options) are available for use even in the command's child commands.
func (c *Command) OptionGroups() (grps []*optionGroup) {
	return c.opts
}

// Add - Same as AddCommand("", "", ...), but passing a populated Command struct.
func (c *Command) Add(cmd *Command) *Command {
	command := c.AddCommand(cmd.Name, cmd.ShortDescription, cmd.LongDescription, cmd.Group, cmd.Filters, cmd.Data)
	command.SubcommandsOptional = cmd.SubcommandsOptional
	return command
}

// AddCommand - Add a command to the default console menu, named "". Please check gonsole.CurrentContext().AddCommand(),
// if you intend to use multiple menus, for more detailed explanations
func (c *Console) AddCommand(name, short, long, group string, filters []string, data func() interface{}) *Command {
	return c.current.cmd.AddCommand(name, short, long, group, filters, data)
}

// Add - Same as AddCommand("", "", ...), but passing a populated Command struct.
func (c *Console) Add(cmd *Command) *Command {
	command := c.current.AddCommand(cmd.Name, cmd.ShortDescription, cmd.LongDescription, cmd.Group, cmd.Filters, cmd.Data)
	command.SubcommandsOptional = cmd.SubcommandsOptional
	return command
}

// HideCommands - Commands, in addition to their menus, can be shown/hidden based
// on a filter string. For example, some commands applying to a Windows host might
// be scattered around different groups, but, having all the filter "windows".
// If "windows" is used as the argument here, all windows commands for the current
// menu are subsquently hidden, until ShowCommands("windows") is called.
func (c *Console) HideCommands(filter string) {
	for _, f := range c.filters {
		if f == filter {
			return
		}
	}
	c.filters = append(c.filters, filter)
}

// ShowCommands - Commands, in addition to their menus, can be shown/hidden based
// on a filter string. For example, some commands applying to a Windows host might
// be scattered around different groups, but, having all the filter "windows".
// Use this function if you have previously called HideCommands("filter") and want
// these commands to be available back under their respective menu.
func (c *Console) ShowCommands(filter string) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	for i, f := range c.filters {
		if f == filter {
			// Remove the element at index i from a.
			copy(c.filters[i:], c.filters[i+1:])     // Shift a[i+1:] left one index.
			c.filters[len(c.filters)-1] = ""         // Erase last element (write zero value).
			c.filters = c.filters[:len(c.filters)-1] // Truncate slice.
		}
	}
}

// FindCommand - Find a subcommand of this command, given the command name.
func (c *Command) FindCommand(name string) (command *Command) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, group := range c.groups {
		for _, cmd := range group.cmds {
			if cmd.Name == name {
				return cmd
			}
		}
	}
	return
}

// GetCommands - Callers of this are for example the TabCompleter, which needs to call
// this regularly in order to have a list of commands belonging to the current menu.
func (c *Console) GetCommands() (groups map[string][]*flags.Command, groupNames []string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	groups = map[string][]*flags.Command{}

	for _, group := range c.current.cmd.groups {
		groupNames = append(groupNames, group.Name)

		for _, cmd := range group.cmds {
			groups[group.Name] = append(groups[group.Name], cmd.cmd)
		}
	}
	return
}

// FindCommand - Find a command among the root ones in the application, for the current menu.
func (c *Console) FindCommand(name string) (command *Command) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, group := range c.current.cmd.groups {
		for _, cmd := range group.cmds {
			if cmd.Name == name {
				return cmd
			}
		}

	}
	return
}

// bindCommands - At every readline loop, we reinstantiate and bind new instances for
// each command. We do not generate those that are filtered with an active filter,
// so that users of the go-flags parser don't have to perform filtering.
func (c *Console) bindCommands() {
	cc := c.current

	// First, reset the parser for the current menu.
	cc.initParser(c.parserOpts)

	// Generate all global options if there are some.
	for _, opt := range cc.cmd.opts {
		cc.parser.AddGroup(opt.short, opt.long, opt.generator())
	}

	// For each (root) command group in this menu, generate all of its commands,
	// and all of their subcommands recursively. Also generates options, etc.
	for _, group := range cc.cmd.groups {
		c.bindCommandGroup(cc.parser, group)
	}
}
