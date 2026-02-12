package sessions

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
	Copyright (C) 2019 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/kill"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// SessionsCmd - Display/interact with sessions.
// SessionsCmd - Display/interact 和 sessions.
func SessionsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	interact, _ := cmd.Flags().GetString("interact")
	killFlag, _ := cmd.Flags().GetString("kill")
	killAll, _ := cmd.Flags().GetBool("kill-all")
	clean, _ := cmd.Flags().GetBool("clean")

	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if killAll {
		con.ActiveTarget.Background()
		for _, session := range sessions.Sessions {
			err := kill.KillSession(session, cmd, con)
			if err != nil {
				con.PrintErrorf("%s\n", err)
			}
			con.Println()
			con.PrintInfof("Killed %s (%s)\n", session.Name, session.ID)
		}
		return
	}

	if clean {
		con.ActiveTarget.Background()
		for _, session := range sessions.Sessions {
			if session.IsDead {
				err := kill.KillSession(session, cmd, con)
				if err != nil {
					con.PrintErrorf("%s", err)
				}
				con.Println()
				con.PrintInfof("Killed %s (%s)", session.Name, session.ID)
			}
		}
		return
	}
	if killFlag != "" {
		session := con.GetSession(killFlag)
		if session == nil {
			con.PrintErrorf("Invalid session name or session number: %s\n", killFlag)
			return
		}
		activeSession := con.ActiveTarget.GetSession()
		if activeSession != nil && session.ID == activeSession.ID {
			con.ActiveTarget.Background()
		}
		err := kill.KillSession(session, cmd, con)
		if err != nil {
			con.PrintErrorf("%s", err)
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
		filter, _ := cmd.Flags().GetString("filter")
		var filterRegex *regexp.Regexp
		if filter != "" {
			var err error

			filterRegex, err = regexp.Compile(filter)
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
		}

		sessionsMap := map[string]*clientpb.Session{}
		for _, session := range sessions.GetSessions() {
			sessionsMap[session.ID] = session
		}
		if 0 < len(sessionsMap) {
			PrintSessions(sessionsMap, filter, filterRegex, con)
		} else {
			con.PrintInfof("No sessions 🙁\n")
		}
	}
}

// PrintSessions - Print the current sessions.
// PrintSessions - Print 当前 sessions.
func PrintSessions(sessions map[string]*clientpb.Session, filter string, filterRegex *regexp.Regexp, con *console.SliverClient) {
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 999
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	wideTermWidth := con.Settings.SmallTermWidth < width

	windowsSessionInList := false
	for _, session := range sessions {
		if session.OS == "windows" {
			windowsSessionInList = true
		}
	}

	if wideTermWidth {
		if windowsSessionInList {
			tw.AppendHeader(table.Row{
				"ID",
				"Name",
				"Transport",
				"Remote Address",
				"Hostname",
				"Username",
				"Process (PID)",
				"Integrity",
				"Operating System",
				"Locale",
				"Last Message",
				"Health",
			})
		} else {
			tw.AppendHeader(table.Row{
				"ID",
				"Name",
				"Transport",
				"Remote Address",
				"Hostname",
				"Username",
				"Process (PID)",
				"Operating System",
				"Locale",
				"Last Message",
				"Health",
			})
		}

	} else {
		tw.AppendHeader(table.Row{
			"ID",
			"Transport",
			"Remote Address",
			"Hostname",
			"Username",
			"Operating System",
			"Health",
		})
	}

	tw.SortBy([]table.SortBy{
		{Name: "ID", Mode: table.Asc},
	})

	for _, session := range sessions {
		style := console.StyleNormal
		if con.ActiveTarget.GetSession() != nil && con.ActiveTarget.GetSession().ID == session.ID {
			style = console.StyleGreen
		}
		var SessionHealth string
		if session.IsDead {
			SessionHealth = console.StyleBoldRed.Render("[DEAD]")
		} else {
			SessionHealth = console.StyleBoldGreen.Render("[ALIVE]")
		}
		burned := ""
		if session.Burned {
			burned = "🔥"
		}
		username := strings.TrimPrefix(session.Username, session.Hostname+"\\") // For non-AD Windows users
		username := strings.TrimPrefix(session.Username, session.Hostname+"\\") // For non__PH0__ Windows 用户

		var rowEntries []string
		if wideTermWidth {
			rowEntries = []string{
				style.Render(ShortSessionID(session.ID)),
				style.Render(session.Name),
				style.Render(session.Transport),
				style.Render(session.RemoteAddress),
				style.Render(session.Hostname),
				style.Render(username),
				style.Render(fmt.Sprintf("%s (%d)", session.Filename, session.PID)),
			}

			if windowsSessionInList {
				rowEntries = append(rowEntries, style.Render(session.Integrity))
			}

			rowEntries = append(rowEntries, []string{
				style.Render(fmt.Sprintf("%s/%s", session.OS, session.Arch)),
				style.Render(session.Locale),
				con.FormatDateDelta(time.Unix(session.LastCheckin, 0), wideTermWidth, false),
				burned + SessionHealth,
			}...)
		} else {
			rowEntries = []string{
				style.Render(ShortSessionID(session.ID)),
				style.Render(session.Transport),
				style.Render(session.RemoteAddress),
				style.Render(session.Hostname),
				style.Render(username),
				style.Render(fmt.Sprintf("%s/%s", session.OS, session.Arch)),
				burned + SessionHealth,
			}
		}
		// Build the row struct
		// Build 行结构
		row := table.Row{}
		for _, entry := range rowEntries {
			row = append(row, entry)
		}
		// Apply filters if any
		// Apply 过滤器（如果有）
		if filter == "" && filterRegex == nil {
			tw.AppendRow(row)
		} else {
			for _, rowEntry := range rowEntries {
				if filter != "" {
					if strings.Contains(rowEntry, filter) {
						tw.AppendRow(row)
						break
					}
				}
				if filterRegex != nil {
					if filterRegex.MatchString(rowEntry) {
						tw.AppendRow(row)
						break
					}
				}
			}
		}
	}

	con.Printf("%s\n", tw.Render())
}

// ShortSessionID - Shorten the session ID.
// ShortSessionID - Shorten session ID.
func ShortSessionID(id string) string {
	return strings.Split(id, "-")[0]
}
