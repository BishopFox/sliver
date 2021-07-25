package console

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	insecureRand "math/rand"
	"path"
	"strconv"

	"github.com/bishopfox/sliver/client/assets"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"gopkg.in/AlecAivazis/survey.v1"

	"time"

	"github.com/desertbit/go-shlex"
	"github.com/desertbit/grumble"
	"github.com/fatih/color"
)

const (
	// ANSI Colors
	Normal    = "\033[0m"
	Black     = "\033[30m"
	Red       = "\033[31m"
	Green     = "\033[32m"
	Orange    = "\033[33m"
	Blue      = "\033[34m"
	Purple    = "\033[35m"
	Cyan      = "\033[36m"
	Gray      = "\033[37m"
	Bold      = "\033[1m"
	Clearln   = "\r\x1b[2K"
	UpN       = "\033[%dA"
	DownN     = "\033[%dB"
	Underline = "\033[4m"

	// Info - Display colorful information
	Info = Bold + Cyan + "[*] " + Normal
	// Warn - Warn a user
	Warn = Bold + Red + "[!] " + Normal
	// Debug - Display debug information
	Debug = Bold + Purple + "[-] " + Normal
	// Woot - Display success
	Woot = Bold + Green + "[$] " + Normal
)

// Observer - A function to call when the sessions changes
type Observer func(*clientpb.Session)

type ActiveSession struct {
	Session    *clientpb.Session
	observers  map[int]Observer
	observerID int
}

type SliverConsoleClient struct {
	App           *grumble.App
	Rpc           rpcpb.SliverRPCClient
	ActiveSession *ActiveSession
	IsServer      bool
}

// BindCmds - Bind extra commands to the app object
type BindCmds func(console *SliverConsoleClient)

// Start - Console entrypoint
func Start(rpc rpcpb.SliverRPCClient, bindCmds BindCmds, extraCmds BindCmds, isServer bool) error {

	con := &SliverConsoleClient{
		App: grumble.New(&grumble.Config{
			Name:                  "Sliver",
			Description:           "Sliver Client",
			HistoryFile:           path.Join(assets.GetRootAppDir(), "history"),
			PromptColor:           color.New(),
			HelpHeadlineColor:     color.New(),
			HelpHeadlineUnderline: true,
			HelpSubCommands:       true,
		}),
		Rpc: rpc,
		ActiveSession: &ActiveSession{
			observers:  map[int]Observer{},
			observerID: 0,
		},
		IsServer: isServer,
	}
	con.App.SetPrintASCIILogo(func(_ *grumble.App) {
		con.PrintLogo()
	})
	con.App.SetPrompt(con.GetPrompt())
	bindCmds(con)
	extraCmds(con)

	con.ActiveSession.AddObserver(func(_ *clientpb.Session) {
		con.App.SetPrompt(con.GetPrompt())
	})

	go con.EventLoop()
	go core.TunnelLoop(rpc)

	err := con.App.Run()
	if err != nil {
		log.Printf("Run loop returned error: %v", err)
	}
	return err
}

