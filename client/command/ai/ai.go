package ai

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// AICmd launches the placeholder Sliver AI TUI.
func AICmd(_ *cobra.Command, con *console.SliverClient, _ []string) {
	model := newAIModel(buildAIContext(con))

	width, height := 100, 30
	if w, h, err := term.GetSize(0); err == nil && w > 0 && h > 0 {
		width, height = w, h
	}

	program := tea.NewProgram(
		model,
		tea.WithWindowSize(width, height),
		tea.WithColorProfile(colorprofile.TrueColor),
	)
	if _, err := program.Run(); err != nil {
		con.PrintErrorf("AI TUI error: %s\n", err)
	}
}

type aiContext struct {
	target        aiTargetSummary
	connection    aiConnectionSummary
	conversations []aiConversation
	planned       []string
	status        string
}

type aiTargetSummary struct {
	Label   string
	Host    string
	OS      string
	Arch    string
	C2      string
	Mode    string
	Details []string
}

type aiConnectionSummary struct {
	Profile  string
	Server   string
	Operator string
	State    string
}

type aiConversation struct {
	Title    string
	Subtitle string
	Messages []aiMessage
}

type aiMessage struct {
	Role string
	Body string
}

func buildAIContext(con *console.SliverClient) aiContext {
	ctx := aiContext{
		target: aiTargetSummary{
			Label: "No active target",
			Host:  "Select a session or beacon with `use`",
			Mode:  "offline preview",
			C2:    "n/a",
			OS:    "unknown",
			Arch:  "unknown",
		},
		connection: aiConnectionSummary{
			Profile:  "<disconnected>",
			Server:   "<unknown>",
			Operator: "<unknown>",
			State:    "idle",
		},
		planned: []string{
			"Streaming chat transcript for operator prompts and assistant replies.",
			"Target-aware actions that can call Sliver commands and summarize output.",
			"Conversation history, pinned context, and tool traces in the side panels.",
		},
		status: "Layout preview only. Prompt submission is captured locally until the AI backend is implemented.",
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
			ctx.target = aiTargetSummary{
				Label: fmt.Sprintf("Session %s", fallback(session.Name, session.ID)),
				Host:  fallback(session.Hostname, "<unknown host>"),
				OS:    fallback(session.OS, "unknown"),
				Arch:  fallback(session.Arch, "unknown"),
				C2:    fallback(session.ActiveC2, "unknown"),
				Mode:  "interactive session",
				Details: []string{
					fmt.Sprintf("User: %s", fallback(session.Username, "<unknown>")),
					fmt.Sprintf("PID: %d", session.PID),
					fmt.Sprintf("Remote: %s", fallback(session.RemoteAddress, "<unknown>")),
				},
			}
		case beacon != nil:
			ctx.target = aiTargetSummary{
				Label: fmt.Sprintf("Beacon %s", fallback(beacon.Name, beacon.ID)),
				Host:  fallback(beacon.Hostname, "<unknown host>"),
				OS:    fallback(beacon.OS, "unknown"),
				Arch:  fallback(beacon.Arch, "unknown"),
				C2:    fallback(beacon.ActiveC2, "unknown"),
				Mode:  "asynchronous beacon",
				Details: []string{
					fmt.Sprintf("User: %s", fallback(beacon.Username, "<unknown>")),
					fmt.Sprintf("Interval: %s", time.Duration(beacon.Interval).String()),
					fmt.Sprintf("Next checkin: %s", formatUnix(beacon.NextCheckin)),
				},
			}
		}
	}

	ctx.conversations = []aiConversation{
		{
			Title:    "Current Target",
			Subtitle: fmt.Sprintf("%s on %s (%s/%s)", ctx.target.Label, ctx.target.Host, ctx.target.OS, ctx.target.Arch),
			Messages: []aiMessage{
				{Role: "system", Body: "Bubble Tea layout preview for the future Sliver AI workspace."},
				{Role: "assistant", Body: fmt.Sprintf("Active target context is loaded as %s with %s over %s.", ctx.target.Mode, ctx.target.Label, ctx.target.C2)},
				{Role: "user", Body: "Summarize the target, operator context, and likely next steps."},
				{Role: "assistant", Body: "Placeholder only: chat execution, command orchestration, and result streaming will be added in a later pass."},
			},
		},
		{
			Title:    "Recon Ideas",
			Subtitle: "Placeholder thread for future guided workflows",
			Messages: []aiMessage{
				{Role: "assistant", Body: "This thread will eventually hold scoped recon plans, command suggestions, and concise result summaries."},
				{Role: "assistant", Body: "For now it exists to exercise the sidebar, transcript pane, and context inspector layout."},
			},
		},
		{
			Title:    "Tooling Backlog",
			Subtitle: "Stubbed product notes for the upcoming AI integration",
			Messages: []aiMessage{
				{Role: "system", Body: "Planned modules: conversation persistence, tool traces, target-aware prompts, and execution approvals."},
				{Role: "assistant", Body: fmt.Sprintf("Current server profile: %s (%s).", ctx.connection.Profile, ctx.connection.State)},
			},
		},
		{
			Title:    "Operator Scratchpad",
			Subtitle: "Local-only placeholder messages",
			Messages: []aiMessage{
				{Role: "assistant", Body: "Submitting a prompt here will append a local placeholder exchange so we can exercise the composer and transcript layout before backend work starts."},
			},
		},
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
