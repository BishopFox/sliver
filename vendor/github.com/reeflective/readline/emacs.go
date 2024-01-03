package readline

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"

	"github.com/rivo/uniseg"

	"github.com/reeflective/readline/inputrc"
	"github.com/reeflective/readline/internal/color"
	"github.com/reeflective/readline/internal/completion"
	"github.com/reeflective/readline/internal/keymap"
	"github.com/reeflective/readline/internal/strutil"
	"github.com/reeflective/readline/internal/term"
)

// standardCommands returns all standard/emacs commands.
// Under each comment are gathered all commands related to the comment's
// subject. When there are two subgroups separated by an empty line, the
// second one comprises commands that are not legacy readline commands.
//
// Modes
// Moving
// Changing text
// Killing and Yanking
// Numeric arguments.
// Macros
// Miscellaneous.
func (rl *Shell) standardCommands() commands {
	widgets := map[string]func(){
		// Modes
		"emacs-editing-mode": rl.emacsEditingMode,

		// Moving
		"forward-char":         rl.forwardChar,
		"backward-char":        rl.backwardChar,
		"forward-word":         rl.forwardWord,
		"backward-word":        rl.backwardWord,
		"shell-forward-word":   rl.forwardShellWord,
		"shell-backward-word":  rl.backwardShellWord,
		"beginning-of-line":    rl.beginningOfLine,
		"end-of-line":          rl.endOfLine,
		"previous-screen-line": rl.upLine,
		"next-screen-line":     rl.downLine,
		"clear-screen":         rl.clearScreen,
		"clear-display":        rl.clearDisplay,
		"redraw-current-line":  rl.Display.Refresh,

		// Changing text
		"end-of-file":                  rl.endOfFile,
		"delete-char":                  rl.deleteChar,
		"backward-delete-char":         rl.backwardDeleteChar,
		"forward-backward-delete-char": rl.forwardBackwardDeleteChar,
		"quoted-insert":                rl.quotedInsert,
		"tab-insert":                   rl.tabInsert,
		"self-insert":                  rl.selfInsert,
		"bracketed-paste-begin":        rl.bracketedPasteBegin,
		"transpose-chars":              rl.transposeChars,
		"transpose-words":              rl.transposeWords,
		"shell-transpose-words":        rl.shellTransposeWords,
		"down-case-word":               rl.downCaseWord,
		"up-case-word":                 rl.upCaseWord,
		"capitalize-word":              rl.capitalizeWord,
		"overwrite-mode":               rl.overwriteMode,
		"delete-horizontal-whitespace": rl.deleteHorizontalWhitespace,

		"delete-word":      rl.deleteWord,
		"quote-region":     rl.quoteRegion,
		"quote-line":       rl.quoteLine,
		"keyword-increase": rl.keywordIncrease,
		"keyword-decrease": rl.keywordDecrease,

		// Killing & yanking
		"kill-line":           rl.killLine,
		"backward-kill-line":  rl.backwardKillLine,
		"unix-line-discard":   rl.backwardKillLine,
		"kill-whole-line":     rl.killWholeLine,
		"kill-word":           rl.killWord,
		"backward-kill-word":  rl.backwardKillWord,
		"unix-word-rubout":    rl.backwardKillWord,
		"kill-region":         rl.killRegion,
		"copy-region-as-kill": rl.copyRegionAsKill,
		"copy-backward-word":  rl.copyBackwardWord,
		"copy-forward-word":   rl.copyForwardWord,
		"yank":                rl.yank,
		"yank-pop":            rl.yankPop,

		"kill-buffer":              rl.killBuffer,
		"shell-kill-word":          rl.shellKillWord,
		"shell-backward-kill-word": rl.shellBackwardKillWord,
		"copy-prev-shell-word":     rl.copyPrevShellWord,

		// Numeric arguments
		"digit-argument": rl.digitArgument,

		// Macros
		"start-kbd-macro":      rl.startKeyboardMacro,
		"end-kbd-macro":        rl.endKeyboardMacro,
		"call-last-kbd-macro":  rl.callLastKeyboardMacro,
		"print-last-kbd-macro": rl.printLastKeyboardMacro,

		"macro-toggle-record": rl.macroToggleRecord,
		"macro-run":           rl.macroRun,

		// Miscellaneous
		"re-read-init-file":         rl.reReadInitFile,
		"abort":                     rl.abort,
		"do-lowercase-version":      rl.doLowercaseVersion,
		"prefix-meta":               rl.prefixMeta,
		"undo":                      rl.undoLast,
		"revert-line":               rl.revertLine,
		"set-mark":                  rl.setMark,
		"exchange-point-and-mark":   rl.exchangePointAndMark,
		"character-search":          rl.characterSearch,
		"character-search-backward": rl.characterSearchBackward,
		"insert-comment":            rl.insertComment,
		"dump-functions":            rl.dumpFunctions,
		"dump-variables":            rl.dumpVariables,
		"dump-macros":               rl.dumpMacros,
		"magic-space":               rl.magicSpace,
		"edit-and-execute-command":  rl.editAndExecuteCommand,
		"edit-command-line":         rl.editCommandLine,

		"redo":                rl.redo,
		"select-keyword-next": rl.selectKeywordNext,
		"select-keyword-prev": rl.selectKeywordPrev,
	}

	return widgets
}

//
// Modes ----------------------------------------------------------------
//

// When in vi command mode, this causes a switch to emacs editing mode.
func (rl *Shell) emacsEditingMode() {
	// Reset any visual selection and iterations.
	rl.selection.Reset()
	rl.Iterations.Reset()
	rl.Buffers.Reset()

	// Cancel completions and hints if any, and reassign the
	// current line/cursor/selection for the cursor check below
	// to be effective. This is needed when in isearch mode.
	rl.Hint.Reset()
	rl.completer.Reset()
	rl.line, rl.cursor, rl.selection = rl.completer.GetBuffer()

	// Update the keymap.
	rl.Keymap.SetMain(keymap.Emacs)
}

//
// Movement -------------------------------------------------------------
//

