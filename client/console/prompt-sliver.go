package console

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/evilsocket/islazy/tui"
	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/context"
)

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

// promptSliver - A prompt used when user is interacting with a sliver implant.
type promptSliver struct {
	Base    string // The left-mode side of the prompt (usually server address, cwd, etc.)
	Context string // The right-most side of prompt, for infos like users, sessions, hosts, etc.

	Callbacks map[string]func() string
	Colors    map[string]string // We use different themes on different menus
}

// render - The sliver prompt computes and forges a prompt string, the same way as in main menu.
func (p *promptSliver) render() (prompt string) {

	// We need the terminal width: the prompt sometimes
	// makes use of both sides for different items.
	sWidth := readline.GetTermWidth()

	// Other lengths that we might need to pass around
	var bWidth int // base prompt width, after computing.
	var cWidth int // Context prompt width

	// Compute all prompt parts independently
	p.Base, bWidth = p.computeBase()
	p.Context, cWidth = p.computeContext(sWidth)

	// Verify that the length of all combined prompt elements is not wider than
	// determined terminal width. If yes, truncate the prompt string accordingly.
	if bWidth+cWidth > sWidth {
		// m.Module = truncate()
	}

	// Get the empty part of the prompt and pad accordingly.
	pad := getPromptPad(sWidth, bWidth, cWidth)

	// Finally, forge the complete prompt string
	prompt = p.Base + pad + p.Context

	return
}

// computeBase - Computes the base prompt (left-side) with potential custom prompt given.
// Returns the width of the computed string, for correct aggregation of all strings.
func (p *promptSliver) computeBase() (ps string, width int) {
	ps += tui.RESET

	// The prompt string is stored in the context package, to be accessed by
	// console configuration commands.
	ps += context.Config.SliverPrompt.Left

	ps += tui.RESET

	// Compute sliver-context callbacks
	for ok, cb := range p.Callbacks {
		ps = strings.Replace(ps, ok, cb(), 1)
	}
	for tok, color := range p.Colors {
		ps = strings.Replace(ps, tok, color, -1)
	}

	// Compute server callbacks, which may be used as well
	for ok, cb := range serverCallbacks {
		ps = strings.Replace(ps, ok, cb(), 1)
	}
	for tok, color := range serverColorCallbacks {
		ps = strings.Replace(ps, tok, color, -1)
	}

	width = getRealLength(ps)

	return
}

// computeContext - Analyses the current state of various server indicators and displays them.
// Because it is the right-most part of the prompt, and that the screen might be small,
// we categorize default (assumed) screen sizes and we adapt the output consequently.
func (p *promptSliver) computeContext(sWidth int) (ps string, width int) {
	ps += tui.RESET

	// The prompt string is stored in the context package, to be accessed by
	// console configuration commands.
	ps += context.Config.SliverPrompt.Right

	ps += tui.RESET // Always at end of prompt line

	// Callbacks
	for ok, cb := range p.Callbacks {
		ps = strings.Replace(ps, ok, cb(), 1)
	}
	for tok, color := range p.Colors {
		ps = strings.Replace(ps, tok, color, -1)
	}

	width = getRealLength(ps)

	return
}

var (
	sliverCallbacks = map[string]func() string{
		"{session_name}": func() string {
			return context.Context.Sliver.Name
		},
		"{wd}": func() string {
			return context.Context.Sliver.WorkingDir
		},
		"{user}": func() string {
			return context.Context.Sliver.Username
		},
		"{host}": func() string {
			return context.Context.Sliver.Hostname
		},
		"{address}": func() string {
			return context.Context.Sliver.RemoteAddress
		},
		"{platform}": func() string {
			os := context.Context.Sliver.OS
			arch := context.Context.Sliver.Arch
			return fmt.Sprintf("%s/%s", os, arch)
		},
		"{os}": func() string {
			return context.Context.Sliver.OS
		},
		"{arch}": func() string {
			return context.Context.Sliver.Arch
		},
		"{status}": func() string {
			if context.Context.Sliver.IsDead {
				return "DEAD"
			}
			return "up"
		},
		"{version}": func() string {
			return context.Context.Sliver.Version
		},
		"{uid}": func() string {
			return context.Context.Sliver.UID
		},
		"{gid}": func() string {
			return context.Context.Sliver.GID
		},
		"{pid}": func() string {
			return strconv.Itoa(int(context.Context.Sliver.PID))
		},
	}

	sliverColorCallbacks = map[string]string{
		// Base tui colors
		"{blink}": "\033[5m", // blinking
		"{bold}":  tui.BOLD,
		"{dim}":   tui.DIM, // for Base Dim
		"{fr}":    tui.RED, // for Base Fore Red
		"{g}":     tui.GREEN,
		"{b}":     tui.BLUE,
		"{y}":     tui.YELLOW,
		"{fw}":    tui.FOREWHITE, // for Base Fore White.
		"{dg}":    tui.BACKDARKGRAY,
		"{br}":    tui.BACKRED,
		"{bg}":    tui.BACKGREEN,
		"{by}":    tui.BACKYELLOW,
		"{blb}":   tui.BACKLIGHTBLUE,
		"{reset}": tui.RESET,
		// Custom colors
		"{ly}":   "\033[38;5;187m",
		"{lb}":   "\033[38;5;117m", // like VSCode var keyword
		"{bddg}": "\033[48;5;237m",
	}
)
