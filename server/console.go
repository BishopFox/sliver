package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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

	console := make(chan ConsoleMsg)
	go consolePrinter(console)

	defer close(events)
	go func() {
		for sliver := range events {
			console <- ConsoleMsg{
				Level: Info,
				Message: fmt.Sprintf("%s - %s (%s) - %s/%s",
					sliver.Name, sliver.RemoteAddress, sliver.Hostname, sliver.Os, sliver.Arch),
			}
		}
	}()

	console <- ConsoleMsg{
		Level:   Info,
		Message: "Welcome to the Sliver shell, please type 'help' for options",
	}

	consoleReader := make(chan string)
	go commandLoop(console, consoleReader)

	reader := bufio.NewReader(os.Stdin)
	buf := make([]byte, 1)
	reader.Read(buf)

}

func consolePrinter(console chan ConsoleMsg) {
	for msg := range console {
		fmt.Printf(clearln+"%s%s", msg.Level, msg.Message)
		if msg.Level != Empty {
			fmt.Printf("\n")
		}
		prompt()
	}
}

func print(console chan ConsoleMsg, msg string) {
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	console <- ConsoleMsg{Level: Empty, Message: msg}
}

func prompt() {
	fmt.Printf(clearln + underline + "sliver" + normal)
	if activeSliver != nil {
		fmt.Printf(bold+red+"(%s)%s", activeSliver.Name, normal)
	}
	fmt.Printf(normal + " > ")
	stdout.Flush()
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

func commandLoop(console chan ConsoleMsg, consoleReader chan string) {
	defer close(console)
	for line := range consoleReader {
		line = strings.TrimSpace(line)
		words := strings.Fields(line)
		if len(words) == 0 {
			console <- ConsoleMsg{Level: Empty, Message: Empty}
			continue
		}
		if words[0] == "exit" {
			console <- ConsoleMsg{Level: Info, Message: "User exit"}
			return
		}
		if cmd, ok := cmdHandlers[words[0]]; ok {
			go cmd.(func(chan ConsoleMsg, []string))(console, words[1:])
		} else {
			console <- ConsoleMsg{
				Level:   Warn,
				Message: fmt.Sprintf("Invalid command '%s'", words[0]),
			}
		}
	}
}

func help(console chan ConsoleMsg, args []string) {
	console <- ConsoleMsg{Level: Empty, Message: fmt.Sprintf(`
%sSliver Commands%s
==============
help        - Display this help message
use <name>  - Use a sliver 
info <name> - Display informationa about a sliver

`, bold, normal)}
}

func info(console chan ConsoleMsg, args []string) {
	if len(args) == 1 {
		sliver := getSliverByName(args[0])
		if sliver != nil {
			print(console, "\n")
			print(console, fmt.Sprintf(bold+"ID: %s%s", normal, sliver.ID))
			print(console, fmt.Sprintf(bold+"Name: %s%s", normal, sliver.Name))
			print(console, fmt.Sprintf(bold+"Hostname: %s%s", normal, sliver.Hostname))
			print(console, fmt.Sprintf(bold+"Username: %s%s", normal, sliver.Username))
			print(console, fmt.Sprintf(bold+"UID: %s%s", normal, sliver.Uid))
			print(console, fmt.Sprintf(bold+"GID: %s%s", normal, sliver.Gid))
			print(console, fmt.Sprintf(bold+"OS: %s%s", normal, sliver.Os))
			print(console, fmt.Sprintf(bold+"Arch: %s%s", normal, sliver.Arch))
			print(console, fmt.Sprintf(bold+"Remote Address: %s%s", normal, sliver.RemoteAddress))
			print(console, "\n")
		} else {
			console <- ConsoleMsg{
				Level:   Warn,
				Message: fmt.Sprintf("No sliver with name '%s'", args[0]),
			}
		}
	} else {
		console <- ConsoleMsg{Level: Warn, Message: "Please provide a sliver name"}
	}
}
