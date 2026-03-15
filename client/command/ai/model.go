package ai

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	aithinking "github.com/bishopfox/sliver/client/spin/thinking"
	clienttheme "github.com/bishopfox/sliver/client/theme"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/charmbracelet/glamour"
	"golang.org/x/term"
	"google.golang.org/protobuf/proto"
)

const (
	aiMinWidth  = 72
	aiMinHeight = 18

	aiPaneHorizontalChrome = 4
	aiPaneVerticalChrome   = 2
	aiModalDismissDelay    = 250 * time.Millisecond
	aiWindowPollInterval   = 100 * time.Millisecond
)

type aiFocus int

const (
	aiFocusSidebar aiFocus = iota
	aiFocusTranscript
	aiFocusDetails
	aiFocusComposer
)

type aiStyles struct {
	badge            lipgloss.Style
	chip             lipgloss.Style
	chipMuted        lipgloss.Style
	subtleText       lipgloss.Style
	footerHint       lipgloss.Style
	pane             lipgloss.Style
	paneFocused      lipgloss.Style
	paneTitle        lipgloss.Style
	item             lipgloss.Style
	itemSelected     lipgloss.Style
	itemFocused      lipgloss.Style
	heading          lipgloss.Style
	cursor           lipgloss.Style
	inputText        lipgloss.Style
	inputPlaceholder lipgloss.Style
	roleUser         lipgloss.Style
	warning          lipgloss.Style
}

type aiLoadedMsg struct {
	providers     []*clientpb.AIProviderConfig
	config        *clientpb.AIConfigSummary
	conversations []*clientpb.AIConversation
	conversation  *clientpb.AIConversation
	selectedID    string
	status        string
}

type aiConversationCreatedMsg struct {
	conversationID string
	status         string
}

type aiPromptSubmittedMsg struct {
	conversationID string
	conversation   *clientpb.AIConversation
	message        *clientpb.AIConversationMessage
	status         string
}

type aiConversationEventMsg struct {
	conversation *clientpb.AIConversation
}

type aiListenerClosedMsg struct{}

type aiAsyncErrMsg struct {
	err error
}

type aiTranscriptRenderedMsg struct {
	key      string
	rendered string
	lines    []string
}

type aiWindowPollMsg struct {
	width  int
	height int
}

type aiStartupConfigInvalidMsg struct {
	err string
}

type aiModalState struct {
	title          string
	body           string
	dismissReadyAt time.Time
}

type aiModel struct {
	width                int
	height               int
	focus                aiFocus
	ctx                  aiContext
	con                  *console.SliverClient
	listener             <-chan *clientpb.Event
	providers            []*clientpb.AIProviderConfig
	config               *clientpb.AIConfigSummary
	conversations        []*clientpb.AIConversation
	currentConversation  *clientpb.AIConversation
	selectedConversation int
	input                []rune
	cursor               int
	status               string
	loading              bool
	awaitingResponse     bool
	submittingPrompt     bool
	pendingPrompt        string
	modal                *aiModalState
	thinkingAnim         *aithinking.Anim
	submitResults        chan tea.Msg
	styles               aiStyles
	transcriptVersion    int
	transcriptPendingKey string
	transcriptCacheKey   string
	transcriptCache      string
	transcriptCacheLines []string
}

func newAIModel(con *console.SliverClient, ctx aiContext, listener <-chan *clientpb.Event) *aiModel {
	return &aiModel{
		focus:         aiFocusSidebar,
		ctx:           ctx,
		con:           con,
		listener:      listener,
		status:        ctx.status,
		loading:       true,
		thinkingAnim:  newAIThinkingAnim("Working"),
		submitResults: make(chan tea.Msg),
		styles:        newAIStyles(),
	}
}

func newAIThinkingAnim(label string) *aithinking.Anim {
	return aithinking.New(aithinking.Settings{
		Size:        10,
		Label:       label,
		LabelColor:  clienttheme.DefaultMod(500),
		GradColorA:  clienttheme.Primary(),
		GradColorB:  clienttheme.Secondary(),
		CycleColors: true,
		SkipIntro:   true,
	})
}

func newAIStyles() aiStyles {
	return aiStyles{
		badge: lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.DefaultMod(900)).
			Background(clienttheme.Primary()).
			Padding(0, 1),
		chip: lipgloss.NewStyle().
			Foreground(clienttheme.PrimaryMod(700)).
			Background(clienttheme.PrimaryMod(50)).
			Padding(0, 1),
		chipMuted: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(700)).
			Background(clienttheme.DefaultMod(50)).
			Padding(0, 1),
		subtleText: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(500)),
		footerHint: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(700)),
		pane: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clienttheme.DefaultMod(300)).
			Padding(0, 1),
		paneFocused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clienttheme.Primary()).
			Padding(0, 1),
		paneTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.DefaultMod(900)),
		item: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(800)),
		itemSelected: lipgloss.NewStyle().
			Foreground(clienttheme.Primary()).
			Bold(true),
		itemFocused: lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.DefaultMod(900)).
			Background(clienttheme.PrimaryMod(800)),
		heading: lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.DefaultMod(900)),
		cursor: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(900)).
			Background(clienttheme.Primary()),
		inputText: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(900)),
		inputPlaceholder: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(400)),
		roleUser: lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.Warning()),
		warning: lipgloss.NewStyle().
			Foreground(clienttheme.Warning()).
			Bold(true),
	}
}

