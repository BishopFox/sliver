package ai

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/bishopfox/sliver/client/console"
	aithinking "github.com/bishopfox/sliver/client/spin/thinking"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/charmbracelet/x/ansi"
)

func TestPromptConversationTitleUsesFirstNonEmptyLine(t *testing.T) {
	title := promptConversationTitle("\n\n  First line title  \nsecond line")
	if title != "First line title" {
		t.Fatalf("expected first non-empty line, got %q", title)
	}
}

func TestBuildConversationMarkdownUsesFencedBlocksAndOperatorLabel(t *testing.T) {
	conversation := &clientpb.AIConversation{
		OperatorName: "alice",
		Summary:      "Operator context",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "## Reply"},
		},
	}

	markdown := buildConversationMarkdown(conversation)
	expected := []string{
		"Operator context",
		"```text\n[alice]\nHello\n```",
		"```text\n[AI]\n## Reply\n```",
	}
	for _, fragment := range expected {
		if !strings.Contains(markdown, fragment) {
			t.Fatalf("expected markdown to contain %q, got %q", fragment, markdown)
		}
	}
}

func TestBuildConversationMarkdownFallsBackToUserLabel(t *testing.T) {
	conversation := &clientpb.AIConversation{
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	markdown := buildConversationMarkdown(conversation)
	if !strings.Contains(markdown, "```text\n[User]\nHello\n```") {
		t.Fatalf("expected markdown to contain user fallback label, got %q", markdown)
	}
}

func TestConversationAwaitingResponseWhenLastMessageIsUser(t *testing.T) {
	conversation := &clientpb.AIConversation{
		Messages: []*clientpb.AIConversationMessage{
			{Role: "assistant", Content: "Previous reply"},
			{Role: "user", Content: "Still waiting"},
		},
	}

	if !conversationAwaitingResponse(conversation) {
		t.Fatal("expected conversation to be waiting on an assistant response")
	}
}

func TestConversationAwaitingResponseStopsWhenAssistantReplies(t *testing.T) {
	conversation := &clientpb.AIConversation{
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "Question"},
			{Role: "assistant", Content: "Answer"},
		},
	}

	if conversationAwaitingResponse(conversation) {
		t.Fatal("expected conversation to be settled once the assistant replies")
	}
}

func TestPendingLabelUsesThinkingWhenConfigured(t *testing.T) {
	model := &aiModel{
		config: &clientpb.AIConfigSummary{ThinkingLevel: "high"},
	}

	if got := model.pendingLabel(); got != "Thinking" {
		t.Fatalf("expected pending label %q, got %q", "Thinking", got)
	}
}

func TestPendingLabelFallsBackToWorking(t *testing.T) {
	model := &aiModel{}

	if got := model.pendingLabel(); got != "Working" {
		t.Fatalf("expected pending label %q, got %q", "Working", got)
	}
}

