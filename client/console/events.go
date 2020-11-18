package console

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	oThis program is distributed in the hope that it will be useful,
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
	"os"
	"time"

	cmd "github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/client/connection"
	consts "github.com/bishopfox/sliver/client/constants"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/evilsocket/islazy/tui"
)

// startEventHandler - Handle all events coming from the server.
func (c *console) startEventHandler() (err error) {

	// Listen for events on the RPC stream.
	eventStream, err := connection.RPC.Events(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Errorf+"%s\n", err)
		return
	}

	for {
		// Few things to note:
		// 1 - Change the EventTypes or regroup them in the functions below,
		// because each type of event may trigger different console behavior.
		event, err := eventStream.Recv()
		if err == io.EOF || event == nil {
			fmt.Printf(Errorf+"%s\n", err)
			return err
		}

		switch event.EventType {
		case consts.CanaryEvent:
			fmt.Printf("\n\n") // Clear screen a bit before announcing shitty news
			fmt.Printf(util.Warn+tui.BOLD+"WARNING: %s%s has been burned (DNS Canary)\n", normal, event.Session.Name)
			sessions := cmd.GetSessionsByName(event.Session.Name, connection.RPC)
			for _, session := range sessions {
				fmt.Printf("\t🔥 Session #%d is affected\n", session.ID)
			}
			fmt.Println()
			c.Shell.RefreshMultiline(Prompt.Render(), 0, false)

		case consts.JobStoppedEvent:
			job := event.Job
			fmt.Printf(util.Info+"Job #%d stopped (%s/%s)\n", job.ID, job.Protocol, job.Name)
			c.Shell.RefreshMultiline(Prompt.Render(), 0, false)

		case consts.SessionOpenedEvent:
			session := event.Session
			// The HTTP session handling is performed in two steps:
			// - first we add an "empty" session
			// - then we complete the session info when we receive the Register message from the Sliver
			// This check is here to avoid displaying two sessions events for the same session
			if session.OS != "" {
				currentTime := time.Now().Format(time.RFC1123)
				fmt.Printf("\n\n") // Clear screen a bit before announcing the king
				fmt.Printf(Info+"Session #%d %s - %s (%s) - %s/%s - %v\n\n",
					session.ID, session.Name, session.RemoteAddress, session.Hostname, session.OS, session.Arch, currentTime)
			}
			c.Shell.RefreshMultiline(Prompt.Render(), 0, false)

		case consts.SessionUpdateEvent:
			session := event.Session
			currentTime := time.Now().Format(time.RFC1123)
			fmt.Printf("\n\n") // Clear screen a bit before announcing the king
			fmt.Printf(util.Info+"Session #%d has been updated - %v\n\n", session.ID, currentTime)
			c.Shell.RefreshMultiline(Prompt.Render(), 0, false)

		case consts.SessionClosedEvent:
			session := event.Session
			// We print a message here if its not about a session we killed ourselves, and adapt prompt
			if session.ID != cctx.Context.Sliver.ID {
				fmt.Printf("\n\n") // Clear screen a bit before announcing the king
				fmt.Printf(util.Warn+"Lost session #%d %s - %s (%s) - %s/%s\n",
					session.ID, session.Name, session.RemoteAddress, session.Hostname, session.OS, session.Arch)
				c.Shell.RefreshMultiline(Prompt.Render(), 0, false)
			} else {
				// If we have disconnected our own context, we have a 1 sec timelapse to wait for this message.
				time.Sleep(time.Millisecond * 200)
				fmt.Printf("\n" + util.Warn + " Active session disconnected")
			}
			fmt.Println()
		}
	}
}

func eventLoop() {
	eventStream, err := connection.RPC.Events(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	stdout := bufio.NewWriter(os.Stdout)

	for {
		event, err := eventStream.Recv()
		if err == io.EOF || event == nil {
			return
		}

		// Trigger event based on type
		switch event.EventType {

		case consts.JoinedEvent:
			fmt.Printf(clearln+Info+"%s has joined the game\n\n", event.Client.Operator.Name)
		case consts.LeftEvent:
			fmt.Printf(clearln+Info+"%s left the game\n\n", event.Client.Operator)

			fmt.Println()
		}

		// fmt.Printf(getPrompt())
		stdout.Flush()
	}
}

const (
	// ANSI Colors
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
	// Info = bold + cyan + "[*] " + normal
	// Warn - Warn a user
	// Warn = bold + red + "[!] " + normal
	// Debug - Display debug information
	// Debug = bold + purple + "[-] " + normal
	// Woot - Display success
	// Woot = bold + green + "[$] " + normal
)
