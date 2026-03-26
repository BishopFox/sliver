package ai

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bishopfox/sliver/client/console"
	aithinking "github.com/bishopfox/sliver/client/spin/thinking"
	clienttheme "github.com/bishopfox/sliver/client/theme"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/charmbracelet/x/ansi"
)

func assertColorEqual(t *testing.T, got color.Color, want color.Color) {
	t.Helper()
	gotR, gotG, gotB, gotA := got.RGBA()
	wantR, wantG, wantB, wantA := want.RGBA()
	if gotR != wantR || gotG != wantG || gotB != wantB || gotA != wantA {
		t.Fatalf("unexpected color: got rgba(%d,%d,%d,%d), want rgba(%d,%d,%d,%d)", gotR, gotG, gotB, gotA, wantR, wantG, wantB, wantA)
	}
}

func TestPromptConversationTitleUsesFirstNonEmptyLine(t *testing.T) {
	title := promptConversationTitle("\n\n  First line title  \nsecond line")
	if title != "First line title" {
		t.Fatalf("expected first non-empty line, got %q", title)
	}
}

func TestBuildConversationMarkdownUsesMarkdownSectionsAndOperatorLabel(t *testing.T) {
	conversation := &clientpb.AIConversation{
		OperatorName: "alice",
		Summary:      "Operator context",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "## Reply"},
		},
	}

	markdown := buildConversationMarkdown(conversation)
	expected := []string{
		"Operator context",
		"### alice",
		"Hello",
		"### AI",
		"## Reply",
	}
	for _, fragment := range expected {
		if !strings.Contains(markdown, fragment) {
			t.Fatalf("expected markdown to contain %q, got %q", fragment, markdown)
		}
	}
	if strings.Contains(markdown, "```text") {
		t.Fatalf("expected markdown messages to avoid text fences, got %q", markdown)
	}
}

func TestBuildConversationMarkdownFallsBackToUserLabel(t *testing.T) {
	conversation := &clientpb.AIConversation{
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	markdown := buildConversationMarkdown(conversation)
	if !strings.Contains(markdown, "### User\n\nHello") {
		t.Fatalf("expected markdown to contain user fallback label, got %q", markdown)
	}
}

func TestMessageBlockLabelIgnoresUnknownOperatorPlaceholders(t *testing.T) {
	conversation := &clientpb.AIConversation{OperatorName: "<unknown>"}
	message := &clientpb.AIConversationMessage{
		Role:         "user",
		OperatorName: "<unknown>",
		Content:      "Hello",
	}

	label := messageBlockLabel(conversation, message)
	if label != "User" {
		t.Fatalf("expected placeholder operator names to fall back to User, got %q", label)
	}
}

func TestTranscriptSpeakerStyleUsesPrimaryThemeForUserMessages(t *testing.T) {
	styles := transcriptSpeakerStyle("alice", "user")

	assertColorEqual(t, styles.border.GetForeground(), clienttheme.Primary())
	assertColorEqual(t, styles.label.GetBackground(), clienttheme.Primary())
}

func TestBuildConversationMarkdownWithoutConversationAvoidsKeyHints(t *testing.T) {
	markdown := buildConversationMarkdown(nil)
	if strings.Contains(markdown, "`n`") {
		t.Fatalf("expected empty-state markdown to avoid inline key hints, got %q", markdown)
	}
	if !strings.Contains(markdown, "Create a new conversation") {
		t.Fatalf("expected empty-state markdown to describe the action without keybindings, got %q", markdown)
	}
}

func TestRenderTranscriptMarkdownLinesRendersAssistantMarkdown(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.currentConversation = &clientpb.AIConversation{
		ID: "conv-1",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "assistant", Content: "## Reply\n\n- first item"},
		},
	}

	rendered := ansi.Strip(strings.Join(model.renderTranscriptMarkdownLines(48), "\n"))
	if !strings.Contains(rendered, "Reply") {
		t.Fatalf("expected rendered transcript to contain the assistant heading text, got %q", rendered)
	}
	if !strings.Contains(rendered, "• first item") {
		t.Fatalf("expected assistant markdown list to be rendered with glow styling, got %q", rendered)
	}
}

func TestRenderConversationTranscriptLinesWrapsMessagesInBoxes(t *testing.T) {
	conversation := &clientpb.AIConversation{
		OperatorName: "alice",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", OperatorName: "alice", Content: "hello"},
			{Role: "user", OperatorName: "bob", Content: "second voice"},
			{Role: "assistant", Content: "## Reply"},
		},
	}

	renderedRaw := strings.Join(renderConversationTranscriptLines(64, conversation), "\n")
	rendered := ansi.Strip(renderedRaw)
	expected := []string{"alice", "bob", "AI", "hello", "second voice", "Reply"}
	for _, fragment := range expected {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected boxed transcript to contain %q, got %q", fragment, rendered)
		}
	}
	for _, fragment := range []string{"╭", "╰"} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected boxed transcript framing %q, got %q", fragment, rendered)
		}
	}
	if strings.Contains(rendered, "╮") || strings.Contains(rendered, "╯") {
		t.Fatalf("expected message framing to stay open on the right, got %q", rendered)
	}
	for _, needle := range []string{"hello", "second voice"} {
		var found bool
		for _, line := range strings.Split(rendered, "\n") {
			if strings.Contains(line, needle) {
				found = true
				if !strings.HasPrefix(line, "│") {
					t.Fatalf("expected %q line to stay inside the box, got %q", needle, line)
				}
			}
		}
		if !found {
			t.Fatalf("expected to find content line for %q, got %q", needle, rendered)
		}
	}
	if !strings.Contains(renderedRaw, "\x1b[") {
		t.Fatalf("expected boxed transcript to include ANSI styling, got %q", renderedRaw)
	}
}

func TestRenderConversationTranscriptLinesOnlyBoxesMessages(t *testing.T) {
	conversation := &clientpb.AIConversation{
		Summary:      "Operator context",
		SystemPrompt: "Stay concise.",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "assistant", Content: "hello"},
		},
	}

	lines := renderConversationTranscriptLines(64, conversation)
	rendered := ansi.Strip(strings.Join(lines, "\n"))

	for _, fragment := range []string{"Summary", "Operator context", "System Prompt", "Stay concise.", "AI", "hello"} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected transcript to contain %q, got %q", fragment, rendered)
		}
	}

	var summaryLine string
	for _, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, "Summary") {
			summaryLine = line
			break
		}
	}
	if summaryLine == "" {
		t.Fatalf("expected summary header line, got %q", rendered)
	}
	if strings.HasPrefix(summaryLine, "╭") || strings.HasPrefix(summaryLine, "│") || strings.HasPrefix(summaryLine, "╰") {
		t.Fatalf("expected summary to remain unboxed, got %q", summaryLine)
	}
}

func TestRenderConversationTranscriptLinesRendersReasoningAndToolBlocks(t *testing.T) {
	conversation := &clientpb.AIConversation{
		OperatorName: "alice",
		Messages: []*clientpb.AIConversationMessage{
			{
				Kind:       clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_REASONING,
				Visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY,
				State:      clientpb.AIConversationMessageState_AI_MESSAGE_STATE_COMPLETED,
				Content:    "Summary:\nChecked the active target before choosing a tool.",
			},
			{
				Kind:          clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_TOOL_CALL,
				Visibility:    clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY,
				State:         clientpb.AIConversationMessageState_AI_MESSAGE_STATE_FAILED,
				ToolName:      "fs_ls",
				ToolArguments: `{"path":"/tmp","session_id":"session-1"}`,
				ToolResult:    `{"path":"/tmp","exists":true}`,
				ErrorText:     "permission denied",
			},
		},
	}

	renderedRaw := strings.Join(renderConversationTranscriptLines(72, conversation), "\n")
	rendered := ansi.Strip(renderedRaw)
	for _, fragment := range []string{"Reasoning", "Checked the active target", "Tool: fs_ls", "ARGUMENTS", "\"path\": \"/tmp\"", "ERROR", "permission denied", "RESULT"} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected transcript to contain %q, got %q", fragment, rendered)
		}
	}
}

func TestRenderTranscriptHeaderLinesUsesCompactInlineMetadata(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.currentConversation = &clientpb.AIConversation{
		ID:        "conv-1",
		Title:     "Thread",
		Provider:  "openai",
		Model:     "gpt-5.4",
		UpdatedAt: time.Now().Unix(),
		Messages: []*clientpb.AIConversationMessage{
			{Role: "assistant", Content: "hello"},
		},
	}

	lines := model.renderTranscriptHeaderLines(96, 12, renderConversationTranscriptLines(96, model.currentConversation))
	if len(lines) != 2 {
		t.Fatalf("expected compact transcript header to use 2 lines, got %d", len(lines))
	}

	rendered := ansi.Strip(strings.Join(lines, "\n"))
	expected := []string{"Conversation", "Thread", "provider openai", "model gpt-5.4", "1 msgs"}
	for _, fragment := range expected {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected transcript header to contain %q, got %q", fragment, rendered)
		}
	}
}

func TestRenderHeaderUsesSingleCompactRow(t *testing.T) {
	model := newAIModel(nil, aiContext{
		target: aiTargetSummary{
			Label: "Session demo",
		},
		connection: aiConnectionSummary{
			Operator: "alice",
		},
	}, nil)
	model.width = 120
	model.loading = false

	rendered := ansi.Strip(model.renderHeader())
	if strings.Contains(rendered, "\n") {
		t.Fatalf("expected compact header to stay on one line, got %q", rendered)
	}
	if strings.Contains(rendered, "Server-backed AI conversation threads") {
		t.Fatalf("expected compact header to omit the old subtitle, got %q", rendered)
	}
	for _, fragment := range []string{"SLIVER AI", "synced", "alice", "Session demo"} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected compact header to contain %q, got %q", fragment, rendered)
		}
	}
}

func TestTranscriptSpeakerPaletteVariesAcrossUsers(t *testing.T) {
	seen := map[int]struct{}{}
	for _, label := range []string{"alice", "bob", "charlie", "dana", "erin", "frank"} {
		seen[transcriptSpeakerPaletteIndex(label, "user")] = struct{}{}
	}
	if len(seen) < 2 {
		t.Fatalf("expected distinct users to map to more than one transcript color, got %d palette entries", len(seen))
	}
}