func TestComposerEnterStartsAwaitingResponseImmediately(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.focus = aiFocusComposer
	model.loading = false
	model.currentConversation = &clientpb.AIConversation{
		ID:       "conv-1",
		Title:    "Thread",
		Provider: "openai",
		Model:    "gpt-test",
	}
	model.input = []rune("hello")
	model.cursor = len(model.input)

	updated, cmd := model.handleComposerKey(tea.Key{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected enter to queue submit work")
	}

	updatedModel := updated.(*aiModel)
	if !updatedModel.awaitingResponse {
		t.Fatal("expected enter to start awaiting-response state immediately")
	}
	if !updatedModel.submittingPrompt {
		t.Fatal("expected enter to mark prompt submission as in flight")
	}
	if updatedModel.pendingPrompt != "hello" {
		t.Fatalf("expected enter to stage the pending prompt, got %q", updatedModel.pendingPrompt)
	}
	if len(updatedModel.input) != 0 || updatedModel.cursor != 0 {
		t.Fatal("expected enter to clear the composer immediately")
	}
}

func TestPromptSubmittedStartsAwaitingResponseImmediately(t *testing.T) {
	model := newAIModel(nil, aiContext{
		connection: aiConnectionSummary{Operator: "alice"},
	}, nil)
	model.loading = false
	model.submittingPrompt = true
	model.pendingPrompt = "What changed?"
	model.conversations = []*clientpb.AIConversation{
		{ID: "conv-1", Title: "Thread", Provider: "openai", Model: "gpt-test"},
	}
	model.currentConversation = &clientpb.AIConversation{
		ID:           "conv-1",
		Title:        "Thread",
		Provider:     "openai",
		Model:        "gpt-test",
		OperatorName: "alice",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "assistant", Content: "Previous reply"},
		},
	}

	updated, cmd := model.Update(aiPromptSubmittedMsg{
		conversationID: "conv-1",
		conversation: &clientpb.AIConversation{
			ID:       "conv-1",
			Title:    "Thread",
			Provider: "openai",
			Model:    "gpt-test",
		},
		message: &clientpb.AIConversationMessage{
			ID:             "msg-1",
			ConversationID: "conv-1",
			Role:           "user",
			Content:        "What changed?",
		},
		status: "Saved prompt to Thread. Waiting for AI response...",
	})
	if cmd == nil {
		t.Fatal("expected prompt submit to keep the pending animation active")
	}

	updatedModel := updated.(*aiModel)
	if !updatedModel.awaitingResponse {
		t.Fatal("expected prompt submit to enter awaiting-response state immediately")
	}
	last := lastConversationMessage(updatedModel.currentConversation)
	if last == nil || last.GetRole() != "user" || last.GetContent() != "What changed?" {
		t.Fatalf("expected optimistic user message to be appended, got %#v", last)
	}
	if updatedModel.selectedConversation != 0 {
		t.Fatalf("expected submitted conversation to remain selected, got %d", updatedModel.selectedConversation)
	}
	if updatedModel.pendingPrompt != "" {
		t.Fatalf("expected prompt submit to clear the local pending prompt, got %q", updatedModel.pendingPrompt)
	}
	if updatedModel.submittingPrompt {
		t.Fatal("expected prompt submit to clear the submitting state")
	}
}

func TestAsyncErrClearsAwaitingResponseWhenSubmitFails(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.awaitingResponse = true
	model.loading = true
	model.submittingPrompt = true
	model.pendingPrompt = "hello"

	updated, _ := model.Update(aiAsyncErrMsg{err: assertErr("submit failed")})
	updatedModel := updated.(*aiModel)
	if updatedModel.awaitingResponse {
		t.Fatal("expected submit failure to clear awaiting-response state")
	}
	if updatedModel.submittingPrompt {
		t.Fatal("expected submit failure to clear submitting state")
	}
	if updatedModel.pendingPrompt != "" {
		t.Fatalf("expected submit failure to clear pending prompt, got %q", updatedModel.pendingPrompt)
	}
}

func TestStartAwaitingResponseShowsAnimatedFrameImmediately(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.loading = false

	model.startAwaitingResponse()

	rendered := ansi.Strip(strings.Join(model.renderAwaitingResponseLines(80), "\n"))
	if !strings.Contains(rendered, "Working.") {
		t.Fatalf("expected pending placeholder to start with an animated frame, got %q", rendered)
	}
}

func TestStartAwaitingResponseCreatesFreshThinkingAnim(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	original := model.thinkingAnim

	model.startAwaitingResponse()

	if model.thinkingAnim == nil {
		t.Fatal("expected pending state to allocate a thinking animation")
	}
	if model.thinkingAnim == original {
		t.Fatal("expected pending state to create a fresh thinking animation instance")
	}
}

func TestLoadedReplyClearsAwaitingResponse(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.awaitingResponse = true
	model.currentConversation = &clientpb.AIConversation{
		ID:       "conv-1",
		Title:    "Thread",
		Provider: "openai",
		Model:    "gpt-test",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "assistant", Content: "Older reply"},
			{Role: "user", Content: "Pending question"},
		},
	}

	updated, cmd := model.Update(aiLoadedMsg{
		config: &clientpb.AIConfigSummary{Valid: true},
		conversations: []*clientpb.AIConversation{
			{ID: "conv-1", Title: "Thread", Provider: "openai", Model: "gpt-test"},
		},
		conversation: &clientpb.AIConversation{
			ID:       "conv-1",
			Title:    "Thread",
			Provider: "openai",
			Model:    "gpt-test",
			Messages: []*clientpb.AIConversationMessage{
				{Role: "user", Content: "Pending question"},
				{Role: "assistant", Content: "Rendered answer"},
			},
		},
		selectedID: "conv-1",
	})
	if cmd != nil {
		t.Fatalf("expected settled conversation load to stop without queuing animation, got %v", cmd)
	}

	updatedModel := updated.(*aiModel)
	if updatedModel.awaitingResponse {
		t.Fatal("expected awaiting-response state to clear once the assistant reply is loaded")
	}
	last := lastConversationMessage(updatedModel.currentConversation)
	if last == nil || last.GetRole() != "assistant" || last.GetContent() != "Rendered answer" {
		t.Fatalf("expected loaded assistant reply to be current, got %#v", last)
	}
}

