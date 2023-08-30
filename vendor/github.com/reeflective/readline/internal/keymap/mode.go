package keymap

// Mode is a root keymap mode for the shell.
// To each of these keymap modes is bound a keymap.
type Mode string

// These are the root keymaps used in the readline shell.
// Their functioning is similar to how ZSH organizes keymaps.
const (
	// Editor.
	Emacs         = "emacs"
	EmacsMeta     = "emacs-meta"
	EmacsCtrlX    = "emacs-ctlx"
	EmacsStandard = "emacs-standard"

	ViInsert  = "vi-insert"
	Vi        = "vi"
	ViCommand = "vi-command"
	ViMove    = "vi-move"
	Visual    = "vi-visual"
	ViOpp     = "vi-opp"

	// Completion and search.
	Isearch    = "isearch"
	MenuSelect = "menu-select"
)
