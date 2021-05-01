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
	"fmt"
	"log"
	insecureRand "math/rand"
	"time"

	"github.com/desertbit/grumble"
	"github.com/jessevdk/go-flags"
	"github.com/maxlandon/gonsole"

	"github.com/bishopfox/sliver/client/assets"
	cmd "github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/client/completion"
	"github.com/bishopfox/sliver/client/constants"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/core"
	clientLog "github.com/bishopfox/sliver/client/log"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
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

	// ensure that nothing remains when we refresh the prompt
	seqClearScreenBelow = "\x1b[0J"
)

var (
	// Console - The console instance of this client.
	Console = gonsole.NewConsole()
)

// ExtraCmds - Bind extra commands to the app object
type ExtraCmds func(*grumble.App, rpcpb.SliverRPCClient)

// Start - Console entrypoint
func Start(rpc rpcpb.SliverRPCClient, extraCmds ExtraCmds, config *assets.ClientConfig) error {
	// func Start(rpc rpcpb.SliverRPCClient, extraCmds ExtraCmds) error {
	// Keep the config reference
	serverConfig = config

	// Start monitoring tunnels
	go core.TunnelLoop(rpc)

	// Create and setup the client console
	err := setup(rpc)
	if err != nil {
		return fmt.Errorf("Console setup failed: %s", err)
	}

	// Start monitoring all logs from the server and the client.
	err = clientLog.Init(Console, rpc)
	if err != nil {
		return fmt.Errorf("Failed to start log monitor (%s)", err.Error())
	}

	// Start monitoring incoming events
	go eventLoop(rpc)

	// Print banner and version information. (checks last updates)
	printLogo(rpc)

	// Run the console. All errors are handled internally.
	Console.Run()

	return nil
}

// setup - Sets everything directly related to the client "part". This includes the full
// console configuration, setup, history loading, menu contexts, command registration, etc..
func setup(rpc rpcpb.SliverRPCClient) (err error) {

	// Declare server and sliver contexts (menus).
	server := Console.NewMenu(consts.ServerMenu)
	sliver := Console.NewMenu(consts.SliverMenu)

	// The current one is the server
	Console.SwitchMenu(consts.ServerMenu)

	// Get the user's console configuration from the server, and load it in the console.
	config, err := loadConsoleConfig(rpc)
	if err != nil {
		fmt.Printf(Warn + "Failed to load console configuration from server.\n")
		fmt.Printf(Info + "Defaulting to builtin values.\n")
	}
	Console.LoadConfig(config)

	// Set prompts callback functions for both contexts
	server.Prompt.Callbacks = serverCallbacks
	sliver.Prompt.Callbacks = sliverCallbacks

	// Set history sources for both contexts
	setHistorySources()

	// Setup parser details
	Console.SetParserOptions(flags.IgnoreUnknown | flags.HelpFlag)

	// Bind commands. In this function we also add some gonsole-provided
	// default commands, for help and console configuration management.
	// commands.BindCommands(Console)

	return nil
}

// setHistorySources - Both contexts have different history sources available to the user.
func setHistorySources() {

	// Server context
	server := Console.GetMenu(constants.ServerMenu)
	server.SetHistoryCtrlR("user-wise history", UserHist)
	server.SetHistoryAltR("client history", ClientHist)

	// Request a copy of the user history to the server
	getUserHistory()

	// Sliver context
	sliver := Console.GetMenu(constants.SliverMenu)
	sliver.SetHistoryCtrlR("session history", SessionHist)
	sliver.SetHistoryAltR("user-wise history", UserHist)

	// We pass a function to the core package, which will
	// allow to refresh the session history as soon as we
	// interact with it.
	core.SessionHistoryFunc = SessionHist.RefreshLines
}

