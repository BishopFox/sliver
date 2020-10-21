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
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	insecureRand "math/rand"
	"os"
	"path"

	"github.com/bishopfox/sliver/client/assets"
	cmd "github.com/bishopfox/sliver/client/command"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	"time"

	"github.com/desertbit/grumble"
	"github.com/fatih/color"
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

// ExtraCmds - Bind extra commands to the app object
type ExtraCmds func(*grumble.App, rpcpb.SliverRPCClient)

// Start - Console entrypoint
func Start(rpc rpcpb.SliverRPCClient, extraCmds ExtraCmds) error {
	app := grumble.New(&grumble.Config{
		Name:                  "Sliver",
		Description:           "Sliver Client",
		HistoryFile:           path.Join(assets.GetRootAppDir(), "history"),
		Prompt:                getPrompt(),
		PromptColor:           color.New(),
		HelpHeadlineColor:     color.New(),
		HelpHeadlineUnderline: true,
		HelpSubCommands:       true,
	})
	app.SetPrintASCIILogo(func(app *grumble.App) {
		printLogo(app, rpc)
	})

	cmd.BindCommands(app, rpc)
	extraCmds(app, rpc)

	cmd.ActiveSession.AddObserver(func(_ *clientpb.Session) {
		app.SetPrompt(getPrompt())
	})

	go eventLoop(app, rpc)
	go core.TunnelLoop(rpc)

	err := app.Run()
	if err != nil {
		log.Printf("Run loop returned error: %v", err)
	}
	return err
}

func eventLoop(app *grumble.App, rpc rpcpb.SliverRPCClient) {
	eventStream, err := rpc.Events(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	stdout := bufio.NewWriter(os.Stdout)

	for {
		event, err := eventStream.Recv()
		if err == io.EOF || event == nil {
			return
		}

		// Trigger event based on type
		switch event.EventType {

		case consts.CanaryEvent:
			fmt.Printf(clearln+Warn+bold+"WARNING: %s%s has been burned (DNS Canary)\n", normal, event.Session.Name)
			sessions := cmd.GetSessionsByName(event.Session.Name, rpc)
			for _, session := range sessions {
				fmt.Printf(clearln+"\t🔥 Session #%d is affected\n", session.ID)
			}
			fmt.Println()

		case consts.JoinedEvent:
			fmt.Printf(clearln+Info+"%s has joined the game\n\n", event.Client.Operator.Name)
		case consts.LeftEvent:
			fmt.Printf(clearln+Info+"%s left the game\n\n", event.Client.Operator)

		case consts.JobStoppedEvent:
			job := event.Job
			fmt.Printf(clearln+Warn+"Job #%d stopped (%s/%s)\n\n", job.ID, job.Protocol, job.Name)

		case consts.SessionOpenedEvent:
			session := event.Session
			// The HTTP session handling is performed in two steps:
			// - first we add an "empty" session
			// - then we complete the session info when we receive the Register message from the Sliver
			// This check is here to avoid displaying two sessions events for the same session
			if session.OS != "" {
				currentTime := time.Now().Format(time.RFC1123)
				fmt.Printf(clearln+Info+"Session #%d %s - %s (%s) - %s/%s - %v\n\n",
					session.ID, session.Name, session.RemoteAddress, session.Hostname, session.OS, session.Arch, currentTime)
			}

		case consts.SessionUpdateEvent:
			session := event.Session
			currentTime := time.Now().Format(time.RFC1123)
			fmt.Printf(clearln+Info+"Session #%d has been updated - %v\n", session.ID, currentTime)

		case consts.SessionClosedEvent:
			session := event.Session
			fmt.Printf(clearln+Warn+"Lost session #%d %s - %s (%s) - %s/%s\n",
				session.ID, session.Name, session.RemoteAddress, session.Hostname, session.OS, session.Arch)
			activeSession := cmd.ActiveSession.Get()
			if activeSession != nil && activeSession.ID == session.ID {
				cmd.ActiveSession.Set(nil)
				app.SetPrompt(getPrompt())
				fmt.Printf(Warn + " Active session disconnected\n")
			}
			fmt.Println()
		}

		fmt.Printf(getPrompt())
		stdout.Flush()
	}
}

func getPrompt() string {
	prompt := underline + "sliver" + normal
	if cmd.ActiveSession.Get() != nil {
		prompt += fmt.Sprintf(bold+red+" (%s)%s", cmd.ActiveSession.Get().Name, normal)
	}
	prompt += " > "
	return prompt
}

func printLogo(sliverApp *grumble.App, rpc rpcpb.SliverRPCClient) {
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
		fmt.Printf(Info+"Client v%s\n", version.FullVersion())
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
}
