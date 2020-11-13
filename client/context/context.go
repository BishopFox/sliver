package context

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
	"sync"
)

var (
	// Context - The console context object
	Context *ConsoleContext
)

// Menu Contexts
const (
	// Server - "Main menu" of wiregost, gives all commands and completion system
	// available for interacting with server.
	Server = "server"
	// Sliver - Available only when interacting with a sliver implant
	Sliver = "sliver"
)

// ConsoleContext - Stores all variables needed for console context
type ConsoleContext struct {
	Menu                string // Current shell menu
	Jobs                int    // Number of jobs
	Ghosts              int    // Number of connected implants
	NeedsCommandRefresh bool   // A command might or has set this to true.
	mutex               *sync.Mutex
}

// Initialize - The console calls to initialize a new context object, to be shared by
// many components of the console system (completion, command dispatch, prompt, etc.)
func Initialize() {
	Context = &ConsoleContext{
		Menu:  Server,
		mutex: &sync.Mutex{},
	}
	return
}
