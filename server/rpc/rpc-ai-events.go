package rpc

import (
	"context"
	"strings"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	serverai "github.com/bishopfox/sliver/server/ai"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"google.golang.org/protobuf/proto"
)

func publishAIConversationEvent(event *clientpb.AIConversationEvent) {
	if event == nil {
		return
	}
	if strings.TrimSpace(event.GetConversationID()) == "" {
		switch {
		case event.GetConversation() != nil:
			event.ConversationID = event.GetConversation().GetID()
		case event.GetMessage() != nil:
			event.ConversationID = event.GetMessage().GetConversationID()
		}
	}
	if event.GetConversation() != nil && len(event.GetConversation().GetMessages()) > 0 {
		event.Conversation = aiConversationSummary(event.GetConversation())
	}

	data, err := proto.Marshal(event)
	if err != nil {
		rpcLog.Warnf("Failed to marshal AI conversation event: %s", err)
		return
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.AIConversationEvent,
		Data:      data,
	})
}

func aiConversationSummary(conversation *clientpb.AIConversation) *clientpb.AIConversation {
	if conversation == nil {
		return nil
	}
	summary, ok := proto.Clone(conversation).(*clientpb.AIConversation)
	if !ok {
		return conversation
	}
	summary.Messages = nil
	return summary
}

func loadAIConversationSummary(conversationID string) (*clientpb.AIConversation, error) {
	conversation, err := db.AIConversationByID(conversationID, "", false)
	if err != nil {
		return nil, err
	}
	return aiConversationSummary(conversation), nil
}

func publishAIConversationSummaryEvent(eventType clientpb.AIConversationEventType, conversation *clientpb.AIConversation, turnID string, errText string) {
	publishAIConversationEvent(&clientpb.AIConversationEvent{
		EventType:      eventType,
		ConversationID: selectedAIConversationID(conversation),
		TurnID:         strings.TrimSpace(turnID),
		ErrorText:      strings.TrimSpace(errText),
		Conversation:   aiConversationSummary(conversation),
	})
}

func publishAIConversationMessageEvent(
	eventType clientpb.AIConversationEventType,
	conversation *clientpb.AIConversation,
	message *clientpb.AIConversationMessage,
	turnID string,
	errText string,
) {
	publishAIConversationEvent(&clientpb.AIConversationEvent{
		EventType:      eventType,
		ConversationID: selectedAIConversationID(conversation),
		TurnID:         strings.TrimSpace(turnID),
		ErrorText:      strings.TrimSpace(errText),
		Conversation:   aiConversationSummary(conversation),
		Message:        message,
	})
}

func selectedAIConversationID(conversation *clientpb.AIConversation) string {
	if conversation == nil {
		return ""
	}
	return strings.TrimSpace(conversation.GetID())
}

type aiConversationEventSink struct {
	conversationID string
	operatorName   string
	runtime        *serverai.RuntimeConfig
	turnID         string
}

func newAIConversationEventSink(conversationID string, operatorName string, runtime *serverai.RuntimeConfig, turnID string) *aiConversationEventSink {
	return &aiConversationEventSink{
		conversationID: strings.TrimSpace(conversationID),
		operatorName:   strings.TrimSpace(operatorName),
		runtime:        runtime,
		turnID:         strings.TrimSpace(turnID),
	}
}

func (s *aiConversationEventSink) TurnStarted() error {
	conversation, err := s.updateConversationTurnState(s.turnID, clientpb.AIConversationTurnState_AI_TURN_STATE_IN_PROGRESS)
	if err != nil {
		return err
	}
	publishAIConversationSummaryEvent(clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_STARTED, conversation, s.turnID, "")
	return nil
}

func (s *aiConversationEventSink) TurnCompleted() error {
	conversation, err := s.updateConversationTurnState("", clientpb.AIConversationTurnState_AI_TURN_STATE_IDLE)
	if err != nil {
		return err
	}
	publishAIConversationSummaryEvent(clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_COMPLETED, conversation, s.turnID, "")
	return nil
}