// Move forward one character.
func (rl *Shell) forwardChar() {
	startPos := rl.cursor.Pos()

	// Only exception where we actually don't forward a character.
	if rl.Config.GetBool("history-autosuggest") && rl.cursor.Pos() >= rl.line.Len()-1 {
		rl.autosuggestAccept()
	}

	if rl.cursor.Pos() > startPos {
		return
	}

	// Else, we move forward.
	rl.History.SkipSave()
	vii := rl.Iterations.Get()

	for i := 1; i <= vii; i++ {
		rl.cursor.Inc()
	}
}

// Move backward one character.
func (rl *Shell) backwardChar() {
	rl.History.SkipSave()
	vii := rl.Iterations.Get()

	for i := 1; i <= vii; i++ {
		rl.cursor.Dec()
	}
}

// Move to the beginning of the next word. The editor’s idea
// of a word is any sequence of alphanumeric characters.
func (rl *Shell) forwardWord() {
	rl.History.SkipSave()
	vii := rl.Iterations.Get()

	for i := 1; i <= vii; i++ {
		// When we have an autosuggested history and if we are at the end
		// of the line, insert the next word from this suggested line.
		rl.insertAutosuggestPartial(true)

		forward := rl.line.ForwardEnd(rl.line.Tokenize, rl.cursor.Pos())
		rl.cursor.Move(forward + 1)
	}
}

// Move to the beginning of the current or previousword. The editor’s
// idea of a word is any sequence of alphanumeric characters.
func (rl *Shell) backwardWord() {
	rl.History.SkipSave()

	vii := rl.Iterations.Get()
	for i := 1; i <= vii; i++ {
		backward := rl.line.Backward(rl.line.Tokenize, rl.cursor.Pos())
		rl.cursor.Move(backward)
	}
}

// Move forward to the beginning of the next word.
// The editor's idea of a word is defined by classic sh-style word splitting:
// any non-spaced sequence of characters, or a quoted sequence.
func (rl *Shell) forwardShellWord() {
	vii := rl.Iterations.Get()

	for i := 1; i <= vii; i++ {
		rl.selection.SelectAShellWord()
		_, _, tepos, _ := rl.selection.Pop()
		rl.cursor.Set(tepos)
	}
}

// Move to the beginning of the current or previous word.
// The editor's idea of a word is defined by classic sh-style word splitting:
// any non-spaced sequence of characters, or a quoted sequence.
func (rl *Shell) backwardShellWord() {
	vii := rl.Iterations.Get()

	for i := 1; i <= vii; i++ {
		// First go the beginning of the blank word
		startPos := rl.cursor.Pos()
		backward := rl.line.Backward(rl.line.TokenizeSpace, startPos)
		rl.cursor.Move(backward)

		// Now try to find enclosing quotes from here.
		bpos, _ := rl.selection.SelectAShellWord()
		rl.cursor.Set(bpos)
	}
}

// Move to the beginning of the line. If already at the beginning
// of the line, move to the beginning of the previous line, if any.
func (rl *Shell) beginningOfLine() {
	rl.History.SkipSave()

	// Handle 0 as iteration to Vim.
	if !rl.Keymap.IsEmacs() && rl.Iterations.IsSet() {
		rl.Iterations.Add("0")
		return
	}

	rl.cursor.BeginningOfLine()
}

// Move to the end of the line. If already at the end
// of the line, move to the end of the next line, if any.
func (rl *Shell) endOfLine() {
	rl.History.SkipSave()
	// If in Vim command mode, cursor
	// will be brought back once later.
	rl.cursor.EndOfLineAppend()
}

// Move up one line if the current buffer has more than one line.
func (rl *Shell) upLine() {
	lines := rl.Iterations.Get()
	rl.cursor.LineMove(lines * -1)
}

// Move down one line if the current buffer has more than one line.
func (rl *Shell) downLine() {
	lines := rl.Iterations.Get()
	rl.cursor.LineMove(lines)
}

// Clear the current screen and redisplay the prompt and input line.
// This does not clear the terminal's output buffer.
func (rl *Shell) clearScreen() {
	rl.History.SkipSave()

	fmt.Print(term.CursorTopLeft)
	fmt.Print(term.ClearScreen)

	rl.Display.PrintPrimaryPrompt()
}

// Clear the current screen and redisplay the prompt and input line.
// This does clear the terminal's output buffer.
func (rl *Shell) clearDisplay() {
	rl.History.SkipSave()

	fmt.Print(term.CursorTopLeft)
	fmt.Print(term.ClearDisplay)

	rl.Display.PrintPrimaryPrompt()
}

//
// Changing Text --------------------------------------------------------
//

// The character indicating end-of-file as set, for example,
// by “stty”.  If this character is read when there are no
// characters on the line, and point is at the beginning of
// the line, readline interprets it as the end of input and
// returns EOF.
func (rl *Shell) endOfFile() {
	switch rl.line.Len() {
	case 0:
		rl.Display.AcceptLine()
		rl.History.Accept(false, false, io.EOF)
	default:
		rl.deleteChar()
	}
}

// Delete the character under the cursor.
func (rl *Shell) deleteChar() {
	rl.History.Save()

	vii := rl.Iterations.Get()

	// Delete the chars in the line anyway
	for i := 1; i <= vii; i++ {
		rl.line.CutRune(rl.cursor.Pos())
	}
}

// Delete the character behind the cursor.
// If the character to delete is a matching character
// (quote/brackets/braces,etc) and that the character
// under the cursor is its matching counterpart, this
// character will also be deleted.
func (rl *Shell) backwardDeleteChar() {
	if rl.Keymap.Main() == keymap.ViInsert {
		rl.History.SkipSave()
	} else {
		rl.History.Save()
	}

	// We might currently have a selected candidate inserted,
	// and thus we should accept it as part of the real input
	// line before cutting any character.
	completion.UpdateInserted(rl.completer)

	if rl.cursor.Pos() == 0 {
		return
	}

	vii := rl.Iterations.Get()

	switch vii {
	case 1:
		// Handle removal of autopairs characters.
		if rl.Config.GetBool("autopairs") {
			completion.AutopairDelete(rl.line, rl.cursor)
		}

		// And then delete the character under cursor.
		rl.cursor.Dec()
		rl.line.CutRune(rl.cursor.Pos())

	default:
		for i := 1; i <= vii; i++ {
			rl.cursor.Dec()
			rl.line.CutRune(rl.cursor.Pos())
		}
	}
}

