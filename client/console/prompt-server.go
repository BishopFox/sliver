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
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/evilsocket/islazy/tui"
	"github.com/bishopfox/sliver/client/readline"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/context"
)

// promptServer - A prompt used when user is in the main/core menu.
type promptServer struct {
	Base    string // The left-mode side of the prompt (usually server address, cwd, etc.)
	Context string // The right-most side of prompt, for infos like users, sessions, hosts, etc.

	Callbacks map[string]func() string
	Colors    map[string]string // We use different themes on different menus
}

// render - The core prompt computes all necessary values, forges a prompt string
// and returns it for being printed by the shell.
func (p *promptServer) render() (prompt string) {

	// We need the terminal width: the prompt sometimes
	// makes use of both sides for different items.
	sWidth := readline.GetTermWidth()

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
func (p *promptServer) computeBase() (ps string, width int) {
	ps += tui.RESET

	// The prompt string is stored in the context package, to be accessed by
	// console configuration commands.
	ps += context.Config.ServerPrompt.Left

	// Ensure colors do not screw up the next input.
	ps += tui.RESET

	// Compute callback values
	for ok, cb := range p.Callbacks {
		ps = strings.Replace(ps, ok, cb(), 1)
	}
	for tok, color := range p.Colors {
		ps = strings.Replace(ps, tok, color, -1)
	}

	width = getRealLength(ps)

	return
}

// computeContext - Analyses the current state of various server indicators and displays them.
// Because it is the right-most part of the prompt, and that the screen might be small,
// we categorize default (assumed) screen sizes and we adapt the output consequently.
func (p *promptServer) computeContext(sWidth int) (ps string, width int) {
	ps += tui.RESET

	// The prompt string is stored in the context package, to be accessed by
	// console configuration commands.
	ps += context.Config.ServerPrompt.Right

	// Ensure colors do not screw up the next input.
	ps += tui.RESET

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
	// serverCallbacks - All items needed by the prompt when in Server menu.
	serverCallbacks = map[string]func() string{
		// Local working directory
		"{cwd}": func() string {
			cwd, err := os.Getwd()
			if err != nil {
				return "ERROR: Could not get working directory !"
			}
			return cwd
		},
		// Server IP
		"{server_ip}": func() string {
			return assets.Config.LHost
		},
		// Local IP address
		"{local_ip}": func() string {
			addrs, _ := net.InterfaceAddrs()
			var ip string
			for _, addr := range addrs {
				network, ok := addr.(*net.IPNet)
				if ok && !network.IP.IsLoopback() && network.IP.To4() != nil {
					ip = network.IP.String()
				}
			}
			return ip
		},
		// Jobs and/or listeners
		"{jobs}": func() string {
			return strconv.Itoa(context.Context.Jobs)
		},
		// Sessions
		"{sessions}": func() string {
			return strconv.Itoa(context.Context.Slivers)
		},
		"{timestamp}": func() string {
			return time.Now().Format("15:04:05.000")
		},
	}

	// serverColorCallbacks - All colors and effects needed in the main menu
	serverColorCallbacks = map[string]string{
		// Base tui colors
		"{blink}": "\033[5m", // blinking
		"{bold}":  tui.BOLD,
		"{dim}":   tui.DIM,
		"{fr}":    tui.RED,
		"{g}":     tui.GREEN,
		"{b}":     tui.BLUE,
		"{y}":     tui.YELLOW,
		"{fw}":    tui.FOREWHITE,
		"{bdg}":   tui.BACKDARKGRAY,
		"{br}":    tui.BACKRED,
		"{bg}":    tui.BACKGREEN,
		"{by}":    tui.BACKYELLOW,
		"{blb}":   tui.BACKLIGHTBLUE,
		"{reset}": tui.RESET,
		// Custom colors
		"{ly}":   "\033[38;5;187m",
		"{lb}":   "\033[38;5;117m", // like VSCode var keyword
		"{db}":   "\033[38;5;24m",
		"{bddg}": "\033[48;5;237m",
	}
)
