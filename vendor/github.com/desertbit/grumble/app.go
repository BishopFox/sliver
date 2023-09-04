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
	"io"
	"os"
	"strings"

	"github.com/desertbit/closer/v3"
	shlex "github.com/desertbit/go-shlex"
	"github.com/desertbit/readline"
	"github.com/fatih/color"
)

// App is the entrypoint.
type App struct {
	closer.Closer

	rl            *readline.Instance
	config        *Config
	commands      Commands
	isShell       bool
	currentPrompt string

	flags   Flags
	flagMap FlagMap

	args Args

	initHook  func(a *App, flags FlagMap) error
	shellHook func(a *App) error

	printHelp        func(a *App, shell bool)
	printCommandHelp func(a *App, cmd *Command, shell bool)
	interruptHandler func(a *App, count int)
	printASCIILogo   func(a *App)
}

// New creates a new app.
// Panics if the config is invalid.
func New(c *Config) (a *App) {
	// Prepare the config.
	c.SetDefaults()
	err := c.Validate()
	if err != nil {
		panic(err)
	}

	// APP.
	a = &App{
		Closer:           closer.New(),
		config:           c,
		currentPrompt:    c.prompt(),
		flagMap:          make(FlagMap),
		printHelp:        defaultPrintHelp,
		printCommandHelp: defaultPrintCommandHelp,
		interruptHandler: defaultInterruptHandler,
	}

	// Register the builtin flags.
	a.flags.Bool("h", "help", false, "display help")
	a.flags.BoolL("nocolor", false, "disable color output")

	// Register the user flags, if present.
	if c.Flags != nil {
		c.Flags(&a.flags)
	}

	return
}

// SetPrompt sets a new prompt.
func (a *App) SetPrompt(p string) {
	if !a.config.NoColor {
		p = a.config.PromptColor.Sprint(p)
	}
	a.currentPrompt = p
}

// SetDefaultPrompt resets the current prompt to the default prompt as
// configured in the config.
func (a *App) SetDefaultPrompt() {
	a.currentPrompt = a.config.prompt()
}

// IsShell indicates, if this is a shell session.
func (a *App) IsShell() bool {
	return a.isShell
}

// Config returns the app's config value.
func (a *App) Config() *Config {
	return a.config
}

// Commands returns the app's commands.
// Access is not thread-safe. Only access during command execution.
func (a *App) Commands() *Commands {
	return &a.commands
}

// PrintError prints the given error.
func (a *App) PrintError(err error) {
	if a.config.NoColor {
		a.Printf("error: %v\n", err)
	} else {
		a.config.ErrorColor.Fprint(a, "error: ")
		a.Printf("%v\n", err)
	}
}

// Print writes to terminal output.
// Print writes to standard output if terminal output is not yet active.
func (a *App) Print(args ...interface{}) (int, error) {
	return fmt.Fprint(a, args...)
}

// Printf formats according to a format specifier and writes to terminal output.
// Printf writes to standard output if terminal output is not yet active.
func (a *App) Printf(format string, args ...interface{}) (int, error) {
	return fmt.Fprintf(a, format, args...)
}

// Println writes to terminal output followed by a newline.
// Println writes to standard output if terminal output is not yet active.
func (a *App) Println(args ...interface{}) (int, error) {
	return fmt.Fprintln(a, args...)
}

// OnInit sets the function which will be executed before the first command
// is executed. App flags can be handled here.
func (a *App) OnInit(f func(a *App, flags FlagMap) error) {
	a.initHook = f
}

// OnShell sets the function which will be executed before the shell starts.
func (a *App) OnShell(f func(a *App) error) {
	a.shellHook = f
}

// SetInterruptHandler sets the interrupt handler function.
func (a *App) SetInterruptHandler(f func(a *App, count int)) {
	a.interruptHandler = f
}

// SetPrintHelp sets the print help function.
func (a *App) SetPrintHelp(f func(a *App, shell bool)) {
	a.printHelp = f
}

