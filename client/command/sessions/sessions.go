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

	"github.com/bishopfox/sliver/client/command/kill"
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
	killFlag := ctx.Flags.String("kill")
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
			err := kill.KillSession(session, true, con)
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
				err := kill.KillSession(session, true, con)
				if err != nil {
					con.PrintErrorf("%s\n", err)
				}
				con.Println()
				con.PrintInfof("Killed %s (%d)\n", session.Name, session.ID)
			}
		}
		return
	}
	if killFlag != "" {
		session := con.GetSession(killFlag)
		activeSession := con.ActiveTarget.GetSession()
		if activeSession != nil && session.ID == activeSession.ID {
			con.ActiveTarget.Background()
		}
		err := kill.KillSession(session, true, con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
		}
		return
	}

	if interact != "" {
		session := con.GetSession(interact)
		if session != nil {
			con.ActiveTarget.Set(session, nil)
			con.PrintInfof("Active session %s (%s)\n", session.Name, ShortSessionID(session.ID))
		} else {
			con.PrintErrorf("Invalid session name or session number: %s\n", interact)
		}
	} else {
		sessionsMap := map[string]*clientpb.Session{}
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
func PrintSessions(sessions map[string]*clientpb.Session, con *console.SliverConsoleClient) {
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
	tw.SortBy([]table.SortBy{
		{Name: "ID", Mode: table.Asc},
	})

	for _, session := range sessions {
		color := console.Normal
		if con.ActiveTarget.GetSession() != nil && con.ActiveTarget.GetSession().ID == session.ID {
			color = console.Green
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
		tw.AppendRow(table.Row{
			fmt.Sprintf(color+"%s"+console.Normal, ShortSessionID(session.ID)),
			fmt.Sprintf(color+"%s"+console.Normal, session.Name),
			fmt.Sprintf(color+"%s"+console.Normal, session.Transport),
			fmt.Sprintf(color+"%s"+console.Normal, session.RemoteAddress),
			fmt.Sprintf(color+"%s"+console.Normal, session.Hostname),
			fmt.Sprintf(color+"%s"+console.Normal, session.Username),
			fmt.Sprintf(color+"%s/%s"+console.Normal, session.OS, session.Arch),
			fmt.Sprintf(color+"%s"+console.Normal, time.Unix(session.LastCheckin, 0).Format(time.RFC1123)),
			burned + SessionHealth,
		})
	}

	con.Printf("%s\n", tw.Render())
}

// ShortSessionID - Shorten the session ID
func ShortSessionID(id string) string {
	return strings.Split(id, "-")[0]
}
