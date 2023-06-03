package keymap

import "github.com/reeflective/readline/inputrc"

// menuselectKeys are the default keymaps in menuselect mode.
var menuselectKeys = map[string]inputrc.Bind{
	unescape(`\C-i`):    {Action: "menu-complete"},
	unescape(`\C-N`):    {Action: "menu-complete"},
	unescape(`\C-P`):    {Action: "menu-complete-backward"},
	unescape(`\e[Z`):    {Action: "menu-complete-backward"},
	unescape(`\C-@`):    {Action: "accept-and-menu-complete"},
	unescape(`\C-F`):    {Action: "menu-incremental-search"},
	unescape(`\e[A`):    {Action: "menu-complete-backward"},
	unescape(`\e[B`):    {Action: "menu-complete"},
	unescape(`\e[C`):    {Action: "menu-complete"},
	unescape(`\e[D`):    {Action: "menu-complete-backward"},
	unescape(`\e[1;5A`): {Action: "menu-complete-prev-tag"},
	unescape(`\e[1;5B`): {Action: "menu-complete-next-tag"},
}

// isearchCommands is a subset of commands that are valid in incremental-search mode.
var isearchCommands = []string{
	// Edition
	"abort",
	"backward-delete-char",
	"backward-kill-word",
	"backward-kill-line",
	"unix-line-discard",
	"unix-word-rubout",
	"vi-unix-word-rubout",
	"clear-screen",
	"clear-display",
	"magic-space",
	"vi-movement-mode",
	"yank",
	"self-insert",

	// History
	"accept-and-infer-next-history",
	"accept-line",
	"accept-and-hold",
	"operate-and-get-next",
	"history-incremental-search-forward",
	"history-incremental-search-backward",
	"forward-search-history",
	"reverse-search-history",
	"history-search-forward",
	"history-search-backward",
	"history-substring-search-forward",
	"history-substring-search-backward",
	"incremental-forward-search-history",
	"incremental-reverse-search-history",
}

// nonIsearchCommands is an even more restricted set of commands
// that are used when a non-incremental search mode is active.
var nonIsearchCommands = []string{
	"abort",
	"accept-line",
	"backward-delete-char",
	"backward-kill-word",
	"backward-kill-line",
	"unix-line-discard",
	"unix-word-rubout",
	"vi-unix-word-rubout",
	"self-insert",
}

// getContextBinds is in charge of returning the precise list of binds
// that are relevant in a given context (local/main keymap). Some submodes
// (like non/incremental search) will further restrict the set of binds.
func (m *Engine) getContextBinds(main bool) (binds map[string]inputrc.Bind) {
	// First get the unfiltered list
	// of binds for the current keymap.
	if main {
		binds = m.config.Binds[string(m.main)]
	} else {
		binds = m.config.Binds[string(m.local)]
	}

	// No filtering possible on the local keymap, or if no binds.
	if !main || len(binds) == 0 {
		return
	}

	// Then possibly restrict in some submodes.
	switch {
	case m.Local() == Isearch:
		binds = m.restrictCommands(m.main, isearchCommands)
	case m.nonIncSearch:
		binds = m.restrictCommands(m.main, nonIsearchCommands)
	}

	return
}

func (m *Engine) restrictCommands(mode Mode, commands []string) map[string]inputrc.Bind {
	if len(commands) == 0 {
		return m.config.Binds[string(mode)]
	}

	isearch := make(map[string]inputrc.Bind)

	for seq, command := range m.config.Binds[string(mode)] {
		// Widget must be a valid isearch widget
		if !isValidCommand(command.Action, commands) {
			continue
		}

		// Or bind to our temporary isearch keymap
		isearch[seq] = command
	}

	return isearch
}

func isValidCommand(widget string, commands []string) bool {
	for _, isw := range commands {
		if isw == widget {
			return true
		}
	}

	return false
}
