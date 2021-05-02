package gonsole

import (
	"sync"

	"github.com/jessevdk/go-flags"
	"github.com/maxlandon/readline"
)

// Console - An integrated console instance.
type Console struct {
	// shell - The underlying shell provides the core readline functionality,
	// including but not limited to: inputs, completions, hints, history.
	shell *readline.Instance

	// Completer - The completion engine is available to the user for registering
	// default completion generators. A list of them is available to be bound
	// to either or both command/option argument completions. Console menus
	// are not relevant here, the user should not worry.
	Completer *CommandCompleter

	// PreLoopHooks - All the functions in this list will be executed,
	// in their respective orders, before the console starts reading
	// any user input (ie, before redrawing the prompt).
	PreLoopHooks []func()

	// PreRunHooks - Once the user has entered a command, but before executing it
	// with the application go-flags parser, the console will execute every func
	// in this list.
	PreRunHooks []func()

	// PreRunLineHooks - Same as PreRunHooks, but will have an effect on the
	// input line being ultimately provided to the command parser. This might
	// be used by people who want to apply supplemental, specific processing
	// on the command input line.
	PreRunLineHooks []func(raw []string) (args []string, err error)

	// If true, leavs a newline between command line input and their output.
	LeaveNewline     bool
	PreOutputNewline bool

	// Contexts - The various menus hold a list of command instantiators
	// structured by groups. These groups are needed for completions and helps.
	menus   map[string]*Menu
	current *Menu // The name of the current menu

	// parser - Contains the whole aspect of command registering, parsing,
	// processing, and execution. There is only one parser at a time,
	// because it is recreated & repopulated at each console execution loop.
	parserOpts flags.Options

	// A list of tags by which commands may have been registered, and which
	// can be set to true in order to hide all of the tagged commands.
	filters []string

	// True if the console is currently running a command. This is used by
	// the various asynchronous log/message functions, which need to adapt their
	// behavior (do we reprint the prompt, where, etc) based on this.
	isExecuting bool

	// config - Holds all configuration elements for all menus (input mode,
	// prompt strings and setups, hints, etc)
	config            *Config
	configCommandName string

	// concurrency management.
	mutex *sync.RWMutex
}

// NewConsole - Instantiates a new console application, with sane but powerful defaults.
// This instance can then be passed around and used to bind commands, setup additional
// things, print asynchronous messages, or modify various operating parameters on the fly.
func NewConsole() (c *Console) {

	c = &Console{
		menus: map[string]*Menu{},
		mutex: &sync.RWMutex{},
	}

	// Default configuration
	c.loadDefaultConfig()

	// Setup the readline instance
	c.shell = readline.NewInstance()

	// Input mode
	if c.config.InputMode == InputEmacs {
		c.shell.InputMode = readline.Emacs
	} else {
		c.shell.InputMode = readline.Vim
	}
	c.shell.ShowVimMode = true
	c.shell.VimModeColorize = true

	// Default menu, "" (empty name)
	c.current = c.NewMenu("")

	// Setup completers, hints, etc. We pass 2 functions as parameters,
	// so that the engine can query the commands for the currently active menu.
	engine := newCommandCompleter(c)

	c.shell.TabCompleter = engine.tabCompleter
	c.shell.MaxTabCompleterRows = c.config.MaxTabCompleterRows
	c.shell.HintText = engine.hintCompleter
	c.shell.SyntaxHighlighter = engine.syntaxHighlighter

	// Available to the user for default completers.
	c.Completer = engine

	// Setup the prompt (all menus)
	c.shell.MultilinePrompt = c.config.Prompts[c.current.Name].MultilinePrompt
	c.shell.Multiline = c.config.Prompts[c.current.Name].Multiline
	c.current.Prompt.loadFromConfig(c.config.Prompts[c.current.Name])

	// Setup CtrlR history with an in-memory one by default
	c.current.SetHistoryCtrlR("client history (in-memory)", new(readline.ExampleHistory))

	// Set default options for the parser
	c.parserOpts = flags.HelpFlag | flags.IgnoreUnknown

	return
}
