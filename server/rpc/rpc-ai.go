package rpc

import (
	"context"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	serverai "github.com/bishopfox/sliver/server/ai"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/rpc/aitools"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (rpc *Server) aiOperatorName(ctx context.Context, fallback string) string {
	operatorName := strings.TrimSpace(rpc.getClientCommonName(ctx))
	if operatorName != "" {
		return operatorName
	}
	return strings.TrimSpace(fallback)
}

func validateAIProvider(provider string) error {
	if strings.TrimSpace(provider) == "" {
		return status.Error(codes.InvalidArgument, "missing AI provider")
	}
	if !serverai.IsSupportedProvider(provider) {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("unsupported AI provider %q", provider))
	}
	return nil
}

func validateAIThinkingLevel(thinkingLevel string) error {
	thinkingLevel = strings.TrimSpace(thinkingLevel)
	if thinkingLevel == "" {
		return nil
	}
	if serverai.NormalizeThinkingLevelValue(thinkingLevel) == "" {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("unsupported AI thinking level %q", thinkingLevel))
	}
	return nil
}

func (rpc *Server) GetAIProviders(ctx context.Context, _ *commonpb.Empty) (*clientpb.AIProviderConfigs, error) {
	return &clientpb.AIProviderConfigs{
		Providers: serverai.ConfiguredProviders(),
		Config:    serverai.SafeConfigSummary(),
	}, nil
}

func (rpc *Server) GetAIConversations(ctx context.Context, _ *commonpb.Empty) (*clientpb.AIConversations, error) {
	conversations, err := db.AIConversationsByOperator("")
	if err != nil {
		return nil, rpcError(err)
	}
	return &clientpb.AIConversations{Conversations: conversations}, nil
}

func (rpc *Server) GetAIConversation(ctx context.Context, req *clientpb.AIConversationReq) (*clientpb.AIConversation, error) {
	if req == nil || strings.TrimSpace(req.ID) == "" {
		return nil, status.Error(codes.InvalidArgument, "missing AI conversation id")
	}

	conversation, err := db.AIConversationByID(req.ID, "", req.IncludeMessages)
	if err != nil {
		return nil, rpcError(err)
	}
	return conversation, nil
}

func (rpc *Server) SaveAIConversation(ctx context.Context, req *clientpb.AIConversation) (*clientpb.AIConversation, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing AI conversation")
	}
	if err := validateAIProvider(req.Provider); err != nil {
		return nil, err
	}
	if err := validateAIThinkingLevel(req.ThinkingLevel); err != nil {
		return nil, err
	}

	req.OperatorName = rpc.aiOperatorName(ctx, req.OperatorName)
	conversation, err := db.SaveAIConversation(req, req.OperatorName)
	if err != nil {
		return nil, rpcError(err)
	}
	publishAIConversationSummaryEvent(clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_CONVERSATION_UPDATED, conversation, "", "")
	return conversation, nil
}

func (rpc *Server) DeleteAIConversation(ctx context.Context, req *clientpb.AIConversationReq) (*commonpb.Empty, error) {
	if req == nil || strings.TrimSpace(req.ID) == "" {
		return nil, status.Error(codes.InvalidArgument, "missing AI conversation id")
	}

	operatorName := rpc.aiOperatorName(ctx, "")
	err := db.DeleteAIConversation(req.ID, "")
	if err != nil {
		return nil, rpcError(err)
	}
	publishAIConversationEvent(&clientpb.AIConversationEvent{
		EventType:      clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_CONVERSATION_DELETED,
		ConversationID: req.ID,
		Conversation: &clientpb.AIConversation{
			ID:           req.ID,
			OperatorName: operatorName,
		},
	})
	return &commonpb.Empty{}, nil
}

func (rpc *Server) GetAIConversationMessages(ctx context.Context, req *clientpb.AIConversationReq) (*clientpb.AIConversationMessages, error) {
	if req == nil || strings.TrimSpace(req.ID) == "" {
		return nil, status.Error(codes.InvalidArgument, "missing AI conversation id")
	}

	messages, err := db.AIConversationMessagesByID(req.ID, "")
	if err != nil {
		return nil, rpcError(err)
	}
	return messages, nil
}

func (rpc *Server) SaveAIConversationMessage(ctx context.Context, req *clientpb.AIConversationMessage) (*clientpb.AIConversationMessage, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing AI conversation message")
	}
	if strings.TrimSpace(req.ConversationID) == "" {
		return nil, status.Error(codes.InvalidArgument, "missing AI conversation id")
	}
	if strings.TrimSpace(req.Role) == "" {
		return nil, status.Error(codes.InvalidArgument, "missing AI conversation role")
	}
	if req.Provider != "" {
		if err := validateAIProvider(req.Provider); err != nil {
			return nil, err
		}
	}

	req.OperatorName = rpc.aiOperatorName(ctx, req.OperatorName)
	message, err := db.SaveAIConversationMessage(req, req.OperatorName)
	if err != nil {
		return nil, rpcError(err)
	}

	conversation, err := loadAIConversationSummary(message.GetConversationID())
	if err != nil {
		return nil, rpcError(err)
	}
	publishAIConversationMessageEvent(
		clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED,
		conversation,
		message,
		message.GetTurnID(),
		message.GetErrorText(),
	)
	if shouldTriggerAICompletion(req, message) {
		go rpc.runAIConversationCompletion(message.GetConversationID(), message.GetOperatorName())
	}

	return message, nil
}

