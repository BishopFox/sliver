package gonsole

import (
	"github.com/jessevdk/go-flags"
	"github.com/maxlandon/readline"
)

// Menu - A menu is a simple way to seggregate commands based on
// the environment to which they belong. For instance, when using a menu
// specific to some host/user, or domain of activity, commands will vary.
type Menu struct {
	Name   string  // The name of the context, used for many things here and there.
	Prompt *Prompt // A dedicated prompt with its own callbacks and colors

	// UnknownCommandHandler - The user can specify a function that will
	// be executed if the error raised by the application parser is a
	// ErrUnknownCommand error. This might be used for executing the
	// input line directly via a system shell, or any os.Exec mean...
	UnknownCommandHandler func(args []string) error

	// Each menu has its own command parser, which executes dispatched commands
	parser *flags.Parser

	// Command - The menu embeds a command so that users
	// can more explicitly register commands to a given menu.
	cmd *Command

	// Each menu can have two specific history sources
	historyCtrlRName string
	historyCtrlR     readline.History
	historyAltRName  string
	historyAltR      readline.History

	// expansionComps - A list of completion generators that are triggered when
	// the given string is detected (anywhere, even in other completions) in the input line.
	expansionComps map[rune]CompletionFunc

	// The menu sometimes needs access to some console state.
	console *Console
}

func newMenu(c *Console) *Menu {
	menu := &Menu{
		Prompt: &Prompt{
			Callbacks: map[string]func() string{},
			Colors:    defaultColorCallbacks,
			console:   c,
		},
		cmd:            NewCommand(),
		expansionComps: map[rune]CompletionFunc{},
		console:        c,
	}
	return menu
}

// PromptConfig - Returns the prompt object used to setup the prompt. It is actually
// a configuration, because it can also be printed and exported by a config command.
func (m *Menu) PromptConfig() *PromptConfig {
	return m.console.config.Prompts[m.Name]
}

// Commands - Returns the list of child gonsole.Commands for this command. You can set
// anything to them, these changes will persist for the lifetime of the application,
// or until you deregister this command or one of its childs.
func (m *Menu) Commands() (cmds []*Command) {
	return m.cmd.Commands()
}

// CommandGroups - Returns the command's child commands, structured in their respective groups.
// Commands having been assigned no specific group are the group named "".
func (m *Menu) CommandGroups() (grps []*commandGroup) {
	return m.cmd.groups
}

// OptionGroups - Returns all groups of options that are bound to this command. These
// groups (and their options) are available for use even in the command's child commands.
func (m *Menu) OptionGroups() (grps []*optionGroup) {
	return m.cmd.opts
}

// AddGlobalOptions - Add global options for this menu command parser. Will appear in all commands.
func (m *Menu) AddGlobalOptions(shortDescription, longDescription string, data func() interface{}) {
	m.cmd.AddGlobalOptions(shortDescription, longDescription, data)
}

// AddCommand - Add a command to this menu. This command will be available when this menu is active.
func (m *Menu) AddCommand(name, short, long, group string, filters []string, data func() interface{}) *Command {
	return m.cmd.AddCommand(name, short, long, group, filters, data)
}

// SetHistoryCtrlR - Set the history source triggered with Ctrl-R
func (m *Menu) SetHistoryCtrlR(name string, hist readline.History) {
	m.historyCtrlRName = name
	m.historyCtrlR = hist
}

// SetHistoryAltR - Set the history source triggered with Alt-r
func (m *Menu) SetHistoryAltR(name string, hist readline.History) {
	m.historyAltRName = name
	m.historyAltR = hist
}

// initParser - Called each time the readline loops, before rebinding all command instances.
func (m *Menu) initParser(opts flags.Options) {
	m.parser = flags.NewNamedParser(m.Name, opts)
}

// NewMenu - Create a new command menu, to which the user
// can attach any number of commands (with any nesting), as
// well as some specific items like history sources, prompt
// configurations, sets of expanded variables, and others.
func (c *Console) NewMenu(name string) (ctx *Menu) {
	c.mutex.RLock()
	ctx = newMenu(c)
	ctx.Name = name
	c.menus[name] = ctx
	c.mutex.RUnlock()

	// Load default prompt configuration
	c.config.Prompts[ctx.Name] = newDefaultPromptConfig(ctx.Name)
	return
}

// GetMenu - Given a name, return the appropriate menu.
// If the menu does not exists, it returns nil
func (c *Console) GetMenu(name string) (ctx *Menu) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if menu, exists := c.menus[name]; exists {
		return menu
	}
	return
}

// SwitchMenu - Given a name, the console switches its command menu:
// The next time the console rebinds all of its commands, it will only bind those
// that belong to this new menu. If the menu is invalid, i.e that no commands
// are bound to this menu name, the current menu is kept.
func (c *Console) SwitchMenu(menu string) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	for _, ctx := range c.menus {
		if ctx.Name == menu {
			c.current = ctx
		}
	}

	// Contexts have some specific configuration values, reload them.
	c.reloadConfig()

	// Bind history sources
	c.shell.SetHistoryCtrlR(c.current.historyCtrlRName, c.current.historyCtrlR)
	c.shell.SetHistoryAltR(c.current.historyAltRName, c.current.historyAltR)
}

// CurrentMenu - Return the current console menu. Because the Context
// is just a reference, any modifications to this menu will persist.
func (c *Console) CurrentMenu() *Menu {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.current
}

// AddGlobalOptions - Add global options for this menu command parser. Will appear in all commands.
func (c *Console) AddGlobalOptions(shortDescription, longDescription string, data func() interface{}) {
	c.menus[""].cmd.AddGlobalOptions(shortDescription, longDescription, data)
}

// SetHistoryCtrlR - Set the history source triggered with Ctrl-R
func (c *Console) SetHistoryCtrlR(name string, hist readline.History) {
	c.menus[""].historyCtrlRName = name
	c.menus[""].historyCtrlR = hist
}

// SetHistoryAltR - Set the history source triggered with Alt-r
func (c *Console) SetHistoryAltR(name string, hist readline.History) {
	c.menus[""].historyAltRName = name
	c.menus[""].historyAltR = hist
}
