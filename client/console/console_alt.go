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
	"github.com/maxlandon/readline"
)

var (
	// Console - The client console object
	Console = newConsole()
)

// newConsole - Instantiates a new console with some default behavior.
// We modify/add elements of behavior later in setup.
func newConsole() *console {

	console := &console{
		Shell: readline.NewInstance(),
	}

	return console
}

// console - Central object of the client UI. Only one instance of this object
// lives in the client executable (instantiated with newConsole() above).
type console struct {
	Shell *readline.Instance // Provides input loop and completion system.
}

// Connect - The console connects to the server and authenticates. Note that all
// config information (access points and security details) have been loaded already.
func (c *console) Connect() (err error) {

	// Connect to server (performs password-based and TLS authentication)

	// Listen for incoming server/implant events.

	return
}

// Setup - The console sets up various elements such as the completion system, hints,
// syntax highlighting, prompt system, commands binding, and client environment loading.
func (c *console) Setup() (err error) {

	// Prompt

	// Completions, hints and syntax highlighting

	// History (client and user-wide)

	// Client-side environment

	// Commands binding

	return
}

// Start - The console calls connection and setup functions, and starts the input loop.
func (c *console) Start() {

	// Connect to server and authenticate

	// Setup console elements

	// Start input loop

	for {
		// Recompute prompt each time, before anything.

		// Read input line

		// Split and sanitize input

		// Process various tokens on input (environment variables, paths, etc.)

		// Execute the command input: all input is passed to the current
		// context parser, which will deal with it on its own.
	}
}

// sanitizeInput - Trims spaces and other unwished elements from the input line.
func sanitizeInput(line string) (sanitized []string, empty bool) {
	return
}
