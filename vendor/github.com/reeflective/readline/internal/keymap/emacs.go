package keymap

import "github.com/reeflective/readline/inputrc"

var unescape = inputrc.Unescape

// emacsKeys are the default keymaps in Emacs mode.
var emacsKeys = map[string]inputrc.Bind{
	unescape(`\C-D`):     {Action: "end-of-file"},
	unescape(`\C-h`):     {Action: "backward-kill-word"},
	unescape(`\C-N`):     {Action: "down-line-or-history"},
	unescape(`\C-P`):     {Action: "up-line-or-history"},
	unescape(`\C-x\C-b`): {Action: "vi-match"},
	unescape(`\C-x\C-e`): {Action: "edit-command-line"},
	unescape(`\C-x\C-n`): {Action: "infer-next-history"},
	unescape(`\C-x\C-o`): {Action: "overwrite-mode"},
	unescape(`\C-Xr`):    {Action: "reverse-search-history"},
	unescape(`\C-Xs`):    {Action: "forward-search-history"},
	unescape(`\C-Xu`):    {Action: "undo"},
	unescape(`\M-\C-^`):  {Action: "copy-prev-word"},
	unescape(`\M-'`):     {Action: "quote-line"},
	unescape(`\M-<`):     {Action: "beginning-of-buffer-or-history"},
	unescape(`\M->`):     {Action: "end-of-buffer-or-history"},
	unescape(`\M-c`):     {Action: "capitalize-word"},
	unescape(`\M-d`):     {Action: "kill-word"},
	unescape(`\M-m`):     {Action: "copy-prev-shell-word"},
	unescape(`\M-n`):     {Action: "history-search-forward"},
	unescape(`\M-p`):     {Action: "history-search-backward"},
	unescape(`\M-u`):     {Action: "up-case-word"},
	unescape(`\M-w`):     {Action: "kill-region"},
	unescape(`\M-|`):     {Action: "vi-goto-column"},
}
