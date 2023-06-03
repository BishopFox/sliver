// Package readline provides a modern, pure Go `readline` shell implementation,
// with full `.inputrc` and legacy readline command/option support, and extended
// with various commands,options and tools commonly found in more modern shells.
//
// Example usage:
//
//	     // Create a new shell with a custom prompt.
//			rl := readline.NewShell()
//			rl.Prompt.Primary(func() string { return "> "} )
//
//		 // Display the prompt, read user input.
//		 for {
//		     line, err := rl.Readline()
//		     if err != nil {
//		         break
//		     }
//		     fmt.Println(line)
//		 }
package readline

import (
	"errors"
	"fmt"
	"os"

	"github.com/reeflective/readline/inputrc"
	"github.com/reeflective/readline/internal/color"
	"github.com/reeflective/readline/internal/completion"
	"github.com/reeflective/readline/internal/core"
	"github.com/reeflective/readline/internal/display"
	"github.com/reeflective/readline/internal/history"
	"github.com/reeflective/readline/internal/keymap"
	"github.com/reeflective/readline/internal/macro"
	"github.com/reeflective/readline/internal/term"
)

// ErrInterrupt is returned when the interrupt sequence
// is pressed on the keyboard. The sequence is usually Ctrl-C.
var ErrInterrupt = errors.New(os.Interrupt.String())

// Readline displays the readline prompt and reads user input.
// It can return from the call because of different things:
//
//   - When the user accepts the line (generally with Enter).
//   - If a particular keystroke mapping returns an error.
//     (Ctrl-C returns ErrInterrupt, Ctrl-D returns io.EOF).
//
// In all cases, the current input line is returned along with any error,
// and it is up to the caller to decide what to do with the line result.
// When the error is not nil, the returned line is not written to history.
func (rl *Shell) Readline() (string, error) {
	descriptor := int(os.Stdin.Fd())

	state, err := term.MakeRaw(descriptor)
	if err != nil {
		return "", err
	}
	defer term.Restore(descriptor, state)

	// Prompts and cursor styles
	rl.Display.PrintPrimaryPrompt()
	defer rl.Display.RefreshTransient()
	defer fmt.Print(keymap.CursorStyle("default"))

	rl.init()

	// Terminal resize events
	resize := display.WatchResize(rl.Display)
	defer close(resize)

	for {
		// Whether or not the command is resolved, let the macro
		// engine record the keys if currently recording a macro.
		// This is done before flushing all used keys, on purpose.
		macro.RecordKeys(rl.Macros)

		// Get the rid of the keys that were consumed during the
		// previous command run. This may include keys that have
		// been consumed but did not match any command.
		core.FlushUsed(rl.Keys)

		// Since we always update helpers after being asked to read
		// for user input again, we do it before actually reading it.
		rl.Display.Refresh()

		// Block and wait for available user input keys.
		// These might be read on stdin, or already available because
		// the macro engine has fed some keys in bulk when running one.
		core.WaitAvailableKeys(rl.Keys, rl.Config)

		// 1 - Local keymap (Completion/Isearch/Vim operator pending).
		bind, command, prefixed := keymap.MatchLocal(rl.Keymap)
		if prefixed {
			continue
		}

		accepted, line, err := rl.run(false, bind, command)
		if accepted {
			return line, err
		} else if command != nil {
			continue
		}

		// Past the local keymap, our actions have a direct effect
		// on the line or on the cursor position, so we must first
		// "reset" or accept any completion state we're in, if any,
		// such as a virtually inserted candidate.
		completion.UpdateInserted(rl.completer)

		// 2 - Main keymap (Vim command/insertion, Emacs).
		bind, command, prefixed = keymap.MatchMain(rl.Keymap)
		if prefixed {
			continue
		}

		accepted, line, err = rl.run(true, bind, command)
		if accepted {
			return line, err
		}

		// Reaching this point means the last key/sequence has not
		// been dispatched down to a command: therefore this key is
		// undefined for the current local/main keymaps.
		rl.handleUndefined(bind, command)
	}
}