// Delete the character under the cursor, unless the cursor is at the
// end of the line, in which case the character behind the cursor is deleted.
func (rl *Shell) forwardBackwardDeleteChar() {
	switch rl.cursor.Pos() {
	case rl.line.Len():
		rl.backwardDeleteChar()
	default:
		rl.deleteChar()
	}
}

// Add the next character that you type to the line verbatim.
// This is how to insert characters like C-q, for example.
func (rl *Shell) quotedInsert() {
	rl.History.SkipSave()
	rl.completer.TrimSuffix()

	done := rl.Keymap.PendingCursor()
	defer done()

	key, _ := rl.Keys.ReadKey()

	quoted, _ := strutil.Quote(key)

	rl.cursor.InsertAt(quoted...)
}

// Insert a tab character.
func (rl *Shell) tabInsert() {
	rl.History.SkipSave()

	rl.cursor.InsertAt('\t')
}

// Insert the character typed.
func (rl *Shell) selfInsert() {
	rl.History.SkipSave()

	// Handle suffix-autoremoval for inserted completions.
	rl.completer.TrimSuffix()

	key := rl.Keys.Caller()

	// Handle autopair insertion (for the closer only)
	searching, _, _ := rl.completer.NonIncrementallySearching()
	isearch := rl.Keymap.Local() == keymap.Isearch

	if !searching && !isearch && rl.Config.GetBool("autopairs") {
		if jump := completion.AutopairInsertOrJump(key[0], rl.line, rl.cursor); jump {
			return
		}
	}

	var quoted []rune
	var length int

	if rl.Config.GetBool("output-meta") && key[0] != inputrc.Esc {
		quoted = append(quoted, key[0])
		length = uniseg.StringWidth(string(quoted))
	} else {
		quoted, length = strutil.Quote(key[0])
	}

	rl.cursor.InsertAt(quoted...)
	rl.cursor.Move(-1 * len(quoted))
	rl.cursor.Move(length)
}

func (rl *Shell) bracketedPasteBegin() {
	// keys, _ := rl.Keys.PeekAllBytes()
	// fmt.Println(string(keys))
}

// Drag the character before point forward over the character
// at point, moving point forward as well.  If point is at the
// end of the line, then this transposes the two characters
// before point.  Negative arguments have no effect.
func (rl *Shell) transposeChars() {
	if rl.cursor.Pos() < 2 || rl.line.Len() < 2 {
		rl.History.SkipSave()
		return
	}

	rl.History.Save()

	switch {
	case rl.cursor.Pos() == rl.line.Len():
		last := (*rl.line)[rl.cursor.Pos()-1]
		blast := (*rl.line)[rl.cursor.Pos()-2]
		(*rl.line)[rl.cursor.Pos()-2] = last
		(*rl.line)[rl.cursor.Pos()-1] = blast
	default:
		last := rl.cursor.Char()
		blast := (*rl.line)[rl.cursor.Pos()-1]
		(*rl.line)[rl.cursor.Pos()-1] = last
		rl.cursor.ReplaceWith(blast)
	}
}

// Drag the word before point past the word after point,
// moving point over that word as well.  If point is at the
// end of the line, this transposes the last two words on the
// line. If a numeric argument is given, the word to transpose
// is chosen backward.
func (rl *Shell) transposeWords() {
	rl.History.Save()

	startPos := rl.cursor.Pos()
	rl.cursor.ToFirstNonSpace(true)
	rl.cursor.CheckCommand()

	// Save the current word and move the cursor to its beginning
	rl.viSelectInWord()
	rl.selection.Visual(false)
	toTranspose, tbpos, tepos, _ := rl.selection.Pop()

	// Then move some number of words.
	// Either use words backward (if we are at end of line) or forward.
	rl.cursor.Set(tbpos)

	if tepos >= rl.line.Len()-1 || rl.Iterations.IsSet() {
		rl.backwardWord()
	} else {
		rl.viForwardWord()
	}

	// Save the word to transpose with
	rl.viSelectInWord()
	rl.selection.Visual(false)
	transposeWith, wbpos, wepos, _ := rl.selection.Pop()

	// We might be on the first word of the line,
	// in which case we don't do anything.
	if tbpos == 0 {
		rl.cursor.Set(startPos)
		return
	}

	// If we went forward rather than backward, swap everything.
	if wbpos > tbpos {
		wbpos, tbpos = tbpos, wbpos
		wepos, tepos = tepos, wepos
		transposeWith, toTranspose = toTranspose, transposeWith
	}

	// Assemble the newline
	begin := string((*rl.line)[:wbpos])
	newLine := append([]rune(begin), []rune(toTranspose)...)
	newLine = append(newLine, (*rl.line)[wepos:tbpos]...)
	newLine = append(newLine, []rune(transposeWith)...)
	newLine = append(newLine, (*rl.line)[tepos:]...)
	rl.line.Set(newLine...)

	// And replace the cursor
	rl.cursor.Set(tepos)
}

// Drag the shell word before point past the shell word after point,
// moving point over that shell word as well. If point is at the
// end of the line, this transposes the last two words on the line.
// If a numeric argument is given, the word to transpose is chosen backward.
func (rl *Shell) shellTransposeWords() {
	rl.History.Save()

	startPos := rl.cursor.Pos()

	// Save the current word
	rl.viSelectAShellWord()
	toTranspose, tbpos, tepos, _ := rl.selection.Pop()

	// First move back the number of words
	rl.cursor.Set(tbpos)
	rl.backwardShellWord()

	// Save the word to transpose with
	rl.viSelectAShellWord()
	transposeWith, wbpos, wepos, _ := rl.selection.Pop()

	// We might be on the first word of the line,
	// in which case we don't do anything.
	if wepos > tbpos {
		rl.cursor.Set(startPos)
		return
	}

	// Assemble the newline
	begin := string((*rl.line)[:wbpos])
	newLine := append([]rune(begin), []rune(toTranspose)...)
	newLine = append(newLine, (*rl.line)[wepos:tbpos]...)
	newLine = append(newLine, []rune(transposeWith)...)
	newLine = append(newLine, (*rl.line)[tepos:]...)
	rl.line.Set(newLine...)

	// And replace cursor
	rl.cursor.Set(tepos)
}

