package docs

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	glamourstyles "charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	clienttheme "github.com/bishopfox/sliver/client/theme"
	embeddeddocs "github.com/bishopfox/sliver/docs"
	"golang.org/x/term"
)

const (
	docsMinWidth           = 72
	docsMinHeight          = 18
	docsFooterHeight       = 1
	docsPaneGap            = 1
	docsPaneHeaderLines    = 1
	docsNarrowWidth        = 96
	docsBrowserMinWidth    = 28
	docsBrowserMaxWidth    = 40
	docsWindowPollInterval = 100 * time.Millisecond
)

var (
	markdownTrimChars    = regexp.MustCompile(`^[#>\-\*\+\d\.\)\s` + "`" + `]+`)
	yamlFrontmatterBound = regexp.MustCompile(`(?m)^---\r?\n(\s*\r?\n)?`)
)

type docsFocus int

const (
	docsFocusBrowser docsFocus = iota
	docsFocusViewer
)

type docEntry struct {
	Name        string
	Content     string
	Description string
}

type docsItem struct {
	entry docEntry
}

func (d docsItem) Title() string       { return d.entry.Name }
func (d docsItem) Description() string { return d.entry.Description }
func (d docsItem) FilterValue() string { return d.entry.Name + " " + d.entry.Description }

type docsStyles struct {
	app         lipgloss.Style
	pane        lipgloss.Style
	paneFocused lipgloss.Style
	header      lipgloss.Style
	headerMeta  lipgloss.Style
	footer      lipgloss.Style
	footerMuted lipgloss.Style
	empty       lipgloss.Style
	error       lipgloss.Style
	minSize     lipgloss.Style
}

type docsWindowPollMsg struct {
	width  int
	height int
}

type docsModel struct {
	width          int
	height         int
	focus          docsFocus
	browser        list.Model
	viewer         viewport.Model
	entries        []docEntry
	entriesByName  map[string]docEntry
	currentDocName string
	renderCache    map[string]string
	styles         docsStyles
}

func loadDocEntries() ([]docEntry, error) {
	loaded, err := embeddeddocs.All()
	if err != nil {
		return nil, err
	}

	entries := make([]docEntry, 0, len(loaded.Docs))
	for _, doc := range loaded.Docs {
		entries = append(entries, docEntry{
			Name:        doc.Name,
			Content:     doc.Content,
			Description: summarizeMarkdown(doc.Content),
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}

func newDocsModel(entries []docEntry) *docsModel {
	items := make([]list.Item, 0, len(entries))
	entriesByName := make(map[string]docEntry, len(entries))
	for _, entry := range entries {
		items = append(items, docsItem{entry: entry})
		entriesByName[entry.Name] = entry
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(2)
	delegate.SetSpacing(0)
	delegate.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(clienttheme.DefaultMod(900)).
		Padding(0, 0, 0, 1)
	delegate.Styles.NormalDesc = lipgloss.NewStyle().
		Foreground(clienttheme.DefaultMod(500)).
		Padding(0, 0, 0, 1)
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(clienttheme.Primary()).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(clienttheme.Primary()).
		Padding(0, 0, 0, 1)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(clienttheme.DefaultMod(700)).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(clienttheme.Primary()).
		Padding(0, 0, 0, 1)
	delegate.Styles.DimmedTitle = lipgloss.NewStyle().
		Foreground(clienttheme.DefaultMod(400)).
		Padding(0, 0, 0, 1)
	delegate.Styles.DimmedDesc = lipgloss.NewStyle().
		Foreground(clienttheme.DefaultMod(300)).
		Padding(0, 0, 0, 1)
	delegate.Styles.FilterMatch = lipgloss.NewStyle().
		Underline(true).
		Foreground(clienttheme.Secondary())

	browser := list.New(items, delegate, 0, 0)
	browser.DisableQuitKeybindings()
	browser.Title = "Docs"
	browser.SetShowTitle(false)
	browser.SetShowHelp(false)
	browser.SetShowStatusBar(true)
	browser.SetShowPagination(true)
	browser.SetStatusBarItemName("doc", "docs")
	browser.FilterInput.Prompt = "Search: "
	browser.FilterInput.Placeholder = "Filter docs"
	browser.Styles.TitleBar = lipgloss.NewStyle()
	browser.Styles.Filter.Cursor.Color = clienttheme.Primary()
	browser.Styles.Filter.Focused.Prompt = lipgloss.NewStyle().Foreground(clienttheme.Primary())
	browser.Styles.Filter.Blurred.Prompt = lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(500))
	browser.Styles.StatusBar = lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(500))
	browser.Styles.StatusEmpty = lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(400))
	browser.Styles.StatusBarActiveFilter = lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(900))
	browser.Styles.StatusBarFilterCount = lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(400))
	browser.Styles.NoItems = lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(400))
	browser.Styles.ActivePaginationDot = lipgloss.NewStyle().Foreground(clienttheme.Primary()).SetString("•")
	browser.Styles.InactivePaginationDot = lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(300)).SetString("•")
	browser.Styles.ArabicPagination = lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(500))
	browser.Styles.DividerDot = lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(300)).SetString(" • ")
	browser.Styles.PaginationStyle = lipgloss.NewStyle()

	viewer := viewport.New()
	viewer.MouseWheelEnabled = true
	viewer.SoftWrap = false

	model := &docsModel{
		focus:         docsFocusBrowser,
		browser:       browser,
		viewer:        viewer,
		entries:       entries,
		entriesByName: entriesByName,
		renderCache:   make(map[string]string),
		styles:        newDocsStyles(),
	}

	if idx := preferredDocIndex(entries); idx >= 0 {
		model.browser.Select(idx)
		model.setCurrentDoc(entries[idx].Name, false)
	}

	return model
}

