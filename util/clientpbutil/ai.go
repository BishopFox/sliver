package clientpbutil

import "github.com/bishopfox/sliver/protobuf/clientpb"

// AIConversationMessageIncludesContext reports whether a message should be
// included in the model-visible context window.
func AIConversationMessageIncludesContext(message *clientpb.AIConversationMessage) bool {
	if message == nil {
		return false
	}
	if message.IncludeInContext != nil {
		return message.GetIncludeInContext()
	}
	return message.GetVisibility() == clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT
}