func TestTranscriptSpeakerStyleUsesPrimaryBorderForUserMessages(t *testing.T) {
	userStyles := transcriptSpeakerStyle("alice", "user")
	userBorder := transcriptSpeakerStyle("alice", "user").border.Render("│")
	wantUserBorder := lipgloss.NewStyle().Foreground(clienttheme.Primary()).Render("│")
	if userBorder != wantUserBorder {
		t.Fatalf("expected user message border to use theme primary color")
	}

	wantUserLabel := lipgloss.NewStyle().
		Bold(true).
		Foreground(clienttheme.DefaultMod(900)).
		Background(clienttheme.Primary()).
		Padding(0, 1).
		Render("alice")
	if got := userStyles.label.Render("alice"); got != wantUserLabel {
		t.Fatalf("expected user message label to use theme primary color")
	}

	wantUserMeta := lipgloss.NewStyle().Foreground(clienttheme.Primary()).Render("openai | gpt-5.4")
	if got := userStyles.meta.Render("openai | gpt-5.4"); got != wantUserMeta {
		t.Fatalf("expected user message metadata to use theme primary color")
	}

	assistantBorder := transcriptSpeakerStyle("AI", "assistant").border.Render("│")
	if assistantBorder == wantUserBorder {
		t.Fatalf("expected assistant message border to keep its non-primary speaker color")
	}
}

func TestWindowResizeQueuesTranscriptRenderAsync(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 100
	model.height = 30
	model.currentConversation = &clientpb.AIConversation{
		ID: "conv-1",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "assistant", Content: "## Reply\n\n- first item"},
		},
	}
	model.invalidateTranscriptCache()

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	if cmd == nil {
		t.Fatal("expected resize to queue transcript rendering")
	}

	updatedModel := updated.(*aiModel)
	if updatedModel.transcriptPendingKey == "" {
		t.Fatal("expected resize to mark a transcript render as pending")
	}

	placeholder := ansi.Strip(strings.Join(updatedModel.transcriptDisplayLines(updatedModel.currentTranscriptWidth()), "\n"))
	if !strings.Contains(placeholder, "Rendering transcript") {
		t.Fatalf("expected resize to show a placeholder while rendering, got %q", placeholder)
	}

	msg := cmd()
	renderedMsg, ok := msg.(aiTranscriptRenderedMsg)
	if !ok {
		t.Fatalf("expected transcript render message, got %T", msg)
	}
	if !strings.Contains(ansi.Strip(strings.Join(renderedMsg.lines, "\n")), "• first item") {
		t.Fatalf("expected rendered resize transcript to include markdown formatting, got %#v", renderedMsg.lines)
	}

	updated, _ = updatedModel.Update(renderedMsg)
	updatedModel = updated.(*aiModel)
	if updatedModel.transcriptPendingKey != "" {
		t.Fatal("expected transcript render to clear the pending state")
	}

	rendered := ansi.Strip(strings.Join(updatedModel.transcriptDisplayLines(updatedModel.currentTranscriptWidth()), "\n"))
	if !strings.Contains(rendered, "• first item") {
		t.Fatalf("expected cached transcript render after resize, got %q", rendered)
	}
}

func TestWindowPollSchedulesWindowSizeMsgAndKeepsPolling(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 100
	model.height = 30

	updated, cmd := model.Update(aiWindowPollMsg{width: 120, height: 36})
	if cmd == nil {
		t.Fatal("expected poll tick to emit a window size message and keep polling")
	}

	updatedModel := updated.(*aiModel)
	if updatedModel.width != 100 || updatedModel.height != 30 {
		t.Fatalf("expected poll tick to leave dimensions unchanged until a WindowSizeMsg arrives, got %dx%d", updatedModel.width, updatedModel.height)
	}

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected poll tick to return a tea.BatchMsg, got %T", msg)
	}
	if len(batch) != 2 {
		t.Fatalf("expected poll tick batch to contain 2 commands, got %d", len(batch))
	}

	var sawWindowSizeMsg bool
	var sawPollTick bool
	for _, subcmd := range batch {
		if subcmd == nil {
			continue
		}
		submsg := subcmd()
		switch msg := submsg.(type) {
		case tea.WindowSizeMsg:
			sawWindowSizeMsg = true
			if msg.Width != 120 || msg.Height != 36 {
				t.Fatalf("expected polled window size 120x36, got %dx%d", msg.Width, msg.Height)
			}
		case aiWindowPollMsg:
			sawPollTick = true
		}
	}

	if !sawWindowSizeMsg {
		t.Fatal("expected poll tick to include a tea.WindowSizeMsg")
	}
	if !sawPollTick {
		t.Fatal("expected poll tick to schedule the next polling tick")
	}
}

func TestApplyWindowSizeSkipsUnchangedDimensions(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 100
	model.height = 30

	if cmd := model.applyWindowSize(100, 30); cmd != nil {
		t.Fatal("expected unchanged dimensions to avoid scheduling a rerender")
	}
}

func TestViewWidthTracksResize(t *testing.T) {
	model := newAIModel(nil, aiContext{
		target: aiTargetSummary{
			Label: "No active target",
			Host:  "Select a session or beacon with `use`",
			OS:    "unknown",
			Arch:  "unknown",
			C2:    "n/a",
		},
		connection: aiConnectionSummary{
			Profile:  "<disconnected>",
			Server:   "<unknown>",
			Operator: "<unknown>",
			State:    "ready",
		},
	}, nil)
	model.loading = false
	model.width = 140
	model.height = 36
	model.conversations = []*clientpb.AIConversation{
		{ID: "conv-1", Title: "New conversation", Provider: "openai", Model: "gpt-5.4"},
	}
	model.currentConversation = &clientpb.AIConversation{
		ID:        "conv-1",
		Title:     "New conversation",
		Provider:  "openai",
		Model:     "gpt-5.4",
		UpdatedAt: time.Now().Unix(),
		Messages: []*clientpb.AIConversationMessage{
			{Role: "assistant", Content: "## Reply\n\n- first item"},
		},
	}

	renderCmd := model.scheduleTranscriptRender()
	if renderCmd == nil {
		t.Fatal("expected initial transcript render command")
	}
	msg := renderCmd()
	renderedMsg, ok := msg.(aiTranscriptRenderedMsg)
	if !ok {
		t.Fatalf("expected initial transcript render message, got %T", msg)
	}
	updated, _ := model.Update(renderedMsg)
	model = updated.(*aiModel)

	view := model.View()
	if got := lipgloss.Width(view.Content); got > model.width {
		t.Fatalf("expected initial view width <= %d, got %d", model.width, got)
	}
	if got := lipgloss.Height(view.Content); got > model.height {
		t.Fatalf("expected initial view height <= %d, got %d", model.height, got)
	}

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 108, Height: 32})
	if cmd == nil {
		t.Fatal("expected resize to queue transcript rendering")
	}
	model = updated.(*aiModel)
	msg = cmd()
	renderedMsg, ok = msg.(aiTranscriptRenderedMsg)
	if !ok {
		t.Fatalf("expected resize transcript render message, got %T", msg)
	}
	updated, _ = model.Update(renderedMsg)
	model = updated.(*aiModel)

	view = model.View()
	if got := lipgloss.Width(view.Content); got > model.width {
		t.Fatalf("expected resized view width <= %d, got %d", model.width, got)
	}
	if got := lipgloss.Height(view.Content); got > model.height {
		t.Fatalf("expected resized view height <= %d, got %d", model.height, got)
	}

	for _, line := range strings.Split(view.Content, "\n") {
		if got := ansi.StringWidth(line); got > model.width {
			t.Fatalf("expected resized view line width <= %d, got %d for %q", model.width, got, ansi.Strip(line))
		}
	}
}

func TestViewRendersAtHeightSeventeen(t *testing.T) {
	model := newAIModel(nil, aiContext{
		target: aiTargetSummary{
			Label: "No active target",
		},
		connection: aiConnectionSummary{
			Operator: "alice",
		},
	}, nil)
	model.width = 96
	model.height = 17
	model.loading = false
	model.status = "Ready"

	rendered := ansi.Strip(model.View().Content)
	if strings.Contains(rendered, "Terminal too small for the AI conversation view.") {
		t.Fatalf("expected 96x17 to render the TUI, got %q", rendered)
	}
	for _, fragment := range []string{"Conversations", "Composer", "focus: sidebar"} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected rendered minimum-height view to contain %q, got %q", fragment, rendered)
		}
	}
}

func TestConversationAwaitingResponseWhenLastMessageIsUser(t *testing.T) {
	conversation := &clientpb.AIConversation{
		Messages: []*clientpb.AIConversationMessage{
			{Role: "assistant", Content: "Previous reply"},
			{Role: "user", Content: "Still waiting"},
		},
	}

	if !conversationAwaitingResponse(conversation) {
		t.Fatal("expected conversation to be waiting on an assistant response")
	}
}

func TestConversationAwaitingResponseStopsWhenAssistantReplies(t *testing.T) {
	conversation := &clientpb.AIConversation{
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "Question"},
			{Role: "assistant", Content: "Answer"},
		},
	}

	if conversationAwaitingResponse(conversation) {
		t.Fatal("expected conversation to be settled once the assistant replies")
	}
}

func TestConversationAwaitingResponseUsesTurnState(t *testing.T) {
	conversation := &clientpb.AIConversation{
		TurnState: clientpb.AIConversationTurnState_AI_TURN_STATE_IN_PROGRESS,
		Messages: []*clientpb.AIConversationMessage{
			{
				Role:       "assistant",
				Content:    "Listing files",
				Kind:       clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_TOOL_CALL,
				Visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY,
			},
		},
	}

	if !conversationAwaitingResponse(conversation) {
		t.Fatal("expected in-progress turn state to keep the conversation pending")
	}

	conversation.TurnState = clientpb.AIConversationTurnState_AI_TURN_STATE_FAILED
	if conversationAwaitingResponse(conversation) {
		t.Fatal("expected failed turn state to clear the pending marker")
	}
}