func newDocsStyles() docsStyles {
	return docsStyles{
		app: lipgloss.NewStyle().
			Padding(0, 0, 0, 0),
		pane: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clienttheme.DefaultMod(300)).
			Padding(0, 1),
		paneFocused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clienttheme.Primary()).
			Padding(0, 1),
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.DefaultMod(900)),
		headerMeta: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(500)),
		footer: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(700)),
		footerMuted: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(500)),
		empty: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(500)),
		error: lipgloss.NewStyle().
			Foreground(clienttheme.Danger()),
		minSize: lipgloss.NewStyle().
			Foreground(clienttheme.Warning()).
			Bold(true),
	}
}

func preferredDocIndex(entries []docEntry) int {
	for i, entry := range entries {
		if strings.EqualFold(entry.Name, "Getting Started") {
			return i
		}
	}
	if len(entries) == 0 {
		return -1
	}
	return 0
}

func summarizeMarkdown(markdown string) string {
	text := string(removeFrontmatter([]byte(markdown)))
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "```") {
			continue
		}
		line = markdownTrimChars.ReplaceAllString(line, "")
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return "Embedded Sliver documentation"
}

func renderMarkdownWithGlow(width int, markdown string) (string, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(glamourstyles.DarkStyle),
		glamour.WithWordWrap(maxInt(12, width)),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return "", err
	}

	rendered, err := renderer.Render(string(removeFrontmatter([]byte(markdown))))
	if err != nil {
		return "", err
	}

	lines := strings.Split(rendered, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n"), nil
}

func removeFrontmatter(content []byte) []byte {
	if frontmatterBoundaries := detectFrontmatter(content); frontmatterBoundaries[0] == 0 {
		return content[frontmatterBoundaries[1]:]
	}
	return content
}

func detectFrontmatter(content []byte) []int {
	if matches := yamlFrontmatterBound.FindAllIndex(content, 2); len(matches) > 1 {
		return []int{matches[0][0], matches[1][1]}
	}
	return []int{-1, -1}
}

func (m *docsModel) Init() tea.Cmd {
	return docsWindowPollCmd()
}

func (m *docsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.applyWindowSize(msg.Width, msg.Height)
		return m, nil

	case docsWindowPollMsg:
		return m, tea.Batch(m.windowSizeCmd(msg.width, msg.height), docsWindowPollCmd())

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.browser, cmd = m.browser.Update(msg)
	cmds = append(cmds, cmd)
	m.syncSelectionFromBrowser()

	m.viewer, cmd = m.viewer.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *docsModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	if m.focus == docsFocusBrowser && m.browser.FilterState() == list.Filtering {
		updated, cmd := m.browser.Update(msg)
		m.browser = updated
		m.syncSelectionFromBrowser()
		return m, cmd
	}

	switch msg.Code {
	case tea.KeyTab:
		if m.focus == docsFocusBrowser {
			m.focus = docsFocusViewer
		} else {
			m.focus = docsFocusBrowser
		}
		return m, nil

	case tea.KeyEsc:
		if m.focus == docsFocusViewer {
			m.focus = docsFocusBrowser
			return m, nil
		}
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "/":
		m.focus = docsFocusBrowser
		updated, cmd := m.browser.Update(msg)
		m.browser = updated
		m.syncSelectionFromBrowser()
		return m, cmd

	case "enter":
		if m.focus == docsFocusBrowser {
			m.focus = docsFocusViewer
			return m, nil
		}
	}

	if m.focus == docsFocusBrowser {
		updated, cmd := m.browser.Update(msg)
		m.browser = updated
		m.syncSelectionFromBrowser()
		return m, cmd
	}

	updated, cmd := m.viewer.Update(msg)
	m.viewer = updated
	return m, cmd
}

