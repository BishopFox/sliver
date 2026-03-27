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
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"image/color"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	aithinking "github.com/bishopfox/sliver/client/spin/thinking"
	clienttheme "github.com/bishopfox/sliver/client/theme"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/util/clientpbutil"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
	"google.golang.org/protobuf/proto"
)

const (
	aiMinWidth  = 72
	aiMinHeight = 17

	aiPaneHorizontalChrome     = 4
	aiPaneVerticalChrome       = 2
	aiTranscriptScrollbarWidth = 1
	aiTranscriptMouseWheelStep = 3
	aiModalDismissDelay        = 250 * time.Millisecond
	aiWindowPollInterval       = 100 * time.Millisecond
	aiToastDuration            = 10 * time.Second
)

type aiFocus int
type aiModalKind int
type aiModalFocusTarget int

const (
	aiFocusSidebar aiFocus = iota
	aiFocusTranscript
	aiFocusComposer
)

const (
	aiModalKindInfo aiModalKind = iota
	aiModalKindDeleteConfirm
	aiModalKindContext
	aiModalKindNewConversation
	aiModalKindThinkingLevel
	aiModalKindTargetSelect
	aiModalKindExperimentalWarning
)

const (
	aiExperimentalWarningTitle        = ">>> WARNING >>>"
	aiExperimentalWarningBody         = "WARNING: This functionality is provided on an EXPERIMENTAL basis and may be UNSAFE or unstable. No guarantees are made regarding reliability or data integrity. Use at your own risk."
	aiExperimentalWarningCancelLabel  = "I can't read"
	aiExperimentalWarningConfirmLabel = "I won't be surprised if the model nukes a machine"
)

const (
	aiModalFocusInput aiModalFocusTarget = iota
	aiModalFocusCancel
	aiModalFocusConfirm
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
	optionFocused    lipgloss.Style
	heading          lipgloss.Style
	cursor           lipgloss.Style
	inputText        lipgloss.Style
	inputPlaceholder lipgloss.Style
	roleUser         lipgloss.Style
	warning          lipgloss.Style
	danger           lipgloss.Style
	selection        lipgloss.Style
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

type aiConversationDeletedMsg struct {
	conversationID string
	selectedID     string
	status         string
}

type aiConversationUpdatedMsg struct {
	conversation *clientpb.AIConversation
	status       string
}

type aiConversationTargetUpdatedMsg struct {
	conversation *clientpb.AIConversation
	target       aiTargetSummary
	status       string
}

type aiConversationEventMsg struct {
	event *clientpb.AIConversationEvent
}

type aiListenerClosedMsg struct{}

type aiAsyncErrMsg struct {
	err error
}

type aiToastMsg struct {
	level   string
	message string
}

type aiToastExpiredMsg struct {
	id uint64
}

type aiTranscriptRenderedMsg struct {
	key      string
	rendered string
	lines    []string
	content  []aiTranscriptContentLine
}

type aiWindowPollMsg struct {
	width  int
	height int
}

type aiStartupConfigInvalidMsg struct {
	err string
}

type aiTargetOptionsLoadedMsg struct {
	options        []aiTargetSelectionOption
	selectedOption int
	status         string
}

type aiModalState struct {
	kind           aiModalKind
	title          string
	body           string
	dismissReadyAt time.Time
	focus          aiModalFocusTarget
	input          []rune
	cursor         int
	confirmDelete  bool
	selectedOption int
	conversationID string
	selectedID     string
	status         string
}

type aiThinkingLevelOption struct {
	label       string
	value       string
	description string
}

type aiTargetSelectionOption struct {
	label    string
	metadata []string
	target   aiTargetSummary
	session  *clientpb.Session
	beacon   *clientpb.Beacon
	active   bool
}

type aiContextField struct {
	label string
	value string
	muted bool
}

type aiContextSection struct {
	title  string
	fields []aiContextField
}

type aiPaneRect struct {
	x      int
	y      int
	width  int
	height int
}

func (r aiPaneRect) contains(x, y int) bool {
	return r.width > 0 &&
		r.height > 0 &&
		x >= r.x &&
		x < r.x+r.width &&
		y >= r.y &&
		y < r.y+r.height
}

type aiPaneRects struct {
	sidebar    aiPaneRect
	transcript aiPaneRect
	composer   aiPaneRect
}

type aiToastState struct {
	id        uint64
	level     string
	message   string
	createdAt time.Time
	expiresAt time.Time
	bar       progress.Model
}

type aiTranscriptContentLine struct {
	styled              string
	selectableStart     int
	selectableText      string
	selectablePrefix    string
	selectableAreaWidth int
}

func (l aiTranscriptContentLine) selectableWidth() int {
	return ansi.StringWidth(l.selectableText)
}

func (l aiTranscriptContentLine) hasSelectableText() bool {
	return l.selectableAreaWidth > 0 && l.selectableWidth() > 0
}

type aiTranscriptSelectionPoint struct {
	line int
	col  int
}

type aiTranscriptSelection struct {
	anchor   aiTranscriptSelectionPoint
	active   aiTranscriptSelectionPoint
	dragging bool
}

type aiModel struct {
	width                   int
	height                  int
	focus                   aiFocus
	ctx                     aiContext
	con                     *console.SliverClient
	listener                <-chan *clientpb.Event
	providers               []*clientpb.AIProviderConfig
	config                  *clientpb.AIConfigSummary
	conversations           []*clientpb.AIConversation
	currentConversation     *clientpb.AIConversation
	selectedConversation    int
	input                   []rune
	cursor                  int
	status                  string
	loading                 bool
	awaitingResponse        bool
	submittingPrompt        bool
	pendingPrompt           string
	modal                   *aiModalState
	thinkingAnim            *aithinking.Anim
	submitResults           chan tea.Msg
	styles                  aiStyles
	transcriptVersion       int
	transcriptPendingKey    string
	transcriptCacheKey      string
	transcriptCache         string
	transcriptCacheLines    []string
	transcriptCacheContent  []aiTranscriptContentLine
	transcriptScroll        int
	transcriptFollow        bool
	transcriptSelection     *aiTranscriptSelection
	transcriptScrollbarDrag bool
	targetSelectionOptions  []aiTargetSelectionOption
	toast                   *aiToastState
	nextToastID             uint64
}

func newAIModel(con *console.SliverClient, ctx aiContext, listener <-chan *clientpb.Event) *aiModel {
	return &aiModel{
		focus:            aiFocusSidebar,
		ctx:              ctx,
		con:              con,
		listener:         listener,
		status:           ctx.status,
		loading:          true,
		thinkingAnim:     newAIThinkingAnim("Working"),
		submitResults:    make(chan tea.Msg),
		styles:           newAIStyles(),
		transcriptFollow: true,
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
		optionFocused: lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.DefaultMod(900)).
			Background(clienttheme.PrimaryMod(200)),
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
		danger: lipgloss.NewStyle().
			Foreground(clienttheme.Danger()).
			Bold(true),
		selection: lipgloss.NewStyle().
			Bold(true).
			Underline(true).
			Foreground(clienttheme.DefaultMod(900)).
			Background(clienttheme.WarningMod(200)),
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

	cmds := []tea.Cmd{aiWindowPollCmd()}
	if m.modal == nil || m.modal.kind != aiModalKindExperimentalWarning {
		cmds = append(cmds, m.startAIStartup())
	}
	return tea.Batch(cmds...)
}

func (m *aiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.modal != nil {
		return m.handleModalMsg(msg)
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
			kind:           aiModalKindInfo,
			title:          "AI Configuration Error",
			body:           msg.err,
			dismissReadyAt: time.Now().Add(aiModalDismissDelay),
		}
		return m, nil

	case aiLoadedMsg:
		previousConversationID := selectedConversationID(m.currentConversation, "")
		m.loading = false
		m.providers = msg.providers
		m.config = msg.config
		m.conversations = msg.conversations
		m.currentConversation = msg.conversation
		if selectedConversationID(msg.conversation, msg.selectedID) != previousConversationID {
			m.pinTranscriptToLatest()
		}
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
			m.status = "No AI conversations yet. Create one or send a prompt below."
		} else if m.status == "" ||
			strings.HasPrefix(m.status, "Loading") ||
			strings.HasPrefix(m.status, "Refreshing") ||
			strings.HasPrefix(m.status, "Saved prompt to") ||
			strings.HasPrefix(m.status, "Waiting for AI response") ||
			strings.HasPrefix(m.status, "Deleted ") {
			m.status = "Loaded AI conversations from the server."
		}
		awaitingCmd := m.syncAwaitingResponse()
		m.syncTranscriptViewport()
		return m, tea.Batch(
			awaitingCmd,
			m.scheduleTranscriptRender(),
		)

	case aiConversationCreatedMsg:
		m.loading = true
		m.status = msg.status
		return m, loadAIStateCmd(m.con, m.ctx.target, msg.conversationID)

	case aiConversationUpdatedMsg:
		m.loading = false
		if msg.conversation != nil {
			m.applyConversationSnapshot(msg.conversation)
		}
		if strings.TrimSpace(msg.status) != "" {
			m.status = msg.status
		} else {
			m.status = "Conversation updated."
		}
		m.invalidateTranscriptCache()
		awaitingCmd := m.syncAwaitingResponse()
		m.syncTranscriptViewport()
		return m, tea.Batch(
			awaitingCmd,
			m.scheduleTranscriptRender(),
		)

	case aiConversationTargetUpdatedMsg:
		m.loading = false
		if msg.conversation != nil {
			sessionID, beaconID := normalizedAITargetIDs(msg.target.SessionID, msg.target.BeaconID)
			msg.conversation.TargetSessionID = sessionID
			msg.conversation.TargetBeaconID = beaconID
			m.applyConversationSnapshot(msg.conversation)
			if m.currentConversation != nil && strings.TrimSpace(m.currentConversation.GetID()) == strings.TrimSpace(msg.conversation.GetID()) {
				m.currentConversation.TargetSessionID = sessionID
				m.currentConversation.TargetBeaconID = beaconID
			}
			if idx := conversationIndexByID(m.conversations, msg.conversation.GetID()); idx >= 0 && m.conversations[idx] != nil {
				m.conversations[idx].TargetSessionID = sessionID
				m.conversations[idx].TargetBeaconID = beaconID
			}
		}
		if strings.TrimSpace(msg.status) != "" {
			m.status = msg.status
		} else {
			m.status = "Active target updated."
		}
		return m, nil

	case aiPromptSubmittedMsg:
		m.loading = false
		m.submittingPrompt = false
		m.pendingPrompt = ""
		m.status = msg.status
		m.pinTranscriptToLatest()
		m.applyOptimisticPrompt(msg.conversation, msg.message)
		awaitingCmd := m.syncAwaitingResponse()
		m.syncTranscriptViewport()
		return m, tea.Batch(
			awaitingCmd,
			m.scheduleTranscriptRender(),
		)

	case aiConversationDeletedMsg:
		m.loading = true
		m.removeConversation(msg.conversationID)
		if m.currentConversation != nil && strings.TrimSpace(m.currentConversation.GetID()) == strings.TrimSpace(msg.conversationID) {
			m.currentConversation = nil
			m.awaitingResponse = false
			m.submittingPrompt = false
			m.pendingPrompt = ""
		}
		m.pinTranscriptToLatest()
		m.invalidateTranscriptCache()
		m.syncTranscriptViewport()
		m.status = msg.status
		return m, loadAIStateWithStatusCmd(m.con, m.ctx.target, msg.selectedID, msg.status)

	case aiConversationEventMsg:
		m.applyConversationEvent(msg.event)
		m.syncTranscriptViewport()
		return m, tea.Batch(
			waitForAIConversationEventCmd(m.listener),
			m.syncAwaitingResponse(),
			m.scheduleTranscriptRender(),
		)

	case aiToastMsg:
		return m, tea.Batch(
			waitForAIConversationEventCmd(m.listener),
			m.showToast(msg.level, msg.message),
		)

	case aiToastExpiredMsg:
		if m.toast != nil && m.toast.id == msg.id {
			m.toast = nil
		}
		return m, nil

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
		m.clearTranscriptSelection()
		if msg.err != nil {
			m.status = "AI sync failed: " + msg.err.Error()
		}
		m.syncTranscriptViewport()
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
		m.transcriptCacheContent = append([]aiTranscriptContentLine(nil), msg.content...)
		m.syncTranscriptViewport()
		return m, nil

	case tea.PasteMsg:
		if m.focus == aiFocusComposer {
			return m.handleComposerPaste(msg)
		}
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

	case tea.MouseClickMsg:
		return m.handleMouseClick(tea.Mouse(msg))

	case tea.MouseReleaseMsg:
		return m.handleMouseRelease(tea.Mouse(msg))

	case tea.MouseMotionMsg:
		return m.handleMouseMotion(tea.Mouse(msg))

	case tea.MouseWheelMsg:
		return m.handleMouseWheel(tea.Mouse(msg))
	}

	return m, nil
}

func (m *aiModel) handleGlobalKey(key tea.Key) (tea.Model, tea.Cmd) {
	if key.Mod.Contains(tea.ModCtrl) && key.Code == 's' {
		return m.showTargetSelectModal()
	}

	switch key.Code {
	case tea.KeyTab:
		m.focus = (m.focus + 1) % (aiFocusComposer + 1)
		return m, nil

	case tea.KeyEsc:
		return m, tea.Quit

	case tea.KeyUp:
		if m.focus == aiFocusSidebar {
			return m, m.moveSelection(-1)
		}
		if m.focus == aiFocusTranscript {
			m.scrollTranscript(-1)
		}
		return m, nil

	case tea.KeyDown:
		if m.focus == aiFocusSidebar {
			return m, m.moveSelection(1)
		}
		if m.focus == aiFocusTranscript {
			m.scrollTranscript(1)
		}
		return m, nil

	case tea.KeyPgUp:
		if m.focus == aiFocusTranscript {
			m.scrollTranscriptPage(-1)
		}
		return m, nil

	case tea.KeyPgDown:
		if m.focus == aiFocusTranscript {
			m.scrollTranscriptPage(1)
		}
		return m, nil

	case tea.KeyHome:
		if m.focus == aiFocusTranscript {
			m.scrollTranscriptToTop()
		}
		return m, nil

	case tea.KeyEnd:
		if m.focus == aiFocusTranscript {
			m.scrollTranscriptToBottom()
		}
		return m, nil

	case tea.KeyEnter:
		if m.focus == aiFocusSidebar {
			m.focus = aiFocusTranscript
			m.status = "Conversation opened."
		}
		return m, nil

	case tea.KeyDelete:
		return m.showDeleteConversationModal()
	}

	switch key.Text {
	case "q":
		return m, tea.Quit

	case "j":
		if m.focus == aiFocusSidebar {
			return m, m.moveSelection(1)
		}
		if m.focus == aiFocusTranscript {
			m.scrollTranscript(1)
		}

	case "k":
		if m.focus == aiFocusSidebar {
			return m, m.moveSelection(-1)
		}
		if m.focus == aiFocusTranscript {
			m.scrollTranscript(-1)
		}

	case "g":
		if m.focus == aiFocusTranscript {
			m.scrollTranscriptToTop()
		}

	case "G":
		if m.focus == aiFocusTranscript {
			m.scrollTranscriptToBottom()
		}

	case "n":
		return m.showNewConversationModal()

	case "r":
		m.loading = true
		m.status = "Refreshing AI conversations..."
		return m, loadAIStateCmd(m.con, m.ctx.target, m.selectedConversationID())

	case "x":
		return m.showDeleteConversationModal()

	case "t":
		return m.showThinkingLevelModal()
	}

	return m, nil
}

func (m *aiModel) handleMouseClick(mouse tea.Mouse) (tea.Model, tea.Cmd) {
	if mouse.Button != tea.MouseLeft {
		return m, nil
	}

	focus, ok := m.paneFocusAt(mouse.X, mouse.Y)
	if !ok {
		m.transcriptScrollbarDrag = false
		return m, nil
	}

	m.focus = focus
	switch focus {
	case aiFocusSidebar:
		m.transcriptScrollbarDrag = false
		m.clearTranscriptSelection()
		if idx, ok := m.sidebarConversationIndexAt(mouse.X, mouse.Y); ok {
			return m, m.selectConversation(idx)
		}
		return m, nil

	case aiFocusTranscript:
		if row, ok := m.transcriptScrollbarRowAt(mouse.X, mouse.Y, false); ok {
			m.transcriptScrollbarDrag = true
			m.clearTranscriptSelection()
			m.scrollTranscriptToScrollbarRow(row)
			m.status = "Conversation focused. Drag the scrollbar, drag across message text to select it, or use the wheel to scroll."
			return m, nil
		}
		m.transcriptScrollbarDrag = false
		m.clearTranscriptSelection()
		m.beginTranscriptSelection(mouse.X, mouse.Y)
		if m.transcriptSelection != nil {
			m.status = "Selecting message text. Release to copy the selection."
			return m, nil
		}
		m.status = "Conversation focused. Drag across message text to select and copy it, drag the scrollbar, or use the wheel to scroll."
		return m, nil

	default:
		m.transcriptScrollbarDrag = false
		m.clearTranscriptSelection()
		return m, nil
	}
}

func (m *aiModel) handleMouseRelease(mouse tea.Mouse) (tea.Model, tea.Cmd) {
	if mouse.Button != tea.MouseLeft {
		return m, nil
	}
	if m.transcriptScrollbarDrag {
		m.transcriptScrollbarDrag = false
		return m, nil
	}
	if m.transcriptSelection == nil || !m.transcriptSelection.dragging {
		return m, nil
	}

	m.updateTranscriptSelection(mouse.X, mouse.Y, true)
	m.transcriptSelection.dragging = false

	if m.isCollapsedTranscriptSelection() {
		m.clearTranscriptSelection()
		return m, nil
	}

	selected := m.selectedTranscriptText()
	if selected == "" {
		m.clearTranscriptSelection()
		return m, nil
	}

	copyTranscriptSelectionToClipboard(selected)
	m.status = fmt.Sprintf("Copied %d characters from the selected transcript text.", utf8.RuneCountInString(selected))
	return m, tea.SetClipboard(selected)
}

