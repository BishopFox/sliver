package main

import (
	"fmt"
	"strings"

	"github.com/jessevdk/go-flags"

	"github.com/maxlandon/readline"
	"github.com/maxlandon/readline/completers"
)

// This file shows a typical way of using readline in a loop.

func main() {
	// Instantiate a console object
	console := newConsole()

	// Bind commands to the console
	bindCommands()

	// Setup the console completers, prompts, and input modes
	console.setup()

	// Start the readline loop (blocking)
	console.Start()
}

// newConsole - Instantiates a new console with some default behavior.
// We modify/add elements of behavior later in setup.
func newConsole() *console {
	console := &console{
		shell:  readline.NewInstance(),
		parser: commandParser,
	}
	return console
}

// console - A simple console example.
type console struct {
	shell  *readline.Instance
	parser *flags.Parser
}

// setup - The console sets up various elements such as the completion system, hints,
// syntax highlighting, prompt system, commands binding, and client environment loading.
func (c *console) setup() (err error) {

	// Input mode & defails
	c.shell.InputMode = readline.Vim // Could be readline.Emacs for emacs input mode.
	c.shell.ShowVimMode = true
	c.shell.VimModeColorize = true

	// Prompt: we want a two-line prompt, with a custom indicator after the Vim status
	c.shell.SetPrompt("readline ")
	c.shell.Multiline = true
	c.shell.MultilinePrompt = " > "

	// Instantiate a default completer associated with the parser
	// declared in commands.go, and embedded into the console struct.
	// The error is muted, because we don't pass an nil parser, therefore no problems.
	defaultCompleter, _ := completers.NewCommandCompleter(c.parser)

	// Register the completer for command/option completions, hints and syntax highlighting.
	// The completer can handle all of them.
	c.shell.TabCompleter = defaultCompleter.TabCompleter
	c.shell.HintText = defaultCompleter.HintCompleter
	c.shell.SyntaxHighlighter = defaultCompleter.SyntaxHighlighter

	// History: by default the history is in-memory, use it with Ctrl-R

	return
}

// Start - The console has a working RPC connection: we setup all
// things pertaining to the console itself, and start the input loop.
func (c *console) Start() (err error) {

	// Setup console elements
	err = c.setup()
	if err != nil {
		return fmt.Errorf("Console setup failed: %s", err)
	}

	// Start input loop
	for {
		// Read input line
		line, _ := c.Readline()

		// Split and sanitize input
		sanitized, empty := sanitizeInput(line)
		if empty {
			continue
		}

		// Process various tokens on input (environment variables, paths, etc.)
		// These tokens will be expaneded by completers anyway, so this is not absolutely required.
		envParsed, _ := completers.ParseEnvironmentVariables(sanitized)

		// Other types of tokens, needed by commands who expect a certain type
		// of arguments, such as paths with spaces.
		tokenParsed := c.parseTokens(envParsed)

		// Execute the command and print any errors
		if _, parserErr := c.parser.ParseArgs(tokenParsed); parserErr != nil {
			fmt.Println(readline.RED + "[Error] " + readline.RESET + parserErr.Error() + "\n")
		}
	}
}

// Readline - Add an empty line between input line and command output.
func (c *console) Readline() (line string, err error) {
	line, err = c.shell.Readline()
	fmt.Println()
	return
}

// sanitizeInput - Trims spaces and other unwished elements from the input line.
func sanitizeInput(line string) (sanitized []string, empty bool) {

	// Assume the input is not empty
	empty = false

	// Trim border spaces
	trimmed := strings.TrimSpace(line)
	if len(line) < 1 {
		empty = true
		return
	}
	unfiltered := strings.Split(trimmed, " ")

	// Catch any eventual empty items
	for _, arg := range unfiltered {
		if arg != "" {
			sanitized = append(sanitized, arg)
		}
	}
	return
}

// parseTokens - Parse and process any special tokens that are not treated by environment-like parsers.
func (c *console) parseTokens(sanitized []string) (parsed []string) {

	// PATH SPACE TOKENS
	// Catch \ tokens, which have been introduced in paths where some directories have spaces in name.
	// For each of these splits, we concatenate them with the next string.
	// This will also inspect commands/options/arguments, but there is no reason why a backlash should be present in them.
	var pathAdjusted []string
	var roll bool
	var arg string
	for i := range sanitized {
		if strings.HasSuffix(sanitized[i], "\\") {
			// If we find a suffix, replace with a space. Go on with next input
			arg += strings.TrimSuffix(sanitized[i], "\\") + " "
			roll = true
		} else if roll {
			// No suffix but part of previous input. Add it and go on.
			arg += sanitized[i]
			pathAdjusted = append(pathAdjusted, arg)
			arg = ""
			roll = false
		} else {
			// Default, we add our path and go on.
			pathAdjusted = append(pathAdjusted, sanitized[i])
		}
	}
	parsed = pathAdjusted

	// Add new function here, act on parsed []string from now on, not sanitized
	return
}