func TestPendingLabelUsesThinkingWhenConfigured(t *testing.T) {
	model := &aiModel{
		config: &clientpb.AIConfigSummary{ThinkingLevel: "high"},
	}

	if got := model.pendingLabel(); got != "Thinking" {
		t.Fatalf("expected pending label %q, got %q", "Thinking", got)
	}
}

func TestPendingLabelFallsBackToWorking(t *testing.T) {
	model := &aiModel{}

	if got := model.pendingLabel(); got != "Working" {
		t.Fatalf("expected pending label %q, got %q", "Working", got)
	}
}

func TestRenderFooterUsesPaneSpecificControls(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 200
	model.focus = aiFocusSidebar
	model.conversations = []*clientpb.AIConversation{{ID: "conv-1", Title: "Thread"}}
	model.currentConversation = &clientpb.AIConversation{ID: "conv-1", Title: "Thread"}

	footer := ansi.Strip(model.renderFooter())
	expected := []string{"focus: sidebar", "j/k: move", "enter: open", "x: delete", "q/esc: quit"}
	for _, fragment := range expected {
		if !strings.Contains(footer, fragment) {
			t.Fatalf("expected sidebar footer to contain %q, got %q", fragment, footer)
		}
	}
	if strings.Contains(footer, "enter: send") {
		t.Fatalf("expected sidebar footer to avoid composer controls, got %q", footer)
	}

	model.focus = aiFocusTranscript
	footer = ansi.Strip(model.renderFooter())
	expected = []string{"focus: conversation", "j/k: scroll", "pgup/pgdn: page", "g/G: ends", "x: delete", "q/esc: quit"}
	for _, fragment := range expected {
		if !strings.Contains(footer, fragment) {
			t.Fatalf("expected transcript footer to contain %q, got %q", fragment, footer)
		}
	}
	if strings.Contains(footer, "enter: open") || strings.Contains(footer, "enter: send") {
		t.Fatalf("expected transcript footer to avoid sidebar/composer controls, got %q", footer)
	}
	renderedTranscript := ansi.Strip(model.renderTranscript(96, 12))
	if strings.Contains(renderedTranscript, "scroll j/k pgup/pgdn g/G") {
		t.Fatalf("expected transcript pane to omit inline focus controls, got %q", renderedTranscript)
	}

	model.focus = aiFocusComposer
	footer = ansi.Strip(model.renderFooter())
	expected = []string{"focus: composer", "tab: sidebar", "enter: send", "/exit: quit", "ctrl+o: context", "ctrl+u: clear", "esc: blur", "ctrl+c: quit"}
	for _, fragment := range expected {
		if !strings.Contains(footer, fragment) {
			t.Fatalf("expected composer footer to contain %q, got %q", fragment, footer)
		}
	}
	if strings.Contains(footer, "x: delete") {
		t.Fatalf("expected composer footer to avoid sidebar controls, got %q", footer)
	}
}

func TestTabCyclesVisiblePanesOnly(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)

	updated, _ := model.handleGlobalKey(tea.Key{Code: tea.KeyTab})
	if got := updated.(*aiModel).focus; got != aiFocusTranscript {
		t.Fatalf("expected tab to move from sidebar to transcript, got %s", got.String())
	}

	updated, _ = updated.(*aiModel).handleGlobalKey(tea.Key{Code: tea.KeyTab})
	if got := updated.(*aiModel).focus; got != aiFocusComposer {
		t.Fatalf("expected tab to move from transcript to composer, got %s", got.String())
	}

	updated, _ = updated.(*aiModel).handleGlobalKey(tea.Key{Code: tea.KeyTab})
	if got := updated.(*aiModel).focus; got != aiFocusSidebar {
		t.Fatalf("expected tab to wrap back to sidebar, got %s", got.String())
	}
}

func TestAIViewLeavesMouseSelectionToTerminal(t *testing.T) {
	view := aiView("hello")
	if view.MouseMode != tea.MouseModeNone {
		t.Fatalf("expected AI view mouse mode %v, got %v", tea.MouseModeNone, view.MouseMode)
	}
}

func TestMouseClickSwitchesFocusAcrossPanesWide(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 108
	model.height = 32
	model.focus = aiFocusSidebar

	rects := model.currentPaneRects()

	updated, _ := model.Update(tea.MouseClickMsg{
		X:      rects.transcript.x + rects.transcript.width/2,
		Y:      rects.transcript.y + rects.transcript.height/2,
		Button: tea.MouseLeft,
	})
	if got := updated.(*aiModel).focus; got != aiFocusTranscript {
		t.Fatalf("expected transcript click to focus transcript, got %s", got.String())
	}

	rects = updated.(*aiModel).currentPaneRects()
	updated, _ = updated.(*aiModel).Update(tea.MouseClickMsg{
		X:      rects.composer.x + rects.composer.width/2,
		Y:      rects.composer.y + rects.composer.height/2,
		Button: tea.MouseLeft,
	})
	if got := updated.(*aiModel).focus; got != aiFocusComposer {
		t.Fatalf("expected composer click to focus composer, got %s", got.String())
	}

	rects = updated.(*aiModel).currentPaneRects()
	updated, _ = updated.(*aiModel).Update(tea.MouseClickMsg{
		X:      rects.sidebar.x + rects.sidebar.width/2,
		Y:      rects.sidebar.y + rects.sidebar.height/2,
		Button: tea.MouseLeft,
	})
	if got := updated.(*aiModel).focus; got != aiFocusSidebar {
		t.Fatalf("expected sidebar click to focus sidebar, got %s", got.String())
	}
}

func TestMouseClickSwitchesFocusAcrossPanesStacked(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 76
	model.height = 24
	model.focus = aiFocusSidebar

	rects := model.currentPaneRects()
	if rects.transcript.y <= rects.sidebar.y {
		t.Fatalf("expected stacked transcript pane below sidebar, got sidebar=%+v transcript=%+v", rects.sidebar, rects.transcript)
	}

	updated, _ := model.Update(tea.MouseClickMsg{
		X:      rects.transcript.x + rects.transcript.width/2,
		Y:      rects.transcript.y + rects.transcript.height/2,
		Button: tea.MouseLeft,
	})
	if got := updated.(*aiModel).focus; got != aiFocusTranscript {
		t.Fatalf("expected transcript click to focus transcript in stacked layout, got %s", got.String())
	}

	rects = updated.(*aiModel).currentPaneRects()
	updated, _ = updated.(*aiModel).Update(tea.MouseClickMsg{
		X:      rects.composer.x + rects.composer.width/2,
		Y:      rects.composer.y + rects.composer.height/2,
		Button: tea.MouseLeft,
	})
	if got := updated.(*aiModel).focus; got != aiFocusComposer {
		t.Fatalf("expected composer click to focus composer in stacked layout, got %s", got.String())
	}
}

func TestRenderComposerOmitsInlineControls(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 120
	model.status = "Ready"

	rendered := ansi.Strip(model.renderComposer(4))
	if strings.Contains(rendered, "ctrl+u clear") || strings.Contains(rendered, "q quit") {
		t.Fatalf("expected composer pane to omit inline control hints, got %q", rendered)
	}
	if strings.Contains(rendered, "Ready") {
		t.Fatalf("expected composer pane to omit the inline status row, got %q", rendered)
	}
}

func TestComposerPasteMsgInsertsAtCursor(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.focus = aiFocusComposer
	model.input = []rune("hello")
	model.cursor = len(model.input)

	updated, cmd := model.Update(tea.PasteMsg{Content: " world"})
	if cmd != nil {
		t.Fatalf("expected paste to update in place without commands, got %v", cmd)
	}

	updatedModel := updated.(*aiModel)
	if got := string(updatedModel.input); got != "hello world" {
		t.Fatalf("expected pasted composer input, got %q", got)
	}
	if updatedModel.cursor != len([]rune("hello world")) {
		t.Fatalf("expected cursor at end of pasted input, got %d", updatedModel.cursor)
	}
}

func TestTranscriptFocusDisablesMouseForNativeSelection(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 108
	model.height = 28
	model.focus = aiFocusTranscript
	view := model.View()
	if view.MouseMode != tea.MouseModeNone {
		t.Fatalf("expected transcript focus to disable mouse reporting, got %v", view.MouseMode)
	}
}

func TestTranscriptClickPromptsNativeSelection(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 108
	model.height = 28
	model.focus = aiFocusSidebar
	rect := model.currentPaneRects().transcript
	updated, _ := model.Update(tea.MouseClickMsg{
		X:      rect.x + rect.width/2,
		Y:      rect.y + rect.height/2,
		Button: tea.MouseLeft,
	})
	model = updated.(*aiModel)
	if got := model.focus; got != aiFocusTranscript {
		t.Fatalf("expected transcript click to focus transcript, got %s", got.String())
	}
	if !strings.Contains(model.status, "Drag again") {
		t.Fatalf("expected transcript click to explain native selection flow, got %q", model.status)
	}
}

func TestLayoutHeightsKeepWideComposerCompact(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 120
	model.height = 30

	headerHeight, composerHeight, footerHeight, bodyHeight := model.layoutHeights()
	if headerHeight != 1 {
		t.Fatalf("expected header height 1, got %d", headerHeight)
	}
	if composerHeight != 4 {
		t.Fatalf("expected wide composer height 4, got %d", composerHeight)
	}
	if footerHeight != 1 {
		t.Fatalf("expected footer height 1, got %d", footerHeight)
	}
	if bodyHeight != 24 {
		t.Fatalf("expected body height 24 after reclaiming the composer status row, got %d", bodyHeight)
	}
}

func TestRenderPaneClampsEmbeddedContentToViewport(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.focus = aiFocusTranscript

	rendered := model.renderPane(24, 6, aiFocusTranscript, []string{
		"Conversation",
		"line one\nline two\nline three\nline four",
		strings.Repeat("x", 48),
	})

	if got := lipgloss.Width(rendered); got != 24 {
		t.Fatalf("expected pane width 24, got %d", got)
	}
	if got := lipgloss.Height(rendered); got != 6 {
		t.Fatalf("expected pane height 6, got %d", got)
	}

	for _, line := range strings.Split(rendered, "\n") {
		if got := ansi.StringWidth(line); got > 24 {
			t.Fatalf("expected pane line width <= 24, got %d for %q", got, ansi.Strip(line))
		}
	}
}