func (con *SliverConsoleClient) EventLoop() {
	eventStream, err := con.Rpc.Events(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	for {
		event, err := eventStream.Recv()
		if err == io.EOF || event == nil {
			return
		}

		// Trigger event based on type
		switch event.EventType {

		case consts.CanaryEvent:
			con.Printf(Clearln+Warn+Bold+"WARNING: %s%s has been burned (DNS Canary)\n", Normal, event.Session.Name)
			sessions := con.GetSessionsByName(event.Session.Name)
			for _, session := range sessions {
				con.Printf(Clearln+"\t🔥 Session #%d is affected\n", session.ID)
			}
			con.Println()

		case consts.WatchtowerEvent:
			msg := string(event.Data)
			con.Printf(Clearln+Warn+Bold+"WARNING: %s%s has been burned (seen on %s)\n", Normal, event.Session.Name, msg)
			sessions := con.GetSessionsByName(event.Session.Name)
			for _, session := range sessions {
				con.PrintWarnf("\t🔥 Session #%d is affected\n", session.ID)
			}
			con.Println()

		case consts.JoinedEvent:
			con.PrintInfof("%s has joined the game\n\n", event.Client.Operator.Name)
		case consts.LeftEvent:
			con.PrintInfof("%s left the game\n\n", event.Client.Operator.Name)

		case consts.JobStoppedEvent:
			job := event.Job
			con.PrintWarnf("Job #%d stopped (%s/%s)\n\n", job.ID, job.Protocol, job.Name)

		case consts.SessionOpenedEvent:
			session := event.Session
			// The HTTP session handling is performed in two steps:
			// - first we add an "empty" session
			// - then we complete the session info when we receive the Register message from the Sliver
			// This check is here to avoid displaying two sessions events for the same session
			if session.OS != "" {
				currentTime := time.Now().Format(time.RFC1123)
				con.PrintInfof("Session #%d %s - %s (%s) - %s/%s - %v\n\n",
					session.ID, session.Name, session.RemoteAddress, session.Hostname, session.OS, session.Arch, currentTime)
			}

		case consts.SessionUpdateEvent:
			session := event.Session
			currentTime := time.Now().Format(time.RFC1123)
			con.Printf(Clearln+Info+"Session #%d has been updated - %v\n", session.ID, currentTime)

		case consts.SessionClosedEvent:
			session := event.Session
			con.Printf(Clearln+Warn+"Lost session #%d %s - %s (%s) - %s/%s\n",
				session.ID, session.Name, session.RemoteAddress, session.Hostname, session.OS, session.Arch)
			activeSession := con.ActiveSession.Get()
			if activeSession != nil && activeSession.ID == session.ID {
				con.ActiveSession.Set(nil)
				con.App.SetPrompt(con.GetPrompt())
				con.Printf(Warn + " Active session disconnected\n")
			}
			con.Println()
		}

		con.triggerReactions(event)

		con.Printf(Clearln + con.GetPrompt())
		bufio.NewWriter(con.App.Stdout()).Flush()
	}
}

func (con *SliverConsoleClient) triggerReactions(event *clientpb.Event) {
	reactions := core.Reactions.On(event.EventType)
	if len(reactions) == 0 {
		return
	}

	// We need some special handling for SessionOpenedEvent to
	// set the new session as the active session
	currentActiveSession := con.ActiveSession.Get()
	defer con.ActiveSession.Set(currentActiveSession)
	con.ActiveSession.Set(nil)
	if event.EventType == consts.SessionOpenedEvent {
		if event.Session == nil || event.Session.OS == "" {
			return // Half-open session, do not execute any command
		}
		con.ActiveSession.Set(event.Session)
	}

	for _, reaction := range reactions {
		for _, line := range reaction.Commands {
			con.PrintInfof("Execute reaction: '%s'\n", line)
			args, err := shlex.Split(line, true)
			if err != nil {
				con.PrintErrorf("Reaction command has invalid args: %s\n", err)
				continue
			}
			err = con.App.RunCommand(args)
			if err != nil {
				con.PrintErrorf("Reaction command error: %s\n", err)
			}
		}
	}
}

func (con *SliverConsoleClient) GetPrompt() string {
	prompt := Underline + "sliver" + Normal
	if con.IsServer {
		prompt = Bold + "[server] " + Normal + Underline + "sliver" + Normal
	}
	if con.ActiveSession.Get() != nil {
		prompt += fmt.Sprintf(Bold+Red+" (%s)%s", con.ActiveSession.Get().Name, Normal)
	}
	prompt += " > "
	return Clearln + prompt
}

func (con *SliverConsoleClient) PrintLogo() {
	serverVer, err := con.Rpc.GetVersion(context.Background(), &commonpb.Empty{})
	if err != nil {
		panic(err.Error())
	}
	dirty := ""
	if serverVer.Dirty {
		dirty = fmt.Sprintf(" - %sDirty%s", Bold, Normal)
	}
	serverSemVer := fmt.Sprintf("%d.%d.%d", serverVer.Major, serverVer.Minor, serverVer.Patch)

	insecureRand.Seed(time.Now().Unix())
	logo := asciiLogos[insecureRand.Intn(len(asciiLogos))]
	con.Println(logo)
	con.Println("All hackers gain " + abilities[insecureRand.Intn(len(abilities))])
	con.Printf(Info+"Server v%s - %s%s\n", serverSemVer, serverVer.Commit, dirty)
	if version.GitCommit != serverVer.Commit {
		con.Printf(Info+"Client %s\n", version.FullVersion())
	}
	con.Println(Info + "Welcome to the sliver shell, please type 'help' for options")
	con.Println()
	if serverVer.Major != int32(version.SemanticVersion()[0]) {
		con.Printf(Warn + "Warning: Client and server may be running incompatible versions.\n")
	}
	con.CheckLastUpdate()
}

func (con *SliverConsoleClient) CheckLastUpdate() {
	now := time.Now()
	lastUpdate := getLastUpdateCheck()
	compiledAt, err := version.Compiled()
	if err != nil {
		log.Printf("Failed to parse compiled at timestamp %s", err)
		return
	}

	day := 24 * time.Hour
	if compiledAt.Add(30 * day).Before(now) {
		if lastUpdate == nil || lastUpdate.Add(30*day).Before(now) {
			con.Printf(Info + "Check for updates with the 'update' command\n\n")
		}
	}
}

func getLastUpdateCheck() *time.Time {
	appDir := assets.GetRootAppDir()
	lastUpdateCheckPath := path.Join(appDir, consts.LastUpdateCheckFileName)
	data, err := ioutil.ReadFile(lastUpdateCheckPath)
	if err != nil {
		log.Printf("Failed to read last update check %s", err)
		return nil
	}
	unixTime, err := strconv.Atoi(string(data))
	if err != nil {
		log.Printf("Failed to parse last update check %s", err)
		return nil
	}
	lastUpdate := time.Unix(int64(unixTime), 0)
	return &lastUpdate
}

// GetSession - Get session by session ID or name
func (con *SliverConsoleClient) GetSession(arg string) *clientpb.Session {
	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return nil
	}
	for _, session := range sessions.GetSessions() {
		if session.Name == arg || fmt.Sprintf("%d", session.ID) == arg {
			return session
		}
	}
	return nil
}