// Lowercase the current (or following) word. With a negative argument,
// lowercase the previous word, but do not move point.
func (rl *Shell) downCaseWord() {
	rl.History.Save()

	startPos := rl.cursor.Pos()

	// Save the current word
	rl.cursor.Inc()
	backward := rl.line.Backward(rl.line.Tokenize, rl.cursor.Pos())
	rl.cursor.Move(backward)

	rl.selection.Mark(rl.cursor.Pos())
	forward := rl.line.ForwardEnd(rl.line.Tokenize, rl.cursor.Pos())
	rl.cursor.Move(forward)

	rl.selection.ReplaceWith(unicode.ToLower)
	rl.cursor.Set(startPos)
}

// Uppercase the current (or following) word.  With a negative argument,
// uppercase the previous word, but do not move point.
func (rl *Shell) upCaseWord() {
	rl.History.Save()

	startPos := rl.cursor.Pos()

	// Save the current word
	rl.cursor.Inc()
	backward := rl.line.Backward(rl.line.Tokenize, rl.cursor.Pos())
	rl.cursor.Move(backward)

	rl.selection.Mark(rl.cursor.Pos())
	forward := rl.line.ForwardEnd(rl.line.Tokenize, rl.cursor.Pos())
	rl.cursor.Move(forward)

	rl.selection.ReplaceWith(unicode.ToUpper)
	rl.cursor.Set(startPos)
}

// Capitalize the current (or following) word.  With a negative argument,
// capitalize the previous word, but do not move point.
func (rl *Shell) capitalizeWord() {
	if rl.line.Len() == 0 {
		return
	}

	rl.History.Save()

	startPos := rl.cursor.Pos()

	rl.cursor.Inc()
	backward := rl.line.Backward(rl.line.Tokenize, rl.cursor.Pos())
	rl.cursor.Move(backward)

	letter := rl.cursor.Char()
	rl.cursor.ReplaceWith(unicode.ToUpper(letter))
	rl.cursor.Set(startPos)
}

// Toggle overwrite mode. In overwrite mode, characters bound to
// self-insert replace the text at point rather than pushing the
// text to the right.  Characters bound to backward-delete-char
// replace the character before point with a space.
func (rl *Shell) overwriteMode() {
	// We store the current line as an undo item first, but will not
	// store any intermediate changes (in the loop below) as undo items.
	rl.History.Save()

	done := rl.Keymap.PendingCursor()
	defer done()

	// All replaced characters are stored, to be used with backspace
	cache := make([]rune, 0)

	// Don't use the delete cache past the end of the line
	lineStart := rl.line.Len()

	// The replace mode is quite special in that it does escape back
	// to the main readline loop: it keeps reading characters and inserts
	// them as long as the escape key is not pressed.
	for {
		// We read a character to use first.
		key, isAbort := rl.Keys.ReadKey()
		if isAbort {
			break
		}

		// If the key is a backspace, we go back one character
		if string(key) == inputrc.Unescape(string(`\C-?`)) {
			if rl.cursor.Pos() > lineStart {
				rl.backwardDeleteChar()
			} else if rl.cursor.Pos() > 0 {
				rl.cursor.Dec()
			}

			// And recover the last replaced character
			if len(cache) > 0 && rl.cursor.Pos() < lineStart {
				key = cache[len(cache)-1]
				cache = cache[:len(cache)-1]

				rl.cursor.ReplaceWith(key)
			}
		} else {
			// If the cursor is at the end of the line,
			// we insert the character instead of replacing.
			if rl.line.Len() == rl.cursor.Pos() {
				rl.cursor.InsertAt(key)
			} else {
				cache = append(cache, rl.cursor.Char())
				rl.cursor.ReplaceWith(key)
				rl.cursor.Inc()
			}
		}

		// Update the line
		rl.Display.Refresh()
	}
}

// Delete all spaces and tabs around point.
func (rl *Shell) deleteHorizontalWhitespace() {
	rl.History.Save()

	startPos := rl.cursor.Pos()

	rl.cursor.ToFirstNonSpace(false)

	if rl.cursor.Pos() != startPos {
		rl.cursor.Inc()
	}

	bpos := rl.cursor.Pos()
	rl.cursor.ToFirstNonSpace(true)

	if rl.cursor.Pos() != startPos {
		rl.cursor.Dec()
	}

	epos := rl.cursor.Pos()
	rl.line.Cut(bpos, epos)
	rl.cursor.Set(bpos)
}

// Delete the current word from the cursor point up to the end of it.
func (rl *Shell) deleteWord() {
	rl.History.Save()

	rl.selection.Mark(rl.cursor.Pos())
	forward := rl.line.ForwardEnd(rl.line.Tokenize, rl.cursor.Pos())
	rl.cursor.Move(forward)

	rl.selection.Cut()
}

// Quote the region from the cursor to the mark.
func (rl *Shell) quoteRegion() {
	rl.History.Save()

	rl.selection.Surround('\'', '\'')
	rl.cursor.Inc()
}

// Quote the entire line buffer.
func (rl *Shell) quoteLine() {
	if rl.line.Len() == 0 {
		return
	}

	for pos, char := range *rl.line {
		if char == '\n' {
			break
		}

		if char == '\'' {
			(*rl.line)[pos] = '"'
		}
	}

	rl.line.Insert(0, '\'')
	rl.line.Insert(rl.line.Len(), '\'')
}

// Modifies the current word under the cursor, increasing it.
// The following word types can be incremented/decremented:
//
//	Booleans: true|false, t|f, on|off, yes|no, y|n.
//	Operators: &&|||, ++|--, ==|!=, ===| !==, +| -, -| *, *| /, /| +, and| or.
//	Hex digits 0xDe => 0xdf, 0xdE => 0xDF, 0xde0 => 0xddf, 0xffffffffffffffff => 0x0000000000000000.
//	Binary digits: 0b1 => 0b10, 0B0 => 0B1, etc.
//	Integers.
func (rl *Shell) keywordIncrease() {
	rl.History.Save()
	rl.keywordSwitch(true)
}

// Modifies the current word under the cursor, decreasing it.
// The following word types can be incremented/decremented:
//
//	Booleans: true|false, t|f, on|off, yes|no, y|n.
//	Operators: &&|||, ++|--, ==|!=, ===| !==, +| -, -| *, *| /, /| +, and| or.
//	Hex digits 0xDe => 0xdf, 0xdE => 0xDF, 0xde0 => 0xddf, 0xffffffffffffffff => 0x0000000000000000.
//	Binary digits: 0b1 => 0b10, 0B0 => 0B1, etc.
//	Integers.
func (rl *Shell) keywordDecrease() {
	rl.History.Save()
	rl.keywordSwitch(false)
}