func (m *aiModel) Init() tea.Cmd {
	if m.width == 0 || m.height == 0 {
		if w, h, err := term.GetSize(0); err == nil && w > 0 && h > 0 {
			m.width = w
			m.height = h
		} else {
			m.width = 100
			m.height = 30
		}
	}

	cmds := []tea.Cmd{loadAIStateCmd(m.con, "")}
	if cmd := m.scheduleTranscriptRender(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	cmds = append(cmds, aiWindowPollCmd())
	if m.listener != nil {
		cmds = append(cmds, waitForAIConversationEventCmd(m.listener))
	}
	return tea.Batch(cmds...)
}

func (m *aiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.modal != nil {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			return m, m.applyWindowSize(msg.Width, msg.Height)
		case aiWindowPollMsg:
			return m, tea.Batch(m.windowSizeCmd(msg.width, msg.height), aiWindowPollCmd())
		case tea.KeyPressMsg:
			if time.Now().Before(m.modal.dismissReadyAt) {
				return m, nil
			}
			return m, tea.Quit
		default:
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, m.applyWindowSize(msg.Width, msg.Height)

	case aiWindowPollMsg:
		return m, tea.Batch(m.windowSizeCmd(msg.width, msg.height), aiWindowPollCmd())

	case aiStartupConfigInvalidMsg:
		m.loading = false
		m.status = msg.err
		m.modal = &aiModalState{
			title:          "AI Configuration Error",
			body:           msg.err,
			dismissReadyAt: time.Now().Add(aiModalDismissDelay),
		}
		return m, nil

	case aiLoadedMsg:
		m.loading = false
		m.providers = msg.providers
		m.config = msg.config
		m.conversations = msg.conversations
		m.currentConversation = msg.conversation
		if conversationContainsUserPrompt(m.currentConversation, m.pendingPrompt) {
			m.pendingPrompt = ""
		}
		m.selectedConversation = conversationIndexByID(m.conversations, selectedConversationID(msg.conversation, msg.selectedID))
		if m.selectedConversation < 0 {
			m.selectedConversation = 0
		}
		m.invalidateTranscriptCache()
		if strings.TrimSpace(msg.status) != "" {
			m.status = msg.status
		} else if len(m.conversations) == 0 {
			m.status = "No AI conversations yet. Press n or send a prompt to create one."
		} else if m.status == "" ||
			strings.HasPrefix(m.status, "Loading") ||
			strings.HasPrefix(m.status, "Refreshing") ||
			strings.HasPrefix(m.status, "Saved prompt to") ||
			strings.HasPrefix(m.status, "Waiting for AI response") {
			m.status = "Loaded AI conversations from the server."
		}
		return m, tea.Batch(
			m.syncAwaitingResponse(),
			m.scheduleTranscriptRender(),
		)

	case aiConversationCreatedMsg:
		m.loading = true
		m.status = msg.status
		return m, loadAIStateCmd(m.con, msg.conversationID)

	case aiPromptSubmittedMsg:
		m.loading = false
		m.submittingPrompt = false
		m.pendingPrompt = ""
		m.status = msg.status
		m.applyOptimisticPrompt(msg.conversation, msg.message)
		return m, tea.Batch(
			m.syncAwaitingResponse(),
			m.scheduleTranscriptRender(),
		)

	case aiConversationEventMsg:
		selectedID := m.selectedConversationID()
		if selectedID == "" && msg.conversation != nil {
			selectedID = msg.conversation.GetID()
		}
		if !isRelevantAIConversationEvent(msg.conversation, m.ctx.connection.Operator) {
			return m, waitForAIConversationEventCmd(m.listener)
		}
		if m.shouldSkipConversationEventReload(msg.conversation) {
			return m, waitForAIConversationEventCmd(m.listener)
		}

		m.loading = true
		m.status = "Conversation updated on the server. Refreshing..."
		return m, tea.Batch(
			waitForAIConversationEventCmd(m.listener),
			loadAIStateCmd(m.con, selectedID),
		)

	case aiListenerClosedMsg:
		m.status = "AI event stream closed. Reopen the AI TUI to resume live updates."
		return m, nil

	case aiAsyncErrMsg:
		m.loading = false
		if m.submittingPrompt {
			m.submittingPrompt = false
			m.awaitingResponse = false
			m.pendingPrompt = ""
		}
		if msg.err != nil {
			m.status = "AI sync failed: " + msg.err.Error()
		}
		return m, nil

	case aiTranscriptRenderedMsg:
		if msg.key != m.transcriptRenderKey(m.currentTranscriptWidth()) {
			if m.transcriptPendingKey == msg.key {
				m.transcriptPendingKey = ""
			}
			return m, nil
		}
		m.transcriptPendingKey = ""
		m.transcriptCacheKey = msg.key
		m.transcriptCache = msg.rendered
		m.transcriptCacheLines = append([]string(nil), msg.lines...)
		return m, nil

	case aithinking.StepMsg:
		if !m.awaitingResponse || m.thinkingAnim == nil {
			return m, nil
		}
		return m, m.thinkingAnim.Animate(msg)

	case tea.KeyPressMsg:
		key := msg.Key()
		if key.Mod.Contains(tea.ModCtrl) && key.Code == 'c' {
			return m, tea.Quit
		}
		if m.focus == aiFocusComposer {
			return m.handleComposerKey(key)
		}
		return m.handleGlobalKey(key)
	}

	return m, nil
}

func (m *aiModel) handleGlobalKey(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyTab:
		m.focus = (m.focus + 1) % 4
		m.status = "Focus moved to " + m.focus.String() + "."
		return m, nil

	case tea.KeyEsc:
		return m, tea.Quit

	case tea.KeyUp:
		if m.focus == aiFocusSidebar {
			return m, m.moveSelection(-1)
		}
		return m, nil

	case tea.KeyDown:
		if m.focus == aiFocusSidebar {
			return m, m.moveSelection(1)
		}
		return m, nil

	case tea.KeyEnter:
		if m.focus == aiFocusSidebar {
			m.focus = aiFocusTranscript
			m.status = "Conversation opened."
		}
		return m, nil
	}

	switch key.Text {
	case "q":
		return m, tea.Quit

	case "j":
		if m.focus == aiFocusSidebar {
			return m, m.moveSelection(1)
		}

	case "k":
		if m.focus == aiFocusSidebar {
			return m, m.moveSelection(-1)
		}

	case "n":
		if m.loading {
			m.status = "A conversation sync is already in progress."
			return m, nil
		}
		m.loading = true
		m.status = "Creating a new AI conversation..."
		return m, createConversationCmd(m.con, m.defaultProvider(), m.defaultModel(), "New conversation")

	case "r":
		m.loading = true
		m.status = "Refreshing AI conversations..."
		return m, loadAIStateCmd(m.con, m.selectedConversationID())
	}

	return m, nil
}

func (m *aiModel) handleComposerKey(key tea.Key) (tea.Model, tea.Cmd) {
	if key.Code == tea.KeyTab {
		m.focus = aiFocusSidebar
		m.status = "Focus moved to " + m.focus.String() + "."
		return m, nil
	}
	if key.Code == tea.KeyEsc {
		m.focus = aiFocusTranscript
		m.status = "Composer blurred. Press q or Esc again outside the composer to exit."
		return m, nil
	}
	if key.Mod.Contains(tea.ModCtrl) && key.Code == 'u' {
		m.input = nil
		m.cursor = 0
		m.status = "Prompt cleared."
		return m, nil
	}

	switch key.Code {
	case tea.KeyEnter:
		prompt := strings.TrimSpace(string(m.input))
		if prompt == "" {
			m.status = "Type a prompt first."
			return m, nil
		}
		if m.isBusy() {
			m.status = "Wait for the current AI request to finish before sending another prompt."
			return m, nil
		}

		m.submittingPrompt = true
		m.pendingPrompt = prompt
		m.status = "Saving prompt to the server..."
		m.input = nil
		m.cursor = 0
		submitResults := m.submitResults
		go func() {
			submitResults <- submitPromptMsg(m.con, m.currentConversation, m.defaultProvider(), m.defaultModel(), prompt)
		}()
		return m, tea.Batch(
			m.startAwaitingResponse(),
			waitForAISubmitResultCmd(submitResults),
		)

	case tea.KeyBackspace:
		if m.cursor > 0 {
			m.input = append(m.input[:m.cursor-1], m.input[m.cursor:]...)
			m.cursor--
		}

	case tea.KeyDelete:
		if m.cursor < len(m.input) {
			m.input = append(m.input[:m.cursor], m.input[m.cursor+1:]...)
		}

	case tea.KeyLeft:
		if m.cursor > 0 {
			m.cursor--
		}

	case tea.KeyRight:
		if m.cursor < len(m.input) {
			m.cursor++
		}

	case tea.KeyHome:
		m.cursor = 0

	case tea.KeyEnd:
		m.cursor = len(m.input)

	default:
		if key.Text != "" {
			if looksLikeTerminalResponseFragment(key.Text) {
				return m, nil
			}
			insert := []rune(key.Text)
			m.input = append(m.input[:m.cursor], append(insert, m.input[m.cursor:]...)...)
			m.cursor += len(insert)
		}
	}

	return m, nil
}

func (m *aiModel) View() tea.View {
	if m.modal != nil {
		return aiView(m.renderModal())
	}
	if m.width < aiMinWidth || m.height < aiMinHeight {
		return aiView(m.renderTooSmall())
	}

	headerHeight := 1
	if m.width >= 90 {
		headerHeight = 2
	}
	composerHeight := 4
	if m.width >= 96 {
		composerHeight = 5
	}
	footerHeight := 1
	bodyHeight := maxInt(1, m.height-headerHeight-composerHeight-footerHeight)

	header := m.renderHeader(headerHeight)
	body := m.renderBody(bodyHeight)
	composer := m.renderComposer(composerHeight)
	footer := m.renderFooter()

	frame := lipgloss.JoinVertical(lipgloss.Left, header, body, composer, footer)
	frame = lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, frame)
	return aiView(frame)
}

