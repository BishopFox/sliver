package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

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
		"help": help,
		"info": info,
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

	reader := make(chan string)
	go lineReader(term, reader)
	for {
		select {
		case line := <-reader:
			fmt.Fprintln(term, "", line)
			if line == "exit" {
				return
			}
			words := strings.Fields(line)
			if cmd, ok := cmdHandlers[words[0]]; ok {
				go cmd.(func(*terminal.Terminal, []string))(term, words[1:])
			} else {
				msg := fmt.Sprintf(Warn+"Invalid command '%s'", words[0])
				fmt.Fprintln(term, "", msg)
			}
		case sliver := <-events:
			msg := fmt.Sprintf(Info+"New connection: %s", sliver.Name)
			fmt.Fprintln(term, "", msg)
		}
	}
}

func lineReader(term *terminal.Terminal, reader chan string) {
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
		}
	}
}

func setPrompt(term *terminal.Terminal) {
	prompt := fmt.Sprintf(clearln + underline + "sliver" + normal)
	if activeSliver != nil {
		prompt += fmt.Sprintf(bold+red+"(%s)%s", activeSliver.Name, normal)
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
func help() {

}

func info() {

}