// SetPrintCommandHelp sets the print help function for a single command.
func (a *App) SetPrintCommandHelp(f func(a *App, c *Command, shell bool)) {
	a.printCommandHelp = f
}

// SetPrintASCIILogo sets the function to print the ASCII logo.
func (a *App) SetPrintASCIILogo(f func(a *App)) {
	a.printASCIILogo = func(a *App) {
		if !a.config.NoColor {
			a.config.ASCIILogoColor.Set()
			defer color.Unset()
		}
		f(a)
	}
}

// Write to the underlying output, using readline if available.
func (a *App) Write(p []byte) (int, error) {
	return a.Stdout().Write(p)
}

// Stdout returns a writer to Stdout, using readline if available.
// Note that calling before Run() will return a different instance.
func (a *App) Stdout() io.Writer {
	if a.rl != nil {
		return a.rl.Stdout()
	}
	return os.Stdout
}

// Stderr returns a writer to Stderr, using readline if available.
// Note that calling before Run() will return a different instance.
func (a *App) Stderr() io.Writer {
	if a.rl != nil {
		return a.rl.Stderr()
	}
	return os.Stderr
}

// AddCommand adds a new command.
// Panics on error.
func (a *App) AddCommand(cmd *Command) {
	a.addCommand(cmd, true)
}

// addCommand adds a new command.
// If addHelpFlag is true, a help flag is automatically
// added to the command which displays its usage on use.
// Panics on error.
func (a *App) addCommand(cmd *Command, addHelpFlag bool) {
	err := cmd.validate()
	if err != nil {
		panic(err)
	}
	cmd.registerFlagsAndArgs(addHelpFlag)

	a.commands.Add(cmd)
}

// RunCommand runs a single command.
func (a *App) RunCommand(args []string) error {
	// Parse the arguments string and obtain the command path to the root,
	// and the command flags.
	cmds, fg, args, err := a.commands.parse(args, a.flagMap, false)
	if err != nil {
		return err
	} else if len(cmds) == 0 {
		return fmt.Errorf("unknown command, try 'help'")
	}

	// The last command is the final command.
	cmd := cmds[len(cmds)-1]

	// Print the command help if the command run function is nil or if the help flag is set.
	if fg.Bool("help") || cmd.Run == nil {
		a.printCommandHelp(a, cmd, a.isShell)
		return nil
	}

	// Parse the arguments.
	cmdArgMap := make(ArgMap)
	args, err = cmd.args.parse(args, cmdArgMap)
	if err != nil {
		return err
	}

	// Check, if values from the argument string are not consumed (and therefore invalid).
	if len(args) > 0 {
		return fmt.Errorf("invalid usage of command '%s' (unconsumed input '%s'), try 'help'", cmd.Name, strings.Join(args, " "))
	}

	// Create the context and pass the rest args.
	ctx := newContext(a, cmd, fg, cmdArgMap)

	// Run the command.
	err = cmd.Run(ctx)
	if err != nil {
		return err
	}

	return nil
}

