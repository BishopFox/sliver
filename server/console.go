package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	pb "sliver/protobuf"
	"strconv"
	"strings"
	"time"

	"github.com/chzyer/readline"
)

const (
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
	activeSliver *Sliver
	cmdTimeout   = 10 * time.Second

	// Stylizes known processes in the `ps` command
	knownProcs = map[string]string{
		"ccSvcHst.exe": red, // SEP
		"cb.exe":       red, // Carbon Black
	}

	cmdHandlers = map[string]interface{}{
		"help":       helpCmd,
		"sessions":   sessionsCmd,
		"background": backgroundCmd,
		"info":       infoCmd,
		"use":        useCmd,
		"generate":   generateCmd,
		"msf":        msfCmd,
		"inject":     injectCmd,
		"ps":         psCmd,
		"ping":       pingCmd,
		"kill":       killCmd,

		"ls":       lsCmd,
		"cd":       cdCmd,
		"pwd":      pwdCmd,
		"cat":      catCmd,
		"download": downloadCmd,
		"upload":   uploadCmd,
	}
)

var completer = readline.NewPrefixCompleter(
	readline.PcItem("help",
		readline.PcItem("sessions"),
		readline.PcItem("background"),
		readline.PcItem("info"),
		readline.PcItem("use"),
		readline.PcItem("generate"),
		readline.PcItem("msf"),
		readline.PcItem("inject"),
		readline.PcItem("ps"),
		readline.PcItem("ping"),
		readline.PcItem("kill"),
		readline.PcItem("ls"),
		readline.PcItem("cd"),
		readline.PcItem("pwd"),
		readline.PcItem("cat"),
		readline.PcItem("download"),
		readline.PcItem("upload"),
	),
	readline.PcItem("sessions",
		readline.PcItem("-i"),
	),
	readline.PcItem("background"),
	readline.PcItem("info"),
	readline.PcItem("use"),
	readline.PcItem("generate",
		readline.PcItem("-save"),
		readline.PcItem("-lhost"),
		readline.PcItem("-lport"),
		readline.PcItem("-os"),
		readline.PcItem("-arch"),
		readline.PcItem("-debug"),
	),
	readline.PcItem("msf",
		readline.PcItem("-payload"),
		readline.PcItem("-lhost"),
		readline.PcItem("-lport"),
		readline.PcItem("-encoder"),
		readline.PcItem("-iterations"),
	),
	readline.PcItem("inject",
		readline.PcItem("-pid"),
		readline.PcItem("-payload"),
		readline.PcItem("-lhost"),
		readline.PcItem("-lport"),
		readline.PcItem("-encoder"),
		readline.PcItem("-iterations"),
	),
	readline.PcItem("ps"),
	readline.PcItem("ping"),
	readline.PcItem("kill"),
	readline.PcItem("ls"),
	readline.PcItem("cd"),
	readline.PcItem("pwd"),
	readline.PcItem("cat"),
	readline.PcItem("download"),
	readline.PcItem("upload"),
)

func startConsole(events chan Event) {
	term, err := readline.NewEx(&readline.Config{
		Prompt:          getPrompt(),
		HistoryFile:     path.Join(GetRootAppDir(), "history"),
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	})
	if err != nil {
		panic(err)
	}
	defer term.Close()
	defer close(events)

	go eventLoop(term, events)

	for {
		line, err := term.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}
		line = strings.TrimSpace(line)
		if line == "exit" {
			return
		}
		words := strings.Fields(line)
		if len(words) < 1 {
			continue
		}
		if cmd, ok := cmdHandlers[words[0]]; ok {
			cmd.(func(*readline.Instance, []string))(term, words[1:])
		} else {
			fmt.Fprintf(term, "\n"+Warn+"Invalid command '%s'\n", words[0])
		}

	}

}

func eventLoop(term *readline.Instance, events chan Event) {
	for event := range events {
		sliver := event.Sliver
		switch event.EventType {
		case "connected":
			fmt.Fprintf(term, Info+"Session #%d %s - %s (%s) - %s/%s\n",
				sliver.Id, sliver.Name, sliver.RemoteAddress, sliver.Hostname, sliver.Os, sliver.Arch)
		case "disconnected":
			fmt.Fprintf(term, Warn+"Lost session #%d %s - %s (%s) - %s/%s\n",
				sliver.Id, sliver.Name, sliver.RemoteAddress, sliver.Hostname, sliver.Os, sliver.Arch)
			if activeSliver != nil && sliver.Id == activeSliver.Id {
				activeSliver = nil
				term.SetPrompt(getPrompt())
				term.Refresh()
				fmt.Fprintf(term, Warn+"Warning: Active sliver diconnected\n")
			}
		}
	}
}

func filterInput(r rune) (rune, bool) {
	switch r {
	// block CtrlZ feature
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}

func localFileList(path string) func(string) []string {
	return func(line string) []string {
		names := make([]string, 0)
		files, _ := ioutil.ReadDir(path)
		for _, f := range files {
			names = append(names, f.Name())
		}
		return names
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
