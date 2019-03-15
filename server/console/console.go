package console

import (
	"fmt"
	insecureRand "math/rand"
	"sliver/client/command"
	clientconsole "sliver/client/console"
	consts "sliver/client/constants"
	clientcore "sliver/client/core"
	"sliver/client/help"
	sliverpb "sliver/protobuf/sliver"

	"time"

	"github.com/desertbit/grumble"

	"sliver/server/generate"
	"sliver/server/transport"
)

// Start - Starts the server console
func Start() {
	send := make(chan *sliverpb.Envelope) // To "server"
	recv := make(chan *sliverpb.Envelope) // From "server"

	transport.LocalClientConnect(send, recv) // Simulates a client connection

	server := clientcore.BindSliverServer(send, recv)
	go server.ResponseMapper()
	clientconsole.Start(server, serverOnlyCmds)
}

// ServerOnlyCmds - Server only commands
func serverOnlyCmds(app *grumble.App, server *clientcore.SliverServer) {

	// [ Multiplayer ] -----------------------------------------------------------------

	app.AddCommand(&grumble.Command{
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
		HelpGroup: consts.MultiplayerHelpGroup,
	})

	app.AddCommand(&grumble.Command{
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
		HelpGroup: consts.MultiplayerHelpGroup,
	})

	app.AddCommand(&grumble.Command{
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
		HelpGroup: consts.MultiplayerHelpGroup,
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

func printLogo(app *grumble.App) {
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