func (s *aiConversationEventSink) TurnFailed(failure error) error {
	errText := ""
	if failure != nil {
		errText = strings.TrimSpace(failure.Error())
	}
	conversation, err := s.updateConversationTurnState("", clientpb.AIConversationTurnState_AI_TURN_STATE_FAILED)
	if err != nil {
		return err
	}
	publishAIConversationSummaryEvent(clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_FAILED, conversation, s.turnID, errText)
	return nil
}

func (s *aiConversationEventSink) ChatMessage(ctx context.Context, item serverai.AgenticChatMessage) error {
	visibility := clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT
	if item.UIOnly {
		visibility = clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY
	}
	role := strings.ToLower(strings.TrimSpace(item.Role))
	if role == "" {
		role = "assistant"
	}

	message := &clientpb.AIConversationMessage{
		ConversationID:   s.conversationID,
		OperatorName:     s.operatorName,
		Provider:         s.provider(),
		Model:            s.model(),
		Role:             role,
		Content:          strings.TrimSpace(item.Content),
		FinishReason:     strings.TrimSpace(item.Status),
		Kind:             clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
		Visibility:       visibility,
		IncludeInContext: aiOptionalBool(item.IncludeInContext),
		State:            normalizeAIMessageState(item.Status, clientpb.AIConversationMessageState_AI_MESSAGE_STATE_COMPLETED),
		TurnID:           s.turnID,
		ItemID:           strings.TrimSpace(item.ItemID),
	}
	_, err := s.saveMessageAndPublish(message, clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED, "")
	return err
}

func (s *aiConversationEventSink) ReasoningItem(ctx context.Context, item serverai.AgenticReasoningItem) error {
	content := strings.TrimSpace(item.Content)
	summary := strings.TrimSpace(item.Summary)
	if content == "" {
		content = summary
	} else if summary != "" && summary != content {
		content = "Summary:\n" + summary + "\n\n" + content
	}

	message := &clientpb.AIConversationMessage{
		ConversationID:   s.conversationID,
		OperatorName:     s.operatorName,
		Provider:         s.provider(),
		Model:            s.model(),
		Role:             "assistant",
		Content:          content,
		FinishReason:     strings.TrimSpace(item.Status),
		Kind:             clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_REASONING,
		Visibility:       clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY,
		IncludeInContext: aiOptionalBool(false),
		State:            normalizeAIMessageState(item.Status, clientpb.AIConversationMessageState_AI_MESSAGE_STATE_COMPLETED),
		TurnID:           s.turnID,
		ItemID:           strings.TrimSpace(item.ItemID),
	}
	_, err := s.saveMessageAndPublish(message, clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED, "")
	return err
}

func (s *aiConversationEventSink) ToolCallStarted(ctx context.Context, item serverai.AgenticToolCall) error {
	message := &clientpb.AIConversationMessage{
		ConversationID:   s.conversationID,
		OperatorName:     s.operatorName,
		Provider:         s.provider(),
		Model:            s.model(),
		Role:             "assistant",
		Kind:             clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_TOOL_CALL,
		Visibility:       clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY,
		IncludeInContext: aiOptionalBool(false),
		State:            clientpb.AIConversationMessageState_AI_MESSAGE_STATE_IN_PROGRESS,
		TurnID:           s.turnID,
		ItemID:           strings.TrimSpace(item.ItemID),
		ToolCallID:       strings.TrimSpace(item.CallID),
		ToolName:         strings.TrimSpace(item.Name),
		ToolArguments:    strings.TrimSpace(item.Arguments),
		FinishReason:     strings.TrimSpace(item.Status),
	}
	_, err := s.saveMessageAndPublish(message, clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_STARTED, "")
	return err
}