// GetSessionsByName - Return all sessions for an Implant by name
func (con *SliverConsoleClient) GetSessionsByName(name string) []*clientpb.Session {
	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return nil
	}
	matched := []*clientpb.Session{}
	for _, session := range sessions.GetSessions() {
		if session.Name == name {
			matched = append(matched, session)
		}
	}
	return matched
}

// GetActiveSessionConfig - Get the active sessions's config
func (con *SliverConsoleClient) GetActiveSessionConfig() *clientpb.ImplantConfig {
	session := con.ActiveSession.Get()
	if session == nil {
		return nil
	}
	c2s := []*clientpb.ImplantC2{}
	c2s = append(c2s, &clientpb.ImplantC2{
		URL:      session.GetActiveC2(),
		Priority: uint32(0),
	})
	config := &clientpb.ImplantConfig{
		Name:    session.GetName(),
		GOOS:    session.GetOS(),
		GOARCH:  session.GetArch(),
		Debug:   true,
		Evasion: session.GetEvasion(),

		MaxConnectionErrors: uint32(1000),
		ReconnectInterval:   uint32(60),
		PollInterval:        uint32(1),

		Format:      clientpb.OutputFormat_SHELLCODE,
		IsSharedLib: true,
		C2:          c2s,
	}
	return config
}

// This should be called for any dangerous (OPSEC-wise) functions
func (con *SliverConsoleClient) IsUserAnAdult() bool {
	confirm := false
	prompt := &survey.Confirm{Message: "This action is bad OPSEC, are you an adult?"}
	survey.AskOne(prompt, &confirm, nil)
	return confirm
}

func (con *SliverConsoleClient) Printf(format string, args ...interface{}) (n int, err error) {
	return fmt.Fprintf(con.App.Stdout(), format, args...)
}

func (con *SliverConsoleClient) Println(args ...interface{}) (n int, err error) {
	return fmt.Fprintln(con.App.Stdout(), args...)
}

func (con *SliverConsoleClient) PrintInfof(format string, args ...interface{}) (n int, err error) {
	return fmt.Fprintf(con.App.Stdout(), Clearln+Info+format, args...)
}

func (con *SliverConsoleClient) PrintWarnf(format string, args ...interface{}) (n int, err error) {
	return fmt.Fprintf(con.App.Stdout(), Clearln+Warn+format, args...)
}

func (con *SliverConsoleClient) PrintErrorf(format string, args ...interface{}) (n int, err error) {
	return fmt.Fprintf(con.App.Stderr(), Clearln+Warn+format, args...)
}

