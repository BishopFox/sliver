package readline

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
)

var rxMultiline = regexp.MustCompile(`[\r\n]+`)

// Readline displays the readline prompt.
// It will return a string (user entered data) or an error.
func (rl *Instance) Readline() (string, error) {
	fd := int(os.Stdin.Fd())
	state, err := MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer Restore(fd, state)

	// Prompt Init
	// Here we have to either print prompt and return new line (multiline)
	if rl.Multiline {
		fmt.Println(rl.mainPrompt)
	}
	rl.stillOnRefresh = false
	rl.computePrompt() // initialise the prompt for first print

	// Line Init & Cursor
	rl.line = []rune{}
	rl.currentComp = []rune{} // No virtual completion yet
	rl.lineComp = []rune{}    // So no virtual line either
	rl.modeViMode = vimInsert

	// rl.pos has become the "netted" cursor
	// position, so taking the cursor into account
	rl.pos = 0
	rl.posY = 0

	// Completion && hints init
	rl.resetHintText()
	rl.resetTabCompletion()
	rl.getHintText()

	// History Init
	// We need this set to the last command, so that we can access it quickly
	rl.histPos = 0
	rl.viUndoHistory = []undoItem{{line: "", pos: 0}}

	// Multisplit
	if len(rl.multisplit) > 0 {
		r := []rune(rl.multisplit[0])
		rl.editorInput(r)
		rl.carridgeReturn()
		if len(rl.multisplit) > 1 {
			rl.multisplit = rl.multisplit[1:]
		} else {
			rl.multisplit = []string{}
		}
		return string(rl.line), nil
	}

	// Finally, print any hints or completions
	// if the TabCompletion engines so desires
	rl.renderHelpers()

	for {
		rl.viUndoSkipAppend = false
		b := make([]byte, 1024)
		var i int

		if !rl.skipStdinRead {
			var err error
			i, err = os.Stdin.Read(b)
			if err != nil {
				return "", err
			}
		}

		rl.skipStdinRead = false
		r := []rune(string(b))

		if isMultiline(r[:i]) || len(rl.multiline) > 0 {
			rl.multiline = append(rl.multiline, b[:i]...)
			if i == len(b) {
				continue
			}

			if !rl.allowMultiline(rl.multiline) {
				rl.multiline = []byte{}
				continue
			}

			s := string(rl.multiline)
			rl.multisplit = rxMultiline.Split(s, -1)

			r = []rune(rl.multisplit[0])
			rl.modeViMode = vimInsert
			rl.editorInput(r)
			rl.carridgeReturn()
			rl.multiline = []byte{}
			if len(rl.multisplit) > 1 {
				rl.multisplit = rl.multisplit[1:]
			} else {
				rl.multisplit = []string{}
			}
			return string(rl.line), nil
		}

		s := string(r[:i])
		if rl.evtKeyPress[s] != nil {
			rl.clearHelpers()

			ret := rl.evtKeyPress[s](s, rl.line, rl.pos)

			rl.clearLine()
			rl.line = append(ret.NewLine, []rune{}...)
			rl.updateHelpers() // rl.echo
			rl.pos = ret.NewPos

			if ret.ClearHelpers {
				rl.resetHelpers()
			} else {
				rl.updateHelpers()
			}

			if len(ret.HintText) > 0 {
				rl.hintText = ret.HintText
				rl.clearHelpers()
				rl.renderHelpers()
			}
			if !ret.ForwardKey {
				continue
			}
			if ret.CloseReadline {
				rl.clearHelpers()
				return string(rl.line), nil
			}
		}

		switch b[0] {
		case charCtrlC:
			rl.clearHelpers()
			return "", CtrlC

		case charEOF:
			rl.clearHelpers()
			return "", EOF

		case charCtrlF:
			rl.resetVirtualComp()

			if !rl.modeTabCompletion {
				rl.modeTabCompletion = true
			}

			// Both these settings apply to when we already
			// are in completion mode and when we are not.
			rl.searchMode = CompletionFind
			rl.modeAutoFind = true

			// Switch from history to completion search
			if rl.modeTabCompletion && rl.searchMode == HistoryFind {
				rl.searchMode = CompletionFind
			}

			rl.updateTabFind([]rune{})
			rl.viUndoSkipAppend = true

		case charCtrlR:
			rl.resetVirtualComp()

			rl.mainHist = true // false before
			rl.searchMode = HistoryFind
			rl.modeAutoFind = true
			rl.modeTabCompletion = true

			rl.modeTabFind = true
			rl.updateTabFind([]rune{})
			rl.viUndoSkipAppend = true

		case charCtrlE:
			rl.resetVirtualComp()

			rl.mainHist = false // true before
			rl.searchMode = HistoryFind
			rl.modeAutoFind = true
			rl.modeTabCompletion = true

			rl.modeTabFind = true
			rl.updateTabFind([]rune{})
			rl.viUndoSkipAppend = true

		case charCtrlG:
			if rl.modeAutoFind {
				rl.resetTabFind()
				rl.resetHelpers()
				rl.renderHelpers()
			}

		case charCtrlU:
			rl.resetVirtualComp()

			rl.clearLine()
			rl.resetHelpers()

		case charTab:
			if rl.modeTabCompletion && !rl.compConfirmWait {
				rl.tabCompletionSelect = true
				rl.moveTabCompletionHighlight(1, 0)
				rl.updateVirtualComp()
				rl.renderHelpers()
				rl.viUndoSkipAppend = true
			} else {
				rl.getTabCompletion()

				// If too many completions and no yet confirmed, ask user for completion
				comps, lines := rl.getCompletionCount()
				if rl.compConfirmWait {
				}
				if ((lines > GetTermLength()) || (lines > rl.MaxTabCompleterRows)) && !rl.compConfirmWait {
					sentence := fmt.Sprintf("%s show all %d completions (%d lines) ?",
						FOREWHITE, comps, lines)
					rl.promptCompletionConfirm(sentence)
					continue
				}

				rl.compConfirmWait = false
				rl.modeTabCompletion = true

				// Also here, if only one candidate is available, automatically
				// insert it and don't bother printing completions.
				// Quit the tab completion mode to avoid asking to the user to press
				// Enter twice to actually run the command
				if rl.hasOneCandidate() {
					rl.insertCandidate()

					// Refresh first, and then quit the completion mode
					// rl.renderHelpers()
					rl.updateHelpers()
					rl.viUndoSkipAppend = true
					rl.resetTabCompletion()
					continue
				}

				rl.updateHelpers()
				// rl.renderHelpers()
				rl.viUndoSkipAppend = true
				continue
			}

			// Once we have a completion candidate, insert it in the virtual input line.
			// This will thus not filter other candidates, despite printing the current one.
			// rl.updateVirtualComp()

			// rl.renderHelpers()
			// rl.updateHelpers()
			// rl.viUndoSkipAppend = true

		// Clear the entire screen. Reprints completions if they were shown.
		case charCtrlL:
			print(seqClearScreen)
			print(seqCursorTopLeft)
			if rl.Multiline {
				fmt.Println(rl.mainPrompt)
			}
			print(seqClearScreenBelow)

			rl.resetHintText()
			rl.getHintText()
			rl.renderHelpers()

		case '\r':
			fallthrough
		case '\n':
			if rl.modeTabCompletion {
				// if rl.modeTabCompletion && !rl.modeTabFind {
				cur := rl.getCurrentGroup()

				// Check that there is a group indeed, as we might have no completions.
				if cur == nil {
					rl.clearHelpers()
					rl.resetTabCompletion()
					rl.renderHelpers()
					continue
				}

				// IF we have a prefix and completions printed, but no candidate
				// (in which case the completion is ""), we immediately return.
				completion := cur.getCurrentCell(rl)
				prefix := len(rl.tcPrefix)
				if prefix > len(completion) {
					rl.carridgeReturn()
					return string(rl.line), nil
				}

				// Else, we insert the completion candidate in the real input line.
				// This is in fact nothing more than assigning the virtual input line.
				// By default we add a space, unless completion group asks otherwise.
				rl.compAddSpace = true
				rl.resetVirtualComp()

				// Reset completions and update input line
				rl.clearHelpers()
				rl.resetTabCompletion()
				rl.renderHelpers()

				continue
			}
			rl.carridgeReturn()
			return string(rl.line), nil

		case charBackspace, charBackspace2:
			if rl.modeTabFind || rl.modeAutoFind {
				rl.backspaceTabFind()
				rl.viUndoSkipAppend = true
			} else {
				rl.resetVirtualComp()

				rl.backspace()
				rl.renderHelpers()
			}

		case charEscape:
			// We always refresh the completion candidates, except if we are currently
			// cycling through them, because then it would just append the candidate.
			if rl.modeTabCompletion {
				if string(r[:i]) != seqShiftTab &&
					string(r[:i]) != seqForwards && string(r[:i]) != seqBackwards &&
					string(r[:i]) != seqUp && string(r[:i]) != seqDown {
					rl.resetVirtualComp()
				}
			}

			// If we are in a prompt completion confirm, we escape it
			if rl.compConfirmWait {
				rl.compConfirmWait = false
			}

			rl.escapeSeq(r[:i])

		default:
			rl.resetVirtualComp()

			// Not sure that CompletionFind is useful, nor one of the other two
			if rl.modeAutoFind || rl.modeTabFind {
				// if rl.modeAutoFind || rl.modeTabFind && rl.searchMode == CompletionFind {
				rl.updateTabFind(r[:i])
				rl.viUndoSkipAppend = true
			} else {
				rl.editorInput(r[:i])
				if len(rl.multiline) > 0 && rl.modeViMode == vimKeys {
					rl.skipStdinRead = true
				}
			}
		}

		// if !rl.viUndoSkipAppend {
		//         rl.viUndoHistory = append(rl.viUndoHistory, rl.line)
		// }
		rl.undoAppendHistory()
	}
}