// Switches the current word under the cursor, increasing or decreasing it.
func (rl *Shell) keywordSwitch(increase bool) {
	cpos := strutil.AdjustNumberOperatorPos(rl.cursor.Pos(), *rl.line)

	// Select in word and get the selection positions
	bpos, epos := rl.line.SelectWord(cpos)
	epos++

	// Move the cursor backward if needed/possible
	if bpos != 0 && ((*rl.line)[bpos-1] == '+' || (*rl.line)[bpos-1] == '-') {
		bpos--
	}

	// Get the selection string
	selection := string((*rl.line)[bpos:epos])

	// For each of the keyword handlers, run it, which returns
	// false/none if didn't operate, then continue to next handler.
	for _, switcher := range strutil.KeywordSwitchers() {
		vii := rl.Iterations.Get()

		changed, word, obpos, oepos := switcher(selection, increase, vii)
		if !changed {
			continue
		}

		// We are only interested in the end position after all runs
		epos = bpos + oepos
		bpos += obpos

		if cpos < bpos || cpos >= epos {
			continue
		}

		// Update the line and the cursor, and return
		// since we have a handler that has been ran.
		begin := string((*rl.line)[:bpos])
		end := string((*rl.line)[epos:])

		newLine := append([]rune(begin), []rune(word)...)
		newLine = append(newLine, []rune(end)...)
		rl.line.Set(newLine...)
		rl.cursor.Set(bpos + len(word) - 1)

		return
	}
}

//
// Killing & Yanking ----------------------------------------------------------
//

// Kill from the cursor to the end of the line. If already
// on the end of the line, kill the newline character.
func (rl *Shell) killLine() {
	rl.Iterations.Reset()
	rl.History.Save()

	if rl.line.Len() == 0 {
		return
	}

	cpos := rl.cursor.Pos()
	rl.cursor.EndOfLineAppend()

	rl.selection.MarkRange(cpos, rl.cursor.Pos())
	text := rl.selection.Cut()

	rl.Buffers.Write([]rune(text)...)
	rl.cursor.Set(cpos)
}

// Kill backward to the beginning of the line.
func (rl *Shell) backwardKillLine() {
	rl.Iterations.Reset()
	rl.History.Save()

	if rl.line.Len() == 0 {
		return
	}

	cpos := rl.cursor.Pos()
	rl.cursor.BeginningOfLine()

	rl.selection.MarkRange(rl.cursor.Pos(), cpos)
	text := rl.selection.Cut()

	rl.Buffers.Write([]rune(text)...)
}

// Kill all characters on the current line, no matter where point is.
func (rl *Shell) killWholeLine() {
	rl.History.Save()

	if rl.line.Len() == 0 {
		return
	}

	rl.Buffers.Write(*rl.line...)
	rl.line.Cut(0, rl.line.Len())
}

// Kill the entire buffer.
func (rl *Shell) killBuffer() {
	rl.History.Save()

	if rl.line.Len() == 0 {
		return
	}

	rl.Buffers.Write(*rl.line...)
	rl.line.Cut(0, rl.line.Len())
}

// Kill the current word from the cursor point up to the end of it.
func (rl *Shell) killWord() {
	rl.History.Save()

	bpos := rl.cursor.Pos()

	rl.cursor.ToFirstNonSpace(true)
	forward := rl.line.Forward(rl.line.TokenizeSpace, rl.cursor.Pos())
	rl.cursor.Move(forward - 1)
	epos := rl.cursor.Pos()

	rl.selection.MarkRange(bpos, epos)
	rl.Buffers.Write([]rune(rl.selection.Cut())...)
	rl.cursor.Set(bpos)
}

// Kill the word behind point. Word boundaries
// are the same as those used by backward-word.
func (rl *Shell) backwardKillWord() {
	rl.History.Save()
	rl.History.SkipSave()

	rl.selection.Mark(rl.cursor.Pos())
	adjust := rl.line.Backward(rl.line.Tokenize, rl.cursor.Pos())
	rl.cursor.Move(adjust)

	rl.Buffers.Write([]rune(rl.selection.Cut())...)
}

// Kill the text between the point and mark (saved cursor
// position).  This text is referred to as the region.
func (rl *Shell) killRegion() {
	rl.History.Save()

	if !rl.selection.Active() {
		return
	}

	rl.Buffers.Write([]rune(rl.selection.Cut())...)
}

// Copy the text in the region to the kill buffer.
func (rl *Shell) copyRegionAsKill() {
	rl.History.SkipSave()

	if !rl.selection.Active() {
		return
	}

	rl.Buffers.Write([]rune(rl.selection.Text())...)
	rl.selection.Reset()
}

// Copy the word before point to the kill buffer.
// The word boundaries are the same as backward-word.
func (rl *Shell) copyBackwardWord() {
	rl.History.Save()

	rl.selection.Mark(rl.cursor.Pos())
	adjust := rl.line.Backward(rl.line.Tokenize, rl.cursor.Pos())
	rl.cursor.Move(adjust)

	rl.Buffers.Write([]rune(rl.selection.Text())...)
	rl.selection.Reset()
}

// Copy the word following point to the kill buffer.
// The word boundaries are the same as forward-word.
func (rl *Shell) copyForwardWord() {
	rl.History.Save()

	rl.selection.Mark(rl.cursor.Pos())
	adjust := rl.line.Forward(rl.line.Tokenize, rl.cursor.Pos())
	rl.cursor.Move(adjust + 1)

	rl.Buffers.Write([]rune(rl.selection.Text())...)
	rl.selection.Reset()
}

// Yank the top of the kill ring into the buffer at point.
func (rl *Shell) yank() {
	buf := rl.Buffers.Active()

	vii := rl.Iterations.Get()

	for i := 1; i <= vii; i++ {
		rl.cursor.InsertAt(buf...)
	}
}

// Rotate the kill ring, and yank the new top.
// Only works following yank or yank-pop.
func (rl *Shell) yankPop() {
	vii := rl.Iterations.Get()

	for i := 1; i <= vii; i++ {
		buf := rl.Buffers.Pop()
		rl.cursor.InsertAt(buf...)
	}
}