func (m *aiModel) renderModal() string {
	boxWidth := minInt(maxInt(20, m.width-4), 84)
	bodyWidth := maxInt(20, boxWidth-6)
	bodyLines := wrapText(m.modal.body, bodyWidth)

	lines := []string{
		m.styles.warning.Width(bodyWidth).Render(m.modal.title),
		"",
	}
	for _, line := range bodyLines {
		lines = append(lines, lipgloss.NewStyle().Width(bodyWidth).Render(line))
	}
	lines = append(lines,
		"",
		m.styles.subtleText.Width(bodyWidth).Render("Press any key to return to the REPL."),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(clienttheme.Warning()).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *aiModel) renderTooSmall() string {
	lines := []string{
		m.styles.badge.Render("SLIVER AI"),
		"",
		m.styles.warning.Render("Terminal too small for the AI conversation view."),
		m.styles.subtleText.Render("Resize to at least 72x18 to view the sidebar, markdown transcript, and context panes."),
	}
	return strings.Join(lines, "\n")
}

func (m *aiModel) renderHeader(height int) string {
	statusChip := "synced"
	if m.loading {
		statusChip = "syncing"
	} else if m.awaitingResponse {
		statusChip = strings.ToLower(m.pendingLabel())
	}

	pieces := []string{
		m.styles.badge.Render("SLIVER AI"),
		m.styles.chip.Render(statusChip),
		m.styles.chipMuted.Render(m.ctx.connection.Operator),
		m.styles.chipMuted.Render(m.ctx.target.Label),
		m.styles.chipMuted.Render(m.layoutName()),
	}
	row := fitStyledPieces(m.width, pieces)
	row = lipgloss.NewStyle().Width(m.width).Render(row)
	if height == 1 {
		return row
	}

	subtitle := "Server-backed AI conversation threads with live sync across connected clients."
	return lipgloss.JoinVertical(
		lipgloss.Left,
		row,
		m.styles.subtleText.Width(m.width).Render(truncateText(subtitle, m.width)),
	)
}

func (m *aiModel) renderBody(height int) string {
	switch {
	case m.width >= 110:
		sidebarWidth := clampInt(m.width/5, 24, 28)
		detailsWidth := clampInt(m.width/4, 28, 34)
		transcriptWidth := maxInt(34, m.width-sidebarWidth-detailsWidth)
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderSidebar(sidebarWidth, height),
			m.renderTranscript(transcriptWidth, height),
			m.renderDetails(detailsWidth, height),
		)

	case m.width >= 78:
		sidebarWidth := clampInt(m.width/4, 24, 28)
		mainWidth := maxInt(34, m.width-sidebarWidth)
		detailsHeight := clampInt(height/3, 6, 9)
		if height-detailsHeight < 7 {
			detailsHeight = maxInt(4, height-7)
		}
		chatHeight := maxInt(6, height-detailsHeight)
		mainColumn := lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderTranscript(mainWidth, chatHeight),
			m.renderDetails(mainWidth, detailsHeight),
		)
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderSidebar(sidebarWidth, height),
			mainColumn,
		)

	default:
		sidebarHeight := clampInt(height/4, 5, 7)
		detailsHeight := clampInt(height/4, 5, 8)
		transcriptHeight := maxInt(6, height-sidebarHeight-detailsHeight)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderSidebar(m.width, sidebarHeight),
			m.renderTranscript(m.width, transcriptHeight),
			m.renderDetails(m.width, detailsHeight),
		)
	}
}

func (m *aiModel) renderSidebar(width, height int) string {
	innerWidth := innerPaneWidth(width)
	innerHeight := innerPaneHeight(height)
	lines := []string{
		m.styles.paneTitle.Render("Conversations"),
	}

	if len(m.conversations) == 0 {
		lines = append(lines,
			"",
			m.styles.subtleText.Width(innerWidth).Render(truncateText("No stored AI conversations yet.", innerWidth)),
			m.styles.subtleText.Width(innerWidth).Render(truncateText("Press n to create one or send a prompt below.", innerWidth)),
		)
		return m.renderPane(width, height, aiFocusSidebar, headLines(lines, innerHeight))
	}

	bodyHeight := maxInt(1, innerHeight-1)
	for i, conversation := range m.conversations {
		line := conversationListLabel(conversation, maxInt(1, innerWidth-2))
		if i == m.selectedConversation {
			line = "> " + line
			style := m.styles.itemSelected
			if m.focus == aiFocusSidebar {
				style = m.styles.itemFocused
			}
			lines = append(lines, style.Width(innerWidth).Render(line))
		} else {
			lines = append(lines, m.styles.item.Width(innerWidth).Render("  "+line))
		}
		if len(lines)-1 >= bodyHeight {
			break
		}
	}

	if remaining := bodyHeight - (len(lines) - 1); remaining > 2 {
		lines = append(lines, "")
		hints := []string{
			"n new thread",
			"r refresh",
			"j/k move",
		}
		for _, hint := range hints {
			if len(lines)-1 >= bodyHeight {
				break
			}
			lines = append(lines, m.styles.subtleText.Width(innerWidth).Render(truncateText(hint, innerWidth)))
		}
	}

	return m.renderPane(width, height, aiFocusSidebar, lines)
}

func (m *aiModel) renderTranscript(width, height int) string {
	innerWidth := innerPaneWidth(width)
	innerHeight := innerPaneHeight(height)

	lines := []string{
		m.styles.paneTitle.Render("Conversation"),
	}

	title := "No conversation selected"
	subtitle := "Create a thread or choose one from the sidebar."
	if m.currentConversation != nil {
		title = conversationTitle(m.currentConversation)
		subtitle = conversationSubtitle(m.currentConversation)
	}

	lines = append(lines,
		m.styles.heading.Width(innerWidth).Render(truncateText(title, innerWidth)),
		m.styles.subtleText.Width(innerWidth).Render(truncateText(subtitle, innerWidth)),
	)

	bodyHeight := maxInt(1, innerHeight-len(lines))
	contentLines := m.renderTranscriptDisplayContentLines(innerWidth)
	lines = append(lines, tailLines(contentLines, bodyHeight)...)

	return m.renderPane(width, height, aiFocusTranscript, lines)
}

