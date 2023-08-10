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
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/bishopfox/sliver/client/assets"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/reeflective/console"
	"github.com/reeflective/readline"
	"github.com/reeflective/team/client"
)

const (
	defaultTimeout = 60
)

const (
	// ANSI Colors.
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

	// Info - Display colorful information.
	Info = Bold + Cyan + "[*] " + Normal
	// Warn - Warn a user.
	Warn = Bold + Red + "[!] " + Normal
	// Debug - Display debug information.
	Debug = Bold + Purple + "[-] " + Normal
	// Woot - Display success.
	Woot = Bold + Green + "[$] " + Normal
	// Success - Diplay success.
	Success = Bold + Green + "[+] " + Normal
)

// Observer - A function to call when the sessions changes.
type (
	Observer           func(*clientpb.Session, *clientpb.Beacon)
	BeaconTaskCallback func(*clientpb.BeaconTask)
)

// SliverClient is a general-purpose, interface-agnostic client.
//
// It allows to use the Sliver toolset with an arbitrary number of remote/local
// Sliver servers, either through its API (RPC client), CLI (the command tree),
// or through an arbitrary mix of those.
//
// The Sliver client will by default be used with remote, mutual TLS authenticated
// connections to Sliver teamservers, which configurations are found in the teamclient
// directory.
// However, the teamclient API/CLI offers ways to manage and use those configurations,
// which means that users of this Sliver client may build arbitrarily complex server
// selection/connection strategies.
type SliverClient struct {
	// Core client
	App      *console.Console
	Settings *assets.ClientSettings
	IsServer bool
	isCLI    bool

	// Teamclient & remotes
	Teamclient   *client.Client
	dialer       *transport.TeamClient
	connectHooks []func() error

	// Logging
	jsonHandler slog.Handler
	printf      func(format string, args ...any) (int, error)
	closeLogs   []func()

	// Sliver-specific
	Rpc                      rpcpb.SliverRPCClient
	ActiveTarget             *ActiveTarget
	EventListeners           *sync.Map
	BeaconTaskCallbacks      map[string]BeaconTaskCallback
	BeaconTaskCallbacksMutex *sync.Mutex
}

// NewSliverClient is the general-purpose Sliver Client constructor.
//
// The returned client includes and is ready to use the following:
//   - A reeflective/team.Client to manage, use and interact with an arbitrary
//     number of Sliver teamservers. This includes connecting, registering RPC
//     client interfaces, logging, authenticating and disconnecting.
//   - A console application, which can either be used closed-loop, or in a classic
//     exec-once CLI style. Users of this client are free to use either at will.
//   - Cobra-command runner methods to be included in new commands and completers.
//   - Methods to set and interact with a Sliver implant target.
//   - Various logging/streaming utilities.
//
// Any error returned from this call is critical, meaning that given the current
// options (teamclient, gRPC, etc), the SliverClient is not able to work properly.
func NewSliverClient(opts ...grpc.DialOption) (con *SliverClient, err error) {
	// Create the client core, everything interface-related.
	con = newClient()

	// Our reeflective/team.Client needs our gRPC stack.
	con.dialer = transport.NewClient(opts...)

	var clientOpts []client.Options
	clientOpts = append(clientOpts,
		client.WithHomeDirectory(assets.GetRootAppDir()),
		client.WithDialer(con.dialer),
	)

	// Create a new reeflective/team.Client, which is in charge of selecting,
	// and connecting with, remote Sliver teamserver configurations, etc.
	// Includes client backend logging, authentication, core teamclient methods...
	con.Teamclient, err = client.New("sliver", con, clientOpts...)
	if err != nil {
		return nil, err
	}

	return con, nil
}

// newClient creates the sliver client (and console), creating menus and prompts.
// The returned console does neither have commands nor a working RPC connection yet,
// thus has not started monitoring any server events, or started the application.
func newClient() *SliverClient {
	assets.Setup(false, false)
	settings, _ := assets.LoadSettings()

	con := &SliverClient{
		App:                      console.New("sliver"),
		Settings:                 settings,
		isCLI:                    true,
		printf:                   fmt.Printf,
		ActiveTarget:             newActiveTarget(),
		EventListeners:           &sync.Map{},
		BeaconTaskCallbacks:      map[string]BeaconTaskCallback{},
		BeaconTaskCallbacksMutex: &sync.Mutex{},
	}

	con.App.SetPrintLogo(func(_ *console.Console) {
		con.PrintLogo()
	})

	// Readline-shell (edition) settings
	if settings.VimMode {
		con.App.Shell().Config.Set("editing-mode", "vi")
	}

	// Global console settings
	con.App.NewlineBefore = true
	con.App.NewlineAfter = true

	// Server menu.
	server := con.App.Menu(consts.ServerMenu)
	server.Prompt().Primary = con.GetPrompt
	server.AddInterrupt(readline.ErrInterrupt, con.exitConsole) // Ctrl-C

	histPath := filepath.Join(assets.GetRootAppDir(), "history")
	server.AddHistorySourceFile("server history", histPath)

	// Implant menu.
	sliver := con.App.NewMenu(consts.ImplantMenu)
	sliver.Prompt().Primary = con.GetPrompt
	sliver.AddInterrupt(io.EOF, con.exitImplantMenu) // Ctrl-D

	// The active target needs access to the console
	// to automatically switch between command menus.
	con.ActiveTarget.con = con

	return con
}