// Kill the shell word behind point. Word boundaries
// are the same as those used by backward-word.
func (rl *Shell) shellKillWord() {
	startPos := rl.cursor.Pos()

	// select the shell word, and if the cursor position
	// has changed, we delete the part after the initial one.
	rl.viSelectAShellWord()

	_, epos := rl.selection.Pos()

	rl.Buffers.Write([]rune((*rl.line)[startPos:epos])...)
	rl.line.Cut(startPos, epos)
	rl.cursor.Set(startPos)

	rl.selection.Reset()
}

// Kill the shell word behind point.
func (rl *Shell) shellBackwardKillWord() {
	startPos := rl.cursor.Pos()
	if rl.line.Len() == 0 || startPos == 0 {
		return
	}

	rl.History.Save()

	// Always ignore the character under cursor.
	rl.cursor.Dec()
	rl.cursor.ToFirstNonSpace(false)

	unclosed, bpos := strutil.GetQuotedWordStart((*rl.line)[:startPos])

	if !unclosed {
		rl.viSelectAShellWord()
		bpos, _ = rl.selection.Pos()
	}

	rl.cursor.Set(bpos)
	rl.cursor.ToFirstNonSpace(true)
	bpos = rl.cursor.Pos()

	rl.Buffers.Write([]rune((*rl.line)[bpos:startPos])...)
	rl.line.Cut(bpos, startPos)
	rl.selection.Reset()
}

// Like copy-prev-word, but the word is found by using shell parsing,
// whereas copy-prev-word looks for blanks. This makes a difference
// when the word is quoted and contains spaces.
func (rl *Shell) copyPrevShellWord() {
	rl.History.Save()

	posInit := rl.cursor.Pos()

	// First go back to the beginning of the current word,
	// then go back again to the beginning of the previous.
	rl.backwardShellWord()
	rl.backwardShellWord()

	// Select the current shell word
	rl.viSelectAShellWord()

	word := rl.selection.Text()

	// Replace the cursor before reassembling the line.
	rl.cursor.Set(posInit)
	rl.selection.InsertAt(rl.cursor.Pos(), -1)
	rl.cursor.Move(len(word))
}

//
// Numeric Arguments -----------------------------------------------------------
//

// digitArgument is used both in Emacs and Vim modes,
// but strips the Alt modifier used in Emacs mode.
func (rl *Shell) digitArgument() {
	rl.History.SkipSave()
	keys := rl.Keys.Caller()

	// Strip the Alt modifier.
	if len(keys) > 1 && keys[0] == inputrc.Esc {
		keys = keys[1:]
	}

	rl.Iterations.Add(string(keys))
}

//
// Macros ----------------------------------------------------------------------
//

// Begin saving the characters typed into the current keyboard macro.
func (rl *Shell) startKeyboardMacro() {
	rl.Macros.StartRecord(rune(0))
}

// Stop saving the characters typed into the current
// keyboard macro and store the definition.
func (rl *Shell) endKeyboardMacro() {
	rl.Macros.StopRecord()
}

// Re-execute the last keyboard macro defined, by making the
// characters in the macro appear as if typed at the keyboard.
func (rl *Shell) callLastKeyboardMacro() {
	rl.Macros.RunLastMacro()
}

// Print the last keyboard macro defined in a format suitable for the inputrc file.
func (rl *Shell) printLastKeyboardMacro() {
	rl.Display.ClearHelpers()

	rl.Macros.PrintLastMacro()

	rl.Prompt.PrimaryPrint()
	rl.Display.Refresh()
}

// Either starts recording a macro (if not yet recording), or stops it.
// If the command is about to start recording a macro, it will read an
// additional argument key (must be a letter), to be used as the macro
// "name", just like macro recording and use work in Vim.
// This command thus works "Vim-style", and should probably be used only
// when using Vim editing mode.
func (rl *Shell) macroToggleRecord() {
	if rl.Macros.Recording() {
		rl.Macros.StopRecord()

		return
	}

	done := rl.Keymap.PendingCursor()
	defer done()

	rl.Hint.SetTemporary(color.Dim + "REC (macro arg)")
	rl.Display.Refresh()

	key, isAbort := rl.Keys.ReadKey()
	if isAbort {
		return
	}

	rl.Macros.StartRecord(key)
}

// Reads a key from the keyboard, and runs the macro stored for this key identitier.
// This mimics the Vim-style or running macros. If no macro is recorded for this key,
// or if the key is invalid, nothing happens.
func (rl *Shell) macroRun() {
	done := rl.Keymap.PendingCursor()
	defer done()

	rl.Hint.SetTemporary(color.Dim + "Run (macro arg)")
	rl.Display.Refresh()

	key, isAbort := rl.Keys.ReadKey()
	if isAbort {
		return
	}

	rl.Macros.RunMacro(key)
}

//
// Miscellaneous ---------------------------------------------------------------
//

// Read in the contents of the inputrc file, and incorporate
// any bindings or variable assignments found there.
func (rl *Shell) reReadInitFile() {
	main := rl.Keymap.Main()

	err := rl.Keymap.ReloadConfig(rl.Opts...)
	if err != nil {
		rl.Hint.SetTemporary(color.FgRed + "Inputrc reload error: " + err.Error())
		return
	}

	defer rl.Keymap.UpdateCursor()

	// Reload keymap settings and cursor
	newMain := rl.Keymap.Main()

	if main != newMain {
		switch newMain {
		case keymap.Emacs, keymap.EmacsStandard, keymap.EmacsMeta, keymap.EmacsCtrlX:
			rl.emacsEditingMode()
		case keymap.Vi, keymap.ViCommand, keymap.ViMove:
			rl.viCommandMode()
		case keymap.ViInsert:
			rl.viInsertMode()
		}
	}

	// Notify successfully reloaded
	rl.Hint.SetTemporary(color.FgGreen + "Inputrc reloaded")
}

