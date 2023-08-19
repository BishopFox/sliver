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
	"fmt"
	"log"
	insecureRand "math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/AlecAivazis/survey/v2"

	"github.com/reeflective/console"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// GetPrompt returns the prompt string computed for the current context.
func (con *SliverClient) GetPrompt() string {
	prompt := Underline + "sliver" + Normal
	if con.IsServer {
		prompt = Bold + "[server] " + Normal + Underline + "sliver" + Normal
	}
	if con.ActiveTarget.GetSession() != nil {
		prompt += fmt.Sprintf(Bold+Red+" (%s)%s", con.ActiveTarget.GetSession().Name, Normal)
	} else if con.ActiveTarget.GetBeacon() != nil {
		prompt += fmt.Sprintf(Bold+Blue+" (%s)%s", con.ActiveTarget.GetBeacon().Name, Normal)
	}
	prompt += " > "
	return prompt
}

// ExitConfirm tries to exit the Sliver go program.
// It will prompt on stdin for confirmation if:
// - The program is a Sliver server and has active slivers under management.
// - The program is a client and has active port forwarders or SOCKS proxies.
// In any of those cases and without confirm, the function does nothing.
func (con *SliverClient) ExitConfirm() {
	fmt.Println("Exiting...")

	if con.IsServer {
		sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
		if err != nil {
			os.Exit(1)
		}
		beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
		if err != nil {
			os.Exit(1)
		}
		if 0 < len(sessions.Sessions) || 0 < len(beacons.Beacons) {
			con.Printf("There are %d active sessions and %d active beacons.\n", len(sessions.Sessions), len(beacons.Beacons))
			confirm := false
			prompt := &survey.Confirm{Message: "Are you sure you want to exit?"}
			survey.AskOne(prompt, &confirm)
			if !confirm {
				return
			}
		}
	}

	// Client might have portfwds/socks
	portfwds := core.Portfwds.List()
	servers := core.SocksProxies.List()

	if len(portfwds) > 0 {
		con.Printf("There are %d active (bind) port forwarders.\n", len(portfwds))
	}

	if len(servers) > 0 {
		con.Printf("There are %d active SOCKS servers.\n", len(servers))
	}

	if len(portfwds)+len(servers) > 0 {
		confirm := false
		prompt := &survey.Confirm{Message: "Are you sure you want to exit?"}
		survey.AskOne(prompt, &confirm)
		if !confirm {
			return
		}
	}

	os.Exit(0)
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

	con.PrintInfof("(Use Ctrl-C/SIGINT to exit, Ctrl-Z to background)")

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

func (con *SliverClient) waitSignalOrClose() error {
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

	if con.waitingResult == nil {
		con.waitingResult = make(chan bool)
	}

	select {
	case sig := <-sigchan:
		con.PrintInfof("Received signal %s\n", sig)
	case <-con.waitingResult:
		con.waitingResult = make(chan bool)
		return nil
	}

	return nil
}

// printLogo prints the Sliver console logo.
func (con *SliverClient) printLogo() {
	serverVer, err := con.Rpc.GetVersion(context.Background(), &commonpb.Empty{})
	if err != nil {
		log.Fatal(err)
	}
	dirty := ""
	if serverVer.Dirty {
		dirty = fmt.Sprintf(" - %sDirty%s", Bold, Normal)
	}
	serverSemVer := fmt.Sprintf("%d.%d.%d", serverVer.Major, serverVer.Minor, serverVer.Patch)

	logo := asciiLogos[insecureRand.Intn(len(asciiLogos))]
	fmt.Println(strings.ReplaceAll(logo, "\n", "\r\n"))
	fmt.Println("All hackers gain " + abilities[insecureRand.Intn(len(abilities))] + "\r")
	fmt.Printf(Info+"Server v%s - %s%s\r\n", serverSemVer, serverVer.Commit, dirty)
	if version.GitCommit != serverVer.Commit {
		fmt.Printf(Info+"Client %s\r\n", version.FullVersion())
	}
	fmt.Println(Info + "Welcome to the sliver shell, please type 'help' for options\r")
	if serverVer.Major != int32(version.SemanticVersion()[0]) {
		fmt.Printf(Warn + "Warning: Client and server may be running incompatible versions.\r\n")
	}
	con.CheckLastUpdate()
}

// exitConsole prompts the user for confirmation to exit the console.
func (c *SliverClient) exitConsole(_ *console.Console) {
	c.ExitConfirm()
}

// exitImplantMenu uses the background command to detach from the implant menu.
func (c *SliverClient) exitImplantMenu(_ *console.Console) {
	c.ActiveTarget.Background()
	c.PrintInfof("Background ...\n")
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