func TestRenderTranscriptContentLinesIncludesPendingAssistantBlock(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.awaitingResponse = true
	model.currentConversation = &clientpb.AIConversation{
		ID:           "conv-1",
		OperatorName: "alice",
		Provider:     "openai",
		Model:        "gpt-test",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "hello"},
		},
	}

	lines := model.renderTranscriptContentLines(48)
	rendered := strings.Join(lines, "\n")
	if !strings.Contains(rendered, "[AI]") {
		t.Fatalf("expected pending assistant block label in transcript, got %q", rendered)
	}
	if !strings.Contains(rendered, ".") {
		t.Fatalf("expected pending assistant animation content in transcript, got %q", rendered)
	}
	if !strings.Contains(rendered, "hello") {
		t.Fatalf("expected existing transcript content to remain, got %q", rendered)
	}
}

func TestRenderTranscriptContentLinesIncludesPendingUserPrompt(t *testing.T) {
	model := newAIModel(nil, aiContext{
		connection: aiConnectionSummary{Operator: "alice"},
	}, nil)
	model.pendingPrompt = "still saving"
	model.awaitingResponse = true

	lines := model.renderTranscriptContentLines(48)
	rendered := ansi.Strip(strings.Join(lines, "\n"))
	if !strings.Contains(rendered, "[alice]") {
		t.Fatalf("expected pending prompt to render the operator label, got %q", rendered)
	}
	if !strings.Contains(rendered, "still saving") {
		t.Fatalf("expected pending prompt content in transcript, got %q", rendered)
	}
	if !strings.Contains(rendered, "[AI]") {
		t.Fatalf("expected pending assistant placeholder to stay visible, got %q", rendered)
	}
}

func TestLoadedConversationClearsPendingPromptWhenUserMessagePersists(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.pendingPrompt = "What changed?"

	updated, _ := model.Update(aiLoadedMsg{
		config: &clientpb.AIConfigSummary{Valid: true},
		conversations: []*clientpb.AIConversation{
			{ID: "conv-1", Title: "Thread", Provider: "openai", Model: "gpt-test"},
		},
		conversation: &clientpb.AIConversation{
			ID:       "conv-1",
			Title:    "Thread",
			Provider: "openai",
			Model:    "gpt-test",
			Messages: []*clientpb.AIConversationMessage{
				{Role: "user", Content: "What changed?"},
			},
		},
		selectedID: "conv-1",
	})

	updatedModel := updated.(*aiModel)
	if updatedModel.pendingPrompt != "" {
		t.Fatalf("expected loaded conversation to clear the persisted pending prompt, got %q", updatedModel.pendingPrompt)
	}
}