func TestViewKeepsComposerVisibleWhenTranscriptLinesEmbedNewlines(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.loading = false
	model.width = 120
	model.height = 24
	model.status = "Ready"
	model.conversations = []*clientpb.AIConversation{
		{ID: "conv-1", Title: "Thread", Provider: "openai", Model: "gpt-5.4"},
	}
	model.currentConversation = &clientpb.AIConversation{
		ID:        "conv-1",
		Title:     "Thread",
		Provider:  "openai",
		Model:     "gpt-5.4",
		UpdatedAt: time.Now().Unix(),
		Messages: []*clientpb.AIConversationMessage{
			{Role: "assistant", Content: "hello"},
		},
	}

	width := model.currentTranscriptWidth()
	model.transcriptCacheKey = model.transcriptRenderKey(width)
	model.transcriptCacheLines = []string{
		"╭─ box ─╮",
		"│ alpha │\n│ beta  │\n│ gamma │\n│ delta │\n│ eps   │",
		"╰───────╯",
	}

	view := model.View()
	rendered := ansi.Strip(view.Content)

	if !strings.Contains(rendered, "Composer") {
		t.Fatalf("expected composer pane to remain visible, got %q", rendered)
	}
	if !strings.Contains(rendered, ">>>") {
		t.Fatalf("expected composer input line to remain visible, got %q", rendered)
	}
	if !strings.Contains(rendered, "focus: sidebar") {
		t.Fatalf("expected footer to remain visible, got %q", rendered)
	}
	if got := lipgloss.Height(view.Content); got > model.height {
		t.Fatalf("expected view height <= %d, got %d", model.height, got)
	}
}

func TestHandleGlobalKeyScrollsTranscript(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 108
	model.height = 32
	model.focus = aiFocusTranscript
	model.loading = false

	messages := make([]*clientpb.AIConversationMessage, 0, 16)
	for i := 0; i < 16; i++ {
		messages = append(messages, &clientpb.AIConversationMessage{
			Role:    "assistant",
			Content: fmt.Sprintf("Message %02d\n\n- detail", i),
		})
	}
	model.currentConversation = &clientpb.AIConversation{
		ID:       "conv-1",
		Title:    "Thread",
		Provider: "openai",
		Model:    "gpt-test",
		Messages: messages,
	}

	renderCmd := model.scheduleTranscriptRender()
	if renderCmd == nil {
		t.Fatal("expected transcript render command")
	}

	msg := renderCmd()
	renderedMsg, ok := msg.(aiTranscriptRenderedMsg)
	if !ok {
		t.Fatalf("expected transcript render message, got %T", msg)
	}

	updated, _ := model.Update(renderedMsg)
	model = updated.(*aiModel)
	initialScroll := model.transcriptScroll
	if initialScroll == 0 {
		t.Fatalf("expected transcript to start pinned near the bottom, got scroll %d", initialScroll)
	}

	updated, _ = model.handleGlobalKey(tea.Key{Text: "k"})
	model = updated.(*aiModel)
	if model.transcriptFollow {
		t.Fatal("expected transcript scroll-up to disable follow mode")
	}
	if model.transcriptScroll != initialScroll-1 {
		t.Fatalf("expected transcript scroll to move up by one line, got %d want %d", model.transcriptScroll, initialScroll-1)
	}

	updated, _ = model.handleGlobalKey(tea.Key{Text: "G"})
	model = updated.(*aiModel)
	if !model.transcriptFollow {
		t.Fatal("expected transcript end-jump to restore follow mode")
	}
	if model.transcriptScroll != initialScroll {
		t.Fatalf("expected transcript end-jump to restore bottom scroll %d, got %d", initialScroll, model.transcriptScroll)
	}
}

func TestMouseWheelScrollsTranscriptAndFocusesConversation(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 108
	model.height = 32
	model.focus = aiFocusSidebar
	model.loading = false

	messages := make([]*clientpb.AIConversationMessage, 0, 24)
	for i := 0; i < 24; i++ {
		messages = append(messages, &clientpb.AIConversationMessage{
			Role:    "assistant",
			Content: fmt.Sprintf("Message %02d\n\n- detail", i),
		})
	}
	model.currentConversation = &clientpb.AIConversation{
		ID:       "conv-1",
		Title:    "Thread",
		Provider: "openai",
		Model:    "gpt-test",
		Messages: messages,
	}

	renderCmd := model.scheduleTranscriptRender()
	if renderCmd == nil {
		t.Fatal("expected transcript render command")
	}

	msg := renderCmd()
	renderedMsg, ok := msg.(aiTranscriptRenderedMsg)
	if !ok {
		t.Fatalf("expected transcript render message, got %T", msg)
	}

	updated, _ := model.Update(renderedMsg)
	model = updated.(*aiModel)
	initialScroll := model.transcriptScroll
	if initialScroll < aiTranscriptMouseWheelStep {
		t.Fatalf("expected transcript to have enough scrollback for wheel scrolling, got %d", initialScroll)
	}

	rect := model.currentPaneRects().transcript
	updated, _ = model.Update(tea.MouseWheelMsg{
		X:      rect.x + rect.width/2,
		Y:      rect.y + rect.height/2,
		Button: tea.MouseWheelUp,
	})
	model = updated.(*aiModel)

	if got := model.focus; got != aiFocusTranscript {
		t.Fatalf("expected transcript wheel to focus transcript, got %s", got.String())
	}
	if model.transcriptFollow {
		t.Fatal("expected wheel-up scrolling to disable follow mode")
	}
	if want := initialScroll - aiTranscriptMouseWheelStep; model.transcriptScroll != want {
		t.Fatalf("expected wheel-up scroll %d, got %d", want, model.transcriptScroll)
	}

	updated, _ = model.Update(tea.MouseWheelMsg{
		X:      rect.x + rect.width/2,
		Y:      rect.y + rect.height/2,
		Button: tea.MouseWheelDown,
	})
	model = updated.(*aiModel)
	if !model.transcriptFollow {
		t.Fatal("expected wheel-down at the bottom to restore follow mode")
	}
	if model.transcriptScroll != initialScroll {
		t.Fatalf("expected wheel-down to restore bottom scroll %d, got %d", initialScroll, model.transcriptScroll)
	}
}

func TestRenderTranscriptShowsScrollbarAndTracksScrollPosition(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 108
	model.height = 32
	model.focus = aiFocusTranscript
	model.loading = false

	messages := make([]*clientpb.AIConversationMessage, 0, 18)
	for i := 0; i < 18; i++ {
		messages = append(messages, &clientpb.AIConversationMessage{
			Role:    "assistant",
			Content: fmt.Sprintf("Message %02d detail", i),
		})
	}
	model.currentConversation = &clientpb.AIConversation{
		ID:       "conv-1",
		Title:    "Thread",
		Provider: "openai",
		Model:    "gpt-test",
		Messages: messages,
	}

	renderCmd := model.scheduleTranscriptRender()
	if renderCmd == nil {
		t.Fatal("expected transcript render command")
	}

	msg := renderCmd()
	renderedMsg, ok := msg.(aiTranscriptRenderedMsg)
	if !ok {
		t.Fatalf("expected transcript render message, got %T", msg)
	}

	updated, _ := model.Update(renderedMsg)
	model = updated.(*aiModel)

	viewportWidth, viewportHeight := model.currentTranscriptViewportSize()
	contentLines := model.renderTranscriptDisplayContentLines(viewportWidth)
	bottomCells := model.transcriptScrollbarCells(viewportHeight, len(contentLines))
	if len(bottomCells) != viewportHeight {
		t.Fatalf("expected scrollbar height %d, got %d", viewportHeight, len(bottomCells))
	}
	if !bottomCells[len(bottomCells)-1] {
		t.Fatalf("expected bottom-pinned scrollbar thumb to reach the last row, got %#v", bottomCells)
	}

	paneWidth, paneHeight := model.currentTranscriptPaneSize()
	rendered := ansi.Strip(model.renderTranscript(paneWidth, paneHeight))
	if !strings.Contains(rendered, "#") || !strings.Contains(rendered, ":") {
		t.Fatalf("expected rendered transcript to include scrollbar thumb and track, got %q", rendered)
	}
	for _, line := range strings.Split(rendered, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ":") {
			t.Fatalf("expected scrollbar to stay on the right edge without wrapping into content, got %q", rendered)
		}
	}
	for i, line := range strings.Split(rendered, "\n") {
		if i == 0 || i == paneHeight-1 {
			continue
		}
		if !strings.HasSuffix(line, "│") {
			t.Fatalf("expected transcript pane right border to stay visible on line %d, got %q in %q", i, line, rendered)
		}
	}

	model.scrollTranscriptToTop()
	topCells := model.transcriptScrollbarCells(viewportHeight, len(contentLines))
	if !topCells[0] {
		t.Fatalf("expected top-scrolled scrollbar thumb to reach the first row, got %#v", topCells)
	}
	if topCells[len(topCells)-1] {
		t.Fatalf("expected top-scrolled scrollbar thumb to move away from the last row, got %#v", topCells)
	}
}

