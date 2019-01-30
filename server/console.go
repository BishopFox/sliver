package main

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
	pb "sliver/protobuf"
	"strconv"
	"strings"
	"time"

	"github.com/desertbit/grumble"
	"github.com/fatih/color"
)

const (
	helpStr = "help"

	sessionsStr   = "sessions"
	backgroundStr = "background"
	infoStr       = "info"
	useStr        = "use"
	generateStr   = "generate"

	jobsStr = "jobs"
	mtlsStr = "mtls"
	dnsStr  = "dns"

	msfStr    = "msf"
	injectStr = "inject"

	psStr   = "ps"
	pingStr = "ping"
	killStr = "kill"

	getPIDStr = "getpid"
	getUIDStr = "getuid"
	getGIDStr = "getgid"
	whoamiStr = "whoami"

	lsStr       = "ls"
	rmStr       = "rm"
	mkdirStr    = "mkdir"
	cdStr       = "cd"
	pwdStr      = "pwd"
	catStr      = "cat"
	downloadStr = "download"
	uploadStr   = "upload"
	procdumpStr = "procdump"
)

var (
	activeSliver *Sliver

	events = make(chan Event, 64)

	cmdTimeout = 10 * time.Second
)

func startConsole() {

	sliverApp := grumble.New(&grumble.Config{
		Name:                  "sliver",
		Description:           "Bishop Fox - Sliver",
		HistoryFile:           path.Join(GetRootAppDir(), "history"),
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
		jobMutex.Lock()
		for ID, job := range *jobs {
			job.JobCtrl <- true
			delete(*jobs, ID)
		}
		jobMutex.Unlock()

		// Cleanup sliver connections
		for _, sliver := range *hive {
			hiveMutex.Lock()
			if _, ok := (*hive)[sliver.ID]; ok {
				delete(*hive, sliver.ID)
				close(sliver.Send)
			}
			hiveMutex.Unlock()
		}

		close(events) // Cleanup eventLoop()
	}()

	go eventLoop(sliverApp, events)

	err := sliverApp.Run()
	if err != nil {
		log.Printf("Run loop returned error: %v", err)
	}
}

func eventLoop(sliverApp *grumble.App, events chan Event) {
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

	sliverApp.AddCommand(&grumble.Command{
		Name:      helpStr,
		Help:      getHelpFor(helpStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			helpCmd(ctx)
			return nil
		},
	})

	// [ Jobs ] -----------------------------------------------------------------

	sliverApp.AddCommand(&grumble.Command{
		Name: jobsStr,
		Help: getHelpFor(jobsStr),
		Flags: func(f *grumble.Flags) {
			f.Int("k", "kill", -1, "kill a background job")
		},
		Run: func(ctx *grumble.Context) error {
			jobsCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name: mtlsStr,
		Help: getHelpFor(mtlsStr),
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
		Name: dnsStr,
		Help: getHelpFor(dnsStr),
		Flags: func(f *grumble.Flags) {
			f.String("d", "domain", "", "parent domain to use for DNS C2")
		},
		Run: func(ctx *grumble.Context) error {
			startDNSListenerCmd(ctx)
			return nil
		},
	})

	// [ Commands ] --------------------------------------------------------------

	sliverApp.AddCommand(&grumble.Command{
		Name: sessionsStr,
		Help: getHelpFor(sessionsStr),
		Flags: func(f *grumble.Flags) {
			f.String("i", "interact", "", "interact with a sliver")
		},
		Run: func(ctx *grumble.Context) error {
			sessionsCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name: backgroundStr,
		Help: getHelpFor(backgroundStr),
		Run: func(ctx *grumble.Context) error {
			backgroundCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      killStr,
		Help:      getHelpFor(killStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			killCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      infoStr,
		Help:      getHelpFor(infoStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			infoCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      useStr,
		Help:      getHelpFor(useStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			useCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name: generateStr,
		Help: getHelpFor(generateStr),
		Flags: func(f *grumble.Flags) {
			f.String("o", "os", WINDOWS, "operating system")
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
		Name: msfStr,
		Help: getHelpFor(msfStr),
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
		Name: injectStr,
		Help: getHelpFor(injectStr),
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
		Name: psStr,
		Help: getHelpFor(psStr),
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
		Name:      pingStr,
		Help:      getHelpFor(pingStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			pingCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name: getPIDStr,
		Help: getHelpFor(getPIDStr),
		Run: func(ctx *grumble.Context) error {
			if activeSliver != nil {
				fmt.Printf("\n"+Info+"%d\n\n", activeSliver.PID)
			}
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name: getUIDStr,
		Help: getHelpFor(getUIDStr),
		Run: func(ctx *grumble.Context) error {
			if activeSliver != nil {
				fmt.Printf("\n"+Info+"%s\n\n", activeSliver.UID)
			}
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name: getGIDStr,
		Help: getHelpFor(getGIDStr),
		Run: func(ctx *grumble.Context) error {
			if activeSliver != nil {
				fmt.Printf("\n"+Info+"%s\n\n", activeSliver.GID)
			}
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name: whoamiStr,
		Help: getHelpFor(whoamiStr),
		Run: func(ctx *grumble.Context) error {
			if activeSliver != nil {
				fmt.Printf("\n"+Info+"%s\n\n", activeSliver.Username)
			}
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name: lsStr,
		Help: getHelpFor(lsStr),
		Run: func(ctx *grumble.Context) error {
			lsCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      rmStr,
		Help:      getHelpFor(rmStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			rmCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      mkdirStr,
		Help:      getHelpFor(mkdirStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			mkdirCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      cdStr,
		Help:      getHelpFor(cdStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			cdCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name: pwdStr,
		Help: getHelpFor(pwdStr),
		Run: func(ctx *grumble.Context) error {
			pwdCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      catStr,
		Help:      getHelpFor(catStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			catCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      downloadStr,
		Help:      getHelpFor(downloadStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			downloadCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name:      uploadStr,
		Help:      getHelpFor(uploadStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			uploadCmd(ctx)
			return nil
		},
	})

	sliverApp.AddCommand(&grumble.Command{
		Name: procdumpStr,
		Help: getHelpFor(procdumpStr),
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

func getSliver(name string) *Sliver {
	id, err := strconv.Atoi(name)
	name = strings.ToUpper(name)
	hiveMutex.Lock()
	defer hiveMutex.Unlock()
	if err == nil {
		if sliver, ok := (*hive)[id]; ok {
			return sliver
		}
	}
	for _, sliver := range *hive {
		if sliver.Name == name {
			return sliver
		}
	}
	return nil
}

// Sends a protobuf request to the active sliver and returns the response
func activeSliverRequest(msgType string, reqID string, data []byte) (pb.Envelope, error) {
	if activeSliver == nil {
		return pb.Envelope{}, errors.New("No active sliver")
	}
	resp := make(chan pb.Envelope)
	(*activeSliver).Resp[reqID] = resp
	defer func() {
		activeSliver.RespMutex.Lock()
		defer activeSliver.RespMutex.Unlock()
		close(resp)
		delete((*activeSliver).Resp, reqID)
	}()
	(*activeSliver).Send <- pb.Envelope{
		Id:   reqID,
		Type: msgType,
		Data: data,
	}

	var respEnvelope pb.Envelope
	select {
	case respEnvelope = <-resp:
	case <-time.After(cmdTimeout):
		return pb.Envelope{}, errors.New("timeout")
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