func shouldTriggerAICompletion(req *clientpb.AIConversationMessage, saved *clientpb.AIConversationMessage) bool {
	if req == nil || saved == nil {
		return false
	}
	if strings.TrimSpace(req.GetID()) != "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(saved.GetRole()), "user")
}

func (rpc *Server) runAIConversationCompletion(conversationID string, operatorName string) {
	conversation, err := db.AIConversationByID(conversationID, "", true)
	if err != nil {
		rpcLog.Warnf("Failed to load AI conversation %q for completion: %s", conversationID, err)
		return
	}

	runtime, resolveErr := serverai.ResolveRuntimeConfig(configs.GetServerConfig(), conversation)
	if runtime != nil && (runtime.Provider != "" || runtime.Model != "") {
		if updatedConversation, err := persistAIConversationRuntime(conversation, operatorName, runtime); err != nil {
			rpcLog.Warnf("Failed to persist AI conversation runtime for %q: %s", conversationID, err)
		} else if updatedConversation != nil {
			conversation = updatedConversation
		}
	}
	turnID, err := newAITurnID()
	if err != nil {
		rpcLog.Warnf("Failed to create AI turn id for %q: %s", conversationID, err)
		return
	}
	sink := newAIConversationEventSink(conversationID, operatorName, runtime, turnID)
	if err := sink.TurnStarted(); err != nil {
		rpcLog.Warnf("Failed to mark AI turn %q as started for %q: %s", turnID, conversationID, err)
		return
	}

	if resolveErr != nil {
		persistAIConversationFailure(sink, resolveErr)
		if err := sink.TurnFailed(resolveErr); err != nil {
			rpcLog.Warnf("Failed to mark AI turn %q as failed for %q: %s", turnID, conversationID, err)
		}
		return
	}

	var completion *serverai.Completion
	if serverai.SupportsAgenticConversation(runtime) {
		completion, err = serverai.CompleteConversationAgentic(
			context.Background(),
			runtime,
			conversation,
			aitools.NewExecutor(rpc, conversation),
			sink,
		)
	} else {
		completion, err = serverai.CompleteConversation(context.Background(), runtime, conversation)
	}
	if err != nil {
		persistAIConversationFailure(sink, err)
		if turnErr := sink.TurnFailed(err); turnErr != nil {
			rpcLog.Warnf("Failed to mark AI turn %q as failed for %q: %s", turnID, conversationID, turnErr)
		}
		return
	}

	message, err := db.SaveAIConversationMessage(&clientpb.AIConversationMessage{
		ConversationID:    conversationID,
		OperatorName:      operatorName,
		Provider:          completion.Provider,
		Model:             completion.Model,
		Role:              "assistant",
		Content:           completion.Content,
		ProviderMessageID: completion.ProviderMessageID,
		FinishReason:      completion.FinishReason,
		Kind:              clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
		Visibility:        clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
		IncludeInContext:  aiOptionalBool(true),
		State:             clientpb.AIConversationMessageState_AI_MESSAGE_STATE_COMPLETED,
		TurnID:            turnID,
	}, operatorName)
	if err != nil {
		rpcLog.Warnf("Failed to save AI assistant reply for %q: %s", conversationID, err)
		return
	}

	summary, err := loadAIConversationSummary(conversationID)
	if err != nil {
		rpcLog.Warnf("Failed to load AI conversation summary for %q: %s", conversationID, err)
	} else {
		publishAIConversationMessageEvent(
			clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED,
			summary,
			message,
			turnID,
			"",
		)
	}
	if err := sink.TurnCompleted(); err != nil {
		rpcLog.Warnf("Failed to mark AI turn %q as completed for %q: %s", turnID, conversationID, err)
	}
}

func persistAIConversationRuntime(conversation *clientpb.AIConversation, operatorName string, runtime *serverai.RuntimeConfig) (*clientpb.AIConversation, error) {
	if conversation == nil || runtime == nil {
		return conversation, nil
	}
	if strings.TrimSpace(runtime.Provider) == "" && strings.TrimSpace(runtime.Model) == "" {
		return conversation, nil
	}

	provider := fallbackAIString(conversation.GetProvider(), runtime.Provider)
	model := fallbackAIString(conversation.GetModel(), runtime.Model)
	if provider == conversation.GetProvider() && model == conversation.GetModel() {
		return conversation, nil
	}

	return db.SaveAIConversation(&clientpb.AIConversation{
		ID:              conversation.GetID(),
		OperatorName:    conversation.GetOperatorName(),
		Provider:        provider,
		Model:           model,
		ThinkingLevel:   conversation.GetThinkingLevel(),
		Title:           conversation.GetTitle(),
		Summary:         conversation.GetSummary(),
		SystemPrompt:    conversation.GetSystemPrompt(),
		ActiveTurnID:    conversation.GetActiveTurnID(),
		TurnState:       conversation.GetTurnState(),
		TargetSessionID: conversation.GetTargetSessionID(),
		TargetBeaconID:  conversation.GetTargetBeaconID(),
	}, operatorName)
}

func persistAIConversationFailure(sink *aiConversationEventSink, failure error) {
	content := "AI request failed."
	errText := ""
	if failure != nil && strings.TrimSpace(failure.Error()) != "" {
		errText = strings.TrimSpace(failure.Error())
		content = "AI request failed: " + errText
	}
	if sink == nil {
		return
	}
	if err := sink.SaveFailureMessage(content, errText); err != nil {
		rpcLog.Warnf("Failed to save AI failure message for %q: %s", sink.conversationID, err)
	}
}

func newAITurnID() (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func fallbackAIString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}
