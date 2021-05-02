package gonsole

import (
	"fmt"
	"strings"

	ansi "github.com/acarl005/stripansi"

	"github.com/maxlandon/readline"
)

// Prompt - Computes all prompts used on the shell for a given menu.
// You can register two sorts of callbacks to it, so you can give
// customized prompt elements to be used by the end user.
type Prompt struct {
	// Callbacks - A list of value callbacks to be used, preferably of the following form:
	// "{key}": func() "value". (notice the brackets). Each of callbacks found
	// in the prompt strings will be replaced by the function they're mapped to.
	Callbacks map[string]func() string
	// Colors - A more optional feature, because this console library automatically
	// populates it with a few callback colors, coming from evilsocket's libs.
	// Please also use brackets (though not mandatory): "{key}": "value".
	Colors map[string]string

	left    string // The leftmost prompt
	right   string // The rightmost prompt, currently same line as left.
	newline bool   // If true, leaves a new line before showing command output.

	console *Console
}

// RefreshPromptLog - A simple function to print a string message (a log, or more broadly,
// an asynchronous event) without bothering the user, and by "pushing" the prompt below the message.
// If this function is called while a command is running, the console will simply print the log
// below the current line, and will not print the prompt. In any other case this function will work normally.
func (c *Console) RefreshPromptLog(log string) {
	if c.isExecuting {
		fmt.Print(log)
	} else {
		c.shell.RefreshPromptLog(log)
	}
}

// RefreshPromptCustom - Refresh the console prompt with custom values. This works differently from
// RefreshPromptLog, in that it does mandatorily erases the current (full) prompt if offset to one.
// However, like RefreshPromptLog, it will not reprint the prompt if the console is currently executing a command.
//
// @log         => An optional log message to print before refreshing the prompt. Does nothing if nil
// @prompt      => If not nil (""), will use this prompt instead of the currently set prompt.
// @offset      => Used to set the number of lines to go upward, before reprinting. Set to 0 if not used.
func (c *Console) RefreshPromptCustom(log string, prompt string, offset int) {
	if c.isExecuting {
		fmt.Print(log)
	} else {
		fmt.Print(log)
		c.shell.RefreshPromptCustom(prompt, offset, false)
	}
}

// RefreshPromptInPlace - Refreshes the prompt in the very same place he is, with a string
// that will list only until the next execution loop. This might be used in conjunction with
// c.Menu["myMenu"].Prompt.Render(). Like other Refresh functions, it will not reprint the
// prompt if the console is currently executing a command.
func (c *Console) RefreshPromptInPlace(prompt string) {
	if !c.isExecuting {
		c.shell.RefreshPromptInPlace(prompt)
	}
}

// Gathers all per-execution-loop refresh and synchronization that needs to occur
// betwee the application, the readline instance, the context prompt and the config.
func (p *Prompt) refreshPromptSettings() {
	conf := p.console.config
	current := p.console.current
	if _, exist := conf.Prompts[current.Name]; !exist {
		conf.Prompts[current.Name] = newDefaultPromptConfig(current.Name)
	}

	// Load the configuration
	p.loadFromConfig(conf.Prompts[current.Name])

	// Apply to underlying readline lib.
	p.console.shell.SetPrompt(p.Render())
	p.console.shell.Multiline = current.PromptConfig().Multiline
	p.console.shell.MultilinePrompt = current.PromptConfig().MultilinePrompt
}

// Render - The core prompt computes all necessary values, forges a prompt string
// and returns it for being printed by the shell. You might need to access it if
// you want to tinker with it while using one the console.RefreshPrompt() functions.
func (p *Prompt) Render() (prompt string) {

	// We need the terminal width: the prompt sometimes
	// makes use of both sides for different items.
	sWidth := readline.GetTermWidth()

	// Compute all prompt parts independently
	left, bWidth := p.computeCallbacks(p.left)
	right, cWidth := p.computeCallbacks(p.right)

	// Return the left prompt if we don't have multiline prompt.
	if !p.console.config.Prompts[p.console.current.Name].Multiline {
		return left
	}

	// Verify that the length of all combined prompt elements is not wider than
	// determined terminal width. If yes, truncate the prompt string accordingly.
	if bWidth+cWidth > sWidth {
		// m.Module = truncate()
	}

	// Get the empty part of the prompt and pad accordingly.
	pad := getPromptPad(sWidth, bWidth, cWidth)

	// Finally, forge the complete prompt string
	prompt = left + pad + right

	// Don't mess with input line colors
	prompt += readline.RESET

	return
}

// computeBase - Computes the base prompt (left-side) with potential custom prompt given.
// Returns the width of the computed string, for correct aggregation of all strings.
func (p *Prompt) computeCallbacks(raw string) (ps string, width int) {
	ps = raw

	// Compute callback values
	for ok, cb := range p.Callbacks {
		ps = strings.Replace(ps, ok, cb(), 1)
	}
	for tok, color := range p.Colors {
		ps = strings.Replace(ps, tok, color, -1)
	}

	width = getRealLength(ps)

	return
}

// The prompt takes its values from the configuration. This allows to have
// a synchronized/actualized configuration file to export at any time.
func (p *Prompt) loadFromConfig(promptConf *PromptConfig) {
	if promptConf == nil {
		return
	}
	p.left = promptConf.Left
	p.right = promptConf.Right
	p.newline = promptConf.NewlineAfter
}

// getRealLength - Some strings will have ANSI escape codes, which might be wrongly
// interpreted as legitimate parts of the strings. This will bother if some prompt
// components depend on other's length, so we always pass the string in this for
// getting its real-printed length.
func getRealLength(s string) (l int) {
	return len(ansi.Strip(s))
}

func getPromptPad(total, base, menu int) (pad string) {
	var padLength = total - base - menu
	for i := 0; i < padLength; i++ {
		pad += " "
	}
	return
}

var (
	// defaultColorCallbacks - All colors and effects needed in the main menu
	defaultColorCallbacks = map[string]string{
		// Base readline colors
		"{blink}": "\033[5m", // blinking
		"{bold}":  readline.BOLD,
		"{dim}":   readline.DIM,
		"{fr}":    readline.RED,
		"{g}":     readline.GREEN,
		"{b}":     readline.BLUE,
		"{y}":     readline.YELLOW,
		"{fw}":    readline.FOREWHITE,
		"{bdg}":   readline.BACKDARKGRAY,
		"{br}":    readline.BACKRED,
		"{bg}":    readline.BACKGREEN,
		"{by}":    readline.BACKYELLOW,
		"{blb}":   readline.BACKLIGHTBLUE,
		"{reset}": readline.RESET,
		// Custom colors
		"{ly}":   "\033[38;5;187m",
		"{lb}":   "\033[38;5;117m", // like VSCode var keyword
		"{db}":   "\033[38;5;24m",
		"{bddg}": "\033[48;5;237m",
	}
)