func (m *aiModel) handleMouseMotion(mouse tea.Mouse) (tea.Model, tea.Cmd) {
	if m.transcriptScrollbarDrag {
		if mouse.Button != tea.MouseLeft {
			return m, nil
		}

		if row, ok := m.transcriptScrollbarRowAt(mouse.X, mouse.Y, true); ok {
			m.focus = aiFocusTranscript
			m.scrollTranscriptToScrollbarRow(row)
		}
		return m, nil
	}
	if m.transcriptSelection == nil || !m.transcriptSelection.dragging {
		return m, nil
	}
	if mouse.Button != tea.MouseLeft {
		return m, nil
	}

	m.focus = aiFocusTranscript
	m.updateTranscriptSelection(mouse.X, mouse.Y, true)
	return m, nil
}

func (m *aiModel) handleMouseWheel(mouse tea.Mouse) (tea.Model, tea.Cmd) {
	if !m.currentPaneRects().transcript.contains(mouse.X, mouse.Y) {
		return m, nil
	}

	switch mouse.Button {
	case tea.MouseWheelUp:
		m.focus = aiFocusTranscript
		m.scrollTranscript(-aiTranscriptMouseWheelStep)
	case tea.MouseWheelDown:
		m.focus = aiFocusTranscript
		m.scrollTranscript(aiTranscriptMouseWheelStep)
	}

	return m, nil
}

func (m *aiModel) handleComposerKey(key tea.Key) (tea.Model, tea.Cmd) {
	if key.Code == tea.KeyTab {
		m.focus = aiFocusSidebar
		return m, nil
	}
	if key.Mod.Contains(tea.ModCtrl) && key.Code == 'o' {
		return m.showContextModal()
	}
	if key.Mod.Contains(tea.ModCtrl) && key.Code == 's' {
		return m.showTargetSelectModal()
	}
	if key.Mod.Contains(tea.ModCtrl) && key.Code == 't' {
		return m.showThinkingLevelModal()
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
		if cmd, handled := m.handleComposerSlashCommand(prompt); handled {
			return m, cmd
		}
		if m.isBusy() {
			m.status = "Wait for the current AI request to finish before sending another prompt."
			return m, nil
		}

		m.clearTranscriptSelection()
		m.submittingPrompt = true
		m.pendingPrompt = prompt
		m.pinTranscriptToLatest()
		m.status = "Saving prompt to the server..."
		m.input = nil
		m.cursor = 0
		submitResults := m.submitResults
		go func() {
			submitResults <- submitPromptMsg(m.con, m.ctx.target, m.currentConversation, m.defaultProvider(), m.defaultModel(), prompt)
		}()
		awaitingCmd := m.startAwaitingResponse()
		m.syncTranscriptViewport()
		return m, tea.Batch(
			awaitingCmd,
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
			m.insertComposerRunes([]rune(key.Text))
		}
	}

	return m, nil
}

func (m *aiModel) handleComposerPaste(msg tea.PasteMsg) (tea.Model, tea.Cmd) {
	if looksLikeTerminalResponseFragment(msg.Content) {
		return m, nil
	}
	if msg.Content == "" {
		return m, nil
	}
	m.insertComposerRunes([]rune(msg.Content))
	return m, nil
}

func (m *aiModel) insertComposerRunes(insert []rune) {
	if len(insert) == 0 {
		return
	}
	m.input = append(m.input[:m.cursor], append(insert, m.input[m.cursor:]...)...)
	m.cursor += len(insert)
}

func (m *aiModel) handleComposerSlashCommand(input string) (tea.Cmd, bool) {
	command, _, ok := parseComposerSlashCommand(input)
	if !ok {
		return nil, false
	}

	switch command {
	case "/exit":
		m.input = nil
		m.cursor = 0
		return tea.Quit, true
	default:
		m.status = fmt.Sprintf("Unknown composer command %q. Available: /exit.", command)
		return nil, true
	}
}

func parseComposerSlashCommand(input string) (string, []string, bool) {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		return "", nil, false
	}
	return strings.ToLower(fields[0]), fields[1:], true
}

func (m *aiModel) handleModalMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, m.applyWindowSize(msg.Width, msg.Height)
	case aiWindowPollMsg:
		return m, tea.Batch(m.windowSizeCmd(msg.width, msg.height), aiWindowPollCmd())
	case aiConversationEventMsg:
		m.applyConversationEvent(msg.event)
		m.syncTranscriptViewport()
		return m, tea.Batch(
			waitForAIConversationEventCmd(m.listener),
			m.syncAwaitingResponse(),
			m.scheduleTranscriptRender(),
		)
	case aiToastMsg:
		return m, tea.Batch(
			waitForAIConversationEventCmd(m.listener),
			m.showToast(msg.level, msg.message),
		)
	case aiToastExpiredMsg:
		if m.toast != nil && m.toast.id == msg.id {
			m.toast = nil
		}
		return m, nil
	case aiListenerClosedMsg:
		m.status = "AI event stream closed. Reopen the AI TUI to resume live updates."
		return m, nil
	case aiTargetOptionsLoadedMsg:
		if m.modal == nil || m.modal.kind != aiModalKindTargetSelect {
			return m, nil
		}
		m.targetSelectionOptions = append([]aiTargetSelectionOption(nil), msg.options...)
		if len(m.targetSelectionOptions) > 0 {
			m.modal.selectedOption = clampInt(msg.selectedOption, 0, len(m.targetSelectionOptions)-1)
		} else {
			m.modal.selectedOption = 0
		}
		m.modal.status = strings.TrimSpace(msg.status)
		return m, nil
	case tea.KeyPressMsg:
		if msg.Key().Mod.Contains(tea.ModCtrl) && msg.Key().Code == 'c' {
			return m, tea.Quit
		}
		switch m.modal.kind {
		case aiModalKindDeleteConfirm:
			return m.handleDeleteConfirmModalKey(msg.Key())
		case aiModalKindContext:
			return m.handleContextModalKey(msg.Key())
		case aiModalKindNewConversation:
			return m.handleNewConversationModalKey(msg.Key())
		case aiModalKindThinkingLevel:
			return m.handleThinkingLevelModalKey(msg.Key())
		case aiModalKindTargetSelect:
			return m.handleTargetSelectModalKey(msg.Key())
		case aiModalKindExperimentalWarning:
			return m.handleExperimentalWarningModalKey(msg.Key())
		default:
			return m.handleInfoModalKey()
		}
	default:
		return m, nil
	}
}

func (m *aiModel) handleInfoModalKey() (tea.Model, tea.Cmd) {
	if time.Now().Before(m.modal.dismissReadyAt) {
		return m, nil
	}
	return m, tea.Quit
}

func (m *aiModel) handleDeleteConfirmModalKey(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyEsc:
		m.modal = nil
		return m, nil
	case tea.KeyLeft:
		m.modal.confirmDelete = false
		return m, nil
	case tea.KeyRight:
		m.modal.confirmDelete = true
		return m, nil
	case tea.KeyTab:
		m.modal.confirmDelete = !m.modal.confirmDelete
		return m, nil
	case tea.KeyEnter:
		if !m.modal.confirmDelete {
			m.modal = nil
			return m, nil
		}
		return m.confirmDeleteConversation()
	}

	switch key.Text {
	case "h":
		m.modal.confirmDelete = false
		return m, nil
	case "l":
		m.modal.confirmDelete = true
		return m, nil
	case "n", "q":
		m.modal = nil
		return m, nil
	case "y":
		m.modal.confirmDelete = true
		return m.confirmDeleteConversation()
	}

	return m, nil
}

func (m *aiModel) handleContextModalKey(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyEsc, tea.KeyEnter, tea.KeyTab:
		m.modal = nil
		return m, nil
	}

	switch key.Text {
	case "q", "c":
		m.modal = nil
		return m, nil
	}

	return m, nil
}

func (m *aiModel) handleThinkingLevelModalKey(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyEsc:
		m.modal = nil
		m.status = "Canceled thinking level update."
		return m, nil
	case tea.KeyUp:
		m.moveThinkingLevelSelection(-1)
		return m, nil
	case tea.KeyDown:
		m.moveThinkingLevelSelection(1)
		return m, nil
	case tea.KeyHome:
		m.modal.selectedOption = 0
		return m, nil
	case tea.KeyEnd:
		options := m.thinkingLevelModalOptions()
		if len(options) > 0 {
			m.modal.selectedOption = len(options) - 1
		}
		return m, nil
	case tea.KeyEnter:
		return m.confirmThinkingLevelUpdate()
	}

	switch key.Text {
	case "q":
		m.modal = nil
		m.status = "Canceled thinking level update."
		return m, nil
	case "j":
		m.moveThinkingLevelSelection(1)
		return m, nil
	case "k":
		m.moveThinkingLevelSelection(-1)
		return m, nil
	}

	return m, nil
}

func (m *aiModel) handleTargetSelectModalKey(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyEsc:
		m.modal = nil
		m.targetSelectionOptions = nil
		m.status = "Canceled active target update."
		return m, nil
	case tea.KeyUp:
		m.moveTargetSelection(-1)
		return m, nil
	case tea.KeyDown:
		m.moveTargetSelection(1)
		return m, nil
	case tea.KeyHome:
		m.modal.selectedOption = 0
		return m, nil
	case tea.KeyEnd:
		if len(m.targetSelectionOptions) > 0 {
			m.modal.selectedOption = len(m.targetSelectionOptions) - 1
		}
		return m, nil
	case tea.KeyEnter:
		return m.confirmTargetSelectionUpdate()
	}

	switch key.Text {
	case "q":
		m.modal = nil
		m.targetSelectionOptions = nil
		m.status = "Canceled active target update."
		return m, nil
	case "j":
		m.moveTargetSelection(1)
		return m, nil
	case "k":
		m.moveTargetSelection(-1)
		return m, nil
	}

	return m, nil
}

func (m *aiModel) handleExperimentalWarningModalKey(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyLeft:
		m.modal.focus = aiModalFocusCancel
		return m, nil
	case tea.KeyRight:
		m.modal.focus = aiModalFocusConfirm
		return m, nil
	case tea.KeyTab:
		if m.modal.focus == aiModalFocusConfirm {
			m.modal.focus = aiModalFocusCancel
		} else {
			m.modal.focus = aiModalFocusConfirm
		}
		return m, nil
	case tea.KeyEnter:
		if m.modal.focus == aiModalFocusConfirm {
			return m.acceptExperimentalWarning()
		}
		return m, tea.Quit
	}

	switch key.Text {
	case "h":
		m.modal.focus = aiModalFocusCancel
		return m, nil
	case "l":
		m.modal.focus = aiModalFocusConfirm
		return m, nil
	case "q", "n":
		return m, tea.Quit
	case "y":
		m.modal.focus = aiModalFocusConfirm
		return m.acceptExperimentalWarning()
	}

	return m, nil
}

func (m *aiModel) handleNewConversationModalKey(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyEsc:
		return m.cancelNewConversationModal()
	case tea.KeyTab:
		switch m.modal.focus {
		case aiModalFocusInput:
			m.modal.focus = aiModalFocusCancel
		case aiModalFocusCancel:
			m.modal.focus = aiModalFocusConfirm
		default:
			m.modal.focus = aiModalFocusInput
		}
		return m, nil
	case tea.KeyEnter:
		if m.modal.focus == aiModalFocusCancel {
			return m.cancelNewConversationModal()
		}
		return m.confirmCreateConversation()
	case tea.KeyLeft:
		if m.modal.focus == aiModalFocusInput {
			if m.modal.cursor > 0 {
				m.modal.cursor--
			}
			return m, nil
		}
		m.modal.focus = aiModalFocusCancel
		return m, nil
	case tea.KeyRight:
		if m.modal.focus == aiModalFocusInput {
			if m.modal.cursor < len(m.modal.input) {
				m.modal.cursor++
			}
			return m, nil
		}
		m.modal.focus = aiModalFocusConfirm
		return m, nil
	case tea.KeyHome:
		if m.modal.focus == aiModalFocusInput {
			m.modal.cursor = 0
		}
		return m, nil
	case tea.KeyEnd:
		if m.modal.focus == aiModalFocusInput {
			m.modal.cursor = len(m.modal.input)
		}
		return m, nil
	case tea.KeyBackspace:
		if m.modal.focus == aiModalFocusInput && m.modal.cursor > 0 {
			m.modal.input = append(m.modal.input[:m.modal.cursor-1], m.modal.input[m.modal.cursor:]...)
			m.modal.cursor--
		}
		return m, nil
	case tea.KeyDelete:
		if m.modal.focus == aiModalFocusInput && m.modal.cursor < len(m.modal.input) {
			m.modal.input = append(m.modal.input[:m.modal.cursor], m.modal.input[m.modal.cursor+1:]...)
		}
		return m, nil
	}

	if m.modal.focus != aiModalFocusInput {
		switch key.Text {
		case "q":
			return m.cancelNewConversationModal()
		case "h":
			m.modal.focus = aiModalFocusCancel
			return m, nil
		case "l":
			m.modal.focus = aiModalFocusConfirm
			return m, nil
		}
		return m, nil
	}

	if key.Text != "" {
		if looksLikeTerminalResponseFragment(key.Text) {
			return m, nil
		}
		insert := []rune(key.Text)
		m.modal.input = append(m.modal.input[:m.modal.cursor], append(insert, m.modal.input[m.modal.cursor:]...)...)
		m.modal.cursor += len(insert)
	}

	return m, nil
}

func (m *aiModel) cancelNewConversationModal() (tea.Model, tea.Cmd) {
	m.modal = nil
	m.status = "Canceled new conversation creation."
	return m, nil
}

func (m *aiModel) moveThinkingLevelSelection(delta int) {
	if m.modal == nil {
		return
	}
	options := m.thinkingLevelModalOptions()
	if len(options) == 0 {
		m.modal.selectedOption = 0
		return
	}
	m.modal.selectedOption = clampInt(m.modal.selectedOption+delta, 0, len(options)-1)
}

func (m *aiModel) moveTargetSelection(delta int) {
	if m.modal == nil {
		return
	}
	if len(m.targetSelectionOptions) == 0 {
		m.modal.selectedOption = 0
		return
	}
	m.modal.selectedOption = clampInt(m.modal.selectedOption+delta, 0, len(m.targetSelectionOptions)-1)
}

func (m *aiModel) confirmThinkingLevelUpdate() (tea.Model, tea.Cmd) {
	if m.modal == nil || m.currentConversation == nil {
		return m, nil
	}

	options := m.thinkingLevelModalOptions()
	if len(options) == 0 {
		m.modal = nil
		m.status = "No thinking level options are available."
		return m, nil
	}

	selectedOption := clampInt(m.modal.selectedOption, 0, len(options)-1)
	thinkingLevel := options[selectedOption].value
	if normalizeAIThinkingLevel(m.currentConversation.GetThinkingLevel()) == thinkingLevel {
		m.modal = nil
		m.status = "Thinking level unchanged."
		return m, nil
	}

	m.modal = nil
	m.loading = true
	m.status = "Saving thinking level..."
	return m, updateConversationThinkingLevelCmd(m.con, m.currentConversation, thinkingLevel)
}

func (m *aiModel) confirmTargetSelectionUpdate() (tea.Model, tea.Cmd) {
	if m.modal == nil {
		return m, nil
	}
	if len(m.targetSelectionOptions) == 0 {
		status := strings.TrimSpace(m.modal.status)
		if status == "" {
			status = "No sessions or beacons are currently available."
		}
		m.status = status
		return m, nil
	}

	selectedOption := clampInt(m.modal.selectedOption, 0, len(m.targetSelectionOptions)-1)
	option := m.targetSelectionOptions[selectedOption]
	sameActiveTarget := sameAITargetSelectionSummary(m.ctx.target, option.target)
	sameConversationTarget := conversationUsesTarget(m.currentConversation, option.target)

	m.applyTargetSelectionOption(option)
	m.modal = nil
	m.targetSelectionOptions = nil

	if sameActiveTarget && (m.currentConversation == nil || sameConversationTarget) {
		m.status = "Active target unchanged."
		return m, nil
	}

	targetLabel := fallback(option.target.Label, option.label)
	if m.currentConversation == nil || strings.TrimSpace(m.currentConversation.GetID()) == "" || sameConversationTarget {
		m.status = "Active target set to " + targetLabel + "."
		return m, nil
	}

	m.loading = true
	m.status = "Saving active target..."
	return m, updateConversationTargetCmd(m.con, m.currentConversation, option.target)
}

func (m *aiModel) confirmCreateConversation() (tea.Model, tea.Cmd) {
	if m.modal == nil {
		return m, nil
	}

	title := strings.TrimSpace(string(m.modal.input))
	if title == "" {
		m.modal.focus = aiModalFocusInput
		m.status = "Type a conversation name first."
		return m, nil
	}

	m.modal = nil
	m.loading = true
	m.status = "Creating a new AI conversation..."
	return m, createConversationCmd(m.con, m.ctx.target, m.defaultProvider(), m.defaultModel(), title)
}

func (m *aiModel) confirmDeleteConversation() (tea.Model, tea.Cmd) {
	if m.modal == nil {
		return m, nil
	}

	conversationID := strings.TrimSpace(m.modal.conversationID)
	selectedID := strings.TrimSpace(m.modal.selectedID)
	status := strings.TrimSpace(m.modal.status)
	m.modal = nil

	if conversationID == "" {
		m.status = "No AI conversation selected."
		return m, nil
	}
	if status == "" {
		status = "Deleted AI conversation."
	}

	m.loading = true
	m.status = "Deleting AI conversation..."
	return m, deleteConversationCmd(m.con, conversationID, selectedID, status)
}

func (m *aiModel) View() tea.View {
	if m.width < aiMinWidth || m.height < aiMinHeight {
		content := m.renderTooSmall()
		if m.toast != nil {
			content = m.renderToastOverlay(content)
		}
		if m.modal != nil {
			content = m.renderModalOverlay(content)
		}
		view := aiView(content)
		return view
	}

	_, composerHeight, _, bodyHeight := m.layoutHeights()

	header := m.renderHeader()
	body := m.renderBody(bodyHeight)
	composer := m.renderComposer(composerHeight)
	footer := m.renderFooter()

	frame := lipgloss.JoinVertical(lipgloss.Left, header, body, composer, footer)
	frame = lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, frame)
	if m.toast != nil {
		frame = m.renderToastOverlay(frame)
	}
	if m.modal != nil {
		frame = m.renderModalOverlay(frame)
	}
	frame = clampANSIBlock(frame, m.width, m.height)
	view := aiView(frame)
	return view
}

