package server

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
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/help"
)

// Help - Print help for the current context (lists all commands)
type Help struct {
	Positional struct {
		Component string
	} `positional-args:"true"`
}

// Execute - Print help for the current context (lists all commands)
func (h *Help) Execute(args []string) (err error) {

	// If no component argument is asked for
	if h.Positional.Component == "" {
		help.PrintMenuHelp("")
		return
	}

	// If a precise meny is asked for
	if h.Positional.Component == cctx.Server {
		help.PrintMenuHelp(cctx.Server)
		return
	}
	if h.Positional.Component == cctx.Sliver {
		help.PrintMenuHelp(cctx.Sliver)
		return
	}

	parser := cctx.Commands.GetCommands()

	// If a command is asked for
	for _, cmd := range parser.Commands() {
		if cmd.Name == h.Positional.Component {
			help.PrintCommandHelp(cmd)
		}
	}

	return
}
