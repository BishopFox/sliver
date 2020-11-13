package commands

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
	"github.com/jessevdk/go-flags"

	"github.com/bishopfox/sliver/client/constants"
)

var (
	// Server - All commands available in the main (server) menu are processed
	// by this Server parser. This is the basis of context separation for various
	// completions, hints, prompt system, etc.
	Server = flags.NewNamedParser("server", flags.None)

	// Sliver - The parser used to process all commands directed at sliver implants.
	Sliver = flags.NewNamedParser("sliver", flags.None)
)

// BindCommands - Binds all commands to their appropriate parsers, which have been instantiated already.
func BindCommands() (err error) {
	err = bindServerCommands()
	if err != nil {
		return
	}
	err = bindSliverCommands()
	if err != nil {
		return
	}

	return
}

// All commands concerning the server and/or the console itself are bound in this function.
func bindServerCommands() (err error) {

	ex, err := Server.AddCommand(constants.ExitStr, "Exit from the client/server console",
		"Exit from the client/server console", &Exit{})
	ex.Namespace = "Core"
	return
}

// All commands for controlling sliver implants are bound in this function.
func bindSliverCommands() (err error) {
	return
}
