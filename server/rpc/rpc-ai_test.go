package rpc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	serverai "github.com/bishopfox/sliver/server/ai"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSaveAIConversationMessageCompletesConversationAndPublishesEvents(t *testing.T) {
	setupAIRPCTestEnv(t)

	requests := make(chan struct{}, 1)
	restoreClient := serverai.SetHTTPClientForTests(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("unexpected provider request path: %q", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer openai-key" {
				t.Fatalf("unexpected authorization header: %q", r.Header.Get("Authorization"))
			}
			requests <- struct{}{}
			return jsonResponse(http.StatusOK, `{
			"id": "resp_success",
			"model": "gpt-5.2",
			"status": "completed",
			"output": [
				{
					"type": "message",
					"role": "assistant",
					"content": [
						{"type": "output_text", "text": "Assistant reply from the provider"}
					]
				}
			]
		}`), nil
		}),
	})
	defer restoreClient()

	saveOpenAICompletionConfig(t, "gpt-5.2", "high", "openai-key", "https://openai.example/v1")

	client, cleanup := newBufnetRPCClient(t)
	defer cleanup()

	streamCtx, cancelStream := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStream()
	eventStream, err := client.Events(streamCtx, &commonpb.Empty{})
	if err != nil {
		t.Fatalf("start events stream: %v", err)
	}

	conversation, err := client.SaveAIConversation(context.Background(), &clientpb.AIConversation{
		OperatorName: "alice",
		Provider:     serverai.ProviderOpenAI,
		Title:        "Workflow test",
	})
	if err != nil {
		t.Fatalf("save conversation: %v", err)
	}

	waitForAIConversationEvent(t, eventStream, conversation.GetID())

	savedUserMessage, err := client.SaveAIConversationMessage(context.Background(), &clientpb.AIConversationMessage{
		ConversationID: conversation.GetID(),
		OperatorName:   "alice",
		Role:           "user",
		Content:        "Walk me through the gRPC flow.",
	})
	if err != nil {
		t.Fatalf("save conversation message: %v", err)
	}

	events := waitForAIConversationFlow(t, eventStream, conversation.GetID(), clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_COMPLETED)
	assertAIConversationEventSequence(t, events, []aiConversationEventExpectation{
		{
			eventType:  clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED,
			role:       "user",
			kind:       clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
			visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
			state:      clientpb.AIConversationMessageState_AI_MESSAGE_STATE_COMPLETED,
		},
		{
			eventType: clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_STARTED,
		},
		{
			eventType:  clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED,
			role:       "assistant",
			kind:       clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
			visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
			state:      clientpb.AIConversationMessageState_AI_MESSAGE_STATE_COMPLETED,
		},
		{
			eventType: clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_COMPLETED,
		},
	})
	if got := events[0].GetMessage().GetID(); got != savedUserMessage.GetID() {
		t.Fatalf("expected first flow event to reference saved user message %q, got %q", savedUserMessage.GetID(), got)
	}

	current, err := client.GetAIConversation(context.Background(), &clientpb.AIConversationReq{
		ID:              conversation.GetID(),
		IncludeMessages: true,
	})
	if err != nil {
		t.Fatalf("refresh ai conversation: %v", err)
	}
	if len(current.GetMessages()) != 2 {
		t.Fatalf("unexpected message count: got=%d want=%d", len(current.GetMessages()), 2)
	}
	if current.GetMessages()[1].GetContent() != "Assistant reply from the provider" {
		t.Fatalf("unexpected assistant reply: %q", current.GetMessages()[1].GetContent())
	}
	if current.GetMessages()[1].GetProviderMessageID() != "resp_success" {
		t.Fatalf("unexpected provider message id: %q", current.GetMessages()[1].GetProviderMessageID())
	}
	if current.GetModel() != "gpt-5.2" {
		t.Fatalf("unexpected conversation model: %q", current.GetModel())
	}

	select {
	case <-requests:
	default:
		t.Fatal("expected the server to call the provider API")
	}
}