// eventLoop - Print events coming from the server
func eventLoop(rpc rpcpb.SliverRPCClient) {

	// Call the server events stream.
	events, err := rpc.Events(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	for !isDone(events.Context()) {
		event, err := events.Recv()
		if err != nil {
			fmt.Printf(Warn + "It seems that the Sliver Server disconnected, falling back...\n")
			return
		}

		switch event.EventType {
		case consts.CanaryEvent:
			fmt.Printf("\n\n") // Clear screen a bit before announcing shitty news
			fmt.Printf(Warn+"WARNING: %s%s has been burned (DNS Canary)\n", normal, event.Session.Name)
			sessions := getSessionsByName(event.Session.Name, transport.RPC)
			var alert string
			for _, session := range sessions {
				alert += fmt.Sprintf("\t🔥 Session #%d is affected\n", session.ID)
			}
			Console.RefreshPromptLog(alert)

		case consts.JobStoppedEvent:
			job := event.Job
			line := fmt.Sprintf(Info+"Job #%d stopped (%s/%s)\n", job.ID, job.Protocol, job.Name)
			Console.RefreshPromptLog(line)

		case consts.SessionOpenedEvent:
			session := event.Session

			// Create a new session data cache for completions
			completion.Cache.AddSessionCache(session)

			// Clear the screen
			fmt.Print(seqClearScreenBelow)

			// The HTTP session handling is performed in two steps:
			// - first we add an "empty" session
			// - then we complete the session info when we receive the Register message from the Sliver
			// This check is here to avoid displaying two sessions events for the same session
			var news string
			if session.OS != "" {
				currentTime := time.Now().Format(time.RFC1123)
				news += fmt.Sprintf("\n\n") // Clear screen a bit before announcing the king
				news += fmt.Sprintf(Info+"Session #%d %s - %s (%s) - %s/%s - %v\n\n",
					session.ID, session.Name, session.RemoteAddress, session.Hostname, session.OS, session.Arch, currentTime)
				prompt := Console.CurrentMenu().Prompt.Render()
				Console.RefreshPromptCustom(news, prompt, 0)
			}

		case consts.SessionUpdateEvent:
			session := event.Session
			currentTime := time.Now().Format(time.RFC1123)
			updated := fmt.Sprintf(Info+"Session #%d has been updated - %v\n", session.ID, currentTime)
			if core.ActiveSession != nil && session.ID == core.ActiveSession.ID {
				prompt := Console.CurrentMenu().Prompt.Render()
				Console.RefreshPromptCustom(updated, prompt, 0)
			} else {
				Console.RefreshPromptLog(updated)
			}

		case consts.SessionClosedEvent:
			session := event.Session
			var lost string

			// If the session is our current session, we notify the console
			if core.ActiveSession != nil && session.ID == core.ActiveSession.ID {
				Console.SwitchMenu(consts.ServerMenu)
				core.ActiveSession = nil
			}

			// We print a message here if its not about a session we killed ourselves, and adapt prompt
			lost += fmt.Sprintf(Warn+"Lost session #%d %s - %s (%s) - %s/%s\n",
				session.ID, session.Name, session.RemoteAddress, session.Hostname, session.OS, session.Arch)
			Console.RefreshPromptLog(lost)

			// In any case, delete the completion data cache for the session, if any.
			completion.Cache.RemoveSessionData(session)
		}
	}
}

func isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// getSessionsByName - Return all sessions for an Implant by name
func getSessionsByName(name string, rpc rpcpb.SliverRPCClient) []*clientpb.Session {
	sessions, err := rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil
	}
	matched := []*clientpb.Session{}
	for _, session := range sessions.GetSessions() {
		if session.Name == name {
			matched = append(matched, session)
		}
	}
	return matched
}