func (m *aiModel) renderDetails(width, height int) string {
	innerWidth := innerPaneWidth(width)
	innerHeight := innerPaneHeight(height)

	lines := []string{
		m.styles.paneTitle.Render("Context"),
		m.styles.heading.Render("Target"),
	}

	targetLines := []string{
		m.ctx.target.Label,
		m.ctx.target.Host,
		m.ctx.target.OS + "/" + m.ctx.target.Arch + " via " + m.ctx.target.C2,
	}
	targetLines = append(targetLines, m.ctx.target.Details...)
	for _, line := range targetLines {
		lines = append(lines, m.styles.item.Width(innerWidth).Render(truncateText(line, innerWidth)))
	}

	lines = append(lines, m.styles.heading.Render("Connection"))
	connectionLines := []string{
		m.ctx.connection.Profile,
		m.ctx.connection.Server,
		"operator: " + m.ctx.connection.Operator,
		"state: " + m.ctx.connection.State,
	}
	for _, line := range connectionLines {
		lines = append(lines, m.styles.subtleText.Width(innerWidth).Render(truncateText(line, innerWidth)))
	}

	lines = append(lines, m.styles.heading.Render("Providers"))
	if len(m.providers) == 0 {
		lines = append(lines, m.styles.subtleText.Width(innerWidth).Render("No AI providers reported by the server."))
	} else {
		for _, provider := range m.providers {
			lines = append(lines, m.styles.subtleText.Width(innerWidth).Render(truncateText(providerDisplay(provider), innerWidth)))
		}
	}

	lines = append(lines, m.styles.heading.Render("Defaults"))
	if m.config == nil {
		lines = append(lines, m.styles.subtleText.Width(innerWidth).Render("AI defaults unavailable."))
	} else {
		defaultLines := []string{
			"provider: " + fallback(m.config.GetProvider(), "<unset>"),
			"model: " + fallback(m.config.GetModel(), "provider default"),
			"thinking: " + fallback(m.config.GetThinkingLevel(), "provider default"),
		}
		for _, line := range defaultLines {
			lines = append(lines, m.styles.subtleText.Width(innerWidth).Render(truncateText(line, innerWidth)))
		}
	}

	if m.currentConversation != nil {
		lines = append(lines, m.styles.heading.Render("Thread"))
		threadLines := []string{
			"id: " + shortenID(m.currentConversation.GetID()),
			"provider: " + fallback(m.currentConversation.GetProvider(), "<unset>"),
			"model: " + fallback(m.currentConversation.GetModel(), "<default>"),
			fmt.Sprintf("messages: %d", len(m.currentConversation.GetMessages())),
			"updated: " + formatUnix(m.currentConversation.GetUpdatedAt()),
		}
		for _, line := range threadLines {
			lines = append(lines, m.styles.subtleText.Width(innerWidth).Render(truncateText(line, innerWidth)))
		}
	}

	return m.renderPane(width, height, aiFocusDetails, headLines(lines, innerHeight))
}

func (m *aiModel) renderComposer(height int) string {
	innerWidth := innerPaneWidth(m.width)
	innerHeight := innerPaneHeight(height)

	lines := []string{
		m.styles.paneTitle.Render("Composer"),
		m.renderInputLine(innerWidth),
	}

	status := truncateText(m.status, innerWidth)
	lines = append(lines, m.styles.subtleText.Width(innerWidth).Render(status))

	if innerHeight-len(lines) > 0 {
		help := "tab focus  enter send  n new-thread  r refresh  ctrl+u clear  esc blur  q quit"
		lines = append(lines, m.styles.subtleText.Width(innerWidth).Render(truncateText(help, innerWidth)))
	}

	return m.renderPane(m.width, height, aiFocusComposer, headLines(lines, innerHeight))
}

func (m *aiModel) renderTranscriptContentLines(width int) []string {
	contentLines := append([]string(nil), m.renderTranscriptMarkdownLines(width)...)
	pendingUserLines := m.renderPendingPromptLines(width)
	pendingLines := m.renderAwaitingResponseLines(width)

	for _, block := range [][]string{pendingUserLines, pendingLines} {
		if len(block) == 0 {
			continue
		}
		if len(contentLines) > 0 {
			contentLines = append(contentLines, "")
		}
		contentLines = append(contentLines, block...)
	}
	return contentLines
}

func (m *aiModel) renderTranscriptDisplayContentLines(width int) []string {
	contentLines := append([]string(nil), m.transcriptDisplayLines(width)...)
	pendingUserLines := m.renderPendingPromptLines(width)
	pendingLines := m.renderAwaitingResponseLines(width)

	for _, block := range [][]string{pendingUserLines, pendingLines} {
		if len(block) == 0 {
			continue
		}
		if len(contentLines) > 0 {
			contentLines = append(contentLines, "")
		}
		contentLines = append(contentLines, block...)
	}
	return contentLines
}

func (m *aiModel) renderPendingPromptLines(width int) []string {
	prompt := strings.TrimSpace(m.pendingPrompt)
	if prompt == "" {
		return nil
	}

	label := messageBlockLabel(m.currentConversation, &clientpb.AIConversationMessage{
		OperatorName: m.ctx.connection.Operator,
		Role:         "user",
	})
	lines := []string{
		m.styles.subtleText.Width(width).Render("[" + label + "]"),
	}
	for _, line := range wrapText(prompt, maxInt(1, width)) {
		lines = append(lines, lipgloss.NewStyle().Width(width).Render(line))
	}
	if m.currentConversation != nil && len(m.currentConversation.GetMessages()) > 0 {
		lines = append([]string{m.styles.subtleText.Width(width).Render(strings.Repeat("-", maxInt(8, minInt(width, 32))))}, lines...)
	}
	return lines
}

func (m *aiModel) renderAwaitingResponseLines(width int) []string {
	if !m.awaitingResponse || m.thinkingAnim == nil {
		return nil
	}

	lines := []string{
		m.styles.subtleText.Width(width).Render("[AI]"),
		lipgloss.NewStyle().Width(width).Render(m.thinkingAnim.Render()),
	}

	if m.currentConversation != nil && len(m.currentConversation.GetMessages()) > 0 {
		lines = append([]string{m.styles.subtleText.Width(width).Render(strings.Repeat("-", maxInt(8, minInt(width, 32))))}, lines...)
	}
	return lines
}

func (m *aiModel) renderFooter() string {
	pieces := []string{
		m.styles.chipMuted.Render("focus: " + m.focus.String()),
		m.styles.footerHint.Render("tab: next pane"),
		m.styles.footerHint.Render("n: new thread"),
		m.styles.footerHint.Render("r: refresh"),
		m.styles.footerHint.Render("enter: select/send"),
		m.styles.footerHint.Render("q: quit"),
	}
	return lipgloss.NewStyle().Width(m.width).Render(fitStyledPieces(m.width, pieces))
}

func (m *aiModel) renderPane(width, height int, focus aiFocus, lines []string) string {
	body := strings.Join(headLines(lines, innerPaneHeight(height)), "\n")
	style := m.styles.pane
	if m.focus == focus {
		style = m.styles.paneFocused
	}
	return style.Width(width).Height(height).Render(body)
}

func (m *aiModel) renderInputLine(width int) string {
	prefix := m.styles.roleUser.Render("YOU")
	available := maxInt(1, width-4)
	content := m.renderInputContent(available)
	return prefix + " " + content
}

func (m *aiModel) renderInputContent(width int) string {
	if len(m.input) == 0 {
		placeholder := truncateText("Ask Sliver AI about the active target...", maxInt(1, width))
		if m.focus == aiFocusComposer {
			if width == 1 {
				return m.styles.cursor.Render(" ")
			}
			return m.styles.cursor.Render(" ") + m.styles.inputPlaceholder.Render(truncateText(placeholder, width-1))
		}
		return m.styles.inputPlaceholder.Render(placeholder)
	}

	visible, cursor := inputWindow(m.input, m.cursor, width)
	var b strings.Builder
	for i, r := range visible {
		ch := string(r)
		if i == cursor && m.focus == aiFocusComposer {
			b.WriteString(m.styles.cursor.Render(ch))
			continue
		}
		b.WriteString(m.styles.inputText.Render(ch))
	}
	if cursor == len(visible) && m.focus == aiFocusComposer && lipgloss.Width(b.String()) < width {
		b.WriteString(m.styles.cursor.Render(" "))
	}
	return b.String()
}

func (m *aiModel) isBusy() bool {
	return m.loading || m.awaitingResponse
}