func (m *docsModel) View() tea.View {
	if m.width < docsMinWidth || m.height < docsMinHeight {
		view := tea.NewView(m.styles.minSize.Render(
			fmt.Sprintf("Docs viewer needs at least %dx%d. Current size: %dx%d", docsMinWidth, docsMinHeight, m.width, m.height),
		))
		view.AltScreen = true
		return view
	}

	bodyHeight := maxInt(1, m.height-docsFooterHeight)
	browserPane, viewerPane := m.renderPanes(bodyHeight)
	footer := m.styles.footer.Render(m.footerText())

	layout := lipgloss.JoinVertical(lipgloss.Left, lipgloss.JoinHorizontal(lipgloss.Top, browserPane, strings.Repeat(" ", docsPaneGap), viewerPane), footer)
	if m.isNarrow() {
		layout = lipgloss.JoinVertical(lipgloss.Left, browserPane, viewerPane, footer)
	}
	view := tea.NewView(m.styles.app.Render(layout))
	view.AltScreen = true
	return view
}

func (m *docsModel) renderPanes(bodyHeight int) (string, string) {
	if m.isNarrow() {
		browserHeight := clampInt(bodyHeight/3, 8, 12)
		viewerHeight := maxInt(5, bodyHeight-browserHeight-docsPaneGap)
		browserPane := m.renderBrowserPane(m.width, browserHeight)
		viewerPane := m.renderViewerPane(m.width, viewerHeight)
		return browserPane, viewerPane
	}

	browserWidth := clampInt(m.width/3, docsBrowserMinWidth, docsBrowserMaxWidth)
	viewerWidth := maxInt(docsMinWidth-browserWidth, m.width-browserWidth-docsPaneGap)

	return m.renderBrowserPane(browserWidth, bodyHeight), m.renderViewerPane(viewerWidth, bodyHeight)
}

func (m *docsModel) renderBrowserPane(width, height int) string {
	style := m.styles.pane
	if m.focus == docsFocusBrowser {
		style = m.styles.paneFocused
	}

	innerWidth := maxInt(1, width-style.GetHorizontalFrameSize())
	header := truncateText(
		m.styles.header.Render("Docs Browser")+" "+m.styles.headerMeta.Render(m.browserMeta()),
		innerWidth,
	)
	body := m.browser.View()
	content := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return style.Width(width).Height(height).Render(content)
}

func (m *docsModel) renderViewerPane(width, height int) string {
	style := m.styles.pane
	if m.focus == docsFocusViewer {
		style = m.styles.paneFocused
	}

	innerWidth := maxInt(1, width-style.GetHorizontalFrameSize())
	header := truncateText(
		m.styles.header.Render(m.viewerTitle())+" "+m.styles.headerMeta.Render(m.viewerMeta()),
		innerWidth,
	)

	body := m.viewer.View()
	if strings.TrimSpace(body) == "" {
		body = m.styles.empty.Render("Select a document from the browser to start reading.")
	}
	content := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return style.Width(width).Height(height).Render(content)
}

func (m *docsModel) applyWindowSize(width, height int) {
	m.width = maxInt(0, width)
	m.height = maxInt(0, height)

	if m.width < docsMinWidth || m.height < docsMinHeight {
		return
	}

	bodyHeight := maxInt(1, m.height-docsFooterHeight)
	if m.isNarrow() {
		browserHeight := clampInt(bodyHeight/3, 8, 12)
		viewerHeight := maxInt(5, bodyHeight-browserHeight-docsPaneGap)
		m.setBrowserSize(m.width, browserHeight)
		m.setViewerSize(m.width, viewerHeight)
	} else {
		browserWidth := clampInt(m.width/3, docsBrowserMinWidth, docsBrowserMaxWidth)
		viewerWidth := maxInt(docsMinWidth-browserWidth, m.width-browserWidth-docsPaneGap)
		m.setBrowserSize(browserWidth, bodyHeight)
		m.setViewerSize(viewerWidth, bodyHeight)
	}

	m.renderCurrentDoc(true)
}

