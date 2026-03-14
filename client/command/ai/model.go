package ai

import (
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	clienttheme "github.com/bishopfox/sliver/client/theme"
	"golang.org/x/term"
)

const (
	aiMinWidth  = 72
	aiMinHeight = 18

	aiPaneHorizontalChrome = 4
	aiPaneVerticalChrome   = 2
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
	roleSystem       lipgloss.Style
	roleAssistant    lipgloss.Style
	roleUser         lipgloss.Style
	heading          lipgloss.Style
	cursor           lipgloss.Style
	inputText        lipgloss.Style
	inputPlaceholder lipgloss.Style
	warning          lipgloss.Style
}

type aiModel struct {
	width                int
	height               int
	focus                aiFocus
	ctx                  aiContext
	selectedConversation int
	input                []rune
	cursor               int
	status               string
	styles               aiStyles
}

func newAIModel(ctx aiContext) *aiModel {
	return &aiModel{
		focus:  aiFocusSidebar,
		ctx:    ctx,
		status: ctx.status,
		styles: newAIStyles(),
	}
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
		roleSystem: lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.Secondary()),
		roleAssistant: lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.Primary()),
		roleUser: lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.Warning()),
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
	return nil
}

func (m *aiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
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
			m.moveSelection(-1)
		}
		return m, nil
	case tea.KeyDown:
		if m.focus == aiFocusSidebar {
			m.moveSelection(1)
		}
		return m, nil
	case tea.KeyEnter:
		if m.focus == aiFocusSidebar {
			m.status = "Selected placeholder thread: " + m.currentConversation().Title + "."
		}
		return m, nil
	}

	switch key.Text {
	case "q":
		return m, tea.Quit
	case "j":
		if m.focus == aiFocusSidebar {
			m.moveSelection(1)
		}
	case "k":
		if m.focus == aiFocusSidebar {
			m.moveSelection(-1)
		}
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
		m.capturePlaceholderPrompt()
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
			insert := []rune(key.Text)
			m.input = append(m.input[:m.cursor], append(insert, m.input[m.cursor:]...)...)
			m.cursor += len(insert)
		}
	}
	return m, nil
}

func (m *aiModel) View() tea.View {
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

	return aiView(lipgloss.JoinVertical(lipgloss.Left, header, body, composer, footer))
}

func (m *aiModel) renderTooSmall() string {
	lines := []string{
		m.styles.badge.Render("SLIVER AI"),
		"",
		m.styles.warning.Render("Terminal too small for the AI layout preview."),
		m.styles.subtleText.Render("Resize to at least 72x18 to see the full sidebar/chat/context shell."),
	}
	return strings.Join(lines, "\n")
}

