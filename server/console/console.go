package console

import (
	"bufio"
	"fmt"
	"log"
	insecureRand "math/rand"
	"os"
	"path"
	"sliver/client/command"
	consts "sliver/client/constants"
	"sliver/client/help"
	pb "sliver/protobuf/client"
	"time"

	"sliver/server/rpc"

	"github.com/desertbit/grumble"
	"github.com/fatih/color"

	"sliver/server/assets"
	"sliver/server/core"
	"sliver/server/generate"
)

var (
	activeSliver *core.Sliver
	cmdTimeout   = 10 * time.Second
)

// Start - Starts the main server console
func Start() {

	sliverApp := grumble.New(&grumble.Config{
		Name:                  "sliver",
		Description:           "Bishop Fox - Sliver",
		HistoryFile:           path.Join(assets.GetRootAppDir(), "history"),
		Prompt:                getPrompt(),
		PromptColor:           color.New(),
		HelpHeadlineColor:     color.New(),
		HelpHeadlineUnderline: true,
		HelpSubCommands:       true,
	})
	sliverApp.SetPrintASCIILogo(printLogo)

	serverOnlyCmds(sliverApp)

	handlers := rpc.GetRPCHandlers()
	command.Init(sliverApp, func(envelope *pb.Envelope, timeout time.Duration) *pb.Envelope {
		resp := make(chan *pb.Envelope)
		defer close(resp)
		if handler, ok := (*handlers)[envelope.Type]; ok {
			go handler(envelope.Data, func(data []byte, err error) {
				errStr := ""
				if err != nil {
					errStr = fmt.Sprintf("%v", err)
				}
				resp <- &pb.Envelope{
					ID:    envelope.ID,
					Data:  data,
					Error: errStr,
				}
			})
			return <-resp
		}
		fmt.Println()
		return nil
	})

	command.ActiveSliver.AddObserver(func() {
		sliverApp.SetPrompt(getPrompt())
	})

	events := core.EventBroker.Subscribe()
	defer core.EventBroker.Unsubscribe(events)
	go eventLoop(sliverApp, events)

	err := sliverApp.Run()
	if err != nil {
		log.Printf("Run loop returned error: %v", err)
	}
}

func eventLoop(sliverApp *grumble.App, events chan core.Event) {
	stdout := bufio.NewWriter(os.Stdout)
	for event := range events {

		switch event.EventType {
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
				sliver.ID, sliver.Name, sliver.RemoteAddress, sliver.Hostname, sliver.Os, sliver.Arch)
		case consts.DisconnectedEvent:
			sliver := event.Sliver
			fmt.Printf(clearln+Warn+"Lost session #%d %s - %s (%s) - %s/%s\n",
				sliver.ID, sliver.Name, sliver.RemoteAddress, sliver.Hostname, sliver.Os, sliver.Arch)
			activeSliver := command.ActiveSliver.Sliver
			if activeSliver != nil && int32(sliver.ID) == activeSliver.ID {
				command.ActiveSliver.SetActiveSliver(nil)
				sliverApp.SetPrompt(getPrompt())
				fmt.Printf(Warn + "Warning: Active sliver diconnected\n")
			}
			fmt.Println()
		}
		fmt.Printf(getPrompt())
		stdout.Flush()
	}
}

func serverOnlyCmds(sliverApp *grumble.App) {

	// [ Multiplayer ] -----------------------------------------------------------------

	sliverApp.AddCommand(&grumble.Command{
		Name:     consts.MultiplayerModeStr,
		Help:     "Enable multiplayer mode",
		LongHelp: help.GetHelpFor(consts.MultiplayerModeStr),
		Flags: func(f *grumble.Flags) {
			f.String("s", "server", "", "interface to bind server to")
			f.Int("l", "lport", 31337, "tcp listen port")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			startMultiplayerModeCmd(ctx)
			fmt.Println()
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     consts.NewPlayerStr,
		Help:     "Create a new player config file",
		LongHelp: help.GetHelpFor(consts.NewPlayerStr),
		Flags: func(f *grumble.Flags) {
			f.String("o", "os", generate.WINDOWS, "operating system")
			f.String("a", "arch", "amd64", "cpu architecture")
			f.String("h", "lhost", "", "listen host")
			f.Int("l", "lport", 31337, "listen port")
			f.Bool("d", "debug", false, "enable debug features")
			f.String("s", "save", "", "directory/file to the binary to")
			f.String("n", "operator", "", "operator name")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			newPlayerCmd(ctx)
			fmt.Println()
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     consts.ListPlayerStr,
		Help:     "List players connected to the server",
		LongHelp: help.GetHelpFor(consts.ListPlayerStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			listPlayersCmd(ctx)
			fmt.Println()
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     consts.KickPlayerStr,
		Help:     "Kick a player from the server",
		LongHelp: help.GetHelpFor(consts.KickPlayerStr),
		Flags: func(f *grumble.Flags) {
			f.String("o", "operator", "", "operator name")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			kickPlayerCmd(ctx)
			fmt.Println()
			return nil
		},
	})

}

func getPrompt() string {
	prompt := underline + "sliver" + normal
	if command.ActiveSliver.Sliver != nil {
		prompt += fmt.Sprintf(bold+red+" (%s)%s", command.ActiveSliver.Sliver.Name, normal)
	}
	prompt += " > "
	return prompt
}

func printLogo(sliverApp *grumble.App) {
	insecureRand.Seed(time.Now().Unix())
	logo := asciiLogos[insecureRand.Intn(len(asciiLogos))]
	fmt.Println(logo)
	fmt.Println(Info + "Welcome to the sliver server shell, please type 'help' for options")
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