func (m *aiModel) renderModal() string {
	switch m.modal.kind {
	case aiModalKindDeleteConfirm:
		return m.renderDeleteConfirmModal()
	case aiModalKindContext:
		return m.renderContextModal()
	case aiModalKindNewConversation:
		return m.renderNewConversationModal()
	case aiModalKindThinkingLevel:
		return m.renderThinkingLevelModal()
	case aiModalKindTargetSelect:
		return m.renderTargetSelectModal()
	case aiModalKindExperimentalWarning:
		return m.renderExperimentalWarningModal()
	default:
		return m.renderInfoModal()
	}
}

func (m *aiModel) renderInfoModal() string {
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

	return box
}

func (m *aiModel) renderDeleteConfirmModal() string {
	boxWidth := minInt(maxInt(20, m.width-4), 84)
	bodyWidth := maxInt(20, boxWidth-6)
	bodyLines := wrapText(m.modal.body, bodyWidth)

	lines := []string{
		m.styles.danger.Width(bodyWidth).Render(m.modal.title),
		"",
	}
	for _, line := range bodyLines {
		lines = append(lines, lipgloss.NewStyle().Width(bodyWidth).Render(line))
	}
	lines = append(lines, "", m.renderDeleteConfirmActions(bodyWidth))

	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(clienttheme.Danger()).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

	return box
}

func (m *aiModel) renderContextModal() string {
	boxWidth := minInt(maxInt(44, m.width-6), 96)
	bodyWidth := maxInt(24, boxWidth-6)

	lines := []string{
		lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.Primary()).
			Width(bodyWidth).
			Render(m.modal.title),
		m.styles.subtleText.Width(bodyWidth).Render("Active target, connection, provider defaults, and thread metadata."),
		"",
	}

	sectionBlocks := m.renderContextModalSections(bodyWidth)
	for i, block := range sectionBlocks {
		lines = append(lines, strings.Split(block, "\n")...)
		if i < len(sectionBlocks)-1 {
			lines = append(lines, "")
		}
	}

	dismissHint := m.styles.chip.Width(bodyWidth).Render("esc / enter / q closes")
	maxLines := maxInt(6, m.height-6)
	if len(lines)+2 > maxLines {
		lines = headLines(lines, maxInt(1, maxLines-3))
		lines = append(lines, m.styles.subtleText.Width(bodyWidth).Render("Resize the terminal to view more context."))
	}
	lines = append(lines, "", dismissHint)

	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(clienttheme.Primary()).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

	return box
}

func (m *aiModel) renderNewConversationModal() string {
	boxWidth := minInt(maxInt(36, m.width-6), 84)
	bodyWidth := maxInt(24, boxWidth-6)

	lines := []string{
		lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.Primary()).
			Width(bodyWidth).
			Render(m.modal.title),
		m.styles.subtleText.Width(bodyWidth).Render("Name the new conversation before creating it."),
		"",
		m.renderNewConversationInput(bodyWidth),
		"",
		m.renderNewConversationActions(bodyWidth),
		"",
		m.styles.chip.Width(bodyWidth).Render("tab: focus  enter: create  esc: cancel"),
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(clienttheme.Primary()).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

	return box
}

func (m *aiModel) renderThinkingLevelModal() string {
	boxWidth := minInt(maxInt(52, m.width-6), 96)
	bodyWidth := maxInt(28, boxWidth-6)

	lines := []string{
		lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.Primary()).
			Width(bodyWidth).
			Render(m.modal.title),
		m.styles.subtleText.Width(bodyWidth).Render("Choose the reasoning effort hint for future turns in this conversation."),
		m.styles.subtleText.Width(bodyWidth).Render("Drivers may map or ignore this setting depending on backend support."),
		"",
	}

	for _, optionLine := range m.renderThinkingLevelModalOptions(bodyWidth) {
		lines = append(lines, optionLine)
	}

	lines = append(lines,
		"",
		m.styles.chip.Width(bodyWidth).Render("j/k or up/down: select  enter: apply  esc/q: cancel"),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(clienttheme.Primary()).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

	return box
}

func (m *aiModel) renderTargetSelectModal() string {
	boxWidth := minInt(maxInt(60, m.width-6), 112)
	bodyWidth := maxInt(32, boxWidth-6)

	description := "Select the session or beacon to make active in this AI view."
	if m.currentConversation != nil && strings.TrimSpace(m.currentConversation.GetID()) != "" {
		description = "Select the session or beacon used as the active target and current thread target."
	}

	lines := []string{
		lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.Primary()).
			Width(bodyWidth).
			Render(m.modal.title),
		m.styles.subtleText.Width(bodyWidth).Render(description),
		"",
	}

	if len(m.targetSelectionOptions) == 0 {
		status := strings.TrimSpace(m.modal.status)
		if status == "" {
			status = "Loading sessions and beacons from the server..."
		}
		lines = append(lines, m.styles.subtleText.Width(bodyWidth).Render(status))
	} else {
		maxOptionLines := maxInt(6, m.height-16)
		for _, optionLine := range m.renderTargetSelectModalOptions(bodyWidth, maxOptionLines) {
			lines = append(lines, optionLine)
		}
	}

	lines = append(lines,
		"",
		m.styles.chip.Width(bodyWidth).Render("j/k or up/down: select  enter: apply  esc/q: cancel"),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(clienttheme.Primary()).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

	return box
}

func (m *aiModel) renderExperimentalWarningModal() string {
	boxWidth := minInt(maxInt(44, m.width-6), 100)
	bodyWidth := maxInt(24, boxWidth-6)
	bodyLines := wrapText(m.modal.body, bodyWidth)

	lines := []string{
		m.styles.danger.Width(bodyWidth).Render(m.modal.title),
		"",
	}
	for _, line := range bodyLines {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(clienttheme.Danger()).
			Width(bodyWidth).
			Render(line))
	}
	lines = append(lines,
		"",
		m.renderExperimentalWarningActions(bodyWidth),
		"",
		m.styles.subtleText.Width(bodyWidth).Render("tab/h/l: focus  enter: select  esc/q: cancel"),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(clienttheme.Danger()).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

	return box
}

func (m *aiModel) renderThinkingLevelModalOptions(width int) []string {
	options := m.thinkingLevelModalOptions()
	lines := make([]string, 0, len(options)*2)
	for idx, option := range options {
		prefix := "  "
		style := m.styles.item
		if m.modal != nil && idx == m.modal.selectedOption {
			prefix = "> "
			style = m.styles.optionFocused
		}
		lines = append(lines, style.Width(width).Render(truncateText(prefix+option.label, width)))
		if option.description != "" {
			for _, line := range wrapText(option.description, maxInt(1, width-3)) {
				lines = append(lines, m.styles.subtleText.Width(width).Render("   "+line))
			}
		}
	}
	return lines
}

func (m *aiModel) renderTargetSelectModalOptions(width int, maxLines int) []string {
	if len(m.targetSelectionOptions) == 0 || maxLines <= 0 {
		return nil
	}

	blocks := make([][]string, 0, len(m.targetSelectionOptions))
	for idx, option := range m.targetSelectionOptions {
		blocks = append(blocks, m.renderTargetSelectModalOption(width, idx, option))
	}

	start, end := targetSelectionVisibleRange(blocks, maxLines, m.modal.selectedOption)
	lines := make([]string, 0, maxLines)
	for idx := start; idx < end; idx++ {
		lines = append(lines, blocks[idx]...)
	}
	return lines
}

func (m *aiModel) renderTargetSelectModalOption(width int, idx int, option aiTargetSelectionOption) []string {
	prefix := "  "
	style := m.styles.item
	if m.modal != nil && idx == m.modal.selectedOption {
		prefix = "> "
		style = m.styles.optionFocused
	}

	label := option.label
	if option.active {
		label += " [active]"
	}

	lines := []string{
		style.Width(width).Render(truncateText(prefix+label, width)),
	}
	for _, metadata := range option.metadata {
		if strings.TrimSpace(metadata) == "" {
			continue
		}
		for _, line := range wrapText(metadata, maxInt(1, width-3)) {
			lines = append(lines, m.styles.subtleText.Width(width).Render("   "+line))
		}
	}
	return lines
}

func (m *aiModel) renderContextModalSections(width int) []string {
	sections := m.contextModalSections()
	blocks := make([]string, 0, len(sections))
	for _, section := range sections {
		blocks = append(blocks, m.renderContextModalSection(width, section))
	}
	return blocks
}

func (m *aiModel) contextModalSections() []aiContextSection {
	targetFields := []aiContextField{
		{label: "Target", value: fallback(m.ctx.target.Label, "No active target")},
		{label: "Host", value: fallback(m.ctx.target.Host, "<unknown host>")},
		{label: "Platform", value: fallback(m.ctx.target.OS, "unknown") + "/" + fallback(m.ctx.target.Arch, "unknown")},
		{label: "Mode", value: fallback(m.ctx.target.Mode, "<unknown mode>"), muted: true},
		{label: "C2", value: fallback(m.ctx.target.C2, "unknown"), muted: true},
	}
	for _, detail := range m.ctx.target.Details {
		if strings.TrimSpace(detail) == "" {
			continue
		}
		targetFields = append(targetFields, aiContextField{label: "Detail", value: detail, muted: true})
	}

	connectionFields := []aiContextField{
		{label: "Profile", value: fallback(m.ctx.connection.Profile, "<profile unavailable>")},
		{label: "Server", value: fallback(m.ctx.connection.Server, "<unknown>")},
		{label: "Operator", value: fallback(m.ctx.connection.Operator, "<unknown>"), muted: true},
		{label: "State", value: fallback(m.ctx.connection.State, "<unknown>"), muted: true},
	}

	providerFields := []aiContextField{}
	if len(m.providers) == 0 {
		providerFields = append(providerFields, aiContextField{
			label: "Available",
			value: "No AI providers reported by the server.",
			muted: true,
		})
	} else {
		for _, provider := range m.providers {
			if provider == nil {
				continue
			}
			status := "not configured"
			if provider.GetConfigured() {
				status = "configured"
			}
			providerFields = append(providerFields, aiContextField{
				label: fallback(provider.GetName(), "<unnamed>"),
				value: status,
				muted: true,
			})
		}
	}

	defaultFields := []aiContextField{}
	if m.config == nil {
		defaultFields = append(defaultFields, aiContextField{
			label: "Status",
			value: "AI defaults unavailable.",
			muted: true,
		})
	} else {
		defaultFields = append(defaultFields,
			aiContextField{label: "Provider", value: fallback(m.config.GetProvider(), "<unset>")},
			aiContextField{label: "Model", value: fallback(m.config.GetModel(), "provider default")},
			aiContextField{label: "Thinking", value: defaultThinkingLevelSummary(m.config), muted: true},
		)
	}

	threadFields := []aiContextField{}
	if m.currentConversation == nil {
		threadFields = append(threadFields, aiContextField{
			label: "Status",
			value: "No conversation selected yet.",
			muted: true,
		})
	} else {
		threadFields = append(threadFields,
			aiContextField{label: "Title", value: conversationTitle(m.currentConversation)},
			aiContextField{label: "ID", value: shortenID(m.currentConversation.GetID()), muted: true},
			aiContextField{label: "Provider", value: fallback(m.currentConversation.GetProvider(), "<unset>")},
			aiContextField{label: "Model", value: fallback(m.currentConversation.GetModel(), "<default>"), muted: true},
			aiContextField{label: "Thinking", value: conversationThinkingLevelSummary(m.currentConversation, m.config), muted: true},
			aiContextField{label: "Messages", value: fmt.Sprintf("%d", len(m.currentConversation.GetMessages()))},
			aiContextField{label: "Updated", value: formatUnix(m.currentConversation.GetUpdatedAt()), muted: true},
		)
	}

	return []aiContextSection{
		{title: "Target", fields: targetFields},
		{title: "Connection", fields: connectionFields},
		{title: "Thread", fields: threadFields},
		{title: "Defaults", fields: defaultFields},
		{title: "Providers", fields: providerFields},
	}
}

func (m *aiModel) renderContextModalSection(width int, section aiContextSection) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(clienttheme.PrimaryMod(200)).
		Background(clienttheme.PrimaryMod(900)).
		Padding(0, 1).
		Render(section.title)

	lines := []string{title}
	for _, field := range section.fields {
		lines = append(lines, m.renderContextModalFieldLines(width, field)...)
	}

	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

func (m *aiModel) renderContextModalFieldLines(width int, field aiContextField) []string {
	labelWidth := clampInt(width/6, 8, 12)
	valueWidth := maxInt(1, width-(labelWidth+4))
	value := strings.TrimSpace(field.value)
	if value == "" {
		value = "<unset>"
	}

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(clienttheme.DefaultMod(700))
	valueStyle := lipgloss.NewStyle().
		Foreground(clienttheme.DefaultMod(900))
	if field.muted {
		valueStyle = valueStyle.Foreground(clienttheme.DefaultMod(600))
	}

	label := truncateText(field.label, labelWidth)
	wrapped := wrapText(value, valueWidth)
	lines := make([]string, 0, len(wrapped))
	for i, line := range wrapped {
		labelText := strings.Repeat(" ", labelWidth)
		if i == 0 {
			labelText = labelStyle.Width(labelWidth).Render(label)
		}
		lines = append(lines, lipgloss.NewStyle().Width(width).Render(
			lipgloss.NewStyle().Foreground(clienttheme.PrimaryMod(500)).Render("|")+
				" "+
				labelText+
				" "+
				valueStyle.Render(line),
		))
	}
	return lines
}

func (m *aiModel) renderModalOverlay(base string) string {
	box := m.renderModal()
	boxWidth := lipgloss.Width(box)
	boxHeight := lipgloss.Height(box)
	left := maxInt(0, (m.width-boxWidth)/2)
	top := maxInt(0, (m.height-boxHeight)/2)
	return overlayContent(base, box, left, top, m.width)
}

func (m *aiModel) renderToastOverlay(base string) string {
	box := m.renderToast()
	if box == "" {
		return base
	}
	boxWidth := lipgloss.Width(box)
	left := maxInt(0, m.width-boxWidth-2)
	return overlayContent(base, box, left, 1, m.width)
}

func (m *aiModel) renderToast() string {
	if m.toast == nil || strings.TrimSpace(m.toast.message) == "" {
		return ""
	}

	boxWidth := minInt(maxInt(28, m.width/3), 72)
	bodyWidth := maxInt(20, boxWidth-6)
	backgroundColor := clienttheme.DefaultMod(25)

	title := "Event"
	accentColor := clienttheme.Primary()
	switch strings.ToLower(strings.TrimSpace(m.toast.level)) {
	case "error", "warn", "warning":
		title = "Warning"
		accentColor = clienttheme.Danger()
	case "success":
		title = "Notice"
		accentColor = clienttheme.Success()
	}

	lineStyle := lipgloss.NewStyle().
		Width(bodyWidth).
		Background(backgroundColor).
		Foreground(clienttheme.DefaultMod(900))
	titleStyle := lineStyle.Copy().
		Bold(true).
		Foreground(accentColor)
	timeStyle := lineStyle.Copy().
		Foreground(clienttheme.DefaultMod(600)).
		Align(lipgloss.Right)

	now := time.Now()
	remaining := m.toast.remaining(now)
	percentLeft := m.toast.fractionRemaining(now)
	m.toast.bar.SetWidth(bodyWidth)
	barLine := lineStyle.Render(m.toast.bar.ViewAs(percentLeft))

	lines := []string{titleStyle.Render(truncateText(title, bodyWidth))}
	for _, line := range wrapText(strings.TrimSpace(m.toast.message), bodyWidth) {
		lines = append(lines, lineStyle.Render(line))
	}
	lines = append(lines,
		barLine,
		timeStyle.Render(formatAIToastTimeLeft(remaining)),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Background(backgroundColor).
		Padding(0, 1).
		Render(strings.Join(lines, "\n"))
}

func (m *aiModel) renderTooSmall() string {
	lines := []string{
		m.styles.badge.Render("SLIVER AI"),
		"",
		m.styles.warning.Render("Terminal too small for the AI conversation view."),
		m.styles.subtleText.Render("Resize to at least 72x17 to view the sidebar, markdown transcript, and composer."),
	}
	return strings.Join(lines, "\n")
}

func (m *aiModel) renderHeader() string {
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
	}
	return lipgloss.NewStyle().Width(m.width).Render(fitStyledPieces(m.width, pieces))
}

func (m *aiModel) renderBody(height int) string {
	switch {
	case m.width >= 78:
		sidebarWidth := clampInt(m.width/4, 24, 28)
		transcriptWidth := maxInt(40, m.width-sidebarWidth)
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderSidebar(sidebarWidth, height),
			m.renderTranscript(transcriptWidth, height),
		)

	default:
		sidebarHeight := clampInt(height/4, 5, 7)
		transcriptHeight := maxInt(6, height-sidebarHeight)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderSidebar(m.width, sidebarHeight),
			m.renderTranscript(m.width, transcriptHeight),
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
			m.styles.subtleText.Width(innerWidth).Render(truncateText("Create one or send a prompt below.", innerWidth)),
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
				style = m.styles.optionFocused
			}
			lines = append(lines, style.Width(innerWidth).Render(line))
		} else {
			lines = append(lines, m.styles.item.Width(innerWidth).Render("  "+line))
		}
		if len(lines)-1 >= bodyHeight {
			break
		}
	}

	return m.renderPane(width, height, aiFocusSidebar, lines)
}

func (m *aiModel) renderTranscript(width, height int) string {
	innerWidth := innerPaneWidth(width)
	innerHeight := innerPaneHeight(height)
	contentWidth := m.transcriptContentWidth(innerWidth)
	content := m.renderTranscriptDisplayContent(contentWidth)
	contentLines := transcriptContentStyledLines(content)
	bodyHeight := maxInt(1, innerHeight-m.transcriptHeaderLineCount())
	headerLines := m.renderTranscriptHeaderLines(innerWidth, bodyHeight, contentLines)
	lines := append([]string(nil), headerLines...)
	lines = append(lines, m.visibleTranscriptLines(content, bodyHeight)...)

	pane := m.renderPane(width, height, aiFocusTranscript, lines)
	scrollbar := m.renderTranscriptScrollbar(bodyHeight, len(contentLines))
	if scrollbar != "" {
		pane = overlayContent(
			pane,
			scrollbar,
			maxInt(0, width-4),
			1+m.transcriptHeaderLineCount(),
			width,
		)
	}
	return overlayContent(
		pane,
		m.renderTranscriptRightEdge(height),
		maxInt(0, width-1),
		0,
		width,
	)
}

