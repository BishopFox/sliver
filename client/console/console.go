package console

import (
	"bufio"
	"fmt"
	"log"
	insecureRand "math/rand"
	"os"
	"path"
	"sliver/client/assets"
	cmd "sliver/client/command"
	consts "sliver/client/constants"
	"sliver/client/core"

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
func Start(server *core.SliverServer, extraCmds ExtraCmds) {
	app := grumble.New(&grumble.Config{
		Name:                  "Sliver",
		Description:           "Bishop Fox - Sliver Client",
		HistoryFile:           path.Join(assets.GetRootAppDir(), "history"),
		Prompt:                getPrompt(),
		PromptColor:           color.New(),
		HelpHeadlineColor:     color.New(),
		HelpHeadlineUnderline: true,
		HelpSubCommands:       true,
	})
	app.SetPrintASCIILogo(printLogo)

	cmd.Init(app, server)
	extraCmds(app, server)

	cmd.ActiveSliver.AddObserver(func() {
		app.SetPrompt(getPrompt())
	})

	go eventLoop(app, server)

	err := app.Run()
	if err != nil {
		log.Printf("Run loop returned error: %v", err)
	}
}

func eventLoop(app *grumble.App, server *core.SliverServer) {
	stdout := bufio.NewWriter(os.Stdout)
	for event := range server.Events {

		switch event.EventType {

		case consts.ServerErrorStr:
			fmt.Printf(clearln + Warn + "Server connection error!\n\n")
			os.Exit(1)

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
				fmt.Printf(Warn + "Warning: Active sliver diconnected\n")
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
	fmt.Println(Info + "Welcome to the sliver shell, please type 'help' for options")
	fmt.Println()
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