// StartConsole is a blocking call that starts the Sliver closed console.
// The command/events/log outputs use the specific-console fmt.Printer,
// because the console needs to query the terminal for cursor positions
// when asynchronously printing logs (that is, when no command is running).
func (con *SliverClient) StartConsole() error {
	con.isCLI = false
	con.printf = con.App.TransientPrintf

	return con.App.Start()
}

// Expose or hide commands if the active target does support them (or not).
// Ex; hide Windows commands on Linux implants, Wireguard tools on HTTP C2, etc.
func (con *SliverClient) ExposeCommands() {
	con.App.ShowCommands()

	filters := con.ActiveTarget.Filters()

	// Use all defined filters.
	con.App.HideCommands(filters...)
}

func (con *SliverClient) GetSession(arg string) *clientpb.Session {
	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintWarnf("%s", err)
		return nil
	}
	for _, session := range sessions.GetSessions() {
		if session.Name == arg || strings.HasPrefix(session.ID, arg) {
			return session
		}
	}
	return nil
}

func (con *SliverClient) GetBeacon(arg string) *clientpb.Beacon {
	beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintWarnf("%s", err)
		return nil
	}
	for _, session := range beacons.GetBeacons() {
		if session.Name == arg || strings.HasPrefix(session.ID, arg) {
			return session
		}
	}
	return nil
}

// GetSessionsByName - Return all sessions for an Implant by name.
func (con *SliverClient) GetSessionsByName(name string) []*clientpb.Session {
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
// TODO: Switch to query config based on ConfigID.
func (con *SliverClient) GetActiveSessionConfig() *clientpb.ImplantConfig {
	session := con.ActiveTarget.GetSession()
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
		ReconnectInterval:   int64(60),
		Format:              clientpb.OutputFormat_SHELLCODE,
		IsSharedLib:         true,
		C2:                  c2s,
	}
	return config
}

func (con *SliverClient) SpinUntil(message string, ctrl chan bool) {
	go spin.Until(os.Stdout, message, ctrl)
}

// WaitSignal listens for os.Signals and returns when receiving one of the following:
// SIGINT, SIGTERM, SIGQUIT.
//
// This can be used for commands which should block if executed in an exec-once CLI run:
// if the command is ran in the closed-loop console, this function will not monitor signals
// and return immediately.
func (con *SliverClient) WaitSignal() error {
	if !con.isCLI {
		return nil
	}

	sigchan := make(chan os.Signal, 1)

	signal.Notify(
		sigchan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		// syscall.SIGKILL,
	)

	sig := <-sigchan
	con.PrintInfof("Received signal %s\n", sig)

	return nil
}

// FormatDateDelta - Generate formatted date string of the time delta between then and now.
func (con *SliverClient) FormatDateDelta(t time.Time, includeDate bool, color bool) string {
	nextTime := t.Format(time.UnixDate)

	var interval string

	if t.Before(time.Now()) {
		if includeDate {
			interval = fmt.Sprintf("%s (%s ago)", nextTime, time.Since(t).Round(time.Second))
		} else {
			interval = time.Since(t).Round(time.Second).String()
		}
		if color {
			interval = fmt.Sprintf("%s%s%s", Bold+Red, interval, Normal)
		}
	} else {
		if includeDate {
			interval = fmt.Sprintf("%s (in %s)", nextTime, time.Until(t).Round(time.Second))
		} else {
			interval = time.Until(t).Round(time.Second).String()
		}
		if color {
			interval = fmt.Sprintf("%s%s%s", Bold+Green, interval, Normal)
		}
	}
	return interval
}

// GrpcContext - Generate a context for a GRPC request, if no grumble context or an invalid flag is provided 60 seconds is used instead.
func (con *SliverClient) GrpcContext(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	if cmd == nil {
		return context.WithTimeout(context.Background(), 60*time.Second)
	}

	timeOutF, _ := cmd.Flags().GetInt64("timeout")
	timeout := time.Duration(int64(time.Second) * timeOutF)
	if timeout < 1 {
		timeout = 60 * time.Second
	}
	return context.WithTimeout(context.Background(), timeout)
}

// UnwrapServerErr unwraps errors returned by gRPC method calls.
// Should be used to return every non-nil resp, err := con.Rpc.Function().
func (con *SliverClient) UnwrapServerErr(err error) error {
	return errors.New(status.Convert(err).Message())
}

func (con *SliverClient) CheckLastUpdate() {
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
	lastUpdateCheckPath := filepath.Join(appDir, consts.LastUpdateCheckFileName)
	data, err := os.ReadFile(lastUpdateCheckPath)
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

func compCommandCalled(cmd *cobra.Command) bool {
	for _, compCmd := range cmd.Root().Commands() {
		if compCmd != nil && compCmd.Name() == "_carapace" && compCmd.CalledAs() != "" {
			return true
		}
	}

	return false
}
