package ai

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"net"
	"sync"
	"testing"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

func TestLoadAIStateCmdLoadsConversationOverRPC(t *testing.T) {
	server := &aiRPCServer{
		providersResp: &clientpb.AIProviderConfigs{
			Providers: []*clientpb.AIProviderConfig{{Name: "openai", Configured: true}},
			Config: &clientpb.AIConfigSummary{
				Provider: "openai",
				Model:    "gpt-test",
				Valid:    true,
			},
		},
		conversationsResp: &clientpb.AIConversations{
			Conversations: []*clientpb.AIConversation{
				{ID: "conv-1", Provider: "openai", Model: "gpt-test", Title: "First"},
				{ID: "conv-2", Provider: "openai", Model: "gpt-test", Title: "Second"},
			},
		},
		conversationByID: map[string]*clientpb.AIConversation{
			"conv-1": {
				ID:       "conv-1",
				Provider: "openai",
				Model:    "gpt-test",
				Title:    "First",
				Messages: []*clientpb.AIConversationMessage{
					{Role: "user", Content: "Hello"},
				},
			},
		},
	}

	rpcClient, cleanup := newAITestRPCClient(t, server)
	defer cleanup()

	cmd := loadAIStateCmd(&console.SliverClient{Rpc: rpcClient}, aiTargetSummary{}, "")
	msg := cmd()

	loaded, ok := msg.(aiLoadedMsg)
	if !ok {
		t.Fatalf("expected aiLoadedMsg, got %T", msg)
	}
	if loaded.selectedID != "conv-1" {
		t.Fatalf("unexpected selected conversation id: got=%q want=%q", loaded.selectedID, "conv-1")
	}
	if loaded.conversation == nil || loaded.conversation.GetID() != "conv-1" {
		t.Fatalf("unexpected loaded conversation: %+v", loaded.conversation)
	}
	if len(loaded.conversations) != 2 {
		t.Fatalf("unexpected conversation count: got=%d want=%d", len(loaded.conversations), 2)
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	if server.getProvidersCalls != 1 {
		t.Fatalf("expected GetAIProviders to be called once, got %d", server.getProvidersCalls)
	}
	if server.getConversationsCalls != 1 {
		t.Fatalf("expected GetAIConversations to be called once, got %d", server.getConversationsCalls)
	}
	if len(server.getConversationReqs) != 1 {
		t.Fatalf("expected GetAIConversation to be called once, got %d", len(server.getConversationReqs))
	}
	if server.getConversationReqs[0].GetID() != "conv-1" || !server.getConversationReqs[0].GetIncludeMessages() {
		t.Fatalf("unexpected GetAIConversation request: %+v", server.getConversationReqs[0])
	}
}

func TestLoadAIStateCmdCreatesConversationWhenNoneExist(t *testing.T) {
	server := &aiRPCServer{
		providersResp: &clientpb.AIProviderConfigs{
			Providers: []*clientpb.AIProviderConfig{{Name: "openai", Configured: true}},
			Config: &clientpb.AIConfigSummary{
				Provider: "openai",
				Model:    "gpt-test",
				Valid:    true,
			},
		},
		conversationsResp: &clientpb.AIConversations{},
		saveConversationResp: &clientpb.AIConversation{
			ID:       "conv-created",
			Provider: "openai",
			Model:    "gpt-test",
			Title:    "New conversation",
		},
		conversationByID: map[string]*clientpb.AIConversation{
			"conv-created": {
				ID:       "conv-created",
				Provider: "openai",
				Model:    "gpt-test",
				Title:    "New conversation",
			},
		},
	}

	rpcClient, cleanup := newAITestRPCClient(t, server)
	defer cleanup()

	msg := loadAIStateCmd(&console.SliverClient{Rpc: rpcClient}, aiTargetSummary{}, "")()

	loaded, ok := msg.(aiLoadedMsg)
	if !ok {
		t.Fatalf("expected aiLoadedMsg, got %T", msg)
	}
	if loaded.status != "Created a new AI conversation." {
		t.Fatalf("unexpected status: %q", loaded.status)
	}
	if loaded.selectedID != "conv-created" {
		t.Fatalf("unexpected selected conversation id: %q", loaded.selectedID)
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.saveConversationReqs) != 1 {
		t.Fatalf("expected SaveAIConversation to be called once, got %d", len(server.saveConversationReqs))
	}
	request := server.saveConversationReqs[0]
	if request.GetProvider() != "openai" || request.GetModel() != "gpt-test" || request.GetTitle() != "New conversation" {
		t.Fatalf("unexpected SaveAIConversation request: %+v", request)
	}
}

func TestSubmitPromptCmdCreatesConversationAndSavesUserMessage(t *testing.T) {
	server := &aiRPCServer{
		saveConversationResp: &clientpb.AIConversation{
			ID:       "conv-created",
			Provider: "openai",
			Model:    "gpt-test",
			Title:    "Explain the workflow.",
		},
		saveMessageResp: &clientpb.AIConversationMessage{
			ID:             "msg-1",
			ConversationID: "conv-created",
			Role:           "user",
			Content:        "Explain the workflow.\n\nWith details.",
		},
	}

	rpcClient, cleanup := newAITestRPCClient(t, server)
	defer cleanup()

	msg := submitPromptCmd(
		&console.SliverClient{Rpc: rpcClient},
		aiTargetSummary{SessionID: "session-1"},
		nil,
		"openai",
		"gpt-test",
		"Explain the workflow.\n\nWith details.",
	)()

	submitted, ok := msg.(aiPromptSubmittedMsg)
	if !ok {
		t.Fatalf("expected aiPromptSubmittedMsg, got %T", msg)
	}
	if submitted.conversationID != "conv-created" {
		t.Fatalf("unexpected conversation id: %q", submitted.conversationID)
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.saveConversationReqs) != 1 {
		t.Fatalf("expected SaveAIConversation to be called once, got %d", len(server.saveConversationReqs))
	}
	if server.saveConversationReqs[0].GetTitle() != "Explain the workflow." {
		t.Fatalf("unexpected conversation title: %q", server.saveConversationReqs[0].GetTitle())
	}
	if server.saveConversationReqs[0].GetTargetSessionID() != "session-1" {
		t.Fatalf("expected target session to be forwarded, got %+v", server.saveConversationReqs[0])
	}
	if len(server.saveMessageReqs) != 1 {
		t.Fatalf("expected SaveAIConversationMessage to be called once, got %d", len(server.saveMessageReqs))
	}
	request := server.saveMessageReqs[0]
	if request.GetConversationID() != "conv-created" || request.GetProvider() != "openai" || request.GetModel() != "gpt-test" || request.GetRole() != "user" {
		t.Fatalf("unexpected SaveAIConversationMessage request: %+v", request)
	}
	if request.GetContent() != "Explain the workflow.\n\nWith details." {
		t.Fatalf("unexpected prompt content: %q", request.GetContent())
	}
}

func TestSubmitPromptCmdUsesExistingConversationSettings(t *testing.T) {
	server := &aiRPCServer{
		saveMessageResp: &clientpb.AIConversationMessage{
			ID:             "msg-1",
			ConversationID: "conv-1",
			Role:           "user",
			Content:        "hello",
		},
	}

	rpcClient, cleanup := newAITestRPCClient(t, server)
	defer cleanup()

	msg := submitPromptCmd(
		&console.SliverClient{Rpc: rpcClient},
		aiTargetSummary{},
		&clientpb.AIConversation{ID: "conv-1", Provider: "anthropic", Model: "claude-test", Title: "Thread"},
		"openai",
		"gpt-test",
		"hello",
	)()

	if _, ok := msg.(aiPromptSubmittedMsg); !ok {
		t.Fatalf("expected aiPromptSubmittedMsg, got %T", msg)
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.saveConversationReqs) != 0 {
		t.Fatalf("did not expect SaveAIConversation to be called, got %d calls", len(server.saveConversationReqs))
	}
	if len(server.saveMessageReqs) != 1 {
		t.Fatalf("expected SaveAIConversationMessage to be called once, got %d", len(server.saveMessageReqs))
	}
	request := server.saveMessageReqs[0]
	if request.GetProvider() != "anthropic" || request.GetModel() != "claude-test" {
		t.Fatalf("unexpected conversation settings in message request: %+v", request)
	}
}

func TestDeleteConversationCmdSendsDeleteRequest(t *testing.T) {
	server := &aiRPCServer{}

	rpcClient, cleanup := newAITestRPCClient(t, server)
	defer cleanup()

	msg := deleteConversationCmd(
		&console.SliverClient{Rpc: rpcClient},
		"conv-1",
		"conv-2",
		`Deleted "Thread".`,
	)()

	deleted, ok := msg.(aiConversationDeletedMsg)
	if !ok {
		t.Fatalf("expected aiConversationDeletedMsg, got %T", msg)
	}
	if deleted.conversationID != "conv-1" || deleted.selectedID != "conv-2" {
		t.Fatalf("unexpected delete message: %+v", deleted)
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.deleteConversationReqs) != 1 {
		t.Fatalf("expected DeleteAIConversation to be called once, got %d", len(server.deleteConversationReqs))
	}
	if server.deleteConversationReqs[0].GetID() != "conv-1" {
		t.Fatalf("unexpected DeleteAIConversation request: %+v", server.deleteConversationReqs[0])
	}
}

func TestWaitForAIConversationEventCmdFiltersForAIEvents(t *testing.T) {
	eventPayload := &clientpb.AIConversationEvent{
		EventType: clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_CONVERSATION_UPDATED,
		Conversation: &clientpb.AIConversation{
			ID:           "conv-1",
			OperatorName: "alice",
		},
	}
	data, err := proto.Marshal(eventPayload)
	if err != nil {
		t.Fatalf("marshal conversation event: %v", err)
	}

	listener := make(chan *clientpb.Event, 2)
	listener <- &clientpb.Event{EventType: consts.WatchtowerEvent}
	listener <- &clientpb.Event{EventType: consts.AIConversationEvent, Data: data}

	msg := waitForAIConversationEventCmd(listener)()

	aiEvent, ok := msg.(aiConversationEventMsg)
	if !ok {
		t.Fatalf("expected aiConversationEventMsg, got %T", msg)
	}
	if aiEvent.event == nil || aiEvent.event.GetConversation() == nil || aiEvent.event.GetConversation().GetID() != "conv-1" {
		t.Fatalf("unexpected AI conversation event payload: %+v", aiEvent.event)
	}
}

func TestWaitForAIConversationEventCmdReturnsClosedMessage(t *testing.T) {
	listener := make(chan *clientpb.Event)
	close(listener)

	msg := waitForAIConversationEventCmd(listener)()
	if _, ok := msg.(aiListenerClosedMsg); !ok {
		t.Fatalf("expected aiListenerClosedMsg, got %T", msg)
	}
}

func TestAIModelAppliesSharedConversationEventsWithoutReload(t *testing.T) {
	listener := make(chan *clientpb.Event)
	close(listener)

	model := newAIModel(nil, aiContext{
		connection: aiConnectionSummary{Operator: "alice"},
	}, listener)
	model.loading = false
	model.conversations = []*clientpb.AIConversation{{ID: "conv-1", Title: "First"}}

	updated, cmd := model.Update(aiConversationEventMsg{event: &clientpb.AIConversationEvent{
		EventType: clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_CONVERSATION_UPDATED,
		Conversation: &clientpb.AIConversation{
			ID:           "conv-2",
			OperatorName: "bob",
			Title:        "Shared thread",
		},
	}})

	nextModel := updated.(*aiModel)
	if nextModel.loading {
		t.Fatal("did not expect shared conversation event to trigger a reload")
	}
	if conversationIndexByID(nextModel.conversations, "conv-2") < 0 {
		t.Fatalf("expected shared conversation to be merged locally, got %+v", nextModel.conversations)
	}
	if cmd == nil {
		t.Fatal("expected shared conversation event to continue listening for events")
	}
}

func TestAIModelAppliesDeleteEventsWhileAwaitingResponse(t *testing.T) {
	listener := make(chan *clientpb.Event)
	close(listener)

	model := newAIModel(nil, aiContext{
		connection: aiConnectionSummary{Operator: "alice"},
	}, listener)
	model.loading = false
	model.awaitingResponse = true
	model.currentConversation = &clientpb.AIConversation{
		ID:           "conv-2",
		OperatorName: "alice",
		UpdatedAt:    100,
		TurnState:    clientpb.AIConversationTurnState_AI_TURN_STATE_IN_PROGRESS,
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "still waiting"},
		},
	}
	model.conversations = []*clientpb.AIConversation{
		{ID: "conv-2", Title: "Current"},
		{ID: "conv-1", Title: "Fallback"},
	}

	updated, cmd := model.Update(aiConversationEventMsg{event: &clientpb.AIConversationEvent{
		EventType:      clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_CONVERSATION_DELETED,
		ConversationID: "conv-2",
		Conversation: &clientpb.AIConversation{
			ID: "conv-2",
		},
	}})

	nextModel := updated.(*aiModel)
	if nextModel.loading {
		t.Fatal("did not expect delete tombstone to trigger a reload")
	}
	if nextModel.currentConversation != nil {
		t.Fatalf("expected deleted conversation to be cleared, got %+v", nextModel.currentConversation)
	}
	if conversationIndexByID(nextModel.conversations, "conv-2") >= 0 {
		t.Fatalf("expected deleted conversation to be removed, got %+v", nextModel.conversations)
	}
	if cmd == nil {
		t.Fatal("expected delete tombstone to continue listening for events")
	}
}

