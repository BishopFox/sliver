package ai

import (
	"strings"
	"testing"

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