func printLogo(rpc rpcpb.SliverRPCClient) {
	serverVer, err := rpc.GetVersion(context.Background(), &commonpb.Empty{})
	if err != nil {
		panic(err.Error())
	}
	dirty := ""
	if serverVer.Dirty {
		dirty = fmt.Sprintf(" - %sDirty%s", bold, normal)
	}
	serverSemVer := fmt.Sprintf("%d.%d.%d", serverVer.Major, serverVer.Minor, serverVer.Patch)

	insecureRand.Seed(time.Now().Unix())
	logo := asciiLogos[insecureRand.Intn(len(asciiLogos))]
	fmt.Println(logo)
	fmt.Println("All hackers gain " + abilities[insecureRand.Intn(len(abilities))])
	fmt.Printf(Info+"Server v%s - %s%s\n", serverSemVer, serverVer.Commit, dirty)
	if version.GitCommit != serverVer.Commit {
		fmt.Printf(Info+"Client %s\n", version.FullVersion())
	}
	fmt.Println(Info + "Welcome to the sliver shell, please type 'help' for options")
	fmt.Println()
	if serverVer.Major != int32(version.SemanticVersion()[0]) {
		fmt.Printf(Warn + "Warning: Client and server may be running incompatible versions.\n")
	}
	checkLastUpdate()
}

func checkLastUpdate() {
	now := time.Now()
	lastUpdate := cmd.GetLastUpdateCheck()
	compiledAt, err := version.Compiled()
	if err != nil {
		log.Printf("Failed to parse compiled at timestamp %s", err)
		return
	}

	day := 24 * time.Hour
	if compiledAt.Add(30 * day).Before(now) {
		if lastUpdate == nil || lastUpdate.Add(30*day).Before(now) {
			fmt.Printf(Info + "Check for updates with the 'update' command\n\n")
		}
	}
}

var abilities = []string{
	"first strike",
	"vigilance",
	"haste",
	"indestructible",
	"hexproof",
	"deathtouch",
	"fear",
	"epic",
	"ninjitsu",
	"recover",
	"persist",
	"conspire",
	"reinforce",
	"exalted",
	"annihilator",
	"infect",
	"undying",
	"living weapon",
	"miracle",
	"scavenge",
	"cipher",
	"evolve",
	"dethrone",
	"hidden agenda",
	"prowess",
	"dash",
	"exploit",
	"renown",
	"skulk",
	"improvise",
	"assist",
	"jump-start",
}

var asciiLogos = []string{
	red + `
 	  ██████  ██▓     ██▓ ██▒   █▓▓█████  ██▀███
	▒██    ▒ ▓██▒    ▓██▒▓██░   █▒▓█   ▀ ▓██ ▒ ██▒
	░ ▓██▄   ▒██░    ▒██▒ ▓██  █▒░▒███   ▓██ ░▄█ ▒
	  ▒   ██▒▒██░    ░██░  ▒██ █░░▒▓█  ▄ ▒██▀▀█▄
	▒██████▒▒░██████▒░██░   ▒▀█░  ░▒████▒░██▓ ▒██▒
	▒ ▒▓▒ ▒ ░░ ▒░▓  ░░▓     ░ ▐░  ░░ ▒░ ░░ ▒▓ ░▒▓░
	░ ░▒  ░ ░░ ░ ▒  ░ ▒ ░   ░ ░░   ░ ░  ░  ░▒ ░ ▒░
	░  ░  ░    ░ ░    ▒ ░     ░░     ░     ░░   ░
		  ░      ░  ░ ░        ░     ░  ░   ░
` + normal,

	green + `
    ███████╗██╗     ██╗██╗   ██╗███████╗██████╗
    ██╔════╝██║     ██║██║   ██║██╔════╝██╔══██╗
    ███████╗██║     ██║██║   ██║█████╗  ██████╔╝
    ╚════██║██║     ██║╚██╗ ██╔╝██╔══╝  ██╔══██╗
    ███████║███████╗██║ ╚████╔╝ ███████╗██║  ██║
    ╚══════╝╚══════╝╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝
` + normal,

	bold + gray + `
.------..------..------..------..------..------.
|S.--. ||L.--. ||I.--. ||V.--. ||E.--. ||R.--. |
| :/\: || :/\: || (\/) || :(): || (\/) || :(): |
| :\/: || (__) || :\/: || ()() || :\/: || ()() |
| '--'S|| '--'L|| '--'I|| '--'V|| '--'E|| '--'R|
` + "`------'`------'`------'`------'`------'`------'" + `
` + normal,
}
