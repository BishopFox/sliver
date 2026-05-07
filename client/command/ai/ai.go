package ai

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/termio"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var openAIProgramTTY = tea.OpenTTY

// AICmd launches the AI conversation TUI.
func AICmd(_ *cobra.Command, con *console.SliverClient, _ []string) {
	if con == nil || con.Rpc == nil {
		if con != nil {
			con.PrintErrorf("Connect to a server before using `ai`.\n")
		}
		return
	}

	listenerID, listener := con.CreateEventListener()
	defer con.RemoveEventListener(listenerID)
	restoreEventNotifications := con.SuppressEventNotifications()
	defer restoreEventNotifications()

	model := newAIModel(con, buildAIContext(con), listener)
	model.showExperimentalWarningModal()

	width, height := 100, 30
	if w, h, err := term.GetSize(0); err == nil && w > 0 && h > 0 {
		width, height = w, h
	}

	opts := []tea.ProgramOption{
		tea.WithWindowSize(width, height),
		tea.WithColorProfile(colorprofile.TrueColor),
	}
	ttyOpts, cleanup := configureAIProgramTTY()
	if cleanup != nil {
		defer cleanup()
	}
	opts = append(opts, ttyOpts...)

	program := tea.NewProgram(model, opts...)
	if _, err := program.Run(); err != nil {
		con.PrintErrorf("AI TUI error: %s\n", err)
	}
}

// Console logging can swap stdout/stderr to pipes so session output can be tee'd
// to disk. Bubble Tea needs a real TTY output to enable terminal features like
// mouse reporting, so bind the AI TUI back to the interactive terminal when
// stdout is no longer the controlling TTY.
func configureAIProgramTTY() ([]tea.ProgramOption, func()) {
	stdinTTY := isAITTY(os.Stdin)
	stdoutTTY := isAITTY(os.Stdout)

	switch {
	case !stdinTTY || stdoutTTY:
		return nil, nil

	case isAITTY(termio.InteractiveInput()) && isAITTY(termio.InteractiveOutput()):
		return []tea.ProgramOption{
			tea.WithInput(termio.InteractiveInput()),
			tea.WithOutput(termio.InteractiveOutput()),
		}, nil

	default:
		inTTY, outTTY, err := openAIProgramTTY()
		if err != nil {
			return nil, nil
		}
		return []tea.ProgramOption{
				tea.WithInput(inTTY),
				tea.WithOutput(outTTY),
			}, func() {
				_ = inTTY.Close()
				if outTTY != inTTY {
					_ = outTTY.Close()
				}
			}
	}
}

func isAITTY(file *os.File) bool {
	return file != nil && term.IsTerminal(int(file.Fd()))
}

type aiContext struct {
	target     aiTargetSummary
	connection aiConnectionSummary
	status     string
}

type aiTargetSummary struct {
	SessionID string
	BeaconID  string
	Label     string
	Host      string
	OS        string
	Arch      string
	C2        string
	Mode      string
	Details   []string
}

type aiConnectionSummary struct {
	Profile  string
	Server   string
	Operator string
	State    string
}

func defaultAITargetSummary() aiTargetSummary {
	return aiTargetSummary{
		Label: "No active target",
		Host:  "Select a session or beacon with `use` or ctrl+s",
		Mode:  "offline preview",
		C2:    "n/a",
		OS:    "unknown",
		Arch:  "unknown",
	}
}

func aiSessionTargetSummary(session *clientpb.Session) aiTargetSummary {
	if session == nil {
		return defaultAITargetSummary()
	}

	return aiTargetSummary{
		SessionID: session.ID,
		Label:     fmt.Sprintf("Session %s", fallback(session.Name, session.ID)),
		Host:      fallback(session.Hostname, "<unknown host>"),
		OS:        fallback(session.OS, "unknown"),
		Arch:      fallback(session.Arch, "unknown"),
		C2:        fallback(session.ActiveC2, "unknown"),
		Mode:      "interactive session",
		Details: []string{
			fmt.Sprintf("User: %s", fallback(session.Username, "<unknown>")),
			fmt.Sprintf("PID: %d", session.PID),
			fmt.Sprintf("Remote: %s", fallback(session.RemoteAddress, "<unknown>")),
		},
	}
}

func aiBeaconTargetSummary(beacon *clientpb.Beacon) aiTargetSummary {
	if beacon == nil {
		return defaultAITargetSummary()
	}

	return aiTargetSummary{
		BeaconID: beacon.ID,
		Label:    fmt.Sprintf("Beacon %s", fallback(beacon.Name, beacon.ID)),
		Host:     fallback(beacon.Hostname, "<unknown host>"),
		OS:       fallback(beacon.OS, "unknown"),
		Arch:     fallback(beacon.Arch, "unknown"),
		C2:       fallback(beacon.ActiveC2, "unknown"),
		Mode:     "asynchronous beacon",
		Details: []string{
			fmt.Sprintf("User: %s", fallback(beacon.Username, "<unknown>")),
			fmt.Sprintf("PID: %d", beacon.PID),
			fmt.Sprintf("Remote: %s", fallback(beacon.RemoteAddress, "<unknown>")),
			fmt.Sprintf("Interval: %s", time.Duration(beacon.Interval).String()),
			fmt.Sprintf("Next checkin: %s", formatUnix(beacon.NextCheckin)),
		},
	}
}

func buildAIContext(con *console.SliverClient) aiContext {
	ctx := aiContext{
		target: defaultAITargetSummary(),
		connection: aiConnectionSummary{
			Profile:  "<disconnected>",
			Server:   "<unknown>",
			Operator: "<unknown>",
			State:    "idle",
		},
		status: "Loading AI conversations from the server...",
	}

	if con != nil {
		if details, state, ok := con.CurrentConnection(); ok {
			ctx.connection.State = strings.ToLower(state.String())
			if details != nil && details.Config != nil {
				ctx.connection.Profile = fallback(details.ConfigKey, "<profile unavailable>")
				ctx.connection.Server = fmt.Sprintf("%s:%d", details.Config.LHost, details.Config.LPort)
				ctx.connection.Operator = fallback(details.Config.Operator, "<unknown>")
			}
		}

		session, beacon := con.ActiveTarget.Get()
		switch {
		case session != nil:
			ctx.target = aiSessionTargetSummary(session)
		case beacon != nil:
			ctx.target = aiBeaconTargetSummary(beacon)
		}
	}

	return ctx
}

func fallback(value, def string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	return value
}

func formatUnix(ts int64) string {
	if ts <= 0 {
		return "<unknown>"
	}
	return time.Unix(ts, 0).Local().Format("2006-01-02 15:04:05")
}