// init gathers all steps to perform at the beginning of readline loop.
func (rl *Shell) init() {
	// Reset core editor components.
	core.FlushUsed(rl.Keys)
	rl.line.Set()
	rl.cursor.Set(0)
	rl.cursor.ResetMark()
	rl.selection.Reset()
	rl.Buffers.Reset()
	rl.History.Reset()
	rl.History.Save()
	rl.Iterations.Reset()

	// Some accept-* commands must fetch a specific
	// line outright, or keep the accepted one.
	history.Init(rl.History)

	// Reset/initialize user interface components.
	rl.Hint.Reset()
	rl.completer.ResetForce()
	display.Init(rl.Display, rl.SyntaxHighlighter)
}

// run wraps the execution of a target command/sequence with various pre/post actions
// and setup steps (buffers setup, cursor checks, iterations, key flushing, etc...)
func (rl *Shell) run(main bool, bind inputrc.Bind, command func()) (bool, string, error) {
	// An empty bind match in the local keymap means nothing
	// should be done, the main keymap must work it out.
	if !main && bind.Action == "" {
		return false, "", nil
	}

	// If the resolved bind is a macro itself, reinject its
	// bound sequence back to the key stack.
	if bind.Macro {
		macro := inputrc.Unescape(bind.Action)
		rl.Keys.Feed(false, []rune(macro)...)
	}

	// The completion system might have control of the
	// input line and be using it with a virtual insertion,
	// so it knows which line and cursor we should work on.
	rl.line, rl.cursor, rl.selection = rl.completer.GetBuffer()

	// The command might be nil, because the provided key sequence
	// did not match any. We regardless execute everything related
	// to the command, like any pending ones, and cursor checks.
	rl.execute(command)

	// Either print/clear iterations/active registers hints.
	rl.updatePosRunHints()

	// If the command just run was using the incremental search
	// buffer (acting on it), update the list of matches.
	rl.completer.UpdateIsearch()

	// Work is done: ask the completion system to
	// return the correct input line and cursor.
	rl.line, rl.cursor, rl.selection = rl.completer.GetBuffer()

	// History: save the last action to the line history,
	// and return with the call to the history system that
	// checks if the line has been accepted (entered), in
	// which case this will automatically write the history
	// sources and set up errors/returned line values.
	rl.History.SaveWithCommand(bind)

	return rl.History.LineAccepted()
}

// Run the dispatched command, any pending operator
// commands (Vim mode) and some post-run checks.
func (rl *Shell) execute(command func()) {
	if command != nil {
		command()
	}

	// Only run pending-operator commands when the command we
	// just executed has not had any influence on iterations.
	if !rl.Iterations.IsPending() {
		rl.Keymap.RunPending()
	}

	// Update/check cursor positions after run.
	switch rl.Keymap.Main() {
	case keymap.ViCommand, keymap.ViMove, keymap.Vi:
		rl.cursor.CheckCommand()
	default:
		rl.cursor.CheckAppend()
	}
}

// Some commands show their current status as a hint (iterations/macro).
func (rl *Shell) updatePosRunHints() {
	hint := core.ResetPostRunIterations(rl.Iterations)
	register, selected := rl.Buffers.IsSelected()

	if hint == "" && !selected && !rl.Macros.Recording() {
		rl.Hint.ResetPersist()
		return
	}

	if hint != "" {
		rl.Hint.Persist(hint)
	} else if selected {
		rl.Hint.Persist(color.Dim + fmt.Sprintf("(register: %s)", register))
	}
}

// handleUndefined is in charge of all actions to take when the
// last key/sequence was not dispatched down to a readline command.
func (rl *Shell) handleUndefined(bind inputrc.Bind, cmd func()) {
	if bind.Action != "" || cmd != nil {
		return
	}

	// Undefined keys incremental-search mode cancels it.
	if rl.Keymap.Local() == keymap.Isearch {
		rl.Hint.Reset()
		rl.completer.Reset()
	}
}
