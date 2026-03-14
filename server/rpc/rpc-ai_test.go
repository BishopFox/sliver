package rpc

import (
	"context"
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

	if _, err := client.SaveAIConversationMessage(context.Background(), &clientpb.AIConversationMessage{
		ConversationID: conversation.GetID(),
		OperatorName:   "alice",
		Role:           "user",
		Content:        "Walk me through the gRPC flow.",
	}); err != nil {
		t.Fatalf("save conversation message: %v", err)
	}

	current := waitForAIConversationRole(t, client, eventStream, conversation.GetID(), "assistant", 2)
	if len(current.GetMessages()) != 2 {
		t.Fatalf("unexpected message count: got=%d want=%d", len(current.GetMessages()), 2)
	}
	if current.GetMessages()[1].GetContent() != "Assistant reply from the provider" {
		t.Fatalf("unexpected assistant reply: %q", current.GetMessages()[1].GetContent())
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

	if _, err := client.SaveAIConversationMessage(context.Background(), &clientpb.AIConversationMessage{
		ConversationID: conversation.GetID(),
		OperatorName:   "alice",
		Role:           "user",
		Content:        "Trigger a provider error.",
	}); err != nil {
		t.Fatalf("save conversation message: %v", err)
	}

	current := waitForAIConversationRole(t, client, eventStream, conversation.GetID(), "system", 2)
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
	if err := testDB.AutoMigrate(&models.AIConversation{}, &models.AIConversationMessage{}); err != nil {
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
			APIKey:  apiKey,
			BaseURL: baseURL,
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

		conversation := &clientpb.AIConversation{}
		if len(event.GetData()) > 0 {
			if err := proto.Unmarshal(event.GetData(), conversation); err != nil {
				t.Fatalf("unmarshal ai conversation event: %v", err)
			}
		}
		if conversation.GetID() == conversationID {
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

		conversation := &clientpb.AIConversation{}
		if len(event.GetData()) > 0 {
			if err := proto.Unmarshal(event.GetData(), conversation); err != nil {
				t.Fatalf("unmarshal ai conversation event: %v", err)
			}
		}
		if conversation.GetID() != conversationID {
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
		if messages[len(messages)-1].GetRole() == role && eventCount >= minEvents {
			return current
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
