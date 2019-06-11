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
	"fmt"
	"log"
	insecureRand "math/rand"
	"os"
	"path"

	"github.com/bishopfox/sliver/client/assets"
	cmd "github.com/bishopfox/sliver/client/command"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/version"

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
type ExtraCmds func(*grumble.App, *core.SliverServer)

// Start - Console entrypoint
func Start(server *core.SliverServer, extraCmds ExtraCmds) error {
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
	app.SetPrintASCIILogo(printLogo)

	cmd.BindCommands(app, server)
	extraCmds(app, server)

	cmd.ActiveSliver.AddObserver(func() {
		app.SetPrompt(getPrompt())
	})

	go eventLoop(app, server)

	err := app.Run()
	if err != nil {
		log.Printf("Run loop returned error: %v", err)
	}
	return err
}

func eventLoop(app *grumble.App, server *core.SliverServer) {
	stdout := bufio.NewWriter(os.Stdout)
	for event := range server.Events {

		switch event.EventType {

		case consts.CanaryEvent:
			fmt.Printf(clearln+Warn+bold+"WARNING: %s%s has been burned (DNS Canary)\n", normal, event.Sliver.Name)
			sessions := cmd.SliverSessionsByName(event.Sliver.Name, server.RPC)
			for _, sliver := range sessions {
				fmt.Printf(clearln+"\t🔥 Session #%d is affected\n", sliver.ID)
			}
			fmt.Println()

		case consts.ServerErrorStr:
			fmt.Printf(clearln + Warn + "Server connection error!\n\n")
			os.Exit(4)

		case consts.JoinedEvent:
			fmt.Printf(clearln+Info+"%s has joined the game\n\n", event.Client.Operator)
		case consts.LeftEvent:
			fmt.Printf(clearln+Info+"%s left the game\n\n", event.Client.Operator)

		case consts.StoppedEvent:
			job := event.Job
			fmt.Printf(clearln+Warn+"Job #%d stopped (%s/%s)\n\n", job.ID, job.Protocol, job.Name)

		case consts.ConnectedEvent:
			sliver := event.Sliver
			fmt.Printf(clearln+Info+"Session #%d %s - %s (%s) - %s/%s\n\n",
				sliver.ID, sliver.Name, sliver.RemoteAddress, sliver.Hostname, sliver.OS, sliver.Arch)

		case consts.DisconnectedEvent:
			sliver := event.Sliver
			fmt.Printf(clearln+Warn+"Lost session #%d %s - %s (%s) - %s/%s\n",
				sliver.ID, sliver.Name, sliver.RemoteAddress, sliver.Hostname, sliver.OS, sliver.Arch)
			activeSliver := cmd.ActiveSliver.Sliver
			if activeSliver != nil && sliver.ID == activeSliver.ID {
				cmd.ActiveSliver.SetActiveSliver(nil)
				app.SetPrompt(getPrompt())
				fmt.Printf(Warn + " Active sliver diconnected\n")
			}
			fmt.Println()

		}

		fmt.Printf(getPrompt())
		stdout.Flush()
	}
}

func getPrompt() string {
	prompt := underline + "sliver" + normal
	if cmd.ActiveSliver.Sliver != nil {
		prompt += fmt.Sprintf(bold+red+" (%s)%s", cmd.ActiveSliver.Sliver.Name, normal)
	}
	prompt += " > "
	return prompt
}

func printLogo(sliverApp *grumble.App) {
	insecureRand.Seed(time.Now().Unix())
	logo := asciiLogos[insecureRand.Intn(len(asciiLogos))]
	fmt.Println(logo)
	fmt.Println("All hackers gain " + abilities[insecureRand.Intn(len(abilities))])
	fmt.Printf(Info+"v%s\n", version.FullVersion())
	fmt.Println(Info + "Welcome to the sliver shell, please type 'help' for options")
	fmt.Println()
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
