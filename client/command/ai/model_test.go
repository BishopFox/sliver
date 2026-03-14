package ai

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestPromptConversationTitleUsesFirstNonEmptyLine(t *testing.T) {
	title := promptConversationTitle("\n\n  First line title  \nsecond line")
	if title != "First line title" {
		t.Fatalf("expected first non-empty line, got %q", title)
	}
}

func TestBuildConversationMarkdownUsesFencedBlocksAndOperatorLabel(t *testing.T) {
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
		"```text\n[alice]\nHello\n```",
		"```text\n[AI]\n## Reply\n```",
	}
	for _, fragment := range expected {
		if !strings.Contains(markdown, fragment) {
			t.Fatalf("expected markdown to contain %q, got %q", fragment, markdown)
		}
	}
}

func TestBuildConversationMarkdownFallsBackToUserLabel(t *testing.T) {
	conversation := &clientpb.AIConversation{
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	markdown := buildConversationMarkdown(conversation)
	if !strings.Contains(markdown, "```text\n[User]\nHello\n```") {
		t.Fatalf("expected markdown to contain user fallback label, got %q", markdown)
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

func TestIsRelevantAIConversationEventHonorsOperatorName(t *testing.T) {
	event := &clientpb.AIConversation{OperatorName: "alice"}
	if !isRelevantAIConversationEvent(event, "alice") {
		t.Fatal("expected matching operator names to be relevant")
	}
	if isRelevantAIConversationEvent(event, "bob") {
		t.Fatal("expected mismatched operator names to be ignored")
	}
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