func TestSaveAIConversationMessageCompletesOpenAIWithoutExplicitBaseURL(t *testing.T) {
	setupAIRPCTestEnv(t)

	requests := make(chan string, 1)
	restoreClient := serverai.SetHTTPClientForTests(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requests <- r.URL.Path
			return jsonResponse(http.StatusOK, `{
			"id": "resp_default_base",
			"model": "gpt-5.4",
			"status": "completed",
			"output": [
				{
					"type": "message",
					"role": "assistant",
					"content": [
						{"type": "output_text", "text": "Assistant reply from default OpenAI base URL"}
					]
				}
			]
		}`), nil
		}),
	})
	defer restoreClient()

	saveOpenAICompletionConfig(t, "gpt-5.4", "high", "openai-key", "")

	client, cleanup := newBufnetRPCClient(t)
	defer cleanup()

	streamCtx, cancelStream := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStream()
	eventStream, err := client.Events(streamCtx, &commonpb.Empty{})
	if err != nil {
		t.Fatalf("start events stream: %v", err)
	}

	conversation, err := client.SaveAIConversation(context.Background(), &clientpb.AIConversation{
		OperatorName: "alice",
		Provider:     serverai.ProviderOpenAI,
		Title:        "OpenAI default base URL",
	})
	if err != nil {
		t.Fatalf("save conversation: %v", err)
	}

	waitForAIConversationEvent(t, eventStream, conversation.GetID())

	if _, err := client.SaveAIConversationMessage(context.Background(), &clientpb.AIConversationMessage{
		ConversationID: conversation.GetID(),
		OperatorName:   "alice",
		Role:           "user",
		Content:        "Confirm the default OpenAI base URL path works.",
	}); err != nil {
		t.Fatalf("save conversation message: %v", err)
	}

	events := waitForAIConversationFlow(t, eventStream, conversation.GetID(), clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_COMPLETED)
	assertAIConversationEventSequence(t, events, []aiConversationEventExpectation{
		{
			eventType:  clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED,
			role:       "user",
			kind:       clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
			visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
			state:      clientpb.AIConversationMessageState_AI_MESSAGE_STATE_COMPLETED,
		},
		{
			eventType: clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_STARTED,
		},
		{
			eventType:  clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED,
			role:       "assistant",
			kind:       clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
			visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
			state:      clientpb.AIConversationMessageState_AI_MESSAGE_STATE_COMPLETED,
		},
		{
			eventType: clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_COMPLETED,
		},
	})

	current, err := client.GetAIConversation(context.Background(), &clientpb.AIConversationReq{
		ID:              conversation.GetID(),
		IncludeMessages: true,
	})
	if err != nil {
		t.Fatalf("refresh ai conversation: %v", err)
	}
	if len(current.GetMessages()) != 2 {
		t.Fatalf("unexpected message count: got=%d want=%d", len(current.GetMessages()), 2)
	}
	if current.GetMessages()[1].GetContent() != "Assistant reply from default OpenAI base URL" {
		t.Fatalf("unexpected assistant reply: %q", current.GetMessages()[1].GetContent())
	}

	select {
	case gotPath := <-requests:
		if gotPath != "/v1/responses" {
			t.Fatalf("unexpected default-base provider path: got=%q want=%q", gotPath, "/v1/responses")
		}
	default:
		t.Fatal("expected the server to call the provider API with the default OpenAI base URL")
	}
}

