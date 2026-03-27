package clientpbutil

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestAIConversationMessageIncludesContextHonorsExplicitFlag(t *testing.T) {
	message := &clientpb.AIConversationMessage{
		Visibility:       clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
		IncludeInContext: boolPtr(false),
	}

	if AIConversationMessageIncludesContext(message) {
		t.Fatalf("expected explicit include-in-context=false to exclude the message")
	}
}

func TestAIConversationMessageIncludesContextFallsBackToVisibility(t *testing.T) {
	message := &clientpb.AIConversationMessage{
		Visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
	}

	if !AIConversationMessageIncludesContext(message) {
		t.Fatalf("expected context visibility to include the message when the flag is unset")
	}
}

func boolPtr(value bool) *bool {
	return &value
}