func TestViewKeepsTranscriptRightBorderVisibleWhileScrolled(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 120
	model.height = 32
	model.focus = aiFocusTranscript
	model.loading = false
	model.conversations = []*clientpb.AIConversation{{ID: "conv-1", Title: "Thread"}}

	messages := make([]*clientpb.AIConversationMessage, 0, 18)
	for i := 0; i < 18; i++ {
		messages = append(messages, &clientpb.AIConversationMessage{
			Role:    "assistant",
			Content: fmt.Sprintf("Message %02d detail", i),
		})
	}
	model.currentConversation = &clientpb.AIConversation{
		ID:       "conv-1",
		Title:    "Thread",
		Provider: "openai",
		Model:    "gpt-test",
		Messages: messages,
	}

	renderCmd := model.scheduleTranscriptRender()
	if renderCmd == nil {
		t.Fatal("expected transcript render command")
	}

	msg := renderCmd()
	renderedMsg, ok := msg.(aiTranscriptRenderedMsg)
	if !ok {
		t.Fatalf("expected transcript render message, got %T", msg)
	}

	updated, _ := model.Update(renderedMsg)
	model = updated.(*aiModel)
	model.scrollTranscript(-6)

	view := ansi.Strip(model.View().Content)
	lines := strings.Split(view, "\n")
	headerHeight, _, _, _ := model.layoutHeights()
	sidebarWidth := clampInt(model.width/4, 24, 28)
	transcriptWidth := maxInt(40, model.width-sidebarWidth)
	rightEdge := sidebarWidth + transcriptWidth - 1
	_, transcriptPaneHeight := model.currentTranscriptPaneSize()
	topRow := headerHeight
	bottomRow := headerHeight + transcriptPaneHeight - 1

	if topRow >= len(lines) {
		t.Fatalf("expected transcript top row %d within view height %d", topRow, len(lines))
	}
	if []rune(lines[topRow])[rightEdge] != '╮' {
		t.Fatalf("expected transcript top-right corner at row %d col %d, got %q in %q", topRow, rightEdge, string([]rune(lines[topRow])[rightEdge]), lines[topRow])
	}

	for row := headerHeight + 1; row < headerHeight+transcriptPaneHeight-1 && row < len(lines); row++ {
		line := []rune(lines[row])
		if rightEdge >= len(line) {
			t.Fatalf("expected transcript right edge index %d within line %d: %q", rightEdge, row, lines[row])
		}
		if line[rightEdge] != '│' {
			t.Fatalf("expected transcript right border at row %d col %d, got %q in %q", row, rightEdge, string(line[rightEdge]), lines[row])
		}
	}
	if bottomRow >= len(lines) {
		t.Fatalf("expected transcript bottom row %d within view height %d", bottomRow, len(lines))
	}
	if []rune(lines[bottomRow])[rightEdge] != '╯' {
		t.Fatalf("expected transcript bottom-right corner at row %d col %d, got %q in %q", bottomRow, rightEdge, string([]rune(lines[bottomRow])[rightEdge]), lines[bottomRow])
	}
}

func TestComposerEnterStartsAwaitingResponseImmediately(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.focus = aiFocusComposer
	model.loading = false
	model.currentConversation = &clientpb.AIConversation{
		ID:       "conv-1",
		Title:    "Thread",
		Provider: "openai",
		Model:    "gpt-test",
	}
	model.input = []rune("hello")
	model.cursor = len(model.input)

	updated, cmd := model.handleComposerKey(tea.Key{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected enter to queue submit work")
	}

	updatedModel := updated.(*aiModel)
	if !updatedModel.awaitingResponse {
		t.Fatal("expected enter to start awaiting-response state immediately")
	}
	if !updatedModel.submittingPrompt {
		t.Fatal("expected enter to mark prompt submission as in flight")
	}
	if updatedModel.pendingPrompt != "hello" {
		t.Fatalf("expected enter to stage the pending prompt, got %q", updatedModel.pendingPrompt)
	}
	if len(updatedModel.input) != 0 || updatedModel.cursor != 0 {
		t.Fatal("expected enter to clear the composer immediately")
	}
}

func TestComposerSlashExitQuitsProgram(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.focus = aiFocusComposer
	model.loading = true
	model.input = []rune("/exit")
	model.cursor = len(model.input)

	updated, cmd := model.handleComposerKey(tea.Key{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected /exit to quit the TUI")
	}

	updatedModel := updated.(*aiModel)
	if updatedModel.submittingPrompt {
		t.Fatal("expected /exit to avoid prompt submission")
	}
	if updatedModel.pendingPrompt != "" {
		t.Fatalf("expected /exit to avoid staging a prompt, got %q", updatedModel.pendingPrompt)
	}
	if len(updatedModel.input) != 0 || updatedModel.cursor != 0 {
		t.Fatal("expected /exit to clear the composer before quitting")
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %#v", msg)
	}
}

func TestComposerUnknownSlashCommandStaysLocal(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.focus = aiFocusComposer
	model.input = []rune("/wat")
	model.cursor = len(model.input)

	updated, cmd := model.handleComposerKey(tea.Key{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("did not expect unknown slash command to queue work, got %v", cmd)
	}

	updatedModel := updated.(*aiModel)
	if updatedModel.submittingPrompt {
		t.Fatal("expected unknown slash command to avoid prompt submission")
	}
	if updatedModel.awaitingResponse {
		t.Fatal("expected unknown slash command to avoid awaiting-response state")
	}
	if updatedModel.status != `Unknown composer command "/wat". Available: /exit.` {
		t.Fatalf("unexpected unknown command status: %q", updatedModel.status)
	}
	if string(updatedModel.input) != "/wat" {
		t.Fatalf("expected unknown slash command to remain in the composer, got %q", string(updatedModel.input))
	}
}

func TestComposerCtrlOOpensContextModal(t *testing.T) {
	model := newAIModel(nil, aiContext{
		target: aiTargetSummary{
			Label: "Session demo",
			Host:  "demo-host",
			OS:    "linux",
			Arch:  "amd64",
			C2:    "mtls",
			Mode:  "interactive session",
		},
		connection: aiConnectionSummary{
			Profile:  "default",
			Server:   "127.0.0.1:31337",
			Operator: "alice",
			State:    "ready",
		},
	}, nil)
	model.focus = aiFocusComposer

	updated, cmd := model.handleComposerKey(tea.Key{Code: 'o', Mod: tea.ModCtrl})
	if cmd != nil {
		t.Fatalf("did not expect context modal to queue work, got %v", cmd)
	}

	updatedModel := updated.(*aiModel)
	if updatedModel.modal == nil || updatedModel.modal.kind != aiModalKindContext {
		t.Fatalf("expected context modal, got %+v", updatedModel.modal)
	}
	if updatedModel.modal.title != "Context" {
		t.Fatalf("expected context modal title, got %+v", updatedModel.modal)
	}
}

func TestShowDeleteConversationModalTargetsSelectedConversation(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.loading = false
	model.focus = aiFocusSidebar
	model.conversations = []*clientpb.AIConversation{
		{ID: "conv-1", Title: "One"},
		{ID: "conv-2", Title: "Two"},
	}
	model.currentConversation = &clientpb.AIConversation{ID: "conv-1", Title: "One"}
	model.selectedConversation = 1

	updated, cmd := model.handleGlobalKey(tea.Key{Text: "x"})
	if cmd != nil {
		t.Fatalf("did not expect delete modal to start RPC work yet, got %v", cmd)
	}

	updatedModel := updated.(*aiModel)
	if updatedModel.modal == nil || updatedModel.modal.kind != aiModalKindDeleteConfirm {
		t.Fatalf("expected delete confirmation modal, got %+v", updatedModel.modal)
	}
	if updatedModel.modal.conversationID != "conv-2" {
		t.Fatalf("expected selected conversation to be targeted, got %+v", updatedModel.modal)
	}
	if updatedModel.modal.selectedID != "conv-1" {
		t.Fatalf("expected delete modal to preselect the remaining conversation, got %+v", updatedModel.modal)
	}
}

func TestShowNewConversationModalOnNKey(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.loading = false

	updated, cmd := model.handleGlobalKey(tea.Key{Text: "n", Code: 'n'})
	if cmd != nil {
		t.Fatalf("did not expect new conversation modal to start RPC work yet, got %v", cmd)
	}

	updatedModel := updated.(*aiModel)
	if updatedModel.modal == nil || updatedModel.modal.kind != aiModalKindNewConversation {
		t.Fatalf("expected new conversation modal, got %+v", updatedModel.modal)
	}
	if updatedModel.modal.title != "New Conversation" {
		t.Fatalf("expected new conversation modal title, got %+v", updatedModel.modal)
	}
	if got := string(updatedModel.modal.input); got != "New conversation" {
		t.Fatalf("expected default modal input, got %q", got)
	}
	if updatedModel.modal.focus != aiModalFocusInput {
		t.Fatalf("expected modal input focus, got %v", updatedModel.modal.focus)
	}
	if updatedModel.modal.cursor != len([]rune("New conversation")) {
		t.Fatalf("expected cursor at end of default title, got %d", updatedModel.modal.cursor)
	}
}

func TestNewConversationModalCreatesConversationWithTypedTitle(t *testing.T) {
	server := &aiRPCServer{
		saveConversationResp: &clientpb.AIConversation{
			ID:       "conv-created",
			Provider: "openai",
			Model:    "gpt-test",
			Title:    "Operator notes",
		},
	}

	rpcClient, cleanup := newAITestRPCClient(t, server)
	defer cleanup()

	model := newAIModel(&console.SliverClient{Rpc: rpcClient}, aiContext{}, nil)
	model.loading = false

	updated, cmd := model.handleGlobalKey(tea.Key{Text: "n", Code: 'n'})
	if cmd != nil {
		t.Fatalf("did not expect modal open to queue work, got %v", cmd)
	}
	model = updated.(*aiModel)

	for range len([]rune("New conversation")) {
		updated, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
		if cmd != nil {
			t.Fatalf("did not expect backspace to queue work, got %v", cmd)
		}
		model = updated.(*aiModel)
	}

	for _, r := range "Operator notes" {
		updated, cmd = model.Update(tea.KeyPressMsg{Text: string(r), Code: r})
		if cmd != nil {
			t.Fatalf("did not expect typing to queue work, got %v", cmd)
		}
		model = updated.(*aiModel)
	}

	updated, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected enter to create the conversation")
	}

	updatedModel := updated.(*aiModel)
	if updatedModel.modal != nil {
		t.Fatalf("expected modal to close after create, got %+v", updatedModel.modal)
	}
	if !updatedModel.loading {
		t.Fatal("expected create to mark the model as loading")
	}

	msg := cmd()
	created, ok := msg.(aiConversationCreatedMsg)
	if !ok {
		t.Fatalf("expected aiConversationCreatedMsg, got %T", msg)
	}
	if created.conversationID != "conv-created" {
		t.Fatalf("unexpected created conversation id: %q", created.conversationID)
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.saveConversationReqs) != 1 {
		t.Fatalf("expected SaveAIConversation to be called once, got %d", len(server.saveConversationReqs))
	}
	if got := server.saveConversationReqs[0].GetTitle(); got != "Operator notes" {
		t.Fatalf("expected created title %q, got %q", "Operator notes", got)
	}
}

func TestNewConversationModalRequiresTitle(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.modal = &aiModalState{
		kind:  aiModalKindNewConversation,
		title: "New Conversation",
		focus: aiModalFocusInput,
	}

	updated, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("did not expect empty title submit to queue work, got %v", cmd)
	}

	updatedModel := updated.(*aiModel)
	if updatedModel.modal == nil {
		t.Fatal("expected modal to stay open when the title is empty")
	}
	if updatedModel.status != "Type a conversation name first." {
		t.Fatalf("unexpected status: %q", updatedModel.status)
	}
}

func TestNewConversationModalCancelActionClosesModal(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.loading = false

	updated, cmd := model.handleGlobalKey(tea.Key{Text: "n", Code: 'n'})
	if cmd != nil {
		t.Fatalf("did not expect modal open to queue work, got %v", cmd)
	}
	model = updated.(*aiModel)

	updated, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if cmd != nil {
		t.Fatalf("did not expect tab to queue work, got %v", cmd)
	}
	model = updated.(*aiModel)
	if model.modal == nil || model.modal.focus != aiModalFocusCancel {
		t.Fatalf("expected cancel action to be focused, got %+v", model.modal)
	}

	updated, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("did not expect cancel to queue work, got %v", cmd)
	}
	if updated.(*aiModel).modal != nil {
		t.Fatal("expected cancel action to close the new conversation modal")
	}
}

