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
	"github.com/bishopfox/sliver/client/command/output"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// SessionResult represents a single session in structured output.
type SessionResult struct {
	ID             string `json:"id" yaml:"id"`
	Name           string `json:"name" yaml:"name"`
	Transport      string `json:"transport" yaml:"transport"`
	RemoteAddress  string `json:"remoteAddress" yaml:"remoteAddress"`
	Hostname       string `json:"hostname" yaml:"hostname"`
	Username       string `json:"username" yaml:"username"`
	Process        string `json:"process" yaml:"process"`
	PID            uint32 `json:"pid" yaml:"pid"`
	OS             string `json:"os" yaml:"os"`
	Arch           string `json:"arch" yaml:"arch"`
	Locale         string `json:"locale" yaml:"locale"`
	Integrity      string `json:"integrity,omitempty" yaml:"integrity,omitempty"`
	LastCheckin    int64  `json:"lastCheckin" yaml:"lastCheckin"`
	IsDead         bool   `json:"isDead" yaml:"isDead"`
	Burned         bool   `json:"burned" yaml:"burned"`
	Active         bool   `json:"active" yaml:"active"`
}

// SessionListResult represents the sessions list in structured output.
type SessionListResult struct {
	Sessions []SessionResult `json:"sessions" yaml:"sessions"`
}

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
			format := output.GetOutputFormat(cmd)
			if format != output.FormatText {
				PrintSessionsStructured(sessionsMap, filter, filterRegex, con, format)
			} else {
				PrintSessions(sessionsMap, filter, filterRegex, con)
			}
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
		row := table.Row{}
		for _, entry := range rowEntries {
			row = append(row, entry)
		}
		// Apply filters if any
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
func ShortSessionID(id string) string {
	return strings.Split(id, "-")[0]
}

// PrintSessionsStructured prints sessions in JSON or YAML format.
func PrintSessionsStructured(sessions map[string]*clientpb.Session, filter string, filterRegex *regexp.Regexp, con *console.SliverClient, format output.OutputFormat) {
	result := SessionListResult{
		Sessions: make([]SessionResult, 0),
	}

	for _, session := range sessions {
		username := strings.TrimPrefix(session.Username, session.Hostname+"\\")
		sr := SessionResult{
			ID:            session.ID,
			Name:          session.Name,
			Transport:     session.Transport,
			RemoteAddress: session.RemoteAddress,
			Hostname:      session.Hostname,
			Username:      username,
			Process:       session.Filename,
			PID:           session.PID,
			OS:            session.OS,
			Arch:          session.Arch,
			Locale:        session.Locale,
			Integrity:     session.Integrity,
			LastCheckin:   session.LastCheckin,
			IsDead:        session.IsDead,
			Burned:        session.Burned,
			Active:        con.ActiveTarget.GetSession() != nil && con.ActiveTarget.GetSession().ID == session.ID,
		}

		// Apply filters
		if filter != "" || filterRegex != nil {
			match := false
			fields := []string{
				session.ID, session.Name, session.Transport, session.RemoteAddress,
				session.Hostname, username, session.Filename, session.OS, session.Arch,
			}
			if filter != "" {
				for _, field := range fields {
					if strings.Contains(field, filter) {
						match = true
						break
					}
				}
			}
			if !match && filterRegex != nil {
				for _, field := range fields {
					if filterRegex.MatchString(field) {
						match = true
						break
					}
				}
			}
			if !match {
				continue
			}
		}

		result.Sessions = append(result.Sessions, sr)
	}

	if err := output.PrintStructured(result, format); err != nil {
		con.PrintErrorf("Failed to format output: %s\n", err)
	}
}