type aiRPCServer struct {
	rpcpb.UnimplementedSliverRPCServer

	mu sync.Mutex

	providersResp        *clientpb.AIProviderConfigs
	conversationsResp    *clientpb.AIConversations
	conversationByID     map[string]*clientpb.AIConversation
	saveConversationResp *clientpb.AIConversation
	saveMessageResp      *clientpb.AIConversationMessage

	getProvidersCalls      int
	getConversationsCalls  int
	getConversationReqs    []*clientpb.AIConversationReq
	deleteConversationReqs []*clientpb.AIConversationReq
	saveConversationReqs   []*clientpb.AIConversation
	saveMessageReqs        []*clientpb.AIConversationMessage
	getConversationStart   chan struct{}
	getConversationWait    <-chan struct{}
	saveMessageStarted     chan struct{}
	saveMessageRelease     <-chan struct{}
}

func (s *aiRPCServer) GetAIProviders(context.Context, *commonpb.Empty) (*clientpb.AIProviderConfigs, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getProvidersCalls++
	if s.providersResp == nil {
		return &clientpb.AIProviderConfigs{}, nil
	}
	return proto.Clone(s.providersResp).(*clientpb.AIProviderConfigs), nil
}

func (s *aiRPCServer) GetAIConversations(context.Context, *commonpb.Empty) (*clientpb.AIConversations, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getConversationsCalls++
	if s.conversationsResp == nil {
		return &clientpb.AIConversations{}, nil
	}
	return proto.Clone(s.conversationsResp).(*clientpb.AIConversations), nil
}

