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

import (
	"fmt"

	ansi "github.com/acarl005/stripansi"

	"github.com/bishopfox/sliver/client/context"
)

var (
	// Prompt - The prompt singleton object used by the console.
	Prompt *prompt
)

// prompt - The prompt object is in charge of computing values, refreshing and printing them.
type prompt struct {
	Server *promptServer   // Main menu
	Sliver *promptSliver // Sliver session menu
}

// initPrompt - Sets up the console root prompt system and binds it to readline.
func (c *console) initPrompt() {

	// Prompt settings (Vim-mode and stuff)
	c.Shell.Multiline = true   // spaceship-like prompt (2-line)
	c.Shell.ShowVimMode = true // with Vim mode status

	Prompt = &prompt{
		Server: &promptServer{
			Callbacks: serverCallbacks,
			Colors:    serverColorCallbacks,
		},
		Sliver: &promptSliver{
			Callbacks: sliverCallbacks,
			Colors:    sliverColorCallbacks,
		},
	}

	c.Shell.SetPrompt(Prompt.Render())
}

// ComputePrompt - Recompute prompt. This function may trigger some complex behavior:
// It reads various things on the current console context, and refreshes the prompt
// sometimes in a special way (like overwriting it fully or partly).
func (p *prompt) Compute() {
	line := p.Render()
	Console.Shell.SetPrompt(line)

	// Live a line between output and next prompt
	fmt.Println()

	// Check for refresh
	// if context.Context.NeedsCommandRefresh {
	//         p.RefreshOnCommand()
	// }
}

// Render - The prompt determines in which context we currently are (core or sliver), and asks
// the corresponding 'sub-prompt' to compute itself and return its string.
func (p *prompt) Render() (prompt string) {
	switch context.Context.Menu {
	case context.Server:
		prompt = p.Server.render()
	case context.Sliver:
		prompt = p.Sliver.render()
	}
	return
}

// getPromptPad - The prompt has the length of each of its subcomponents, and the terminal
// width. Based on this, it computes and returns a string pad for the prompt.
func (p *prompt) getPromptPad(total, base, module, context int) (pad string) {
	return
}

func getPromptPad(total, base, context int) (pad string) {
	var padLength = total - base - context
	for i := 0; i < padLength; i++ {
		pad += " "
	}
	return
}

// getRealLength - Some strings will have ANSI escape codes, which might be wrongly
// interpreted as legitimate parts of the strings. This will bother if some prompt
// components depend on other's length, so we always pass the string in this for
// getting its real-printed length.
func getRealLength(s string) (l int) {
	return len(ansi.Strip(s))
}
