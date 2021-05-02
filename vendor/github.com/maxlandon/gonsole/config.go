package gonsole

import (
	"fmt"

	"github.com/maxlandon/readline"
)

// InputMode - The input & navigation mode of the console. (Vim or Emacs)
type InputMode string

const (
	// InputVim - Vim editing system, with modes, registers, navigation and iterations.
	InputVim InputMode = "vim"
	// InputEmacs - Emacs-style mode, without most of Emacs binding (not a real Emacs).
	InputEmacs InputMode = "emacs"
)

// Config - The console configuration (prompts, hints, modes, etc)
type Config struct {
	InputMode           InputMode                `json:"input_mode"`
	Prompts             map[string]*PromptConfig `json:"prompts"`
	Hints               bool                     `json:"hints"`
	MaxTabCompleterRows int                      `json:"max_tab_completer_rows"`
	Highlighting        map[string]string        `json:"highlighting"`
}

// NewDefaultConfig - Users wishing to setup a special console configuration should
// use this function in order to ensure there are no nil maps anywhere, and with defaults.
func NewDefaultConfig() *Config {
	return &Config{
		InputMode:           InputEmacs,
		Prompts:             map[string]*PromptConfig{},
		Hints:               true,
		MaxTabCompleterRows: 50,
		Highlighting: map[string]string{
			"{command}":          readline.BOLD,
			"{command-argument}": readline.FOREWHITE,
			"{option}":           readline.BOLD,
			"{option-argument}":  readline.FOREWHITE,
			"{hint-text}":        "\033[38;5;248m",
		},
	}
}

// PromptConfig - Contains all the information needed for the PromptConfig of a given menu.
type PromptConfig struct {
	Left            string `json:"left"`
	Right           string `json:"right"`
	NewlineAfter    bool   `json:"newline_after"`
	NewlineBefore   bool   `json:"newline_before"`
	Multiline       bool   `json:"multiline"`
	MultilinePrompt string `json:"multiline_prompt"`
}

// newDefaultPromptConfig - Newly created menus have a default prompt configuration
func newDefaultPromptConfig(menu string) *PromptConfig {
	return &PromptConfig{
		Left:            fmt.Sprintf("gonsole (%s)", menu),
		Right:           "",
		NewlineAfter:    true,
		NewlineBefore:   true,
		Multiline:       true,
		MultilinePrompt: " > ",
	}
}

// LoadConfig - Loads a config struct, but does not immediately refresh
// the prompt. Settings will apply as they are needed by the console.
func (c *Console) LoadConfig(conf *Config) {
	if conf == nil {
		return
	}

	// Ensure no fields are nil
	if conf.Prompts == nil {
		conf.Prompts = map[string]*PromptConfig{"": newDefaultPromptConfig("")}
	}

	// Users might forget to load default highlighting maps.
	if conf.Highlighting == nil {
		conf.Highlighting = map[string]string{
			"{command}":          readline.BOLD,
			"{command-argument}": readline.FOREWHITE,
			"{option}":           readline.BOLD,
			"{option-argument}":  readline.FOREWHITE,
			"{hint-text}":        "\033[38;5;248m",
		}
	}
	// Then load and apply all componenets that need a refresh now
	c.config = conf

	// Setup the prompt, and input mode
	c.reloadConfig()

	return
}

// GetConfig - The console exports its configuration.
func (c *Console) GetConfig() (conf *Config) {
	return c.config
}

// loadDefaultConfig - Sane defaults for the gonsole Console.
func (c *Console) loadDefaultConfig() {
	c.config = NewDefaultConfig()
	// Make a default prompt for this application
	c.config.Prompts[""] = newDefaultPromptConfig("")
}

func (c *Console) reloadConfig() {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Setup the prompt
	c.current.Prompt.loadFromConfig(c.config.Prompts[c.current.Name])
	c.shell.SetPrompt(c.current.Prompt.Render())
	c.shell.MultilinePrompt = c.config.Prompts[c.current.Name].MultilinePrompt
	c.shell.Multiline = c.config.Prompts[c.current.Name].Multiline
	c.PreOutputNewline = c.config.Prompts[c.current.Name].NewlineAfter
	c.LeaveNewline = c.config.Prompts[c.current.Name].NewlineBefore

	// Input mode
	if c.config.InputMode == InputEmacs {
		c.shell.InputMode = readline.Emacs
	} else {
		c.shell.InputMode = readline.Vim
	}
}