func (s *aiRPCServer) GetAIConversation(_ context.Context, req *clientpb.AIConversationReq) (*clientpb.AIConversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getConversationReqs = append(s.getConversationReqs, proto.Clone(req).(*clientpb.AIConversationReq))
	if s.getConversationStart != nil {
		select {
		case s.getConversationStart <- struct{}{}:
		default:
		}
	}
	if s.getConversationWait != nil {
		s.mu.Unlock()
		<-s.getConversationWait
		s.mu.Lock()
	}
	if s.conversationByID == nil {
		return &clientpb.AIConversation{}, nil
	}
	conversation := s.conversationByID[req.GetID()]
	if conversation == nil {
		return &clientpb.AIConversation{}, nil
	}
	return proto.Clone(conversation).(*clientpb.AIConversation), nil
}

func (s *aiRPCServer) SaveAIConversation(_ context.Context, req *clientpb.AIConversation) (*clientpb.AIConversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saveConversationReqs = append(s.saveConversationReqs, proto.Clone(req).(*clientpb.AIConversation))
	if s.saveConversationResp == nil {
		return proto.Clone(req).(*clientpb.AIConversation), nil
	}
	return proto.Clone(s.saveConversationResp).(*clientpb.AIConversation), nil
}

func (s *aiRPCServer) DeleteAIConversation(_ context.Context, req *clientpb.AIConversationReq) (*commonpb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteConversationReqs = append(s.deleteConversationReqs, proto.Clone(req).(*clientpb.AIConversationReq))
	return &commonpb.Empty{}, nil
}

func (s *aiRPCServer) SaveAIConversationMessage(_ context.Context, req *clientpb.AIConversationMessage) (*clientpb.AIConversationMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saveMessageReqs = append(s.saveMessageReqs, proto.Clone(req).(*clientpb.AIConversationMessage))
	if s.saveMessageStarted != nil {
		select {
		case s.saveMessageStarted <- struct{}{}:
		default:
		}
	}
	if s.saveMessageRelease != nil {
		s.mu.Unlock()
		<-s.saveMessageRelease
		s.mu.Lock()
	}
	if s.saveMessageResp == nil {
		return proto.Clone(req).(*clientpb.AIConversationMessage), nil
	}
	return proto.Clone(s.saveMessageResp).(*clientpb.AIConversationMessage), nil
}

func newAITestRPCClient(t *testing.T, srv rpcpb.SliverRPCServer) (rpcpb.SliverRPCClient, func()) {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	rpcpb.RegisterSliverRPCServer(grpcServer, srv)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	dialer := func(context.Context, string) (net.Conn, error) { return listener.Dial() }
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		grpcServer.Stop()
		_ = listener.Close()
		t.Fatalf("dial bufconn: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return rpcpb.NewSliverRPCClient(conn), cleanup
}