func TestNewConversationModalCancelsOnEscape(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.modal = &aiModalState{
		kind:  aiModalKindNewConversation,
		title: "New Conversation",
		focus: aiModalFocusInput,
		input: []rune("New conversation"),
	}

	updated, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if cmd != nil {
		t.Fatalf("did not expect escape to queue work, got %v", cmd)
	}
	if updated.(*aiModel).modal != nil {
		t.Fatal("expected escape to close the new conversation modal")
	}
}

func TestDeleteConversationModalCancelsOnEscape(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.modal = &aiModalState{
		kind:           aiModalKindDeleteConfirm,
		conversationID: "conv-1",
	}

	updated, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if cmd != nil {
		t.Fatalf("did not expect cancel to queue work, got %v", cmd)
	}
	if updated.(*aiModel).modal != nil {
		t.Fatal("expected escape to close the delete confirmation modal")
	}
}

func TestContextModalCancelsOnEscape(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.modal = &aiModalState{
		kind:  aiModalKindContext,
		title: "Context",
	}

	updated, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if cmd != nil {
		t.Fatalf("did not expect context modal escape to queue work, got %v", cmd)
	}
	if updated.(*aiModel).modal != nil {
		t.Fatal("expected escape to close the context modal")
	}
}

func TestShowExperimentalWarningModalDefaultsToCancelFocus(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)

	model.showExperimentalWarningModal()

	if model.modal == nil || model.modal.kind != aiModalKindExperimentalWarning {
		t.Fatalf("expected experimental warning modal, got %+v", model.modal)
	}
	if model.modal.title != aiExperimentalWarningTitle {
		t.Fatalf("unexpected warning title: %q", model.modal.title)
	}
	if model.modal.body != aiExperimentalWarningBody {
		t.Fatalf("unexpected warning body: %q", model.modal.body)
	}
	if model.modal.focus != aiModalFocusCancel {
		t.Fatalf("expected cancel focus by default, got %v", model.modal.focus)
	}
}

func TestExperimentalWarningModalCancelsOnEnterFromDefaultFocus(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.showExperimentalWarningModal()

	updated, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if updated == nil {
		t.Fatal("expected model to be returned")
	}
	if cmd == nil {
		t.Fatal("expected cancel action to quit")
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %#v", msg)
	}
}

func TestExperimentalWarningModalAcceptsAfterTabFocus(t *testing.T) {
	model := newAIModel(nil, aiContext{status: "Loading AI conversations from the server..."}, nil)
	model.showExperimentalWarningModal()

	updated, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if cmd != nil {
		t.Fatalf("did not expect focus change to queue work, got %v", cmd)
	}
	model = updated.(*aiModel)
	if model.modal == nil || model.modal.focus != aiModalFocusConfirm {
		t.Fatalf("expected confirm focus, got %+v", model.modal)
	}

	updated, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected accept to start AI startup")
	}
	updatedModel := updated.(*aiModel)
	if updatedModel.modal != nil {
		t.Fatalf("expected warning modal to close after accept, got %+v", updatedModel.modal)
	}
	if updatedModel.status != "Loading AI conversations from the server..." {
		t.Fatalf("unexpected status after accept: %q", updatedModel.status)
	}
}

func TestModalViewRetainsBackgroundContent(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 120
	model.height = 30
	model.loading = false
	model.conversations = []*clientpb.AIConversation{{ID: "conv-1", Title: "Thread"}}
	model.currentConversation = &clientpb.AIConversation{ID: "conv-1", Title: "Thread"}
	model.modal = &aiModalState{
		kind:           aiModalKindDeleteConfirm,
		title:          "Delete Conversation?",
		body:           `Delete "Thread" and all of its stored messages from the server? This cannot be undone.`,
		conversationID: "conv-1",
	}

	view := ansi.Strip(model.View().Content)
	if !strings.Contains(view, "Conversations") {
		t.Fatalf("expected modal view to retain the background TUI, got %q", view)
	}
	if !strings.Contains(view, "Delete Conversation?") {
		t.Fatalf("expected modal view to include the overlay content, got %q", view)
	}
}

func TestContextModalViewIncludesStyledContextContent(t *testing.T) {
	model := newAIModel(nil, aiContext{
		target: aiTargetSummary{
			Label:   "Session demo",
			Host:    "demo-host",
			OS:      "linux",
			Arch:    "amd64",
			C2:      "mtls",
			Mode:    "interactive session",
			Details: []string{"User: alice"},
		},
		connection: aiConnectionSummary{
			Profile:  "default",
			Server:   "127.0.0.1:31337",
			Operator: "alice",
			State:    "ready",
		},
	}, nil)
	model.width = 120
	model.height = 30
	model.loading = false
	model.conversations = []*clientpb.AIConversation{{ID: "conv-1", Title: "Thread"}}
	model.currentConversation = &clientpb.AIConversation{
		ID:       "conv-1",
		Title:    "Thread",
		Provider: "openai",
		Model:    "gpt-test",
	}
	model.providers = []*clientpb.AIProviderConfig{{Name: "openai", Configured: true}}
	model.config = &clientpb.AIConfigSummary{Provider: "openai", Model: "gpt-test", ThinkingLevel: "high"}
	model.modal = &aiModalState{
		kind:  aiModalKindContext,
		title: "Context",
	}

	view := ansi.Strip(model.View().Content)
	expected := []string{"Composer", "focus: sidebar", "Context", "Target", "Connection", "Thread", "Session demo", "127.0.0.1:31337"}
	for _, fragment := range expected {
		if !strings.Contains(view, fragment) {
			t.Fatalf("expected context modal view to contain %q, got %q", fragment, view)
		}
	}
}

func TestNewConversationModalViewIncludesOverlayContent(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 120
	model.height = 30
	model.loading = false
	model.conversations = []*clientpb.AIConversation{{ID: "conv-1", Title: "Thread"}}
	model.currentConversation = &clientpb.AIConversation{ID: "conv-1", Title: "Thread"}
	model.modal = &aiModalState{
		kind:   aiModalKindNewConversation,
		title:  "New Conversation",
		focus:  aiModalFocusInput,
		input:  []rune("New conversation"),
		cursor: len([]rune("New conversation")),
	}

	view := ansi.Strip(model.View().Content)
	expected := []string{"Conversations", "New Conversation", "Name", "New conversation", "Cancel", "Create"}
	for _, fragment := range expected {
		if !strings.Contains(view, fragment) {
			t.Fatalf("expected new conversation modal view to contain %q, got %q", fragment, view)
		}
	}
}

func TestExperimentalWarningModalViewIncludesDangerContent(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.width = 120
	model.height = 30
	model.loading = false
	model.conversations = []*clientpb.AIConversation{{ID: "conv-1", Title: "Thread"}}
	model.currentConversation = &clientpb.AIConversation{ID: "conv-1", Title: "Thread"}
	model.showExperimentalWarningModal()

	view := ansi.Strip(model.View().Content)
	expected := []string{
		"Conversations",
		aiExperimentalWarningTitle,
		"provided on an EXPERIMENTAL basis",
		"reliability or data integrity",
		aiExperimentalWarningCancelLabel,
		aiExperimentalWarningConfirmLabel,
	}
	for _, fragment := range expected {
		if !strings.Contains(view, fragment) {
			t.Fatalf("expected warning modal view to contain %q, got %q", fragment, view)
		}
	}
}

func TestDeletedConversationMsgRemovesConversationBeforeReload(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.loading = false
	model.awaitingResponse = true
	model.pendingPrompt = "waiting"
	model.conversations = []*clientpb.AIConversation{
		{ID: "conv-1", Title: "One"},
		{ID: "conv-2", Title: "Two"},
	}
	model.currentConversation = &clientpb.AIConversation{
		ID:    "conv-1",
		Title: "One",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "waiting"},
		},
	}

	updated, cmd := model.Update(aiConversationDeletedMsg{
		conversationID: "conv-1",
		selectedID:     "conv-2",
		status:         `Deleted "One".`,
	})
	if cmd == nil {
		t.Fatal("expected delete completion to reload AI state")
	}

	updatedModel := updated.(*aiModel)
	if len(updatedModel.conversations) != 1 || updatedModel.conversations[0].GetID() != "conv-2" {
		t.Fatalf("expected deleted conversation to be removed locally, got %+v", updatedModel.conversations)
	}
	if updatedModel.currentConversation != nil {
		t.Fatalf("expected deleted current conversation to be cleared, got %+v", updatedModel.currentConversation)
	}
	if updatedModel.awaitingResponse {
		t.Fatal("expected delete to clear pending response state for the removed conversation")
	}
	if updatedModel.pendingPrompt != "" {
		t.Fatalf("expected delete to clear pending prompt, got %q", updatedModel.pendingPrompt)
	}
	if updatedModel.status != `Deleted "One".` {
		t.Fatalf("expected delete status to be preserved, got %q", updatedModel.status)
	}
}

