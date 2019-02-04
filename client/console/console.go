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
	"sliver/client/core"
	"sliver/client/transport"

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

// Start - Main entrypoint
func Start() {
	configs := assets.GetConfigs()
	if len(configs) == 0 {
		fmt.Printf(Warn+"No config files found at %s\n", assets.GetConfigDir())
		return
	}
	config := selectConfig()
	if config == nil {
		return
	}
	rpc, recv, err := transport.Connect(config)
	if err != nil {
		fmt.Printf(Warn+"Connection to server failed %v", err)
		return
	}
	defer func() {
		close(rpc)
		close(recv)
	}()

	sliverServer := core.BindSliverServer(rpc, recv)
	go sliverServer.ResponseMapper()

	startConsole(sliverServer)
}

// Start - Console entrypoint
func startConsole(sliverServer *core.SliverServer) {
	sliverClientApp := grumble.New(&grumble.Config{
		Name:                  "consts.sliver client",
		Description:           "Bishop Fox - Sliver Client",
		HistoryFile:           path.Join(assets.GetRootAppDir(), "history"),
		Prompt:                getPrompt(),
		PromptColor:           color.New(),
		HelpHeadlineColor:     color.New(),
		HelpHeadlineUnderline: true,
		HelpSubCommands:       true,
	})
	sliverClientApp.SetPrintASCIILogo(printLogo)

	cmd.Init(sliverClientApp, sliverServer.RequestResponse)

	cmd.ActiveSliver.AddObserver(func() {
		sliverClientApp.SetPrompt(getPrompt())
	})

	go eventLoop(sliverClientApp, core.Events)

	err := sliverClientApp.Run()
	if err != nil {
		log.Printf("Run loop returned error: %v", err)
	}
}

func eventLoop(sliverApp *grumble.App, events chan core.Event) {
	stdout := bufio.NewWriter(os.Stdout)
	for event := range events {
		sliver := event.Sliver
		job := event.Job
		switch event.EventType {
		case "stopped":
			fmt.Printf(clearln+Warn+"Job #%d stopped (%s/%s)\n", job.ID, job.Protocol, job.Name)
		case "connected":
			fmt.Printf(clearln+Info+"Session #%d %s - %s (%s) - %s/%s\n",
				sliver.ID, sliver.Name, sliver.RemoteAddress, sliver.Hostname, sliver.OS, sliver.Arch)
		case "disconnected":
			fmt.Printf(clearln+Warn+"Lost session #%d %s - %s (%s) - %s/%s\n",
				sliver.ID, sliver.Name, sliver.RemoteAddress, sliver.Hostname, sliver.OS, sliver.Arch)
			if cmd.ActiveSliver.Sliver != nil && sliver.ID == cmd.ActiveSliver.Sliver.ID {
				cmd.ActiveSliver.SetActiveSliver(nil)
				fmt.Printf(Warn + "Warning: Active sliver diconnected\n")
			}
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
