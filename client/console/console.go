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
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/maxlandon/readline"
	"google.golang.org/grpc"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/commands"
	"github.com/bishopfox/sliver/client/completers"
	"github.com/bishopfox/sliver/client/connection"
	"github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/util"
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
	admin bool               // Are we an admin console running a server.
}

// connect - The console connects to the server and authenticates. Note that all
// config information (access points and security details) have been loaded already.
func (c *console) connect(admin bool) (conn *grpc.ClientConn, err error) {

	// Connect to server (performs TLS authentication)
	if admin {
		conn, err = connection.ConnectLocal()
	} else {
		conn, err = connection.ConnectTLS()
	}

	// Register RPC Service Client.
	connection.RPC = rpcpb.NewSliverRPCClient(conn)
	if connection.RPC == nil {
		return nil, errors.New("could not register gRPC Client, instance is nil")
	}

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

	initLogging() // textfile log

	// This context object will hold some state about the
	// console (which implant we're interacting, jobs, etc.)
	context.Initialize()

	// This computes all callbacks and base prompt strings
	// for the first time, and then binds it to the console.
	c.initPrompt()

	// Completions, hints and syntax highlighting
	c.Shell.TabCompleter = completers.TabCompleter
	c.Shell.HintText = completers.HintCompleter
	c.Shell.SyntaxHighlighter = completers.SyntaxHighlighter

	// History (client and user-wide)
	// c.Shell.History = ClientHist
	c.Shell.AltHistory = UserHist

	// Client-side environment
	err = util.LoadClientEnv()
	if err != nil {
		return fmt.Errorf("could not load client OS env (%s)", err)
	}

	return
}

// Start - The console calls connection and setup functions, and starts the input loop.
func (c *console) Start(admin bool) (err error) {

	c.admin = admin // Set server admin mode.

	// Connect to server and authenticate
	conn, err := c.connect(admin)
	if err != nil {
		log.Fatalf(Error+"Connection to server failed: %s", err)
	}
	defer conn.Close()

	// Setup console elements
	err = c.setup()
	if err != nil {
		log.Fatalf(Error+"Console setup failed: %s", err)
	}

	// Commands binding. Per-context parsers are setup here.
	err = commands.BindCommands(admin)

	// Print banner and version information.
	// This will also check the last update time.
	printLogo()

	// Start input loop
	for {
		// Recompute prompt each time, before anything.
		Prompt.Compute()

		// Read input line
		line, _ := c.Readline()

		// Split and sanitize input
		sanitized, empty := sanitizeInput(line)
		if empty {
			continue
		}

		// Process various tokens on input (environment variables, paths, etc.)
		envParsed, _ := util.ParseEnvironmentVariables(sanitized)

		// Other types of tokens, needed by commands who expect a certain type
		// of arguments, such as paths with spaces.
		tokenParsed := c.parseTokens(envParsed)

		// Execute the command input: all input is passed to the current
		// context parser, which will deal with it on its own. We never return
		// errors from this call, as any of them happening follows a certain
		// number of fallback paths (special commands, error printing, etc.).
		// We should not have to exit the console because of an error here, anyway.
		c.ExecuteCommand(tokenParsed)
	}
}

// Readline - Add an empty line between input line and command output.
func (c *console) Readline() (line string, err error) {
	line, err = c.Shell.Readline()
	fmt.Println()
	return
}

// sanitizeInput - Trims spaces and other unwished elements from the input line.
func sanitizeInput(line string) (sanitized []string, empty bool) {

	// Assume the input is not empty
	empty = false

	// Trim border spaces
	trimmed := strings.TrimSpace(line)
	if len(line) < 1 {
		empty = true
		return
	}
	unfiltered := strings.Split(trimmed, " ")

	// Catch any eventual empty items
	for _, arg := range unfiltered {
		if arg != "" {
			sanitized = append(sanitized, arg)
		}
	}

	return
}

// parseTokens - Parse and process any special tokens that are not treated by environment-like parsers.
func (c *console) parseTokens(sanitized []string) (parsed []string) {

	// PATH SPACE TOKENS
	// Catch \ tokens, which have been introduced in paths where some directories have spaces in name.
	// For each of these splits, we concatenate them with the next string.
	// This will also inspect commands/options/arguments, but there is no reason why a backlash should be present in them.
	var pathAdjusted []string
	var roll bool
	var arg string
	for i := range sanitized {
		if strings.HasSuffix(sanitized[i], "\\") {
			// If we find a suffix, replace with a space. Go on with next input
			arg += strings.TrimSuffix(sanitized[i], "\\") + " "
			roll = true
		} else if roll {
			// No suffix but part of previous input. Add it and go on.
			arg += sanitized[i]
			pathAdjusted = append(pathAdjusted, arg)
			arg = ""
			roll = false
		} else {
			// Default, we add our path and go on.
			pathAdjusted = append(pathAdjusted, sanitized[i])
		}
	}
	parsed = pathAdjusted

	// Add new function here, act on parsed []string from now on, not sanitized
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
	return
}
