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
	"regexp"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/kill"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/command/targettable"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// SessionsCmd - Display/interact with sessions.
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
func PrintSessions(sessions map[string]*clientpb.Session, filter string, filterRegex *regexp.Regexp, con *console.SliverClient) {
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 999
	}
	tw := renderSessions(sessions, filter, filterRegex, con, width, time.Now())
	con.Printf("%s\n", tw.Render())
}

func renderSessions(sessions map[string]*clientpb.Session, filter string, filterRegex *regexp.Regexp, con *console.SliverClient, width int, now time.Time) table.Writer {
	hasIntegrity := false
	for _, session := range sessions {
		if session.OS == "windows" {
			hasIntegrity = true
			break
		}
	}

	return targettable.Fit(width, hasIntegrity, func(layout targettable.Layout) table.Writer {
		return buildSessionsTable(sessions, filter, filterRegex, con, layout, hasIntegrity, now)
	})
}

func buildSessionsTable(sessions map[string]*clientpb.Session, filter string, filterRegex *regexp.Regexp, con *console.SliverClient, layout targettable.Layout, hasIntegrity bool, now time.Time) table.Writer {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))

	if layout.Compact {
		tw.AppendHeader(table.Row{
			"ID",
			"Transport",
			"Remote Address",
			"Hostname",
			"Username",
			"Operating System",
			"Health",
		})
	} else {
		header := table.Row{
			"ID",
			"Name",
			"Transport",
			"Remote Address",
			"Hostname",
			"Username",
			"Process (PID)",
		}
		if hasIntegrity && layout.IncludeIntegrity {
			header = append(header, "Integrity")
		}
		header = append(header, "Operating System")
		if layout.IncludeLocale {
			header = append(header, "Locale")
		}
		header = append(header, "Last Message", "Health")
		tw.AppendHeader(header)
	}

	tw.SortBy([]table.SortBy{
		{Name: "ID", Mode: table.Asc},
	})

	activeSessionID := ""
	if con.ActiveTarget != nil {
		if activeSession := con.ActiveTarget.GetSession(); activeSession != nil {
			activeSessionID = activeSession.ID
		}
	}

	for _, session := range sessions {
		username := strings.TrimPrefix(session.Username, session.Hostname+"\\") // For non-AD Windows users
		lastMessage := formatRelativeTime(time.Unix(session.LastCheckin, 0), now)
		health := "[ALIVE]"
		if session.IsDead {
			health = "[DEAD]"
		}
		burned := ""
		if session.Burned {
			burned = "🔥"
		}
		canonicalEntries := []string{
			ShortSessionID(session.ID),
			session.Name,
			session.Transport,
			session.RemoteAddress,
			session.Hostname,
			username,
			fmt.Sprintf("%s (%d)", session.Filename, session.PID),
		}
		if hasIntegrity {
			canonicalEntries = append(canonicalEntries, session.Integrity)
		}
		canonicalEntries = append(canonicalEntries,
			fmt.Sprintf("%s/%s", session.OS, session.Arch),
			session.Locale,
			lastMessage,
			burned+health,
		)
		if !matchesSessionFilter(canonicalEntries, filter, filterRegex) {
			continue
		}

		style := console.StyleNormal
		if activeSessionID == session.ID {
			style = console.StyleGreen
		}
		var sessionHealth string
		if session.IsDead {
			sessionHealth = console.StyleBoldRed.Render("[DEAD]")
		} else {
			sessionHealth = console.StyleBoldGreen.Render("[ALIVE]")
		}

		var rowEntries []string
		if layout.Compact {
			rowEntries = []string{
				style.Render(ShortSessionID(session.ID)),
				style.Render(session.Transport),
				style.Render(session.RemoteAddress),
				style.Render(session.Hostname),
				style.Render(username),
				style.Render(fmt.Sprintf("%s/%s", session.OS, session.Arch)),
				burned + sessionHealth,
			}
		} else {
			processName := session.Filename
			if layout.ProcessBasename {
				processName = targettable.ProcessName(processName)
			}
			rowEntries = []string{
				style.Render(ShortSessionID(session.ID)),
				style.Render(session.Name),
				style.Render(session.Transport),
				style.Render(session.RemoteAddress),
				style.Render(session.Hostname),
				style.Render(username),
				style.Render(fmt.Sprintf("%s (%d)", processName, session.PID)),
			}

			if hasIntegrity && layout.IncludeIntegrity {
				rowEntries = append(rowEntries, style.Render(session.Integrity))
			}

			rowEntries = append(rowEntries, style.Render(fmt.Sprintf("%s/%s", session.OS, session.Arch)))
			if layout.IncludeLocale {
				rowEntries = append(rowEntries, style.Render(session.Locale))
			}
			rowEntries = append(rowEntries, lastMessage, burned+sessionHealth)
		}
		// Build the row struct
		row := table.Row{}
		for _, entry := range rowEntries {
			row = append(row, entry)
		}
		tw.AppendRow(row)
	}

	return tw
}

func matchesSessionFilter(entries []string, filter string, filterRegex *regexp.Regexp) bool {
	if filter == "" && filterRegex == nil {
		return true
	}
	for _, entry := range entries {
		if filter != "" && strings.Contains(entry, filter) {
			return true
		}
		if filterRegex != nil && filterRegex.MatchString(entry) {
			return true
		}
	}
	return false
}

func formatRelativeTime(t time.Time, now time.Time) string {
	if t.After(now) {
		return fmt.Sprintf("in %s", t.Sub(now).Round(time.Second))
	}
	return fmt.Sprintf("%s ago", now.Sub(t).Round(time.Second))
}

// ShortSessionID - Shorten the session ID.
func ShortSessionID(id string) string {
	return strings.Split(id, "-")[0]
}