func (con *SliverConsoleClient) SpinUntil(message string, ctrl chan bool) {
	go spin.Until(con.App.Stdout(), message, ctrl)
}

//
// -------------------------- [ Active Session ] --------------------------
//

// GetInteractive - GetInteractive the active session
func (s *ActiveSession) GetInteractive() *clientpb.Session {
	if s.Session == nil {
		fmt.Printf(Warn + "Please select an active session via `use`\n")
		return nil
	}
	return s.Session
}

// Get - Same as Get() but doesn't print a warning
func (s *ActiveSession) Get() *clientpb.Session {
	if s.Session == nil {
		return nil
	}
	return s.Session
}

// AddObserver - Observers to notify when the active session changes
func (s *ActiveSession) AddObserver(observer Observer) int {
	s.observerID++
	s.observers[s.observerID] = observer
	return s.observerID
}

func (s *ActiveSession) RemoveObserver(observerID int) {
	delete(s.observers, observerID)
}

func (s *ActiveSession) Request(ctx *grumble.Context) *commonpb.Request {
	if s.Session == nil {
		return nil
	}
	timeout := int(time.Second) * ctx.Flags.Int("timeout")
	return &commonpb.Request{
		SessionID: s.Session.ID,
		Timeout:   int64(timeout),
	}
}

// Set - Change the active session
func (s *ActiveSession) Set(session *clientpb.Session) {
	s.Session = session
	for _, observer := range s.observers {
		observer(s.Session)
	}
}

// Background - Background the active session
func (s *ActiveSession) Background() {
	s.Session = nil
	for _, observer := range s.observers {
		observer(nil)
	}
}

var abilities = []string{
	"first strike",
	"vigilance",
	"haste",
	"indestructible",
	"hexproof",
	"deathtouch",
	"fear",
	"epic",
	"ninjitsu",
	"recover",
	"persist",
	"conspire",
	"reinforce",
	"exalted",
	"annihilator",
	"infect",
	"undying",
	"living weapon",
	"miracle",
	"scavenge",
	"cipher",
	"evolve",
	"dethrone",
	"hidden agenda",
	"prowess",
	"dash",
	"exploit",
	"renown",
	"skulk",
	"improvise",
	"assist",
	"jump-start",
}

var asciiLogos = []string{
	Red + `
 	  ██████  ██▓     ██▓ ██▒   █▓▓█████  ██▀███
	▒██    ▒ ▓██▒    ▓██▒▓██░   █▒▓█   ▀ ▓██ ▒ ██▒
	░ ▓██▄   ▒██░    ▒██▒ ▓██  █▒░▒███   ▓██ ░▄█ ▒
	  ▒   ██▒▒██░    ░██░  ▒██ █░░▒▓█  ▄ ▒██▀▀█▄
	▒██████▒▒░██████▒░██░   ▒▀█░  ░▒████▒░██▓ ▒██▒
	▒ ▒▓▒ ▒ ░░ ▒░▓  ░░▓     ░ ▐░  ░░ ▒░ ░░ ▒▓ ░▒▓░
	░ ░▒  ░ ░░ ░ ▒  ░ ▒ ░   ░ ░░   ░ ░  ░  ░▒ ░ ▒░
	░  ░  ░    ░ ░    ▒ ░     ░░     ░     ░░   ░
		  ░      ░  ░ ░        ░     ░  ░   ░
` + Normal,

	Green + `
    ███████╗██╗     ██╗██╗   ██╗███████╗██████╗
    ██╔════╝██║     ██║██║   ██║██╔════╝██╔══██╗
    ███████╗██║     ██║██║   ██║█████╗  ██████╔╝
    ╚════██║██║     ██║╚██╗ ██╔╝██╔══╝  ██╔══██╗
    ███████║███████╗██║ ╚████╔╝ ███████╗██║  ██║
    ╚══════╝╚══════╝╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝
` + Normal,

	Bold + Gray + `
.------..------..------..------..------..------.
|S.--. ||L.--. ||I.--. ||V.--. ||E.--. ||R.--. |
| :/\: || :/\: || (\/) || :(): || (\/) || :(): |
| :\/: || (__) || :\/: || ()() || :\/: || ()() |
| '--'S|| '--'L|| '--'I|| '--'V|| '--'E|| '--'R|
` + "`------'`------'`------'`------'`------'`------'" + `
` + Normal,
}