func (m *aiModel) renderComposer(height int) string {
	innerWidth := innerPaneWidth(m.width)
	innerHeight := innerPaneHeight(height)

	lines := []string{
		lipgloss.NewStyle().Width(innerWidth).Render(fitStyledPieces(innerWidth, []string{
			m.styles.paneTitle.Render("Composer"),
			m.styles.chipMuted.Render("ctrl+o context"),
			m.styles.chipMuted.Render("ctrl+s target"),
			m.styles.chipMuted.Render("ctrl+t thinking"),
		})),
		m.renderInputLine(innerWidth),
	}

	return m.renderPane(m.width, height, aiFocusComposer, headLines(lines, innerHeight))
}

func (m *aiModel) renderTranscriptContentLines(width int) []string {
	return transcriptContentStyledLines(m.renderTranscriptContent(width))
}

func (m *aiModel) renderTranscriptDisplayContentLines(width int) []string {
	return transcriptContentStyledLines(m.renderTranscriptDisplayContent(width))
}

func (m *aiModel) renderPendingPromptLines(width int) []string {
	return transcriptContentStyledLines(m.renderPendingPromptContent(width))
}

func (m *aiModel) renderPendingPromptContent(width int) []aiTranscriptContentLine {
	prompt := strings.TrimSpace(m.pendingPrompt)
	if prompt == "" {
		return nil
	}

	label := messageBlockLabel(m.currentConversation, &clientpb.AIConversationMessage{
		OperatorName: m.ctx.connection.Operator,
		Role:         "user",
	})
	return renderTranscriptBoxBlockContent(width, label, "user", []string{"pending"}, wrapText(prompt, maxInt(1, width-4)), true)
}

func (m *aiModel) renderAwaitingResponseLines(width int) []string {
	return transcriptContentStyledLines(m.renderAwaitingResponseContent(width))
}

func (m *aiModel) renderAwaitingResponseContent(width int) []aiTranscriptContentLine {
	if !m.awaitingResponse || m.thinkingAnim == nil {
		return nil
	}

	return renderTranscriptBoxBlockContent(width, "AI", "assistant", []string{strings.ToLower(m.pendingLabel())}, []string{m.thinkingAnim.Render()}, false)
}

func (m *aiModel) renderTranscriptContent(width int) []aiTranscriptContentLine {
	content := renderConversationTranscript(width, m.currentConversation)
	for _, block := range [][]aiTranscriptContentLine{
		m.renderPendingPromptContent(width),
		m.renderAwaitingResponseContent(width),
	} {
		content = appendTranscriptContentBlock(content, block)
	}
	return content
}

func (m *aiModel) renderTranscriptDisplayContent(width int) []aiTranscriptContentLine {
	content := append([]aiTranscriptContentLine(nil), m.transcriptDisplayContent(width)...)
	for _, block := range [][]aiTranscriptContentLine{
		m.renderPendingPromptContent(width),
		m.renderAwaitingResponseContent(width),
	} {
		content = appendTranscriptContentBlock(content, block)
	}
	return content
}

func (m *aiModel) renderFooter() string {
	pieces := []string{m.styles.chipMuted.Render("focus: " + m.focus.String())}
	for _, hint := range m.footerHints() {
		pieces = append(pieces, m.styles.footerHint.Render(hint))
	}
	return lipgloss.NewStyle().Width(m.width).Render(fitStyledPieces(m.width, pieces))
}

func (m *aiModel) footerHints() []string {
	switch m.focus {
	case aiFocusSidebar:
		hints := []string{"tab: next", "j/k: move", "enter: open"}
		if m.deleteTargetConversation() != nil {
			hints = append(hints, "x: delete")
		}
		hints = append(hints, "n: new", "t: thinking", "r: refresh", "q/esc: quit")
		return hints
	case aiFocusTranscript:
		hints := []string{"tab: next", "j/k: scroll", "pgup/pgdn: page", "g/G: ends", "mouse: wheel/select text"}
		if m.deleteTargetConversation() != nil {
			hints = append(hints, "x: delete")
		}
		hints = append(hints, "n: new", "t: thinking", "r: refresh", "q/esc: quit")
		return hints
	case aiFocusComposer:
		return []string{"tab: sidebar", "enter: send", "/exit: quit", "ctrl+o: context", "ctrl+s: target", "ctrl+t: thinking", "ctrl+u: clear", "ctrl+c: quit"}
	default:
		return []string{"tab: next", "q/esc: quit"}
	}
}

func (m *aiModel) renderDeleteConfirmActions(width int) string {
	cancelStyle := lipgloss.NewStyle().
		Foreground(clienttheme.DefaultMod(700)).
		Background(clienttheme.DefaultMod(50)).
		Padding(0, 1)
	deleteStyle := lipgloss.NewStyle().
		Foreground(clienttheme.Danger()).
		Padding(0, 1)

	if !m.modal.confirmDelete {
		cancelStyle = cancelStyle.
			Bold(true).
			Foreground(clienttheme.DefaultMod(900)).
			Background(clienttheme.DefaultMod(200))
	} else {
		deleteStyle = deleteStyle.
			Bold(true).
			Foreground(clienttheme.DefaultMod(900)).
			Background(clienttheme.Danger())
	}

	actions := lipgloss.JoinHorizontal(
		lipgloss.Top,
		cancelStyle.Render("Cancel"),
		" ",
		deleteStyle.Render("Delete"),
	)
	return lipgloss.Place(width, 1, lipgloss.Center, lipgloss.Center, actions)
}

func (m *aiModel) renderNewConversationInput(width int) string {
	contentWidth := maxInt(1, width-4)
	label := lipgloss.NewStyle().
		Bold(true).
		Foreground(clienttheme.Primary()).
		Render("Name")
	borderColor := clienttheme.PrimaryMod(400)
	if m.modal.focus == aiModalFocusInput {
		borderColor = clienttheme.Primary()
	}
	field := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Render(clampANSIBlock(m.renderNewConversationInputContent(contentWidth), contentWidth, 1))
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Width(width).Render(label),
		field,
	)
}

func (m *aiModel) renderNewConversationInputContent(width int) string {
	if len(m.modal.input) == 0 {
		placeholder := truncateText("New conversation", maxInt(1, width))
		if m.modal.focus == aiModalFocusInput {
			if width == 1 {
				return m.styles.cursor.Render(" ")
			}
			return m.styles.cursor.Render(" ") + m.styles.inputPlaceholder.Render(truncateText(placeholder, width-1))
		}
		return m.styles.inputPlaceholder.Render(placeholder)
	}

	visible, cursor := inputWindow(m.modal.input, m.modal.cursor, width)
	var b strings.Builder
	for i, r := range visible {
		ch := string(r)
		if i == cursor && m.modal.focus == aiModalFocusInput {
			b.WriteString(m.styles.cursor.Render(ch))
			continue
		}
		b.WriteString(m.styles.inputText.Render(ch))
	}
	if cursor == len(visible) && m.modal.focus == aiModalFocusInput && lipgloss.Width(b.String()) < width {
		b.WriteString(m.styles.cursor.Render(" "))
	}
	return b.String()
}

func (m *aiModel) renderNewConversationActions(width int) string {
	cancelStyle := lipgloss.NewStyle().
		Foreground(clienttheme.DefaultMod(700)).
		Background(clienttheme.DefaultMod(50)).
		Padding(0, 1)
	createStyle := lipgloss.NewStyle().
		Foreground(clienttheme.PrimaryMod(700)).
		Background(clienttheme.PrimaryMod(50)).
		Padding(0, 1)

	switch m.modal.focus {
	case aiModalFocusCancel:
		cancelStyle = cancelStyle.
			Bold(true).
			Foreground(clienttheme.DefaultMod(900)).
			Background(clienttheme.DefaultMod(200))
	case aiModalFocusConfirm:
		createStyle = createStyle.
			Bold(true).
			Foreground(clienttheme.DefaultMod(900)).
			Background(clienttheme.Primary())
	}

	actions := lipgloss.JoinHorizontal(
		lipgloss.Top,
		cancelStyle.Render("Cancel"),
		" ",
		createStyle.Render("Create"),
	)
	return lipgloss.Place(width, 1, lipgloss.Center, lipgloss.Center, actions)
}

func (m *aiModel) renderExperimentalWarningActions(width int) string {
	cancelStyle := lipgloss.NewStyle().
		Foreground(clienttheme.DefaultMod(900)).
		Background(clienttheme.DefaultMod(50)).
		Padding(0, 1)
	confirmStyle := lipgloss.NewStyle().
		Foreground(clienttheme.Danger()).
		Background(clienttheme.DangerMod(50)).
		Padding(0, 1)

	switch m.modal.focus {
	case aiModalFocusConfirm:
		confirmStyle = confirmStyle.
			Bold(true).
			Foreground(clienttheme.DefaultMod(900)).
			Background(clienttheme.Danger())
	default:
		cancelStyle = cancelStyle.
			Bold(true).
			Background(clienttheme.DefaultMod(200))
	}

	actions := lipgloss.JoinHorizontal(
		lipgloss.Top,
		cancelStyle.Render(aiExperimentalWarningCancelLabel),
		" ",
		confirmStyle.Render(aiExperimentalWarningConfirmLabel),
	)
	return lipgloss.Place(width, 1, lipgloss.Center, lipgloss.Center, actions)
}

func (m *aiModel) renderPane(width, height int, focus aiFocus, lines []string) string {
	innerWidth := innerPaneWidth(width)
	innerHeight := innerPaneHeight(height)
	body := clampANSIBlock(strings.Join(lines, "\n"), innerWidth, innerHeight)
	style := m.styles.pane
	if m.focus == focus {
		style = m.styles.paneFocused
	}
	return style.Width(width).MaxWidth(width).Height(height).MaxHeight(height).Render(body)
}

func (m *aiModel) startAIStartup() tea.Cmd {
	cmds := []tea.Cmd{loadAIStateCmd(m.con, m.ctx.target, "")}
	if cmd := m.scheduleTranscriptRender(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if m.listener != nil {
		cmds = append(cmds, waitForAIConversationEventCmd(m.listener))
	}
	return tea.Batch(cmds...)
}

func (m *aiModel) renderInputLine(width int) string {
	prefix := m.styles.roleUser.Render(">>>")
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

func (m *aiModel) layoutHeights() (int, int, int, int) {
	headerHeight := 1
	composerHeight := 4
	footerHeight := 1
	bodyHeight := maxInt(1, m.height-headerHeight-composerHeight-footerHeight)
	return headerHeight, composerHeight, footerHeight, bodyHeight
}

func (m *aiModel) currentPaneRects() aiPaneRects {
	if m.width < aiMinWidth || m.height < aiMinHeight {
		return aiPaneRects{}
	}

	headerHeight, composerHeight, _, bodyHeight := m.layoutHeights()
	bodyY := headerHeight
	composerY := bodyY + bodyHeight

	switch {
	case m.width >= 78:
		sidebarWidth := clampInt(m.width/4, 24, 28)
		return aiPaneRects{
			sidebar: aiPaneRect{
				x:      0,
				y:      bodyY,
				width:  sidebarWidth,
				height: bodyHeight,
			},
			transcript: aiPaneRect{
				x:      sidebarWidth,
				y:      bodyY,
				width:  maxInt(40, m.width-sidebarWidth),
				height: bodyHeight,
			},
			composer: aiPaneRect{
				x:      0,
				y:      composerY,
				width:  m.width,
				height: composerHeight,
			},
		}
	default:
		sidebarHeight := clampInt(bodyHeight/4, 5, 7)
		return aiPaneRects{
			sidebar: aiPaneRect{
				x:      0,
				y:      bodyY,
				width:  m.width,
				height: sidebarHeight,
			},
			transcript: aiPaneRect{
				x:      0,
				y:      bodyY + sidebarHeight,
				width:  m.width,
				height: maxInt(6, bodyHeight-sidebarHeight),
			},
			composer: aiPaneRect{
				x:      0,
				y:      composerY,
				width:  m.width,
				height: composerHeight,
			},
		}
	}
}

func (m *aiModel) paneFocusAt(x, y int) (aiFocus, bool) {
	rects := m.currentPaneRects()
	switch {
	case rects.sidebar.contains(x, y):
		return aiFocusSidebar, true
	case rects.transcript.contains(x, y):
		return aiFocusTranscript, true
	case rects.composer.contains(x, y):
		return aiFocusComposer, true
	default:
		return aiFocusSidebar, false
	}
}

func (m *aiModel) sidebarConversationIndexAt(x, y int) (int, bool) {
	rect := m.currentPaneRects().sidebar
	if !rect.contains(x, y) || len(m.conversations) == 0 {
		return 0, false
	}

	innerY := y - rect.y - 1
	if innerY <= 0 {
		return 0, false
	}

	maxVisible := minInt(len(m.conversations), maxInt(1, innerPaneHeight(rect.height)-1))
	idx := innerY - 1
	if idx < 0 || idx >= maxVisible {
		return 0, false
	}
	return idx, true
}

func (m *aiModel) currentTranscriptPaneSize() (int, int) {
	_, _, _, bodyHeight := m.layoutHeights()

	switch {
	case m.width >= 78:
		sidebarWidth := clampInt(m.width/4, 24, 28)
		return maxInt(40, m.width-sidebarWidth), bodyHeight
	default:
		sidebarHeight := clampInt(bodyHeight/4, 5, 7)
		return m.width, maxInt(6, bodyHeight-sidebarHeight)
	}
}

func (m *aiModel) currentTranscriptViewportSize() (int, int) {
	paneWidth, paneHeight := m.currentTranscriptPaneSize()
	innerWidth := innerPaneWidth(paneWidth)
	innerHeight := innerPaneHeight(paneHeight)
	return m.transcriptContentWidth(innerWidth), maxInt(1, innerHeight-m.transcriptHeaderLineCount())
}

func (m *aiModel) transcriptHeaderLineCount() int {
	return 2
}

func (m *aiModel) renderTranscriptHeaderLines(width, viewportHeight int, contentLines []string) []string {
	title := "No conversation selected"
	line1Label := m.styles.paneTitle.Render("Conversation")
	if m.currentConversation != nil {
		title = conversationTitle(m.currentConversation)
	}
	titleWidth := maxInt(1, width-lipgloss.Width(line1Label)-1)
	line1 := lipgloss.NewStyle().Width(width).Render(
		line1Label + " " + m.styles.heading.Render(truncateText(title, titleWidth)),
	)

	if m.currentConversation == nil {
		subtitle := "Create a thread or choose one from the sidebar."
		return []string{
			line1,
			m.styles.subtleText.Width(width).Render(truncateText(subtitle, width)),
		}
	}

	pieces := []string{
		m.styles.chip.Render("provider " + fallback(m.currentConversation.GetProvider(), "<unset>")),
		m.styles.chipMuted.Render("model " + fallback(m.currentConversation.GetModel(), "<default>")),
		m.styles.chipMuted.Render("thinking " + effectiveThinkingLevelChipLabel(m.currentConversation, m.config)),
		m.styles.chipMuted.Render(fmt.Sprintf("%d msgs", len(m.currentConversation.GetMessages()))),
	}
	if scroll := m.transcriptScrollSummary(viewportHeight, len(contentLines)); scroll != "" {
		pieces = append(pieces, m.styles.chipMuted.Render(scroll))
	}
	if updated := formatUnix(m.currentConversation.GetUpdatedAt()); updated != "<unknown>" {
		pieces = append(pieces, m.styles.subtleText.Render("updated "+updated))
	}

	return []string{
		line1,
		lipgloss.NewStyle().Width(width).Render(fitStyledPieces(width, pieces)),
	}
}

func (m *aiModel) transcriptScrollSummary(viewportHeight, totalLines int) string {
	if viewportHeight <= 0 || totalLines <= 0 {
		return ""
	}
	maxScroll := maxInt(0, totalLines-viewportHeight)
	scroll := clampInt(m.transcriptScroll, 0, maxScroll)
	if maxScroll == 0 {
		end := minInt(totalLines, scroll+viewportHeight)
		return fmt.Sprintf("lines %d/%d", end, totalLines)
	}
	if m.transcriptFollow || scroll >= maxScroll {
		scroll = maxScroll
		start := minInt(totalLines, scroll+1)
		end := minInt(totalLines, scroll+viewportHeight)
		return fmt.Sprintf("latest %d-%d/%d", start, end, totalLines)
	}
	start := minInt(totalLines, scroll+1)
	end := minInt(totalLines, scroll+viewportHeight)
	return fmt.Sprintf("lines %d-%d/%d", start, end, totalLines)
}

func (m *aiModel) visibleTranscriptLines(content []aiTranscriptContentLine, viewportHeight int) []string {
	if viewportHeight <= 0 || len(content) == 0 {
		return nil
	}

	scroll, end := m.currentTranscriptScrollRange(len(content), viewportHeight)
	visible := make([]string, 0, maxInt(0, end-scroll))
	for idx := scroll; idx < end; idx++ {
		visible = append(visible, m.renderVisibleTranscriptLine(content, idx))
	}
	return visible
}

func (m *aiModel) transcriptScrollbarCells(viewportHeight, totalLines int) []bool {
	if viewportHeight <= 0 || totalLines <= viewportHeight {
		return nil
	}

	maxScroll := maxInt(1, totalLines-viewportHeight)
	scroll := clampInt(m.transcriptScroll, 0, maxScroll)
	if m.transcriptFollow {
		scroll = maxScroll
	}

	thumbHeight := maxInt(1, (viewportHeight*viewportHeight+totalLines/2)/totalLines)
	thumbHeight = minInt(viewportHeight, thumbHeight)
	trackSpan := maxInt(0, viewportHeight-thumbHeight)
	thumbStart := 0
	if trackSpan > 0 {
		thumbStart = (scroll*trackSpan + maxScroll/2) / maxScroll
	}

	cells := make([]bool, viewportHeight)
	for i := 0; i < thumbHeight; i++ {
		idx := thumbStart + i
		if idx >= 0 && idx < len(cells) {
			cells[idx] = true
		}
	}
	return cells
}

func (m *aiModel) renderTranscriptScrollbar(viewportHeight, totalLines int) string {
	cells := m.transcriptScrollbarCells(viewportHeight, totalLines)
	if len(cells) == 0 {
		return ""
	}

	lines := make([]string, 0, len(cells))
	for _, thumb := range cells {
		lines = append(lines, m.renderTranscriptScrollbarCell(thumb))
	}
	return strings.Join(lines, "\n")
}

func (m *aiModel) currentTranscriptScrollRange(totalLines, viewportHeight int) (int, int) {
	if viewportHeight <= 0 || totalLines <= 0 {
		return 0, 0
	}

	maxScroll := maxInt(0, totalLines-viewportHeight)
	scroll := clampInt(m.transcriptScroll, 0, maxScroll)
	if m.transcriptFollow {
		scroll = maxScroll
	}
	return scroll, minInt(totalLines, scroll+viewportHeight)
}

func (m *aiModel) transcriptScrollbarRowAt(x, y int, clamp bool) (int, bool) {
	rect := m.currentPaneRects().transcript
	if rect.width <= 0 || rect.height <= 0 {
		return 0, false
	}

	contentWidth, viewportHeight := m.currentTranscriptViewportSize()
	if contentWidth <= 0 || viewportHeight <= 0 {
		return 0, false
	}

	totalLines := len(m.renderTranscriptDisplayContent(contentWidth))
	if totalLines <= viewportHeight {
		return 0, false
	}

	bodyTopY := rect.y + 1 + m.transcriptHeaderLineCount()
	bodyBottomY := bodyTopY + viewportHeight - 1
	if !rect.contains(x, y) {
		if !clamp || x < rect.x || x >= rect.x+rect.width {
			return 0, false
		}
		y = clampInt(y, bodyTopY, bodyBottomY)
	}

	innerWidth := innerPaneWidth(rect.width)
	innerX := x - rect.x - 2
	if innerX < contentWidth || innerX >= innerWidth {
		if !clamp {
			return 0, false
		}
		innerX = clampInt(innerX, contentWidth, maxInt(contentWidth, innerWidth-1))
	}

	innerY := y - rect.y - 1
	if innerY < m.transcriptHeaderLineCount() {
		if !clamp {
			return 0, false
		}
		innerY = m.transcriptHeaderLineCount()
	}

	bodyY := innerY - m.transcriptHeaderLineCount()
	if bodyY < 0 || bodyY >= viewportHeight {
		if !clamp {
			return 0, false
		}
		bodyY = clampInt(bodyY, 0, viewportHeight-1)
	}

	return bodyY, true
}

func (m *aiModel) renderTranscriptRightEdge(height int) string {
	if height <= 0 {
		return ""
	}

	style := lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(300))
	if m.focus == aiFocusTranscript {
		style = style.Foreground(clienttheme.Primary())
	}

	border := lipgloss.RoundedBorder()
	lines := make([]string, 0, height)
	for i := 0; i < height; i++ {
		switch {
		case i == 0:
			lines = append(lines, style.Render(border.TopRight))
		case i == height-1:
			lines = append(lines, style.Render(border.BottomRight))
		default:
			lines = append(lines, style.Render(border.Right))
		}
	}
	return strings.Join(lines, "\n")
}

func (m *aiModel) renderTranscriptScrollbarCell(thumb bool) string {
	if thumb {
		style := lipgloss.NewStyle().Foreground(clienttheme.PrimaryMod(500))
		if m.focus == aiFocusTranscript {
			style = style.Foreground(clienttheme.Primary()).Bold(true)
		}
		return style.Render("#")
	}
	return lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(300)).Render(":")
}