func (m *aiModel) moveSelection(delta int) tea.Cmd {
	if len(m.conversations) == 0 {
		return nil
	}

	next := clampInt(m.selectedConversation+delta, 0, len(m.conversations)-1)
	if next == m.selectedConversation {
		return nil
	}

	m.selectedConversation = next
	m.loading = true
	m.status = "Loading conversation..."
	return loadAIStateCmd(m.con, m.conversations[next].GetID())
}

func (m *aiModel) selectedConversationID() string {
	if m.currentConversation != nil && strings.TrimSpace(m.currentConversation.GetID()) != "" {
		return m.currentConversation.GetID()
	}
	if m.selectedConversation >= 0 && m.selectedConversation < len(m.conversations) {
		return m.conversations[m.selectedConversation].GetID()
	}
	return ""
}

func (m *aiModel) defaultProvider() string {
	if m.config != nil && strings.TrimSpace(m.config.GetProvider()) != "" {
		return strings.TrimSpace(m.config.GetProvider())
	}
	for _, provider := range m.providers {
		if provider.GetConfigured() && strings.TrimSpace(provider.GetName()) != "" {
			return provider.GetName()
		}
	}
	for _, provider := range m.providers {
		if strings.TrimSpace(provider.GetName()) != "" {
			return provider.GetName()
		}
	}
	return "openai"
}

func (m *aiModel) defaultModel() string {
	if m.config == nil {
		return ""
	}
	return strings.TrimSpace(m.config.GetModel())
}

func (m *aiModel) pendingLabel() string {
	if m.config != nil && strings.TrimSpace(m.config.GetThinkingLevel()) != "" {
		return "Thinking"
	}
	return "Working"
}

func (m *aiModel) pendingStatus() string {
	return "Waiting for AI response..."
}

func (m *aiModel) applyWindowSize(width, height int) tea.Cmd {
	if width <= 0 || height <= 0 {
		return nil
	}
	if m.width == width && m.height == height {
		return nil
	}

	m.width = width
	m.height = height
	return m.scheduleTranscriptRender()
}

func (m *aiModel) windowSizeCmd(width, height int) tea.Cmd {
	if width <= 0 || height <= 0 {
		return nil
	}
	if m.width == width && m.height == height {
		return nil
	}
	return func() tea.Msg {
		return tea.WindowSizeMsg{Width: width, Height: height}
	}
}

func (m *aiModel) startAwaitingResponse() tea.Cmd {
	wasAwaiting := m.awaitingResponse
	m.awaitingResponse = true

	if strings.TrimSpace(m.status) == "" || strings.HasPrefix(m.status, "Saving prompt") {
		m.status = m.pendingStatus()
	}

	if !wasAwaiting {
		m.thinkingAnim = newAIThinkingAnim(m.pendingLabel())
	}
	if m.thinkingAnim != nil {
		m.thinkingAnim.SetLabel(m.pendingLabel())
	}

	if !wasAwaiting && m.thinkingAnim != nil {
		return m.thinkingAnim.Start()
	}
	return nil
}

func (m *aiModel) syncAwaitingResponse() tea.Cmd {
	wasAwaiting := m.awaitingResponse
	m.awaitingResponse = conversationAwaitingResponse(m.currentConversation)

	if m.awaitingResponse && !wasAwaiting {
		m.thinkingAnim = newAIThinkingAnim(m.pendingLabel())
	}
	if m.thinkingAnim != nil {
		m.thinkingAnim.SetLabel(m.pendingLabel())
	}

	if m.awaitingResponse && (m.status == "" || strings.HasPrefix(m.status, "Loaded AI conversations") || strings.HasPrefix(m.status, "Conversation updated")) {
		m.status = m.pendingStatus()
	}

	if m.awaitingResponse && !wasAwaiting && m.thinkingAnim != nil {
		return m.thinkingAnim.Start()
	}
	return nil
}

func (m *aiModel) applyOptimisticPrompt(conversation *clientpb.AIConversation, message *clientpb.AIConversationMessage) {
	if conversation == nil && message == nil {
		return
	}

	conversationID := ""
	if conversation != nil {
		conversationID = strings.TrimSpace(conversation.GetID())
	}
	if conversationID == "" && message != nil {
		conversationID = strings.TrimSpace(message.GetConversationID())
	}
	if conversationID == "" {
		return
	}

	optimistic := cloneConversation(m.currentConversation)
	if optimistic == nil || strings.TrimSpace(optimistic.GetID()) != conversationID {
		optimistic = cloneConversation(conversation)
	}
	if optimistic == nil {
		optimistic = &clientpb.AIConversation{ID: conversationID}
	}

	if conversation != nil {
		mergeConversationMetadata(optimistic, conversation)
	}
	if optimistic.GetOperatorName() == "" {
		optimistic.OperatorName = strings.TrimSpace(m.ctx.connection.Operator)
	}

	if message != nil {
		pending := cloneConversationMessage(message)
		if pending != nil {
			if strings.TrimSpace(pending.GetConversationID()) == "" {
				pending.ConversationID = conversationID
			}
			if strings.TrimSpace(pending.GetOperatorName()) == "" {
				pending.OperatorName = optimistic.GetOperatorName()
			}
			if strings.TrimSpace(pending.GetProvider()) == "" {
				pending.Provider = optimistic.GetProvider()
			}
			if strings.TrimSpace(pending.GetModel()) == "" {
				pending.Model = optimistic.GetModel()
			}

			optimistic.Messages = append(optimistic.Messages, pending)

			lastUpdate := maxInt64(pending.GetUpdatedAt(), pending.GetCreatedAt())
			if lastUpdate == 0 {
				lastUpdate = time.Now().Unix()
			}
			if lastUpdate > optimistic.GetUpdatedAt() {
				optimistic.UpdatedAt = lastUpdate
			}
		}
	}

	m.currentConversation = optimistic
	m.selectedConversation = m.upsertConversation(optimistic)
	m.invalidateTranscriptCache()
}

func (m *aiModel) upsertConversation(conversation *clientpb.AIConversation) int {
	if conversation == nil {
		return m.selectedConversation
	}

	summary := cloneConversation(conversation)
	if summary == nil {
		return m.selectedConversation
	}

	idx := conversationIndexByID(m.conversations, summary.GetID())
	if idx >= 0 {
		m.conversations[idx] = summary
		return idx
	}

	m.conversations = append([]*clientpb.AIConversation{summary}, m.conversations...)
	return 0
}

func (m *aiModel) renderTranscriptMarkdown(width int) string {
	lines := m.renderTranscriptMarkdownLines(width)
	return strings.Join(lines, "\n")
}

func (m *aiModel) renderTranscriptMarkdownLines(width int) []string {
	key := m.transcriptRenderKey(width)
	if key == m.transcriptCacheKey && len(m.transcriptCacheLines) > 0 {
		return m.transcriptCacheLines
	}

	markdown := buildConversationMarkdown(m.currentConversation)
	rendered, err := renderMarkdownWithGlow(width, markdown)
	if err != nil {
		rendered = markdown
	}

	m.transcriptCacheKey = key
	m.transcriptCache = strings.TrimSpace(rendered)
	if m.transcriptCache == "" {
		m.transcriptCache = "_No messages yet._"
	}
	m.transcriptCacheLines = strings.Split(m.transcriptCache, "\n")
	return m.transcriptCacheLines
}

func (m *aiModel) transcriptRenderKey(width int) string {
	return fmt.Sprintf("%d:%d", m.transcriptVersion, width)
}

func (m *aiModel) invalidateTranscriptCache() {
	m.transcriptVersion++
	m.transcriptPendingKey = ""
	m.transcriptCacheKey = ""
	m.transcriptCache = ""
	m.transcriptCacheLines = nil
}