// Abort the current editing command.
// If one of the completion or non/incremental-search modes
// are active, only cancel them and nothing else.
func (rl *Shell) abort() {
	// Reset any visual selection and iterations.
	rl.Iterations.Reset()
	rl.selection.Reset()

	// Cancel active completion insertion and/or incremental search.
	if rl.completer.AutoCompleting() || rl.completer.IsInserting() {
		rl.Hint.Reset()
		rl.completer.ResetForce()

		return
	}

	// Cancel non-incremental search modes.
	searching, _, _ := rl.completer.NonIncrementallySearching()
	if searching {
		rl.completer.NonIsearchStop()
		return
	}

	// And only return to the caller if the abort was
	// called by one of the builtin/config terminators.
	// All others should generally be OS signals.
	if !rl.Keymap.InputIsTerminator() {
		return
	}

	if rl.Config.GetBool("echo-control-characters") {
		key := rl.Keys.Caller()
		if key[0] == rune(inputrc.Unescape(`\C-C`)[0]) {
			quoted, _ := strutil.Quote(key[0])
			fmt.Print(string(quoted))
		}
	}

	// If no line was active,
	rl.Display.AcceptLine()
	rl.History.Accept(false, false, ErrInterrupt)
}

// If the metafied character x is uppercase, run the command
// that is bound to the corresponding metafied lowercase character.
// The behavior is undefined if x is already lowercase.
func (rl *Shell) doLowercaseVersion() {
	rl.History.SkipSave()

	keys := rl.Keys.Caller()

	escapePrefix := false

	// Get rid of the escape if it's a prefix
	if len(keys) > 1 && keys[0] == inputrc.Esc {
		escapePrefix = true
		keys = keys[1:]
	} else if len(keys) == 1 && inputrc.IsMeta(keys[0]) {
		keys = []rune{inputrc.Demeta(keys[0])}
	}

	// Undefined behavior if the key is already lowercase.
	if unicode.IsLower(keys[0]) {
		return
	}

	keys[0] = unicode.ToLower(keys[0])

	// Feed back the keys with meta prefix or encoding
	if escapePrefix {
		input := append([]rune{inputrc.Esc}, keys...)
		rl.Keys.Feed(false, input...)
	} else {
		rl.Keys.Feed(false, inputrc.Enmeta(keys[0]))
	}
}

// Metafy the next character typed.  ESC f is equivalent to Meta-f.
func (rl *Shell) prefixMeta() {
	rl.History.SkipSave()

	done := rl.Keymap.PendingCursor()
	defer done()

	key, isAbort := rl.Keys.ReadKey()
	if isAbort {
		return
	}

	// And feed them back to be used on the next loop.
	keys := append([]rune{inputrc.Esc}, key)
	rl.Keys.Feed(false, keys...)
}

// Incrementally undo the last text modification.
// Note that when invoked from vi command mode, the full
// prior change made in insert mode is reverted, the changes
// having been merged when command mode was selected.
func (rl *Shell) undoLast() {
	rl.History.Undo()
}

// Undo all changes made to this line.
// This is like executing the undo command enough
// times to return the line to its initial state.
func (rl *Shell) revertLine() {
	rl.History.Revert()
}

// Set the mark to the point. If a numeric argument is
// supplied, the mark is set to that position.
func (rl *Shell) setMark() {
	switch {
	case rl.Iterations.IsSet():
		rl.cursor.SetMark()
	default:
		cpos := rl.cursor.Pos()
		mark := rl.Iterations.Get()

		if mark > rl.line.Len()-1 {
			return
		}

		rl.cursor.Set(mark)
		rl.cursor.SetMark()
		rl.cursor.Set(cpos)
	}
}

// Swap the point with the mark.  The current cursor position
// is set to the saved position, and the old cursor position
// is saved as the mark.
func (rl *Shell) exchangePointAndMark() {
	// Deactivate mark if out of bound
	if rl.cursor.Mark() > rl.line.Len() {
		rl.cursor.ResetMark()
	}

	// And set it to start if negative.
	if rl.cursor.Mark() < 0 {
		cpos := rl.cursor.Pos()
		rl.cursor.Set(0)
		rl.cursor.SetMark()
		rl.cursor.Set(cpos)
	} else {
		mark := rl.cursor.Mark()

		rl.cursor.SetMark()
		rl.cursor.Set(mark)

		rl.selection.MarkRange(rl.cursor.Mark(), rl.cursor.Pos())
		rl.selection.Visual(false)
	}
}

// A character is read and point is moved to the next
// occurrence of that character.  A negative argument
// searches for previous occurrences.
func (rl *Shell) characterSearch() {
	if rl.Iterations.Get() < 0 {
		rl.viFindChar(false, false)
	} else {
		rl.viFindChar(true, false)
	}
}

// A character is read and point is moved to the previous
// occurrence of that character.  A negative argument
// searches for subsequent occurrences.
func (rl *Shell) characterSearchBackward() {
	if rl.Iterations.Get() < 0 {
		rl.viFindChar(true, false)
	} else {
		rl.viFindChar(false, false)
	}
}

// Without a numeric argument, the value of the readline
// comment-begin variable is inserted at the beginning of the
// current line.  If a numeric argument is supplied, this
// command acts as a toggle: if the characters at the
// beginning of the line do not match the value of
// comment-begin, the value is inserted, otherwise the
// characters in comment-begin are deleted from the beginning
// of the line.  In either case, the line is accepted as if a
// newline had been typed.  The default value of
// comment-begin makes the current line a shell comment.
// If a numeric argument causes the comment character to be
// removed, the line will be executed by the shell.
func (rl *Shell) insertComment() {
	comment := strings.Trim(rl.Config.GetString("comment-begin"), "\"")

	switch {
	case !rl.Iterations.IsSet():
		// Without numeric argument, insert comment at the beginning of the line.
		cpos := rl.cursor.Pos()
		rl.cursor.BeginningOfLine()
		rl.cursor.InsertAt([]rune(comment)...)
		rl.cursor.Set(cpos)

	default:
		// Or with one, toggle the current line commenting.
		cpos := rl.cursor.Pos()
		rl.cursor.BeginningOfLine()

		bpos := rl.cursor.Pos()
		epos := bpos + len(comment)

		rl.cursor.Set(cpos)

		commentFits := epos < rl.line.Len()

		if commentFits && string((*rl.line)[bpos:epos]) == comment {
			rl.line.Cut(bpos, epos)
			rl.cursor.Move(-1 * len(comment))
		} else {
			rl.line.Insert(bpos, []rune(comment)...)
			rl.cursor.Move(1 * len(comment))
		}
	}

	// Either case, accept the line as it is.
	rl.acceptLineWith(false, false)
}