func (m *aiModel) pinTranscriptToLatest() {
	m.transcriptFollow = true
}

func (m *aiModel) syncTranscriptViewport() {
	width, viewportHeight := m.currentTranscriptViewportSize()
	if width <= 0 || viewportHeight <= 0 {
		return
	}

	contentLines := m.renderTranscriptDisplayContentLines(width)
	maxScroll := maxInt(0, len(contentLines)-viewportHeight)
	if m.transcriptFollow {
		m.transcriptScroll = maxScroll
		return
	}

	m.transcriptScroll = clampInt(m.transcriptScroll, 0, maxScroll)
	if m.transcriptScroll >= maxScroll {
		m.transcriptScroll = maxScroll
		m.transcriptFollow = true
	}
}

func (m *aiModel) scrollTranscript(delta int) {
	width, viewportHeight := m.currentTranscriptViewportSize()
	if width <= 0 || viewportHeight <= 0 {
		return
	}

	contentLines := m.renderTranscriptDisplayContentLines(width)
	maxScroll := maxInt(0, len(contentLines)-viewportHeight)
	if maxScroll == 0 {
		m.transcriptScroll = 0
		m.transcriptFollow = true
		return
	}

	start := m.transcriptScroll
	if m.transcriptFollow {
		start = maxScroll
	}
	m.transcriptScroll = clampInt(start+delta, 0, maxScroll)
	m.transcriptFollow = m.transcriptScroll >= maxScroll
}

func (m *aiModel) scrollTranscriptToScrollbarRow(row int) {
	width, viewportHeight := m.currentTranscriptViewportSize()
	if width <= 0 || viewportHeight <= 0 {
		return
	}

	totalLines := len(m.renderTranscriptDisplayContent(width))
	maxScroll := maxInt(0, totalLines-viewportHeight)
	if maxScroll == 0 {
		m.transcriptScroll = 0
		m.transcriptFollow = true
		return
	}

	row = clampInt(row, 0, viewportHeight-1)
	thumbHeight := maxInt(1, (viewportHeight*viewportHeight+totalLines/2)/totalLines)
	thumbHeight = minInt(viewportHeight, thumbHeight)
	trackSpan := maxInt(0, viewportHeight-thumbHeight)
	if trackSpan == 0 {
		m.transcriptScroll = maxScroll
		m.transcriptFollow = true
		return
	}

	thumbStart := clampInt(row-thumbHeight/2, 0, trackSpan)
	m.transcriptScroll = (thumbStart*maxScroll + trackSpan/2) / trackSpan
	m.transcriptFollow = m.transcriptScroll >= maxScroll
}

func (m *aiModel) scrollTranscriptPage(delta int) {
	_, viewportHeight := m.currentTranscriptViewportSize()
	m.scrollTranscript(delta * maxInt(1, viewportHeight-1))
}

func (m *aiModel) scrollTranscriptToTop() {
	width, viewportHeight := m.currentTranscriptViewportSize()
	if width <= 0 || viewportHeight <= 0 {
		return
	}

	contentLines := m.renderTranscriptDisplayContentLines(width)
	maxScroll := maxInt(0, len(contentLines)-viewportHeight)
	m.transcriptScroll = 0
	m.transcriptFollow = maxScroll == 0
}

func (m *aiModel) scrollTranscriptToBottom() {
	m.transcriptFollow = true
	m.syncTranscriptViewport()
}

func (m *aiModel) isBusy() bool {
	return m.loading || m.awaitingResponse
}

func (m *aiModel) moveSelection(delta int) tea.Cmd {
	if len(m.conversations) == 0 {
		return nil
	}

	next := clampInt(m.selectedConversation+delta, 0, len(m.conversations)-1)
	return m.selectConversation(next)
}

func (m *aiModel) selectConversation(index int) tea.Cmd {
	if index < 0 || index >= len(m.conversations) {
		return nil
	}

	target := m.conversations[index]
	if target == nil || strings.TrimSpace(target.GetID()) == "" {
		return nil
	}
	if index == m.selectedConversation &&
		m.currentConversation != nil &&
		strings.TrimSpace(m.currentConversation.GetID()) == strings.TrimSpace(target.GetID()) {
		return nil
	}

	m.selectedConversation = index
	m.loading = true
	m.status = "Loading conversation..."
	return loadAIStateCmd(m.con, m.ctx.target, target.GetID())
}

func (m *aiModel) showContextModal() (tea.Model, tea.Cmd) {
	m.modal = &aiModalState{
		kind:  aiModalKindContext,
		title: "Context",
	}
	m.status = "Context opened."
	return m, nil
}

func (m *aiModel) showExperimentalWarningModal() {
	m.modal = &aiModalState{
		kind:  aiModalKindExperimentalWarning,
		title: aiExperimentalWarningTitle,
		body:  aiExperimentalWarningBody,
		focus: aiModalFocusCancel,
	}
}

func (m *aiModel) acceptExperimentalWarning() (tea.Model, tea.Cmd) {
	m.modal = nil
	m.loading = true
	m.status = fallback(m.ctx.status, "Loading AI conversations from the server...")
	return m, m.startAIStartup()
}

func (m *aiModel) showNewConversationModal() (tea.Model, tea.Cmd) {
	if m.loading {
		m.status = "A conversation sync is already in progress."
		return m, nil
	}

	input := []rune("New conversation")
	m.modal = &aiModalState{
		kind:   aiModalKindNewConversation,
		title:  "New Conversation",
		focus:  aiModalFocusInput,
		input:  input,
		cursor: len(input),
	}
	m.status = "Name the new conversation."
	return m, nil
}

func (m *aiModel) showThinkingLevelModal() (tea.Model, tea.Cmd) {
	if m.loading {
		m.status = "Wait for the current conversation sync to finish before changing the thinking level."
		return m, nil
	}
	if m.awaitingResponse || m.submittingPrompt {
		m.status = "Wait for the current AI request to finish before changing the thinking level."
		return m, nil
	}
	if m.currentConversation == nil || strings.TrimSpace(m.currentConversation.GetID()) == "" {
		m.status = "No AI conversation selected."
		return m, nil
	}

	m.modal = &aiModalState{
		kind:           aiModalKindThinkingLevel,
		title:          "Thinking Level",
		selectedOption: aiThinkingLevelOptionIndex(m.currentConversation.GetThinkingLevel()),
		conversationID: m.currentConversation.GetID(),
	}
	m.status = "Select a thinking level for future turns."
	return m, nil
}

func (m *aiModel) showTargetSelectModal() (tea.Model, tea.Cmd) {
	if m.loading {
		m.status = "Wait for the current conversation sync to finish before changing the active target."
		return m, nil
	}
	if m.awaitingResponse || m.submittingPrompt {
		m.status = "Wait for the current AI request to finish before changing the active target."
		return m, nil
	}

	conversationTargetSessionID := ""
	conversationTargetBeaconID := ""
	if m.currentConversation != nil {
		conversationTargetSessionID = m.currentConversation.GetTargetSessionID()
		conversationTargetBeaconID = m.currentConversation.GetTargetBeaconID()
	}

	m.targetSelectionOptions = nil
	m.modal = &aiModalState{
		kind:   aiModalKindTargetSelect,
		title:  "Active Target",
		status: "Loading sessions and beacons from the server...",
	}
	m.status = "Selecting an active target."
	return m, loadAITargetSelectionOptionsCmd(m.con, m.ctx.target, conversationTargetSessionID, conversationTargetBeaconID)
}

func (m *aiModel) showDeleteConversationModal() (tea.Model, tea.Cmd) {
	if m.loading {
		m.status = "Wait for the current conversation sync to finish before deleting."
		return m, nil
	}
	if m.submittingPrompt {
		m.status = "Wait for the current prompt to finish saving before deleting the conversation."
		return m, nil
	}

	target := m.deleteTargetConversation()
	if target == nil || strings.TrimSpace(target.GetID()) == "" {
		m.status = "No AI conversation selected."
		return m, nil
	}

	title := conversationTitle(target)
	m.modal = &aiModalState{
		kind:           aiModalKindDeleteConfirm,
		title:          "Delete Conversation?",
		body:           fmt.Sprintf("Delete %q and all of its stored messages from the server? This cannot be undone.", title),
		conversationID: target.GetID(),
		selectedID:     m.nextConversationIDAfterDelete(target.GetID()),
		status:         fmt.Sprintf("Deleted %q.", title),
	}
	return m, nil
}

func (m *aiModel) deleteTargetConversation() *clientpb.AIConversation {
	if m.selectedConversation >= 0 && m.selectedConversation < len(m.conversations) {
		target := m.conversations[m.selectedConversation]
		if target == nil {
			return m.currentConversation
		}
		if m.currentConversation != nil && strings.TrimSpace(m.currentConversation.GetID()) == strings.TrimSpace(target.GetID()) {
			return m.currentConversation
		}
		return target
	}
	return m.currentConversation
}

func (m *aiModel) nextConversationIDAfterDelete(conversationID string) string {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" || len(m.conversations) == 0 {
		return ""
	}

	idx := conversationIndexByID(m.conversations, conversationID)
	if idx < 0 {
		return ""
	}
	if len(m.conversations) == 1 {
		return ""
	}
	if idx < len(m.conversations)-1 {
		return m.conversations[idx+1].GetID()
	}
	return m.conversations[idx-1].GetID()
}

func (m *aiModel) removeConversation(conversationID string) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" || len(m.conversations) == 0 {
		return
	}

	filtered := make([]*clientpb.AIConversation, 0, len(m.conversations))
	for _, conversation := range m.conversations {
		if conversation == nil || strings.TrimSpace(conversation.GetID()) == conversationID {
			continue
		}
		filtered = append(filtered, conversation)
	}
	m.conversations = filtered

	next := conversationIndexByID(m.conversations, m.selectedConversationID())
	if next < 0 {
		next = 0
	}
	m.selectedConversation = next
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

func (m *aiModel) thinkingLevelModalOptions() []aiThinkingLevelOption {
	return aiThinkingLevelOptions(m.defaultThinkingLevel())
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

func (m *aiModel) defaultThinkingLevel() string {
	if m.config == nil {
		return ""
	}
	return normalizeAIThinkingLevel(m.config.GetThinkingLevel())
}

func (m *aiModel) pendingLabel() string {
	switch effectiveConversationThinkingLevel(m.currentConversation, m.config) {
	case "low", "medium", "high", "xhigh":
		return "Thinking"
	default:
		return "Working"
	}
}

func (m *aiModel) pendingStatus() string {
	return "Waiting for AI response..."
}

func newAIToastProgress(level string) progress.Model {
	bar := progress.New(
		progress.WithWidth(1),
		progress.WithoutPercentage(),
		progress.WithColors(aiToastAccentColor(level)),
		progress.WithFillCharacters(progress.DefaultFullCharFullBlock, ' '),
	)
	return bar
}

func aiToastAccentColor(level string) color.Color {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "error", "warn", "warning":
		return clienttheme.Danger()
	case "success":
		return clienttheme.Success()
	default:
		return clienttheme.Primary()
	}
}

