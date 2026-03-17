package docs

import (
	tea "charm.land/bubbletea/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// DocsCmd launches the embedded docs browser TUI.
func DocsCmd(_ *cobra.Command, con *console.SliverClient, _ []string) {
	entries, err := loadDocEntries()
	if err != nil {
		if con != nil {
			con.PrintErrorf("Failed to load embedded docs: %s\n", err)
		}
		return
	}

	model := newDocsModel(entries)

	width, height := 100, 32
	if w, h, err := term.GetSize(0); err == nil && w > 0 && h > 0 {
		width, height = w, h
	}

	program := tea.NewProgram(
		model,
		tea.WithWindowSize(width, height),
		tea.WithColorProfile(colorprofile.TrueColor),
	)
	if _, err := program.Run(); err != nil && con != nil {
		con.PrintErrorf("Docs TUI error: %s\n", err)
	}
}
