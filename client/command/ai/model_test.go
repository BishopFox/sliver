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

func TestBuildConversationMarkdownIncludesRolesAndContent(t *testing.T) {
	conversation := &clientpb.AIConversation{
		Summary: "Operator context",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "## Reply"},
		},
	}

	markdown := buildConversationMarkdown(conversation)
	expected := []string{
		"Operator context",
		"## You",
		"Hello",
		"## Assistant",
		"## Reply",
	}
	for _, fragment := range expected {
		if !strings.Contains(markdown, fragment) {
			t.Fatalf("expected markdown to contain %q, got %q", fragment, markdown)
		}
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