// Run the application and parse the command line arguments.
// This method blocks.
func (a *App) Run() (err error) {
	defer a.Close()

	// Sort all commands by their name.
	a.commands.SortRecursive()

	// Remove the program name from the args.
	args := os.Args
	if len(args) > 0 {
		args = args[1:]
	}

	// Parse the app command line flags.
	args, err = a.flags.parse(args, a.flagMap)
	if err != nil {
		return err
	}

	// Check if nocolor was set.
	a.config.NoColor = a.flagMap.Bool("nocolor")

	// Determine if this is a shell session.
	a.isShell = len(args) == 0

	// Add general builtin commands.
	a.addCommand(&Command{
		Name: "help",
		Help: "use 'help [command]' for command help",
		Args: func(a *Args) {
			a.StringList("command", "the name of the command")
		},
		Run: func(c *Context) error {
			args := c.Args.StringList("command")
			if len(args) == 0 {
				a.printHelp(a, a.isShell)
				return nil
			}
			cmd, _, err := a.commands.FindCommand(args)
			if err != nil {
				return err
			} else if cmd == nil {
				a.PrintError(fmt.Errorf("command not found"))
				return nil
			}
			a.printCommandHelp(a, cmd, a.isShell)
			return nil
		},
		isBuiltin: true,
	}, false)

	// Check if help should be displayed.
	if a.flagMap.Bool("help") {
		a.printHelp(a, false)
		return nil
	}

	// Add shell builtin commands.
	// Ensure to add all commands before running the init hook.
	// If the init hook does something with the app commands, then these should also be included.
	if a.isShell {
		a.AddCommand(&Command{
			Name: "exit",
			Help: "exit the shell",
			Run: func(c *Context) error {
				c.Stop()
				return nil
			},
			isBuiltin: true,
		})
		a.AddCommand(&Command{
			Name: "clear",
			Help: "clear the screen",
			Run: func(c *Context) error {
				readline.ClearScreen(a.rl)
				return nil
			},
			isBuiltin: true,
		})
	}

	// Run the init hook.
	if a.initHook != nil {
		err = a.initHook(a, a.flagMap)
		if err != nil {
			return err
		}
	}

	// Check if a command chould be executed in non-interactive mode.
	if !a.isShell {
		return a.RunCommand(args)
	}

	// Create the readline instance.
	a.rl, err = readline.NewEx(&readline.Config{
		Prompt:                 a.currentPrompt,
		HistorySearchFold:      true, // enable case-insensitive history searching
		DisableAutoSaveHistory: true,
		HistoryFile:            a.config.HistoryFile,
		HistoryLimit:           a.config.HistoryLimit,
		AutoComplete:           newCompleter(&a.commands),
	})
	if err != nil {
		return err
	}
	a.OnClose(a.rl.Close)

	// Run the shell hook.
	if a.shellHook != nil {
		err = a.shellHook(a)
		if err != nil {
			return err
		}
	}

	// Print the ASCII logo.
	if a.printASCIILogo != nil {
		a.printASCIILogo(a)
	}

	// Run the shell.
	return a.runShell()
}

func (a *App) runShell() error {
	var interruptCount int
	var lines []string
	multiActive := false

Loop:
	for !a.IsClosing() {
		// Set the prompt.
		if multiActive {
			a.rl.SetPrompt(a.config.multiPrompt())
		} else {
			a.rl.SetPrompt(a.currentPrompt)
		}
		multiActive = false

		// Readline.
		line, err := a.rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				interruptCount++
				a.interruptHandler(a, interruptCount)
				continue Loop
			} else if err == io.EOF {
				return nil
			} else {
				return err
			}
		}

		// Reset the interrupt count.
		interruptCount = 0

		// Handle multiline input.
		if strings.HasSuffix(line, "\\") {
			multiActive = true
			line = strings.TrimSpace(line[:len(line)-1]) // Add without suffix and trim spaces.
			lines = append(lines, line)
			continue Loop
		}
		lines = append(lines, strings.TrimSpace(line))

		line = strings.Join(lines, " ")
		line = strings.TrimSpace(line)
		lines = lines[:0]

		// Skip if the line is empty.
		if len(line) == 0 {
			continue Loop
		}

		// Save command history.
		err = a.rl.SaveHistory(line)
		if err != nil {
			a.PrintError(err)
			continue Loop
		}

		// Split the line to args.
		args, err := shlex.Split(line, true)
		if err != nil {
			a.PrintError(fmt.Errorf("invalid args: %v", err))
			continue Loop
		}

		// Execute the command.
		err = a.RunCommand(args)
		if err != nil {
			a.PrintError(err)
			// Do not continue the Loop here. We want to handle command changes below.
		}

		// Sort the commands again if they have changed (Add or remove action).
		if a.commands.hasChanged() {
			a.commands.SortRecursive()
			a.commands.unsetChanged()
		}
	}

	return nil
}
