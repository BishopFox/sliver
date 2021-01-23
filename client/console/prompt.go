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
	"net"
	"os"
	"strconv"
	"strings"

	ansi "github.com/acarl005/stripansi"
	"github.com/evilsocket/islazy/tui"
	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/context"
)

var (
	// Prompt - The prompt singleton object used by the console.
	Prompt *prompt
)

// prompt - The prompt object is in charge of computing values, refreshing and printing them.
type prompt struct {
	Server *promptCore   // Main menu
	Sliver *promptSliver // Sliver session menu
}

// initPrompt - Sets up the console root prompt system and binds it to readline.
func (c *console) initPrompt() {

	// Prompt settings (Vim-mode and stuff)
	c.Shell.Multiline = true   // spaceship-like prompt (2-line)
	c.Shell.ShowVimMode = true // with Vim mode status

	Prompt = &prompt{
		Server: &promptCore{
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

// applyCallbacks - For each '{value}' in the prompt string, compute value and replace it.
func (p *prompt) applyCallbacks(in string) (out string, length int) {
	return
}

// getPromptPad - The prompt has the length of each of its subcomponents, and the terminal
// width. Based on this, it computes and returns a string pad for the prompt.
func (p *prompt) getPromptPad(total, base, module, context int) (pad string) {
	return
}

// promptCore - A prompt used when user is in the main/core menu.
type promptCore struct {
	Base    string // The left-mode side of the prompt (usually server address, cwd, etc.)
	Context string // The right-most side of prompt, for infos like users, sessions, hosts, etc.

	Callbacks map[string]func() string
	Colors    map[string]string // We use different themes on different menus
}

// render - The core prompt computes all necessary values, forges a prompt string
// and returns it for being printed by the shell.
func (p *promptCore) render() (prompt string) {

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
func (p *promptCore) computeBase() (ps string, width int) {
	ps += tui.RESET

	// If we don't have any LHost value loaded, it means we did not load a config
	// and that therefore we are the server.
	if assets.ServerLHost == "{{.LHost}}" {
		ps += "{bddg}{y} server {fw}@{local_ip} {reset}"
	} else {
		ps += "{bddg}@{server_ip}{reset}"
	}

	// Current working directory
	ps += " {dim}in {bold}{b}{cwd}"
	ps += tui.RESET

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
func (p *promptCore) computeContext(sWidth int) (ps string, width int) {
	ps += tui.RESET

	// Compute all values needed
	var items string

	// Half of my 13" laptop is around 115, and I have useless gaps of 5.
	// This means we have a small console space.
	if (105 < sWidth) && (sWidth < 120) {
		items = "{y}{jobs}{fw}, {b}{sessions}{fw}{reset}"
	}
	// Two console cannot be side by side on a 13" laptop, but
	// we can on a 15" screen.
	if (120 < sWidth) && (sWidth < 140) {
		items = "{y}{jobs}{fw}, {b}{sessions}{fw}{reset}"
	}
	// Half of my 19" monitor is around 165, again with gaps of 5.
	// Here we can have to consoles side by side.
	if (140 < sWidth) && (sWidth < 175) {
		items = "{y}{jobs}{fw} jobs, {b}{sessions}{fw} sessions"
	}

	// Here we have one big console on a screen, no restrictions.
	if 175 < sWidth {
		items = "{y}{jobs}{fw} jobs, {b}{sessions}{fw} sessions"
	}

	// Finally add the items string inside its container.
	ps += fmt.Sprintf("{dim}[{reset}%s{dim}]", items)

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

func getPromptPad(total, base, context int) (pad string) {
	var padLength = total - base - context
	for i := 0; i < padLength; i++ {
		pad += " "
	}
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
			return assets.ServerLHost
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

	ps += "{bddg} {fr}{session_name} {reset}"             // Session name
	ps += "{bold} {user}{dim}@{reset}{bold}{host}{reset}" // User@host combination
	ps += " {dim}in{reset} {bold}{b}{cwd}"                // Current working directory
	ps += tui.RESET

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
func (p *promptSliver) computeContext(sWidth int) (ps string, width int) {
	ps += tui.RESET

	// Compute all values needed
	var items string

	// Half of my 13" laptop is around 115, and I have useless gaps of 5.
	// This means we have a small console space.
	if (105 < sWidth) && (sWidth < 120) {
		items = "{y}{bold}{platform}{reset}, {bold}{g}{address}{fw}{reset}"
	}
	// Two console cannot be side by side on a 13" laptop, but
	// we can on a 15" screen.
	if (120 < sWidth) && (sWidth < 140) {
		items = "{y}{bold}{platform}{reset}, {bold}{g}{address}{fw}{reset}"
	}
	// Half of my 19" monitor is around 165, again with gaps of 5.
	// Here we can have to consoles side by side.
	if (140 < sWidth) && (sWidth < 175) {
		items = "{y}{bold}{platform}{reset}, {bold}{g}{address}{fw}{reset}"
	}

	// Here we have one big console on a screen, no restrictions.
	if 175 < sWidth {
		items = "{y}{bold}{platform}{reset}, {bold}{g}{address}{fw}{reset}"
	}

	// Finally add the items string inside its container.
	ps += fmt.Sprintf("{dim}[{reset}%s{dim}]", items)

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
		"{cwd}": func() string {
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
		"{status}": func() string {
			if context.Context.Sliver.IsDead {
				return "DEAD"
			}
			return "up"
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
		"{lb}":    tui.BACKLIGHTBLUE,
		"{reset}": tui.RESET,
		// Custom colors
		"{ly}": "\033[38;5;187m",
		// "{lb}":   "\033[38;5;117m", // like VSCode var keyword
		"{bddg}": "\033[48;5;237m",
	}
)

// getRealLength - Some strings will have ANSI escape codes, which might be wrongly
// interpreted as legitimate parts of the strings. This will bother if some prompt
// components depend on other's length, so we always pass the string in this for
// getting its real-printed length.
func getRealLength(s string) (l int) {
	return len(ansi.Strip(s))
}