func TestLooksLikeTerminalResponseFragment(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{text: "]11;rgb:0000/0000/0000", want: true},
		{text: "[39;17R", want: true},
		{text: "[35;39R", want: true},
		{text: "rgb:0000/0000/0000", want: true},
		{text: "hello world", want: false},
		{text: "[User]", want: false},
	}

	for _, tc := range tests {
		if got := looksLikeTerminalResponseFragment(tc.text); got != tc.want {
			t.Fatalf("looksLikeTerminalResponseFragment(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

func TestAIProgramAnimatesWhileSubmitPending(t *testing.T) {
	releaseSubmit := make(chan struct{})
	submitStarted := make(chan struct{}, 1)
	server := &aiRPCServer{
		saveMessageStarted: submitStarted,
		saveMessageRelease: releaseSubmit,
		saveMessageResp: &clientpb.AIConversationMessage{
			ID:             "msg-1",
			ConversationID: "conv-1",
			Role:           "user",
			Content:        "hello",
		},
	}

	rpcClient, cleanup := newAITestRPCClient(t, server)
	defer cleanup()

	inner := newAIModel(&console.SliverClient{Rpc: rpcClient}, aiContext{
		connection: aiConnectionSummary{Operator: "alice"},
	}, nil)
	inner.loading = false
	inner.focus = aiFocusComposer
	inner.width = 100
	inner.height = 30
	inner.config = &clientpb.AIConfigSummary{
		Valid:    true,
		Provider: "openai",
		Model:    "gpt-test",
	}
	inner.currentConversation = &clientpb.AIConversation{
		ID:           "conv-1",
		Title:        "Thread",
		Provider:     "openai",
		Model:        "gpt-test",
		OperatorName: "alice",
	}
	inner.input = []rune("hello")
	inner.cursor = len(inner.input)

	observed := &observingAIProgramModel{
		inner:     inner,
		started:   make(chan struct{}),
		stepSeen:  make(chan struct{}, 4),
		submitted: make(chan struct{}, 1),
	}

	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	program := tea.NewProgram(
		observed,
		tea.WithContext(ctx),
		tea.WithInput(nil),
		tea.WithOutput(&out),
		tea.WithWindowSize(inner.width, inner.height),
		tea.WithoutSignals(),
		tea.WithoutSignalHandler(),
	)

	errCh := make(chan error, 1)
	go func() {
		_, err := program.Run()
		errCh <- err
	}()

	select {
	case <-observed.started:
	case <-ctx.Done():
		t.Fatal("program did not start")
	}

	program.Send(tea.KeyPressMsg{Code: tea.KeyEnter})

	select {
	case <-submitStarted:
	case <-ctx.Done():
		t.Fatal("submit RPC did not start")
	}

	select {
	case <-observed.stepSeen:
	case <-time.After(750 * time.Millisecond):
		t.Fatal("expected thinking animation to advance while submit was still pending")
	}

	select {
	case <-observed.submitted:
		t.Fatal("submit finished before the blocked RPC was released")
	default:
	}

	close(releaseSubmit)

	select {
	case <-observed.submitted:
	case <-ctx.Done():
		t.Fatal("submit result did not return after release")
	}

	program.Quit()
	if err := <-errCh; err != nil {
		t.Fatalf("program run failed: %v", err)
	}
}

func TestAIProgramSkipsRedundantPendingConversationReload(t *testing.T) {
	blockConversationLoad := make(chan struct{})
	getConversationStarted := make(chan struct{}, 1)
	server := &aiRPCServer{
		getConversationStart: getConversationStarted,
		getConversationWait:  blockConversationLoad,
		providersResp: &clientpb.AIProviderConfigs{
			Config: &clientpb.AIConfigSummary{
				Valid:    true,
				Provider: "openai",
				Model:    "gpt-test",
			},
		},
		conversationsResp: &clientpb.AIConversations{
			Conversations: []*clientpb.AIConversation{
				{ID: "conv-1", Title: "Thread", Provider: "openai", Model: "gpt-test", UpdatedAt: 100},
			},
		},
		conversationByID: map[string]*clientpb.AIConversation{
			"conv-1": {
				ID:           "conv-1",
				Title:        "Thread",
				Provider:     "openai",
				Model:        "gpt-test",
				OperatorName: "alice",
				UpdatedAt:    100,
				Messages: []*clientpb.AIConversationMessage{
					{Role: "user", Content: "hello", CreatedAt: 100, UpdatedAt: 100},
				},
			},
		},
	}

	rpcClient, cleanup := newAITestRPCClient(t, server)
	defer cleanup()

	inner := newAIModel(&console.SliverClient{Rpc: rpcClient}, aiContext{
		connection: aiConnectionSummary{Operator: "alice"},
	}, nil)
	inner.loading = false
	inner.width = 100
	inner.height = 30
	inner.currentConversation = &clientpb.AIConversation{
		ID:           "conv-1",
		Title:        "Thread",
		Provider:     "openai",
		Model:        "gpt-test",
		OperatorName: "alice",
		UpdatedAt:    100,
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "hello", CreatedAt: 100, UpdatedAt: 100},
		},
	}
	inner.conversations = []*clientpb.AIConversation{
		{ID: "conv-1", Title: "Thread", Provider: "openai", Model: "gpt-test", UpdatedAt: 100},
	}
	initCmd := inner.startAwaitingResponse()

	observed := &observingAIProgramModel{
		inner:     inner,
		initCmd:   initCmd,
		started:   make(chan struct{}),
		stepSeen:  make(chan struct{}, 8),
		submitted: make(chan struct{}, 1),
	}

	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	program := tea.NewProgram(
		observed,
		tea.WithContext(ctx),
		tea.WithInput(nil),
		tea.WithOutput(&out),
		tea.WithWindowSize(inner.width, inner.height),
		tea.WithoutSignals(),
		tea.WithoutSignalHandler(),
	)

	errCh := make(chan error, 1)
	go func() {
		_, err := program.Run()
		errCh <- err
	}()

	select {
	case <-observed.started:
	case <-ctx.Done():
		t.Fatal("program did not start")
	}

	select {
	case <-observed.stepSeen:
	case <-time.After(750 * time.Millisecond):
		t.Fatal("expected animation to start before redundant event processing")
	}

	program.Send(aiConversationEventMsg{conversation: &clientpb.AIConversation{
		ID:           "conv-1",
		OperatorName: "alice",
		UpdatedAt:    100,
	}})

	select {
	case <-getConversationStarted:
		t.Fatal("expected redundant pending event to avoid reloading the conversation")
	case <-time.After(250 * time.Millisecond):
	}

	select {
	case <-observed.stepSeen:
	case <-time.After(750 * time.Millisecond):
		t.Fatal("expected animation to keep advancing after the redundant pending event")
	}

	program.Quit()
	if err := <-errCh; err != nil {
		t.Fatalf("program run failed: %v", err)
	}
	close(blockConversationLoad)
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

type testErr string

func (e testErr) Error() string {
	return string(e)
}

func assertErr(message string) error {
	return testErr(message)
}

type observingAIProgramModel struct {
	inner       *aiModel
	initCmd     tea.Cmd
	started     chan struct{}
	stepSeen    chan struct{}
	submitted   chan struct{}
	startedOnce sync.Once
}

func (m *observingAIProgramModel) Init() tea.Cmd {
	return m.initCmd
}

func (m *observingAIProgramModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case aithinking.StepMsg:
		select {
		case m.stepSeen <- struct{}{}:
		default:
		}
	case aiPromptSubmittedMsg:
		select {
		case m.submitted <- struct{}{}:
		default:
		}
	}

	_, cmd := m.inner.Update(msg)
	return m, cmd
}

func (m *observingAIProgramModel) View() tea.View {
	m.startedOnce.Do(func() {
		close(m.started)
	})
	return m.inner.View()
}

func TestStartupConfigErrorModalIgnoresImmediateKeypress(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)

	updated, cmd := model.Update(aiStartupConfigInvalidMsg{err: "server AI configuration is invalid"})
	if cmd != nil {
		t.Fatalf("did not expect command when showing modal, got %v", cmd)
	}

	updatedModel := updated.(*aiModel)
	if updatedModel.modal == nil {
		t.Fatal("expected modal to be visible")
	}

	updated, cmd = updatedModel.Update(tea.KeyPressMsg{})
	if cmd != nil {
		t.Fatalf("expected immediate keypress to be ignored, got %v", cmd)
	}

	stillOpen := updated.(*aiModel)
	if stillOpen.modal == nil {
		t.Fatal("expected modal to remain visible after immediate keypress")
	}
}

func TestStartupConfigErrorModalQuitsAfterDismissDelay(t *testing.T) {
	model := newAIModel(nil, aiContext{}, nil)
	model.modal = &aiModalState{
		title:          "AI Configuration Error",
		body:           "server AI configuration is invalid",
		dismissReadyAt: time.Now().Add(-time.Second),
	}

	updated, cmd := model.Update(tea.KeyPressMsg{})
	if updated == nil {
		t.Fatal("expected model to be returned")
	}
	if cmd == nil {
		t.Fatal("expected keypress after dismiss delay to quit")
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %#v", msg)
	}
}
