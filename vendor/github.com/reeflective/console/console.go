package console

import (
	"fmt"
	"strings"
	"sync"

	"github.com/reeflective/readline"
	"github.com/reeflective/readline/inputrc"
)

// Console is an integrated console application instance.
type Console struct {
	// Application
	name        string           // Used in the prompt, and for readline `.inputrc` application-specific settings.
	shell       *readline.Shell  // Provides readline functionality (inputs, completions, hints, history)
	printLogo   func(c *Console) // Simple logo printer.
	menus       map[string]*Menu // Different command trees, prompt engines, etc.
	filters     []string         // Hide commands based on their attributes and current context.
	isExecuting bool             // Used by log functions, which need to adapt behavior (print the prompt, , etc)
	printed     bool             // Used to adjust asynchronous messages too.
	mutex       *sync.RWMutex    // Concurrency management.

	// Execution

	// Leave an empty line before executing the command.
	NewlineBefore bool

	// Leave an empty line after executing the command.
	// Note that if you also want this newline to be used when logging messages
	// with TransientPrintf(), Printf() calls, you should leave this to false,
	// and add a leading newline to your prompt instead: the readline shell will
	// know how to handle it in all situations.
	NewlineAfter bool

	// PreReadlineHooks - All the functions in this list will be executed,
	// in their respective orders, before the console starts reading
	// any user input (ie, before redrawing the prompt).
	PreReadlineHooks []func() error

	// PreCmdRunLineHooks - Same as PreCmdRunHooks, but will have an effect on the
	// input line being ultimately provided to the command parser. This might
	// be used by people who want to apply supplemental, specific processing
	// on the command input line.
	PreCmdRunLineHooks []func(args []string) ([]string, error)

	// PreCmdRunHooks - Once the user has entered a command, but before executing
	// the target command, the console will execute every function in this list.
	// These hooks are distinct from the cobra.PreRun() or OnInitialize hooks,
	// and might be used in combination with them.
	PreCmdRunHooks []func() error

	// PostCmdRunHooks are run after the target cobra command has been executed.
	// These hooks are distinct from the cobra.PreRun() or OnFinalize hooks,
	// and might be used in combination with them.
	PostCmdRunHooks []func() error
}

// New - Instantiates a new console application, with sane but powerful defaults.
// This instance can then be passed around and used to bind commands, setup additional
// things, print asynchronous messages, or modify various operating parameters on the fly.
// The app parameter is an optional name of the application using this console.
func New(app string) *Console {
	console := &Console{
		name:  app,
		shell: readline.NewShell(inputrc.WithApp(strings.ToLower(app))),
		menus: make(map[string]*Menu),
		mutex: &sync.RWMutex{},
	}

	// Quality of life improvements.
	console.setupShell()

	// Make a default menu and make it current.
	// Each menu is created with a default prompt engine.
	defaultMenu := console.NewMenu("")
	defaultMenu.active = true

	// Set the history for this menu
	for _, name := range defaultMenu.historyNames {
		console.shell.History.Add(name, defaultMenu.histories[name])
	}

	// Syntax highlighting, multiline callbacks, etc.
	console.shell.AcceptMultiline = console.acceptMultiline
	console.shell.SyntaxHighlighter = console.highlightSyntax

	// Completion
	console.shell.Completer = console.complete
	console.defaultStyleConfig()

	return console
}

// Shell returns the console readline shell instance, so that the user can
// further configure it or use some of its API for lower-level stuff.
func (c *Console) Shell() *readline.Shell {
	return c.shell
}

// SetPrintLogo - Sets the function that will be called to print the logo.
func (c *Console) SetPrintLogo(f func(c *Console)) {
	c.printLogo = f
}

// NewMenu - Create a new command menu, to which the user
// can attach any number of commands (with any nesting), as
// well as some specific items like history sources, prompt
// configurations, sets of expanded variables, and others.
func (c *Console) NewMenu(name string) *Menu {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	menu := newMenu(name, c)
	c.menus[name] = menu

	return menu
}

// ActiveMenu - Return the currently used console menu.
func (c *Console) ActiveMenu() *Menu {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.activeMenu()
}

// Menu returns one of the console menus by name, or nil if no menu is found.
func (c *Console) Menu(name string) *Menu {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.menus[name]
}

// SwitchMenu - Given a name, the console switches its command menu:
// The next time the console rebinds all of its commands, it will only bind those
// that belong to this new menu. If the menu is invalid, i.e that no commands
// are bound to this menu name, the current menu is kept.
func (c *Console) SwitchMenu(menu string) {
	c.mutex.Lock()
	target, found := c.menus[menu]
	c.mutex.Unlock()

	if found && target != nil {
		// Only switch if the target menu was found.
		current := c.activeMenu()
		if current != nil && target == current {
			return
		}

		if current != nil {
			current.active = false
		}

		target.active = true

		// Remove the currently bound history sources
		// (old menu) and bind the ones peculiar to this one.
		c.shell.History.Delete()

		for _, name := range target.historyNames {
			c.shell.History.Add(name, target.histories[name])
		}

		// Regenerate the commands, outputs and everything related.
		target.resetPreRun()
	}
}

// TransientPrintf prints a string message (a log, or more broadly, an asynchronous event)
// without bothering the user, displaying the message and "pushing" the prompt below it.
// The message is printed regardless of the current menu.
//
// If this function is called while a command is running, the console will simply print the log
// below the line, and will not print the prompt. In any other case this function works normally.
func (c *Console) TransientPrintf(msg string, args ...any) (n int, err error) {
	if c.isExecuting {
		return fmt.Printf(msg, args...)
	}

	// If the last message we printed asynchronously
	// immediately precedes this new message, move up
	// another row, so we don't waste too much space.
	if c.printed && c.NewlineAfter {
		fmt.Print("\x1b[1A")
	}

	if c.NewlineAfter {
		msg += "\n"
	}

	c.printed = true

	return c.shell.PrintTransientf(msg, args...)
}

// Printf prints a string message (a log, or more broadly, an asynchronous event)
// below the current prompt. The message is printed regardless of the current menu.
//
// If this function is called while a command is running, the console will simply print the log
// below the line, and will not print the prompt. In any other case this function works normally.
func (c *Console) Printf(msg string, args ...any) (n int, err error) {
	if c.isExecuting {
		return fmt.Printf(msg, args...)
	}

	return c.shell.Printf(msg, args...)
}

// SystemEditor - This function is a renamed-reexport of the underlying readline.StartEditorWithBuffer
// function, which enables you to conveniently edit files/buffers from within the console application.
// Naturally, the function will block until the editor is exited, and the updated buffer is returned.
// The filename parameter can be used to pass a specific filename.ext pattern, which might be useful
// if the editor has builtin filetype plugin functionality.
func (c *Console) SystemEditor(buffer []byte, filetype string) ([]byte, error) {
	emacs := c.shell.Config.GetString("editing-mode") == "emacs"

	edited, err := c.shell.Buffers.EditBuffer([]rune(string(buffer)), "", filetype, emacs)

	return []byte(string(edited)), err
}

func (c *Console) setupShell() {
	cfg := c.shell.Config

	// Some options should be set to on because they
	// are quite neceessary for efficient console use.
	cfg.Set("skip-completed-text", true)
	cfg.Set("menu-complete-display-prefix", true)
}

func (c *Console) activeMenu() *Menu {
	for _, menu := range c.menus {
		if menu.active {
			return menu
		}
	}

	// Else return the default menu.
	return c.menus[""]
}