func (t *aiToastState) remaining(now time.Time) time.Duration {
	if t == nil {
		return 0
	}
	if now.IsZero() {
		now = time.Now()
	}
	remaining := t.expiresAt.Sub(now)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (t *aiToastState) fractionRemaining(now time.Time) float64 {
	if t == nil {
		return 0
	}
	total := t.expiresAt.Sub(t.createdAt)
	if total <= 0 {
		return 0
	}
	return clampFloat64(float64(t.remaining(now))/float64(total), 0, 1)
}

func formatAIToastTimeLeft(remaining time.Duration) string {
	if remaining < 0 {
		remaining = 0
	}
	return fmt.Sprintf("%.1fs left", remaining.Seconds())
}

func (m *aiModel) showToast(level string, message string) tea.Cmd {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil
	}

	now := time.Now()
	m.nextToastID++
	m.toast = &aiToastState{
		id:        m.nextToastID,
		level:     strings.ToLower(strings.TrimSpace(level)),
		message:   message,
		createdAt: now,
		expiresAt: now.Add(aiToastDuration),
		bar:       newAIToastProgress(level),
	}
	return aiToastExpiryCmd(m.toast.id)
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
	m.syncTranscriptViewport()
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
	optimistic.OperatorName = normalizedConversationOperatorName(optimistic.GetOperatorName())
	if optimistic.GetOperatorName() == "" {
		optimistic.OperatorName = normalizedConversationOperatorName(m.ctx.connection.Operator)
	}

	if message != nil {
		pending := cloneConversationMessage(message)
		if pending != nil {
			if strings.TrimSpace(pending.GetConversationID()) == "" {
				pending.ConversationID = conversationID
			}
			pending.OperatorName = normalizedConversationOperatorName(pending.GetOperatorName())
			if strings.TrimSpace(pending.GetOperatorName()) == "" {
				pending.OperatorName = optimistic.GetOperatorName()
			}
			if strings.TrimSpace(pending.GetProvider()) == "" {
				pending.Provider = optimistic.GetProvider()
			}
			if strings.TrimSpace(pending.GetModel()) == "" {
				pending.Model = optimistic.GetModel()
			}

			m.upsertConversationMessage(optimistic, pending)

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
	m.syncTranscriptViewport()
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

func (m *aiModel) applyConversationEvent(event *clientpb.AIConversationEvent) {
	if event == nil {
		return
	}

	conversationID := strings.TrimSpace(event.GetConversationID())
	if conversationID == "" && event.GetConversation() != nil {
		conversationID = strings.TrimSpace(event.GetConversation().GetID())
	}
	if conversationID == "" && event.GetMessage() != nil {
		conversationID = strings.TrimSpace(event.GetMessage().GetConversationID())
	}

	if event.GetConversation() != nil {
		m.applyConversationSnapshot(event.GetConversation())
	}
	if event.GetMessage() != nil {
		m.applyConversationMessageEvent(conversationID, event.GetMessage())
	}

	switch event.GetEventType() {
	case clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_CONVERSATION_DELETED:
		m.removeConversation(conversationID)
		if m.currentConversation != nil && strings.TrimSpace(m.currentConversation.GetID()) == conversationID {
			m.currentConversation = nil
			m.awaitingResponse = false
			m.submittingPrompt = false
			m.pendingPrompt = ""
			if len(m.conversations) > 0 {
				m.selectedConversation = clampInt(m.selectedConversation, 0, len(m.conversations)-1)
			} else {
				m.selectedConversation = 0
			}
		}
		if strings.TrimSpace(m.status) == "" {
			m.status = "Conversation deleted on the server."
		}

	case clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_FAILED:
		if strings.TrimSpace(event.GetErrorText()) != "" && m.currentConversation != nil && strings.TrimSpace(m.currentConversation.GetID()) == conversationID {
			m.status = "AI request failed: " + strings.TrimSpace(event.GetErrorText())
		}

	case clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_STARTED:
		if m.currentConversation != nil && strings.TrimSpace(m.currentConversation.GetID()) == conversationID {
			m.status = m.pendingStatus()
		}

	case clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_TURN_COMPLETED:
		if m.currentConversation != nil && strings.TrimSpace(m.currentConversation.GetID()) == conversationID {
			m.pendingPrompt = ""
			if m.status == "" || strings.HasPrefix(m.status, "Waiting for AI response") || strings.HasPrefix(m.status, "Thinking") {
				m.status = "AI response synced from the server."
			}
		}

	case clientpb.AIConversationEventType_AI_CONVERSATION_EVENT_TYPE_MESSAGE_COMPLETED:
		if event.GetMessage() != nil && strings.EqualFold(strings.TrimSpace(event.GetMessage().GetRole()), "assistant") {
			m.pendingPrompt = ""
			if m.currentConversation != nil && strings.TrimSpace(m.currentConversation.GetID()) == conversationID {
				m.status = "AI response synced from the server."
			}
		}
	}

	if m.currentConversation != nil {
		m.selectedConversation = conversationIndexByID(m.conversations, m.currentConversation.GetID())
		if m.selectedConversation < 0 {
			m.selectedConversation = 0
		}
	}
	m.invalidateTranscriptCache()
}

func (m *aiModel) applyConversationSnapshot(conversation *clientpb.AIConversation) {
	if conversation == nil {
		return
	}

	summary := cloneConversation(conversation)
	if summary == nil {
		return
	}
	summary.Messages = nil
	m.selectedConversation = m.upsertConversation(summary)

	conversationID := strings.TrimSpace(conversation.GetID())
	switch {
	case m.currentConversation == nil && conversationID != "":
		if m.selectedConversationID() == "" || m.selectedConversationID() == conversationID {
			m.currentConversation = summary
		}
	case m.currentConversation != nil && strings.TrimSpace(m.currentConversation.GetID()) == conversationID:
		mergeConversationMetadata(m.currentConversation, conversation)
	}
}

func (m *aiModel) applyConversationMessageEvent(conversationID string, message *clientpb.AIConversationMessage) {
	if message == nil {
		return
	}
	if conversationID == "" {
		conversationID = strings.TrimSpace(message.GetConversationID())
	}
	if conversationID == "" {
		return
	}

	if m.currentConversation == nil || strings.TrimSpace(m.currentConversation.GetID()) != conversationID {
		return
	}
	m.upsertConversationMessage(m.currentConversation, message)
}

func (m *aiModel) upsertConversationMessage(conversation *clientpb.AIConversation, message *clientpb.AIConversationMessage) {
	if conversation == nil || message == nil {
		return
	}

	cloned := cloneConversationMessage(message)
	if cloned == nil {
		return
	}

	for idx, existing := range conversation.GetMessages() {
		if sameConversationMessage(existing, cloned) {
			conversation.Messages[idx] = cloned
			sortConversationMessages(conversation.Messages)
			return
		}
	}

	conversation.Messages = append(conversation.Messages, cloned)
	sortConversationMessages(conversation.Messages)
}

func (m *aiModel) renderTranscriptMarkdown(width int) string {
	lines := m.renderTranscriptMarkdownLines(width)
	return strings.Join(lines, "\n")
}

func (m *aiModel) renderTranscriptMarkdownLines(width int) []string {
	markdown := buildConversationMarkdown(m.currentConversation)
	rendered, err := renderMarkdownWithGlow(width, markdown)
	if err != nil {
		rendered = markdown
	}

	rendered = strings.TrimSpace(rendered)
	if rendered == "" {
		rendered = "_No messages yet._"
	}
	return strings.Split(rendered, "\n")
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
	m.transcriptCacheContent = nil
	m.clearTranscriptSelection()
}

func (m *aiModel) transcriptDisplayLines(width int) []string {
	return transcriptContentStyledLines(m.transcriptDisplayContent(width))
}

func (m *aiModel) transcriptDisplayContent(width int) []aiTranscriptContentLine {
	key := m.transcriptRenderKey(width)
	if key == m.transcriptCacheKey && len(m.transcriptCacheContent) > 0 {
		return m.transcriptCacheContent
	}
	if key == m.transcriptCacheKey && len(m.transcriptCacheLines) > 0 {
		content := make([]aiTranscriptContentLine, 0, len(m.transcriptCacheLines))
		for _, line := range m.transcriptCacheLines {
			content = append(content, aiTranscriptContentLine{styled: line})
		}
		return content
	}

	placeholder := "Rendering transcript..."
	if m.currentConversation == nil {
		placeholder = "Loading conversation..."
	}
	return []aiTranscriptContentLine{{
		styled: m.styles.subtleText.Width(maxInt(1, width)).Render(truncateText(placeholder, maxInt(1, width))),
	}}
}

func (m *aiModel) transcriptContentWidth(innerWidth int) int {
	return maxInt(1, innerWidth-m.transcriptScrollbarColumns(innerWidth))
}

func (m *aiModel) transcriptScrollbarColumns(innerWidth int) int {
	if innerWidth <= 1 {
		return 0
	}
	if innerWidth == 2 {
		return 1
	}
	return aiTranscriptScrollbarWidth + 1
}

func (m *aiModel) currentTranscriptWidth() int {
	paneWidth, _ := m.currentTranscriptPaneSize()
	return m.transcriptContentWidth(innerPaneWidth(paneWidth))
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

	return func() tea.Msg {
		content := renderConversationTranscript(width, conversation)
		lines := transcriptContentStyledLines(content)
		rendered := strings.Join(lines, "\n")
		rendered = strings.TrimSpace(rendered)
		if rendered == "" {
			rendered = "_No messages yet._"
		}
		return aiTranscriptRenderedMsg{
			key:      key,
			rendered: rendered,
			lines:    strings.Split(rendered, "\n"),
			content:  content,
		}
	}
}

func (m *aiModel) shouldSkipConversationEventReload(conversation *clientpb.AIConversation) bool {
	if conversation == nil || !m.awaitingResponse {
		return false
	}
	if isAIConversationDeleteEvent(conversation) {
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

func isAIConversationDeleteEvent(conversation *clientpb.AIConversation) bool {
	if conversation == nil {
		return false
	}
	if strings.TrimSpace(conversation.GetID()) == "" {
		return false
	}
	if conversation.GetCreatedAt() != 0 || conversation.GetUpdatedAt() != 0 {
		return false
	}
	if len(conversation.GetMessages()) != 0 {
		return false
	}
	if strings.TrimSpace(conversation.GetProvider()) != "" ||
		strings.TrimSpace(conversation.GetModel()) != "" ||
		strings.TrimSpace(conversation.GetThinkingLevel()) != "" ||
		strings.TrimSpace(conversation.GetTitle()) != "" ||
		strings.TrimSpace(conversation.GetSummary()) != "" ||
		strings.TrimSpace(conversation.GetSystemPrompt()) != "" {
		return false
	}
	return true
}

func (m aiFocus) String() string {
	switch m {
	case aiFocusSidebar:
		return "sidebar"
	case aiFocusTranscript:
		return "conversation"
	case aiFocusComposer:
		return "composer"
	default:
		return "unknown"
	}
}

func aiView(content string) tea.View {
	view := tea.NewView(content)
	view.AltScreen = true
	view.MouseMode = tea.MouseModeAllMotion
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

func aiToastExpiryCmd(id uint64) tea.Cmd {
	if id == 0 {
		return nil
	}
	return tea.Tick(aiToastDuration, func(time.Time) tea.Msg {
		return aiToastExpiredMsg{id: id}
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

func loadAIStateCmd(con *console.SliverClient, target aiTargetSummary, selectedID string) tea.Cmd {
	return loadAIStateWithStatusCmd(con, target, selectedID, "")
}

func loadAIStateWithStatusCmd(con *console.SliverClient, target aiTargetSummary, selectedID string, baseStatus string) tea.Cmd {
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
		status := strings.TrimSpace(baseStatus)
		if len(conversations) == 0 {
			createdConversation, err := con.Rpc.SaveAIConversation(grpcCtx, &clientpb.AIConversation{
				Provider:        config.GetProvider(),
				Model:           config.GetModel(),
				Title:           "New conversation",
				TargetSessionID: strings.TrimSpace(target.SessionID),
				TargetBeaconID:  strings.TrimSpace(target.BeaconID),
			})
			if err != nil {
				return aiAsyncErrMsg{err: err}
			}
			conversations = []*clientpb.AIConversation{createdConversation}
			activeID = createdConversation.GetID()
			if status == "" {
				status = "Created a new AI conversation."
			} else {
				status += " Created a new AI conversation."
			}
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

func loadAITargetSelectionOptionsCmd(con *console.SliverClient, activeTarget aiTargetSummary, conversationTargetSessionID string, conversationTargetBeaconID string) tea.Cmd {
	return func() tea.Msg {
		if con == nil || con.Rpc == nil {
			return aiTargetOptionsLoadedMsg{status: "AI RPC client is unavailable."}
		}

		grpcCtx, cancel := con.GrpcContext(nil)
		defer cancel()

		sessionsResp, err := con.Rpc.GetSessions(grpcCtx, &commonpb.Empty{})
		if err != nil {
			return aiTargetOptionsLoadedMsg{status: "Failed to load sessions: " + err.Error()}
		}

		beaconsResp, err := con.Rpc.GetBeacons(grpcCtx, &commonpb.Empty{})
		if err != nil {
			return aiTargetOptionsLoadedMsg{status: "Failed to load beacons: " + err.Error()}
		}

		options := buildAITargetSelectionOptions(sessionsResp.GetSessions(), beaconsResp.GetBeacons(), activeTarget)
		status := "Select the session or beacon to make active."
		if len(options) == 0 {
			status = "No sessions or beacons are currently available."
		}

		return aiTargetOptionsLoadedMsg{
			options:        options,
			selectedOption: aiTargetSelectionOptionIndex(options, activeTarget, conversationTargetSessionID, conversationTargetBeaconID),
			status:         status,
		}
	}
}

func createConversationCmd(con *console.SliverClient, target aiTargetSummary, provider string, model string, title string) tea.Cmd {
	return func() tea.Msg {
		if con == nil || con.Rpc == nil {
			return aiAsyncErrMsg{err: fmt.Errorf("AI RPC client is unavailable")}
		}

		grpcCtx, cancel := con.GrpcContext(nil)
		defer cancel()

		conversation, err := con.Rpc.SaveAIConversation(grpcCtx, &clientpb.AIConversation{
			Provider:        provider,
			Model:           strings.TrimSpace(model),
			Title:           strings.TrimSpace(title),
			TargetSessionID: strings.TrimSpace(target.SessionID),
			TargetBeaconID:  strings.TrimSpace(target.BeaconID),
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

func deleteConversationCmd(con *console.SliverClient, conversationID string, selectedID string, status string) tea.Cmd {
	return func() tea.Msg {
		if con == nil || con.Rpc == nil {
			return aiAsyncErrMsg{err: fmt.Errorf("AI RPC client is unavailable")}
		}

		grpcCtx, cancel := con.GrpcContext(nil)
		defer cancel()

		_, err := con.Rpc.DeleteAIConversation(grpcCtx, &clientpb.AIConversationReq{ID: conversationID})
		if err != nil {
			return aiAsyncErrMsg{err: err}
		}

		return aiConversationDeletedMsg{
			conversationID: conversationID,
			selectedID:     selectedID,
			status:         status,
		}
	}
}

func updateConversationThinkingLevelCmd(con *console.SliverClient, conversation *clientpb.AIConversation, thinkingLevel string) tea.Cmd {
	return func() tea.Msg {
		if con == nil || con.Rpc == nil {
			return aiAsyncErrMsg{err: fmt.Errorf("AI RPC client is unavailable")}
		}
		if conversation == nil || strings.TrimSpace(conversation.GetID()) == "" {
			return aiAsyncErrMsg{err: fmt.Errorf("AI conversation is unavailable")}
		}

		grpcCtx, cancel := con.GrpcContext(nil)
		defer cancel()

		req := conversationSaveRequest(conversation)
		req.ThinkingLevel = normalizeAIThinkingLevel(thinkingLevel)

		savedConversation, err := con.Rpc.SaveAIConversation(grpcCtx, req)
		if err != nil {
			return aiAsyncErrMsg{err: err}
		}

		status := "Thinking level reset to provider default."
		if req.GetThinkingLevel() != "" {
			status = "Thinking level set to " + req.GetThinkingLevel() + "."
		}
		return aiConversationUpdatedMsg{
			conversation: savedConversation,
			status:       status,
		}
	}
}

func updateConversationTargetCmd(con *console.SliverClient, conversation *clientpb.AIConversation, target aiTargetSummary) tea.Cmd {
	return func() tea.Msg {
		if con == nil || con.Rpc == nil {
			return aiAsyncErrMsg{err: fmt.Errorf("AI RPC client is unavailable")}
		}
		if conversation == nil || strings.TrimSpace(conversation.GetID()) == "" {
			return aiAsyncErrMsg{err: fmt.Errorf("AI conversation is unavailable")}
		}

		grpcCtx, cancel := con.GrpcContext(nil)
		defer cancel()

		sessionID, beaconID := normalizedAITargetIDs(target.SessionID, target.BeaconID)
		req := conversationSaveRequest(conversation)
		req.TargetSessionID = sessionID
		req.TargetBeaconID = beaconID

		savedConversation, err := con.Rpc.SaveAIConversation(grpcCtx, req)
		if err != nil {
			return aiAsyncErrMsg{err: err}
		}

		return aiConversationTargetUpdatedMsg{
			conversation: savedConversation,
			target:       aiTargetSummary{SessionID: sessionID, BeaconID: beaconID, Label: target.Label},
			status:       "Active target set to " + fallback(target.Label, "selected target") + ".",
		}
	}
}

func submitPromptCmd(con *console.SliverClient, target aiTargetSummary, conversation *clientpb.AIConversation, provider string, model string, prompt string) tea.Cmd {
	return func() tea.Msg {
		return submitPromptMsg(con, target, conversation, provider, model, prompt)
	}
}

func submitPromptMsg(con *console.SliverClient, target aiTargetSummary, conversation *clientpb.AIConversation, provider string, model string, prompt string) tea.Msg {
	if con == nil || con.Rpc == nil {
		return aiAsyncErrMsg{err: fmt.Errorf("AI RPC client is unavailable")}
	}

	grpcCtx, cancel := con.GrpcContext(nil)
	defer cancel()

	activeConversation := conversation
	var err error
	if activeConversation == nil || strings.TrimSpace(activeConversation.GetID()) == "" {
		activeConversation, err = con.Rpc.SaveAIConversation(grpcCtx, &clientpb.AIConversation{
			Provider:        provider,
			Model:           strings.TrimSpace(model),
			Title:           promptConversationTitle(prompt),
			TargetSessionID: strings.TrimSpace(target.SessionID),
			TargetBeaconID:  strings.TrimSpace(target.BeaconID),
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
			if event == nil {
				continue
			}
			switch event.GetEventType() {
			case consts.AIConversationEvent:
				aiEvent := &clientpb.AIConversationEvent{}
				if len(event.GetData()) > 0 {
					if err := proto.Unmarshal(event.GetData(), aiEvent); err != nil {
						continue
					}
				}
				return aiConversationEventMsg{event: aiEvent}
			case consts.ClientToastEvent:
				return aiToastMsg{
					level:   strings.TrimSpace(event.GetErr()),
					message: strings.TrimSpace(string(event.GetData())),
				}
			}
		}
	}
}

func buildConversationMarkdown(conversation *clientpb.AIConversation) string {
	if conversation == nil {
		return "### No conversation selected\n\nCreate a new conversation or submit a prompt to start one."
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

func normalizeAIThinkingLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low", "medium", "high", "xhigh", "disabled":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func effectiveConversationThinkingLevel(conversation *clientpb.AIConversation, config *clientpb.AIConfigSummary) string {
	if conversation != nil {
		if thinkingLevel := normalizeAIThinkingLevel(conversation.GetThinkingLevel()); thinkingLevel != "" {
			return thinkingLevel
		}
	}
	if config != nil {
		return normalizeAIThinkingLevel(config.GetThinkingLevel())
	}
	return ""
}

func defaultThinkingLevelSummary(config *clientpb.AIConfigSummary) string {
	if config == nil {
		return "provider default"
	}
	if thinkingLevel := normalizeAIThinkingLevel(config.GetThinkingLevel()); thinkingLevel != "" {
		return thinkingLevel
	}
	return "provider default"
}

func conversationThinkingLevelSummary(conversation *clientpb.AIConversation, config *clientpb.AIConfigSummary) string {
	if conversation != nil {
		if thinkingLevel := normalizeAIThinkingLevel(conversation.GetThinkingLevel()); thinkingLevel != "" {
			return thinkingLevel
		}
	}
	if defaultLevel := normalizeAIThinkingLevel(config.GetThinkingLevel()); defaultLevel != "" {
		return "provider default (" + defaultLevel + ")"
	}
	return "provider default"
}

func effectiveThinkingLevelChipLabel(conversation *clientpb.AIConversation, config *clientpb.AIConfigSummary) string {
	if thinkingLevel := effectiveConversationThinkingLevel(conversation, config); thinkingLevel != "" {
		return thinkingLevel
	}
	return "default"
}

func aiThinkingLevelOptionIndex(value string) int {
	for idx, option := range aiThinkingLevelOptions("") {
		if option.value == normalizeAIThinkingLevel(value) {
			return idx
		}
	}
	return 0
}

func aiThinkingLevelOptions(defaultThinkingLevel string) []aiThinkingLevelOption {
	defaultDescription := "Use the server or provider default."
	if defaultThinkingLevel = normalizeAIThinkingLevel(defaultThinkingLevel); defaultThinkingLevel != "" {
		defaultDescription = "Use the server or provider default (" + defaultThinkingLevel + ")."
	}

	return []aiThinkingLevelOption{
		{
			label:       "Provider default",
			value:       "",
			description: defaultDescription,
		},
		{
			label:       "Low",
			value:       "low",
			description: "Prefer faster responses with lighter reasoning effort.",
		},
		{
			label:       "Medium",
			value:       "medium",
			description: "Balance speed and reasoning depth.",
		},
		{
			label:       "High",
			value:       "high",
			description: "Prefer deeper reasoning for harder turns.",
		},
		{
			label:       "X-High",
			value:       "xhigh",
			description: "Prefer the maximum supported reasoning effort.",
		},
		{
			label:       "Disabled",
			value:       "disabled",
			description: "Do not request an explicit reasoning effort from the backend.",
		},
	}
}

func buildAITargetSelectionOptions(sessions []*clientpb.Session, beacons []*clientpb.Beacon, activeTarget aiTargetSummary) []aiTargetSelectionOption {
	sessionList := make([]*clientpb.Session, 0, len(sessions))
	for _, session := range sessions {
		if session != nil {
			sessionList = append(sessionList, session)
		}
	}
	beaconList := make([]*clientpb.Beacon, 0, len(beacons))
	for _, beacon := range beacons {
		if beacon != nil {
			beaconList = append(beaconList, beacon)
		}
	}

	sort.SliceStable(sessionList, func(i, j int) bool {
		return aiTargetSelectionSortKey(sessionList[i].GetName(), sessionList[i].GetHostname(), sessionList[i].GetID()) <
			aiTargetSelectionSortKey(sessionList[j].GetName(), sessionList[j].GetHostname(), sessionList[j].GetID())
	})
	sort.SliceStable(beaconList, func(i, j int) bool {
		return aiTargetSelectionSortKey(beaconList[i].GetName(), beaconList[i].GetHostname(), beaconList[i].GetID()) <
			aiTargetSelectionSortKey(beaconList[j].GetName(), beaconList[j].GetHostname(), beaconList[j].GetID())
	})

	options := make([]aiTargetSelectionOption, 0, len(sessionList)+len(beaconList))
	for _, session := range sessionList {
		target := aiSessionTargetSummary(session)
		options = append(options, aiTargetSelectionOption{
			label: sessionTargetSelectionLabel(target),
			metadata: []string{
				aiTargetSelectionMetadataLine("ID "+shortenID(session.GetID()), fallback(target.Host, "<unknown host>"), fallback(target.OS, "unknown")+"/"+fallback(target.Arch, "unknown"), fallback(target.Mode, "interactive session")),
				aiTargetSelectionMetadataLine("C2 "+fallback(target.C2, "unknown"), "User: "+fallback(session.GetUsername(), "<unknown>"), fmt.Sprintf("PID: %d", session.GetPID()), "Remote: "+fallback(session.GetRemoteAddress(), "<unknown>")),
			},
			target:  target,
			session: session,
			active:  sameAITargetSelectionIDs(session.GetID(), "", activeTarget.SessionID, activeTarget.BeaconID),
		})
	}
	for _, beacon := range beaconList {
		target := aiBeaconTargetSummary(beacon)
		options = append(options, aiTargetSelectionOption{
			label: beaconTargetSelectionLabel(target),
			metadata: []string{
				aiTargetSelectionMetadataLine("ID "+shortenID(beacon.GetID()), fallback(target.Host, "<unknown host>"), fallback(target.OS, "unknown")+"/"+fallback(target.Arch, "unknown"), fallback(target.Mode, "asynchronous beacon")),
				aiTargetSelectionMetadataLine("C2 "+fallback(target.C2, "unknown"), "User: "+fallback(beacon.GetUsername(), "<unknown>"), fmt.Sprintf("PID: %d", beacon.GetPID()), "Remote: "+fallback(beacon.GetRemoteAddress(), "<unknown>")),
				aiTargetSelectionMetadataLine(fmt.Sprintf("Interval: %s", time.Duration(beacon.GetInterval()).String()), "Next checkin: "+formatUnix(beacon.GetNextCheckin())),
			},
			target: target,
			beacon: beacon,
			active: sameAITargetSelectionIDs("", beacon.GetID(), activeTarget.SessionID, activeTarget.BeaconID),
		})
	}
	return options
}

func aiTargetSelectionSortKey(name, host, id string) string {
	return strings.ToLower(strings.TrimSpace(name) + "\x00" + strings.TrimSpace(host) + "\x00" + strings.TrimSpace(id))
}

func sessionTargetSelectionLabel(target aiTargetSummary) string {
	return fallback(target.Label, "Session "+fallback(target.SessionID, "<unknown>"))
}

func beaconTargetSelectionLabel(target aiTargetSummary) string {
	return fallback(target.Label, "Beacon "+fallback(target.BeaconID, "<unknown>"))
}

func aiTargetSelectionMetadataLine(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if text := strings.TrimSpace(part); text != "" {
			filtered = append(filtered, text)
		}
	}
	return strings.Join(filtered, " | ")
}

func aiTargetSelectionOptionIndex(options []aiTargetSelectionOption, activeTarget aiTargetSummary, conversationTargetSessionID string, conversationTargetBeaconID string) int {
	activeSessionID, activeBeaconID := normalizedAITargetIDs(activeTarget.SessionID, activeTarget.BeaconID)
	for idx, option := range options {
		optionSessionID, optionBeaconID := normalizedAITargetIDs(option.target.SessionID, option.target.BeaconID)
		if sameAITargetSelectionIDs(optionSessionID, optionBeaconID, activeSessionID, activeBeaconID) {
			return idx
		}
	}

	conversationTargetSessionID, conversationTargetBeaconID = normalizedAITargetIDs(conversationTargetSessionID, conversationTargetBeaconID)
	for idx, option := range options {
		optionSessionID, optionBeaconID := normalizedAITargetIDs(option.target.SessionID, option.target.BeaconID)
		if sameAITargetSelectionIDs(optionSessionID, optionBeaconID, conversationTargetSessionID, conversationTargetBeaconID) {
			return idx
		}
	}

	return 0
}

func targetSelectionVisibleRange(blocks [][]string, maxLines int, selected int) (int, int) {
	if len(blocks) == 0 || maxLines <= 0 {
		return 0, 0
	}

	selected = clampInt(selected, 0, len(blocks)-1)
	start, end := selected, selected+1
	used := len(blocks[selected])
	for {
		extended := false
		if start > 0 && used+len(blocks[start-1]) <= maxLines {
			start--
			used += len(blocks[start])
			extended = true
		}
		if end < len(blocks) && used+len(blocks[end]) <= maxLines {
			used += len(blocks[end])
			end++
			extended = true
		}
		if !extended {
			break
		}
	}
	return start, end
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

func conversationSaveRequest(conversation *clientpb.AIConversation) *clientpb.AIConversation {
	if conversation == nil {
		return &clientpb.AIConversation{}
	}
	return &clientpb.AIConversation{
		ID:              strings.TrimSpace(conversation.GetID()),
		OperatorName:    strings.TrimSpace(conversation.GetOperatorName()),
		Provider:        strings.TrimSpace(conversation.GetProvider()),
		Model:           strings.TrimSpace(conversation.GetModel()),
		ThinkingLevel:   normalizeAIThinkingLevel(conversation.GetThinkingLevel()),
		Title:           strings.TrimSpace(conversation.GetTitle()),
		Summary:         strings.TrimSpace(conversation.GetSummary()),
		SystemPrompt:    strings.TrimSpace(conversation.GetSystemPrompt()),
		ActiveTurnID:    strings.TrimSpace(conversation.GetActiveTurnID()),
		TurnState:       conversation.GetTurnState(),
		TargetSessionID: strings.TrimSpace(conversation.GetTargetSessionID()),
		TargetBeaconID:  strings.TrimSpace(conversation.GetTargetBeaconID()),
	}
}

func normalizedAITargetIDs(sessionID, beaconID string) (string, string) {
	sessionID = strings.TrimSpace(sessionID)
	beaconID = strings.TrimSpace(beaconID)
	switch {
	case sessionID != "":
		return sessionID, ""
	case beaconID != "":
		return "", beaconID
	default:
		return "", ""
	}
}

func sameAITargetSelectionIDs(leftSessionID, leftBeaconID, rightSessionID, rightBeaconID string) bool {
	leftSessionID, leftBeaconID = normalizedAITargetIDs(leftSessionID, leftBeaconID)
	rightSessionID, rightBeaconID = normalizedAITargetIDs(rightSessionID, rightBeaconID)
	return leftSessionID == rightSessionID && leftBeaconID == rightBeaconID
}

func sameAITargetSelectionSummary(left aiTargetSummary, right aiTargetSummary) bool {
	return sameAITargetSelectionIDs(left.SessionID, left.BeaconID, right.SessionID, right.BeaconID)
}

func conversationUsesTarget(conversation *clientpb.AIConversation, target aiTargetSummary) bool {
	if conversation == nil {
		return false
	}
	return sameAITargetSelectionIDs(conversation.GetTargetSessionID(), conversation.GetTargetBeaconID(), target.SessionID, target.BeaconID)
}

func (m *aiModel) applyTargetSelectionOption(option aiTargetSelectionOption) {
	sessionID, beaconID := normalizedAITargetIDs(option.target.SessionID, option.target.BeaconID)
	m.ctx.target = option.target
	m.ctx.target.SessionID = sessionID
	m.ctx.target.BeaconID = beaconID
	if m.con != nil && m.con.ActiveTarget != nil {
		m.con.ActiveTarget.Set(option.session, option.beacon)
	}
}

func normalizedConversationOperatorName(name string) string {
	name = strings.TrimSpace(name)
	switch strings.ToLower(name) {
	case "", "unknown", "<unknown>", "<unknown operator>", "<disconnected>":
		return ""
	default:
		return name
	}
}

func sameConversationMessage(left *clientpb.AIConversationMessage, right *clientpb.AIConversationMessage) bool {
	if left == nil || right == nil {
		return false
	}
	leftID := strings.TrimSpace(left.GetID())
	rightID := strings.TrimSpace(right.GetID())
	if leftID != "" && rightID != "" && leftID == rightID {
		return true
	}
	leftItemID := strings.TrimSpace(left.GetItemID())
	rightItemID := strings.TrimSpace(right.GetItemID())
	return leftItemID != "" && rightItemID != "" && leftItemID == rightItemID
}

func sortConversationMessages(messages []*clientpb.AIConversationMessage) {
	sort.SliceStable(messages, func(i, j int) bool {
		left := messages[i]
		right := messages[j]
		if left == nil || right == nil {
			return left != nil
		}
		if left.GetSequence() != right.GetSequence() {
			return left.GetSequence() < right.GetSequence()
		}
		leftCreated := maxInt64(left.GetCreatedAt(), left.GetUpdatedAt())
		rightCreated := maxInt64(right.GetCreatedAt(), right.GetUpdatedAt())
		if leftCreated != rightCreated {
			return leftCreated < rightCreated
		}
		return strings.TrimSpace(left.GetID()) < strings.TrimSpace(right.GetID())
	})
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
	dst.ThinkingLevel = normalizeAIThinkingLevel(src.GetThinkingLevel())
	if title := strings.TrimSpace(src.GetTitle()); title != "" {
		dst.Title = title
	}
	if summary := strings.TrimSpace(src.GetSummary()); summary != "" {
		dst.Summary = summary
	}
	if systemPrompt := strings.TrimSpace(src.GetSystemPrompt()); systemPrompt != "" {
		dst.SystemPrompt = systemPrompt
	}
	dst.ActiveTurnID = strings.TrimSpace(src.GetActiveTurnID())
	dst.TurnState = src.GetTurnState()
	if targetSessionID := strings.TrimSpace(src.GetTargetSessionID()); targetSessionID != "" {
		dst.TargetSessionID = targetSessionID
	}
	if targetBeaconID := strings.TrimSpace(src.GetTargetBeaconID()); targetBeaconID != "" {
		dst.TargetBeaconID = targetBeaconID
	}
}

func conversationAwaitingResponse(conversation *clientpb.AIConversation) bool {
	if conversation == nil {
		return false
	}
	if conversation.GetTurnState() == clientpb.AIConversationTurnState_AI_TURN_STATE_IN_PROGRESS {
		return true
	}
	if conversation.GetTurnState() == clientpb.AIConversationTurnState_AI_TURN_STATE_FAILED {
		return false
	}
	message := lastContextConversationMessage(conversation)
	if message == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(message.GetRole()), "user")
}

func lastContextConversationMessage(conversation *clientpb.AIConversation) *clientpb.AIConversationMessage {
	if conversation == nil {
		return nil
	}
	messages := conversation.GetMessages()
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i] == nil {
			continue
		}
		if !clientpbutil.AIConversationMessageIncludesContext(messages[i]) {
			continue
		}
		if messages[i].GetKind() != clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT {
			continue
		}
		return messages[i]
	}
	return nil
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
		content = strings.Trim(formatConversationMessageMarkdownContent(message), "\n")
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

func formatConversationMessageMarkdownContent(message *clientpb.AIConversationMessage) string {
	if message == nil {
		return ""
	}
	switch message.GetKind() {
	case clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_TOOL_CALL:
		parts := []string{}
		if args := strings.TrimSpace(formatStructuredBlock(message.GetToolArguments())); args != "" {
			parts = append(parts, "Arguments:\n"+args)
		}
		if result := strings.TrimSpace(formatStructuredBlock(message.GetToolResult())); result != "" {
			parts = append(parts, "Result:\n"+result)
		}
		if errText := strings.TrimSpace(message.GetErrorText()); errText != "" {
			parts = append(parts, "Error:\n"+errText)
		}
		return joinTextBlocks(parts)
	default:
		return message.GetContent()
	}
}

func messageBlockLabel(conversation *clientpb.AIConversation, message *clientpb.AIConversationMessage) string {
	if message != nil {
		switch message.GetKind() {
		case clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_REASONING:
			return "Reasoning"
		case clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_TOOL_CALL:
			if toolName := strings.TrimSpace(message.GetToolName()); toolName != "" {
				return "Tool: " + toolName
			}
			return "Tool Call"
		}
	}

	role := ""
	if message != nil {
		role = strings.ToLower(strings.TrimSpace(message.GetRole()))
	}

	switch role {
	case "user":
		if message != nil {
			if operatorName := normalizedConversationOperatorName(message.GetOperatorName()); operatorName != "" {
				return operatorName
			}
		}
		if conversation != nil {
			if operatorName := normalizedConversationOperatorName(conversation.GetOperatorName()); operatorName != "" {
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

func messageBlockRole(message *clientpb.AIConversationMessage) string {
	if message == nil {
		return ""
	}
	switch message.GetKind() {
	case clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_REASONING:
		return "reasoning"
	case clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_TOOL_CALL:
		return "tool"
	default:
		return strings.ToLower(strings.TrimSpace(message.GetRole()))
	}
}

func conversationMessageStateLabel(message *clientpb.AIConversationMessage) string {
	if message == nil {
		return ""
	}
	switch message.GetState() {
	case clientpb.AIConversationMessageState_AI_MESSAGE_STATE_IN_PROGRESS:
		return "in progress"
	case clientpb.AIConversationMessageState_AI_MESSAGE_STATE_FAILED:
		return "failed"
	default:
		if message.GetKind() == clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_TOOL_CALL ||
			message.GetKind() == clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_REASONING {
			return "completed"
		}
		return ""
	}
}

func joinTextBlocks(blocks []string) string {
	filtered := make([]string, 0, len(blocks))
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		filtered = append(filtered, block)
	}
	return strings.Join(filtered, "\n\n")
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

type transcriptSpeakerStyles struct {
	border lipgloss.Style
	label  lipgloss.Style
	meta   lipgloss.Style
}

func renderConversationTranscriptLines(width int, conversation *clientpb.AIConversation) []string {
	return transcriptContentStyledLines(renderConversationTranscript(width, conversation))
}

func renderConversationTranscript(width int, conversation *clientpb.AIConversation) []aiTranscriptContentLine {
	if width <= 0 {
		return nil
	}

	if conversation == nil {
		return renderTranscriptSectionBlockContent(width, "Conversation", "system", nil, wrapText("Create a new conversation or submit a prompt to start one.", maxInt(1, width)))
	}

	lines := []aiTranscriptContentLine{}

	if summary := strings.TrimSpace(conversation.GetSummary()); summary != "" {
		lines = appendTranscriptContentBlock(lines, renderTranscriptSectionBlockContent(width, "Summary", "system", nil, wrapText(summary, maxInt(1, width))))
	}
	if systemPrompt := strings.TrimSpace(conversation.GetSystemPrompt()); systemPrompt != "" {
		lines = appendTranscriptContentBlock(lines, renderTranscriptSectionBlockContent(width, "System Prompt", "system", nil, wrapText(systemPrompt, maxInt(1, width))))
	}

	messageCount := 0
	for _, message := range conversation.GetMessages() {
		if message == nil {
			continue
		}
		messageCount++
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
		lines = appendTranscriptContentBlock(lines, renderTranscriptBoxBlockContent(
			width,
			messageBlockLabel(conversation, message),
			messageBlockRole(message),
			meta,
			renderConversationMessageBodyLines(maxInt(1, width-4), message),
			true,
		))
	}

	if messageCount == 0 {
		lines = appendTranscriptContentBlock(lines, renderTranscriptSectionBlockContent(width, "Conversation", "system", nil, wrapText("No messages yet. Type a prompt below to start this thread.", maxInt(1, width))))
	}

	return lines
}

func renderConversationMessageBlockLines(width int, conversation *clientpb.AIConversation, message *clientpb.AIConversationMessage) []string {
	meta := []string{}
	if ts := formatUnix(message.GetCreatedAt()); ts != "<unknown>" {
		meta = append(meta, ts)
	}
	if state := conversationMessageStateLabel(message); state != "" {
		meta = append(meta, state)
	}
	if provider := strings.TrimSpace(message.GetProvider()); provider != "" {
		meta = append(meta, provider)
	}
	if model := strings.TrimSpace(message.GetModel()); model != "" {
		meta = append(meta, model)
	}

	return renderTranscriptBoxBlock(
		width,
		messageBlockLabel(conversation, message),
		messageBlockRole(message),
		meta,
		renderConversationMessageBodyLines(maxInt(1, width-4), message),
	)
}

func renderConversationMessageBodyLines(width int, message *clientpb.AIConversationMessage) []string {
	if message == nil {
		return nil
	}

	switch message.GetKind() {
	case clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_REASONING:
		return renderTranscriptPlainBodyLines(width, message.GetContent(), "Reasoning was used, but the provider did not return a visible summary.")
	case clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_TOOL_CALL:
		return renderToolCallBodyLines(width, message)
	default:
		return renderTranscriptMessageBodyLines(width, message.GetContent())
	}
}

func renderTranscriptMessageBodyLines(width int, content string) []string {
	content = strings.Trim(content, "\n")
	if strings.TrimSpace(content) == "" {
		return []string{"Empty message."}
	}

	rendered, err := renderMarkdownWithGlow(maxInt(8, width), content)
	if err != nil {
		return wrapText(content, width)
	}

	rendered = strings.Trim(rendered, "\n")
	if strings.TrimSpace(rendered) == "" {
		return []string{"Empty message."}
	}
	return trimRenderedLines(strings.Split(rendered, "\n"))
}

func renderTranscriptPlainBodyLines(width int, content string, empty string) []string {
	content = strings.Trim(content, "\n")
	if strings.TrimSpace(content) == "" {
		return []string{empty}
	}

	rawLines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, rawLine := range rawLines {
		if rawLine == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, wrapText(rawLine, maxInt(1, width))...)
	}
	if len(lines) == 0 {
		return []string{empty}
	}
	return lines
}

func renderToolCallBodyLines(width int, message *clientpb.AIConversationMessage) []string {
	if message == nil {
		return []string{"Tool call details unavailable."}
	}

	lines := []string{}
	addSection := func(title string, body string, fallback string) {
		body = strings.TrimSpace(body)
		if body == "" && fallback == "" {
			return
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, strings.ToUpper(title))
		lines = append(lines, renderTranscriptPlainBodyLines(width, formatStructuredBlock(body), fallback)...)
	}

	addSection("Arguments", message.GetToolArguments(), "No arguments.")
	if errText := strings.TrimSpace(message.GetErrorText()); errText != "" {
		addSection("Error", errText, "")
	}
	if result := strings.TrimSpace(message.GetToolResult()); result != "" {
		addSection("Result", result, "")
	} else if message.GetState() == clientpb.AIConversationMessageState_AI_MESSAGE_STATE_IN_PROGRESS {
		addSection("Result", "", "Waiting for tool output...")
	}
	if len(lines) == 0 {
		return []string{"Tool call details unavailable."}
	}
	return lines
}

func formatStructuredBlock(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(raw), "", "  "); err == nil {
		return pretty.String()
	}
	return raw
}

func renderTranscriptBoxBlock(width int, label, role string, meta []string, contentLines []string) []string {
	return transcriptContentStyledLines(renderTranscriptBoxBlockContent(width, label, role, meta, contentLines, true))
}

func renderTranscriptBoxBlockContent(width int, label, role string, meta []string, contentLines []string, selectable bool) []aiTranscriptContentLine {
	if width <= 0 {
		return nil
	}

	styles := transcriptSpeakerStyle(label, role)
	label = fallback(label, "Message")
	if len(contentLines) == 0 {
		contentLines = []string{""}
	}

	pieces := []string{styles.label.Render(label)}
	if metaText := strings.Join(meta, " | "); strings.TrimSpace(metaText) != "" {
		pieces = append(pieces, styles.meta.Render(metaText))
	}

	header := fitStyledPieces(maxInt(1, width-3), pieces)
	lines := []aiTranscriptContentLine{{styled: renderTranscriptBoxTopLine(width, styles.border, header)}}
	for _, line := range contentLines {
		text := ansi.Strip(line)
		text = strings.TrimRight(text, " ")
		prefix, paddedContent, contentWidth := renderTranscriptBoxContentParts(width, styles.border, line)
		rendered := aiTranscriptContentLine{styled: prefix + paddedContent}
		if selectable && ansi.StringWidth(text) > 0 {
			rendered.selectableStart = ansi.StringWidth(prefix)
			rendered.selectableText = text
			rendered.selectablePrefix = prefix
			rendered.selectableAreaWidth = contentWidth
		}
		lines = append(lines, rendered)
	}
	lines = append(lines, aiTranscriptContentLine{styled: renderTranscriptBoxBottomLine(width, styles.border)})
	return lines
}

func renderTranscriptSectionBlock(width int, label, role string, meta []string, contentLines []string) []string {
	return transcriptContentStyledLines(renderTranscriptSectionBlockContent(width, label, role, meta, contentLines))
}

func renderTranscriptSectionBlockContent(width int, label, role string, meta []string, contentLines []string) []aiTranscriptContentLine {
	if width <= 0 {
		return nil
	}

	styles := transcriptSpeakerStyle(label, role)
	label = fallback(label, "Message")
	if len(contentLines) == 0 {
		contentLines = []string{""}
	}

	pieces := []string{styles.label.Render(label)}
	if metaText := strings.Join(meta, " | "); strings.TrimSpace(metaText) != "" {
		pieces = append(pieces, styles.meta.Render(metaText))
	}

	lines := []aiTranscriptContentLine{{styled: padANSIRight(fitStyledPieces(width, pieces), width)}}
	for _, line := range contentLines {
		lines = append(lines, aiTranscriptContentLine{styled: padANSIRight(line, width)})
	}
	return lines
}

func transcriptSpeakerStyle(label, role string) transcriptSpeakerStyles {
	label = strings.TrimSpace(label)
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "user" {
		accent := clienttheme.Primary()
		return transcriptSpeakerStyles{
			border: lipgloss.NewStyle().Foreground(accent),
			label: lipgloss.NewStyle().
				Bold(true).
				Foreground(clienttheme.DefaultMod(900)).
				Background(accent).
				Padding(0, 1),
			meta: lipgloss.NewStyle().Foreground(accent),
		}
	}
	if role == "tool" || role == "reasoning" {
		accent := clienttheme.DefaultMod(600)
		return transcriptSpeakerStyles{
			border: lipgloss.NewStyle().Foreground(accent),
			label: lipgloss.NewStyle().
				Bold(true).
				Foreground(clienttheme.DefaultMod(50)).
				Background(clienttheme.DefaultMod(700)).
				Padding(0, 1),
			meta: lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(500)),
		}
	}
	if role == "system" && strings.EqualFold(label, "Conversation") {
		accent := clienttheme.PrimaryMod(200)
		return transcriptSpeakerStyles{
			border: lipgloss.NewStyle().Foreground(accent),
			label: lipgloss.NewStyle().
				Bold(true).
				Foreground(clienttheme.DefaultMod(900)).
				Background(accent).
				Padding(0, 1),
			meta: lipgloss.NewStyle().Foreground(accent),
		}
	}

	accent := transcriptSpeakerPalette()[transcriptSpeakerPaletteIndex(label, role)]
	borderColor := accent
	headerColor := accent
	if role == "user" {
		borderColor = clienttheme.Primary()
		headerColor = clienttheme.Primary()
	}
	return transcriptSpeakerStyles{
		border: lipgloss.NewStyle().Foreground(borderColor),
		label: lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.DefaultMod(900)).
			Background(headerColor).
			Padding(0, 1),
		meta: lipgloss.NewStyle().Foreground(headerColor),
	}
}

func renderTranscriptBoxTopLine(width int, border lipgloss.Style, header string) string {
	if width <= 0 {
		return ""
	}
	if width == 1 {
		return border.Render("╭")
	}
	if width == 2 {
		return border.Render("╭─")
	}

	line := border.Render("╭─")
	if strings.TrimSpace(ansi.Strip(header)) != "" {
		line += " " + header
	}
	return padANSIRight(line, width)
}

func renderTranscriptBoxBottomLine(width int, border lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	if width == 1 {
		return border.Render("╰")
	}
	return padANSIRight(border.Render("╰─"), width)
}

func renderTranscriptBoxContentLine(width int, border lipgloss.Style, content string) string {
	prefix, paddedContent, _ := renderTranscriptBoxContentParts(width, border, content)
	return prefix + paddedContent
}

func renderTranscriptBoxContentParts(width int, border lipgloss.Style, content string) (string, string, int) {
	if width <= 0 {
		return "", "", 0
	}
	if width == 1 {
		return border.Render("│"), "", 0
	}

	contentWidth := width - 2
	return border.Render("│ "), padANSIRight(content, contentWidth), contentWidth
}

func transcriptSpeakerPalette() []color.Color {
	return []color.Color{
		clienttheme.Primary(),
		clienttheme.Secondary(),
		clienttheme.Warning(),
		clienttheme.Danger(),
		clienttheme.DefaultMod(700),
		clienttheme.PrimaryMod(700),
		clienttheme.SecondaryMod(700),
	}
}

func transcriptSpeakerPaletteIndex(label, role string) int {
	role = strings.ToLower(strings.TrimSpace(role))
	switch role {
	case "assistant":
		return 1
	case "system":
		return 4
	}

	label = strings.TrimSpace(strings.ToLower(label))
	if label == "" {
		label = role
	}

	hash := fnv.New32a()
	_, _ = hash.Write([]byte(label))
	palette := transcriptSpeakerPalette()
	return int(hash.Sum32() % uint32(len(palette)))
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

func transcriptContentStyledLines(content []aiTranscriptContentLine) []string {
	lines := make([]string, 0, len(content))
	for _, line := range content {
		lines = append(lines, line.styled)
	}
	return lines
}

func appendTranscriptContentBlock(content []aiTranscriptContentLine, block []aiTranscriptContentLine) []aiTranscriptContentLine {
	if len(block) == 0 {
		return content
	}
	if len(content) > 0 {
		content = append(content, aiTranscriptContentLine{styled: ""})
	}
	return append(content, block...)
}

func (m *aiModel) clearTranscriptSelection() {
	m.transcriptSelection = nil
}

func (m *aiModel) beginTranscriptSelection(x, y int) {
	point, ok := m.transcriptSelectionPointAt(x, y, false)
	if !ok {
		m.clearTranscriptSelection()
		return
	}
	m.transcriptSelection = &aiTranscriptSelection{
		anchor:   point,
		active:   point,
		dragging: true,
	}
}

func (m *aiModel) updateTranscriptSelection(x, y int, clamp bool) {
	if m.transcriptSelection == nil {
		return
	}
	point, ok := m.transcriptSelectionPointAt(x, y, clamp)
	if !ok {
		return
	}
	m.transcriptSelection.active = point
}

func (m *aiModel) transcriptSelectionPointAt(x, y int, clamp bool) (aiTranscriptSelectionPoint, bool) {
	rect := m.currentPaneRects().transcript
	if !rect.contains(x, y) {
		return aiTranscriptSelectionPoint{}, false
	}

	contentWidth, viewportHeight := m.currentTranscriptViewportSize()
	if contentWidth <= 0 || viewportHeight <= 0 {
		return aiTranscriptSelectionPoint{}, false
	}

	innerX := x - rect.x - 2
	innerY := y - rect.y - 1
	if innerX < 0 || innerY < m.transcriptHeaderLineCount() {
		return aiTranscriptSelectionPoint{}, false
	}

	bodyY := innerY - m.transcriptHeaderLineCount()
	if bodyY < 0 || bodyY >= viewportHeight {
		return aiTranscriptSelectionPoint{}, false
	}

	content := m.renderTranscriptDisplayContent(contentWidth)
	scroll, end := m.currentTranscriptScrollRange(len(content), viewportHeight)
	lineIndex := scroll + bodyY
	if lineIndex < scroll || lineIndex >= end || lineIndex >= len(content) {
		return aiTranscriptSelectionPoint{}, false
	}

	line := content[lineIndex]
	if !line.hasSelectableText() {
		return aiTranscriptSelectionPoint{}, false
	}

	col := innerX - line.selectableStart
	if col < 0 {
		if !clamp {
			return aiTranscriptSelectionPoint{}, false
		}
		col = 0
	}

	width := line.selectableWidth()
	if width <= 0 {
		return aiTranscriptSelectionPoint{}, false
	}
	if col >= width {
		if !clamp {
			return aiTranscriptSelectionPoint{}, false
		}
		col = width - 1
	}

	return aiTranscriptSelectionPoint{line: lineIndex, col: col}, true
}

func (m *aiModel) renderVisibleTranscriptLine(content []aiTranscriptContentLine, idx int) string {
	if idx < 0 || idx >= len(content) {
		return ""
	}

	line := content[idx]
	if m.transcriptSelection == nil || !line.hasSelectableText() {
		return line.styled
	}

	start, end, ok := m.transcriptSelectionRangeForLine(idx, line.selectableWidth())
	if !ok {
		return line.styled
	}
	if line.selectablePrefix != "" && line.selectableAreaWidth > 0 {
		return line.selectablePrefix + padANSIRight(
			renderSelectedTranscriptText(line.selectableText, start, end, m.styles.selection),
			line.selectableAreaWidth,
		)
	}
	return highlightANSIRange(line.styled, line.selectableStart+start, line.selectableStart+end, m.styles.selection)
}

func (m *aiModel) transcriptSelectionRangeForLine(lineIdx, lineWidth int) (int, int, bool) {
	if m.transcriptSelection == nil || lineWidth <= 0 {
		return 0, 0, false
	}

	start, end := normalizeTranscriptSelection(m.transcriptSelection.anchor, m.transcriptSelection.active)
	if lineIdx < start.line || lineIdx > end.line {
		return 0, 0, false
	}

	lineStart := 0
	if lineIdx == start.line {
		lineStart = clampInt(start.col, 0, lineWidth-1)
	}
	lineEnd := lineWidth
	if lineIdx == end.line {
		lineEnd = clampInt(end.col+1, 1, lineWidth)
	}
	if lineStart >= lineEnd {
		return 0, 0, false
	}
	return lineStart, lineEnd, true
}

func (m *aiModel) selectedTranscriptText() string {
	if m.transcriptSelection == nil {
		return ""
	}

	contentWidth, _ := m.currentTranscriptViewportSize()
	if contentWidth <= 0 {
		return ""
	}
	content := m.renderTranscriptDisplayContent(contentWidth)
	if len(content) == 0 {
		return ""
	}

	start, end := normalizeTranscriptSelection(m.transcriptSelection.anchor, m.transcriptSelection.active)
	start.line = clampInt(start.line, 0, len(content)-1)
	end.line = clampInt(end.line, 0, len(content)-1)

	lines := make([]string, 0, end.line-start.line+1)
	for idx := start.line; idx <= end.line; idx++ {
		line := content[idx]
		if !line.hasSelectableText() {
			continue
		}
		lineStart, lineEnd, ok := m.transcriptSelectionRangeForLine(idx, line.selectableWidth())
		if !ok {
			continue
		}
		lines = append(lines, ansi.Cut(line.selectableText, lineStart, lineEnd))
	}
	return strings.Join(lines, "\n")
}

func (m *aiModel) isCollapsedTranscriptSelection() bool {
	if m.transcriptSelection == nil {
		return true
	}
	start, end := normalizeTranscriptSelection(m.transcriptSelection.anchor, m.transcriptSelection.active)
	return start.line == end.line && start.col == end.col
}

func normalizeTranscriptSelection(a, b aiTranscriptSelectionPoint) (aiTranscriptSelectionPoint, aiTranscriptSelectionPoint) {
	if a.line > b.line || (a.line == b.line && a.col > b.col) {
		return b, a
	}
	return a, b
}

func highlightANSIRange(line string, start, end int, style lipgloss.Style) string {
	if start < 0 {
		start = 0
	}
	if end <= start {
		return line
	}
	prefix := ansi.Cut(line, 0, start)
	selected := ansi.Cut(line, start, end)
	if ansi.StringWidth(selected) == 0 {
		return line
	}
	suffix := ansi.Cut(line, end, ansi.StringWidth(line))
	return prefix + style.Render(selected) + suffix
}

func renderSelectedTranscriptText(text string, start, end int, style lipgloss.Style) string {
	if start < 0 {
		start = 0
	}
	if end <= start {
		return text
	}

	width := ansi.StringWidth(text)
	if width == 0 {
		return text
	}

	if start >= width {
		return text
	}
	end = minInt(end, width)
	if end <= start {
		return text
	}

	prefix := ansi.Cut(text, 0, start)
	selected := ansi.Cut(text, start, end)
	suffix := ansi.Cut(text, end, width)
	if ansi.StringWidth(selected) == 0 {
		return text
	}
	return prefix + style.Render(selected) + suffix
}

func copyTranscriptSelectionToClipboard(text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	_ = clipboard.WriteAll(text)
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

func overlayContent(base string, overlay string, left, top, width int) string {
	if strings.TrimSpace(overlay) == "" {
		return base
	}

	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")
	requiredHeight := maxInt(len(baseLines), top+len(overlayLines))
	for len(baseLines) < requiredHeight {
		baseLines = append(baseLines, "")
	}

	for i, overlayLine := range overlayLines {
		row := top + i
		if row < 0 || row >= len(baseLines) {
			continue
		}
		baseLines[row] = overlayLineAt(baseLines[row], overlayLine, left, width)
	}

	return strings.Join(baseLines, "\n")
}

func overlayLineAt(baseLine string, overlayLine string, left, width int) string {
	if left < 0 {
		left = 0
	}
	if width <= 0 {
		width = maxInt(
			ansi.StringWidth(baseLine),
			left+ansi.StringWidth(overlayLine),
		)
	}

	prefix := ansi.Cut(baseLine, 0, minInt(left, width))
	prefixWidth := ansi.StringWidth(prefix)
	if prefixWidth < left {
		prefix += strings.Repeat(" ", left-prefixWidth)
	}

	suffixStart := minInt(width, left+ansi.StringWidth(overlayLine))
	suffix := ansi.Cut(baseLine, suffixStart, width)
	visibleWidth := left + ansi.StringWidth(overlayLine) + ansi.StringWidth(suffix)
	if visibleWidth < width {
		suffix += strings.Repeat(" ", width-visibleWidth)
	}

	return prefix + overlayLine + suffix
}

func clampANSIBlock(text string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	lines := strings.Split(text, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		lines[i] = padANSIRight(line, width)
	}
	return strings.Join(lines, "\n")
}

func trimRenderedLines(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(ansi.Strip(lines[start])) == "" {
		start++
	}

	end := len(lines)
	for end > start && strings.TrimSpace(ansi.Strip(lines[end-1])) == "" {
		end--
	}

	trimmed := append([]string(nil), lines[start:end]...)
	if len(trimmed) == 0 {
		return []string{""}
	}
	return trimmed
}

func padANSIRight(text string, width int) string {
	if width <= 0 {
		return ""
	}

	clipped := ansi.Cut(text, 0, width)
	if padding := width - ansi.StringWidth(clipped); padding > 0 {
		clipped += strings.Repeat(" ", padding)
	}
	return clipped
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

func clampFloat64(value, low, high float64) float64 {
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