func TestSaveAIConversationMessagePublishesFailureMessageWhenProviderErrors(t *testing.T) {
	setupAIRPCTestEnv(t)

	restoreClient := serverai.SetHTTPClientForTests(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusUnauthorized, `{"error":{"message":"bad credentials"}}`), nil
		}),
	})
	defer restoreClient()

	saveOpenAICompletionConfig(t, "gpt-5.2", "high", "broken-key", "https://openai.example/v1")

	client, cleanup := newBufnetRPCClient(t)
	defer cleanup()

	streamCtx, cancelStream := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStream()
	eventStream, err := client.Events(streamCtx, &commonpb.Empty{})
	if err != nil {
		t.Fatalf("start events stream: %v", err)
	}

	conversation, err := client.SaveAIConversation(context.Background(), &clientpb.AIConversation{
		OperatorName: "alice",
		Provider:     serverai.ProviderOpenAI,
		Title:        "Failure test",
	})
	if err != nil {
		t.Fatalf("save conversation: %v", err)
	}

	waitForAIConversationEvent(t, eventStream, conversation.GetID())

	savedUserMessage, err := client.SaveAIConversationMessage(context.Background(), &clientpb.AIConversationMessage{
		ConversationID: conversation.GetID(),
		OperatorName:   "alice",
		Role:           "user",
		Content:        "Trigger a provider error.",
	})
	if err != nil {
		t.Fatalf("save conversation message: %v", err)
	}

	events := waitForAIConversationFlow(t, eventStream, conversation.GetID(), clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_FAILED)
	assertAIConversationEventSequence(t, events, []aiConversationEventExpectation{
		{
			eventType:  clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED,
			role:       "user",
			kind:       clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
			visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
			state:      clientpb.AIConversationMessageState_AI_MESSAGE_STATE_COMPLETED,
		},
		{
			eventType: clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_STARTED,
		},
		{
			eventType:  clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED,
			role:       "system",
			kind:       clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
			visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY,
			state:      clientpb.AIConversationMessageState_AI_MESSAGE_STATE_FAILED,
		},
		{
			eventType: clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_FAILED,
		},
	})
	if got := events[0].GetMessage().GetID(); got != savedUserMessage.GetID() {
		t.Fatalf("expected first flow event to reference saved user message %q, got %q", savedUserMessage.GetID(), got)
	}

	current, err := client.GetAIConversation(context.Background(), &clientpb.AIConversationReq{
		ID:              conversation.GetID(),
		IncludeMessages: true,
	})
	if err != nil {
		t.Fatalf("refresh ai conversation: %v", err)
	}
	if len(current.GetMessages()) != 2 {
		t.Fatalf("unexpected message count: got=%d want=%d", len(current.GetMessages()), 2)
	}
	lastMessage := current.GetMessages()[1]
	if !strings.Contains(lastMessage.GetContent(), "HTTP 401") || !strings.Contains(lastMessage.GetContent(), "bad credentials") {
		t.Fatalf("unexpected failure message: %q", lastMessage.GetContent())
	}
	if lastMessage.GetFinishReason() != "error" {
		t.Fatalf("unexpected finish reason: %q", lastMessage.GetFinishReason())
	}
	if lastMessage.GetVisibility() != clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY {
		t.Fatalf("expected failure message to stay out of the context window, got %+v", lastMessage)
	}
}

