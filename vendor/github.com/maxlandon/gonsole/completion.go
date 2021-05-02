package gonsole

import (
	"github.com/maxlandon/readline"
)

// Console Context Expansion Completions ----------------------------------------------------------------------

// AddExpansionCompletion - Add a completion generator that is triggered when the expansion
// string paramater is detected (anywhere, even in other completions) in the input line.
// Ex: you can pass '$' as an expansion, and a function that will yield environment variables.
func (c *Menu) AddExpansionCompletion(expansion rune, comps CompletionFunc) {

	// Completer
	c.expansionComps[expansion] = comps

	// Add default token highlighter if the config has not one for it.
	if _, found := c.console.config.Highlighting[string(expansion)]; !found {
		c.console.config.Highlighting[string(expansion)] = "{g}"
	}
}

// Static Completion generators ------------------------------------------------------------------------------

// CompletionFunc - Function yielding one or more completions groups.
// The user of this function does not have to worry about any prefix,
// he just has to yield values into a CompletionGroup, and set any
// option/behavior to it if wished.
type CompletionFunc func() (comps []*readline.CompletionGroup)

// AddArgumentCompletion - Given a registered command, add one or more groups of completion items
// (with any display style/options) to one of the command's arguments. Does not need to return a prefix.
// It is VERY IMPORTANT to pass the case-sensitive name of the argument, as declared in the command struct.
// The type of the underlying argument does not matter, and gonsole will correctly yield suggestions based
// on wheteher list are required, are these arguments optional, etc.
func (c *Command) AddArgumentCompletion(arg string, comps CompletionFunc) {
	c.argComps[arg] = comps
}

// AddOptionCompletion - Given a registered command and an option LONG name, add one or
// more groups of completion items to this option's arguments. Does not need to return a prefix.
// It is VERY IMPORTANT to pass the case-sensitive name of the option, as declared in the command struct.
// The type of the underlying argument does not matter, and gonsole will correctly yield suggestions based
// on wheteher list are required, are these arguments optional, etc.
func (c *Command) AddOptionCompletion(arg string, comps CompletionFunc) {
	c.optComps[arg] = comps
}

// Dynamic Completion generators ------------------------------------------------------------------------------

// CompletionFuncDynamic - A function that yields one or more completion groups.
// The prefix parameter should be used with a simple 'if strings.IsPrefix()' condition.
// Please see the project wiki for documentation on how to write more elaborated engines:
// For example, do NOT use or modify the `pref string` return paramater if you don't explicitely need to.
type CompletionFuncDynamic func(prefix string) (pref string, comps []*readline.CompletionGroup)

// AddArgumentCompletionDynamic - Given a registered command, add one or more groups of completion items
// (with any display style/options) to one of the command's arguments. Needs to return a prefix in the compfunc.
// It is VERY IMPORTANT to pass the case-sensitive name of the argument, as declared in the command struct.
// The type of the underlying argument does not matter, and gonsole will correctly yield suggestions based
// on wheteher list are required, are these arguments optional, etc.
// The menu is needed in order to bind these completions to the good command,
// because several menus migh have some being identically named.
func (c *Command) AddArgumentCompletionDynamic(arg string, comps CompletionFuncDynamic) {
	c.argCompsDynamic[arg] = comps
}

// AddOptionCompletionDynamic - Given a registered command and an option LONG name, add one or
// more groups of completion items to this option's arguments.
// It is VERY IMPORTANT to pass the case-sensitive name of the option, as declared in the command struct.
// The type of the underlying argument does not matter, and gonsole will correctly yield suggestions based
// on wheteher list are required, are these arguments optional, etc.
// The menu is needed in order to bind these completions to the good command,
// because several menus migh have some being identically named.
func (c *Command) AddOptionCompletionDynamic(option string, comps CompletionFuncDynamic) {
	c.optCompsDynamic[option] = comps
}