func (rl *Instance) escapeSeq(r []rune) {
	switch string(r) {
	case string(charEscape):
		switch {
		case rl.modeAutoFind:
			rl.resetTabFind()
			rl.clearHelpers()
			rl.resetTabCompletion()
			rl.resetHelpers()
			rl.renderHelpers()

		case rl.modeTabFind:
			rl.resetTabFind()
			rl.resetTabCompletion()

		case rl.modeTabCompletion:
			rl.clearHelpers()
			rl.resetTabCompletion()
			rl.renderHelpers()

		default:
			// If we are in Vim mode, the escape key has its usage.
			// Otherwise in emacs mode the escape key does nothing.
			if rl.InputMode == Vim {
				if rl.pos == len(rl.line) && len(rl.line) > 0 {
					rl.pos--
					// moveCursorBackwards(1)
				}

				rl.modeViMode = vimKeys
				rl.viIteration = ""
				rl.refreshVimStatus()

				// This refreshed and actually prints the new Vim status
				rl.clearHelpers()
				rl.renderHelpers()
			}

		}
		rl.viUndoSkipAppend = true

	case seqDelete:
		if rl.modeTabFind {
			rl.backspaceTabFind()
		} else {
			rl.delete()
		}

	case seqUp:
		if rl.modeTabCompletion {
			rl.tabCompletionSelect = true
			rl.tabCompletionReverse = true
			rl.moveTabCompletionHighlight(-1, 0)
			rl.updateVirtualComp()
			rl.tabCompletionReverse = false
			rl.renderHelpers()
			return
		}
		rl.walkHistory(-1)

	case seqDown:
		if rl.modeTabCompletion {
			rl.tabCompletionSelect = true
			rl.moveTabCompletionHighlight(1, 0)
			rl.updateVirtualComp()
			rl.renderHelpers()
			return
		}
		rl.walkHistory(1)

	case seqBackwards:
		if rl.modeTabCompletion {
			rl.tabCompletionSelect = true
			rl.tabCompletionReverse = true
			rl.moveTabCompletionHighlight(-1, 0)
			rl.updateVirtualComp()
			rl.tabCompletionReverse = false
			rl.renderHelpers()
			return
		}
		if rl.pos > 0 {
			moveCursorBackwards(1)
			rl.pos--
		}
		rl.viUndoSkipAppend = true

	case seqForwards:
		if rl.modeTabCompletion {
			rl.tabCompletionSelect = true
			rl.moveTabCompletionHighlight(1, 0)
			rl.updateVirtualComp()
			rl.renderHelpers()
			return
		}
		if (rl.modeViMode == vimInsert && rl.pos < len(rl.line)) ||
			(rl.modeViMode != vimInsert && rl.pos < len(rl.line)-1) {
			moveCursorForwards(1)
			rl.pos++
		}
		rl.viUndoSkipAppend = true

	case seqHome, seqHomeSc:
		if rl.modeTabCompletion {
			return
		}
		moveCursorBackwards(rl.pos)
		rl.pos = 0
		rl.viUndoSkipAppend = true

	case seqEnd, seqEndSc:
		if rl.modeTabCompletion {
			return
		}
		moveCursorForwards(len(rl.line) - rl.pos)
		rl.pos = len(rl.line)
		rl.viUndoSkipAppend = true

	case seqShiftTab:
		if rl.modeTabCompletion && !rl.compConfirmWait {

			rl.tabCompletionReverse = true // The group will use this to know how to index.

			rl.moveTabCompletionHighlight(-1, 0)

			// Once we have a completion candidate, insert it in the virtual input line.
			// This will thus not filter other candidates, despite printing the current one.
			rl.updateVirtualComp()

			rl.tabCompletionReverse = false
			rl.renderHelpers()
			rl.viUndoSkipAppend = true

			return
		}

	default:
		if rl.modeTabFind {
			return
		}
		// alt+numeric append / delete
		if len(r) == 2 && '1' <= r[1] && r[1] <= '9' {
			if rl.modeViMode == vimDelete {
				rl.vimDelete(r)
				return
			}

			line, err := rl.mainHistory.GetLine(rl.mainHistory.Len() - 1)
			if err != nil {
				return
			}
			if !rl.mainHist {
				line, err = rl.altHistory.GetLine(rl.altHistory.Len() - 1)
				if err != nil {
					return
				}
			}

			tokens, _, _ := tokeniseSplitSpaces([]rune(line), 0)
			pos := int(r[1]) - 48 // convert ASCII to integer
			if pos > len(tokens) {
				return
			}
			rl.insert([]rune(tokens[pos-1]))
		} else {
			rl.viUndoSkipAppend = true
		}
	}
}