func TestPromptSubmittedStartsAwaitingResponseImmediately(t *testing.T) {
	model := newAIModel(nil, aiContext{
		connection: aiConnectionSummary{Operator: "alice"},
	}, nil)
	model.loading = false
	model.submittingPrompt = true
	model.pendingPrompt = "What changed?"
	model.conversations = []*clientpb.AIConversation{
		{ID: "conv-1", Title: "Thread", Provider: "openai", Model: "gpt-test"},
	}
	model.currentConversation = &clientpb.AIConversation{
		ID:           "conv-1",
		Title:        "Thread",
		Provider:     "openai",
		Model:        "gpt-test",
		OperatorName: "alice",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "assistant", Content: "Previous reply"},
		},
	}

	updated, cmd := model.Update(aiPromptSubmittedMsg{
		conversationID: "conv-1",
		conversation: &clientpb.AIConversation{
			ID:       "conv-1",
			Title:    "Thread",
			Provider: "openai",
			Model:    "gpt-test",
		},
		message: &clientpb.AIConversationMessage{
			ID:             "msg-1",
			ConversationID: "conv-1",
			Role:           "user",
			Content:        "What changed?",
		},
		status: "Saved prompt to Thread. Waiting for AI response...",
	})
	if cmd == nil {
		t.Fatal("expected prompt submit to keep the pending animation active")
	}

	updatedModel := updated.(*aiModel)
	if !updatedModel.awaitingResponse {
		t.Fatal("expected prompt submit to enter awaiting-response state immediately")
	}
	last := lastConversationMessage(updatedModel.currentConversation)
	if last == nil || last.GetRole() != "user" || last.GetContent() != "What changed?" {
		t.Fatalf("expected optimistic user message to be appended, got %#v", last)
	}
	if updatedModel.selectedConversation != 0 {
		t.Fatalf("expected submitted conversation to remain selected, got %d", updatedModel.selectedConversation)
	}
	if updatedModel.pendingPrompt != "" {
		t.Fatalf("expected prompt submit to clear the local pending prompt, got %q", updatedModel.pendingPrompt)
	}
	if updatedModel.submittingPrompt {
		t.Fatal("expected prompt submit to clear the submitting state")
	}
}

func TestPromptSubmittedDoesNotDuplicateUserMessageWhenEventArrivesFirst(t *testing.T) {
	model := newAIModel(nil, aiContext{
		connection: aiConnectionSummary{Operator: "<unknown>"},
	}, nil)
	model.loading = false
	model.submittingPrompt = true
	model.pendingPrompt = "What changed?"
	model.conversations = []*clientpb.AIConversation{
		{ID: "conv-1", Title: "Thread", Provider: "openai", Model: "gpt-test"},
	}
	model.currentConversation = &clientpb.AIConversation{
		ID:       "conv-1",
		Title:    "Thread",
		Provider: "openai",
		Model:    "gpt-test",
		Messages: []*clientpb.AIConversationMessage{
			{ID: "assistant-1", Role: "assistant", Content: "Previous reply"},
			{
				ID:             "msg-1",
				ConversationID: "conv-1",
				Role:           "user",
				Content:        "What changed?",
			},
		},
	}

	updated, _ := model.Update(aiPromptSubmittedMsg{
		conversationID: "conv-1",
		conversation: &clientpb.AIConversation{
			ID:       "conv-1",
			Title:    "Thread",
			Provider: "openai",
			Model:    "gpt-test",
		},
		message: &clientpb.AIConversationMessage{
			ID:             "msg-1",
			ConversationID: "conv-1",
			Role:           "user",
			Content:        "What changed?",
		},
		status: "Saved prompt to Thread. Waiting for AI response...",
	})

	updatedModel := updated.(*aiModel)
	if got := len(updatedModel.currentConversation.GetMessages()); got != 2 {
		t.Fatalf("expected submit replay to preserve 2 messages without duplication, got %d", got)
	}
	last := lastConversationMessage(updatedModel.currentConversation)
	if last == nil || last.GetID() != "msg-1" {
		t.Fatalf("expected the existing saved user message to remain the last message, got %#v", last)
	}
	if label := messageBlockLabel(updatedModel.currentConversation, last); label != "User" {
		t.Fatalf("expected unknown operator placeholders to fall back to User, got %q", label)
	}
}

func TestAsyncErrClearsAwaitingResponseWhenSubmitFails(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.awaitingResponse = true
	model.loading = true
	model.submittingPrompt = true
	model.pendingPrompt = "hello"

	updated, _ := model.Update(aiAsyncErrMsg{err: assertErr("submit failed")})
	updatedModel := updated.(*aiModel)
	if updatedModel.awaitingResponse {
		t.Fatal("expected submit failure to clear awaiting-response state")
	}
	if updatedModel.submittingPrompt {
		t.Fatal("expected submit failure to clear submitting state")
	}
	if updatedModel.pendingPrompt != "" {
		t.Fatalf("expected submit failure to clear pending prompt, got %q", updatedModel.pendingPrompt)
	}
}

func TestStartAwaitingResponseShowsAnimatedFrameImmediately(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.loading = false

	model.startAwaitingResponse()

	rendered := ansi.Strip(strings.Join(model.renderAwaitingResponseLines(80), "\n"))
	if !strings.Contains(rendered, "Working.") {
		t.Fatalf("expected pending placeholder to start with an animated frame, got %q", rendered)
	}
}

func TestStartAwaitingResponseCreatesFreshThinkingAnim(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	original := model.thinkingAnim

	model.startAwaitingResponse()

	if model.thinkingAnim == nil {
		t.Fatal("expected pending state to allocate a thinking animation")
	}
	if model.thinkingAnim == original {
		t.Fatal("expected pending state to create a fresh thinking animation instance")
	}
}

func TestLoadedReplyClearsAwaitingResponse(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.awaitingResponse = true
	model.currentConversation = &clientpb.AIConversation{
		ID:       "conv-1",
		Title:    "Thread",
		Provider: "openai",
		Model:    "gpt-test",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "assistant", Content: "Older reply"},
			{Role: "user", Content: "Pending question"},
		},
	}

	updated, cmd := model.Update(aiLoadedMsg{
		config: &clientpb.AIConfigSummary{Valid: true},
		conversations: []*clientpb.AIConversation{
			{ID: "conv-1", Title: "Thread", Provider: "openai", Model: "gpt-test"},
		},
		conversation: &clientpb.AIConversation{
			ID:       "conv-1",
			Title:    "Thread",
			Provider: "openai",
			Model:    "gpt-test",
			Messages: []*clientpb.AIConversationMessage{
				{Role: "user", Content: "Pending question"},
				{Role: "assistant", Content: "Rendered answer"},
			},
		},
		selectedID: "conv-1",
	})
	if cmd != nil {
		t.Fatalf("expected settled conversation load to stop without queuing animation, got %v", cmd)
	}

	updatedModel := updated.(*aiModel)
	if updatedModel.awaitingResponse {
		t.Fatal("expected awaiting-response state to clear once the assistant reply is loaded")
	}
	last := lastConversationMessage(updatedModel.currentConversation)
	if last == nil || last.GetRole() != "assistant" || last.GetContent() != "Rendered answer" {
		t.Fatalf("expected loaded assistant reply to be current, got %#v", last)
	}
}

func TestRenderTranscriptContentLinesIncludesPendingAssistantBlock(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.awaitingResponse = true
	model.currentConversation = &clientpb.AIConversation{
		ID:           "conv-1",
		OperatorName: "alice",
		Provider:     "openai",
		Model:        "gpt-test",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "hello"},
		},
	}

	lines := model.renderTranscriptContentLines(48)
	rendered := ansi.Strip(strings.Join(lines, "\n"))
	if !strings.Contains(rendered, "AI") {
		t.Fatalf("expected pending assistant block label in transcript, got %q", rendered)
	}
	if !strings.Contains(rendered, "╭") || !strings.Contains(rendered, "│") || !strings.Contains(rendered, "╰") {
		t.Fatalf("expected pending assistant block to use box framing, got %q", rendered)
	}
	if !strings.Contains(rendered, ".") {
		t.Fatalf("expected pending assistant animation content in transcript, got %q", rendered)
	}
	if !strings.Contains(rendered, "hello") {
		t.Fatalf("expected existing transcript content to remain, got %q", rendered)
	}
}

func TestRenderTranscriptContentLinesIncludesPendingUserPrompt(t *testing.T) {
	model := newAIModel(nil, aiContext{
		connection: aiConnectionSummary{Operator: "alice"},
	}, nil)
	model.pendingPrompt = "still saving"
	model.awaitingResponse = true

	lines := model.renderTranscriptContentLines(48)
	rendered := ansi.Strip(strings.Join(lines, "\n"))
	if !strings.Contains(rendered, "alice") {
		t.Fatalf("expected pending prompt to render the operator label, got %q", rendered)
	}
	if !strings.Contains(rendered, "╭") || !strings.Contains(rendered, "╰") {
		t.Fatalf("expected pending prompt to use box framing, got %q", rendered)
	}
	if !strings.Contains(rendered, "still saving") {
		t.Fatalf("expected pending prompt content in transcript, got %q", rendered)
	}
	var wrapped bool
	for _, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, "still saving") {
			wrapped = true
			if !strings.HasPrefix(line, "│") {
				t.Fatalf("expected pending prompt content to stay inside the box, got %q", line)
			}
		}
	}
	if !wrapped {
		t.Fatalf("expected pending prompt line in transcript, got %q", rendered)
	}
	if !strings.Contains(rendered, "AI") {
		t.Fatalf("expected pending assistant placeholder to stay visible, got %q", rendered)
	}
}