func (m *aiModel) transcriptDisplayLines(width int) []string {
	key := m.transcriptRenderKey(width)
	if key == m.transcriptCacheKey && len(m.transcriptCacheLines) > 0 {
		return m.transcriptCacheLines
	}

	placeholder := "Rendering transcript..."
	if m.currentConversation == nil {
		placeholder = "Loading conversation..."
	}
	return []string{
		m.styles.subtleText.Width(maxInt(1, width)).Render(truncateText(placeholder, maxInt(1, width))),
	}
}

func (m *aiModel) currentTranscriptWidth() int {
	paneWidth := 0
	switch {
	case m.width >= 110:
		sidebarWidth := clampInt(m.width/5, 24, 28)
		detailsWidth := clampInt(m.width/4, 28, 34)
		paneWidth = maxInt(34, m.width-sidebarWidth-detailsWidth)
	case m.width >= 78:
		sidebarWidth := clampInt(m.width/4, 24, 28)
		paneWidth = maxInt(34, m.width-sidebarWidth)
	default:
		paneWidth = maxInt(1, m.width)
	}
	return maxInt(1, innerPaneWidth(paneWidth))
}

func (m *aiModel) scheduleTranscriptRender() tea.Cmd {
	if m.width <= 0 {
		return nil
	}

	width := m.currentTranscriptWidth()
	key := m.transcriptRenderKey(width)
	if key == m.transcriptCacheKey && len(m.transcriptCacheLines) > 0 {
		return nil
	}
	if key == m.transcriptPendingKey {
		return nil
	}

	m.transcriptPendingKey = key
	conversation := cloneConversation(m.currentConversation)
	markdown := buildConversationMarkdown(conversation)

	return func() tea.Msg {
		rendered, err := renderMarkdownWithGlow(width, markdown)
		if err != nil {
			rendered = markdown
		}
		rendered = strings.TrimSpace(rendered)
		if rendered == "" {
			rendered = "_No messages yet._"
		}
		return aiTranscriptRenderedMsg{
			key:      key,
			rendered: rendered,
			lines:    strings.Split(rendered, "\n"),
		}
	}
}

func (m *aiModel) shouldSkipConversationEventReload(conversation *clientpb.AIConversation) bool {
	if conversation == nil || !m.awaitingResponse {
		return false
	}
	if !conversationAwaitingResponse(m.currentConversation) {
		return false
	}

	currentID := strings.TrimSpace(m.currentConversation.GetID())
	eventID := strings.TrimSpace(conversation.GetID())
	if currentID == "" || eventID == "" || currentID != eventID {
		return false
	}

	eventUpdatedAt := conversation.GetUpdatedAt()
	currentUpdatedAt := m.currentConversation.GetUpdatedAt()
	if eventUpdatedAt > currentUpdatedAt {
		return false
	}

	// Ignore redundant sync events for the same optimistic user message and keep
	// the local pending animation running until the assistant reply arrives.
	return true
}

func (m aiFocus) String() string {
	switch m {
	case aiFocusSidebar:
		return "sidebar"
	case aiFocusTranscript:
		return "conversation"
	case aiFocusDetails:
		return "context"
	case aiFocusComposer:
		return "composer"
	default:
		return "unknown"
	}
}

func (m *aiModel) layoutName() string {
	switch {
	case m.width >= 110:
		return "three-pane"
	case m.width >= 78:
		return "split"
	default:
		return "stacked"
	}
}

func aiView(content string) tea.View {
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func aiWindowPollCmd() tea.Cmd {
	return tea.Tick(aiWindowPollInterval, func(time.Time) tea.Msg {
		width, height, ok := currentTerminalSize()
		if !ok {
			return aiWindowPollMsg{}
		}
		return aiWindowPollMsg{width: width, height: height}
	})
}

func currentTerminalSize() (int, int, bool) {
	for _, fd := range []int{1, 0, 2} {
		width, height, err := term.GetSize(fd)
		if err == nil && width > 0 && height > 0 {
			return width, height, true
		}
	}
	return 0, 0, false
}

func loadAIStateCmd(con *console.SliverClient, selectedID string) tea.Cmd {
	return func() tea.Msg {
		if con == nil || con.Rpc == nil {
			return aiAsyncErrMsg{err: fmt.Errorf("AI RPC client is unavailable")}
		}

		grpcCtx, cancel := con.GrpcContext(nil)
		defer cancel()

		providersResp, err := con.Rpc.GetAIProviders(grpcCtx, &commonpb.Empty{})
		if err != nil {
			return aiAsyncErrMsg{err: err}
		}
		config := safeAIConfigSummary(providersResp)
		if !config.GetValid() {
			return aiStartupConfigInvalidMsg{err: aiConfigError(config)}
		}

		conversationsResp, err := con.Rpc.GetAIConversations(grpcCtx, &commonpb.Empty{})
		if err != nil {
			return aiAsyncErrMsg{err: err}
		}

		conversations := conversationsResp.GetConversations()
		activeID := strings.TrimSpace(selectedID)
		status := ""
		if len(conversations) == 0 {
			createdConversation, err := con.Rpc.SaveAIConversation(grpcCtx, &clientpb.AIConversation{
				Provider: config.GetProvider(),
				Model:    config.GetModel(),
				Title:    "New conversation",
			})
			if err != nil {
				return aiAsyncErrMsg{err: err}
			}
			conversations = []*clientpb.AIConversation{createdConversation}
			activeID = createdConversation.GetID()
			status = "Created a new AI conversation."
		}
		if activeID == "" && len(conversations) > 0 {
			activeID = conversations[0].GetID()
		}

		var conversation *clientpb.AIConversation
		if activeID != "" {
			conversation, err = con.Rpc.GetAIConversation(grpcCtx, &clientpb.AIConversationReq{
				ID:              activeID,
				IncludeMessages: true,
			})
			if err != nil && len(conversations) > 0 {
				fallbackID := conversations[0].GetID()
				if fallbackID != activeID {
					conversation, err = con.Rpc.GetAIConversation(grpcCtx, &clientpb.AIConversationReq{
						ID:              fallbackID,
						IncludeMessages: true,
					})
					activeID = fallbackID
				}
			}
			if err != nil {
				return aiAsyncErrMsg{err: err}
			}
		}

		return aiLoadedMsg{
			providers:     providersResp.GetProviders(),
			config:        config,
			conversations: conversations,
			conversation:  conversation,
			selectedID:    activeID,
			status:        status,
		}
	}
}

func createConversationCmd(con *console.SliverClient, provider string, model string, title string) tea.Cmd {
	return func() tea.Msg {
		if con == nil || con.Rpc == nil {
			return aiAsyncErrMsg{err: fmt.Errorf("AI RPC client is unavailable")}
		}

		grpcCtx, cancel := con.GrpcContext(nil)
		defer cancel()

		conversation, err := con.Rpc.SaveAIConversation(grpcCtx, &clientpb.AIConversation{
			Provider: provider,
			Model:    strings.TrimSpace(model),
			Title:    strings.TrimSpace(title),
		})
		if err != nil {
			return aiAsyncErrMsg{err: err}
		}

		return aiConversationCreatedMsg{
			conversationID: conversation.GetID(),
			status:         "Created a new AI conversation.",
		}
	}
}

func submitPromptCmd(con *console.SliverClient, conversation *clientpb.AIConversation, provider string, model string, prompt string) tea.Cmd {
	return func() tea.Msg {
		return submitPromptMsg(con, conversation, provider, model, prompt)
	}
}

func submitPromptMsg(con *console.SliverClient, conversation *clientpb.AIConversation, provider string, model string, prompt string) tea.Msg {
	if con == nil || con.Rpc == nil {
		return aiAsyncErrMsg{err: fmt.Errorf("AI RPC client is unavailable")}
	}

	grpcCtx, cancel := con.GrpcContext(nil)
	defer cancel()

	activeConversation := conversation
	var err error
	if activeConversation == nil || strings.TrimSpace(activeConversation.GetID()) == "" {
		activeConversation, err = con.Rpc.SaveAIConversation(grpcCtx, &clientpb.AIConversation{
			Provider: provider,
			Model:    strings.TrimSpace(model),
			Title:    promptConversationTitle(prompt),
		})
		if err != nil {
			return aiAsyncErrMsg{err: err}
		}
	}

	messageProvider := strings.TrimSpace(activeConversation.GetProvider())
	if messageProvider == "" {
		messageProvider = strings.TrimSpace(provider)
	}

	message := &clientpb.AIConversationMessage{
		ConversationID: activeConversation.GetID(),
		Provider:       messageProvider,
		Model:          activeConversation.GetModel(),
		Role:           "user",
		Content:        prompt,
	}

	savedMessage, err := con.Rpc.SaveAIConversationMessage(grpcCtx, message)
	if err != nil {
		return aiAsyncErrMsg{err: err}
	}
	if savedMessage == nil {
		savedMessage = cloneConversationMessage(message)
	}

	return aiPromptSubmittedMsg{
		conversationID: activeConversation.GetID(),
		conversation:   cloneConversation(activeConversation),
		message:        savedMessage,
		status:         "Saved prompt to " + conversationTitle(activeConversation) + ". Waiting for AI response...",
	}
}

func waitForAISubmitResultCmd(results <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		if results == nil {
			return aiAsyncErrMsg{err: fmt.Errorf("AI submit queue is unavailable")}
		}
		msg, ok := <-results
		if !ok {
			return aiAsyncErrMsg{err: fmt.Errorf("AI submit queue closed")}
		}
		return msg
	}
}