func (m *docsModel) windowSizeCmd(width, height int) tea.Cmd {
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

func (m *docsModel) setBrowserSize(width, height int) {
	style := m.styles.pane
	innerWidth := maxInt(1, width-style.GetHorizontalFrameSize())
	innerHeight := maxInt(3, height-style.GetVerticalFrameSize()-docsPaneHeaderLines)
	m.browser.SetSize(innerWidth, innerHeight)
}

func (m *docsModel) setViewerSize(width, height int) {
	style := m.styles.pane
	innerWidth := maxInt(12, width-style.GetHorizontalFrameSize())
	innerHeight := maxInt(3, height-style.GetVerticalFrameSize()-docsPaneHeaderLines)
	m.viewer.SetWidth(innerWidth)
	m.viewer.SetHeight(innerHeight)
}

func (m *docsModel) syncSelectionFromBrowser() {
	selected := m.browser.SelectedItem()
	if selected == nil {
		return
	}

	item, ok := selected.(docsItem)
	if !ok {
		return
	}
	if item.entry.Name == m.currentDocName {
		return
	}
	m.setCurrentDoc(item.entry.Name, false)
}

func (m *docsModel) setCurrentDoc(name string, preserveScroll bool) {
	if _, ok := m.entriesByName[name]; !ok {
		return
	}
	m.currentDocName = name
	m.renderCurrentDoc(preserveScroll)
}

func (m *docsModel) renderCurrentDoc(preserveScroll bool) {
	if m.currentDocName == "" || m.viewer.Width() <= 0 {
		return
	}

	entry, ok := m.entriesByName[m.currentDocName]
	if !ok {
		return
	}

	cacheKey := fmt.Sprintf("%s:%d", entry.Name, m.viewer.Width())
	rendered, ok := m.renderCache[cacheKey]
	if !ok {
		var err error
		rendered, err = renderMarkdownWithGlow(m.viewer.Width(), entry.Content)
		if err != nil {
			rendered = m.styles.error.Render(fmt.Sprintf("Failed to render %s\n\n%s", entry.Name, err))
		}
		m.renderCache[cacheKey] = rendered
	}

	scrollPercent := m.viewer.ScrollPercent()
	m.viewer.SetContent(rendered)
	if preserveScroll {
		m.restoreViewerScroll(scrollPercent)
		return
	}
	m.viewer.GotoTop()
}

func (m *docsModel) restoreViewerScroll(scrollPercent float64) {
	maxOffset := maxInt(0, m.viewer.TotalLineCount()-m.viewer.VisibleLineCount())
	m.viewer.SetYOffset(int(math.Round(scrollPercent * float64(maxOffset))))
}

func (m *docsModel) isNarrow() bool {
	return m.width < docsNarrowWidth
}

func (m *docsModel) browserMeta() string {
	if selected := m.browser.SelectedItem(); selected != nil {
		if item, ok := selected.(docsItem); ok {
			return fmt.Sprintf("selected: %s", item.entry.Name)
		}
	}
	return fmt.Sprintf("%d docs", len(m.entries))
}

func (m *docsModel) viewerTitle() string {
	if m.currentDocName == "" {
		return "Document"
	}
	return m.currentDocName
}

func (m *docsModel) viewerMeta() string {
	total := m.viewer.TotalLineCount()
	if total == 0 {
		return "embedded markdown via Glow"
	}
	line := minInt(total, m.viewer.YOffset()+1)
	return fmt.Sprintf("line %d/%d", line, total)
}

func (m *docsModel) footerText() string {
	if m.focus == docsFocusBrowser && m.browser.FilterState() == list.Filtering {
		return "Type to filter docs, enter to apply, esc to cancel, ctrl+c to quit"
	}
	if m.focus == docsFocusBrowser {
		return "tab switch pane, / search, enter open doc, arrows/jk move, q quit"
	}
	return "tab switch pane, / search, arrows/jk scroll, g/G jump, esc browser, q quit"
}

func truncateText(value string, width int) string {
	if lipgloss.Width(value) <= width {
		return value
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(value)
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

func clampInt(value, low, high int) int {
	return minInt(high, maxInt(low, value))
}

func docsWindowPollCmd() tea.Cmd {
	return tea.Tick(docsWindowPollInterval, func(time.Time) tea.Msg {
		width, height, ok := currentTerminalSize()
		if !ok {
			return docsWindowPollMsg{}
		}
		return docsWindowPollMsg{width: width, height: height}
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