func TestLoadedConversationClearsPendingPromptWhenUserMessagePersists(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.pendingPrompt = "What changed?"

	updated, _ := model.Update(aiLoadedMsg{
		config: &clientpb.AIConfigSummary{Valid: true},
		conversations: []*clientpb.AIConversation{
			{ID: "conv-1", Title: "Thread", Provider: "openai", Model: "gpt-test"},
		},
		conversation: &clientpb.AIConversation{
			ID:       "conv-1",
			Title:    "Thread",
			Provider: "openai",
			Model:    "gpt-test",
			Messages: []*clientpb.AIConversationMessage{
				{Role: "user", Content: "What changed?"},
			},
		},
		selectedID: "conv-1",
	})

	updatedModel := updated.(*aiModel)
	if updatedModel.pendingPrompt != "" {
		t.Fatalf("expected loaded conversation to clear the persisted pending prompt, got %q", updatedModel.pendingPrompt)
	}
}

func TestLooksLikeTerminalResponseFragment(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{text: "]11;rgb:0000/0000/0000", want: true},
		{text: "[39;17R", want: true},
		{text: "[35;39R", want: true},
		{text: "rgb:0000/0000/0000", want: true},
		{text: "hello world", want: false},
		{text: "[User]", want: false},
	}

	for _, tc := range tests {
		if got := looksLikeTerminalResponseFragment(tc.text); got != tc.want {
			t.Fatalf("looksLikeTerminalResponseFragment(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

func TestAIProgramAnimatesWhileSubmitPending(t *testing.T) {
	releaseSubmit := make(chan struct{})
	submitStarted := make(chan struct{}, 1)
	server := &aiRPCServer{
		saveMessageStarted: submitStarted,
		saveMessageRelease: releaseSubmit,
		saveMessageResp: &clientpb.AIConversationMessage{
			ID:             "msg-1",
			ConversationID: "conv-1",
			Role:           "user",
			Content:        "hello",
		},
	}

	rpcClient, cleanup := newAITestRPCClient(t, server)
	defer cleanup()

	inner := newAIModel(&console.SliverClient{Rpc: rpcClient}, aiContext{
		connection: aiConnectionSummary{Operator: "alice"},
	}, nil)
	inner.loading = false
	inner.focus = aiFocusComposer
	inner.width = 100
	inner.height = 30
	inner.config = &clientpb.AIConfigSummary{
		Valid:    true,
		Provider: "openai",
		Model:    "gpt-test",
	}
	inner.currentConversation = &clientpb.AIConversation{
		ID:           "conv-1",
		Title:        "Thread",
		Provider:     "openai",
		Model:        "gpt-test",
		OperatorName: "alice",
	}
	inner.input = []rune("hello")
	inner.cursor = len(inner.input)

	observed := &observingAIProgramModel{
		inner:     inner,
		started:   make(chan struct{}),
		stepSeen:  make(chan struct{}, 4),
		submitted: make(chan struct{}, 1),
	}

	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	program := tea.NewProgram(
		observed,
		tea.WithContext(ctx),
		tea.WithInput(nil),
		tea.WithOutput(&out),
		tea.WithWindowSize(inner.width, inner.height),
		tea.WithoutSignals(),
		tea.WithoutSignalHandler(),
	)

	errCh := make(chan error, 1)
	go func() {
		_, err := program.Run()
		errCh <- err
	}()

	select {
	case <-observed.started:
	case <-ctx.Done():
		t.Fatal("program did not start")
	}

	program.Send(tea.KeyPressMsg{Code: tea.KeyEnter})

	select {
	case <-submitStarted:
	case <-ctx.Done():
		t.Fatal("submit RPC did not start")
	}

	select {
	case <-observed.stepSeen:
	case <-time.After(750 * time.Millisecond):
		t.Fatal("expected thinking animation to advance while submit was still pending")
	}

	select {
	case <-observed.submitted:
		t.Fatal("submit finished before the blocked RPC was released")
	default:
	}

	close(releaseSubmit)

	select {
	case <-observed.submitted:
	case <-ctx.Done():
		t.Fatal("submit result did not return after release")
	}

	program.Quit()
	if err := <-errCh; err != nil {
		t.Fatalf("program run failed: %v", err)
	}
}

func TestAIProgramSkipsRedundantPendingConversationReload(t *testing.T) {
	blockConversationLoad := make(chan struct{})
	getConversationStarted := make(chan struct{}, 1)
	server := &aiRPCServer{
		getConversationStart: getConversationStarted,
		getConversationWait:  blockConversationLoad,
		providersResp: &clientpb.AIProviderConfigs{
			Config: &clientpb.AIConfigSummary{
				Valid:    true,
				Provider: "openai",
				Model:    "gpt-test",
			},
		},
		conversationsResp: &clientpb.AIConversations{
			Conversations: []*clientpb.AIConversation{
				{ID: "conv-1", Title: "Thread", Provider: "openai", Model: "gpt-test", UpdatedAt: 100},
			},
		},
		conversationByID: map[string]*clientpb.AIConversation{
			"conv-1": {
				ID:           "conv-1",
				Title:        "Thread",
				Provider:     "openai",
				Model:        "gpt-test",
				OperatorName: "alice",
				UpdatedAt:    100,
				Messages: []*clientpb.AIConversationMessage{
					{Role: "user", Content: "hello", CreatedAt: 100, UpdatedAt: 100},
				},
			},
		},
	}

	rpcClient, cleanup := newAITestRPCClient(t, server)
	defer cleanup()

	inner := newAIModel(&console.SliverClient{Rpc: rpcClient}, aiContext{
		connection: aiConnectionSummary{Operator: "alice"},
	}, nil)
	inner.loading = false
	inner.width = 100
	inner.height = 30
	inner.currentConversation = &clientpb.AIConversation{
		ID:           "conv-1",
		Title:        "Thread",
		Provider:     "openai",
		Model:        "gpt-test",
		OperatorName: "alice",
		UpdatedAt:    100,
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "hello", CreatedAt: 100, UpdatedAt: 100},
		},
	}
	inner.conversations = []*clientpb.AIConversation{
		{ID: "conv-1", Title: "Thread", Provider: "openai", Model: "gpt-test", UpdatedAt: 100},
	}
	initCmd := inner.startAwaitingResponse()

	observed := &observingAIProgramModel{
		inner:     inner,
		initCmd:   initCmd,
		started:   make(chan struct{}),
		stepSeen:  make(chan struct{}, 8),
		submitted: make(chan struct{}, 1),
	}

	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	program := tea.NewProgram(
		observed,
		tea.WithContext(ctx),
		tea.WithInput(nil),
		tea.WithOutput(&out),
		tea.WithWindowSize(inner.width, inner.height),
		tea.WithoutSignals(),
		tea.WithoutSignalHandler(),
	)

	errCh := make(chan error, 1)
	go func() {
		_, err := program.Run()
		errCh <- err
	}()

	select {
	case <-observed.started:
	case <-ctx.Done():
		t.Fatal("program did not start")
	}

	select {
	case <-observed.stepSeen:
	case <-time.After(750 * time.Millisecond):
		t.Fatal("expected animation to start before redundant event processing")
	}

	program.Send(aiConversationEventMsg{event: &clientpb.AIConversationEvent{
		EventType: clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_STARTED,
		Conversation: &clientpb.AIConversation{
			ID:           "conv-1",
			OperatorName: "alice",
			UpdatedAt:    100,
			TurnState:    clientpb.AIConversationTurnState_AI_TURN_STATE_IN_PROGRESS,
		},
	}})

	select {
	case <-getConversationStarted:
		t.Fatal("expected redundant pending event to avoid reloading the conversation")
	case <-time.After(250 * time.Millisecond):
	}

	select {
	case <-observed.stepSeen:
	case <-time.After(750 * time.Millisecond):
		t.Fatal("expected animation to keep advancing after the redundant pending event")
	}

	program.Quit()
	if err := <-errCh; err != nil {
		t.Fatalf("program run failed: %v", err)
	}
	close(blockConversationLoad)
}

type testErr string

func (e testErr) Error() string {
	return string(e)
}

func assertErr(message string) error {
	return testErr(message)
}

type observingAIProgramModel struct {
	inner       *aiModel
	initCmd     tea.Cmd
	started     chan struct{}
	stepSeen    chan struct{}
	submitted   chan struct{}
	startedOnce sync.Once
}

func (m *observingAIProgramModel) Init() tea.Cmd {
	return m.initCmd
}

func (m *observingAIProgramModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case aithinking.StepMsg:
		select {
		case m.stepSeen <- struct{}{}:
		default:
		}
	case aiPromptSubmittedMsg:
		select {
		case m.submitted <- struct{}{}:
		default:
		}
	}

	_, cmd := m.inner.Update(msg)
	return m, cmd
}

func (m *observingAIProgramModel) View() tea.View {
	m.startedOnce.Do(func() {
		close(m.started)
	})
	return m.inner.View()
}

func TestStartupConfigErrorModalIgnoresImmediateKeypress(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)

	updated, cmd := model.Update(aiStartupConfigInvalidMsg{err: "server AI configuration is invalid"})
	if cmd != nil {
		t.Fatalf("did not expect command when showing modal, got %v", cmd)
	}

	updatedModel := updated.(*aiModel)
	if updatedModel.modal == nil {
		t.Fatal("expected modal to be visible")
	}

	updated, cmd = updatedModel.Update(tea.KeyPressMsg{})
	if cmd != nil {
		t.Fatalf("expected immediate keypress to be ignored, got %v", cmd)
	}

	stillOpen := updated.(*aiModel)
	if stillOpen.modal == nil {
		t.Fatal("expected modal to remain visible after immediate keypress")
	}
}

func TestStartupConfigErrorModalQuitsAfterDismissDelay(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.modal = &aiModalState{
		title:          "AI Configuration Error",
		body:           "server AI configuration is invalid",
		dismissReadyAt: time.Now().Add(-time.Second),
	}

	updated, cmd := model.Update(tea.KeyPressMsg{})
	if updated == nil {
		t.Fatal("expected model to be returned")
	}
	if cmd == nil {
		t.Fatal("expected keypress after dismiss delay to quit")
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %#v", msg)
	}
}