// editorInput is an unexported function used to determine what mode of text
// entry readline is currently configured for and then update the line entries
// accordingly.
func (rl *Instance) editorInput(r []rune) {
	switch rl.modeViMode {
	case vimKeys:
		rl.vi(r[0])
		rl.refreshVimStatus()

	case vimDelete:
		rl.vimDelete(r)
		rl.refreshVimStatus()

	case vimReplaceOnce:
		rl.modeViMode = vimKeys
		rl.delete()
		rl.insert([]rune{r[0]})
		rl.refreshVimStatus()

	case vimReplaceMany:
		for _, char := range r {
			rl.delete()
			rl.insert([]rune{char})
		}
		rl.refreshVimStatus()

	default:
		// We reset the history nav counter each time we come here:
		// We don't need it when inserting text.
		rl.histNavIdx = 0
		rl.insert(r)
	}

	if len(rl.multisplit) == 0 {
		rl.syntaxCompletion()
	}
}

func (rl *Instance) carridgeReturn() {
	rl.clearHelpers()
	print("\r\n")
	if rl.HistoryAutoWrite {
		var err error

		// Main history
		if rl.mainHistory != nil {
			rl.histPos, err = rl.mainHistory.Write(string(rl.line))
			if err != nil {
				print(err.Error() + "\r\n")
			}
		}
		// Alternative history
		if rl.altHistory != nil {
			rl.histPos, err = rl.altHistory.Write(string(rl.line))
			if err != nil {
				print(err.Error() + "\r\n")
			}
		}
	}
}

func isMultiline(r []rune) bool {
	for i := range r {
		if (r[i] == '\r' || r[i] == '\n') && i != len(r)-1 {
			return true
		}
	}
	return false
}

func (rl *Instance) allowMultiline(data []byte) bool {
	rl.clearHelpers()
	printf("\r\nWARNING: %d bytes of multiline data was dumped into the shell!", len(data))
	for {
		print("\r\nDo you wish to proceed (yes|no|preview)? [y/n/p] ")

		b := make([]byte, 1024)

		i, err := os.Stdin.Read(b)
		if err != nil {
			return false
		}

		s := string(b[:i])
		print(s)

		switch s {
		case "y", "Y":
			print("\r\n" + rl.mainPrompt)
			return true

		case "n", "N":
			print("\r\n" + rl.mainPrompt)
			return false

		case "p", "P":
			preview := string(bytes.Replace(data, []byte{'\r'}, []byte{'\r', '\n'}, -1))
			if rl.SyntaxHighlighter != nil {
				preview = rl.SyntaxHighlighter([]rune(preview))
			}
			print("\r\n" + preview)

		default:
			print("\r\nInvalid response. Please answer `y` (yes), `n` (no) or `p` (preview)")
		}
	}
}
