package sessions

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
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"github.com/desertbit/grumble"
)

// SessionsCmd - Display/interact with sessions
func SessionsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {

	interact := ctx.Flags.String("interact")
	kill := ctx.Flags.String("kill")
	killAll := ctx.Flags.Bool("kill-all")
	clean := ctx.Flags.Bool("clean")

	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if killAll {
		con.ActiveSession.Background()
		for _, session := range sessions.Sessions {
			err := killSession(session, true, con)
			if err != nil {
				con.PrintErrorf("%s\n", err)
			}
			con.Println()
			con.PrintInfof("Killed %s (%d)\n", session.Name, session.ID)
		}
		return
	}

	if clean {
		con.ActiveSession.Background()
		for _, session := range sessions.Sessions {
			if session.IsDead {
				err := killSession(session, true, con)
				if err != nil {
					con.PrintErrorf("%s\n", err)
				}
				con.Println()
				con.PrintInfof("Killed %s (%d)\n", session.Name, session.ID)
			}
		}
		return
	}
	if kill != "" {
		session := con.GetSession(kill)
		activeSession := con.ActiveSession.Get()
		if activeSession != nil && session.ID == activeSession.ID {
			con.ActiveSession.Background()
		}
		err := killSession(session, true, con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
		}
		return
	}

	if interact != "" {
		session := con.GetSession(interact)
		if session != nil {
			con.ActiveSession.Set(session)
			con.PrintInfof("Active session %s (%d)\n", session.Name, session.ID)
		} else {
			con.PrintErrorf("Invalid session name or session number: %s\n", interact)
		}
	} else {
		sessionsMap := map[uint32]*clientpb.Session{}
		for _, session := range sessions.GetSessions() {
			sessionsMap[session.ID] = session
		}
		if 0 < len(sessionsMap) {
			printSessions(sessionsMap, con)
		} else {
			con.PrintInfof("No sessions 🙁\n")
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
func printSessions(sessions map[uint32]*clientpb.Session, con *console.SliverConsoleClient) {
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
		strings.Repeat("=", len("Health")),
	)
	// strings.Repeat("=", len("Burned")))

	// Sort the keys because maps have a randomized order
	var keys []int
	for _, session := range sessions {
		keys = append(keys, int(session.ID))
	}
	sort.Ints(keys) // Fucking Go can't sort int32's, so we convert to/from int's

	activeIndex := -1
	for index, key := range keys {
		session := sessions[uint32(key)]
		if con.ActiveSession.Get() != nil && con.ActiveSession.Get().ID == session.ID {
			activeIndex = index + 2 // Two lines for the headers
		}

		var SessionHealth string
		if session.IsDead {
			SessionHealth = console.Bold + console.Red + "[DEAD]" + console.Normal
		} else {
			SessionHealth = console.Bold + console.Green + "[ALIVE]" + console.Normal
		}
		burned := ""
		if session.Burned {
			burned = "🔥"
		}
		fmt.Fprintf(table, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			session.ID,
			session.Name,
			session.Transport,
			session.RemoteAddress,
			session.Hostname,
			session.Username,
			fmt.Sprintf("%s/%s", session.OS, session.Arch),
			time.Unix(session.LastCheckin, 0).Format(time.RFC1123),
			burned+SessionHealth,
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
				con.Printf("%s%s%s\n", console.Green, line, console.Normal)
			} else {
				con.Printf("%s\n", line)
			}
		}
	} else {
		con.Printf(outputBuf.String())
	}
}
