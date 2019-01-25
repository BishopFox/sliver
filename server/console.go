package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	pb "sliver/protobuf"
	"strconv"
	"strings"
	"time"

	"github.com/desertbit/grumble"
	"github.com/fatih/color"
)

var (
	activeSliver *Sliver

	cmdTimeout = 10 * time.Second

	// Stylizes known processes in the `ps` command
	knownProcs = map[string]string{
		"ccSvcHst.exe": red, // SEP
		"cb.exe":       red, // Carbon Black
	}
)

func startConsole(events chan Event) {

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

	defer close(events)
	go eventLoop(sliverApp, events)

	sliverApp.Run()
}

func eventLoop(sliverApp *grumble.App, events chan Event) {
	for event := range events {
		sliver := event.Sliver
		switch event.EventType {
		case "connected":
			fmt.Printf(clearln+Info+"Session #%d %s - %s (%s) - %s/%s\n",
				sliver.Id, sliver.Name, sliver.RemoteAddress, sliver.Hostname, sliver.Os, sliver.Arch)
		case "disconnected":
			fmt.Printf(clearln+Warn+"Lost session #%d %s - %s (%s) - %s/%s\n",
				sliver.Id, sliver.Name, sliver.RemoteAddress, sliver.Hostname, sliver.Os, sliver.Arch)
			if activeSliver != nil && sliver.Id == activeSliver.Id {
				activeSliver = nil
				sliverApp.SetPrompt(getPrompt())
				fmt.Printf(Warn + "Warning: Active sliver diconnected\n")
			}
		}
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
			f.String("m", "payload", "", "msf payload")
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
			f.String("m", "payload", "", "msf payload")
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
			f.Int("p", "pid", -1, "pid to inject into")
			f.String("x", "exe", "", "executable name")
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
		Name: lsStr,
		Help: getHelpFor(lsStr),
		Run: func(ctx *grumble.Context) error {
			lsCmd(ctx)
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

func activeSliverRequest(msgType string, reqId string, data []byte) (pb.Envelope, error) {
	if activeSliver == nil {
		return pb.Envelope{}, errors.New("No active sliver")
	}
	resp := make(chan pb.Envelope)
	(*activeSliver).Resp[reqId] = resp
	defer close(resp)
	defer delete((*activeSliver).Resp, reqId)
	(*activeSliver).Send <- pb.Envelope{
		Id:   reqId,
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
	fmt.Println()
	fmt.Println(Info + "Welcome to the sliver shell, please type 'help' for options")
	fmt.Println()
}