func waitForAIConversationEventCmd(listener <-chan *clientpb.Event) tea.Cmd {
	return func() tea.Msg {
		if listener == nil {
			return nil
		}

		for {
			event, ok := <-listener
			if !ok {
				return aiListenerClosedMsg{}
			}
			if event == nil || event.GetEventType() != consts.AIConversationEvent {
				continue
			}

			conversation := &clientpb.AIConversation{}
			if len(event.GetData()) > 0 {
				if err := proto.Unmarshal(event.GetData(), conversation); err != nil {
					continue
				}
			}

			return aiConversationEventMsg{conversation: conversation}
		}
	}
}

func buildConversationMarkdown(conversation *clientpb.AIConversation) string {
	if conversation == nil {
		return "### No conversation selected\n\nCreate a new thread with `n` or submit a prompt to start one."
	}

	var markdown strings.Builder
	if summary := strings.TrimSpace(conversation.GetSummary()); summary != "" {
		markdown.WriteString(summary)
		markdown.WriteString("\n\n---\n\n")
	}
	if systemPrompt := strings.TrimSpace(conversation.GetSystemPrompt()); systemPrompt != "" {
		markdown.WriteString("## System Prompt\n\n```text\n")
		markdown.WriteString(systemPrompt)
		markdown.WriteString("\n```\n\n---\n\n")
	}

	messages := conversation.GetMessages()
	if len(messages) == 0 {
		markdown.WriteString("_No messages yet. Type a prompt below to start this thread._")
		return markdown.String()
	}

	for i, message := range messages {
		meta := []string{}
		if ts := formatUnix(message.GetCreatedAt()); ts != "<unknown>" {
			meta = append(meta, ts)
		}
		if provider := strings.TrimSpace(message.GetProvider()); provider != "" {
			meta = append(meta, provider)
		}
		if model := strings.TrimSpace(message.GetModel()); model != "" {
			meta = append(meta, model)
		}
		markdown.WriteString(conversationMessageMarkdown(conversation, message, meta))
		if i < len(messages)-1 {
			markdown.WriteString("\n\n***\n\n")
		}
	}

	return markdown.String()
}

func renderMarkdownWithGlow(width int, markdown string) (string, error) {
	// Glow renders markdown through Glamour; mirror that render path here so AI
	// conversation content is displayed with the same terminal markdown renderer.
	// Use a fixed style to avoid Glamour's auto-style terminal color queries,
	// which can leak OSC/CSI responses back into the TUI input stream.
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(maxInt(8, width)),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return "", err
	}

	rendered, err := renderer.Render(markdown)
	if err != nil {
		return "", err
	}

	lines := strings.Split(rendered, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n"), nil
}

func isRelevantAIConversationEvent(conversation *clientpb.AIConversation, operatorName string) bool {
	if conversation == nil {
		return true
	}
	eventOperator := strings.TrimSpace(conversation.GetOperatorName())
	currentOperator := strings.TrimSpace(operatorName)
	if eventOperator == "" || currentOperator == "" {
		return true
	}
	return eventOperator == currentOperator
}

func looksLikeTerminalResponseFragment(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	if strings.Contains(text, "rgb:") {
		return true
	}
	if strings.HasPrefix(text, "]10;") || strings.HasPrefix(text, "]11;") || strings.HasPrefix(text, "]12;") {
		return true
	}
	if strings.HasPrefix(text, "[") && strings.HasSuffix(text, "R") {
		for _, r := range text[1 : len(text)-1] {
			if (r < '0' || r > '9') && r != ';' {
				return false
			}
		}
		return true
	}
	return false
}

func conversationContainsUserPrompt(conversation *clientpb.AIConversation, prompt string) bool {
	prompt = strings.TrimSpace(prompt)
	if conversation == nil || prompt == "" {
		return false
	}

	for _, message := range conversation.GetMessages() {
		if message == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(message.GetRole()), "user") &&
			strings.TrimSpace(message.GetContent()) == prompt {
			return true
		}
	}
	return false
}

func selectedConversationID(conversation *clientpb.AIConversation, fallbackID string) string {
	if conversation != nil && strings.TrimSpace(conversation.GetID()) != "" {
		return conversation.GetID()
	}
	return strings.TrimSpace(fallbackID)
}

func conversationIndexByID(conversations []*clientpb.AIConversation, id string) int {
	if strings.TrimSpace(id) == "" {
		if len(conversations) == 0 {
			return 0
		}
		return 0
	}
	for i, conversation := range conversations {
		if conversation.GetID() == id {
			return i
		}
	}
	return -1
}

func conversationTitle(conversation *clientpb.AIConversation) string {
	if conversation == nil {
		return "Untitled conversation"
	}
	title := strings.TrimSpace(conversation.GetTitle())
	if title != "" {
		return title
	}
	if shortID := shortenID(conversation.GetID()); shortID != "" {
		return "Conversation " + shortID
	}
	return "Untitled conversation"
}

func conversationSubtitle(conversation *clientpb.AIConversation) string {
	if conversation == nil {
		return ""
	}

	parts := []string{}
	if provider := strings.TrimSpace(conversation.GetProvider()); provider != "" {
		parts = append(parts, provider)
	}
	if model := strings.TrimSpace(conversation.GetModel()); model != "" {
		parts = append(parts, model)
	}
	if count := len(conversation.GetMessages()); count > 0 {
		parts = append(parts, fmt.Sprintf("%d messages", count))
	}
	if updated := formatUnix(conversation.GetUpdatedAt()); updated != "<unknown>" {
		parts = append(parts, "updated "+updated)
	}
	if len(parts) == 0 {
		return "No provider or message metadata yet."
	}
	return strings.Join(parts, " | ")
}

