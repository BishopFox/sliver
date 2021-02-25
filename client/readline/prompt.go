package readline

import (
	"fmt"
)

// RefreshPromptLog - A simple function to print a string message (a log, or more broadly,
// an asynchronous event) without bothering the user, and by "pushing" the prompt below the message.
func (rl *Instance) RefreshPromptLog(log string) (err error) {

	// We adjust cursor movement, depending on which mode we're currently in.
	if !rl.modeTabCompletion {
		rl.tcUsedY = 1
		// Account for the hint line
	} else if rl.modeTabCompletion && rl.modeAutoFind {
		rl.tcUsedY = 0
	} else {
		rl.tcUsedY = 1
	}

	// Prompt offset
	if rl.Multiline {
		rl.tcUsedY += 1
	} else {
		rl.tcUsedY += 0
	}

	// Clear the current prompt and everything below
	print(seqClearLine)
	if rl.stillOnRefresh {
		moveCursorUp(1)
	}
	rl.stillOnRefresh = true
	moveCursorUp(rl.hintY + rl.tcUsedY)
	moveCursorBackwards(GetTermWidth())
	print("\r\n" + seqClearScreenBelow)

	// Print the log
	fmt.Printf(log)

	// Add a new line between the message and the prompt, so not overloading the UI
	print("\n")

	// Print the prompt
	if rl.Multiline {
		rl.tcUsedY += 3
		fmt.Println(rl.prompt)

	} else {
		rl.tcUsedY += 2
		fmt.Print(rl.prompt)
	}

	// Refresh the line
	rl.renderHelpers()

	return
}

// RefreshPromptInPlace - Refreshes the prompt in the very same place he is.
func (rl *Instance) RefreshPromptInPlace(prompt string) (err error) {

	// We adjust cursor movement, depending on which mode we're currently in.
	// Prompt data intependent
	if !rl.modeTabCompletion {
		rl.tcUsedY = 1
		// Account for the hint line
	} else if rl.modeTabCompletion && rl.modeAutoFind {
		rl.tcUsedY = 0
	} else {
		rl.tcUsedY = 1
	}

	// Update the prompt if a special has been passed.
	if prompt != "" {
		rl.prompt = prompt
	}

	if rl.Multiline {
		rl.tcUsedY += 1
	}

	// Clear the input line and everything below
	print(seqClearLine)
	moveCursorUp(rl.hintY + rl.tcUsedY)
	moveCursorBackwards(GetTermWidth())
	print("\r\n" + seqClearScreenBelow)

	// Add a new line if needed
	if rl.Multiline {
		fmt.Println(rl.prompt)

	} else {
		fmt.Print(rl.prompt)
	}

	// Refresh the line
	rl.renderHelpers()

	return
}

// RefreshPromptCustom - Refresh the console prompt with custom values.
// @prompt      => If not nil (""), will use this prompt instead of the currently set prompt.
// @offset      => Used to set the number of lines to go upward, before reprinting. Set to 0 if not used.
// @clearLine   => If true, will clean the current input line on the next refresh.
func (rl *Instance) RefreshPromptCustom(prompt string, offset int, clearLine bool) (err error) {

	// We adjust cursor movement, depending on which mode we're currently in.
	if !rl.modeTabCompletion {
		rl.tcUsedY = 1
	} else if rl.modeTabCompletion && rl.modeAutoFind { // Account for the hint line
		rl.tcUsedY = 0
	} else {
		rl.tcUsedY = 1
	}

	// Add user-provided offset
	rl.tcUsedY += offset

	// Clear the input line and everything below
	print(seqClearLine)
	moveCursorUp(rl.hintY + rl.tcUsedY)
	moveCursorBackwards(GetTermWidth())
	print("\r\n" + seqClearScreenBelow)

	// Update the prompt if a special has been passed.
	if prompt != "" {
		rl.prompt = prompt
	}

	// Add a new line if needed
	if rl.Multiline && prompt == "" {
	} else if rl.Multiline {
		fmt.Println(rl.prompt)
	} else {
		fmt.Print(rl.prompt)
	}

	// Refresh the line
	rl.renderHelpers()

	// If input line was empty, check that we clear it from detritus
	// The three lines are borrowed from clearLine(), we don't need more.
	if clearLine {
		rl.clearLine()
	}

	return
}

// computePrompt - At any moment, returns an (1st or 2nd line) actualized prompt,
// considering all input mode parameters and prompt string values.
func (rl *Instance) computePrompt() (prompt []rune) {
	switch rl.InputMode {
	case Vim:
		rl.computePromptVim()
	case Emacs:
		rl.computePromptEmacs()
	}
	return
}

func (rl *Instance) computePromptVim() {
	var vimStatus []rune
	var colorPromptOffset int
	// Compute Vim status
	if rl.ShowVimMode {
		switch rl.modeViMode {
		case vimKeys:
			vimStatus = []rune(vimKeysStr)
		case vimInsert:
			vimStatus = []rune(vimInsertStr)
		case vimReplaceOnce:
			vimStatus = []rune(vimReplaceOnceStr)
		case vimReplaceMany:
			vimStatus = []rune(vimReplaceManyStr)
		case vimDelete:
			vimStatus = []rune(vimDeleteStr)
		}

		// Process colors, and get offset for correct cursor position
		bwPromptLen := len(vimStatus)
		vimStatus = rl.colorizeVimPrompt(vimStatus)

		colorPromptLen := len(vimStatus)
		colorPromptOffset = colorPromptLen - bwPromptLen
	}

	// Append any optional prompts for multiline mode
	if rl.Multiline {
		if rl.MultilinePrompt != "" {
			rl.mlnPrompt = append(vimStatus, []rune(rl.MultilinePrompt)...)
		} else {
			rl.mlnPrompt = vimStatus
			rl.mlnPrompt = append(rl.mlnPrompt, rl.mlnArrow...)
		}
	}
	// Equivalent for non-multiline
	if !rl.Multiline {
		if rl.prompt != "" {
			rl.mlnPrompt = append(vimStatus, []rune(" "+rl.prompt)...)
		} else {
			rl.mlnPrompt = append(rl.mlnPrompt, vimStatus...)
		}
		if rl.MultilinePrompt != "" {
			rl.mlnPrompt = append(rl.mlnPrompt, []rune(rl.MultilinePrompt)...)
		} else {
			rl.mlnPrompt = append(rl.mlnPrompt, rl.mlnArrow...)
		}
	}

	// Compute lengths and return
	rl.promptLen = len(rl.mlnPrompt) - colorPromptOffset
}

func (rl *Instance) computePromptEmacs() {
	if rl.Multiline {
		if rl.MultilinePrompt != "" {
			rl.mlnPrompt = []rune(rl.MultilinePrompt)
		} else {
			rl.mlnPrompt = rl.mlnArrow
		}

	}
	if !rl.Multiline {
		if rl.prompt != "" {
			rl.mlnPrompt = append([]rune(rl.prompt), rl.mlnArrow...)
		}
		if rl.MultilinePrompt != "" {
			rl.mlnPrompt = append(rl.mlnPrompt, []rune(rl.MultilinePrompt)...)
		} else {
			rl.mlnPrompt = append(rl.mlnPrompt, rl.mlnArrow...)
		}
	}

	// Compute lengths and return
	rl.promptLen = len(rl.mlnPrompt)
}

func (rl *Instance) colorizeVimPrompt(p []rune) (cp []rune) {
	if rl.VimModeColorize {
		return []rune(fmt.Sprintf("%s%s%s", BOLD, string(p), RESET))
	}

	return p
}
