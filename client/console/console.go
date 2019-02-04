package console

import (
	"bufio"
	"fmt"
	"log"
	insecureRand "math/rand"
	"os"
	"path"
	"sliver/client/assets"
	"sliver/client/transport"
	pb "sliver/protobuf/client"
	"sliver/server/core"
	"sliver/server/generate"
	"sliver/server/help"
	"sync"
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

var (
	activeSliver *core.Sliver
	sliverServer *SliverServer
)

// SliverServer - Server info
type SliverServer struct {
	Send      chan *pb.Envelope
	recv      chan *pb.Envelope
	responses *map[string]chan *pb.Envelope
	mutex     *sync.RWMutex
	Config    *assets.ClientConfig
}

// ResponseMapper - Maps recv'd envelopes to response channels
func (ss *SliverServer) ResponseMapper() {
	for envelope := range ss.recv {
		if envelope.Id != "" {
			ss.mutex.Lock()
			if resp, ok := (*ss.responses)[envelope.Id]; ok {
				resp <- envelope
			}
			ss.mutex.Unlock()
		}
	}
}

// RequestResponse - Send a request envelope and wait for a response (blocking)
func (ss *SliverServer) RequestResponse(envelope *pb.Envelope) *pb.Envelope {
	reqID := core.RandomID()
	envelope.Id = reqID
	resp := make(chan *pb.Envelope)
	ss.AddRespListener(reqID, resp)
	defer ss.RemoveRespListener(reqID)
	ss.Send <- envelope
	respEnvelope := <-resp
	return respEnvelope
}

// AddRespListener - Add a response listener
func (ss *SliverServer) AddRespListener(requestID string, resp chan *pb.Envelope) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	(*ss.responses)[requestID] = resp
}

// RemoveRespListener - Remove a listener
func (ss *SliverServer) RemoveRespListener(requestID string) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	close((*ss.responses)[requestID])
	delete((*ss.responses), requestID)
}

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
	send, recv, err := transport.Connect(config)
	if err != nil {
		fmt.Printf(Warn+"Connection to server failed %v", err)
		return
	}
	defer func() {
		close(send)
		close(recv)
	}()
	sliverServer = &SliverServer{
		Send:      send,
		recv:      recv,
		responses: &map[string]chan *pb.Envelope{},
		mutex:     &sync.RWMutex{},
		Config:    config,
	}
	go sliverServer.ResponseMapper()

	startConsole()
}

func startConsole() {
	sliverClientApp := grumble.New(&grumble.Config{
		Name:                  "help.sliver client",
		Description:           "Bishop Fox - Sliver Client",
		HistoryFile:           path.Join(assets.GetRootAppDir(), "history"),
		Prompt:                getPrompt(),
		PromptColor:           color.New(),
		HelpHeadlineColor:     color.New(),
		HelpHeadlineUnderline: true,
		HelpSubCommands:       true,
	})
	sliverClientApp.SetPrintASCIILogo(printLogo)
	cmdInit(sliverClientApp)

	go eventLoop(sliverClientApp, core.Events)

	err := sliverClientApp.Run()
	if err != nil {
		log.Printf("Run loop returned error: %v", err)
	}
}