// Print all of the functions and their key bindings to the
// readline output stream.  If a numeric argument is
// supplied, the output is formatted in such a way that it
// can be made part of an inputrc file.
func (rl *Shell) dumpFunctions() {
	rl.Display.ClearHelpers()
	fmt.Println()

	defer func() {
		rl.Prompt.PrimaryPrint()
		rl.Display.Refresh()
	}()

	inputrcFormat := rl.Iterations.IsSet()
	rl.Keymap.PrintBinds(string(rl.Keymap.Main()), inputrcFormat)
}

// Print all of the settable variables and their values to
// the readline output stream.  If a numeric argument is
// supplied, the output is formatted in such a way that it
// can be made part of an inputrc file.
func (rl *Shell) dumpVariables() {
	rl.Display.ClearHelpers()
	fmt.Println()

	defer func() {
		rl.Prompt.PrimaryPrint()
		rl.Display.Refresh()
	}()

	// Get all variables and their values, alphabetically sorted.
	var variables []string

	for variable := range rl.Config.Vars {
		variables = append(variables, variable)
	}

	sort.Strings(variables)

	// Either print in inputrc format, or wordly one.
	if rl.Iterations.IsSet() {
		for _, variable := range variables {
			value := rl.Config.Vars[variable]
			fmt.Printf("set %s %v\n", variable, value)
		}
	} else {
		for _, variable := range variables {
			value := rl.Config.Vars[variable]
			fmt.Printf("%s is set to `%v'\n", variable, value)
		}
	}
}

// Print all of the readline key sequences bound to macros
// and the strings they output.  If a numeric argument is
// supplied, the output is formatted in such a way that it
// can be made part of an inputrc file.
func (rl *Shell) dumpMacros() {
	rl.Display.ClearHelpers()
	fmt.Println()

	defer func() {
		rl.Prompt.PrimaryPrint()
		rl.Display.Refresh()
	}()

	// We print the macros bound to the current keymap only.
	binds := rl.Config.Binds[string(rl.Keymap.Main())]
	if len(binds) == 0 {
		return
	}

	var macroBinds []string

	for keys, bind := range binds {
		if bind.Macro {
			macroBinds = append(macroBinds, inputrc.Escape(keys))
		}
	}

	sort.Strings(macroBinds)

	if rl.Iterations.IsSet() {
		for _, key := range macroBinds {
			action := inputrc.Escape(binds[inputrc.Unescape(key)].Action)
			fmt.Printf("\"%s\": \"%s\"\n", key, action)
		}
	} else {
		for _, key := range macroBinds {
			action := inputrc.Escape(binds[inputrc.Unescape(key)].Action)
			fmt.Printf("%s outputs %s\n", key, action)
		}
	}
}

// Invoke an editor on the current command line, and execute the result as shell commands.
// Readline attempts to invoke $VISUAL, $EDITOR, and emacs as the editor, in that order.
func (rl *Shell) editAndExecuteCommand() {
	buffer := *rl.line

	// Edit in editor
	edited, err := rl.Buffers.EditBuffer(buffer, "", "", rl.Keymap.IsEmacs())
	if err != nil || (len(edited) == 0 && len(buffer) != 0) {
		rl.History.SkipSave()

		errStr := strings.ReplaceAll(err.Error(), "\n", "")
		changeHint := fmt.Sprintf(color.FgRed+"Editor error: %s", errStr)
		rl.Hint.SetTemporary(changeHint)

		return
	}

	// Update our line and return it the caller.
	rl.line.Set(edited...)
	rl.Display.AcceptLine()
	rl.History.Accept(false, false, nil)
}

func (rl *Shell) editCommandLine() {
	buffer := *rl.line
	keymapCur := rl.Keymap.Main()

	// Edit in editor
	edited, err := rl.Buffers.EditBuffer(buffer, "", "", rl.Keymap.IsEmacs())
	if err != nil || (len(edited) == 0 && len(buffer) != 0) {
		rl.History.SkipSave()

		errStr := strings.ReplaceAll(err.Error(), "\n", "")
		changeHint := fmt.Sprintf(color.FgRed+"Editor error: %s", errStr)
		rl.Hint.SetTemporary(changeHint)

		return
	}

	// Update our line
	rl.line.Set(edited...)

	// We're done with visual mode when we were in.
	switch keymapCur {
	case keymap.Emacs, keymap.EmacsStandard, keymap.EmacsMeta, keymap.EmacsCtrlX:
		rl.emacsEditingMode()
	}
}

// Incrementally redo undone text modifications.
func (rl *Shell) redo() {
	rl.History.Redo()
}

// Considers the blank word under cursor, and tries a series of regular expressions on it
// to match various patterns: URL and their various subcomponents (host/path/params, etc).
//
// When one of the regular expressions succeeds, the match is visually selected,
// otherwise nothing is selected (if selection was active, it will stay the same)
//
// When repeatedly calling this function while in visual selection mode, the shell will
// cycle through either the current matcher's capturing subgroups (such as the parts of a URL),
// or cycle through the next matcher (for instance, attempting to grap an IP after trying URL).
func (rl *Shell) selectKeywordNext() {
	rl.History.SkipSave()

	// Always try to find a match within the blank word under cursor.
	bpos, epos := rl.line.SelectBlankWord(rl.cursor.Pos())

	// Run the regexp matchers.
	_, epos, match := rl.selection.SelectKeyword(bpos, epos, true)
	if !match {
		return
	}

	// The matchers succeeded, we now have a selection active,
	// but the cursor should be moved to the end of it.
	rl.cursor.Set(epos)
	rl.selection.Visual(false)
}

// Identical to select-keyword-prev, except that the matcher/subgroup cycling occurs backward.
func (rl *Shell) selectKeywordPrev() {
	rl.History.SkipSave()

	// Always try to find a match within the blank word under cursor.
	bpos, epos := rl.line.SelectBlankWord(rl.cursor.Pos())

	// Run the regexp matchers.
	_, epos, match := rl.selection.SelectKeyword(bpos, epos, false)
	if !match {
		return
	}

	// The matchers succeeded, we now have a selection active,
	// but the cursor should be moved to the end of it.
	rl.cursor.Set(epos)
	rl.selection.Visual(false)
}
