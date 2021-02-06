package console

import (
	"context"
	"fmt"
	"time"

	"github.com/evilsocket/islazy/tui"

	"github.com/bishopfox/sliver/client/completers"
	consts "github.com/bishopfox/sliver/client/constants"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/log"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

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

const (
	// ensure that nothing remains when we refresh the prompt
	seqClearScreenBelow = "\x1b[0J"
)

// handleServerEvents - Print events coming from the server
func (c *console) handleServerLogs(rpc rpcpb.SliverRPCClient) {

	// Call the server events stream.
	events, err := rpc.Events(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
		return
	}

	for !isDone(events.Context()) {
		event, err := events.Recv()
		if err != nil {
			fmt.Printf(util.RPCError + tui.Dim(" server ") + tui.Red(err.Error()) + "\n")
			continue
		}

		switch event.EventType {
		case consts.CanaryEvent:
			fmt.Printf("\n\n") // Clear screen a bit before announcing shitty news
			fmt.Printf(util.Warn+tui.BOLD+"WARNING: %s%s has been burned (DNS Canary)\n", tui.RESET, event.Session.Name)
			sessions := getSessionsByName(event.Session.Name, transport.RPC)
			var alert string
			for _, session := range sessions {
				alert += fmt.Sprintf("\t🔥 Session #%d is affected\n", session.ID)
			}
			c.Shell.RefreshPromptLog(alert)

		case consts.JobStoppedEvent:
			cctx.Context.Jobs-- // Decrease context jobs counter
			job := event.Job
			line := fmt.Sprintf(util.Info+"Job #%d stopped (%s/%s)\n", job.ID, job.Protocol, job.Name)

			if log.IsSynchronized() {
				fmt.Print(line)
			} else {
				c.Shell.RefreshPromptLog(line)
			}

		case consts.SessionOpenedEvent:
			session := event.Session

			// Increase context slivers counter
			cctx.Context.Slivers++

			// Create a new session data cache for completions
			completers.Cache.AddSessionCache(session)

			// Clear the screen
			fmt.Print(seqClearScreenBelow)

			// The HTTP session handling is performed in two steps:
			// - first we add an "empty" session
			// - then we complete the session info when we receive the Register message from the Sliver
			// This check is here to avoid displaying two sessions events for the same session
			var news string
			if session.OS != "" {
				currentTime := time.Now().Format(time.RFC1123)
				news += fmt.Sprintf("\n\n") // Clear screen a bit before announcing the king
				news += fmt.Sprintf(util.Info+"Session #%d %s - %s (%s) - %s/%s - %v\n",
					session.ID, session.Name, session.RemoteAddress, session.Hostname, session.OS, session.Arch, currentTime)
				if log.IsSynchronized() {
					fmt.Println(news)
				} else {
					fmt.Println(news)
					c.Shell.RefreshPromptCustom(Prompt.Render(), 0, false)
				}
			}

		case consts.SessionUpdateEvent:
			session := event.Session
			currentTime := time.Now().Format(time.RFC1123)
			updated := fmt.Sprintf(util.Info+"Session #%d has been updated - %v\n", session.ID, currentTime)
			if cctx.Context.Sliver != nil && session.ID == cctx.Context.Sliver.ID {
				if log.IsSynchronized() {
					fmt.Print(updated)
				} else {
					fmt.Print(updated)
					c.Shell.RefreshPromptInPlace(Prompt.Render())
				}
			} else {
				if log.IsSynchronized() {
					fmt.Print(updated)
				} else {
					c.Shell.RefreshPromptLog(updated)
				}

			}

		case consts.SessionClosedEvent:
			cctx.Context.Slivers-- // Decrease context slivers counter
			session := event.Session
			var lost string

			// If the session is our current session, handle this case
			if cctx.Context.Sliver != nil && session.ID == cctx.Context.Sliver.ID {

				// Reset the current session and refresh
				cctx.Context.Menu = cctx.Server
				cctx.Context.Sliver = nil

				lost += fmt.Sprintf(util.Warn+"Lost session #%d %s - %s (%s) - %s/%s\n",
					session.ID, session.Name, session.RemoteAddress, session.Hostname, session.OS, session.Arch)
				if log.IsSynchronized() {
					fmt.Print(lost)
				} else {
					fmt.Println(lost)
					c.Shell.RefreshPromptCustom("", 0, false)
				}

			} else {
				// We print a message here if its not about a session we killed ourselves, and adapt prompt
				lost += fmt.Sprintf(util.Warn+"Lost session #%d %s - %s (%s) - %s/%s\n",
					session.ID, session.Name, session.RemoteAddress, session.Hostname, session.OS, session.Arch)

				if log.IsSynchronized() {
					fmt.Print(lost)
				} else {
					fmt.Print(lost)
					c.Shell.RefreshPromptLog(lost)
				}
			}

			// In any case, delete the completion data cache for the session, if any.
			completers.Cache.RemoveSessionData(session)
		}
	}
}

func isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// getSessionsByName - Return all sessions for an Implant by name
func getSessionsByName(name string, rpc rpcpb.SliverRPCClient) []*clientpb.Session {
	sessions, err := rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
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
