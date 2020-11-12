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
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/commands"
	"github.com/bishopfox/sliver/client/connection"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

var (
	// Console - The client console object
	Console = newConsole()

	// Flags - Used by main function for various details
	displayVersion = flag.Bool("version", false, "print version number")
)

const (
	logFileName = "sliver-client.log"
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

// connect - The console connects to the server and authenticates. Note that all
// config information (access points and security details) have been loaded already.
func (c *console) connect() (err error) {

	// Connect to server (performs TLS authentication)
	conn, err = connection.ConnectTLS()

	// Register RPC Service Client.
	connection.RPC = rpcpb.NewSliverRPCClient(conn)
	// Return fatal if nil

	// Listen for incoming server/implant events. If an error occurs in this
	// loop, the console will exit after logging it.
	go c.startEventHandler()

	// Start message tunnel loop.
	go connection.TunnelLoop()

	return
}

// setup - The console sets up various elements such as the completion system, hints,
// syntax highlighting, prompt system, commands binding, and client environment loading.
func (c *console) setup() (err error) {

	// Prompt. This computes all callbacks and base prompt strings
	// for the first time, and then binds it to the readline console.
	c.initPrompt()

	// Completions, hints and syntax highlighting

	// History (client and user-wide)

	// Client-side environment

	// Commands binding. Per-context parsers are setup here.
	err = commands.BindCommands()

	return
}

// Start - The console calls connection and setup functions, and starts the input loop.
func (c *console) Start() (err error) {

	// Print banner and version information.
	// This will also check the last update time.
	printLogo()

	// Initialize console logging (in textfile)
	initLogging()

	// Connect to server and authenticate
	err = c.connect()
	if err != nil {
		log.Fatal(err)
	}

	// Setup console elements
	err = c.setup()
	if err != nil {
		log.Fatal(err)
	}

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

// Initialize logging
func initLogging() {
	appDir := assets.GetRootAppDir()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	logFile, err := os.OpenFile(path.Join(appDir, logFileName), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(fmt.Sprintf("[!] Error opening file: %s", err))
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	return logFile
}
