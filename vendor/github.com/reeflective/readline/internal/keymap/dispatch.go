package keymap

import (
	"sort"
	"strings"

	"github.com/reeflective/readline/inputrc"
	"github.com/reeflective/readline/internal/core"
	"github.com/reeflective/readline/internal/strutil"
)

// MatchLocal incrementally attempts to match cached input keys against the local keymap.
// Returns the bind if matched, the corresponding command, and if we only matched by prefix.
func MatchLocal(eng *Engine) (bind inputrc.Bind, command func(), prefix bool) {
	if eng.local == "" {
		return
	}

	// Several local keymaps are empty by default: instead we use restricted
	// lists of commands, regardless of the key-sequence their bound to.
	binds := eng.getContextBinds(false)
	if len(binds) == 0 {
		return
	}

	// bind, command, prefix, keys := eng.dispatch(binds)
	bind, prefix, read, matched := eng.dispatchKeys(binds)

	if !bind.Macro {
		command = eng.commands[bind.Action]
	}

	if prefix {
		core.MatchedPrefix(eng.keys, read...)
	} else {
		core.MatchedKeys(eng.keys, matched, read[len(matched):]...)
	}

	// Similarly to the MatchMain() function, give a special treatment to the escape key
	// (if it's alone): using escape in Viopp/menu-complete/isearch should cancel the
	// current mode, thus we return either a Vim movement-mode command, or nothing.
	if eng.isEscapeKey() && (prefix || command == nil) {
		bind, command, prefix = eng.handleEscape(false)
	}

	return bind, command, prefix
}

// MatchMain incrementally attempts to match cached input keys against the local keymap.
// Returns the bind if matched, the corresponding command, and if we only matched by prefix.
func MatchMain(eng *Engine) (bind inputrc.Bind, command func(), prefix bool) {
	if eng.main == "" {
		return
	}

	// Get all binds present in the main keymap. Here, contrary
	// to the local keymap matching, no keymap should be empty.
	binds := eng.getContextBinds(true)
	if len(binds) == 0 {
		return
	}

	// Find the target action, macro or command.
	bind, prefix, read, _ := eng.dispatchKeys(binds)

	if !bind.Macro {
		command = eng.commands[bind.Action]
	}

	// In the main menu, all keys that have been tested against
	// the binds will be dropped after command execution (wether
	// or not there's actually a command to execute).
	if prefix {
		core.MatchedPrefix(eng.keys, read...)
	} else {
		core.MatchedKeys(eng.keys, read)
	}

	// Non-incremental search mode should always insert the keys
	// if they did not exactly match one of the valid commands.
	if eng.nonIncSearch && (command == nil || prefix) {
		bind = inputrc.Bind{Action: "self-insert"}
		eng.active = bind
		command = eng.resolve(bind)
		prefix = false
	}

	// Adjusting for the ESC key: when convert-meta is enabled,
	// many binds will actually match ESC as a prefix. This makes
	// commands like vi-movement-mode unreachable, so if the bind
	// is vi-movement-mode, we return it to be ran regardless of
	// the other binds matching by prefix.
	if eng.isEscapeKey() && !eng.IsEmacs() && prefix {
		bind, command, prefix = eng.handleEscape(true)
	}

	return bind, command, prefix
}

func (m *Engine) dispatchKeys(binds map[string]inputrc.Bind) (bind inputrc.Bind, prefix bool, read, matched []byte) {
	for {
		// Read a single byte from the input buffer.
		// This mimics the way Bash reads input when the inputrc option `byte-oriented` is set.
		// This is because the default binds map is built with byte sequences, not runes, and this
		// has some implications if the terminal is sending 8-bit characters (extanded alphabet).
		key, empty := core.PopKey(m.keys)
		if empty {
			break
		}

		read = append(read, key)

		match, prefixed := m.matchBind(read, binds)

		// If the current keys have no matches but the previous
		// matching process found a prefix, use it with the keys.
		if match.Action == "" && len(prefixed) == 0 {
			prefix = false
			m.active = m.prefixed
			m.prefixed = inputrc.Bind{}

			break
		}

		// From here, there is at least one bind matched, by prefix
		// or exactly, so the key we popped is considered matched.
		matched = append(matched, key)

		// If we matched a prefix, keep the matched bind for later.
		if len(prefixed) > 0 {
			prefix = true

			if match.Action != "" {
				m.prefixed = match
			}

			continue
		}

		// Or an exact match, so drop any prefixed one.
		prefix = false
		m.active = match
		m.prefixed = inputrc.Bind{}

		break
	}

	return m.active, prefix, read, matched
}

func (m *Engine) matchBind(keys []byte, binds map[string]inputrc.Bind) (inputrc.Bind, []inputrc.Bind) {
	var match inputrc.Bind
	var prefixed []inputrc.Bind

	// Make a sorted list with all keys in the binds map.
	var sequences []string
	for sequence := range binds {
		sequences = append(sequences, sequence)
	}

	// Sort the list of sequences by alphabetical order and length.
	sort.Slice(sequences, func(i, j int) bool {
		if len(sequences[i]) == len(sequences[j]) {
			return sequences[i] < sequences[j]
		}
		return len(sequences[i]) < len(sequences[j])
	})

	// Iterate over the sorted list of sequences and find all binds
	// that match the sequence either by prefix or exactly.
	for _, sequence := range sequences {
		seq := strutil.ConvertMeta([]rune(sequence))

		if len(string(keys)) < len(seq) && strings.HasPrefix(seq, string(keys)) {
			prefixed = append(prefixed, binds[sequence])
		}

		if string(keys) == seq {
			match = binds[sequence]
		}
	}

	return match, prefixed
}

func (m *Engine) resolve(bind inputrc.Bind) func() {
	if bind.Macro {
		return nil
	}

	if bind.Action == "" {
		return nil
	}

	return m.commands[bind.Action]
}

// handleEscape is used to override or change the matched command when the escape key has
// been pressed: it might exit completion/isearch menus, use the vi-movement-mode, etc.
func (m *Engine) handleEscape(main bool) (bind inputrc.Bind, cmd func(), pref bool) {
	switch {
	case m.prefixed.Action == "vi-movement-mode":
		// The vi-movement-mode command always has precedence over
		// other binds when we are currently using the main keymap.
		bind = m.prefixed

	case !main && m.IsEmacs() && m.Local() == Isearch:
		// There is no dedicated "soft-escape" of the incremental-search
		// mode when in Emacs keymap, so we use the escape key to cancel
		// the search and return to the main keymap.
		bind = inputrc.Bind{Action: "emacs-editing-mode"}

		core.PopForce(m.keys)

	case !main:
		// When using the local keymap, we simply drop any prefixed
		// or matched bind, so that the key will be matched against
		// the main keymap: between both, completion/isearch menus
		// will likely be cancelled.
		bind = inputrc.Bind{}
	}

	// Drop what needs to, and resolve the command.
	m.prefixed = inputrc.Bind{}

	if bind.Action != "" && !bind.Macro {
		cmd = m.resolve(bind)
	}

	// Drop the escape key in the stack
	if main {
		core.PopForce(m.keys)
	}

	return bind, cmd, pref
}

func (m *Engine) isEscapeKey() bool {
	keys := m.keys.Caller()
	if len(keys) == 0 {
		return false
	}

	if len(keys) > 1 {
		return false
	}

	if keys[0] != inputrc.Esc {
		return false
	}

	return true
}
