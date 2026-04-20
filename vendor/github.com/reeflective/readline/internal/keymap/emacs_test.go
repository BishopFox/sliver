package keymap

import (
	"testing"

	"github.com/reeflective/readline/inputrc"
)

// TestBackspaceBindsToDeleteChar verifies that \C-h (ASCII 8, sent by Backspace
// on some terminals) is bound to backward-delete-char, not backward-kill-word.
//
// GNU readline default: \C-h → backward-delete-char, \M-\C-h → backward-kill-word.
// The emacsKeys override table must not shadow the DefaultBinds() entry for \C-h.
func TestBackspaceBindsToDeleteChar(t *testing.T) {
	backspace := inputrc.Unescape(`\C-h`)

	// Verify DefaultBinds has the correct baseline.
	defaults := inputrc.DefaultBinds()
	emacs := defaults["emacs"]
	if emacs == nil {
		t.Fatal("DefaultBinds missing emacs keymap")
	}
	if got := emacs[backspace].Action; got != "backward-delete-char" {
		t.Errorf("DefaultBinds[emacs][\\C-h] = %q, want %q", got, "backward-delete-char")
	}

	// emacsKeys must not override \C-h with backward-kill-word.
	if bind, ok := emacsKeys[backspace]; ok {
		if bind.Action == "backward-kill-word" {
			t.Errorf("emacsKeys[\\C-h] = %q: this overrides DefaultBinds and causes Backspace to kill the whole word; remove this entry (use \\M-\\C-h for backward-kill-word instead)", bind.Action)
		}
	}
}

// TestMetaBackspaceBindsToKillWord verifies that the word-kill action is still
// reachable via \M-\C-h (Meta+Backspace), which is the GNU readline standard.
func TestMetaBackspaceBindsToKillWord(t *testing.T) {
	metaBackspace := inputrc.Unescape(`\M-\C-h`)
	defaults := inputrc.DefaultBinds()
	emacs := defaults["emacs"]
	if emacs == nil {
		t.Fatal("DefaultBinds missing emacs keymap")
	}
	if got := emacs[metaBackspace].Action; got != "backward-kill-word" {
		t.Errorf("DefaultBinds[emacs][\\M-\\C-h] = %q, want %q (Meta+Backspace should still kill word)", got, "backward-kill-word")
	}
}
