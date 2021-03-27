package completers

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
	"strings"

	"github.com/maxlandon/readline"

	cctx "github.com/bishopfox/sliver/client/context"
)

var (
	urlShemes = []string{
		"tcp",
		"udp",
		"http",
		"https",
		"dns",
		"ftp",
		"ssh",
	}
)

// A user wants to input a URL. We complete shemes and hosts. We pass an indication
// about the current context, so as to refine the list of hosts we give, and we also
// pass in a list of valid shemes for this precise completion: some commands will be
// compatible with only a handful of URL schemes, like meterpreter stagers.
func completeURL(last string, sliverContext bool, validSchemes []string) (prefix string, completions []*readline.CompletionGroup) {

	// Normally, when we enter the function the last input should be nil
	// because we don't "append" a URL to anything  without a space (usually).
	// If its the case, just return schemes with ://
	// Also, we ask the shell not to add a space when inserting the completion
	// because we are not done with the URL yet.
	if last == "" || last == " " {
		completion := &readline.CompletionGroup{
			Name:        "stager protocol",
			MaxLength:   5,
			DisplayType: readline.TabDisplayGrid,
			NoSpace:     true,
		}
		for _, sch := range validSchemes {
			completion.Suggestions = append(completion.Suggestions, sch+"://")
		}
		return last, []*readline.CompletionGroup{completion}
	}

	// If not empty, and the last is suffixed by ://
	// return a list of hosts like for other completions.
	if strings.Contains(last, "://") {

		// Trim the input from the scheme://, otherwise no hosts will match
		parts := strings.Split(last, "://")
		var host string
		if len(parts) > 1 {
			host = parts[1] // Just keep the host, if empty its not a problem.
		}

		if !sliverContext {
			// Client IPv4/IPv6 interfaces
			completions = append(completions, clientInterfaceAddrs(host, false))

			// All implant host addresses reachable through a route.
			completions = append(completions, routedSessionIfaceAddrs(host, 0, false)...)

			return host, completions
		}

		if sliverContext {
			sliverID := cctx.Context.Sliver.Session.ID

			// Else we are starting a handler, or something like. Separate group for active session
			completions = append(completions, sessionIfaceAddrs(host, cctx.Context.Sliver.Session, false))
			// Remaining routed implants
			completions = append(completions, routedSessionIfaceAddrs(host, sliverID, false)...)

			return host, completions
		}
	}

	return
}