func conversationListLabel(conversation *clientpb.AIConversation, width int) string {
	title := conversationTitle(conversation)
	if provider := strings.TrimSpace(conversation.GetProvider()); provider != "" {
		title += " [" + provider + "]"
	}
	return truncateText(title, width)
}

func cloneConversation(conversation *clientpb.AIConversation) *clientpb.AIConversation {
	if conversation == nil {
		return nil
	}
	cloned, ok := proto.Clone(conversation).(*clientpb.AIConversation)
	if !ok {
		return nil
	}
	return cloned
}

func cloneConversationMessage(message *clientpb.AIConversationMessage) *clientpb.AIConversationMessage {
	if message == nil {
		return nil
	}
	cloned, ok := proto.Clone(message).(*clientpb.AIConversationMessage)
	if !ok {
		return nil
	}
	return cloned
}

func mergeConversationMetadata(dst *clientpb.AIConversation, src *clientpb.AIConversation) {
	if dst == nil || src == nil {
		return
	}

	if strings.TrimSpace(dst.GetID()) == "" {
		dst.ID = src.GetID()
	}
	if dst.GetCreatedAt() == 0 && src.GetCreatedAt() != 0 {
		dst.CreatedAt = src.GetCreatedAt()
	}
	if src.GetUpdatedAt() > dst.GetUpdatedAt() {
		dst.UpdatedAt = src.GetUpdatedAt()
	}
	if operator := strings.TrimSpace(src.GetOperatorName()); operator != "" {
		dst.OperatorName = operator
	}
	if provider := strings.TrimSpace(src.GetProvider()); provider != "" {
		dst.Provider = provider
	}
	if model := strings.TrimSpace(src.GetModel()); model != "" {
		dst.Model = model
	}
	if title := strings.TrimSpace(src.GetTitle()); title != "" {
		dst.Title = title
	}
	if summary := strings.TrimSpace(src.GetSummary()); summary != "" {
		dst.Summary = summary
	}
	if systemPrompt := strings.TrimSpace(src.GetSystemPrompt()); systemPrompt != "" {
		dst.SystemPrompt = systemPrompt
	}
}

func conversationAwaitingResponse(conversation *clientpb.AIConversation) bool {
	message := lastConversationMessage(conversation)
	if message == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(message.GetRole()), "user")
}

func lastConversationMessage(conversation *clientpb.AIConversation) *clientpb.AIConversationMessage {
	if conversation == nil {
		return nil
	}
	messages := conversation.GetMessages()
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i] != nil {
			return messages[i]
		}
	}
	return nil
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func providerDisplay(provider *clientpb.AIProviderConfig) string {
	if provider == nil {
		return "<unknown provider>"
	}
	status := "not configured"
	if provider.GetConfigured() {
		status = "configured"
	}
	return fmt.Sprintf("%s: %s", fallback(provider.GetName(), "<unnamed>"), status)
}

func safeAIConfigSummary(resp *clientpb.AIProviderConfigs) *clientpb.AIConfigSummary {
	if resp == nil || resp.GetConfig() == nil {
		return &clientpb.AIConfigSummary{
			Error: "server did not return AI configuration status; update the server and try again",
		}
	}
	return resp.GetConfig()
}

func aiConfigError(config *clientpb.AIConfigSummary) string {
	if config == nil {
		return "server AI configuration is unavailable"
	}
	if err := strings.TrimSpace(config.GetError()); err != "" {
		return err
	}
	return "server AI configuration is invalid"
}

func conversationMessageMarkdown(conversation *clientpb.AIConversation, message *clientpb.AIConversationMessage, meta []string) string {
	label := escapeMarkdownText(messageBlockLabel(conversation, message))
	content := ""
	if message != nil {
		content = strings.Trim(message.GetContent(), "\n")
	}
	if strings.TrimSpace(content) == "" {
		content = "_Empty message._"
	}

	var markdown strings.Builder
	markdown.WriteString("### ")
	markdown.WriteString(label)
	markdown.WriteString("\n\n")
	if len(meta) > 0 {
		markdown.WriteString("> ")
		markdown.WriteString(escapeMarkdownText(strings.Join(meta, " | ")))
		markdown.WriteString("\n\n")
	}
	markdown.WriteString(content)
	return markdown.String()
}

func messageBlockLabel(conversation *clientpb.AIConversation, message *clientpb.AIConversationMessage) string {
	role := ""
	if message != nil {
		role = strings.ToLower(strings.TrimSpace(message.GetRole()))
	}

	switch role {
	case "user":
		if message != nil {
			if operatorName := strings.TrimSpace(message.GetOperatorName()); operatorName != "" {
				return operatorName
			}
		}
		if conversation != nil {
			if operatorName := strings.TrimSpace(conversation.GetOperatorName()); operatorName != "" {
				return operatorName
			}
		}
		return "User"
	case "assistant":
		return "AI"
	case "system":
		return "System"
	default:
		if role == "" {
			return "Message"
		}
		return strings.ToUpper(role[:1]) + role[1:]
	}
}

func escapeMarkdownText(text string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"`", "\\`",
		"*", "\\*",
		"_", "\\_",
		"[", "\\[",
		"]", "\\]",
		"#", "\\#",
		"|", "\\|",
		">", "\\>",
	)
	return replacer.Replace(text)
}

func promptConversationTitle(prompt string) string {
	for _, line := range strings.Split(prompt, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		return truncateText(line, 48)
	}
	return "New conversation"
}

func shortenID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	parts := strings.SplitN(id, "-", 2)
	return parts[0]
}

func inputWindow(input []rune, cursor, width int) ([]rune, int) {
	if width <= 0 {
		return nil, 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(input) {
		cursor = len(input)
	}

	start := 0
	if len(input) > width {
		start = cursor - width + 1
		if start < 0 {
			start = 0
		}
		if start > len(input)-width {
			start = len(input) - width
		}
	}
	end := minInt(len(input), start+width)
	return input[start:end], cursor - start
}

func fitStyledPieces(width int, pieces []string) string {
	if width <= 0 {
		return ""
	}

	var kept []string
	for _, piece := range pieces {
		candidate := strings.Join(append(kept, piece), " ")
		if lipgloss.Width(candidate) > width {
			break
		}
		kept = append(kept, piece)
	}
	return strings.Join(kept, " ")
}

func wrapText(text string, width int) []string {
	if width <= 1 {
		return []string{truncateText(text, width)}
	}

	paragraphs := strings.Split(text, "\n")
	lines := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}

		current := words[0]
		for _, word := range words[1:] {
			if utf8.RuneCountInString(current)+1+utf8.RuneCountInString(word) <= width {
				current += " " + word
				continue
			}
			lines = append(lines, current)
			current = word
		}
		lines = append(lines, current)
	}
	return lines
}

func truncateText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if utf8.RuneCountInString(text) <= width {
		return text
	}
	if width <= 3 {
		return string([]rune(text)[:width])
	}
	runes := []rune(text)
	return string(runes[:width-3]) + "..."
}

func headLines(lines []string, count int) []string {
	if count <= 0 {
		return nil
	}
	if len(lines) <= count {
		return lines
	}
	return lines[:count]
}

func tailLines(lines []string, count int) []string {
	if count <= 0 {
		return nil
	}
	if len(lines) <= count {
		return lines
	}
	return lines[len(lines)-count:]
}

func innerPaneWidth(width int) int {
	return maxInt(1, width-aiPaneHorizontalChrome)
}

func innerPaneHeight(height int) int {
	return maxInt(1, height-aiPaneVerticalChrome)
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
