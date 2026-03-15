package docs

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func sampleDocs() []docEntry {
	return []docEntry{
		{
			Name:        "Getting Started",
			Content:     "# Getting Started\n\n- first step\n- second step",
			Description: "Getting Started",
		},
		{
			Name:        "Operators",
			Content:     "# Operators\n\nTeam workflows",
			Description: "Team workflows",
		},
	}
}

func TestRenderMarkdownWithGlowRendersBulletLists(t *testing.T) {
	rendered, err := renderMarkdownWithGlow(48, "# Title\n\n- first item")
	if err != nil {
		t.Fatalf("renderMarkdownWithGlow(): %v", err)
	}

	plain := ansi.Strip(rendered)
	if !strings.Contains(plain, "Title") {
		t.Fatalf("expected heading text, got %q", plain)
	}
	if !strings.Contains(plain, "• first item") {
		t.Fatalf("expected glow-style bullet rendering, got %q", plain)
	}
}

func TestSummarizeMarkdownPrefersFirstMeaningfulLine(t *testing.T) {
	summary := summarizeMarkdown("# Title\n\n- Bullet\n\nParagraph")
	if summary != "Title" {
		t.Fatalf("expected first heading summary, got %q", summary)
	}
}

func TestDocsModelDefaultsToGettingStarted(t *testing.T) {
	model := newDocsModel(sampleDocs())
	model.applyWindowSize(120, 32)

	if model.currentDocName != "Getting Started" {
		t.Fatalf("expected Getting Started to be selected, got %q", model.currentDocName)
	}
	plain := ansi.Strip(model.viewer.View())
	if !strings.Contains(plain, "first step") {
		t.Fatalf("expected initial viewer content to render selected doc, got %q", plain)
	}
}

func TestDocsModelSlashFocusesBrowserAndStartsFiltering(t *testing.T) {
	model := newDocsModel(sampleDocs())
	model.applyWindowSize(120, 32)
	model.focus = docsFocusViewer

	updated, cmd := model.Update(tea.KeyPressMsg{Text: "/", Code: '/'})
	if cmd == nil {
		t.Fatal("expected slash to delegate to the browser filter")
	}

	updatedModel := updated.(*docsModel)
	if updatedModel.focus != docsFocusBrowser {
		t.Fatalf("expected slash to move focus to browser, got %v", updatedModel.focus)
	}

	msg := cmd()
	updated, _ = updatedModel.Update(msg)
	updatedModel = updated.(*docsModel)
	if updatedModel.browser.FilterState() != list.Filtering && updatedModel.browser.FilterState() != list.FilterApplied {
		t.Fatalf("expected slash to enter browser filtering, got %s", updatedModel.browser.FilterState())
	}
}

func TestDocsModelEnterMovesFocusToViewer(t *testing.T) {
	model := newDocsModel(sampleDocs())
	model.applyWindowSize(120, 32)

	updated, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updatedModel := updated.(*docsModel)
	if updatedModel.focus != docsFocusViewer {
		t.Fatalf("expected enter to focus viewer, got %v", updatedModel.focus)
	}
}

func TestDocsWindowPollSchedulesResizeAndKeepsPolling(t *testing.T) {
	model := newDocsModel(sampleDocs())
	model.applyWindowSize(100, 28)

	updated, cmd := model.Update(docsWindowPollMsg{width: 120, height: 34})
	if cmd == nil {
		t.Fatal("expected window poll to schedule resize handling")
	}

	updatedModel := updated.(*docsModel)
	if updatedModel.width != 100 || updatedModel.height != 28 {
		t.Fatalf("expected poll tick to leave dimensions unchanged until WindowSizeMsg, got %dx%d", updatedModel.width, updatedModel.height)
	}

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected poll tick to return tea.BatchMsg, got %T", msg)
	}
	if len(batch) != 2 {
		t.Fatalf("expected poll tick batch size 2, got %d", len(batch))
	}

	var sawWindowSize bool
	var sawPollTick bool
	for _, subcmd := range batch {
		if subcmd == nil {
			continue
		}
		submsg := subcmd()
		switch msg := submsg.(type) {
		case tea.WindowSizeMsg:
			sawWindowSize = true
			if msg.Width != 120 || msg.Height != 34 {
				t.Fatalf("expected resize 120x34, got %dx%d", msg.Width, msg.Height)
			}
		case docsWindowPollMsg:
			sawPollTick = true
		}
	}

	if !sawWindowSize {
		t.Fatal("expected poll tick to emit WindowSizeMsg")
	}
	if !sawPollTick {
		t.Fatal("expected poll tick to schedule another poll")
	}
}