func (s *aiConversationEventSink) ToolCallCompleted(ctx context.Context, item serverai.AgenticToolCallResult) error {
	errText := ""
	state := clientpb.AIConversationMessageState_AI_MESSAGE_STATE_COMPLETED
	if item.Error != nil {
		errText = strings.TrimSpace(item.Error.Error())
		state = clientpb.AIConversationMessageState_AI_MESSAGE_STATE_FAILED
	}

	message := &clientpb.AIConversationMessage{
		ConversationID:   s.conversationID,
		OperatorName:     s.operatorName,
		Provider:         s.provider(),
		Model:            s.model(),
		Role:             "assistant",
		Kind:             clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_TOOL_CALL,
		Visibility:       clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY,
		IncludeInContext: aiOptionalBool(false),
		State:            state,
		TurnID:           s.turnID,
		ItemID:           strings.TrimSpace(item.ItemID),
		ToolCallID:       strings.TrimSpace(item.CallID),
		ToolName:         strings.TrimSpace(item.Name),
		ToolArguments:    strings.TrimSpace(item.Arguments),
		ToolResult:       strings.TrimSpace(item.Output),
		ErrorText:        errText,
		FinishReason:     strings.TrimSpace(item.Status),
	}
	_, err := s.saveMessageAndPublish(message, clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED, errText)
	return err
}

func (s *aiConversationEventSink) SaveFailureMessage(content string, errText string) error {
	message := &clientpb.AIConversationMessage{
		ConversationID:   s.conversationID,
		OperatorName:     s.operatorName,
		Provider:         s.provider(),
		Model:            s.model(),
		Role:             "system",
		Content:          strings.TrimSpace(content),
		FinishReason:     "error",
		Kind:             clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
		Visibility:       clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY,
		IncludeInContext: aiOptionalBool(false),
		State:            clientpb.AIConversationMessageState_AI_MESSAGE_STATE_FAILED,
		TurnID:           s.turnID,
		ErrorText:        strings.TrimSpace(errText),
	}
	_, err := s.saveMessageAndPublish(message, clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED, errText)
	return err
}

func (s *aiConversationEventSink) saveMessageAndPublish(
	message *clientpb.AIConversationMessage,
	eventType clientpb.AIConversationEventType,
	errText string,
) (*clientpb.AIConversationMessage, error) {
	savedMessage, err := db.SaveAIConversationMessage(message, s.operatorName)
	if err != nil {
		return nil, err
	}
	conversation, err := loadAIConversationSummary(s.conversationID)
	if err != nil {
		return nil, err
	}
	publishAIConversationMessageEvent(eventType, conversation, savedMessage, s.turnID, errText)
	return savedMessage, nil
}

func (s *aiConversationEventSink) updateConversationTurnState(activeTurnID string, state clientpb.AIConversationTurnState) (*clientpb.AIConversation, error) {
	conversation, err := db.AIConversationByID(s.conversationID, "", false)
	if err != nil {
		return nil, err
	}
	conversation.ActiveTurnID = strings.TrimSpace(activeTurnID)
	conversation.TurnState = state
	if conversation.GetProvider() == "" {
		conversation.Provider = s.provider()
	}
	if conversation.GetModel() == "" {
		conversation.Model = s.model()
	}
	if _, err := db.SaveAIConversation(conversation, s.operatorName); err != nil {
		return nil, err
	}
	return loadAIConversationSummary(s.conversationID)
}

func (s *aiConversationEventSink) provider() string {
	if s != nil && s.runtime != nil {
		return strings.TrimSpace(s.runtime.Provider)
	}
	return ""
}

func (s *aiConversationEventSink) model() string {
	if s != nil && s.runtime != nil {
		return strings.TrimSpace(s.runtime.Model)
	}
	return ""
}

func normalizeAIMessageState(status string, fallback clientpb.AIConversationMessageState) clientpb.AIConversationMessageState {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "in_progress":
		return clientpb.AIConversationMessageState_AI_MESSAGE_STATE_IN_PROGRESS
	case "incomplete", "failed", "error":
		return clientpb.AIConversationMessageState_AI_MESSAGE_STATE_FAILED
	case "completed":
		return clientpb.AIConversationMessageState_AI_MESSAGE_STATE_COMPLETED
	default:
		return fallback
	}
}

func aiOptionalBool(value bool) *bool {
	return &value
}
