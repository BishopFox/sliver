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
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/comm"
	"github.com/bishopfox/sliver/client/commands"
	"github.com/bishopfox/sliver/client/completers"
	cctx "github.com/bishopfox/sliver/client/context"
	clientLog "github.com/bishopfox/sliver/client/log"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"

	"github.com/bishopfox/sliver/protobuf/commpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	// Comm System dependencies
	"github.com/golang/protobuf/proto"
	grpcConn "github.com/mitchellh/go-grpc-net-conn"
	"google.golang.org/grpc"
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
	conn  *grpc.ClientConn   // The current raw gRPC client connection to the server.
}

// connect - The console connects to the server and authenticates. Note that all
// config information (access points and security details) have been loaded already.
func (c *console) Connect(conn *grpc.ClientConn, admin bool) (*grpc.ClientConn, error) {

	// Bind this connection as the current console client gRPC connection
	c.conn = conn

	// Register RPC Service Client.
	transport.RPC = rpcpb.NewSliverRPCClient(conn)
	if transport.RPC == nil {
		return nil, errors.New("could not register gRPC Client, instance is nil")
	}

	// If this client is the server itself, register the admin RPC functions.
	c.admin = admin
	if c.admin {
		transport.AdminRPC = rpcpb.NewSliverAdminRPCClient(conn)
		if transport.AdminRPC == nil {
			return nil, errors.New("could not register gRPC Admin Client, instance is nil")
		}
	}

	// Start message tunnel loop.
	go transport.TunnelLoop()

	return conn, nil
}

// setup - The console sets up various elements such as the completion system, hints,
// syntax highlighting, prompt system, commands binding, and client environment loading.
func (c *console) setup() (err error) {

	initLogging() // textfile log

	// Get the user's console configuration from the server
	err = cctx.LoadConsoleConfig(transport.RPC)
	if err != nil {
		fmt.Printf(util.Error + "Failed to load console configuration from server.\n")
		fmt.Printf(util.Info + "Defaulting to builtin values.\n")
	}

	// This context object will hold some state about the
	// console (which implant we're interacting, jobs, etc.)
	cctx.Initialize(transport.RPC)

	// This computes all callbacks and base prompt strings
	// for the first time, and then binds it to the console.
	c.initPrompt()

	// Completions and syntax highlighting
	c.Shell.TabCompleter = completers.TabCompleter
	c.Shell.SyntaxHighlighter = completers.SyntaxHighlighter

	// History (client and user-wide)
	c.Shell.History = ClientHist
	c.Shell.AltHistory = UserHist

	// Client-side environment
	err = util.LoadClientEnv()
	if err != nil {
		return fmt.Errorf("could not load client OS env (%s)", err)
	}

	return
}

// Start - The console has a working RPC connection: we setup all
// things pertaining to the console itself, and start the input loop.
func (c *console) Start() (err error) {

	// Setup console elements
	err = c.setup()
	if err != nil {
		return fmt.Errorf("Console setup failed: %s", err)
	}

	// Start monitoring all logs from the server and the client.
	err = clientLog.Init(c.Shell, Prompt.Render, transport.RPC)
	if err != nil {
		return fmt.Errorf("Failed to start log monitor (%s)", err.Error())
	}

	// Start monitoring incoming events
	go c.handleServerLogs(transport.RPC)

	// Setup the Client Comm system (console proxies & port forwarders)
	err = initComm(transport.RPC, []byte(assets.ServerPrivateKey), assets.CommFingerprint)
	if err != nil {
		fmt.Printf(Warn+"Comm Error: %v \n", err)
	}

	// When we will exit this loop, disconnect gracefully from the server.
	defer c.conn.Close()

	// Commands binding. Per-context parsers are setup here.
	err = commands.BindCommands(c.admin, completers.LoadCompsAdditional)

	// Print banner and version information. (checks last updates)
	printLogo()

	// Start input loop
	for {

		// Some commands can act on the shell properties via the console
		// context package, so we check values and set everything up.
		c.setConfiguredShell()

		// Reset the completion data cache for all registered sessions.
		completers.Cache.Reset()

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
		c.ExecuteCommand(tokenParsed)
	}
}

// live refresh of console properties (input, hints, etc)
func (c *console) setConfiguredShell() {

	// Input
	if !cctx.Config.Vim {
		c.Shell.ShowVimMode = false
	} else {
		c.Shell.ShowVimMode = true
	}

	// Hints are configurable and can deactivated
	if !cctx.Config.Hints {
		c.Shell.HintText = nil
	} else {
		c.Shell.HintText = completers.HintCompleter
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

// initComm - Connect the Client Comm system to the server. This function
// could be moved to the comm package: I kept it here for uniformity of Init()
// function signatures between client/server/implant comm packages.
func initComm(rpc rpcpb.SliverRPCClient, key []byte, fingerprint string) error {

	stream, err := rpc.InitComm(context.Background(), &grpc.EmptyCallOption{})
	if err != nil {
		return err
	}

	// We need to create a callback so the conn knows how to decode/encode
	// arbitrary byte slices for our proto type.
	fieldFunc := func(msg proto.Message) *[]byte {
		return &msg.(*commpb.Bytes).Data
	}

	// Wrap our conn around the response.
	conn := &grpcConn.Conn{
		Stream:   stream,
		Request:  &commpb.Bytes{},
		Response: &commpb.Bytes{},
		Encode:   grpcConn.SimpleEncoder(fieldFunc),
		Decode:   grpcConn.SimpleDecoder(fieldFunc),
	}

	// The connection is a valid net.Conn upon which we can setup SSH.
	// We pass the commonName for SSH public key fingerprinting.
	return comm.InitClient(conn, key, fingerprint)
}