func cmdInit(sliverApp *grumble.App) {

	// [ Jobs ] -----------------------------------------------------------------

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.JobsStr,
		Help:     "Job control",
		LongHelp: help.GetHelpFor(help.JobsStr),
		Flags: func(f *grumble.Flags) {
			f.Int("k", "kill", -1, "kill a background job")
		},
		Run: func(ctx *grumble.Context) error {
			jobsCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.MtlsStr,
		Help:     "Start an mTLS listener",
		LongHelp: help.GetHelpFor(help.MtlsStr),
		Flags: func(f *grumble.Flags) {
			f.String("s", "server", "", "interface to bind server to")
			f.Int("l", "lport", 8888, "tcp listen port")
		},
		Run: func(ctx *grumble.Context) error {
			startMTLSListenerCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.DnsStr,
		Help:     "Start a DNS listener",
		LongHelp: help.GetHelpFor(help.DnsStr),
		Flags: func(f *grumble.Flags) {
			f.String("d", "domain", "", "parent domain to use for DNS C2")
		},
		Run: func(ctx *grumble.Context) error {
			startDNSListenerCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.MultiplayerModeStr,
		Help:     "Enable multiplayer mode",
		LongHelp: help.GetHelpFor(help.MultiplayerModeStr),
		Flags: func(f *grumble.Flags) {
			f.String("s", "server", "", "interface to bind server to")
			f.Int("l", "lport", 31337, "tcp listen port")
		},
		Run: func(ctx *grumble.Context) error {
			startMultiplayerModeCmd(ctx)
			return nil
		},
	})

	// [ Commands ] --------------------------------------------------------------

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.SessionsStr,
		Help:     "Session management",
		LongHelp: help.GetHelpFor(help.SessionsStr),
		Flags: func(f *grumble.Flags) {
			f.String("i", "interact", "", "interact with a sliver")
		},
		Run: func(ctx *grumble.Context) error {
			sessionsCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.BackgroundStr,
		Help:     "Background an active session",
		LongHelp: help.GetHelpFor(help.BackgroundStr),
		Run: func(ctx *grumble.Context) error {
			backgroundCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      help.KillStr,
		Help:      "Kill a remote sliver process",
		LongHelp:  help.GetHelpFor(help.KillStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			killCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      help.InfoStr,
		Help:      "Get info about sliver",
		LongHelp:  help.GetHelpFor(help.InfoStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			infoCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      help.UseStr,
		Help:      "Switch the active sliver",
		LongHelp:  help.GetHelpFor(help.UseStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			useCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.GenerateStr,
		Help:     "Generate a sliver binary",
		LongHelp: help.GetHelpFor(help.GenerateStr),
		Flags: func(f *grumble.Flags) {
			f.String("o", "os", generate.WINDOWS, "operating system")
			f.String("a", "arch", "amd64", "cpu architecture")
			f.String("h", "lhost", "", "listen host")
			f.Int("l", "lport", 8888, "listen port")
			f.Bool("d", "debug", false, "enable debug features")
			f.String("n", "dns", "", "dns c2 parent domain")
			f.String("s", "save", "", "directory/file to the binary to")
		},
		Run: func(ctx *grumble.Context) error {
			generateCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.MsfStr,
		Help:     "Execute a MSF payload",
		LongHelp: help.GetHelpFor(help.MsfStr),
		Flags: func(f *grumble.Flags) {
			f.String("m", "payload", "meterpreter_reverse_https", "msf payload")
			f.String("h", "lhost", "", "listen host")
			f.Int("l", "lport", 4444, "listen port")
			f.String("e", "encoder", "", "msf encoder")
			f.Int("i", "iterations", 1, "iterations of the encoder")
		},
		Run: func(ctx *grumble.Context) error {
			msfCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.InjectStr,
		Help:     "Inject a MSF payload",
		LongHelp: help.GetHelpFor(help.InjectStr),
		Flags: func(f *grumble.Flags) {
			f.Int("p", "pid", -1, "pid to inject into")
			f.String("m", "payload", "meterpreter_reverse_https", "msf payload")
			f.String("h", "lhost", "", "listen host")
			f.Int("l", "lport", 4444, "listen port")
			f.String("e", "encoder", "", "msf encoder")
			f.Int("i", "iterations", 1, "iterations of the encoder")
		},
		Run: func(ctx *grumble.Context) error {
			injectCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.PsStr,
		Help:     "List remote processes",
		LongHelp: help.GetHelpFor(help.PsStr),
		Flags: func(f *grumble.Flags) {
			f.Int("p", "pid", -1, "filter based on pid")
			f.String("e", "exe", "", "filter based on executable name")
			f.String("o", "owner", "", "filter based on owner")
		},
		Run: func(ctx *grumble.Context) error {
			psCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      help.PingStr,
		Help:      "Test connection to sliver",
		LongHelp:  help.GetHelpFor(help.PingStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			pingCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.GetPIDStr,
		Help:     "Get sliver pid",
		LongHelp: help.GetHelpFor(help.GetPIDStr),
		Run: func(ctx *grumble.Context) error {
			if activeSliver != nil {
				fmt.Printf("\n"+Info+"%d\n\n", activeSliver.PID)
			}
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.GetUIDStr,
		Help:     "Get sliver UID",
		LongHelp: help.GetHelpFor(help.GetUIDStr),
		Run: func(ctx *grumble.Context) error {
			if activeSliver != nil {
				fmt.Printf("\n"+Info+"%s\n\n", activeSliver.UID)
			}
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.GetGIDStr,
		Help:     "Get sliver GID",
		LongHelp: help.GetHelpFor(help.GetGIDStr),
		Run: func(ctx *grumble.Context) error {
			if activeSliver != nil {
				fmt.Printf("\n"+Info+"%s\n\n", activeSliver.GID)
			}
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.WhoamiStr,
		Help:     "Get sliver user",
		LongHelp: help.GetHelpFor(help.WhoamiStr),
		Run: func(ctx *grumble.Context) error {
			if activeSliver != nil {
				fmt.Printf("\n"+Info+"%s\n\n", activeSliver.Username)
			}
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.LsStr,
		Help:     "List current directory",
		LongHelp: help.GetHelpFor(help.LsStr),
		Run: func(ctx *grumble.Context) error {
			lsCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      help.RmStr,
		Help:      "Remove a file or directory",
		LongHelp:  help.GetHelpFor(help.RmStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			rmCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      help.MkdirStr,
		Help:      "Make a directory",
		LongHelp:  help.GetHelpFor(help.MkdirStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			mkdirCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      help.CdStr,
		Help:      "Change directory",
		LongHelp:  help.GetHelpFor(help.CdStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			cdCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.PwdStr,
		Help:     "Print working directory",
		LongHelp: help.GetHelpFor(help.PwdStr),
		Run: func(ctx *grumble.Context) error {
			pwdCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      help.CatStr,
		Help:      "Dump file to stdout",
		LongHelp:  help.GetHelpFor(help.CatStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			catCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      help.DownloadStr,
		Help:      "Download a file",
		LongHelp:  help.GetHelpFor(help.DownloadStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			downloadCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      help.UploadStr,
		Help:      "Upload a file",
		LongHelp:  help.GetHelpFor(help.UploadStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			uploadCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      help.ProcdumpStr,
		Help:      "Dump process memory",
		LongHelp:  help.GetHelpFor(help.ProcdumpStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			procdumpCmd(ctx)
			return nil
		},
	})

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
				sliver.ID, sliver.Name, sliver.RemoteAddress, sliver.Hostname, sliver.Os, sliver.Arch)
		case "disconnected":
			fmt.Printf(clearln+Warn+"Lost session #%d %s - %s (%s) - %s/%s\n",
				sliver.ID, sliver.Name, sliver.RemoteAddress, sliver.Hostname, sliver.Os, sliver.Arch)
			if activeSliver != nil && sliver.ID == activeSliver.ID {
				activeSliver = nil
				sliverApp.SetPrompt(getPrompt())
				fmt.Printf(Warn + "Warning: Active sliver diconnected\n")
			}
		}
		fmt.Printf(getPrompt())
		stdout.Flush()
	}
}

func getPrompt() string {
	prompt := underline + "sliver" + normal
	if activeSliver != nil {
		prompt += fmt.Sprintf(bold+red+" (%s)%s", activeSliver.Name, normal)
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
