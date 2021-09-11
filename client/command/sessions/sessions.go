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
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
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
		con.ActiveTarget.Background()
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
		con.ActiveTarget.Background()
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
		activeSession := con.ActiveTarget.GetSession()
		if activeSession != nil && session.ID == activeSession.ID {
			con.ActiveTarget.Background()
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
			con.ActiveTarget.Set(session, nil)
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
			PrintSessions(sessionsMap, con)
		} else {
			con.PrintInfof("No sessions 🙁\n")
		}
	}
}

// PrintSessions - Print the current sessions
func PrintSessions(sessions map[uint32]*clientpb.Session, con *console.SliverConsoleClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"ID",
		"Name",
		"Transport",
		"Remote Address",
		"Hostname",
		"Username",
		"Operating System",
		"Last Check-in",
		"Health",
	})

	activeIndex := -1
	index := 0
	for _, session := range sessions {
		if con.ActiveTarget.GetSession() != nil && con.ActiveTarget.GetSession().ID == session.ID {
			activeIndex = index + 2 // Two lines for the headers
		}
		index++

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
		tw.AppendRow(table.Row{
			session.ID,
			session.Name,
			session.Transport,
			session.RemoteAddress,
			session.Hostname,
			session.Username,
			fmt.Sprintf("%s/%s", session.OS, session.Arch),
			time.Unix(session.LastCheckin, 0).Format(time.RFC1123),
			burned + SessionHealth,
		})
	}
	tw.SortBy([]table.SortBy{
		{Name: "ID", Mode: table.Asc},
	})

	if activeIndex != -1 {
		lines := strings.Split(tw.Render(), "\n")
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
		con.Printf("%s\n", tw.Render())
	}

}