func TestSaveAIConversationMessagePersistsReasoningAndToolBlocks(t *testing.T) {
	setupAIRPCTestEnv(t)

	requestCount := 0
	restoreClient := serverai.SetHTTPClientForTests(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("unexpected provider request path: %q", r.URL.Path)
			}

			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode provider request: %v", err)
			}

			requestCount++
			switch requestCount {
			case 1:
				tools, ok := payload["tools"].([]any)
				if !ok || len(tools) == 0 {
					t.Fatalf("expected first request to advertise tools, got %#v", payload["tools"])
				}
				assertStrictResponseToolSchema(t, tools, "list_sessions_and_beacons", []string{}, map[string][]string{})
				assertStrictResponseToolSchema(t, tools, "fs_ls", []string{"beacon_id", "path", "session_id"}, map[string][]string{
					"beacon_id":  []string{"string", "null"},
					"path":       []string{"string", "null"},
					"session_id": []string{"string", "null"},
				})
				assertStrictResponseToolSchema(t, tools, "fs_cat", []string{"beacon_id", "max_bytes", "max_lines", "path", "session_id"}, map[string][]string{
					"beacon_id":  []string{"string", "null"},
					"max_bytes":  []string{"integer", "null"},
					"max_lines":  []string{"integer", "null"},
					"path":       []string{"string"},
					"session_id": []string{"string", "null"},
				})
				assertStrictResponseToolSchema(t, tools, "fs_pwd", []string{"beacon_id", "session_id"}, map[string][]string{
					"beacon_id":  []string{"string", "null"},
					"session_id": []string{"string", "null"},
				})
				return jsonResponse(http.StatusOK, `{
			"id": "resp_step1",
			"model": "gpt-5.2",
			"status": "completed",
			"output": [
				{
					"id": "reasoning_1",
					"type": "reasoning",
					"summary": [
						{"type": "summary_text", "text": "Need to inspect the available targets first."}
					],
					"content": [
						{"type": "reasoning_text", "text": "Need to inspect the available targets first."}
					],
					"status": "completed"
				},
				{
					"id": "call_item_1",
					"type": "function_call",
					"call_id": "call_1",
					"name": "list_sessions_and_beacons",
					"arguments": "{}",
					"status": "completed"
				}
			]
		}`), nil
			case 2:
				if payload["previous_response_id"] != "resp_step1" {
					t.Fatalf("expected previous_response_id to be chained, got %#v", payload["previous_response_id"])
				}
				input, ok := payload["input"].([]any)
				if !ok || len(input) != 1 {
					t.Fatalf("expected tool output input item, got %#v", payload["input"])
				}
				return jsonResponse(http.StatusOK, `{
			"id": "resp_step2",
			"model": "gpt-5.2",
			"status": "completed",
			"output": [
				{
					"id": "msg_1",
					"type": "message",
					"role": "assistant",
					"status": "completed",
					"content": [
						{"type": "output_text", "text": "Assistant reply after a tool call"}
					]
				}
			]
		}`), nil
			default:
				t.Fatalf("unexpected provider request count %d", requestCount)
				return nil, nil
			}
		}),
	})
	defer restoreClient()

	saveOpenAICompletionConfig(t, "gpt-5.2", "high", "openai-key", "https://openai.example/v1")

	client, cleanup := newBufnetRPCClient(t)
	defer cleanup()

	streamCtx, cancelStream := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStream()
	eventStream, err := client.Events(streamCtx, &commonpb.Empty{})
	if err != nil {
		t.Fatalf("start events stream: %v", err)
	}

	conversation, err := client.SaveAIConversation(context.Background(), &clientpb.AIConversation{
		OperatorName: "alice",
		Provider:     serverai.ProviderOpenAI,
		Title:        "Agentic workflow",
	})
	if err != nil {
		t.Fatalf("save conversation: %v", err)
	}
	waitForAIConversationEvent(t, eventStream, conversation.GetID())

	savedUserMessage, err := client.SaveAIConversationMessage(context.Background(), &clientpb.AIConversationMessage{
		ConversationID: conversation.GetID(),
		OperatorName:   "alice",
		Role:           "user",
		Content:        "Figure out what targets are available and summarize them.",
	})
	if err != nil {
		t.Fatalf("save conversation message: %v", err)
	}

	events := waitForAIConversationFlow(t, eventStream, conversation.GetID(), clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_COMPLETED)
	assertAIConversationEventSequence(t, events, []aiConversationEventExpectation{
		{
			eventType:  clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED,
			role:       "user",
			kind:       clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
			visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
			state:      clientpb.AIConversationMessageState_AI_MESSAGE_STATE_COMPLETED,
		},
		{
			eventType: clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_STARTED,
		},
		{
			eventType:  clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED,
			role:       "assistant",
			kind:       clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_REASONING,
			visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY,
			state:      clientpb.AIConversationMessageState_AI_MESSAGE_STATE_COMPLETED,
			itemID:     "reasoning_1",
		},
		{
			eventType:  clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_STARTED,
			role:       "assistant",
			kind:       clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_TOOL_CALL,
			visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY,
			state:      clientpb.AIConversationMessageState_AI_MESSAGE_STATE_IN_PROGRESS,
			itemID:     "call_item_1",
			toolName:   "list_sessions_and_beacons",
		},
		{
			eventType:  clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED,
			role:       "assistant",
			kind:       clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_TOOL_CALL,
			visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY,
			state:      clientpb.AIConversationMessageState_AI_MESSAGE_STATE_COMPLETED,
			itemID:     "call_item_1",
			toolName:   "list_sessions_and_beacons",
		},
		{
			eventType:  clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED,
			role:       "assistant",
			kind:       clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
			visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
			state:      clientpb.AIConversationMessageState_AI_MESSAGE_STATE_COMPLETED,
		},
		{
			eventType: clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_COMPLETED,
		},
	})
	if got := events[0].GetMessage().GetID(); got != savedUserMessage.GetID() {
		t.Fatalf("expected first flow event to reference saved user message %q, got %q", savedUserMessage.GetID(), got)
	}

	current, err := client.GetAIConversation(context.Background(), &clientpb.AIConversationReq{
		ID:              conversation.GetID(),
		IncludeMessages: true,
	})
	if err != nil {
		t.Fatalf("refresh ai conversation: %v", err)
	}
	if len(current.GetMessages()) != 4 {
		t.Fatalf("unexpected message count: got=%d want=%d", len(current.GetMessages()), 4)
	}

	reasoning := current.GetMessages()[1]
	if reasoning.GetKind() != clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_REASONING {
		t.Fatalf("expected reasoning block, got %+v", reasoning)
	}
	if reasoning.GetVisibility() != clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY {
		t.Fatalf("expected reasoning block to be UI-only, got %+v", reasoning)
	}

	toolCall := current.GetMessages()[2]
	if toolCall.GetKind() != clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_TOOL_CALL {
		t.Fatalf("expected tool call block, got %+v", toolCall)
	}
	if toolCall.GetToolName() != "list_sessions_and_beacons" {
		t.Fatalf("unexpected tool name: %+v", toolCall)
	}
	if strings.TrimSpace(toolCall.GetToolResult()) == "" {
		t.Fatalf("expected tool result to be stored, got %+v", toolCall)
	}
	if toolCall.GetVisibility() != clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY {
		t.Fatalf("expected tool call to be UI-only, got %+v", toolCall)
	}

	reply := current.GetMessages()[3]
	if reply.GetContent() != "Assistant reply after a tool call" {
		t.Fatalf("unexpected assistant reply: %q", reply.GetContent())
	}
	if reply.GetVisibility() != clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT {
		t.Fatalf("expected assistant reply to remain in the context window, got %+v", reply)
	}
}

