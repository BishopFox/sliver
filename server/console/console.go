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
	"log"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/jessevdk/go-flags"
	"github.com/maxlandon/readline"

	clientAssets "github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/commands"
	"github.com/bishopfox/sliver/client/completers"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/help"
	clientLog "github.com/bishopfox/sliver/client/log"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/server/assets"
)

const (
	// ANSI Colors
	normal    = "\033[0m"
	black     = "\033[30m"
	red       = "\033[31m"
	green     = "\033[32m"
	orange    = "\033[33m"
	blue      = "\033[34m"
	purple    = "\033[35m"
	cyan      = "\033[36m"
	gray      = "\033[37m"
	bold      = "\033[1m"
	clearln   = "\r\x1b[2K"
	upN       = "\033[%dA"
	downN     = "\033[%dB"
	underline = "\033[4m"

	// Info - Display colorful information
	Info = bold + cyan + "[*] " + normal
	// Warn - Warn a user
	Warn = bold + red + "[!] " + normal
	// Debug - Display debug information
	Debug = bold + purple + "[-] " + normal
	// Woot - Display success
	Woot = bold + green + "[$] " + normal
)

// Start - Start a server console (locally connected)
func Start() {

	// Declare an Config for the server-as-client, because its Comm system needs a
	// fingerprint value as well for authenticating to itself.
	clientAssets.Config = new(clientAssets.ClientConfig)

	// Get a gRPC client connection (in-memory listener)
	conn, err := connectLocal()
	if err != nil {
		os.Exit(1)
	}

	// Create a new server console, with a special run loop.
	// It embeds a client console so everything else remains the same.
	serverConsole := newServer()

	// Setup the server console
	// Register gRPC services, monitor events, setup the
	// console, logging, prompt/command/completion, etc.
	// This function is specific to the server binary,
	// because it needs to register flag commands (non-RPC, binary only)
	err = serverConsole.Init(conn)
	if err != nil {
		log.Fatal(err)
	}

	// Run the console loop (blocking)
	serverConsole.Run()
}

// Server - The server console needs to have access to admin commands,
// and we cannot make those accessible through RPC for obvious security reasons.
// This type is therefore a wrapper around the client console, with a custom Run()
// function that simply adds the admin commands to the shell.
type Server struct {
	*console.Client
}

// newServer - A new server console
func newServer() *Server {
	return &Server{
		&console.Client{
			Shell: readline.NewInstance(),
		},
	}
}

// Run - The server console simply adds the admin commands
// at each readline loop, everything else remains identical
func (c *Server) Run() {

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

		// SERVER-ONLY: bind admin commands to the parser
		bindServerAdminCommands(cmds)

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

// bindServerAdminCommands - We bind commands only available to the server admin to the console command parser.
// Unfortunately we have to use, for each command, its Aliases field where we register its "namespace".
// There is a namespace field, however it messes up with the option printing/detection/parsing.
func bindServerAdminCommands(parser *flags.Parser) (err error) {

	np, err := parser.AddCommand(constants.NewPlayerStr,
		"Create a new player config file",
		help.GetHelpFor(constants.NewPlayerStr),
		&NewOperator{})
	cctx.Commands.RegisterServerCommand(err, np, constants.AdminGroup)

	kp, err := parser.AddCommand(constants.KickPlayerStr,
		"Kick a player from the server",
		help.GetHelpFor(constants.KickPlayerStr),
		&KickOperator{})
	cctx.Commands.RegisterServerCommand(err, kp, constants.AdminGroup)

	mm, err := parser.AddCommand(constants.MultiplayerModeStr,
		"Enable multiplayer mode on this server",
		help.GetHelpFor(constants.MultiplayerModeStr),
		&MultiplayerMode{})
	cctx.Commands.RegisterServerCommand(err, mm, constants.AdminGroup)

	return
}

func checkForLegacyDB() {
	legacyDBPath := filepath.Join(assets.GetRootAppDir(), "db")
	if _, err := os.Stat(legacyDBPath); !os.IsNotExist(err) {
		fmt.Println("\n" + Warn + bold + "Compatability Warning: " + normal)
		fmt.Println(Warn + "It looks like this server was upgraded from an older version of Sliver.")
		fmt.Println(Warn + "We have switched to using SQL for internal data (before we used BadgerDB)")
		fmt.Printf(Warn+"Regrettably this means there's %sno backwards compatibility%s for implants, etc.\n", bold, normal)
		fmt.Println(Warn + "If you need to use existing implants, stick with any version up to 1.0.9")
		fmt.Println()
		confirm := false
		survey.AskOne(&survey.Confirm{
			Message: "Delete old database (CANNOT BE UNDONE)",
		}, &confirm)
		if confirm {
			os.RemoveAll(legacyDBPath)
		}
	}
}
