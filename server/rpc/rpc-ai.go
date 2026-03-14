package rpc

import (
	"context"
	"fmt"
	"strings"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	serverai "github.com/bishopfox/sliver/server/ai"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
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

func publishAIConversationEvent(conversation *clientpb.AIConversation) {
	if conversation == nil {
		return
	}

	data, err := proto.Marshal(conversation)
	if err != nil {
		rpcLog.Warnf("Failed to marshal AI conversation event: %s", err)
		return
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.AIConversationEvent,
		Data:      data,
	})
}

func (rpc *Server) GetAIProviders(ctx context.Context, _ *commonpb.Empty) (*clientpb.AIProviderConfigs, error) {
	return &clientpb.AIProviderConfigs{
		Providers: serverai.ConfiguredProviders(),
		Config:    serverai.SafeConfigSummary(),
	}, nil
}

func (rpc *Server) GetAIConversations(ctx context.Context, _ *commonpb.Empty) (*clientpb.AIConversations, error) {
	conversations, err := db.AIConversationsByOperator(rpc.aiOperatorName(ctx, ""))
	if err != nil {
		return nil, rpcError(err)
	}
	return &clientpb.AIConversations{Conversations: conversations}, nil
}

func (rpc *Server) GetAIConversation(ctx context.Context, req *clientpb.AIConversationReq) (*clientpb.AIConversation, error) {
	if req == nil || strings.TrimSpace(req.ID) == "" {
		return nil, status.Error(codes.InvalidArgument, "missing AI conversation id")
	}

	conversation, err := db.AIConversationByID(req.ID, rpc.aiOperatorName(ctx, ""), req.IncludeMessages)
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

	req.OperatorName = rpc.aiOperatorName(ctx, req.OperatorName)
	conversation, err := db.SaveAIConversation(req, req.OperatorName)
	if err != nil {
		return nil, rpcError(err)
	}
	publishAIConversationEvent(conversation)
	return conversation, nil
}

func (rpc *Server) DeleteAIConversation(ctx context.Context, req *clientpb.AIConversationReq) (*commonpb.Empty, error) {
	if req == nil || strings.TrimSpace(req.ID) == "" {
		return nil, status.Error(codes.InvalidArgument, "missing AI conversation id")
	}

	operatorName := rpc.aiOperatorName(ctx, "")
	err := db.DeleteAIConversation(req.ID, operatorName)
	if err != nil {
		return nil, rpcError(err)
	}
	publishAIConversationEvent(&clientpb.AIConversation{
		ID:           req.ID,
		OperatorName: operatorName,
	})
	return &commonpb.Empty{}, nil
}

func (rpc *Server) GetAIConversationMessages(ctx context.Context, req *clientpb.AIConversationReq) (*clientpb.AIConversationMessages, error) {
	if req == nil || strings.TrimSpace(req.ID) == "" {
		return nil, status.Error(codes.InvalidArgument, "missing AI conversation id")
	}

	messages, err := db.AIConversationMessagesByID(req.ID, rpc.aiOperatorName(ctx, ""))
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

	conversation, err := db.AIConversationByID(message.ConversationID, req.OperatorName, false)
	if err != nil {
		return nil, rpcError(err)
	}
	publishAIConversationEvent(conversation)
	if shouldTriggerAICompletion(req, message) {
		go runAIConversationCompletion(message.GetConversationID(), message.GetOperatorName())
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

func runAIConversationCompletion(conversationID string, operatorName string) {
	conversation, err := db.AIConversationByID(conversationID, operatorName, true)
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
	if resolveErr != nil {
		persistAIConversationFailure(conversationID, operatorName, runtime, resolveErr)
		return
	}

	completion, err := serverai.CompleteConversation(context.Background(), runtime, conversation)
	if err != nil {
		persistAIConversationFailure(conversationID, operatorName, runtime, err)
		return
	}

	if _, err := db.SaveAIConversationMessage(&clientpb.AIConversationMessage{
		ConversationID:    conversationID,
		OperatorName:      operatorName,
		Provider:          completion.Provider,
		Model:             completion.Model,
		Role:              "assistant",
		Content:           completion.Content,
		ProviderMessageID: completion.ProviderMessageID,
		FinishReason:      completion.FinishReason,
	}, operatorName); err != nil {
		rpcLog.Warnf("Failed to save AI assistant reply for %q: %s", conversationID, err)
		return
	}

	publishAIConversationByID(conversationID, operatorName)
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
		ID:           conversation.GetID(),
		OperatorName: conversation.GetOperatorName(),
		Provider:     provider,
		Model:        model,
		Title:        conversation.GetTitle(),
		Summary:      conversation.GetSummary(),
		SystemPrompt: conversation.GetSystemPrompt(),
	}, operatorName)
}

func persistAIConversationFailure(conversationID string, operatorName string, runtime *serverai.RuntimeConfig, failure error) {
	content := "AI request failed."
	if failure != nil && strings.TrimSpace(failure.Error()) != "" {
		content = "AI request failed: " + strings.TrimSpace(failure.Error())
	}

	message := &clientpb.AIConversationMessage{
		ConversationID: conversationID,
		OperatorName:   operatorName,
		Role:           "system",
		Content:        content,
		FinishReason:   "error",
	}
	if runtime != nil {
		message.Provider = runtime.Provider
		message.Model = runtime.Model
	}

	if _, err := db.SaveAIConversationMessage(message, operatorName); err != nil {
		rpcLog.Warnf("Failed to save AI failure message for %q: %s", conversationID, err)
		return
	}

	publishAIConversationByID(conversationID, operatorName)
}

func publishAIConversationByID(conversationID string, operatorName string) {
	conversation, err := db.AIConversationByID(conversationID, operatorName, false)
	if err != nil {
		rpcLog.Warnf("Failed to publish AI conversation update for %q: %s", conversationID, err)
		return
	}
	publishAIConversationEvent(conversation)
}

func fallbackAIString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}