func TestAIConversationsAreSharedAcrossOperators(t *testing.T) {
	setupAIRPCTestEnv(t)

	rpc := &Server{}
	aliceCtx := contextWithCommonName("alice")
	bobCtx := contextWithCommonName("bob")

	conversation, err := rpc.SaveAIConversation(aliceCtx, &clientpb.AIConversation{
		Provider: serverai.ProviderOpenAI,
		Title:    "Shared thread",
	})
	if err != nil {
		t.Fatalf("save shared conversation: %v", err)
	}
	if conversation.GetOperatorName() != "alice" {
		t.Fatalf("expected conversation creator to be preserved, got %q", conversation.GetOperatorName())
	}

	conversations, err := rpc.GetAIConversations(bobCtx, &commonpb.Empty{})
	if err != nil {
		t.Fatalf("list shared conversations as bob: %v", err)
	}
	if len(conversations.GetConversations()) != 1 {
		t.Fatalf("unexpected shared conversation count: got=%d want=%d", len(conversations.GetConversations()), 1)
	}
	if conversations.GetConversations()[0].GetID() != conversation.GetID() {
		t.Fatalf("unexpected shared conversation id: got=%q want=%q", conversations.GetConversations()[0].GetID(), conversation.GetID())
	}

	current, err := rpc.GetAIConversation(bobCtx, &clientpb.AIConversationReq{
		ID:              conversation.GetID(),
		IncludeMessages: true,
	})
	if err != nil {
		t.Fatalf("load shared conversation as bob: %v", err)
	}
	if current.GetOperatorName() != "alice" {
		t.Fatalf("expected shared conversation to keep alice as creator, got %q", current.GetOperatorName())
	}

	message, err := rpc.SaveAIConversationMessage(bobCtx, &clientpb.AIConversationMessage{
		ConversationID: conversation.GetID(),
		Role:           "system",
		Content:        "Bob joined the shared thread.",
	})
	if err != nil {
		t.Fatalf("save shared conversation message as bob: %v", err)
	}
	if message.GetOperatorName() != "bob" {
		t.Fatalf("expected saved message author to be bob, got %q", message.GetOperatorName())
	}

	current, err = rpc.GetAIConversation(aliceCtx, &clientpb.AIConversationReq{
		ID:              conversation.GetID(),
		IncludeMessages: true,
	})
	if err != nil {
		t.Fatalf("reload shared conversation as alice: %v", err)
	}
	if current.GetOperatorName() != "alice" {
		t.Fatalf("expected shared conversation creator to remain alice, got %q", current.GetOperatorName())
	}
	if len(current.GetMessages()) != 1 {
		t.Fatalf("unexpected shared message count: got=%d want=%d", len(current.GetMessages()), 1)
	}
	if current.GetMessages()[0].GetOperatorName() != "bob" {
		t.Fatalf("expected shared message author to be bob, got %q", current.GetMessages()[0].GetOperatorName())
	}

	if _, err := rpc.DeleteAIConversation(bobCtx, &clientpb.AIConversationReq{ID: conversation.GetID()}); err != nil {
		t.Fatalf("delete shared conversation as bob: %v", err)
	}

	_, err = rpc.GetAIConversation(aliceCtx, &clientpb.AIConversationReq{
		ID:              conversation.GetID(),
		IncludeMessages: true,
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected deleted shared conversation to be missing, got %v", err)
	}
}

func setupAIRPCTestEnv(t *testing.T) {
	t.Helper()

	rootDir := t.TempDir()
	t.Setenv("SLIVER_ROOT_DIR", rootDir)

	originalDB := db.Client
	testDB, err := gorm.Open(sqlite.Open(filepath.Join(rootDir, "ai-rpc-test.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := testDB.AutoMigrate(
		&models.AIConversation{},
		&models.AIConversationMessage{},
		&models.Beacon{},
		&models.BeaconTask{},
	); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	db.Client = testDB

	t.Cleanup(func() {
		sqlDB, err := testDB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
		db.Client = originalDB
	})
}

func saveOpenAICompletionConfig(t *testing.T, model string, thinkingLevel string, apiKey string, baseURL string) {
	t.Helper()

	cfg := configs.GetServerConfig()
	cfg.AI = &configs.AIConfig{
		Provider:      serverai.ProviderOpenAI,
		Model:         model,
		ThinkingLevel: thinkingLevel,
		OpenAI: &configs.AIProviderConfig{
			APIKey:          apiKey,
			BaseURL:         baseURL,
			UseResponsesAPI: boolPtr(true),
		},
		Anthropic: &configs.AIProviderConfig{},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save test server config: %v", err)
	}
}

func waitForAIConversationEvent(t *testing.T, eventStream rpcpb.SliverRPC_EventsClient, conversationID string) {
	t.Helper()

	for {
		event, err := eventStream.Recv()
		if err != nil {
			t.Fatalf("receive ai conversation event: %v", err)
		}
		if event.GetEventType() != consts.AIConversationEvent {
			continue
		}

		conversationEvent := &clientpb.AIConversationEvent{}
		if len(event.GetData()) > 0 {
			if err := proto.Unmarshal(event.GetData(), conversationEvent); err != nil {
				t.Fatalf("unmarshal ai conversation event: %v", err)
			}
		}
		if conversationEvent.GetConversationID() == conversationID {
			return
		}
	}
}

func waitForAIConversationRole(t *testing.T, client rpcpb.SliverRPCClient, eventStream rpcpb.SliverRPC_EventsClient, conversationID string, role string, minEvents int) *clientpb.AIConversation {
	t.Helper()

	eventCount := 0
	for {
		event, err := eventStream.Recv()
		if err != nil {
			t.Fatalf("receive ai conversation event: %v", err)
		}
		if event.GetEventType() != consts.AIConversationEvent {
			continue
		}

		conversationEvent := &clientpb.AIConversationEvent{}
		if len(event.GetData()) > 0 {
			if err := proto.Unmarshal(event.GetData(), conversationEvent); err != nil {
				t.Fatalf("unmarshal ai conversation event: %v", err)
			}
		}
		if conversationEvent.GetConversationID() != conversationID {
			continue
		}

		eventCount++
		current, err := client.GetAIConversation(context.Background(), &clientpb.AIConversationReq{
			ID:              conversationID,
			IncludeMessages: true,
		})
		if err != nil {
			t.Fatalf("refresh ai conversation: %v", err)
		}
		messages := current.GetMessages()
		if len(messages) == 0 {
			continue
		}
		if aiConversationHasRole(current, role) && eventCount >= minEvents {
			return current
		}
	}
}

type aiConversationEventExpectation struct {
	eventType  clientpb.AIConversationEventType
	role       string
	kind       clientpb.AIConversationMessageKind
	visibility clientpb.AIConversationMessageVisibility
	state      clientpb.AIConversationMessageState
	itemID     string
	toolName   string
}

func waitForAIConversationFlow(t *testing.T, eventStream rpcpb.SliverRPC_EventsClient, conversationID string, terminalType clientpb.AIConversationEventType) []*clientpb.AIConversationEvent {
	t.Helper()

	events := []*clientpb.AIConversationEvent{}
	for {
		event, err := eventStream.Recv()
		if err != nil {
			t.Fatalf("receive ai conversation event: %v", err)
		}
		if event.GetEventType() != consts.AIConversationEvent {
			continue
		}

		conversationEvent := &clientpb.AIConversationEvent{}
		if len(event.GetData()) > 0 {
			if err := proto.Unmarshal(event.GetData(), conversationEvent); err != nil {
				t.Fatalf("unmarshal ai conversation event: %v", err)
			}
		}
		if conversationEvent.GetConversationID() != conversationID {
			continue
		}

		events = append(events, conversationEvent)
		if conversationEvent.GetEventType() == terminalType {
			return events
		}
	}
}

func assertAIConversationEventSequence(t *testing.T, events []*clientpb.AIConversationEvent, expectations []aiConversationEventExpectation) {
	t.Helper()

	if len(events) != len(expectations) {
		t.Fatalf("unexpected ai event count: got=%d want=%d\nflow:\n%s", len(events), len(expectations), formatAIConversationEventFlow(events))
	}

	for idx, expectation := range expectations {
		event := events[idx]
		if event.GetEventType() != expectation.eventType {
			t.Fatalf("unexpected ai event[%d] type: got=%s want=%s\nflow:\n%s", idx, event.GetEventType(), expectation.eventType, formatAIConversationEventFlow(events))
		}
		if expectation.role == "" {
			if event.GetMessage() != nil {
				t.Fatalf("expected ai event[%d] to omit a message payload, got %+v", idx, event.GetMessage())
			}
			continue
		}
		message := event.GetMessage()
		if message == nil {
			t.Fatalf("expected ai event[%d] to include a message payload\nflow:\n%s", idx, formatAIConversationEventFlow(events))
		}
		if got := strings.ToLower(strings.TrimSpace(message.GetRole())); got != expectation.role {
			t.Fatalf("unexpected ai event[%d] role: got=%q want=%q\nflow:\n%s", idx, got, expectation.role, formatAIConversationEventFlow(events))
		}
		if message.GetKind() != expectation.kind {
			t.Fatalf("unexpected ai event[%d] kind: got=%s want=%s\nflow:\n%s", idx, message.GetKind(), expectation.kind, formatAIConversationEventFlow(events))
		}
		if message.GetVisibility() != expectation.visibility {
			t.Fatalf("unexpected ai event[%d] visibility: got=%s want=%s\nflow:\n%s", idx, message.GetVisibility(), expectation.visibility, formatAIConversationEventFlow(events))
		}
		if message.GetState() != expectation.state {
			t.Fatalf("unexpected ai event[%d] state: got=%s want=%s\nflow:\n%s", idx, message.GetState(), expectation.state, formatAIConversationEventFlow(events))
		}
		if expectation.itemID != "" && strings.TrimSpace(message.GetItemID()) != expectation.itemID {
			t.Fatalf("unexpected ai event[%d] item id: got=%q want=%q\nflow:\n%s", idx, message.GetItemID(), expectation.itemID, formatAIConversationEventFlow(events))
		}
		if expectation.toolName != "" && strings.TrimSpace(message.GetToolName()) != expectation.toolName {
			t.Fatalf("unexpected ai event[%d] tool name: got=%q want=%q\nflow:\n%s", idx, message.GetToolName(), expectation.toolName, formatAIConversationEventFlow(events))
		}
	}
}

func formatAIConversationEventFlow(events []*clientpb.AIConversationEvent) string {
	parts := make([]string, 0, len(events))
	for _, event := range events {
		if event == nil {
			parts = append(parts, "<nil>")
			continue
		}
		part := event.GetEventType().String()
		if message := event.GetMessage(); message != nil {
			part += " role=" + strings.TrimSpace(message.GetRole())
			part += " kind=" + message.GetKind().String()
			part += " visibility=" + message.GetVisibility().String()
			part += " state=" + message.GetState().String()
			if itemID := strings.TrimSpace(message.GetItemID()); itemID != "" {
				part += " item=" + itemID
			}
			if toolName := strings.TrimSpace(message.GetToolName()); toolName != "" {
				part += " tool=" + toolName
			}
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "\n")
}

func assertStrictResponseToolSchema(t *testing.T, tools []any, name string, required []string, propertyTypes map[string][]string) {
	t.Helper()

	schema := responseToolSchemaByName(t, tools, name)
	if strict, ok := schema["strict"].(bool); !ok || !strict {
		t.Fatalf("expected tool %q to enable strict mode, got %#v", name, schema["strict"])
	}

	parameters, ok := schema["parameters"].(map[string]any)
	if !ok {
		t.Fatalf("expected tool %q parameters map, got %#v", name, schema["parameters"])
	}
	if additionalProperties, ok := parameters["additionalProperties"].(bool); !ok || additionalProperties {
		t.Fatalf("expected tool %q to disable additionalProperties, got %#v", name, parameters["additionalProperties"])
	}
	assertJSONSchemaStringSet(t, name+" required", parameters["required"], required)

	properties, ok := parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected tool %q properties map, got %#v", name, parameters["properties"])
	}
	if len(properties) != len(propertyTypes) {
		t.Fatalf("unexpected tool %q property count: got=%d want=%d", name, len(properties), len(propertyTypes))
	}
	for propertyName, expectedTypes := range propertyTypes {
		property, ok := properties[propertyName].(map[string]any)
		if !ok {
			t.Fatalf("expected tool %q property %q to be a schema object, got %#v", name, propertyName, properties[propertyName])
		}
		assertJSONSchemaStringSet(t, name+"."+propertyName+" type", property["type"], expectedTypes)
	}
}

func responseToolSchemaByName(t *testing.T, tools []any, name string) map[string]any {
	t.Helper()

	for _, rawTool := range tools {
		tool, ok := rawTool.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(anyStringValue(tool["name"])) == name {
			return tool
		}
	}
	t.Fatalf("expected tools payload to include %q, got %#v", name, tools)
	return nil
}

func assertJSONSchemaStringSet(t *testing.T, label string, raw any, want []string) {
	t.Helper()

	got := jsonStringSet(raw)
	if len(got) != len(want) {
		t.Fatalf("unexpected %s count: got=%d want=%d raw=%#v", label, len(got), len(want), raw)
	}
	for _, expected := range want {
		if !got[expected] {
			t.Fatalf("expected %s to contain %q, got %#v", label, expected, raw)
		}
	}
}

func jsonStringSet(raw any) map[string]bool {
	values := map[string]bool{}
	switch typed := raw.(type) {
	case []any:
		for _, rawValue := range typed {
			if value, ok := rawValue.(string); ok {
				value = strings.TrimSpace(value)
				if value != "" {
					values[value] = true
				}
			}
		}
	case []string:
		for _, value := range typed {
			value = strings.TrimSpace(value)
			if value != "" {
				values[value] = true
			}
		}
	case string:
		value := strings.TrimSpace(typed)
		if value != "" {
			values[value] = true
		}
	}
	return values
}

func anyStringValue(raw any) string {
	value, _ := raw.(string)
	return value
}

func aiConversationHasRole(conversation *clientpb.AIConversation, role string) bool {
	if conversation == nil {
		return false
	}

	role = strings.TrimSpace(strings.ToLower(role))
	messages := conversation.GetMessages()
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		if message == nil || strings.TrimSpace(strings.ToLower(message.GetRole())) != role {
			continue
		}
		if role == "assistant" {
			return message.GetKind() == clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT &&
				message.GetVisibility() == clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT
		}
		return true
	}
	return false
}

func waitForAIConversationTurnEvent(t *testing.T, eventStream rpcpb.SliverRPC_EventsClient, conversationID string, wantType clientpb.AIConversationEventType) {
	t.Helper()

	for {
		event, err := eventStream.Recv()
		if err != nil {
			t.Fatalf("receive ai conversation event: %v", err)
		}
		if event.GetEventType() != consts.AIConversationEvent {
			continue
		}

		conversationEvent := &clientpb.AIConversationEvent{}
		if len(event.GetData()) > 0 {
			if err := proto.Unmarshal(event.GetData(), conversationEvent); err != nil {
				t.Fatalf("unmarshal ai conversation event: %v", err)
			}
		}
		if conversationEvent.GetConversationID() != conversationID {
			continue
		}
		if conversationEvent.GetEventType() == wantType {
			return
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func boolPtr(value bool) *bool {
	return &value
}
