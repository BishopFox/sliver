package command

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
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/desertbit/grumble"
)

func sessions(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

	interact := ctx.Flags.String("interact")
	kill := ctx.Flags.String("kill")
	killAll := ctx.Flags.Bool("kill-all")
	clean := ctx.Flags.Bool("clean")

	sessions, err := rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	if killAll {
		ActiveSession.Background()
		for _, session := range sessions.Sessions {
			err := killSession(session, true, rpc)
			if err != nil {
				fmt.Printf(Warn+"%s\n", err)
			}
			fmt.Printf(Info+"\nKilled %s (%d)\n", session.Name, session.ID)
		}
		return
	}

	if clean {
		ActiveSession.Background()
		for _, session := range sessions.Sessions {
			if session.IsDead {
				err := killSession(session, true, rpc)
				if err != nil {
					fmt.Printf(Warn+"%s\n", err)
				}
				fmt.Printf(Info+"\nKilled %s (%d)\n", session.Name, session.ID)
			}
		}
		return
	}
	if kill != "" {
		session := GetSession(kill, rpc)
		activeSession := ActiveSession.Get()
		if activeSession != nil && session.ID == activeSession.ID {
			ActiveSession.Background()
		}
		err := killSession(session, true, rpc)
		if err != nil {
			fmt.Printf(Warn+"%s\n", err)
		}
		return
	}

	if interact != "" {
		session := GetSession(interact, rpc)
		if session != nil {
			ActiveSession.Set(session)
			fmt.Printf(Info+"Active session %s (%d)\n", session.Name, session.ID)
		} else {
			fmt.Printf(Warn+"Invalid session name or session number: %s\n", interact)
		}
	} else {
		sessionsMap := map[uint32]*clientpb.Session{}
		for _, session := range sessions.GetSessions() {
			sessionsMap[session.ID] = session
		}
		if 0 < len(sessionsMap) {
			printSessions(sessionsMap)
		} else {
			fmt.Printf(Info + "No sessions 🙁\n")
		}
	}
}

/*
	So this method is a little more complex than you'd maybe think,
	this is because Go's tabwriter aligns columns by counting bytes
	and since we want to modify the color of the active sliver row
	the number of bytes per row won't line up. So we render the table
	into a buffer and note which row the active sliver is in. Then we
	write each line to the term and insert the ANSI codes just before
	we display the row.
*/
func printSessions(sessions map[uint32]*clientpb.Session) {
	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "ID\tName\tTransport\tRemote Address\tHostname\tUsername\tOperating System\tLast Check-in\tHealth\t")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("ID")),
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Transport")),
		strings.Repeat("=", len("Remote Address")),
		strings.Repeat("=", len("Hostname")),
		strings.Repeat("=", len("Username")),
		strings.Repeat("=", len("Operating System")),
		strings.Repeat("=", len("Last Check-in")),
		strings.Repeat("=", len("Health")))

	// Sort the keys because maps have a randomized order
	var keys []int
	for _, session := range sessions {
		keys = append(keys, int(session.ID))
	}
	sort.Ints(keys) // Fucking Go can't sort int32's, so we convert to/from int's

	activeIndex := -1
	for index, key := range keys {
		session := sessions[uint32(key)]
		if ActiveSession.Get() != nil && ActiveSession.Get().ID == session.ID {
			activeIndex = index + 2 // Two lines for the headers
		}

		var SessionHealth string
		if session.IsDead {
			SessionHealth = bold + red + "[DEAD]" + normal
		} else {
			SessionHealth = bold + green + "[ALIVE]" + normal
		}

		fmt.Fprintf(table, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
			session.ID,
			session.Name,
			session.Transport,
			session.RemoteAddress,
			session.Hostname,
			session.Username,
			fmt.Sprintf("%s/%s", session.OS, session.Arch),
			session.LastCheckin,
			SessionHealth,
		)
	}
	table.Flush()

	if activeIndex != -1 {
		lines := strings.Split(outputBuf.String(), "\n")
		for lineNumber, line := range lines {
			if len(line) == 0 {
				continue
			}
			if lineNumber == activeIndex {
				fmt.Printf("%s%s%s\n", green, line, normal)
			} else {
				fmt.Printf("%s\n", line)
			}
		}
	} else {
		fmt.Printf(outputBuf.String())
	}
}

func use(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	if len(ctx.Args) == 0 {
		fmt.Printf(Warn + "Missing sliver name or session number, see `help use`\n")
		return
	}
	session := GetSession(ctx.Args[0], rpc)
	if session != nil {
		ActiveSession.Set(session)
		fmt.Printf(Info+"Active session %s (%d)\n", session.Name, session.ID)
	} else {
		fmt.Printf(Warn+"Invalid session name or session number '%s'\n", ctx.Args[0])
	}
}

func background(ctx *grumble.Context, _ rpcpb.SliverRPCClient) {
	ActiveSession.Background()
	fmt.Printf(Info + "Background ...\n")
}

func kill(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	err := killSession(session, ctx.Flags.Bool("force"), rpc)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	fmt.Printf(Info+"Killed %s (%d)\n", session.Name, session.ID)
	ActiveSession.Background()
}

func killSession(session *clientpb.Session, force bool, rpc rpcpb.SliverRPCClient) error {
	if session == nil {
		return errors.New("Session does not exist")
	}
	_, err := rpc.KillSession(context.Background(), &sliverpb.KillSessionReq{
		Request: &commonpb.Request{
			SessionID: session.ID,
		},
		Force: force,
	})
	return err
}
