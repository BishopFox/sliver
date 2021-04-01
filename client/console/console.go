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
	cctx "github.com/bishopfox/sliver/client/context"
	clientLog "github.com/bishopfox/sliver/client/log"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

var (
	// Console - The client console object
	Console = newClient()

	// Flags - Used by main function for various details
	displayVersion = flag.Bool("version", false, "print version number")
)

const (
	logFileName = "sliver-client.log"
)

// Client - Central object of the client UI. Only one instance of this object
// lives in the client executable (instantiated with newConsole() above).
type Client struct {
	Shell  *readline.Instance // Provides input loop and completion system.
	Conn   *grpc.ClientConn   // The current raw gRPC client connection to the server.
	Prompt *prompt            // The prompt for all contexts
}

// newClient - Instantiates a new console with some default behavior.
// We modify/add elements of behavior later in setup.
func newClient() *Client {
	console := &Client{
		Shell: readline.NewInstance(),
	}
	return console
}

// Init - The console has a working RPC connection: we setup all
// things pertaining to the console itself, before calling the Run() function.
func (c *Client) Init(conn *grpc.ClientConn) (err error) {

	// Register RPC service clients. This function is called here because both the server
	// and the client binary use the same function, but the server binary
	c.connect(conn)

	// Setup console elements
	err = c.setup()
	if err != nil {
		return fmt.Errorf("Console setup failed: %s", err)
	}

	// Start monitoring all logs from the server and the client.
	err = clientLog.Init(c.Shell, c.Prompt.Render, transport.RPC)
	if err != nil {
		return fmt.Errorf("Failed to start log monitor (%s)", err.Error())
	}

	// Start monitoring incoming events
	go c.handleServerLogs(transport.RPC)

	// Print banner and version information. (checks last updates)
	printLogo()

	return
}

// connect - The console connects to the server and authenticates. Note that all server
// config information (access points and security details) have been loaded already.
func (c *Client) connect(conn *grpc.ClientConn) (*grpc.ClientConn, error) {

	// Bind this connection as the current console client gRPC connection
	c.Conn = conn

	// Register RPC Service Client.
	transport.RPC = rpcpb.NewSliverRPCClient(conn)
	if transport.RPC == nil {
		return nil, errors.New("could not register gRPC Client, instance is nil")
	}

	// Start message tunnel loop.
	go transport.TunnelLoop()

	return conn, nil
}

// setup - The console sets up various elements such as the completion system, hints,
// syntax highlighting, prompt system, commands binding, and client environment loading.
func (c *Client) setup() (err error) {

	initLogging() // textfile log

	// Get the user's console configuration from the server
	err = cctx.LoadConsoleConfig(transport.RPC)
	if err != nil {
		fmt.Printf(util.Error + "Failed to load console configuration from server.\n")
		fmt.Printf(util.Info + "Defaulting to builtin values.\n")
	}

	// This context object will hold some state about the
	// console (which implant we're interacting, jobs, etc.)
	cctx.InitializeConsole(transport.RPC)

	// This computes all callbacks and base prompt strings
	// for the first time, and then binds it to the console.
	c.initPrompt()
	c.Shell.Multiline = true
	c.Shell.ShowVimMode = true
	c.Shell.VimModeColorize = true

	// Completions and syntax highlighting
	c.Shell.TabCompleter = completers.TabCompleter
	c.Shell.SyntaxHighlighter = completers.SyntaxHighlighter

	// History (client and user-wide)
	c.Shell.SetHistoryCtrlE("client history", ClientHist)
	c.Shell.SetHistoryCtrlR("user-wise history", UserHist)

	// Request the user history to server and cache it
	getUserHistory()

	// Client-side environment
	err = util.LoadClientEnv()
	if err != nil {
		return fmt.Errorf("could not load client OS env (%s)", err)
	}

	return
}

// Run - Start the actual readline input loop, required
// per-loop console setup details, and command execution.
func (c *Client) Run() {

	// When we will exit this loop, disconnect gracefully from the server.
	// The latter will take care of notifying other clients/players if needed.
	defer c.Conn.Close()

	for {
		// Some commands can act on the shell properties via the console
		// context package, so we check values and set everything up.
		c.ResetShell()

		// Reset the completion data cache for all registered sessions.
		completers.Cache.Reset()

		// Recompute prompt each time, before anything.
		c.ComputePrompt()

		// Set the different sources of history, depending on context, session.
		c.SetHistory()

		// Reset the log synchroniser, before rebinding the commands, so that
		// if anyone is to use the parser.Active command, it will work until
		// just before rebinding the parser and its commands.
		clientLog.ResetLogSynchroniser()

		// Bind the command parser (and its commands), for the appropriate context.
		// This is before calling the console readline, because the latter needs
		// to be fed a parser for completions, hints, and syntax.
		cmds, err := commands.BindCommands()
		if err != nil {
			fmt.Print(util.CommandError + readline.Red("could not reset commands: "+err.Error()+"\n"))
		}

		// Register the commands for additional completions. Some of these may
		// also add restrained choices to some commands, so this function may
		// actually end up refining even more the parsing granularity of our shell.
		completers.LoadAdditionalCompletions(cmds)

		// Read input line (blocking)
		line, _ := c.Readline()

		// Split and sanitize the user-entered command line input.
		sanitized, empty := c.SanitizeInput(line)
		if empty {
			continue
		}

		// Process various tokens on input (environment variables, paths, etc.)
		envParsed, _ := util.ParseEnvironmentVariables(sanitized)

		// Other types of tokens, needed by commands who expect a certain type
		// of arguments, such as paths with spaces.
		tokenParsed := c.ParseTokens(envParsed)

		// Execute the command input: all input is passed to the current
		// context parser, which will deal with it on its own. We never return
		// errors from this call, as any of them happening follows a certain
		// number of fallback paths (special commands, error printing, etc.).
		c.ExecuteCommand(tokenParsed)
	}
}

// SetHistory - Depending on the context, the alternative history
// source becomes the current Session one.
func (c *Client) SetHistory() {

	// If we are interacting with an implant, set the correct history source
	if cctx.Context.Menu == cctx.Sliver && cctx.Context.Sliver != nil {
		c.Shell.SetHistoryCtrlE("session history", SessionHist)
		getSessionHistory()
	} else {
		c.Shell.SetHistoryCtrlE("client history", ClientHist)
	}
}

// ResetShell - Live refresh of console properties (input, hints, etc),
// following some commands (confi, usually) that may have changed some settings.
func (c *Client) ResetShell() {

	c.Shell.Multiline = true   // spaceship-like prompt (2-line)
	c.Shell.ShowVimMode = true // with Vim mode status

	// Input
	if !cctx.Config.Vim {
		c.Shell.InputMode = readline.Emacs
	} else {
		c.Shell.InputMode = readline.Vim
	}

	// Hints are configurable and can deactivated
	if !cctx.Config.Hints {
		c.Shell.HintText = nil
	} else {
		c.Shell.HintText = completers.HintCompleter
	}
}

// Readline - Add an empty line between input line and command output.
func (c *Client) Readline() (line string, err error) {
	line, err = c.Shell.Readline()
	fmt.Println()
	return
}

// SanitizeInput - Trims spaces and other unwished elements from the input line.
func (c *Client) SanitizeInput(line string) (sanitized []string, empty bool) {

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

// ParseTokens - Parse and process any special tokens that are not treated by environment-like parsers.
func (c *Client) ParseTokens(sanitized []string) (parsed []string) {

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
