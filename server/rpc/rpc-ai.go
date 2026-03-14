package rpc

import (
	"context"
	"fmt"
	"strings"

	serverai "github.com/bishopfox/sliver/server/ai"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/db"
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

func (rpc *Server) GetAIProviders(ctx context.Context, _ *commonpb.Empty) (*clientpb.AIProviderConfigs, error) {
	return &clientpb.AIProviderConfigs{
		Providers: serverai.ConfiguredProviders(),
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
	return conversation, nil
}

func (rpc *Server) DeleteAIConversation(ctx context.Context, req *clientpb.AIConversationReq) (*commonpb.Empty, error) {
	if req == nil || strings.TrimSpace(req.ID) == "" {
		return nil, status.Error(codes.InvalidArgument, "missing AI conversation id")
	}

	err := db.DeleteAIConversation(req.ID, rpc.aiOperatorName(ctx, ""))
	if err != nil {
		return nil, rpcError(err)
	}
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
	return message, nil
}