func (m *aiModel) renderHeader(height int) string {
	pieces := []string{
		m.styles.badge.Render("SLIVER AI"),
		m.styles.chip.Render("layout preview"),
		m.styles.chipMuted.Render(m.ctx.target.Label),
		m.styles.chipMuted.Render(m.ctx.connection.State),
		m.styles.chipMuted.Render(m.layoutName()),
	}
	row := fitStyledPieces(m.width, pieces)
	row = lipgloss.NewStyle().Width(m.width).Render(row)
	if height == 1 {
		return row
	}

	subtitle := "Bubble Tea shell for future target-aware chat, tool traces, and operator context."
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
		detailsWidth := clampInt(m.width/4, 28, 32)
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
		detailsHeight := clampInt(height/3, 5, 8)
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
		detailsHeight := clampInt(height/4, 5, 7)
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
	bodyHeight := maxInt(1, innerHeight-1)

	for i, conversation := range m.ctx.conversations {
		line := truncateText(conversation.Title, maxInt(1, innerWidth-2))
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
			return m.renderPane(width, height, aiFocusSidebar, lines)
		}
	}

	quickPrompts := []string{
		"Quick prompts:",
		"- summarize target",
		"- plan next step",
		"- explain latest result",
	}
	if remaining := bodyHeight - (len(lines) - 1); remaining > 2 {
		lines = append(lines, "")
		for _, prompt := range quickPrompts {
			if len(lines)-1 >= bodyHeight {
				break
			}
			lines = append(lines, m.styles.subtleText.Width(innerWidth).Render(truncateText(prompt, innerWidth)))
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

	conversation := m.currentConversation()
	header := []string{
		m.styles.heading.Width(innerWidth).Render(truncateText(conversation.Title, innerWidth)),
		m.styles.subtleText.Width(innerWidth).Render(truncateText(conversation.Subtitle, innerWidth)),
	}
	lines = append(lines, header...)

	bodyHeight := maxInt(1, innerHeight-len(lines))
	messageLines := make([]string, 0, bodyHeight)
	for _, message := range conversation.Messages {
		messageLines = append(messageLines, m.renderMessageLines(message, innerWidth)...)
	}
	messageLines = tailLines(messageLines, bodyHeight)
	lines = append(lines, messageLines...)

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

	lines = append(lines, m.styles.heading.Render("Planned"))
	for _, line := range m.ctx.planned {
		wrapped := wrapText("- "+line, innerWidth)
		for _, wrappedLine := range wrapped {
			lines = append(lines, m.styles.subtleText.Width(innerWidth).Render(truncateText(wrappedLine, innerWidth)))
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
		help := "tab focus  enter placeholder-send  ctrl+u clear  esc blur  q quit"
		lines = append(lines, m.styles.subtleText.Width(innerWidth).Render(truncateText(help, innerWidth)))
	}

	return m.renderPane(m.width, height, aiFocusComposer, headLines(lines, innerHeight))
}

func (m *aiModel) renderFooter() string {
	pieces := []string{
		m.styles.chipMuted.Render("focus: " + m.focus.String()),
		m.styles.footerHint.Render("tab: next pane"),
		m.styles.footerHint.Render("j/k: move"),
		m.styles.footerHint.Render("enter: select/send"),
		m.styles.footerHint.Render("esc: back"),
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

func (m *aiModel) renderMessageLines(message aiMessage, width int) []string {
	label, style := m.messageLabel(message.Role)
	bodyWidth := maxInt(10, width-5)
	wrapped := wrapText(message.Body, bodyWidth)
	lines := make([]string, 0, len(wrapped)+1)
	for i, line := range wrapped {
		prefix := "    "
		if i == 0 {
			prefix = style.Render(label)
		}
		lines = append(lines, prefix+" "+line)
	}
	lines = append(lines, "")
	return lines
}

func (m *aiModel) messageLabel(role string) (string, lipgloss.Style) {
	switch role {
	case "system":
		return "SYS ", m.styles.roleSystem
	case "user":
		return "YOU ", m.styles.roleUser
	default:
		return "AI  ", m.styles.roleAssistant
	}
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

func (m *aiModel) moveSelection(delta int) {
	if len(m.ctx.conversations) == 0 {
		return
	}
	m.selectedConversation = clampInt(m.selectedConversation+delta, 0, len(m.ctx.conversations)-1)
	m.status = "Viewing placeholder thread: " + m.currentConversation().Title + "."
}

func (m *aiModel) capturePlaceholderPrompt() {
	prompt := strings.TrimSpace(string(m.input))
	if prompt == "" {
		m.status = "Type a prompt first. Submission is still placeholder-only for now."
		return
	}

	conversation := m.currentConversation()
	conversation.Messages = append(conversation.Messages,
		aiMessage{Role: "user", Body: prompt},
		aiMessage{Role: "assistant", Body: "Placeholder response: this prompt was captured locally so we can exercise the transcript layout before wiring the real AI backend."},
	)
	m.input = nil
	m.cursor = 0
	m.status = "Captured placeholder prompt in " + conversation.Title + "."
}

func (m *aiModel) currentConversation() *aiConversation {
	if len(m.ctx.conversations) == 0 {
		m.ctx.conversations = []aiConversation{{Title: "Untitled", Subtitle: "", Messages: nil}}
		m.selectedConversation = 0
	}
	if m.selectedConversation < 0 || m.selectedConversation >= len(m.ctx.conversations) {
		m.selectedConversation = 0
	}
	return &m.ctx.conversations[m.selectedConversation]
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
