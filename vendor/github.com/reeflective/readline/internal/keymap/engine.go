package keymap

import (
	"sort"

	"github.com/reeflective/readline/inputrc"
	"github.com/reeflective/readline/internal/core"
)

// Engine is used to manage the main and local keymaps for the shell.
type Engine struct {
	local        Mode
	main         Mode
	prefixed     inputrc.Bind
	active       inputrc.Bind
	pending      []inputrc.Bind
	skip         bool
	isCaller     bool
	nonIncSearch bool

	keys       *core.Keys
	iterations *core.Iterations
	config     *inputrc.Config
	commands   map[string]func()
}

// NewEngine is a required constructor for the keymap modes manager.
// It initializes the keymaps to their defaults or configured values.
func NewEngine(keys *core.Keys, i *core.Iterations, opts ...inputrc.Option) (*Engine, *inputrc.Config) {
	modes := &Engine{
		main:       Emacs,
		keys:       keys,
		iterations: i,
		config:     inputrc.NewDefaultConfig(),
		commands:   make(map[string]func()),
	}

	// Load the inputrc configurations and set up related things.
	modes.ReloadConfig(opts...)

	return modes, modes.config
}

// Register adds command functions to the list of available commands.
// Each key of the map should be a unique name, not yet used by any
// other builtin/user command, in order not to "overload" the builtins.
func (m *Engine) Register(commands map[string]func()) {
	for name, command := range commands {
		m.commands[name] = command
	}
}

// SetMain sets the main keymap of the shell.
// Valid builtin keymaps are:
// - emacs, emacs-meta, emacs-ctlx, emacs-standard.
// - vi, vi-insert, vi-command, vi-move.
func (m *Engine) SetMain(keymap string) {
	m.main = Mode(keymap)
	m.UpdateCursor()
}

// Main returns the local keymap.
func (m *Engine) Main() Mode {
	return m.main
}

// SetLocal sets the local keymap of the shell.
// Valid builtin keymaps are:
// - vi-opp, vi-visual. (used in commands like yank, change, delete, etc.)
// - isearch, menu-select (used in search and completion).
func (m *Engine) SetLocal(keymap string) {
	m.local = Mode(keymap)
	m.UpdateCursor()
}

// Local returns the local keymap.
func (m *Engine) Local() Mode {
	return m.local
}

// ResetLocal deactivates the local keymap of the shell.
func (m *Engine) ResetLocal() {
	m.local = ""
	m.UpdateCursor()
}

// UpdateCursor reprints the cursor corresponding to the current keymaps.
func (m *Engine) UpdateCursor() {
	switch m.local {
	case ViOpp:
		m.PrintCursor(ViOpp)
		return
	case Visual:
		m.PrintCursor(Visual)
		return
	}

	// But if not, we check for the global keymap
	switch m.main {
	case Emacs, EmacsStandard, EmacsMeta, EmacsCtrlX:
		m.PrintCursor(Emacs)
	case ViInsert:
		m.PrintCursor(ViInsert)
	case ViCommand, ViMove, Vi:
		m.PrintCursor(ViCommand)
	}
}

// PendingCursor changes the cursor to pending mode,
// and returns a function to call once done with it.
func (m *Engine) PendingCursor() (restore func()) {
	m.PrintCursor(ViOpp)

	return func() {
		m.UpdateCursor()
	}
}

// IsEmacs returns true if the main keymap is one of the emacs modes.
func (m *Engine) IsEmacs() bool {
	switch m.main {
	case Emacs, EmacsStandard, EmacsMeta, EmacsCtrlX:
		return true
	default:
		return false
	}
}

// PrintBinds displays a list of currently bound commands (and their sequences)
// to the screen. If inputrcFormat is true, it displays it formatted such that
// the output can be reused in an .inputrc file.
func (m *Engine) PrintBinds(keymap string, inputrcFormat bool) {
	var commands []string

	for command := range m.commands {
		commands = append(commands, command)
	}

	sort.Strings(commands)

	binds := m.config.Binds[keymap]
	if binds == nil {
		return
	}

	// Make a list of all sequences bound to each command.
	allBinds := make(map[string][]string)

	for _, command := range commands {
		for key, bind := range binds {
			if bind.Action != command {
				continue
			}

			commandBinds := allBinds[command]
			commandBinds = append(commandBinds, inputrc.Escape(key))
			allBinds[command] = commandBinds
		}
	}

	if inputrcFormat {
		printBindsInputrc(commands, allBinds)
	} else {
		printBindsReadable(commands, allBinds)
	}
}

// InputIsTerminator returns true when current input keys are one of
// the configured or builtin "terminators", which can be configured
// in .inputrc with the isearch-terminators variable.
func (m *Engine) InputIsTerminator() bool {
	terminators := []string{
		inputrc.Unescape(`\C-G`),
		inputrc.Unescape(`\C-]`),
	}

	binds := make(map[string]inputrc.Bind)

	for _, sequence := range terminators {
		binds[sequence] = inputrc.Bind{Action: "abort", Macro: false}
	}

	bind, _, _, _ := m.dispatchKeys(binds)

	return bind.Action == "abort"
}

// Commands returns the map of all command functions available to the shell.
// This includes the builtin commands (emacs/Vim/history/completion/etc), as
// well as any functions added by the user through Keymap.Register().
// The keys of this map are the names of each corresponding command function.
func (m *Engine) Commands() map[string]func() {
	return m.commands
}

// ActiveCommand returns the sequence/command currently being ran.
func (m *Engine) ActiveCommand() inputrc.Bind {
	return m.active
}

// NonIncrementalSearchStart is used to notify the keymap dispatchers
// that are using a minibuffer, and that the set of valid commands
// should be restrained to a few ones (self-insert/abort/rubout...).
func (m *Engine) NonIncrementalSearchStart() {
	m.nonIncSearch = true
}

// NonIncrementalSearchStop notifies the keymap dispatchers
// that we stopped editing a non-incremental search minibuffer.
func (m *Engine) NonIncrementalSearchStop() {
	m.nonIncSearch = false
}
