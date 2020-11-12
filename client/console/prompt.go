package console

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// prompt - The prompt object is in charge of computing values, refreshing and printing them.
type prompt struct{}

// Render - The prompt determines in which context we currently are (core or sliver), and asks
// the corresponding 'sub-prompt' to compute itself and return its string.
func (p *prompt) Render() (prompt string) {
	return
}

// applyCallbacks - For each '{value}' in the prompt string, compute value and replace it.
func (p *prompt) applyCallbacks(in string) (p string, length int) {
	return
}

// getPromptPad - The prompt has the length of each of its subcomponents, and the terminal
// width. Based on this, it computes and returns a string pad for the prompt.
func (p *prompt) getPromptPad(total, base, module, context int) (pad string) {
	return
}

// promptCore - A prompt used when user is in the main/core menu.
type promptCore struct{}

// render - The core prompt computes all necessary values, forges a prompt string
// and returns it for being prompted by the shell.
func (p *promptCore) render() (prompt string) {

	// We need the terminal width: the prompt sometimes
	// makes use of both sides for different items.

	// For each 'part' (left-side, right-side) of the prompt, we compute its length.

	// Compute all prompt parts independently

	// Verify that the length of all combined prompt elements is not wider than
	// determined terminal width. If yes, truncate the prompt string accordingly.

	// Get the empty part of the prompt and pad accordingly.

	// Finally, forge the complete prompt string

	return
}

// computeBase - Computes the base prompt (left-side) with potential custom prompt given.
// Returns the width of the computed string, for correct aggregation of all strings.
func (p promptCore) computeBase() (p string, width int) {
	return
}

// promptSliver - A prompt used when user is interacting with a sliver implant.
type promptSliver struct{}

// render - The sliver prompt computes and forges a prompt string, the same way as in main menu.
func (p *promptSliver) render() (prompt string) {

	// We need the terminal width: the prompt sometimes
	// makes use of both sides for different items.

	// For each 'part' (left-side, right-side) of the prompt, we compute its length.

	// Compute all prompt parts independently

	// Verify that the length of all combined prompt elements is not wider than
	// determined terminal width. If yes, truncate the prompt string accordingly.

	// Get the empty part of the prompt and pad accordingly.

	// Finally, forge the complete prompt string

	return
}
